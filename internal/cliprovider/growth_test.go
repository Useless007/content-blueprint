package cliprovider

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/workbench"
)

func TestGenerateGrowthSingleReusesSafeCodexArgvSchemaAndPromptBoundary(t *testing.T) {
	packJSON, err := json.Marshal(growthValidPack())
	if err != nil {
		t.Fatal(err)
	}
	var got Command
	runner := &fakeRunner{run: func(_ context.Context, command Command) (ProcessResult, error) {
		got = command
		if flagValue(t, command.Args, "--output-schema") == "" {
			t.Fatal("missing output schema")
		}
		schema, err := os.ReadFile(flagValue(t, command.Args, "--output-schema"))
		wantSchema, schemaErr := growthPackSchemaForBrief(growthValidBrief("offer-audience"))
		if err != nil || schemaErr != nil || string(schema) != wantSchema {
			t.Fatalf("Growth schema = %q, %v", schema, err)
		}
		if err := os.WriteFile(flagValue(t, command.Args, "--output-last-message"), packJSON, 0o600); err != nil {
			t.Fatal(err)
		}
		return ProcessResult{}, nil
	}}
	service := NewWithRunner(runner)
	pack, err := service.GenerateGrowth(context.Background(), growthValidBrief("offer-audience"), Options{Provider: ProviderCodex, Executable: "codex-test.exe", Workflow: WorkflowSingle, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pack, workbench.NormalizePack(growthValidPack())) {
		t.Fatalf("pack = %#v", pack)
	}
	joinedArgs := strings.Join(got.Args, " ")
	for _, required := range []string{"exec", "--ephemeral", "--sandbox", "read-only", "--output-schema", "--output-last-message"} {
		if !strings.Contains(joinedArgs, required) {
			t.Errorf("Codex argv missing %q: %q", required, got.Args)
		}
	}
	if strings.Contains(joinedArgs, "Eight-lesson") || !strings.Contains(string(got.Stdin), "UNTRUSTED_INPUT_JSON") || !strings.Contains(string(got.Stdin), "STYLE_CONTRACT") {
		t.Fatalf("prompt/argv security boundary failed: args=%q prompt=%q", got.Args, got.Stdin)
	}
}

func TestGenerateGrowthTeamUsesThreeIsolatedStagesAndProgress(t *testing.T) {
	strategy := GrowthStrategy{Objective: "Create a bounded offer plan.", AudienceInsight: "Shop owners need clarity.", Plan: []string{"Clarify the audience.", "Map evidence-backed value.", "Add review checks."}, EvidenceSourceIDs: []string{"outline"}, RiskControls: []string{"Do not promise outcomes."}}
	responses := []ProcessResult{claudeEnvelope(t, strategy), claudeEnvelope(t, growthValidPack()), claudeEnvelope(t, growthValidPack())}
	var mu sync.Mutex
	var commands []Command
	runner := &fakeRunner{run: func(_ context.Context, command Command) (ProcessResult, error) {
		mu.Lock()
		defer mu.Unlock()
		commands = append(commands, command)
		response := responses[len(commands)-1]
		return response, nil
	}}
	var progress []ProgressEvent
	service := NewWithRunner(runner)
	pack, err := service.GenerateGrowth(context.Background(), growthValidBrief("offer-audience"), Options{
		Provider: ProviderClaude, Executable: "claude-test.exe", Workflow: WorkflowTeam, Timeout: 2 * time.Second,
		Progress: func(event ProgressEvent) { progress = append(progress, event) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pack.Blocks) != 1 || len(commands) != 3 {
		t.Fatalf("pack/commands = %#v / %d", pack, len(commands))
	}
	brief := growthValidBrief("offer-audience")
	wantStrategySchema, _ := growthStrategySchemaForBrief(brief)
	wantPackSchema, _ := growthPackSchemaForBrief(brief)
	if flagValue(t, commands[0].Args, "--json-schema") != wantStrategySchema || flagValue(t, commands[1].Args, "--json-schema") != wantPackSchema || flagValue(t, commands[2].Args, "--json-schema") != wantPackSchema {
		t.Fatal("team stages did not receive their strict schemas")
	}
	if commands[0].Dir == commands[1].Dir || commands[1].Dir == commands[2].Dir || commands[0].Dir == commands[2].Dir {
		t.Fatalf("team stages shared a workspace: %#v", commands)
	}
	if strings.Contains(string(commands[0].Stdin), `"strategy"`) || !strings.Contains(string(commands[1].Stdin), `"strategy"`) || !strings.Contains(string(commands[2].Stdin), `"draft"`) {
		t.Fatal("team handoffs were not isolated and ordered")
	}
	wantStages := []Stage{StageStrategist, StageStrategist, StageCopywriter, StageCopywriter, StageReviewer, StageReviewer}
	wantStatus := []StageStatus{StageStarted, StageCompleted, StageStarted, StageCompleted, StageStarted, StageCompleted}
	if len(progress) != len(wantStages) {
		t.Fatalf("progress = %#v", progress)
	}
	for index := range progress {
		if progress[index].Stage != wantStages[index] || progress[index].Status != wantStatus[index] || progress[index].Workflow != WorkflowTeam {
			t.Fatalf("progress[%d] = %#v", index, progress[index])
		}
	}
}

func TestGenerateGrowthTeamRepairsOneSemanticallyInvalidProducerPack(t *testing.T) {
	brief := growthValidBrief("facebook-campaign")
	brief.Evidence = nil
	strategy := GrowthStrategy{
		Objective:         "Create a bounded campaign plan.",
		AudienceInsight:   "Page administrators need a reviewable draft.",
		Plan:              []string{"Clarify the offer.", "Draft the campaign.", "Add human review checks."},
		EvidenceSourceIDs: []string{},
		RiskControls:      []string{"Do not claim access to Facebook or customer lists."},
	}
	invalid := growthValidPack()
	invalid.Blocks[0].EvidenceBasis = workbench.BasisMixed
	invalid.Blocks[0].SourceIDs = []string{}
	valid := growthValidPack()
	valid.Blocks[0].EvidenceBasis = workbench.BasisUserInput
	valid.Blocks[0].SourceIDs = []string{}

	responses := []ProcessResult{
		claudeEnvelope(t, strategy),
		claudeEnvelope(t, invalid),
		claudeEnvelope(t, valid),
		claudeEnvelope(t, valid),
	}
	var commands []Command
	runner := &fakeRunner{run: func(_ context.Context, command Command) (ProcessResult, error) {
		commands = append(commands, command)
		return responses[len(commands)-1], nil
	}}

	pack, err := NewWithRunner(runner).GenerateGrowth(context.Background(), brief, Options{
		Provider:   ProviderClaude,
		Executable: "claude-test.exe",
		Workflow:   WorkflowTeam,
		Timeout:    time.Second,
	})
	if err != nil {
		t.Fatalf("one repairable semantic error should not abort the team: %v", err)
	}
	if err := workbench.ValidatePack(brief.PlaybookID, brief.Evidence, pack); err != nil {
		t.Fatalf("repaired pack is invalid: %v", err)
	}
	if len(commands) != 4 {
		t.Fatalf("commands = %d, want strategist + producer + one repair + reviewer", len(commands))
	}
	if !strings.Contains(string(commands[2].Stdin), "mixed evidence requires sourceIds") {
		t.Fatal("repair prompt does not contain the bounded validation reason")
	}
}

func TestGenerateGrowthCancellationStopsRunner(t *testing.T) {
	started := make(chan struct{})
	runner := &fakeRunner{run: func(ctx context.Context, _ Command) (ProcessResult, error) {
		close(started)
		<-ctx.Done()
		return ProcessResult{}, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		_, err := NewWithRunner(runner).GenerateGrowth(ctx, growthValidBrief("offer-audience"), Options{Provider: ProviderClaude, Executable: "claude-test.exe", Timeout: 10 * time.Second})
		errCh <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start")
	}
	cancel()
	select {
	case err := <-errCh:
		if !IsCode(err, CodeCancelled) {
			t.Fatalf("error = %v, want cancelled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runner did not stop")
	}
}

func TestGrowthFacebookPromptContainsCompetitiveAudienceAndStyleGuardrails(t *testing.T) {
	brief := growthValidBrief("facebook-campaign")
	prompt, err := buildGrowthPrompt("producer", brief, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	text := string(prompt)
	for _, required := range []string{"creative gap", "Audience Source Plan", "Do not scrape", "another page's followers", "lawful basis", "direct, concise, concrete Thai", "Avoid canned hype", "CURRENT_EVIDENCE_RULE", "CURRENT_BLOCK_RULE"} {
		if !strings.Contains(text, required) {
			t.Errorf("prompt missing %q", required)
		}
	}
}

func TestGrowthSchemasNarrowKindsAndEvidenceToCurrentBrief(t *testing.T) {
	brief := growthValidBrief("facebook-campaign")
	brief.Evidence = nil
	packSchemaJSON, err := growthPackSchemaForBrief(brief)
	if err != nil {
		t.Fatal(err)
	}
	strategySchemaJSON, err := growthStrategySchemaForBrief(brief)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(packSchemaJSON), &decoded); err != nil {
		t.Fatal(err)
	}
	blockProperties := decoded["properties"].(map[string]any)["blocks"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	kinds := blockProperties["kind"].(map[string]any)["enum"].([]any)
	bases := blockProperties["evidenceBasis"].(map[string]any)["enum"].([]any)
	for _, forbidden := range []string{"table", "code"} {
		for _, value := range kinds {
			if value == forbidden {
				t.Errorf("contextual Facebook schema still allows kind %s", forbidden)
			}
		}
	}
	for _, forbidden := range []string{"supplied_evidence", "mixed", "imported_metric"} {
		for _, value := range bases {
			if value == forbidden {
				t.Errorf("contextual Facebook schema still allows basis %s", forbidden)
			}
		}
	}
	if !strings.Contains(packSchemaJSON, `"maxItems":0`) || !strings.Contains(strategySchemaJSON, `"maxItems":0`) {
		t.Fatal("no-evidence schemas do not force empty source ID arrays")
	}
}

func growthValidBrief(playbookID string) workbench.GrowthBrief {
	playbook, _ := workbench.LookupPlaybook(playbookID)
	inputs := map[string]string{}
	for _, field := range playbook.Fields {
		if field.Required {
			inputs[field.Key] = "Verified user input"
			if field.InputType == "url" {
				inputs[field.Key] = "https://example.com/page"
			}
		}
	}
	return workbench.GrowthBrief{PlaybookID: playbookID, Language: "Thai", BrandVoice: "Direct", Inputs: inputs, Evidence: []domain.EvidenceSource{{ID: "outline", Title: "Outline", URL: "https://example.com/outline", Notes: "The outline lists eight lessons."}}}
}

func growthValidPack() workbench.GrowthPack {
	return workbench.GrowthPack{
		Title: "Growth plan", Summary: "A bounded plan based on supplied evidence.",
		Blocks:        []workbench.GrowthBlock{{ID: "plan", Title: "Plan", Purpose: "Explain the plan", Kind: workbench.BlockProse, Body: "Use the supplied outline.", Items: []workbench.BlockItem{}, Columns: []string{}, Rows: [][]string{}, Code: "", EvidenceBasis: workbench.BasisSuppliedEvidence, SourceIDs: []string{"outline"}}},
		OpenQuestions: []string{}, RiskFlags: []string{"Do not promise outcomes."}, ReviewChecks: []workbench.ReviewCheck{{Status: "review", Label: "Human review", Reason: "Verify facts before use."}},
	}
}
