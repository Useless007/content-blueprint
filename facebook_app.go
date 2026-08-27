package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"ContentBlueprint/internal/cliprovider"
	"ContentBlueprint/internal/facebookcompanion"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const facebookStageEventName = "facebook:ai-stage"

var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)

type FacebookBrief = facebookcompanion.Brief
type FacebookContentPack = facebookcompanion.ContentPack
type FacebookPackSnapshot = facebookcompanion.PackSnapshot

type FacebookProviderState struct {
	ID                    string `json:"id"`
	Label                 string `json:"label"`
	Available             bool   `json:"available"`
	AuthenticationChecked bool   `json:"authenticationChecked"`
	Authenticated         bool   `json:"authenticated"`
	Version               string `json:"version,omitempty"`
	Message               string `json:"message,omitempty"`
}

type FacebookBootstrapData struct {
	Providers []FacebookProviderState `json:"providers"`
	Latest    *FacebookPackSnapshot   `json:"latest,omitempty"`
}

type FacebookGenerationRequest struct {
	RunID      string        `json:"runId"`
	Provider   string        `json:"provider"`
	Workflow   string        `json:"workflow"`
	Model      string        `json:"model,omitempty"`
	Brief      FacebookBrief `json:"brief"`
	TimeoutSec int           `json:"timeoutSec,omitempty"`
}

type FacebookBriefReceipt struct {
	BriefRevision string `json:"briefRevision"`
	UpdatedAt     string `json:"updatedAt"`
}

type FacebookLatestResult struct {
	Found    bool                  `json:"found"`
	Stale    bool                  `json:"stale"`
	Snapshot *FacebookPackSnapshot `json:"snapshot,omitempty"`
}

type FacebookStageUpdate struct {
	RunID      string `json:"runId"`
	Stage      string `json:"stage"`
	Status     string `json:"status"`
	Provider   string `json:"provider"`
	Workflow   string `json:"workflow"`
	Message    string `json:"message"`
	OccurredAt string `json:"occurredAt"`
}

type facebookCoordinator struct {
	store *facebookcompanion.Store
	cli   *cliprovider.Service
	mu    sync.Mutex
	runs  map[string]context.CancelFunc
}

func newFacebookCoordinator() (*facebookCoordinator, error) {
	store, err := facebookcompanion.NewStore("")
	if err != nil {
		return nil, fmt.Errorf("prepare Facebook companion storage: %w", err)
	}
	return &facebookCoordinator{
		store: store,
		cli:   cliprovider.New(),
		runs:  make(map[string]context.CancelFunc),
	}, nil
}

func (app *App) FacebookBootstrap() (FacebookBootstrapData, error) {
	if err := app.facebookReady(); err != nil {
		return FacebookBootstrapData{}, err
	}
	ctx, cancel := context.WithTimeout(app.facebookContext(), 12*time.Second)
	defer cancel()

	providers := make([]FacebookProviderState, 3)
	var wait sync.WaitGroup
	for index, provider := range []cliprovider.Provider{cliprovider.ProviderClaude, cliprovider.ProviderCodex} {
		index, provider := index, provider
		wait.Add(1)
		go func() {
			defer wait.Done()
			status := app.facebook.cli.Status(ctx, provider, "")
			providers[index] = FacebookProviderState{
				ID:                    string(provider),
				Label:                 facebookProviderLabel(string(provider)),
				Available:             status.Available && status.Authenticated,
				AuthenticationChecked: status.AuthenticationChecked,
				Authenticated:         status.Authenticated,
				Version:               status.Version,
				Message:               status.Message,
			}
		}()
	}
	wait.Wait()
	providers[2] = FacebookProviderState{
		ID:                    "mcp",
		Label:                 "MCP · เปิด Claude/Codex เอง",
		Available:             true,
		AuthenticationChecked: true,
		Authenticated:         true,
		Message:               "ซิงก์ brief แล้วให้ Claude/Codex เรียกเครื่องมือ MCP เพื่อส่ง Content Pack กลับมา",
	}

	result := FacebookBootstrapData{Providers: providers}
	if latest, err := app.facebook.store.LoadPack(); err == nil {
		result.Latest = &latest
	} else if !errors.Is(err, facebookcompanion.ErrNotFound) {
		return FacebookBootstrapData{}, fmt.Errorf("read latest Facebook Content Pack: %w", err)
	}
	return result, nil
}

