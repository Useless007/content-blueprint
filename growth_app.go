package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ContentBlueprint/internal/cliprovider"
	"ContentBlueprint/internal/measurement"
	"ContentBlueprint/internal/salesops"
	"ContentBlueprint/internal/workbench"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const growthStageEventName = "growth:ai-stage"

type GrowthBrief = workbench.GrowthBrief
type GrowthPack = workbench.GrowthPack
type GrowthPackSnapshot = workbench.PackSnapshot
type GrowthPlaybook = workbench.Playbook
type GrowthLead = salesops.Lead
type GrowthExperiment = measurement.Experiment
type GrowthExperimentView = measurement.ExperimentView
type GrowthUTMRequest = measurement.UTMRequest
type GrowthUTMResult = measurement.UTMResult

type GrowthGenerationRequest struct {
	RunID      string      `json:"runId"`
	Provider   string      `json:"provider"`
	Workflow   string      `json:"workflow"`
	Model      string      `json:"model,omitempty"`
	Brief      GrowthBrief `json:"brief"`
	TimeoutSec int         `json:"timeoutSec,omitempty"`
}

type GrowthBriefReceipt struct {
	BriefRevision string `json:"briefRevision"`
	UpdatedAt     string `json:"updatedAt"`
}

type GrowthLatestResult struct {
	Found    bool                `json:"found"`
	Stale    bool                `json:"stale"`
	Snapshot *GrowthPackSnapshot `json:"snapshot,omitempty"`
}

type GrowthBootstrapData struct {
	Catalog     []GrowthPlaybook        `json:"catalog"`
	Providers   []FacebookProviderState `json:"providers"`
	Latest      *GrowthPackSnapshot     `json:"latest,omitempty"`
	LatestStale bool                    `json:"latestStale"`
	Leads       []GrowthLead            `json:"leads"`
	Experiments []GrowthExperimentView  `json:"experiments"`
}

type GrowthStageUpdate struct {
	RunID      string `json:"runId"`
	Stage      string `json:"stage"`
	Status     string `json:"status"`
	Provider   string `json:"provider"`
	Workflow   string `json:"workflow"`
	Message    string `json:"message"`
	OccurredAt string `json:"occurredAt"`
}

type growthCoordinator struct {
	store       *workbench.Store
	leads       *salesops.Store
	measurement *measurement.Store
	cli         *cliprovider.Service
	mu          sync.Mutex
	runs        map[string]context.CancelFunc
}

func newGrowthCoordinator() (*growthCoordinator, error) {
	store, err := workbench.NewStore("")
	if err != nil {
		return nil, fmt.Errorf("prepare Growth Workbench storage: %w", err)
	}
	leads, err := salesops.NewStore(store.Directory())
	if err != nil {
		return nil, fmt.Errorf("prepare Growth lead storage: %w", err)
	}
	measurements, err := measurement.NewStore(store.Directory())
	if err != nil {
		return nil, fmt.Errorf("prepare Growth experiment storage: %w", err)
	}
	return &growthCoordinator{store: store, leads: leads, measurement: measurements, cli: cliprovider.New(), runs: make(map[string]context.CancelFunc)}, nil
}

func (app *App) GrowthBootstrap() (GrowthBootstrapData, error) {
	if err := app.growthReady(); err != nil {
		return GrowthBootstrapData{}, err
	}
	ctx, cancel := context.WithTimeout(app.growthContext(), 12*time.Second)
	defer cancel()
	providers := make([]FacebookProviderState, 3)
	var wait sync.WaitGroup
	for index, provider := range []cliprovider.Provider{cliprovider.ProviderClaude, cliprovider.ProviderCodex} {
		index, provider := index, provider
		wait.Add(1)
		go func() {
			defer wait.Done()
			status := app.growth.cli.Status(ctx, provider, "")
			providers[index] = FacebookProviderState{
				ID: string(provider), Label: facebookProviderLabel(string(provider)),
				Available: status.Available && status.Authenticated, AuthenticationChecked: status.AuthenticationChecked,
				Authenticated: status.Authenticated, Version: status.Version, Message: status.Message,
			}
		}()
	}
	wait.Wait()
	providers[2] = FacebookProviderState{ID: "mcp", Label: "MCP · open Claude/Codex manually", Available: true, AuthenticationChecked: true, Authenticated: true, Message: "Sync a Growth Brief and use a configured local MCP workflow."}
	leads, err := app.growth.leads.List()
	if err != nil {
		return GrowthBootstrapData{}, fmt.Errorf("load Growth leads: %w", err)
	}
	experiments, err := app.growth.measurement.List()
	if err != nil {
		return GrowthBootstrapData{}, fmt.Errorf("load Growth experiments: %w", err)
	}
	result := GrowthBootstrapData{Catalog: workbench.Catalog(), Providers: providers, Leads: leads, Experiments: experiments}
	if latest, err := app.growth.store.LoadPack(); err == nil {
		result.Latest = &latest
		brief, briefErr := app.growth.store.LoadBrief()
		result.LatestStale = briefErr != nil || brief.BriefRevision != latest.BriefRevision
	} else if !errors.Is(err, workbench.ErrNotFound) {
		return GrowthBootstrapData{}, fmt.Errorf("load latest Growth Pack: %w", err)
	}
	return result, nil
}

