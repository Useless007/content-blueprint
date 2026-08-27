package domain

import (
	"strings"
	"testing"
)

func TestValidateSettingsAllowsOfficialGeminiEndpoint(t *testing.T) {
	settings := DefaultSettings()
	if err := ValidateSettings(settings); err != nil {
		t.Fatalf("ValidateSettings(default) error = %v", err)
	}
}

func TestValidateSettingsRejectsUntrustedAPIEndpoints(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{name: "plain HTTP", baseURL: "http://generativelanguage.googleapis.com/v1beta/interactions", want: "official Gemini HTTPS host"},
		{name: "lookalike host", baseURL: "https://generativelanguage.googleapis.com.evil.example/v1beta/interactions", want: "official Gemini HTTPS host"},
		{name: "subdomain", baseURL: "https://attacker.generativelanguage.googleapis.com/v1beta/interactions", want: "official Gemini HTTPS host"},
		{name: "custom port", baseURL: "https://generativelanguage.googleapis.com:8443/v1beta/interactions", want: "official Gemini HTTPS host"},
		{name: "userinfo", baseURL: "https://user@generativelanguage.googleapis.com/v1beta/interactions", want: "credentials"},
		{name: "query", baseURL: "https://generativelanguage.googleapis.com/v1beta/interactions?key=elsewhere", want: "query string or fragment"},
		{name: "fragment", baseURL: "https://generativelanguage.googleapis.com/v1beta/interactions#fragment", want: "query string or fragment"},
		{name: "wrong path", baseURL: "https://generativelanguage.googleapis.com/v1beta/models", want: "/v1beta/interactions"},
		{name: "encoded path", baseURL: "https://generativelanguage.googleapis.com/v1beta/%69nteractions", want: "/v1beta/interactions"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := DefaultSettings()
			settings.BaseURL = test.baseURL
			err := ValidateSettings(settings)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateSettings(%q) error = %v, want substring %q", test.baseURL, err, test.want)
			}
		})
	}
}

func TestNormalizeGroundingSourcesFiltersNormalizesAndDeduplicates(t *testing.T) {
	got := NormalizeGroundingSources([]GroundingSource{
		{URL: " HTTPS://Example.COM:443 ", Title: ""},
		{URL: "https://example.com/", Title: "First non-empty title"},
		{URL: "https://example.com/", Title: "Later title"},
		{URL: "http://Example.org:80/path", Title: "HTTP source"},
		{URL: "ftp://example.net/file", Title: "Invalid scheme"},
		{URL: "//example.net/file", Title: "Protocol relative"},
		{URL: "https://user:pass@example.net/private", Title: "Credentials"},
		{URL: `https://example.net/\evil`, Title: "Backslash path"},
	})

	if len(got) != 2 {
		t.Fatalf("NormalizeGroundingSources() = %#v, want two valid unique sources", got)
	}
	if got[0].URL != "https://example.com/" || got[0].Title != "First non-empty title" {
		t.Errorf("first source = %#v", got[0])
	}
	if got[1].URL != "http://example.org/path" || got[1].Title != "HTTP source" {
		t.Errorf("second source = %#v", got[1])
	}
}

func TestValidateBriefRejectsEvidenceIDThatCannotSurviveHTMLMarker(t *testing.T) {
	brief := ContentBrief{
		Keyword: "keyword", Audience: "audience", Intent: "intent", Objective: "objective", Language: "Thai",
		Evidence: []EvidenceSource{{ID: "source with spaces", Title: "Source"}},
	}
	err := ValidateBrief(brief)
	if err == nil || !strings.Contains(err.Error(), "contain only") {
		t.Fatalf("ValidateBrief() error = %v, want safe source-ID validation", err)
	}
}
