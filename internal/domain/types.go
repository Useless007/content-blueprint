package domain

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

const (
	DefaultProvider = "gemini"
	DefaultModel    = "gemini-3.7-flash"
	DefaultBaseURL  = "https://generativelanguage.googleapis.com/v1beta/interactions"
)

var evidenceSourceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

type EvidenceSource struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Notes string `json:"notes"`
}

type GroundingSource struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type ContentBrief struct {
	Keyword                string           `json:"keyword"`
	Audience               string           `json:"audience"`
	Intent                 string           `json:"intent"`
	Objective              string           `json:"objective"`
	BrandVoice             string           `json:"brandVoice"`
	Language               string           `json:"language"`
	AdditionalInstructions string           `json:"additionalInstructions"`
	Evidence               []EvidenceSource `json:"evidence"`
}

type ProviderSettings struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	UseGrounding bool   `json:"useGrounding"`
	BaseURL      string `json:"baseUrl"`
}

type GenerationRequest struct {
	Brief    ContentBrief     `json:"brief"`
	Settings ProviderSettings `json:"settings"`
	APIKey   string           `json:"apiKey,omitempty"`
}

type GeneratedContent struct {
	Title           string        `json:"title"`
	Slug            string        `json:"slug"`
	MetaTitle       string        `json:"metaTitle"`
	MetaDescription string        `json:"metaDescription"`
	SummaryBox      string        `json:"summaryBox"`
	MainContentHTML string        `json:"mainContentHtml"`
	KeyTakeaways    []KeyTakeaway `json:"keyTakeaways"`
	FAQData         []FAQItem     `json:"faqData"`
}

type KeyTakeaway struct {
	Statement string   `json:"statement"`
	SourceIDs []string `json:"sourceIds"`
}

type FAQItem struct {
	Question  string   `json:"question"`
	Answer    string   `json:"answer"`
	SourceIDs []string `json:"sourceIds"`
}

type QualityCheck struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type QualityReport struct {
	Score          int            `json:"score"`
	Checks         []QualityCheck `json:"checks"`
	WordCount      int            `json:"wordCount"`
	SourceCoverage int            `json:"sourceCoverage"`
}

type PromptPreview struct {
	System     string `json:"system"`
	User       string `json:"user"`
	SchemaJSON string `json:"schemaJson"`
}

type Usage struct {
	InputTokens   int `json:"inputTokens"`
	OutputTokens  int `json:"outputTokens"`
	ThoughtTokens int `json:"thoughtTokens"`
	TotalTokens   int `json:"totalTokens"`
}

type GenerationResult struct {
	Content          GeneratedContent  `json:"content"`
	GroundingSources []GroundingSource `json:"groundingSources"`
	Quality          QualityReport     `json:"quality"`
	Prompt           PromptPreview     `json:"prompt"`
	Usage            Usage             `json:"usage"`
	Model            string            `json:"model"`
}

type Project struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Brief            ContentBrief      `json:"brief"`
	Content          *GeneratedContent `json:"content"`
	GroundingSources []GroundingSource `json:"groundingSources"`
	Quality          *QualityReport    `json:"quality"`
	Settings         ProviderSettings  `json:"settings"`
	CreatedAt        string            `json:"createdAt"`
	UpdatedAt        string            `json:"updatedAt"`
}

type ProjectSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Keyword   string `json:"keyword"`
	Score     int    `json:"score"`
	UpdatedAt string `json:"updatedAt"`
}

type BootstrapData struct {
	Settings              ProviderSettings `json:"settings"`
	Projects              []ProjectSummary `json:"projects"`
	APIKeyFromEnvironment bool             `json:"apiKeyFromEnvironment"`
}

func DefaultSettings() ProviderSettings {
	return ProviderSettings{
		Provider:     DefaultProvider,
		Model:        DefaultModel,
		UseGrounding: true,
		BaseURL:      DefaultBaseURL,
	}
}