func (app *App) SyncGrowthBrief(brief GrowthBrief) (GrowthBriefReceipt, error) {
	if err := app.growthReady(); err != nil {
		return GrowthBriefReceipt{}, err
	}
	snapshot, err := app.growth.store.SaveBrief(brief)
	if err != nil {
		return GrowthBriefReceipt{}, err
	}
	return GrowthBriefReceipt{BriefRevision: snapshot.BriefRevision, UpdatedAt: snapshot.UpdatedAt.Format(time.RFC3339)}, nil
}

func (app *App) GetLatestGrowthPack() (GrowthLatestResult, error) {
	if err := app.growthReady(); err != nil {
		return GrowthLatestResult{}, err
	}
	pack, err := app.growth.store.LoadPack()
	if errors.Is(err, workbench.ErrNotFound) {
		return GrowthLatestResult{}, nil
	}
	if err != nil {
		return GrowthLatestResult{}, err
	}
	brief, briefErr := app.growth.store.LoadBrief()
	stale := briefErr != nil || brief.BriefRevision != pack.BriefRevision
	return GrowthLatestResult{Found: true, Stale: stale, Snapshot: &pack}, nil
}

func (app *App) ReviewGrowthPack(briefRevision, status, reviewerNote string) (GrowthPackSnapshot, error) {
	if err := app.growthReady(); err != nil {
		return GrowthPackSnapshot{}, err
	}
	return app.growth.store.ReviewPack(briefRevision, status, reviewerNote)
}

func (app *App) GenerateGrowthPack(request GrowthGenerationRequest) (GrowthPackSnapshot, error) {
	if err := app.growthReady(); err != nil {
		return GrowthPackSnapshot{}, err
	}
	provider := cliprovider.Provider(strings.ToLower(strings.TrimSpace(request.Provider)))
	if !provider.Valid() {
		return GrowthPackSnapshot{}, fmt.Errorf("choose Claude Code or Codex CLI")
	}
	workflow := strings.ToLower(strings.TrimSpace(request.Workflow))
	if workflow == "" {
		workflow = "single"
	}
	if workflow != "single" && workflow != "team" {
		return GrowthPackSnapshot{}, fmt.Errorf("workflow must be single or team")
	}
	runID, err := normalizeRunID(request.RunID)
	if err != nil {
		return GrowthPackSnapshot{}, err
	}
	runContext, finish, err := app.beginGrowthRun(runID)
	if err != nil {
		return GrowthPackSnapshot{}, err
	}
	defer finish()
	briefSnapshot, err := app.growth.store.SaveBrief(request.Brief)
	if err != nil {
		return GrowthPackSnapshot{}, err
	}
	emit := func(stage, status, message string) {
		app.emitGrowthStage(GrowthStageUpdate{RunID: runID, Stage: stage, Status: status, Provider: string(provider), Workflow: workflow, Message: message, OccurredAt: time.Now().UTC().Format(time.RFC3339)})
	}
	if workflow == "team" {
		emit("strategist", "queued", "Waiting for strategy")
		emit("copywriter", "queued", "Waiting for producer")
		emit("reviewer", "queued", "Waiting for evidence review")
	} else {
		emit("copywriter", "queued", "Waiting for Growth Pack producer")
	}
	timeout := time.Duration(request.TimeoutSec) * time.Second
	if timeout == 0 && workflow == "team" {
		timeout = 15 * time.Minute
	}
	progressFailed := false
	pack, err := app.growth.cli.GenerateGrowth(runContext, briefSnapshot.Brief, cliprovider.Options{
		Provider: provider, Model: strings.TrimSpace(request.Model), Timeout: timeout, Workflow: cliprovider.Workflow(workflow),
		Progress: func(progress cliprovider.ProgressEvent) {
			stage := string(progress.Stage)
			if progress.Stage == cliprovider.StageGenerate {
				stage = "copywriter"
			}
			status := "working"
			if progress.Status == cliprovider.StageCompleted {
				status = "done"
			} else if progress.Status == cliprovider.StageFailed {
				status = "error"
				progressFailed = true
			}
			emit(stage, status, growthProgressMessage(stage, status))
		},
	})
	if err != nil {
		if !progressFailed {
			stage := "copywriter"
			if workflow == "team" {
				stage = "strategist"
			}
			var providerErr *cliprovider.Error
			if errors.As(err, &providerErr) {
				switch providerErr.Stage {
				case cliprovider.StageStrategist, cliprovider.StageCopywriter, cliprovider.StageReviewer:
					stage = string(providerErr.Stage)
				case cliprovider.StageGenerate:
					stage = "copywriter"
				}
			}
			emit(stage, "error", safeFacebookError(err))
		}
		return GrowthPackSnapshot{}, err
	}
	if err := runContext.Err(); err != nil {
		return GrowthPackSnapshot{}, err
	}
	emit("browserCourier", "working", "Saving local Growth Pack")
	generatedBy := facebookProviderLabel(string(provider))
	if workflow == "team" {
		generatedBy += " · AI team"
	}
	saved, err := app.growth.store.SavePack(briefSnapshot.BriefRevision, pack, generatedBy)
	if err != nil {
		emit("browserCourier", "error", "Could not save local Growth Pack")
		return GrowthPackSnapshot{}, err
	}
	emit("browserCourier", "done", "Growth Pack is ready for human review")
	return saved, nil
}