func (app *App) SyncFacebookBrief(brief FacebookBrief) (FacebookBriefReceipt, error) {
	if err := app.facebookReady(); err != nil {
		return FacebookBriefReceipt{}, err
	}
	snapshot, err := app.facebook.store.SaveBrief(brief)
	if err != nil {
		return FacebookBriefReceipt{}, err
	}
	return FacebookBriefReceipt{
		BriefRevision: snapshot.BriefRevision,
		UpdatedAt:     snapshot.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (app *App) GetLatestFacebookPack() (FacebookLatestResult, error) {
	if err := app.facebookReady(); err != nil {
		return FacebookLatestResult{}, err
	}
	pack, err := app.facebook.store.LoadPack()
	if errors.Is(err, facebookcompanion.ErrNotFound) {
		return FacebookLatestResult{Found: false}, nil
	}
	if err != nil {
		return FacebookLatestResult{}, err
	}
	brief, briefErr := app.facebook.store.LoadBrief()
	if briefErr != nil && !errors.Is(briefErr, facebookcompanion.ErrNotFound) {
		return FacebookLatestResult{}, briefErr
	}
	stale := errors.Is(briefErr, facebookcompanion.ErrNotFound) || brief.BriefRevision != pack.BriefRevision
	return FacebookLatestResult{Found: true, Stale: stale, Snapshot: &pack}, nil
}

func (app *App) GenerateFacebookPack(request FacebookGenerationRequest) (FacebookPackSnapshot, error) {
	if err := app.facebookReady(); err != nil {
		return FacebookPackSnapshot{}, err
	}
	provider := cliprovider.Provider(strings.ToLower(strings.TrimSpace(request.Provider)))
	if !provider.Valid() {
		return FacebookPackSnapshot{}, fmt.Errorf("เลือก Claude Code หรือ Codex CLI สำหรับการสร้างอัตโนมัติ")
	}
	workflow := strings.ToLower(strings.TrimSpace(request.Workflow))
	if workflow == "" {
		workflow = "single"
	}
	if workflow != "single" && workflow != "team" {
		return FacebookPackSnapshot{}, fmt.Errorf("workflow ต้องเป็น single หรือ team")
	}
	runID, err := normalizeRunID(request.RunID)
	if err != nil {
		return FacebookPackSnapshot{}, err
	}
	runContext, finishRun, err := app.beginFacebookRun(runID)
	if err != nil {
		return FacebookPackSnapshot{}, err
	}
	defer finishRun()
	briefSnapshot, err := app.facebook.store.SaveBrief(request.Brief)
	if err != nil {
		return FacebookPackSnapshot{}, err
	}
	emit := func(stage, status, message string) {
		app.emitFacebookStage(FacebookStageUpdate{
			RunID:      runID,
			Stage:      stage,
			Status:     status,
			Provider:   string(provider),
			Workflow:   workflow,
			Message:    message,
			OccurredAt: time.Now().UTC().Format(time.RFC3339),
		})
	}

	if workflow == "team" {
		emit("strategist", "queued", "รอวิเคราะห์ brief และวางมุมสื่อสาร")
		emit("copywriter", "queued", "รอรับแผนจาก Strategist")
		emit("reviewer", "queued", "รอตรวจร่างจาก Copywriter")
	} else {
		emit("strategist", "done", "Quick draft ใช้ brief ที่ตรวจรูปแบบแล้วโดยตรง")
		emit("copywriter", "queued", "รอสร้าง Facebook Content Pack")
		emit("reviewer", "queued", "รอตรวจรูปแบบและข้อควรระวัง")
	}
	timeout := time.Duration(request.TimeoutSec) * time.Second
	if timeout == 0 && workflow == "team" {
		timeout = 15 * time.Minute
	}
	progressFailed := false
	pack, err := app.facebook.cli.Generate(runContext, briefSnapshot.Brief, cliprovider.Options{
		Provider: provider,
		Model:    strings.TrimSpace(request.Model),
		Timeout:  timeout,
		Workflow: cliprovider.Workflow(workflow),
		Progress: func(progress cliprovider.ProgressEvent) {
			stage := string(progress.Stage)
			if progress.Stage == cliprovider.StageGenerate {
				stage = "copywriter"
			}
			status := "working"
			switch progress.Status {
			case cliprovider.StageCompleted:
				status = "done"
			case cliprovider.StageFailed:
				status = "error"
				progressFailed = true
			}
			emit(stage, status, facebookProgressMessage(stage, status))
		},
	})
	if err != nil {
		if !progressFailed {
			failedStage := "copywriter"
			if workflow == "team" {
				failedStage = "strategist"
			}
			var providerErr *cliprovider.Error
			if errors.As(err, &providerErr) {
				switch providerErr.Stage {
				case cliprovider.StageStrategist, cliprovider.StageCopywriter, cliprovider.StageReviewer:
					failedStage = string(providerErr.Stage)
				case cliprovider.StageGenerate:
					failedStage = "copywriter"
				}
			}
			emit(failedStage, "error", safeFacebookError(err))
		}
		return FacebookPackSnapshot{}, err
	}
	if workflow == "single" {
		emit("reviewer", "working", "กำลังตรวจ schema หลักฐาน และข้อควรระวัง")
	}
	if err := facebookcompanion.ValidateContentPack(pack); err != nil {
		emit("reviewer", "error", "Content Pack ไม่ผ่านการตรวจรูปแบบ")
		return FacebookPackSnapshot{}, err
	}
	if workflow == "single" {
		emit("reviewer", "done", "Content Pack ผ่านการตรวจและพร้อมให้มนุษย์ทบทวน")
	}
	if err := runContext.Err(); err != nil {
		return FacebookPackSnapshot{}, err
	}

	emit("browserCourier", "working", "กำลังส่งผลไปยัง Companion inbox")
	generatedBy := facebookProviderLabel(string(provider))
	if workflow == "team" {
		generatedBy += " · AI team"
	}
	saved, err := app.facebook.store.SavePack(briefSnapshot.BriefRevision, pack, nil, generatedBy)
	if err != nil {
		emit("browserCourier", "error", "บันทึกผลลง Companion inbox ไม่สำเร็จ")
		return FacebookPackSnapshot{}, err
	}
	emit("browserCourier", "done", "พร้อมให้ Chrome/Brave extension รับไปตรวจทาน")
	return saved, nil
}

func (app *App) CancelFacebookGeneration(runID string) bool {
	runID = strings.TrimSpace(runID)
	if app.facebook == nil || runID == "" {
		return false
	}
	app.facebook.mu.Lock()
	cancel, exists := app.facebook.runs[runID]
	app.facebook.mu.Unlock()
	if exists {
		cancel()
	}
	return exists
}

func (app *App) beginFacebookRun(runID string) (context.Context, func(), error) {
	app.facebook.mu.Lock()
	defer app.facebook.mu.Unlock()
	if _, exists := app.facebook.runs[runID]; exists {
		return nil, nil, fmt.Errorf("runId นี้กำลังทำงานอยู่แล้ว")
	}
	ctx, cancel := context.WithCancel(app.facebookContext())
	app.facebook.runs[runID] = cancel
	finish := func() {
		cancel()
		app.facebook.mu.Lock()
		delete(app.facebook.runs, runID)
		app.facebook.mu.Unlock()
	}
	return ctx, finish, nil
}

func (app *App) emitFacebookStage(update FacebookStageUpdate) {
	if app.ctx == nil || app.ctx.Value("events") == nil {
		return
	}
	wailsruntime.EventsEmit(app.ctx, facebookStageEventName, update)
}

func (app *App) facebookReady() error {
	if app.facebook == nil || app.facebook.store == nil || app.facebook.cli == nil {
		return fmt.Errorf("Facebook AI workspace is unavailable")
	}
	return nil
}

func (app *App) facebookContext() context.Context {
	if app.ctx != nil {
		return app.ctx
	}
	return context.Background()
}

func normalizeRunID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", fmt.Errorf("create run ID: %w", err)
		}
		value = hex.EncodeToString(random[:])
	}
	if !runIDPattern.MatchString(value) {
		return "", fmt.Errorf("runId มีรูปแบบไม่ถูกต้อง")
	}
	return value, nil
}

