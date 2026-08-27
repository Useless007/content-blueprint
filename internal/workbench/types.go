package workbench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"ContentBlueprint/internal/domain"
)

const (
	StateVersion       = 1
	MaxBriefJSONBytes  = 300_000
	MaxPackJSONBytes   = 1_000_000
	MaxEvidenceSources = 30
)

var objectIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

type GrowthBrief struct {
	PlaybookID string                  `json:"playbookId"`
	Language   string                  `json:"language"`
	BrandVoice string                  `json:"brandVoice,omitempty"`
	Inputs     map[string]string       `json:"inputs"`
	Evidence   []domain.EvidenceSource `json:"evidence"`
}

type EvidenceBasis string

const (
	BasisUserInput        EvidenceBasis = "user_input"
	BasisSuppliedEvidence EvidenceBasis = "supplied_evidence"
	BasisAIInference      EvidenceBasis = "ai_inference"
	BasisImportedMetric   EvidenceBasis = "imported_metric"
	BasisMixed            EvidenceBasis = "mixed"
)

type BlockKind string

const (
	BlockProse     BlockKind = "prose"
	BlockCards     BlockKind = "cards"
	BlockTable     BlockKind = "table"
	BlockSequence  BlockKind = "sequence"
	BlockChecklist BlockKind = "checklist"
	BlockTasks     BlockKind = "tasks"
	BlockCode      BlockKind = "code"
)

type BlockItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Note  string `json:"note,omitempty"`
}

type GrowthBlock struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Purpose       string        `json:"purpose"`
	Kind          BlockKind     `json:"kind"`
	Body          string        `json:"body"`
	Items         []BlockItem   `json:"items"`
	Columns       []string      `json:"columns"`
	Rows          [][]string    `json:"rows"`
	Code          string        `json:"code"`
	EvidenceBasis EvidenceBasis `json:"evidenceBasis"`
	SourceIDs     []string      `json:"sourceIds"`
}

