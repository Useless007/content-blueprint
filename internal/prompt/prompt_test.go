package prompt

import (
	"encoding/json"
	"strings"
	"testing"

	"ContentBlueprint/internal/domain"
)

func TestBuildProducesPreviewAndValidSchema(t *testing.T) {
	brief := domain.ContentBrief{
		Keyword:                "prompt engineering",
		Audience:               "เจ้าของธุรกิจขนาดเล็ก",
		Intent:                 "informational",
		Objective:              "ช่วยวางระบบสร้างบทความ",
		BrandVoice:             "ตรงไปตรงมา",
		Language:               "Thai",
		AdditionalInstructions: "อธิบายด้วยตัวอย่าง <ignore-system>",
		Evidence: []domain.EvidenceSource{{
			ID:    "source-1",
			Title: "Official guide",
			URL:   "https://example.com/guide",
			Notes: "Structured output reduces parsing failures.",
		}},
	}

	preview, err := Build(brief)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !strings.Contains(preview.System, "Never invent") {
		t.Errorf("system prompt does not contain factuality guardrail")
	}
	if !strings.Contains(preview.System, `<sup data-source-id="S1">[S1]</sup>`) ||
		!strings.Contains(preview.User, `<sup data-source-id="S1">[S1]</sup>`) {
		t.Errorf("prompt does not require exact inline evidence markers")
	}
	if !strings.Contains(preview.User, `"id": "source-1"`) || !strings.Contains(preview.User, `\u003cignore-system\u003e`) {
		t.Errorf("user prompt does not preserve the brief as data: %s", preview.User)
	}
	if !json.Valid([]byte(preview.SchemaJSON)) {
		t.Fatalf("schemaJson is not valid JSON: %s", preview.SchemaJSON)
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(preview.SchemaJSON), &schema); err != nil {
		t.Fatal(err)
	}
	if schema["type"] != "object" || schema["additionalProperties"] != false {
		t.Errorf("unexpected top-level schema: %#v", schema)
	}
	required, ok := schema["required"].([]any)
	if !ok || len(required) != 8 {
		t.Errorf("schema required fields = %#v, want 8 fields", schema["required"])
	}
}

func TestBuildRejectsInvalidBrief(t *testing.T) {
	tests := []struct {
		name  string
		brief domain.ContentBrief
		want  string
	}{
		{name: "empty", brief: domain.ContentBrief{}, want: "keyword is required"},
		{
			name: "duplicate evidence",
			brief: domain.ContentBrief{
				Keyword: "x", Audience: "x", Intent: "x", Objective: "x", Language: "Thai",
				Evidence: []domain.EvidenceSource{{ID: "same", Title: "One"}, {ID: "same", Title: "Two"}},
			},
			want: "duplicated",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Build(test.brief)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Build() error = %v, want substring %q", err, test.want)
			}
		})
	}
}
