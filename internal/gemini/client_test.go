package gemini

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/versioninfo"
)

func TestGenerateSendsStructuredInteractionAndParsesOutput(t *testing.T) {
	content := validContent()
	modelJSON, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	requestData := make(chan map[string]any, 1)
	headerData := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		data, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			t.Errorf("read request: %v", readErr)
		}
		var decoded map[string]any
		if decodeErr := json.Unmarshal(data, &decoded); decodeErr != nil {
			t.Errorf("decode request: %v", decodeErr)
		}
		requestData <- decoded
		headerData <- request.Header.Clone()
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"status": "completed",
			"model":  "gemini-3.7-flash",
			"steps": []any{
				map[string]any{
					"type": "google_search_result",
					"content": []any{map[string]any{
						"type": "text",
						"text": "tool widget",
						"annotations": []any{map[string]any{
							"type": "url_citation", "title": "Ignore tool result", "url": "https://tool.example/result",
						}},
					}},
					"result": map[string]any{"url": "https://widget.example/result"},
				},
				map[string]any{
					"type": "model_output",
					"content": []any{map[string]any{
						"type": "text",
						"text": string(modelJSON),
						"annotations": []any{
							map[string]any{"type": "url_citation", "title": "", "url": " https://EXAMPLE.com:443/article "},
							map[string]any{"type": "url_citation", "title": "First useful title", "url": "https://example.com/article"},
							map[string]any{"type": "url_citation", "title": "Later title", "url": "https://example.com/article"},
							map[string]any{"type": "url_citation", "title": "HTTP source", "url": "http://example.org/source"},
							map[string]any{"type": "url_citation", "title": "Invalid", "url": "ftp://example.net/file"},
							map[string]any{"type": "file_citation", "title": "Wrong type", "url": "https://file.example/document"},
						},
					}},
				},
			},
			"usage": map[string]any{
				"total_input_tokens":   101,
				"total_output_tokens":  202,
				"total_thought_tokens": 33,
				"total_tokens":         336,
			},
		})
	}))
	defer server.Close()

	settings := domain.DefaultSettings()
	client := New(server.Client(), server.URL)
	got, sources, usage, model, err := client.Generate(context.Background(), "session-secret", settings, domain.PromptPreview{
		System:     "system",
		User:       "user",
		SchemaJSON: `{"type":"object"}`,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got.Title != content.Title || model != "gemini-3.7-flash" {
		t.Errorf("Generate() content/model = %#v, %q", got, model)
	}
	if len(sources) != 2 || sources[0].URL != "https://example.com/article" || sources[0].Title != "First useful title" ||
		sources[1].URL != "http://example.org/source" {
		t.Errorf("grounding sources = %#v", sources)
	}
	if usage.InputTokens != 101 || usage.OutputTokens != 202 || usage.ThoughtTokens != 33 || usage.TotalTokens != 336 {
		t.Errorf("usage = %#v", usage)
	}

	payload := <-requestData
	headers := <-headerData
	if headers.Get("x-goog-api-key") != "session-secret" {
		t.Errorf("x-goog-api-key = %q", headers.Get("x-goog-api-key"))
	}
	if headers.Get("User-Agent") != "ContentBlueprint/"+versioninfo.CurrentVersion {
		t.Errorf("User-Agent = %q", headers.Get("User-Agent"))
	}
	if payload["model"] != domain.DefaultModel || payload["input"] != "user" || payload["system_instruction"] != "system" {
		t.Errorf("unexpected request core fields: %#v", payload)
	}
	if _, exists := payload["generation_config"]; exists {
		t.Errorf("request contains generation_config/sampling parameters: %#v", payload)
	}
	format, ok := payload["response_format"].(map[string]any)
	if !ok || format["type"] != "text" || format["mime_type"] != "application/json" {
		t.Errorf("response_format = %#v", payload["response_format"])
	}
	tools, ok := payload["tools"].([]any)
	if !ok || len(tools) != 2 {
		t.Fatalf("tools = %#v, want google_search and url_context", payload["tools"])
	}
	if tools[0].(map[string]any)["type"] != "google_search" || tools[1].(map[string]any)["type"] != "url_context" {
		t.Errorf("tools = %#v", tools)
	}
}

func TestGenerateOmitsToolsWhenGroundingDisabled(t *testing.T) {
	payloads := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(request.Body).Decode(&payload)
		payloads <- payload
		modelJSON, _ := json.Marshal(validContent())
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"status": "completed",
			"steps":  []any{map[string]any{"type": "model_output", "content": []any{map[string]any{"type": "text", "text": string(modelJSON)}}}},
		})
	}))
	defer server.Close()

	settings := domain.DefaultSettings()
	settings.UseGrounding = false
	_, _, _, _, err := New(server.Client(), server.URL).Generate(context.Background(), "key", settings, domain.PromptPreview{
		System: "s", User: "u", SchemaJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := (<-payloads)["tools"]; exists {
		t.Fatal("request should omit tools when grounding is disabled")
	}
}

func TestParseResponseFailureModes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "failed status with numeric error code",
			body: `{"status":"failed","error":{"code":429,"message":"quota exhausted","status":"RESOURCE_EXHAUSTED"}}`,
			want: "quota exhausted",
		},
		{
			name: "non completed status",
			body: `{"status":"in_progress"}`,
			want: `status is "in_progress"`,
		},
		{
			name: "missing output",
			body: `{"status":"completed","steps":[]}`,
			want: "without model text output",
		},
		{
			name: "invalid model JSON",
			body: `{"status":"completed","steps":[{"type":"model_output","content":[{"type":"text","text":"not-json"}]}]}`,
			want: "not valid structured content",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, _, _, err := ParseResponse([]byte(test.body))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ParseResponse() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestParseResponseCollectsSingularAndArrayDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "failed with legacy and resource diagnostics",
			body: `{
				"status":"failed",
				"error":{"code":429,"status":"RESOURCE_EXHAUSTED","message":"legacy quota message"},
				"errors":[
					{"code":"tool_failure","status":"FAILED_PRECONDITION","message":"search tool failed"},
					{"code":"output_invalid","message":"structured output was invalid"}
				]
			}`,
			want: []string{`status is "failed"`, "code=429", "status=RESOURCE_EXHAUSTED", "legacy quota message", "code=tool_failure", "search tool failed", "code=output_invalid", "structured output was invalid"},
		},
		{
			name: "incomplete with diagnostics array",
			body: `{"status":"incomplete","errors":[{"code":"max_output_tokens","message":"Output token limit reached"}]}`,
			want: []string{`status is "incomplete"`, "code=max_output_tokens", "Output token limit reached"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, _, _, err := ParseResponse([]byte(test.body))
			if err == nil {
				t.Fatal("ParseResponse() error = nil")
			}
			for _, want := range test.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("ParseResponse() error = %q, want substring %q", err, want)
				}
			}
		})
	}
}

