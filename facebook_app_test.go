package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"ContentBlueprint/internal/cliprovider"
	"ContentBlueprint/internal/facebookcompanion"
)

type facebookAppFakeRunner struct {
	paths map[string]string
	run   func(context.Context, cliprovider.Command) (cliprovider.ProcessResult, error)
}

type facebookEventCountingContext struct {
	context.Context
	mu    sync.Mutex
	count int
}

func (ctx *facebookEventCountingContext) Value(key any) any {
	if key == "events" {
		ctx.mu.Lock()
		ctx.count++
		ctx.mu.Unlock()
		return nil
	}
	return ctx.Context.Value(key)
}

func (ctx *facebookEventCountingContext) eventAttempts() int {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.count
}

func (runner *facebookAppFakeRunner) LookPath(file string) (string, error) {
	if path, exists := runner.paths[file]; exists {
		return path, nil
	}
	return "", os.ErrNotExist
}

func (runner *facebookAppFakeRunner) Run(ctx context.Context, command cliprovider.Command) (cliprovider.ProcessResult, error) {
	if runner.run == nil {
		return cliprovider.ProcessResult{}, nil
	}
	return runner.run(ctx, command)
}

func TestFacebookBootstrapMapsAuthenticationToAvailability(t *testing.T) {
	runner := &facebookAppFakeRunner{
		paths: facebookAppFakePaths(),
		run: func(_ context.Context, command cliprovider.Command) (cliprovider.ProcessResult, error) {
			switch {
			case hasFacebookAppArgs(command.Args, "--version") && command.Path == "fake-claude":
				return cliprovider.ProcessResult{Stdout: []byte("claude 9.8.7\n")}, nil
			case hasFacebookAppArgs(command.Args, "auth", "status", "--json") && command.Path == "fake-claude":
				return cliprovider.ProcessResult{Stdout: []byte(`{"loggedIn":true,"email":"must-not-escape@example.test"}`)}, nil
			case hasFacebookAppArgs(command.Args, "--version") && command.Path == "fake-codex":
				return cliprovider.ProcessResult{Stdout: []byte("codex-cli 1.2.3\n")}, nil
			case hasFacebookAppArgs(command.Args, "login", "status") && command.Path == "fake-codex":
				return cliprovider.ProcessResult{}, errors.New("not logged in")
			default:
				return cliprovider.ProcessResult{}, fmt.Errorf("unexpected command: path=%q args=%q", command.Path, command.Args)
			}
		},
	}
	app := newFacebookAppForTest(t, runner)

	bootstrap, err := app.FacebookBootstrap()
	if err != nil {
		t.Fatalf("FacebookBootstrap: %v", err)
	}
	if len(bootstrap.Providers) != 3 {
		t.Fatalf("providers length = %d, want 3", len(bootstrap.Providers))
	}

	claude := bootstrap.Providers[0]
	if claude.ID != "claude" || !claude.Available || !claude.AuthenticationChecked || !claude.Authenticated {
		t.Fatalf("Claude provider = %#v, want authenticated and available", claude)
	}
	if claude.Version != "claude 9.8.7" {
		t.Fatalf("Claude version = %q", claude.Version)
	}
	if strings.Contains(claude.Message, "example.test") {
		t.Fatal("Claude account metadata escaped into the bootstrap response")
	}

	codex := bootstrap.Providers[1]
	if codex.ID != "codex" || codex.Available || !codex.AuthenticationChecked || codex.Authenticated {
		t.Fatalf("Codex provider = %#v, want detected but unavailable until authenticated", codex)
	}
	if codex.Version != "codex-cli 1.2.3" || codex.Message == "" {
		t.Fatalf("Codex provider did not preserve safe status details: %#v", codex)
	}

	mcp := bootstrap.Providers[2]
	if mcp.ID != "mcp" || !mcp.Available || !mcp.AuthenticationChecked || !mcp.Authenticated {
		t.Fatalf("MCP provider = %#v, want locally available", mcp)
	}
	if bootstrap.Latest != nil {
		t.Fatalf("latest = %#v, want nil before generation", bootstrap.Latest)
	}
}

