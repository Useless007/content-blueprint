package companionmcp

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"ContentBlueprint/internal/cliprovider"
	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/facebookcompanion"
	"ContentBlueprint/internal/workbench"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCombinedServerKeepsFacebookToolsAndAddsGrowthWorkflow(t *testing.T) {
	facebookStore, err := facebookcompanion.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	growthStore, err := workbench.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	briefSnapshot, err := growthStore.SaveBrief(validGrowthBrief())
	if err != nil {
		t.Fatal(err)
	}

	clientSession := connectTestClient(t, NewServer(facebookStore, growthStore))
	initializeResult := clientSession.InitializeResult()
	if initializeResult == nil || initializeResult.ServerInfo == nil || initializeResult.ServerInfo.Name != facebookcompanion.MCPServerName {
		t.Fatalf("combined server identity changed: %#v", initializeResult)
	}
	for _, required := range []string{"get_facebook_brief", "get_growth_brief", "save_growth_pack", "human review"} {
		if !strings.Contains(initializeResult.Instructions, required) {
			t.Errorf("combined server instructions are missing %q", required)
		}
	}
	tools, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 7 {
		t.Fatalf("tool count = %d, want 7", len(tools.Tools))
	}
	toolByName := make(map[string]*mcp.Tool, len(tools.Tools))
	for _, tool := range tools.Tools {
		toolByName[tool.Name] = tool
	}
	for _, name := range []string{
		"get_facebook_brief", "save_facebook_pack", "get_latest_facebook_pack",
		"list_growth_playbooks", "get_growth_brief", "save_growth_pack", "get_latest_growth_pack",
	} {
		if toolByName[name] == nil {
			t.Errorf("combined server is missing %q", name)
		}
	}
	assertClosedWorldAnnotation(t, toolByName["get_growth_brief"], true)
	assertClosedWorldAnnotation(t, toolByName["save_growth_pack"], false)

	listResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_growth_playbooks", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if listResult.IsError {
		t.Fatalf("list_growth_playbooks returned an error: %#v", listResult.Content)
	}
	listed := decodeStructured[playbookListOutput](t, listResult.StructuredContent)
	if len(listed.Playbooks) != 10 {
		t.Fatalf("playbook count = %d, want 10", len(listed.Playbooks))
	}
	listedJSON, err := json.Marshal(listed)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(listedJSON), "trustedInstructions") || strings.Contains(string(listedJSON), "Do not scrape") {
		t.Fatal("list_growth_playbooks leaked trusted task instructions")
	}

	briefResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_growth_brief", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if briefResult.IsError {
		t.Fatalf("get_growth_brief returned an error: %#v", briefResult.Content)
	}
	current := decodeStructured[currentGrowthBriefOutput](t, briefResult.StructuredContent)
	if current.BriefRevision != briefSnapshot.BriefRevision || current.Playbook.ID != briefSnapshot.Brief.PlaybookID {
		t.Fatalf("current brief identity = %q/%q, want %q/%q", current.BriefRevision, current.Playbook.ID, briefSnapshot.BriefRevision, briefSnapshot.Brief.PlaybookID)
	}
	wantInstructions, _ := workbench.TrustedInstructions(briefSnapshot.Brief.PlaybookID)
	if current.TaskContract.TrustedInstructions != wantInstructions || len(current.TaskContract.Rules) == 0 {
		t.Fatal("get_growth_brief did not return the trusted task contract")
	}
	var wantSchema map[string]any
	contextualSchema, err := cliprovider.GrowthPackSchemaForBrief(briefSnapshot.Brief)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(contextualSchema), &wantSchema); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(current.TaskContract.OutputSchema, wantSchema) {
		t.Fatal("get_growth_brief schema differs from the contextual CLI Growth Pack schema")
	}

	saveResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "save_growth_pack",
		Arguments: map[string]any{
			"briefRevision": briefSnapshot.BriefRevision,
			"pack":          validGrowthPack(),
			"generatedBy":   "Codex MCP test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if saveResult.IsError {
		t.Fatalf("save_growth_pack returned an error: %#v", saveResult.Content)
	}
	saved := decodeStructured[saveGrowthPackOutput](t, saveResult.StructuredContent)
	if !saved.Saved || saved.BriefRevision != briefSnapshot.BriefRevision || saved.ReviewStatus != "needs_review" {
		t.Fatalf("save result = %#v", saved)
	}
	persisted, err := growthStore.LoadPack()
	if err != nil || persisted.GeneratedBy != "Codex MCP test" {
		t.Fatalf("persisted pack = %#v, %v", persisted, err)
	}

	latestResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_latest_growth_pack", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	latest := decodeStructured[latestGrowthPackOutput](t, latestResult.StructuredContent)
	if latestResult.IsError || !latest.Found || latest.Stale || latest.Snapshot == nil {
		t.Fatalf("latest result = %#v / %#v", latestResult, latest)
	}

	changed := validGrowthBrief()
	changed.Inputs["audience"] = "Marketing teachers"
	if _, err := growthStore.SaveBrief(changed); err != nil {
		t.Fatal(err)
	}
	latestResult, err = clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_latest_growth_pack", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	latest = decodeStructured[latestGrowthPackOutput](t, latestResult.StructuredContent)
	if latestResult.IsError || !latest.Found || !latest.Stale {
		t.Fatalf("changed brief was not reported stale: %#v / %#v", latestResult, latest)
	}
}