func facebookProviderLabel(provider string) string {
	switch provider {
	case "claude":
		return "Claude Code"
	case "codex":
		return "Codex CLI"
	default:
		return provider
	}
}

func safeFacebookError(err error) string {
	if cliprovider.IsCode(err, cliprovider.CodeCancelled) {
		return "ผู้ใช้ยกเลิกการทำงานแล้ว"
	}
	if cliprovider.IsCode(err, cliprovider.CodeTimeout) {
		return "AI worker ใช้เวลาเกินกำหนด"
	}
	if cliprovider.IsCode(err, cliprovider.CodeUnavailable) {
		return "ไม่พบ CLI ที่ล็อกอินและพร้อมใช้งาน"
	}
	return "AI worker ทำงานไม่สำเร็จ กรุณาตรวจรายละเอียดและลองใหม่"
}

func facebookProgressMessage(stage, status string) string {
	labels := map[string]string{
		"strategist": "Strategist",
		"copywriter": "Copywriter",
		"reviewer":   "Reviewer",
	}
	label := labels[stage]
	switch status {
	case "done":
		return label + " ส่งมอบงานให้ worker ถัดไปแล้ว"
	case "error":
		return label + " ทำงานไม่สำเร็จและหยุด pipeline แล้ว"
	default:
		return label + " กำลังทำงานใน process แยกแบบ isolated"
	}
}