func TestGenerateFacebookPackSinglePersistsAndLoadsLatestPack(t *testing.T) {
	wantPack := validFacebookAppPack()
	runner := newFacebookAppClaudeRunner(t, wantPack)
	app := newFacebookAppForTest(t, runner)
	brief := validFacebookAppBrief()

	saved, err := app.GenerateFacebookPack(FacebookGenerationRequest{
		RunID:      "single-run",
		Provider:   "claude",
		Workflow:   "single",
		Brief:      brief,
		TimeoutSec: 5,
	})
	if err != nil {
		t.Fatalf("GenerateFacebookPack: %v", err)
	}
	if saved.BriefRevision == "" {
		t.Fatal("saved pack has an empty brief revision")
	}
	if saved.GeneratedBy != "Claude Code" {
		t.Fatalf("generatedBy = %q, want Claude Code", saved.GeneratedBy)
	}
	if !reflect.DeepEqual(saved.Pack, wantPack) {
		t.Fatalf("saved pack = %#v, want %#v", saved.Pack, wantPack)
	}

	latest, err := app.GetLatestFacebookPack()
	if err != nil {
		t.Fatalf("GetLatestFacebookPack: %v", err)
	}
	if !latest.Found || latest.Stale || latest.Snapshot == nil {
		t.Fatalf("latest = %#v, want a current snapshot", latest)
	}
	if latest.Snapshot.BriefRevision != saved.BriefRevision || !reflect.DeepEqual(latest.Snapshot.Pack, wantPack) {
		t.Fatalf("loaded snapshot = %#v, want generated snapshot", latest.Snapshot)
	}

	storedBrief, err := app.facebook.store.LoadBrief()
	if err != nil {
		t.Fatalf("LoadBrief: %v", err)
	}
	if storedBrief.BriefRevision != saved.BriefRevision || !reflect.DeepEqual(storedBrief.Brief, facebookcompanion.NormalizeBrief(brief)) {
		t.Fatalf("stored brief = %#v, want generated brief", storedBrief)
	}
}

func TestGetLatestFacebookPackBecomesStaleAfterBriefChanges(t *testing.T) {
	app := newFacebookAppForTest(t, newFacebookAppClaudeRunner(t, validFacebookAppPack()))
	original := validFacebookAppBrief()
	saved, err := app.GenerateFacebookPack(FacebookGenerationRequest{
		RunID: "stale-source-run", Provider: "claude", Workflow: "single", Brief: original, TimeoutSec: 5,
	})
	if err != nil {
		t.Fatalf("GenerateFacebookPack: %v", err)
	}

	changed := original
	changed.Objective = "Invite readers to register for the live workshop"
	receipt, err := app.SyncFacebookBrief(changed)
	if err != nil {
		t.Fatalf("SyncFacebookBrief: %v", err)
	}
	if receipt.BriefRevision == saved.BriefRevision {
		t.Fatal("changed brief retained the old revision")
	}

	latest, err := app.GetLatestFacebookPack()
	if err != nil {
		t.Fatalf("GetLatestFacebookPack: %v", err)
	}
	if !latest.Found || !latest.Stale || latest.Snapshot == nil {
		t.Fatalf("latest = %#v, want the previous pack marked stale", latest)
	}
	if latest.Snapshot.BriefRevision != saved.BriefRevision {
		t.Fatalf("stale pack revision = %q, want %q", latest.Snapshot.BriefRevision, saved.BriefRevision)
	}
}

func TestGenerateFacebookPackRejectsDuplicateRunBeforeMutatingBrief(t *testing.T) {
	app := newFacebookAppForTest(t, newFacebookAppClaudeRunner(t, validFacebookAppPack()))
	original, err := app.facebook.store.SaveBrief(validFacebookAppBrief())
	if err != nil {
		t.Fatalf("SaveBrief: %v", err)
	}
	_, finish, err := app.beginFacebookRun("duplicate-run")
	if err != nil {
		t.Fatalf("beginFacebookRun: %v", err)
	}
	defer finish()

	changed := validFacebookAppBrief()
	changed.Objective = "This must not replace the active run's brief"
	_, err = app.GenerateFacebookPack(FacebookGenerationRequest{
		RunID: "duplicate-run", Provider: "claude", Workflow: "single", Brief: changed, TimeoutSec: 5,
	})
	if err == nil {
		t.Fatal("GenerateFacebookPack accepted a duplicate active run ID")
	}
	stored, err := app.facebook.store.LoadBrief()
	if err != nil {
		t.Fatalf("LoadBrief: %v", err)
	}
	if stored.BriefRevision != original.BriefRevision || !reflect.DeepEqual(stored.Brief, original.Brief) {
		t.Fatalf("duplicate run mutated stored brief: got %#v, want %#v", stored, original)
	}
}

