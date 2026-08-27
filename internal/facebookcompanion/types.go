package facebookcompanion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	StateVersion        = 1
	MaxBriefJSONBytes   = 200_000
	MaxContentPackBytes = 900_000
)

// EvidenceSource is user-supplied source material. Notes are the only part
// that may be treated as evidence without independently opening the URL.
type EvidenceSource struct {
	ID    string `json:"id" jsonschema:"short unique source identifier"`
	Title string `json:"title" jsonschema:"human-readable source title"`
	URL   string `json:"url,omitempty" jsonschema:"optional absolute HTTP or HTTPS source URL"`
	Notes string `json:"notes,omitempty" jsonschema:"factual notes that the content may rely on"`
}

// Brief is the exact handoff contract shared by the Chrome extension and the
// MCP server. It intentionally contains no Facebook cookies or page data.
type Brief struct {
	Topic                  string           `json:"topic" jsonschema:"campaign, product, service, or topic to write about"`
	Audience               string           `json:"audience" jsonschema:"specific intended Facebook audience"`
	Objective              string           `json:"objective" jsonschema:"desired reader action or communication outcome"`
	Offer                  string           `json:"offer,omitempty" jsonschema:"offer details; do not infer anything not written here"`
	BrandVoice             string           `json:"brandVoice,omitempty" jsonschema:"tone and voice constraints"`
	Language               string           `json:"language" jsonschema:"language requested for the output"`
	ProductDetails         string           `json:"productDetails,omitempty" jsonschema:"verified product or service details"`
	Evidence               []EvidenceSource `json:"evidence,omitempty" jsonschema:"optional evidence sources and factual notes"`
	AdditionalInstructions string           `json:"additionalInstructions,omitempty" jsonschema:"additional editorial constraints from the user"`
}

type CarouselSlide struct {
	Headline string `json:"headline" jsonschema:"short slide headline"`
	Body     string `json:"body" jsonschema:"useful plain-text slide body"`
}

type Reply struct {
	Intent string `json:"intent" jsonschema:"reader intent or likely question"`
	Reply  string `json:"reply" jsonschema:"helpful non-pressuring plain-text response"`
}

// ContentPack is plain text only. The human page admin decides what, if
// anything, gets inserted into Facebook and remains responsible for posting.
type ContentPack struct {
	Hooks           []string        `json:"hooks" jsonschema:"exactly three distinct opening hooks using different angles"`
	LongPost        string          `json:"longPost" jsonschema:"complete scannable Facebook post in plain text"`
	ShortPost       string          `json:"shortPost" jsonschema:"meaningfully shorter Facebook post in plain text"`
	ReelScript      string          `json:"reelScript" jsonschema:"shootable Reel script with spoken hook, body beats, and CTA"`
	CarouselSlides  []CarouselSlide `json:"carouselSlides" jsonschema:"three to ten ordered carousel slides"`
	CTA             string          `json:"cta" jsonschema:"one clear non-deceptive call to action"`
	FirstComment    string          `json:"firstComment" jsonschema:"useful first comment without fake engagement bait"`
	ReplyBank       []Reply         `json:"replyBank" jsonschema:"three to twelve replies covering likely reader intents"`
	ComplianceNotes []string        `json:"complianceNotes" jsonschema:"unsupported, risky, or ambiguous claims for human review; empty when none"`
}

type GroundingSource struct {
	Title string `json:"title,omitempty" jsonschema:"source title"`
	URL   string `json:"url" jsonschema:"absolute HTTP or HTTPS URL"`
}