type ReviewCheck struct {
	Status string `json:"status"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

type GrowthPack struct {
	Title         string        `json:"title"`
	Summary       string        `json:"summary"`
	Blocks        []GrowthBlock `json:"blocks"`
	OpenQuestions []string      `json:"openQuestions"`
	RiskFlags     []string      `json:"riskFlags"`
	ReviewChecks  []ReviewCheck `json:"reviewChecks"`
}

func NormalizeBrief(input GrowthBrief) GrowthBrief {
	input.PlaybookID = strings.TrimSpace(input.PlaybookID)
	input.Language = strings.TrimSpace(input.Language)
	input.BrandVoice = strings.TrimSpace(input.BrandVoice)
	normalizedInputs := make(map[string]string, len(input.Inputs))
	for key, value := range input.Inputs {
		cleanKey := strings.TrimSpace(key)
		normalizedInputs[cleanKey] = strings.TrimSpace(value)
	}
	input.Inputs = normalizedInputs
	if input.Evidence == nil {
		input.Evidence = []domain.EvidenceSource{}
	} else {
		input.Evidence = append([]domain.EvidenceSource(nil), input.Evidence...)
	}
	for index := range input.Evidence {
		input.Evidence[index].ID = strings.TrimSpace(input.Evidence[index].ID)
		input.Evidence[index].Title = strings.TrimSpace(input.Evidence[index].Title)
		input.Evidence[index].URL = strings.TrimSpace(input.Evidence[index].URL)
		if normalizedURL, ok := domain.NormalizeGroundingURL(input.Evidence[index].URL); ok {
			input.Evidence[index].URL = normalizedURL
		}
		input.Evidence[index].Notes = strings.TrimSpace(input.Evidence[index].Notes)
	}
	return input
}

func ValidateBrief(input GrowthBrief) error {
	seenInputKeys := make(map[string]struct{}, len(input.Inputs))
	for key := range input.Inputs {
		normalizedKey := strings.TrimSpace(key)
		if _, exists := seenInputKeys[normalizedKey]; exists {
			return fmt.Errorf("input field %q is duplicated after normalization", normalizedKey)
		}
		seenInputKeys[normalizedKey] = struct{}{}
	}
	brief := NormalizeBrief(input)
	playbook, ok := LookupPlaybook(brief.PlaybookID)
	if !ok {
		return fmt.Errorf("unknown playbookId %q", brief.PlaybookID)
	}
	if brief.Language == "" || len([]rune(brief.Language)) > 80 {
		return fmt.Errorf("language is required and must not exceed 80 characters")
	}
	if len([]rune(brief.BrandVoice)) > 2_000 {
		return fmt.Errorf("brandVoice exceeds 2000 characters")
	}
	allowed := make(map[string]FieldSpec, len(playbook.Fields))
	for _, field := range playbook.Fields {
		allowed[field.Key] = field
	}
	for key, value := range brief.Inputs {
		field, exists := allowed[key]
		if !exists {
			return fmt.Errorf("input field %q is not allowed for playbook %q", key, brief.PlaybookID)
		}
		if len([]rune(value)) > field.MaxChars {
			return fmt.Errorf("input field %q exceeds %d characters", key, field.MaxChars)
		}
		if field.InputType == "url" && value != "" {
			if _, ok := domain.NormalizeGroundingURL(value); !ok {
				return fmt.Errorf("input field %q must be an absolute HTTP(S) URL without credentials", key)
			}
		}
	}
	for _, field := range playbook.Fields {
		if field.Required && strings.TrimSpace(brief.Inputs[field.Key]) == "" {
			return fmt.Errorf("input field %q is required", field.Key)
		}
	}
	if len(brief.Evidence) > MaxEvidenceSources {
		return fmt.Errorf("evidence contains more than %d sources", MaxEvidenceSources)
	}
	seen := make(map[string]struct{}, len(brief.Evidence))
	for index, source := range brief.Evidence {
		if !domain.ValidEvidenceSourceID(source.ID) {
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
			normalized, ok := domain.NormalizeGroundingURL(source.URL)
			if !ok {
				return fmt.Errorf("evidence[%d].url is invalid", index)
			}
			brief.Evidence[index].URL = normalized
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

func BriefRevision(input GrowthBrief) (string, error) {
	if err := ValidateBrief(input); err != nil {
		return "", err
	}
	brief := NormalizeBrief(input)
	encoded, err := canonicalBriefJSON(brief)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func canonicalBriefJSON(brief GrowthBrief) ([]byte, error) {
	// encoding/json sorts string map keys, giving the revision a stable canonical form.
	return json.Marshal(brief)
}

func NormalizePack(input GrowthPack) GrowthPack {
	input.Title = strings.TrimSpace(input.Title)
	input.Summary = strings.TrimSpace(input.Summary)
	if input.Blocks == nil {
		input.Blocks = []GrowthBlock{}
	}
	if input.OpenQuestions == nil {
		input.OpenQuestions = []string{}
	}
	if input.RiskFlags == nil {
		input.RiskFlags = []string{}
	}
	if input.ReviewChecks == nil {
		input.ReviewChecks = []ReviewCheck{}
	}
	for index := range input.Blocks {
		block := &input.Blocks[index]
		block.ID = strings.TrimSpace(block.ID)
		block.Title = strings.TrimSpace(block.Title)
		block.Purpose = strings.TrimSpace(block.Purpose)
		block.Body = strings.TrimSpace(block.Body)
		block.Code = strings.TrimSpace(block.Code)
		if block.Items == nil {
			block.Items = []BlockItem{}
		}
		if block.Columns == nil {
			block.Columns = []string{}
		}
		if block.Rows == nil {
			block.Rows = [][]string{}
		}
		if block.SourceIDs == nil {
			block.SourceIDs = []string{}
		}
		for itemIndex := range block.Items {
			block.Items[itemIndex].Label = strings.TrimSpace(block.Items[itemIndex].Label)
			block.Items[itemIndex].Value = strings.TrimSpace(block.Items[itemIndex].Value)
			block.Items[itemIndex].Note = strings.TrimSpace(block.Items[itemIndex].Note)
		}
		for column := range block.Columns {
			block.Columns[column] = strings.TrimSpace(block.Columns[column])
		}
		for row := range block.Rows {
			for cell := range block.Rows[row] {
				block.Rows[row][cell] = strings.TrimSpace(block.Rows[row][cell])
			}
		}
		for source := range block.SourceIDs {
			block.SourceIDs[source] = strings.TrimSpace(block.SourceIDs[source])
		}
	}
	for index := range input.OpenQuestions {
		input.OpenQuestions[index] = strings.TrimSpace(input.OpenQuestions[index])
	}
	for index := range input.RiskFlags {
		input.RiskFlags[index] = strings.TrimSpace(input.RiskFlags[index])
	}
	for index := range input.ReviewChecks {
		input.ReviewChecks[index].Status = strings.TrimSpace(input.ReviewChecks[index].Status)
		input.ReviewChecks[index].Label = strings.TrimSpace(input.ReviewChecks[index].Label)
		input.ReviewChecks[index].Reason = strings.TrimSpace(input.ReviewChecks[index].Reason)
	}
	return input
}

var allowedKindsByPlaybook = map[string]map[BlockKind]bool{
	"offer-audience":                   {BlockProse: true, BlockCards: true, BlockTable: true, BlockChecklist: true},
	"facebook-campaign":                {BlockProse: true, BlockCards: true, BlockSequence: true, BlockChecklist: true, BlockTasks: true},
	"sales-reply":                      {BlockProse: true, BlockCards: true, BlockSequence: true, BlockTasks: true, BlockChecklist: true},
	"seo-topic-map":                    {BlockProse: true, BlockCards: true, BlockTable: true, BlockTasks: true},
	"seo-content-brief":                {BlockProse: true, BlockCards: true, BlockSequence: true, BlockChecklist: true, BlockTasks: true},
	"seo-onpage-review":                {BlockProse: true, BlockCards: true, BlockChecklist: true, BlockTasks: true, BlockTable: true},
	"seo-internal-links":               {BlockProse: true, BlockTable: true, BlockTasks: true, BlockChecklist: true},
	"seo-structured-data":              {BlockProse: true, BlockCode: true, BlockChecklist: true, BlockTasks: true},
	"seo-search-console-opportunities": {BlockProse: true, BlockCards: true, BlockTable: true, BlockTasks: true, BlockChecklist: true},
	"cross-channel-repurpose":          {BlockProse: true, BlockCards: true, BlockSequence: true, BlockChecklist: true, BlockTasks: true},
}

// AllowedBlockKinds returns a deterministic copy of the finite block kinds a
// playbook accepts. CLI adapters use this to narrow provider-side schemas;
// ValidatePack remains the final authority.
func AllowedBlockKinds(playbookID string) []BlockKind {
	allowed := allowedKindsByPlaybook[strings.TrimSpace(playbookID)]
	ordered := []BlockKind{BlockProse, BlockCards, BlockTable, BlockSequence, BlockChecklist, BlockTasks, BlockCode}
	result := make([]BlockKind, 0, len(allowed))
	for _, kind := range ordered {
		if allowed[kind] {
			result = append(result, kind)
		}
	}
	return result
}

func ValidatePack(playbookID string, evidence []domain.EvidenceSource, input GrowthPack) error {
	pack := NormalizePack(input)
	if _, ok := LookupPlaybook(playbookID); !ok {
		return fmt.Errorf("unknown playbookId %q", playbookID)
	}
	if err := requiredText("title", pack.Title, 1_000); err != nil {
		return err
	}
	if err := requiredText("summary", pack.Summary, 4_000); err != nil {
		return err
	}
	if len(pack.Blocks) < 1 || len(pack.Blocks) > 30 {
		return fmt.Errorf("blocks must contain 1 to 30 items")
	}
	validSources := make(map[string]struct{}, len(evidence))
	for _, source := range evidence {
		validSources[strings.TrimSpace(source.ID)] = struct{}{}
	}
	seenBlocks := make(map[string]struct{}, len(pack.Blocks))
	for index, block := range pack.Blocks {
		if !objectIDPattern.MatchString(block.ID) {
			return fmt.Errorf("blocks[%d].id is invalid", index)
		}
		if _, exists := seenBlocks[block.ID]; exists {
			return fmt.Errorf("block id %q is duplicated", block.ID)
		}
		seenBlocks[block.ID] = struct{}{}
		if !allowedKindsByPlaybook[playbookID][block.Kind] {
			return fmt.Errorf("blocks[%d].kind %q is not allowed for playbook %q", index, block.Kind, playbookID)
		}
		if err := requiredText(fmt.Sprintf("blocks[%d].title", index), block.Title, 1_000); err != nil {
			return err
		}
		if err := requiredText(fmt.Sprintf("blocks[%d].purpose", index), block.Purpose, 2_000); err != nil {
			return err
		}
		if err := validateBlockShape(index, block); err != nil {
			return err
		}
		switch block.EvidenceBasis {
		case BasisUserInput, BasisAIInference:
		case BasisSuppliedEvidence:
			if len(block.SourceIDs) == 0 {
				return fmt.Errorf("blocks[%d] with supplied_evidence requires sourceIds", index)
			}
		case BasisImportedMetric:
			if playbookID != "seo-search-console-opportunities" {
				return fmt.Errorf("blocks[%d] imported_metric is allowed only for the Search Console playbook", index)
			}
		case BasisMixed:
			if len(block.SourceIDs) == 0 {
				return fmt.Errorf("blocks[%d] with mixed evidence requires sourceIds", index)
			}
		default:
			return fmt.Errorf("blocks[%d].evidenceBasis is invalid", index)
		}
		seenSources := map[string]struct{}{}
		for _, sourceID := range block.SourceIDs {
			if _, exists := validSources[sourceID]; !exists {
				return fmt.Errorf("blocks[%d].sourceIds contains unknown source %q", index, sourceID)
			}
			if _, duplicate := seenSources[sourceID]; duplicate {
				return fmt.Errorf("blocks[%d].sourceIds contains duplicate %q", index, sourceID)
			}
			seenSources[sourceID] = struct{}{}
		}
	}
	if err := validateTextList("openQuestions", pack.OpenQuestions, 20, 2_000); err != nil {
		return err
	}
	if err := validateTextList("riskFlags", pack.RiskFlags, 20, 2_000); err != nil {
		return err
	}
	if len(pack.ReviewChecks) < 1 || len(pack.ReviewChecks) > 30 {
		return fmt.Errorf("reviewChecks must contain 1 to 30 items")
	}
	for index, check := range pack.ReviewChecks {
		if check.Status != "ready" && check.Status != "review" && check.Status != "blocked" {
			return fmt.Errorf("reviewChecks[%d].status is invalid", index)
		}
		if err := requiredText(fmt.Sprintf("reviewChecks[%d].label", index), check.Label, 1_000); err != nil {
			return err
		}
		if err := requiredText(fmt.Sprintf("reviewChecks[%d].reason", index), check.Reason, 2_000); err != nil {
			return err
		}
	}
	encoded, err := json.Marshal(pack)
	if err != nil {
		return err
	}
	if len(encoded) > MaxPackJSONBytes {
		return fmt.Errorf("growth pack exceeds %d bytes", MaxPackJSONBytes)
	}
	return nil
}

func validateBlockShape(index int, block GrowthBlock) error {
	if len(block.Items) > 50 || len(block.Columns) > 20 || len(block.Rows) > 100 {
		return fmt.Errorf("blocks[%d] contains too many items, columns, or rows", index)
	}
	for itemIndex, item := range block.Items {
		if err := requiredText(fmt.Sprintf("blocks[%d].items[%d].label", index, itemIndex), item.Label, 1_000); err != nil {
			return err
		}
		if err := requiredText(fmt.Sprintf("blocks[%d].items[%d].value", index, itemIndex), item.Value, 8_000); err != nil {
			return err
		}
		if len([]rune(item.Note)) > 2_000 {
			return fmt.Errorf("blocks[%d].items[%d].note is too long", index, itemIndex)
		}
	}
	for columnIndex, column := range block.Columns {
		if err := requiredText(fmt.Sprintf("blocks[%d].columns[%d]", index, columnIndex), column, 500); err != nil {
			return err
		}
	}
	for rowIndex, row := range block.Rows {
		if len(row) != len(block.Columns) {
			return fmt.Errorf("blocks[%d].rows[%d] does not match columns", index, rowIndex)
		}
		for cellIndex, cell := range row {
			if len([]rune(cell)) > 4_000 {
				return fmt.Errorf("blocks[%d].rows[%d][%d] is too long", index, rowIndex, cellIndex)
			}
		}
	}
	switch block.Kind {
	case BlockProse:
		if block.Body == "" || len(block.Items)+len(block.Columns)+len(block.Rows) > 0 || block.Code != "" {
			return fmt.Errorf("blocks[%d] prose requires only body", index)
		}
	case BlockCards, BlockSequence, BlockChecklist, BlockTasks:
		if len(block.Items) == 0 || block.Body != "" || len(block.Columns)+len(block.Rows) > 0 || block.Code != "" {
			return fmt.Errorf("blocks[%d] %s requires only items", index, block.Kind)
		}
	case BlockTable:
		if len(block.Columns) == 0 || len(block.Rows) == 0 || block.Body != "" || len(block.Items) > 0 || block.Code != "" {
			return fmt.Errorf("blocks[%d] table requires only columns and rows", index)
		}
	case BlockCode:
		if block.Code == "" || len([]rune(block.Code)) > 100_000 || block.Body != "" || len(block.Items)+len(block.Columns)+len(block.Rows) > 0 {
			return fmt.Errorf("blocks[%d] code requires only code", index)
		}
	default:
		return fmt.Errorf("blocks[%d].kind is invalid", index)
	}
	if len([]rune(block.Body)) > 50_000 {
		return fmt.Errorf("blocks[%d].body is too long", index)
	}
	return nil
}

func validateTextList(name string, values []string, maximumItems, maximumChars int) error {
	if len(values) > maximumItems {
		return fmt.Errorf("%s contains too many items", name)
	}
	for index, value := range values {
		if err := requiredText(fmt.Sprintf("%s[%d]", name, index), value, maximumChars); err != nil {
			return err
		}
	}
	return nil
}

func requiredText(name, value string, maximum int) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must be non-empty text", name)
	}
	if len([]rune(value)) > maximum {
		return fmt.Errorf("%s exceeds %d characters", name, maximum)
	}
	return nil
}

func DecodePack(raw []byte, brief GrowthBrief) (GrowthPack, error) {
	if len(raw) == 0 || len(raw) > MaxPackJSONBytes {
		return GrowthPack{}, fmt.Errorf("growth pack is empty or too large")
	}
	var pack GrowthPack
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pack); err != nil {
		return GrowthPack{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return GrowthPack{}, fmt.Errorf("growth pack contains trailing JSON")
	} else if !errors.Is(err, io.EOF) {
		return GrowthPack{}, fmt.Errorf("finish decoding Growth Pack: %w", err)
	}
	pack = NormalizePack(pack)
	if err := ValidatePack(brief.PlaybookID, brief.Evidence, pack); err != nil {
		return GrowthPack{}, err
	}
	return pack, nil
}
