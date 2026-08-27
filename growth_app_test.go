package main

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ContentBlueprint/internal/cliprovider"
	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/measurement"
	"ContentBlueprint/internal/salesops"
	"ContentBlueprint/internal/workbench"
)

func TestGenerateGrowthPackRejectsDuplicateRunBeforeMutatingBrief(t *testing.T) {
	app := newGrowthAppForTest(t, &facebookAppFakeRunner{paths: facebookAppFakePaths()})
	original, err := app.growth.store.SaveBrief(validGrowthAppBrief())
	if err != nil {
		t.Fatal(err)
	}
	_, finish, err := app.beginGrowthRun("duplicate-growth-run")
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	changed := validGrowthAppBrief()
	changed.Inputs["audience"] = "This must not replace the stored brief"
	if _, err := app.GenerateGrowthPack(GrowthGenerationRequest{RunID: "duplicate-growth-run", Provider: "claude", Workflow: "single", Brief: changed, TimeoutSec: 5}); err == nil {
		t.Fatal("duplicate Growth run was accepted")
	}
	stored, err := app.growth.store.LoadBrief()
	if err != nil {
		t.Fatal(err)
	}
	if stored.BriefRevision != original.BriefRevision || !reflect.DeepEqual(stored.Brief, original.Brief) {
		t.Fatalf("duplicate run mutated stored brief: got %#v, want %#v", stored, original)
	}
}

func TestGenerateGrowthPackMapsTypedTeamErrorToMatchingWorkerStage(t *testing.T) {
	strategyEnvelope, err := json.Marshal(map[string]any{
		"type": "result", "subtype": "success", "is_error": false,
		"structured_output": cliprovider.GrowthStrategy{
			Objective: "Create a bounded plan.", AudienceInsight: "Shop owners need clarity.",
			Plan: []string{"Clarify audience.", "Map evidence.", "Add review checks."}, EvidenceSourceIDs: []string{"outline"}, RiskControls: []string{"Do not promise outcomes."},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	runner := &facebookAppFakeRunner{
		paths: facebookAppFakePaths(),
		run: func(_ context.Context, command cliprovider.Command) (cliprovider.ProcessResult, error) {
			if command.Path != "fake-claude" || !hasFacebookAppArgs(command.Args, "-p") {
				return cliprovider.ProcessResult{}, errors.New("unexpected Growth CLI command")
			}
			if err := os.RemoveAll(filepath.Dir(command.Dir)); err != nil {
				return cliprovider.ProcessResult{}, err
			}
			return cliprovider.ProcessResult{Stdout: strategyEnvelope}, nil
		},
	}
	app := newGrowthAppForTest(t, runner)
	eventContext := &facebookEventCountingContext{Context: context.Background()}
	app.ctx = eventContext
	_, err = app.GenerateGrowthPack(GrowthGenerationRequest{RunID: "growth-copywriter-failure", Provider: "claude", Workflow: "team", Brief: validGrowthAppBrief(), TimeoutSec: 5})
	if err == nil {
		t.Fatal("Growth generation succeeded after removing its workflow root")
	}
	var providerErr *cliprovider.Error
	if !errors.As(err, &providerErr) || providerErr.Stage != cliprovider.StageCopywriter {
		t.Fatalf("error = %#v, want typed copywriter stage", err)
	}
	if got := eventContext.eventAttempts(); got != 6 {
		t.Fatalf("event attempts = %d, want queued(3)+strategist(2)+fallback(1)", got)
	}
	assertGrowthFallbackUsesTypedProviderStage(t)
}

func newGrowthAppForTest(t *testing.T, runner cliprovider.Runner) *App {
	t.Helper()
	store, err := workbench.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	leads, err := salesops.NewStore(store.Directory())
	if err != nil {
		t.Fatal(err)
	}
	experiments, err := measurement.NewStore(store.Directory())
	if err != nil {
		t.Fatal(err)
	}
	return &App{ctx: context.Background(), growth: &growthCoordinator{store: store, leads: leads, measurement: experiments, cli: cliprovider.NewWithRunner(runner), runs: make(map[string]context.CancelFunc)}}
}

func validGrowthAppBrief() workbench.GrowthBrief {
	return workbench.GrowthBrief{
		PlaybookID: "offer-audience", Language: "Thai", BrandVoice: "Direct",
		Inputs:   map[string]string{"offer": "Eight-lesson workshop", "audience": "Small shop owners", "problems": "Unclear positioning"},
		Evidence: []domain.EvidenceSource{{ID: "outline", Title: "Course outline", URL: "https://example.com/outline", Notes: "The outline lists eight lessons."}},
	}
}

func assertGrowthFallbackUsesTypedProviderStage(t *testing.T) {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), "growth_app.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var generate *ast.FuncDecl
	for _, declaration := range parsed.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "GenerateGrowthPack" {
			generate = function
			break
		}
	}
	if generate == nil {
		t.Fatal("GenerateGrowthPack was not found")
	}
	mapped := map[string]bool{"StageStrategist": false, "StageCopywriter": false, "StageReviewer": false}
	emitsStage := false
	ast.Inspect(generate.Body, func(node ast.Node) bool {
		if statement, ok := node.(*ast.SwitchStmt); ok && isFacebookProviderStageSelector(statement.Tag) {
			for _, item := range statement.Body.List {
				clause, ok := item.(*ast.CaseClause)
				if !ok || !growthCaseAssignsProviderStage(clause.Body) {
					continue
				}
				for _, expression := range clause.List {
					selector, ok := expression.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					identifier, identifierOK := selector.X.(*ast.Ident)
					if identifierOK && identifier.Name == "cliprovider" {
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
		callee, calleeOK := call.Fun.(*ast.Ident)
		stage, stageOK := call.Args[0].(*ast.Ident)
		status, statusOK := call.Args[1].(*ast.BasicLit)
		if calleeOK && callee.Name == "emit" && stageOK && stage.Name == "stage" && statusOK && status.Value == `"error"` {
			emitsStage = true
		}
		return true
	})
	for stage, found := range mapped {
		if !found {
			t.Errorf("typed %s error is not mapped", stage)
		}
	}
	if !emitsStage {
		t.Error("fallback error is not emitted with the mapped stage")
	}
}

func growthCaseAssignsProviderStage(statements []ast.Stmt) bool {
	for _, statement := range statements {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			continue
		}
		left, leftOK := assignment.Lhs[0].(*ast.Ident)
		call, callOK := assignment.Rhs[0].(*ast.CallExpr)
		if !leftOK || left.Name != "stage" || !callOK || len(call.Args) != 1 {
			continue
		}
		conversion, conversionOK := call.Fun.(*ast.Ident)
		if conversionOK && conversion.Name == "string" && isFacebookProviderStageSelector(call.Args[0]) {
			return true
		}
	}
	return false
}