func TestGenerateFacebookPackMapsTypedTeamErrorToMatchingWorkerStage(t *testing.T) {
	strategyEnvelope, err := json.Marshal(map[string]any{
		"type":     "result",
		"subtype":  "success",
		"is_error": false,
		"structured_output": cliprovider.TeamStrategy{
			AudienceInsight: "Shop owners need a practical plan.",
			Positioning:     "A clear and evidence-based workshop.",
			PrimaryPromise:  "Organize the marketing plan without guaranteeing outcomes.",
			Angles: []cliprovider.StrategyAngle{
				{Name: "Problem", HookApproach: "Start with wasted effort.", Rationale: "Matches a common planning pain point."},
				{Name: "Principle", HookApproach: "Start with the right customer.", Rationale: "Gives readers a decision framework."},
				{Name: "Action", HookApproach: "Start with one clear message.", Rationale: "Offers a practical next step."},
			},
			NarrativeFlow:   []string{"Name the problem.", "Explain the principle.", "Offer the next step."},
			EvidenceUse:     []cliprovider.EvidenceUse{},
			ComplianceRisks: []string{"Do not guarantee sales results."},
		},
	})
	if err != nil {
		t.Fatalf("marshal fake strategist response: %v", err)
	}
	runner := &facebookAppFakeRunner{
		paths: facebookAppFakePaths(),
		run: func(_ context.Context, command cliprovider.Command) (cliprovider.ProcessResult, error) {
			if command.Path != "fake-claude" || !hasFacebookAppArgs(command.Args, "-p") {
				return cliprovider.ProcessResult{}, fmt.Errorf("unexpected command: path=%q args=%q", command.Path, command.Args)
			}
			// Removing the workflow root after a valid strategist response makes
			// preparation of the copywriter directory fail. That error is typed as
			// StageCopywriter and occurs before a copywriter progress callback.
			if err := os.RemoveAll(filepath.Dir(command.Dir)); err != nil {
				return cliprovider.ProcessResult{}, fmt.Errorf("remove fake workflow root: %w", err)
			}
			return cliprovider.ProcessResult{Stdout: strategyEnvelope}, nil
		},
	}
	app := newFacebookAppForTest(t, runner)
	eventContext := &facebookEventCountingContext{Context: context.Background()}
	app.ctx = eventContext

	_, err = app.GenerateFacebookPack(FacebookGenerationRequest{
		RunID: "copywriter-setup-failure", Provider: "claude", Workflow: "team", Brief: validFacebookAppBrief(), TimeoutSec: 5,
	})
	if err == nil {
		t.Fatal("GenerateFacebookPack succeeded after the workflow directory was removed")
	}
	var providerErr *cliprovider.Error
	if !errors.As(err, &providerErr) {
		t.Fatalf("GenerateFacebookPack error = %T %v, want typed provider error", err, err)
	}
	if providerErr.Stage != cliprovider.StageCopywriter {
		t.Fatalf("provider error stage = %q, want %q", providerErr.Stage, cliprovider.StageCopywriter)
	}
	if providerErr.Code != cliprovider.CodeProcess {
		t.Fatalf("provider error code = %q, want %q", providerErr.Code, cliprovider.CodeProcess)
	}
	// Three queued updates, strategist started/completed, and one fallback
	// error update. The sixth attempt proves the no-progress-error path emits.
	if got := eventContext.eventAttempts(); got != 6 {
		t.Fatalf("event attempts = %d, want 6 including the fallback error update", got)
	}
	assertFacebookFallbackUsesTypedProviderStage(t)
}

