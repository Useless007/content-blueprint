package facebookcompanion

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPWorkflowReadsBriefAndSavesPack(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	savedBrief, err := store.SaveBrief(validBrief())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := NewMCPServer(store).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "companion-test", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	})

	tools, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools.Tools))
	}
	briefResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "get_facebook_brief", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if briefResult.IsError || briefResult.StructuredContent == nil {
		t.Fatalf("unexpected brief result: %#v", briefResult)
	}

	arguments := toMap(t, SavePackInput{
		BriefRevision:    savedBrief.BriefRevision,
		Pack:             validPack(),
		GroundingSources: []GroundingSource{{Title: "หลักสูตร", URL: "https://example.com/course"}},
		GeneratedBy:      "Codex test",
	})
	saveResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "save_facebook_pack", Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	if saveResult.IsError || saveResult.StructuredContent == nil {
		t.Fatalf("unexpected save result: %#v", saveResult)
	}
	if _, err := store.LoadPack(); err != nil {
		t.Fatalf("saved pack was not persisted: %v", err)
	}
}

func toMap(t *testing.T, value any) map[string]any {
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
