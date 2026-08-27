package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/versioninfo"
)

const maxResponseBytes = 16 << 20

type Client struct {
	httpClient *http.Client
	endpoint   string
}

func New(httpClient *http.Client, endpoint string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 3 * time.Minute}
	}
	isolatedClient := *httpClient
	// Never follow redirects for an authenticated request. This prevents the
	// x-goog-api-key header from being replayed to a different origin or across
	// an HTTPS-to-HTTP downgrade. A 3xx is returned as an actionable API error.
	isolatedClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = domain.DefaultBaseURL
	}
	return &Client{httpClient: &isolatedClient, endpoint: endpoint}
}

type interactionRequest struct {
	Model             string            `json:"model"`
	Input             string            `json:"input"`
	SystemInstruction string            `json:"system_instruction"`
	Store             bool              `json:"store"`
	ResponseFormat    responseFormat    `json:"response_format"`
	Tools             []interactionTool `json:"tools,omitempty"`
}

type responseFormat struct {
	Type     string          `json:"type"`
	MIMEType string          `json:"mime_type"`
	Schema   json.RawMessage `json:"schema"`
}

type interactionTool struct {
	Type string `json:"type"`
}

type interactionError struct {
	Code    any    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

type interactionResponse struct {
	Status string             `json:"status"`
	Model  string             `json:"model"`
	Error  *interactionError  `json:"error,omitempty"`
	Errors []interactionError `json:"errors,omitempty"`
	Steps  []struct {
		Type    string `json:"type"`
		Content []struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Annotations []struct {
				Type  string `json:"type"`
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"annotations"`
		} `json:"content"`
	} `json:"steps"`
	Usage struct {
		TotalInputTokens   int `json:"total_input_tokens"`
		TotalOutputTokens  int `json:"total_output_tokens"`
		TotalThoughtTokens int `json:"total_thought_tokens"`
		TotalTokens        int `json:"total_tokens"`
	} `json:"usage"`
}

func (client *Client) Generate(
	ctx context.Context,
	apiKey string,
	settings domain.ProviderSettings,
	preview domain.PromptPreview,
) (domain.GeneratedContent, []domain.GroundingSource, domain.Usage, string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", errors.New("Gemini API key is required")
	}
	settings = domain.NormalizeSettings(settings)
	if err := domain.ValidateSettings(settings); err != nil {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", err
	}
	if !json.Valid([]byte(preview.SchemaJSON)) {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", errors.New("prompt response schema is invalid JSON")
	}

	payload := interactionRequest{
		Model:             settings.Model,
		Input:             preview.User,
		SystemInstruction: preview.System,
		Store:             false,
		ResponseFormat: responseFormat{
			Type:     "text",
			MIMEType: "application/json",
			Schema:   json.RawMessage(preview.SchemaJSON),
		},
	}
	if settings.UseGrounding {
		payload.Tools = []interactionTool{{Type: "google_search"}, {Type: "url_context"}}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", fmt.Errorf("encode Gemini request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(body))
	if err != nil {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", fmt.Errorf("create Gemini request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("x-goog-api-key", apiKey)
	request.Header.Set("User-Agent", "ContentBlueprint/"+versioninfo.CurrentVersion)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", fmt.Errorf("call Gemini Interactions API: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := readLimited(response.Body)
	if err != nil {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", fmt.Errorf("read Gemini response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", redactError(apiError(response.StatusCode, responseBody), apiKey)
	}

	content, sources, usage, modelName, err := ParseResponse(responseBody)
	if err != nil {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", redactError(err, apiKey)
	}
	if modelName == "" {
		modelName = settings.Model
	}
	return content, sources, usage, modelName, nil
}

func ParseResponse(data []byte) (domain.GeneratedContent, []domain.GroundingSource, domain.Usage, string, error) {
	var response interactionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", fmt.Errorf("Gemini returned invalid response JSON: %w", err)
	}
	if response.Status != "completed" {
		detail := interactionDiagnostics(response)
		if detail == "" {
			detail = "the response did not include an error message"
		}
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", fmt.Errorf("Gemini interaction status is %q: %s", response.Status, detail)
	}

	output := lastModelOutput(response)
	if strings.TrimSpace(output) == "" {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", errors.New("Gemini interaction completed without model text output")
	}
	content, err := decodeContent(output)
	if err != nil {
		return domain.GeneratedContent{}, nil, domain.Usage{}, "", fmt.Errorf("Gemini model output is not valid structured content: %w", err)
	}
	usage := domain.Usage{
		InputTokens:   response.Usage.TotalInputTokens,
		OutputTokens:  response.Usage.TotalOutputTokens,
		ThoughtTokens: response.Usage.TotalThoughtTokens,
		TotalTokens:   response.Usage.TotalTokens,
	}
	return content, collectGroundingSources(response), usage, response.Model, nil
}

func interactionDiagnostics(response interactionResponse) string {
	diagnostics := make([]interactionError, 0, len(response.Errors)+1)
	if response.Error != nil {
		diagnostics = append(diagnostics, *response.Error)
	}
	diagnostics = append(diagnostics, response.Errors...)
	parts := make([]string, 0, len(diagnostics))
	seen := make(map[string]struct{}, len(diagnostics))
	for _, diagnostic := range diagnostics {
		metadata := make([]string, 0, 2)
		if code := diagnosticCode(diagnostic.Code); code != "" {
			metadata = append(metadata, "code="+code)
		}
		if status := cleanDiagnosticText(diagnostic.Status, 120); status != "" {
			metadata = append(metadata, "status="+status)
		}
		message := cleanDiagnosticText(diagnostic.Message, 500)
		formatted := message
		if len(metadata) > 0 {
			formatted = "[" + strings.Join(metadata, ", ") + "]"
			if message != "" {
				formatted += " " + message
			}
		}
		if formatted == "" {
			continue
		}
		if _, exists := seen[formatted]; exists {
			continue
		}
		seen[formatted] = struct{}{}
		parts = append(parts, formatted)
	}
	return cleanDiagnosticText(strings.Join(parts, "; "), 1200)
}

func diagnosticCode(value any) string {
	switch code := value.(type) {
	case string:
		return cleanDiagnosticText(code, 120)
	case float64:
		return fmt.Sprintf("%g", code)
	case json.Number:
		return cleanDiagnosticText(code.String(), 120)
	default:
		return ""
	}
}

func cleanDiagnosticText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return value
}

func redactError(err error, secret string) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if secret = strings.TrimSpace(secret); secret != "" {
		message = strings.ReplaceAll(message, secret, "[REDACTED]")
	}
	return errors.New(message)
}

func lastModelOutput(response interactionResponse) string {
	last := ""
	for _, step := range response.Steps {
		if step.Type != "model_output" {
			continue
		}
		var builder strings.Builder
		for _, block := range step.Content {
			if block.Type == "text" {
				builder.WriteString(block.Text)
			}
		}
		if builder.Len() > 0 {
			last = builder.String()
		}
	}
	return last
}

func collectGroundingSources(response interactionResponse) []domain.GroundingSource {
	sources := make([]domain.GroundingSource, 0)
	for _, step := range response.Steps {
		if step.Type != "model_output" {
			continue
		}
		for _, block := range step.Content {
			if block.Type != "text" {
				continue
			}
			for _, annotation := range block.Annotations {
				if annotation.Type != "url_citation" {
					continue
				}
				sources = append(sources, domain.GroundingSource{
					Title: annotation.Title,
					URL:   annotation.URL,
				})
			}
		}
	}
	return domain.NormalizeGroundingSources(sources)
}

func decodeContent(output string) (domain.GeneratedContent, error) {
	decoder := json.NewDecoder(strings.NewReader(output))
	decoder.DisallowUnknownFields()
	var content domain.GeneratedContent
	if err := decoder.Decode(&content); err != nil {
		return domain.GeneratedContent{}, err
	}
	if err := ensureEOF(decoder); err != nil {
		return domain.GeneratedContent{}, err
	}
	missing := make([]string, 0, 6)
	if strings.TrimSpace(content.Title) == "" {
		missing = append(missing, "title")
	}
	if strings.TrimSpace(content.Slug) == "" {
		missing = append(missing, "slug")
	}
	if strings.TrimSpace(content.MetaTitle) == "" {
		missing = append(missing, "metaTitle")
	}
	if strings.TrimSpace(content.MetaDescription) == "" {
		missing = append(missing, "metaDescription")
	}
	if strings.TrimSpace(content.SummaryBox) == "" {
		missing = append(missing, "summaryBox")
	}
	if strings.TrimSpace(content.MainContentHTML) == "" {
		missing = append(missing, "mainContentHtml")
	}
	if content.KeyTakeaways == nil {
		missing = append(missing, "keyTakeaways")
	}
	if content.FAQData == nil {
		missing = append(missing, "faqData")
	}
	if len(missing) > 0 {
		return domain.GeneratedContent{}, fmt.Errorf("missing or empty required fields: %s", strings.Join(missing, ", "))
	}
	for index, item := range content.KeyTakeaways {
		if strings.TrimSpace(item.Statement) == "" || item.SourceIDs == nil {
			return domain.GeneratedContent{}, fmt.Errorf("keyTakeaways[%d] is incomplete", index)
		}
	}
	for index, item := range content.FAQData {
		if strings.TrimSpace(item.Question) == "" || strings.TrimSpace(item.Answer) == "" || item.SourceIDs == nil {
			return domain.GeneratedContent{}, fmt.Errorf("faqData[%d] is incomplete", index)
		}
	}
	return content, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("unexpected data after JSON object")
	}
	return err
}

func readLimited(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseBytes {
		return nil, fmt.Errorf("response exceeds %d MiB limit", maxResponseBytes>>20)
	}
	return data, nil
}

func apiError(statusCode int, data []byte) error {
	var envelope struct {
		Error struct {
			Code    any    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	message := ""
	if json.Unmarshal(data, &envelope) == nil {
		message = strings.TrimSpace(envelope.Error.Message)
	}
	if message == "" {
		message = strings.TrimSpace(string(data))
	}
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 600 {
		message = message[:600] + "…"
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return fmt.Errorf("Gemini Interactions API returned HTTP %d: %s", statusCode, message)
}