func TestCancelFacebookGenerationStopsRunner(t *testing.T) {
	started := make(chan struct{})
	var startOnce sync.Once
	runner := &facebookAppFakeRunner{
		paths: facebookAppFakePaths(),
		run: func(ctx context.Context, command cliprovider.Command) (cliprovider.ProcessResult, error) {
			if command.Path != "fake-codex" || !hasFacebookAppArgs(command.Args, "exec") {
				return cliprovider.ProcessResult{}, fmt.Errorf("unexpected command: path=%q args=%q", command.Path, command.Args)
			}
			startOnce.Do(func() { close(started) })
			<-ctx.Done()
			return cliprovider.ProcessResult{}, ctx.Err()
		},
	}
	app := newFacebookAppForTest(t, runner)
	errCh := make(chan error, 1)
	go func() {
		_, err := app.GenerateFacebookPack(FacebookGenerationRequest{
			RunID: "cancel-run", Provider: "codex", Workflow: "single", Brief: validFacebookAppBrief(), TimeoutSec: 10,
		})
		errCh <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("fake CLI runner did not start")
	}
	if !app.CancelFacebookGeneration("cancel-run") {
		t.Fatal("CancelFacebookGeneration returned false for an active run")
	}

	select {
	case err := <-errCh:
		if err == nil || !cliprovider.IsCode(err, cliprovider.CodeCancelled) {
			t.Fatalf("GenerateFacebookPack error = %v, want cancelled provider error", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("generation did not stop after cancellation")
	}
	if app.CancelFacebookGeneration("cancel-run") {
		t.Fatal("completed cancelled run remained registered")
	}
	latest, err := app.GetLatestFacebookPack()
	if err != nil {
		t.Fatalf("GetLatestFacebookPack: %v", err)
	}
	if latest.Found {
		t.Fatalf("cancelled generation persisted a pack: %#v", latest)
	}
}

func newFacebookAppForTest(t *testing.T, runner cliprovider.Runner) *App {
	t.Helper()
	store, err := facebookcompanion.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return &App{
		ctx: context.Background(),
		facebook: &facebookCoordinator{
			store: store,
			cli:   cliprovider.NewWithRunner(runner),
			runs:  make(map[string]context.CancelFunc),
		},
	}
}

func newFacebookAppClaudeRunner(t *testing.T, pack facebookcompanion.ContentPack) *facebookAppFakeRunner {
	t.Helper()
	envelope, err := json.Marshal(map[string]any{
		"type":              "result",
		"subtype":           "success",
		"is_error":          false,
		"structured_output": pack,
	})
	if err != nil {
		t.Fatalf("marshal fake Claude envelope: %v", err)
	}
	return &facebookAppFakeRunner{
		paths: facebookAppFakePaths(),
		run: func(_ context.Context, command cliprovider.Command) (cliprovider.ProcessResult, error) {
			if command.Path != "fake-claude" || !hasFacebookAppArgs(command.Args, "-p") {
				return cliprovider.ProcessResult{}, fmt.Errorf("unexpected command: path=%q args=%q", command.Path, command.Args)
			}
			return cliprovider.ProcessResult{Stdout: envelope}, nil
		},
	}
}

func facebookAppFakePaths() map[string]string {
	return map[string]string{
		"claude":     "fake-claude",
		"claude.exe": "fake-claude",
		"codex":      "fake-codex",
	}
}

func assertFacebookFallbackUsesTypedProviderStage(t *testing.T) {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), "facebook_app.go", nil, 0)
	if err != nil {
		t.Fatalf("parse facebook_app.go: %v", err)
	}
	var generate *ast.FuncDecl
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "GenerateFacebookPack" {
			generate = function
			break
		}
	}
	if generate == nil {
		t.Fatal("GenerateFacebookPack declaration was not found")
	}

	mapped := map[string]bool{
		"StageStrategist": false,
		"StageCopywriter": false,
		"StageReviewer":   false,
	}
	emitsMappedError := false
	ast.Inspect(generate.Body, func(node ast.Node) bool {
		switchStatement, ok := node.(*ast.SwitchStmt)
		if ok && isFacebookProviderStageSelector(switchStatement.Tag) {
			for _, statement := range switchStatement.Body.List {
				clause, ok := statement.(*ast.CaseClause)
				if !ok || !facebookCaseAssignsProviderStage(clause.Body) {
					continue
				}
				for _, expression := range clause.List {
					selector, ok := expression.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					if identifier, ok := selector.X.(*ast.Ident); ok && identifier.Name == "cliprovider" {
						if _, tracked := mapped[selector.Sel.Name]; tracked {
							mapped[selector.Sel.Name] = true
						}
					}
				}
			}
		}
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		callee, ok := call.Fun.(*ast.Ident)
		stage, stageOK := call.Args[0].(*ast.Ident)
		status, statusOK := call.Args[1].(*ast.BasicLit)
		if ok && callee.Name == "emit" && stageOK && stage.Name == "failedStage" && statusOK && status.Value == `"error"` {
			emitsMappedError = true
		}
		return true
	})
	for stage, found := range mapped {
		if !found {
			t.Errorf("typed %s error is not mapped to failedStage", stage)
		}
	}
	if !emitsMappedError {
		t.Error("fallback error update is not emitted with failedStage")
	}
}