func NormalizeSettings(settings ProviderSettings) ProviderSettings {
	settings.Provider = strings.ToLower(strings.TrimSpace(settings.Provider))
	settings.Model = strings.TrimSpace(settings.Model)
	settings.BaseURL = strings.TrimSpace(settings.BaseURL)
	if settings.Provider == "" {
		settings.Provider = DefaultProvider
	}
	if settings.Model == "" {
		settings.Model = DefaultModel
	}
	if settings.BaseURL == "" {
		settings.BaseURL = DefaultBaseURL
	}
	return settings
}

func ValidateSettings(settings ProviderSettings) error {
	settings = NormalizeSettings(settings)
	if settings.Provider != DefaultProvider {
		return fmt.Errorf("unsupported provider %q; only Gemini is available", settings.Provider)
	}
	parsed, err := url.Parse(settings.BaseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "generativelanguage.googleapis.com" || parsed.Port() != "" {
		return fmt.Errorf("base URL must use the official Gemini HTTPS host generativelanguage.googleapis.com")
	}
	if parsed.User != nil {
		return fmt.Errorf("base URL must not contain credentials")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("base URL must not contain a query string or fragment")
	}
	if parsed.Path != "/v1beta/interactions" || (parsed.RawPath != "" && parsed.RawPath != "/v1beta/interactions") {
		return fmt.Errorf("base URL must point to /v1beta/interactions")
	}
	return nil
}

func ValidateBrief(brief ContentBrief) error {
	required := []struct {
		name  string
		value string
	}{
		{"keyword", brief.Keyword},
		{"audience", brief.Audience},
		{"intent", brief.Intent},
		{"objective", brief.Objective},
		{"language", brief.Language},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}

	seen := make(map[string]struct{}, len(brief.Evidence))
	for index, source := range brief.Evidence {
		id := strings.TrimSpace(source.ID)
		if id == "" {
			return fmt.Errorf("evidence source %d is missing an id", index+1)
		}
		if !ValidEvidenceSourceID(id) {
			return fmt.Errorf("evidence source id %q must start with an ASCII letter or number and contain only letters, numbers, dot, underscore, colon, or hyphen (maximum 128 characters)", id)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("evidence source id %q is duplicated", id)
		}
		seen[id] = struct{}{}
		if strings.TrimSpace(source.Title) == "" {
			return fmt.Errorf("evidence source %q is missing a title", id)
		}
		if source.URL != "" {
			if _, ok := NormalizeGroundingURL(source.URL); !ok {
				return fmt.Errorf("evidence source %q has an invalid URL", id)
			}
		}
	}
	return nil
}

func ValidEvidenceSourceID(id string) bool {
	return evidenceSourceIDPattern.MatchString(id)
}

// NormalizeGroundingURL accepts only absolute HTTP(S) URLs and returns a
// stable representation suitable for exact de-duplication and persistence.
func NormalizeGroundingURL(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "\\") {
		return "", false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Opaque != "" || parsed.User != nil || parsed.Hostname() == "" {
		return "", false
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	hostname := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		parsed.Host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		parsed.Host = "[" + hostname + "]"
	} else {
		parsed.Host = hostname
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String(), true
}

func NormalizeGroundingSources(sources []GroundingSource) []GroundingSource {
	normalized := make([]GroundingSource, 0, len(sources))
	byURL := make(map[string]int, len(sources))
	for _, source := range sources {
		normalizedURL, ok := NormalizeGroundingURL(source.URL)
		if !ok {
			continue
		}
		title := strings.TrimSpace(source.Title)
		if index, exists := byURL[normalizedURL]; exists {
			if normalized[index].Title == "" && title != "" {
				normalized[index].Title = title
			}
			continue
		}
		byURL[normalizedURL] = len(normalized)
		normalized = append(normalized, GroundingSource{Title: title, URL: normalizedURL})
	}
	return normalized
}