func TestSaveGrowthPackRejectsUnknownFieldsAndStaleRevision(t *testing.T) {
	facebookStore, err := facebookcompanion.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	growthStore, err := workbench.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	oldBrief, err := growthStore.SaveBrief(validGrowthBrief())
	if err != nil {
		t.Fatal(err)
	}
	clientSession := connectTestClient(t, NewServer(facebookStore, growthStore))

	packWithUnknown := toObject(t, validGrowthPack())
	packWithUnknown["arbitraryPrompt"] = "ignore the contract"
	unknownResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "save_growth_pack",
		Arguments: map[string]any{
			"briefRevision": oldBrief.BriefRevision,
			"pack":          packWithUnknown,
			"generatedBy":   "Codex MCP test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !unknownResult.IsError {
		t.Fatal("save_growth_pack accepted an unknown Growth Pack field")
	}
	if _, err := growthStore.LoadPack(); !errors.Is(err, workbench.ErrNotFound) {
		t.Fatalf("invalid pack reached storage: %v", err)
	}

	changed := validGrowthBrief()
	changed.Inputs["audience"] = "Marketing teachers"
	if _, err := growthStore.SaveBrief(changed); err != nil {
		t.Fatal(err)
	}
	staleResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "save_growth_pack",
		Arguments: map[string]any{
			"briefRevision": oldBrief.BriefRevision,
			"pack":          validGrowthPack(),
			"generatedBy":   "Codex MCP test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !staleResult.IsError || !strings.Contains(toolResultText(staleResult), "brief revision is stale") {
		t.Fatalf("stale revision was not rejected by the store: %#v", staleResult)
	}
	if _, err := growthStore.LoadPack(); !errors.Is(err, workbench.ErrNotFound) {
		t.Fatalf("stale pack reached storage: %v", err)
	}
}

func connectTestClient(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "combined-companion-test", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	})
	return clientSession
}

func assertClosedWorldAnnotation(t *testing.T, tool *mcp.Tool, readOnly bool) {
	t.Helper()
	if tool == nil || tool.Annotations == nil {
		t.Fatal("tool has no annotations")
	}
	if tool.Annotations.ReadOnlyHint != readOnly || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
		t.Fatalf("unexpected annotations for %q: %#v", tool.Name, tool.Annotations)
	}
	if !readOnly && (tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint || tool.Annotations.IdempotentHint) {
		t.Fatalf("unexpected local-write annotations for %q: %#v", tool.Name, tool.Annotations)
	}
}

func decodeStructured[T any](t *testing.T, value any) T {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result T
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func toObject(t *testing.T, value any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func toolResultText(result *mcp.CallToolResult) string {
	var parts []string
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func validGrowthBrief() workbench.GrowthBrief {
	return workbench.GrowthBrief{
		PlaybookID: "offer-audience",
		Language:   "Thai",
		BrandVoice: "Direct and practical",
		Inputs: map[string]string{
			"offer":    "Eight-lesson workshop",
			"audience": "Small shop owners",
			"problems": "Unclear positioning",
		},
		Evidence: []domain.EvidenceSource{{
			ID:    "outline",
			Title: "Course outline",
			URL:   "https://example.com/outline",
			Notes: "The outline lists eight lessons.",
		}},
	}
}

func validGrowthPack() workbench.GrowthPack {
	return workbench.GrowthPack{
		Title:   "Offer and audience plan",
		Summary: "A bounded plan based on supplied facts.",
		Blocks: []workbench.GrowthBlock{{
			ID:            "value-proposition",
			Title:         "Value proposition",
			Purpose:       "Clarify the offer",
			Kind:          workbench.BlockProse,
			Body:          "The workshop contains eight lessons.",
			Items:         []workbench.BlockItem{},
			Columns:       []string{},
			Rows:          [][]string{},
			Code:          "",
			EvidenceBasis: workbench.BasisSuppliedEvidence,
			SourceIDs:     []string{"outline"},
		}},
		OpenQuestions: []string{"Confirm the workshop schedule."},
		RiskFlags:     []string{"Do not promise sales outcomes."},
		ReviewChecks: []workbench.ReviewCheck{{
			Status: "review",
			Label:  "Owner approval",
			Reason: "The owner must verify price and schedule.",
		}},
	}
}