func isFacebookProviderStageSelector(expression ast.Expr) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Stage" {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	return ok && identifier.Name == "providerErr"
}

func facebookCaseAssignsProviderStage(statements []ast.Stmt) bool {
	for _, statement := range statements {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			continue
		}
		left, ok := assignment.Lhs[0].(*ast.Ident)
		call, callOK := assignment.Rhs[0].(*ast.CallExpr)
		if !ok || left.Name != "failedStage" || !callOK || len(call.Args) != 1 {
			continue
		}
		conversion, conversionOK := call.Fun.(*ast.Ident)
		if !conversionOK || conversion.Name != "string" || !isFacebookProviderStageSelector(call.Args[0]) {
			continue
		}
		return true
	}
	return false
}

func hasFacebookAppArgs(args []string, sequence ...string) bool {
	if len(sequence) == 0 || len(sequence) > len(args) {
		return false
	}
	for start := 0; start <= len(args)-len(sequence); start++ {
		matched := true
		for index := range sequence {
			if args[start+index] != sequence[index] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func validFacebookAppBrief() facebookcompanion.Brief {
	return facebookcompanion.Brief{
		Topic:          "Evidence-based marketing workshop",
		Audience:       "Small online shop owners",
		Objective:      "Invite readers to request the course outline",
		Offer:          "Free course outline",
		BrandVoice:     "Clear, practical, and friendly teacher",
		Language:       "Thai",
		ProductDetails: "The workshop contains eight lessons.",
		Evidence: []facebookcompanion.EvidenceSource{{
			ID: "course-outline", Title: "Published course outline", URL: "https://example.com/course", Notes: "The outline lists eight lessons.",
		}},
	}
}

func validFacebookAppPack() facebookcompanion.ContentPack {
	return facebookcompanion.ContentPack{
		Hooks:      []string{"Start with the customer", "A clear plan saves guesswork", "Eight practical lessons"},
		LongPost:   "A practical long-form post grounded in the supplied course outline.",
		ShortPost:  "A concise post grounded in the course outline.",
		ReelScript: "Hook: start with the customer. Body: explain the plan. CTA: request the outline.",
		CarouselSlides: []facebookcompanion.CarouselSlide{
			{Headline: "The challenge", Body: "Marketing without a clear customer wastes effort."},
			{Headline: "The principle", Body: "Define the audience and objective before writing."},
			{Headline: "The next step", Body: "Review the eight-lesson course outline."},
		},
		CTA:          "Request the free course outline.",
		FirstComment: "The outline describes all eight lessons.",
		ReplyBank: []facebookcompanion.Reply{
			{Intent: "Ask about topics", Reply: "The outline lists the topic of each lesson."},
			{Intent: "Ask about format", Reply: "Please review the outline for the verified format details."},
			{Intent: "Request details", Reply: "I can send the course outline for your review."},
		},
		ComplianceNotes: []string{"Confirm any schedule or price before publishing."},
	}
}