type BriefSnapshot struct {
	Version       int       `json:"version"`
	BriefRevision string    `json:"briefRevision"`
	Brief         Brief     `json:"brief"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type PackSnapshot struct {
	Version          int               `json:"version"`
	BriefRevision    string            `json:"briefRevision"`
	Pack             ContentPack       `json:"pack"`
	GroundingSources []GroundingSource `json:"groundingSources,omitempty"`
	GeneratedBy      string            `json:"generatedBy,omitempty"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

func NormalizeBrief(input Brief) Brief {
	input.Topic = strings.TrimSpace(input.Topic)
	input.Audience = strings.TrimSpace(input.Audience)
	input.Objective = strings.TrimSpace(input.Objective)
	input.Offer = strings.TrimSpace(input.Offer)
	input.BrandVoice = strings.TrimSpace(input.BrandVoice)
	input.Language = strings.TrimSpace(input.Language)
	input.ProductDetails = strings.TrimSpace(input.ProductDetails)
	input.AdditionalInstructions = strings.TrimSpace(input.AdditionalInstructions)
	if input.Evidence == nil {
		input.Evidence = []EvidenceSource{}
	}
	for index := range input.Evidence {
		input.Evidence[index].ID = strings.TrimSpace(input.Evidence[index].ID)
		input.Evidence[index].Title = strings.TrimSpace(input.Evidence[index].Title)
		input.Evidence[index].URL = strings.TrimSpace(input.Evidence[index].URL)
		input.Evidence[index].Notes = strings.TrimSpace(input.Evidence[index].Notes)
	}
	return input
}

func ValidateBrief(input Brief) error {
	brief := NormalizeBrief(input)
	for _, required := range []struct {
		name  string
		value string
		max   int
	}{
		{"topic", brief.Topic, 300},
		{"audience", brief.Audience, 1_500},
		{"objective", brief.Objective, 1_500},
		{"language", brief.Language, 80},
	} {
		if required.value == "" {
			return fmt.Errorf("%s is required", required.name)
		}
		if len([]rune(required.value)) > required.max {
			return fmt.Errorf("%s exceeds %d characters", required.name, required.max)
		}
	}
	for _, optional := range []struct {
		name  string
		value string
		max   int
	}{
		{"offer", brief.Offer, 4_000},
		{"brandVoice", brief.BrandVoice, 2_000},
		{"productDetails", brief.ProductDetails, 12_000},
		{"additionalInstructions", brief.AdditionalInstructions, 8_000},
	} {
		if len([]rune(optional.value)) > optional.max {
			return fmt.Errorf("%s exceeds %d characters", optional.name, optional.max)
		}
	}
	if len(brief.Evidence) > 30 {
		return fmt.Errorf("evidence contains more than 30 sources")
	}
	seen := make(map[string]struct{}, len(brief.Evidence))
	for index, source := range brief.Evidence {
		if source.ID == "" || len(source.ID) > 128 || !validEvidenceID(source.ID) {
			return fmt.Errorf("evidence[%d].id is invalid", index)
		}
		if _, exists := seen[source.ID]; exists {
			return fmt.Errorf("evidence id %q is duplicated", source.ID)
		}
		seen[source.ID] = struct{}{}
		if source.Title == "" || len([]rune(source.Title)) > 500 {
			return fmt.Errorf("evidence[%d].title is required and must not exceed 500 characters", index)
		}
		if len([]rune(source.Notes)) > 12_000 {
			return fmt.Errorf("evidence[%d].notes exceeds 12000 characters", index)
		}
		if source.URL != "" {
			if _, err := normalizeHTTPURL(source.URL); err != nil {
				return fmt.Errorf("evidence[%d].url is invalid", index)
			}
		}
	}
	encoded, err := json.Marshal(brief)
	if err != nil {
		return fmt.Errorf("encode brief: %w", err)
	}
	if len(encoded) > MaxBriefJSONBytes {
		return fmt.Errorf("brief exceeds %d bytes", MaxBriefJSONBytes)
	}
	return nil
}

func BriefRevision(brief Brief) (string, error) {
	normalized := NormalizeBrief(brief)
	if err := ValidateBrief(normalized); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("encode brief revision: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func NormalizeContentPack(input ContentPack) ContentPack {
	trimSlice := func(values []string) []string {
		if values == nil {
			return []string{}
		}
		for index := range values {
			values[index] = strings.TrimSpace(values[index])
		}
		return values
	}
	input.Hooks = trimSlice(input.Hooks)
	input.LongPost = strings.TrimSpace(input.LongPost)
	input.ShortPost = strings.TrimSpace(input.ShortPost)
	input.ReelScript = strings.TrimSpace(input.ReelScript)
	input.CTA = strings.TrimSpace(input.CTA)
	input.FirstComment = strings.TrimSpace(input.FirstComment)
	input.ComplianceNotes = trimSlice(input.ComplianceNotes)
	if input.CarouselSlides == nil {
		input.CarouselSlides = []CarouselSlide{}
	}
	for index := range input.CarouselSlides {
		input.CarouselSlides[index].Headline = strings.TrimSpace(input.CarouselSlides[index].Headline)
		input.CarouselSlides[index].Body = strings.TrimSpace(input.CarouselSlides[index].Body)
	}
	if input.ReplyBank == nil {
		input.ReplyBank = []Reply{}
	}
	for index := range input.ReplyBank {
		input.ReplyBank[index].Intent = strings.TrimSpace(input.ReplyBank[index].Intent)
		input.ReplyBank[index].Reply = strings.TrimSpace(input.ReplyBank[index].Reply)
	}
	return input
}

func ValidateContentPack(input ContentPack) error {
	pack := NormalizeContentPack(input)
	if len(pack.Hooks) != 3 {
		return fmt.Errorf("hooks must contain exactly 3 items")
	}
	seenHooks := make(map[string]struct{}, 3)
	for index, hook := range pack.Hooks {
		if err := validateRequiredText(fmt.Sprintf("hooks[%d]", index), hook, 1_000); err != nil {
			return err
		}
		key := strings.ToLower(hook)
		if _, exists := seenHooks[key]; exists {
			return fmt.Errorf("hooks must contain 3 distinct items")
		}
		seenHooks[key] = struct{}{}
	}
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{"longPost", pack.LongPost, 30_000},
		{"shortPost", pack.ShortPost, 5_000},
		{"reelScript", pack.ReelScript, 15_000},
		{"cta", pack.CTA, 3_000},
		{"firstComment", pack.FirstComment, 5_000},
	} {
		if err := validateRequiredText(field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if len(pack.CarouselSlides) < 3 || len(pack.CarouselSlides) > 10 {
		return fmt.Errorf("carouselSlides must contain 3 to 10 items")
	}
	for index, slide := range pack.CarouselSlides {
		if err := validateRequiredText(fmt.Sprintf("carouselSlides[%d].headline", index), slide.Headline, 500); err != nil {
			return err
		}
		if err := validateRequiredText(fmt.Sprintf("carouselSlides[%d].body", index), slide.Body, 2_000); err != nil {
			return err
		}
	}
	if len(pack.ReplyBank) < 3 || len(pack.ReplyBank) > 12 {
		return fmt.Errorf("replyBank must contain 3 to 12 items")
	}
	for index, reply := range pack.ReplyBank {
		if err := validateRequiredText(fmt.Sprintf("replyBank[%d].intent", index), reply.Intent, 500); err != nil {
			return err
		}
		if err := validateRequiredText(fmt.Sprintf("replyBank[%d].reply", index), reply.Reply, 4_000); err != nil {
			return err
		}
	}
	if len(pack.ComplianceNotes) > 12 {
		return fmt.Errorf("complianceNotes must contain no more than 12 items")
	}
	for index, note := range pack.ComplianceNotes {
		if err := validateRequiredText(fmt.Sprintf("complianceNotes[%d]", index), note, 2_000); err != nil {
			return err
		}
	}
	encoded, err := json.Marshal(pack)
	if err != nil {
		return fmt.Errorf("encode content pack: %w", err)
	}
	if len(encoded) > MaxContentPackBytes {
		return fmt.Errorf("content pack exceeds %d bytes", MaxContentPackBytes)
	}
	return nil
}

func NormalizeGroundingSources(input []GroundingSource) ([]GroundingSource, error) {
	if len(input) > 30 {
		return nil, fmt.Errorf("groundingSources contains more than 30 items")
	}
	result := make([]GroundingSource, 0, len(input))
	seen := make(map[string]int, len(input))
	for index, source := range input {
		normalizedURL, err := normalizeHTTPURL(source.URL)
		if err != nil {
			return nil, fmt.Errorf("groundingSources[%d].url is invalid", index)
		}
		title := strings.TrimSpace(source.Title)
		if len([]rune(title)) > 500 {
			return nil, fmt.Errorf("groundingSources[%d].title exceeds 500 characters", index)
		}
		if existing, exists := seen[normalizedURL]; exists {
			if result[existing].Title == "" && title != "" {
				result[existing].Title = title
			}
			continue
		}
		seen[normalizedURL] = len(result)
		result = append(result, GroundingSource{Title: title, URL: normalizedURL})
	}
	return result, nil
}

func validateRequiredText(name, value string, maximum int) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must be non-empty text", name)
	}
	if len([]rune(value)) > maximum {
		return fmt.Errorf("%s exceeds %d characters", name, maximum)
	}
	return nil
}

func validEvidenceID(value string) bool {
	for index, character := range value {
		valid := (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			(index > 0 && (character == '_' || character == '.' || character == ':' || character == '-'))
		if !valid {
			return false
		}
	}
	return value != ""
}

func normalizeHTTPURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 4_096 || strings.Contains(value, "\\") {
		return "", fmt.Errorf("invalid URL")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Opaque != "" || parsed.User != nil || parsed.Hostname() == "" {
		return "", fmt.Errorf("invalid URL")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme")
	}
	parsed.Host = strings.ToLower(parsed.Host)
	if (parsed.Scheme == "https" && parsed.Port() == "443") || (parsed.Scheme == "http" && parsed.Port() == "80") {
		parsed.Host = parsed.Hostname()
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String(), nil
}