func (app *App) CancelGrowthGeneration(runID string) bool {
	runID = strings.TrimSpace(runID)
	if app.growth == nil || runID == "" {
		return false
	}
	app.growth.mu.Lock()
	cancel, exists := app.growth.runs[runID]
	app.growth.mu.Unlock()
	if exists {
		cancel()
	}
	return exists
}

func (app *App) SaveGrowthLead(lead GrowthLead) (GrowthLead, error) {
	if err := app.growthReady(); err != nil {
		return GrowthLead{}, err
	}
	return app.growth.leads.Save(lead)
}

func (app *App) DeleteGrowthLead(id string) error {
	if err := app.growthReady(); err != nil {
		return err
	}
	return app.growth.leads.Delete(id)
}

func (app *App) SaveGrowthExperiment(experiment GrowthExperiment) (GrowthExperimentView, error) {
	if err := app.growthReady(); err != nil {
		return GrowthExperimentView{}, err
	}
	return app.growth.measurement.Save(experiment)
}

func (app *App) DeleteGrowthExperiment(id string) error {
	if err := app.growthReady(); err != nil {
		return err
	}
	return app.growth.measurement.Delete(id)
}

func (app *App) BuildGrowthUTM(request GrowthUTMRequest) (GrowthUTMResult, error) {
	return measurement.BuildUTM(request)
}

func (app *App) beginGrowthRun(runID string) (context.Context, func(), error) {
	app.growth.mu.Lock()
	defer app.growth.mu.Unlock()
	if _, exists := app.growth.runs[runID]; exists {
		return nil, nil, fmt.Errorf("runId is already active")
	}
	ctx, cancel := context.WithCancel(app.growthContext())
	app.growth.runs[runID] = cancel
	return ctx, func() {
		cancel()
		app.growth.mu.Lock()
		delete(app.growth.runs, runID)
		app.growth.mu.Unlock()
	}, nil
}

func (app *App) emitGrowthStage(update GrowthStageUpdate) {
	if app.ctx == nil || app.ctx.Value("events") == nil {
		return
	}
	wailsruntime.EventsEmit(app.ctx, growthStageEventName, update)
}

func (app *App) growthReady() error {
	if app.growth == nil || app.growth.store == nil || app.growth.leads == nil || app.growth.measurement == nil || app.growth.cli == nil {
		return fmt.Errorf("Growth Copilot is unavailable")
	}
	return nil
}

func (app *App) growthContext() context.Context {
	if app.ctx != nil {
		return app.ctx
	}
	return context.Background()
}

func growthProgressMessage(stage, status string) string {
	label := map[string]string{"strategist": "Strategist", "copywriter": "Producer", "reviewer": "Reviewer"}[stage]
	if status == "done" {
		return label + " completed the local handoff"
	}
	if status == "error" {
		return label + " stopped; review the safe error details"
	}
	return label + " is working in an isolated CLI process"
}