func TestGenerateRedactsAPIKeyFromInteractionDiagnostics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":"failed","errors":[{"code":"upstream","message":"request session-secret could not be processed"}]}`))
	}))
	defer server.Close()
	settings := domain.DefaultSettings()
	_, _, _, _, err := New(server.Client(), server.URL).Generate(context.Background(), "session-secret", settings, domain.PromptPreview{SchemaJSON: `{}`})
	if err == nil {
		t.Fatal("Generate() error = nil")
	}
	if strings.Contains(err.Error(), "session-secret") || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Errorf("Generate() error did not redact API key: %q", err)
	}
}

func TestGenerateReturnsActionableHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"error":{"code":401,"message":"API key is invalid"}}`))
	}))
	defer server.Close()
	settings := domain.DefaultSettings()
	_, _, _, _, err := New(server.Client(), server.URL).Generate(context.Background(), "bad", settings, domain.PromptPreview{SchemaJSON: `{}`})
	if err == nil || !strings.Contains(err.Error(), "HTTP 401: API key is invalid") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateDoesNotFollowRedirectWithAPIKey(t *testing.T) {
	targetCalled := make(chan bool, 1)
	target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		targetCalled <- true
		if request.Header.Get("x-goog-api-key") != "" {
			t.Errorf("redirect target received x-goog-api-key")
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", target.URL)
		writer.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()

	settings := domain.DefaultSettings()
	_, _, _, _, err := New(redirector.Client(), redirector.URL).Generate(context.Background(), "must-not-leak", settings, domain.PromptPreview{SchemaJSON: `{}`})
	if err == nil || !strings.Contains(err.Error(), "HTTP 307") {
		t.Fatalf("Generate() redirect error = %v, want HTTP 307", err)
	}
	select {
	case <-targetCalled:
		t.Fatal("redirect target was called")
	default:
	}
}

func validContent() domain.GeneratedContent {
	return domain.GeneratedContent{
		Title:           "A useful title",
		Slug:            "a-useful-title",
		MetaTitle:       "A useful title for search",
		MetaDescription: "A useful description of the article.",
		SummaryBox:      "A short summary.",
		MainContentHTML: "<h2>Overview</h2><p>Useful content.</p>",
		KeyTakeaways:    []domain.KeyTakeaway{{Statement: "Takeaway", SourceIDs: []string{}}},
		FAQData:         []domain.FAQItem{{Question: "Question?", Answer: "Answer.", SourceIDs: []string{}}},
	}
}
