package cliprovider

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"ContentBlueprint/internal/facebookcompanion"
)

type Workflow string

const (
	WorkflowSingle Workflow = "single"
	WorkflowTeam   Workflow = "team"
)

type Stage string

const (
	StageGenerate   Stage = "generate"
	StageStrategist Stage = "strategist"
	StageCopywriter Stage = "copywriter"
	StageReviewer   Stage = "reviewer"
)

type StageStatus string

const (
	StageStarted   StageStatus = "started"
	StageCompleted StageStatus = "completed"
	StageFailed    StageStatus = "failed"
)

// ProgressEvent contains no prompt, generated copy, local path, or account
// data. RunID lets a Wails UI discard events from an older request.
type ProgressEvent struct {
	RunID    string      `json:"runId"`
	Workflow Workflow    `json:"workflow"`
	Stage    Stage       `json:"stage"`
	Status   StageStatus `json:"status"`
	Index    int         `json:"index"`
	Total    int         `json:"total"`
}

type ProgressFunc func(ProgressEvent)

func (workflow Workflow) normalized() (Workflow, error) {
	if workflow == "" {
		return WorkflowSingle, nil
	}
	if workflow != WorkflowSingle && workflow != WorkflowTeam {
		return "", fmt.Errorf("unsupported workflow")
	}
	return workflow, nil
}

// WorkflowStages returns a fresh slice suitable for rendering a stable Wails
// progress indicator before generation starts.
func WorkflowStages(workflow Workflow) ([]Stage, error) {
	workflow, err := workflow.normalized()
	if err != nil {
		return nil, err
	}
	if workflow == WorkflowTeam {
		return []Stage{StageStrategist, StageCopywriter, StageReviewer}, nil
	}
	return []Stage{StageGenerate}, nil
}

func secureRunID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func emitProgress(callback ProgressFunc, event ProgressEvent) {
	if callback == nil {
		return
	}
	// A presentation callback must not crash or corrupt the generation run.
	func() {
		defer func() { _ = recover() }()
		callback(event)
	}()
}

type StrategyAngle struct {
	Name         string `json:"name"`
	HookApproach string `json:"hookApproach"`
	Rationale    string `json:"rationale"`
}

type EvidenceUse struct {
	SourceID     string `json:"sourceId"`
	AllowedClaim string `json:"allowedClaim"`
}

// TeamStrategy is the bounded handoff from strategist to later workers.
type TeamStrategy struct {
	AudienceInsight string          `json:"audienceInsight"`
	Positioning     string          `json:"positioning"`
	PrimaryPromise  string          `json:"primaryPromise"`
	Angles          []StrategyAngle `json:"angles"`
	NarrativeFlow   []string        `json:"narrativeFlow"`
	EvidenceUse     []EvidenceUse   `json:"evidenceUse"`
	ComplianceRisks []string        `json:"complianceRisks"`
}

const maximumStrategyBytes = 100_000

func normalizeStrategy(strategy TeamStrategy) TeamStrategy {
	strategy.AudienceInsight = strings.TrimSpace(strategy.AudienceInsight)
	strategy.Positioning = strings.TrimSpace(strategy.Positioning)
	strategy.PrimaryPromise = strings.TrimSpace(strategy.PrimaryPromise)
	if strategy.Angles == nil {
		strategy.Angles = []StrategyAngle{}
	}
	for index := range strategy.Angles {
		strategy.Angles[index].Name = strings.TrimSpace(strategy.Angles[index].Name)
		strategy.Angles[index].HookApproach = strings.TrimSpace(strategy.Angles[index].HookApproach)
		strategy.Angles[index].Rationale = strings.TrimSpace(strategy.Angles[index].Rationale)
	}
	if strategy.NarrativeFlow == nil {
		strategy.NarrativeFlow = []string{}
	}
	for index := range strategy.NarrativeFlow {
		strategy.NarrativeFlow[index] = strings.TrimSpace(strategy.NarrativeFlow[index])
	}
	if strategy.EvidenceUse == nil {
		strategy.EvidenceUse = []EvidenceUse{}
	}
	for index := range strategy.EvidenceUse {
		strategy.EvidenceUse[index].SourceID = strings.TrimSpace(strategy.EvidenceUse[index].SourceID)
		strategy.EvidenceUse[index].AllowedClaim = strings.TrimSpace(strategy.EvidenceUse[index].AllowedClaim)
	}
	if strategy.ComplianceRisks == nil {
		strategy.ComplianceRisks = []string{}
	}
	for index := range strategy.ComplianceRisks {
		strategy.ComplianceRisks[index] = strings.TrimSpace(strategy.ComplianceRisks[index])
	}
	return strategy
}

func validateStrategy(strategy TeamStrategy, brief facebookcompanion.Brief) error {
	strategy = normalizeStrategy(strategy)
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{"audienceInsight", strategy.AudienceInsight, 2_000},
		{"positioning", strategy.Positioning, 2_000},
		{"primaryPromise", strategy.PrimaryPromise, 2_000},
	} {
		if err := validateWorkflowText(field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if len(strategy.Angles) < 3 || len(strategy.Angles) > 5 {
		return fmt.Errorf("angles must contain 3 to 5 items")
	}
	seenAngles := make(map[string]struct{}, len(strategy.Angles))
	for index, angle := range strategy.Angles {
		if err := validateWorkflowText(fmt.Sprintf("angles[%d].name", index), angle.Name, 300); err != nil {
			return err
		}
		if err := validateWorkflowText(fmt.Sprintf("angles[%d].hookApproach", index), angle.HookApproach, 1_000); err != nil {
			return err
		}
		if err := validateWorkflowText(fmt.Sprintf("angles[%d].rationale", index), angle.Rationale, 1_500); err != nil {
			return err
		}
		key := strings.ToLower(angle.Name + "\x00" + angle.HookApproach)
		if _, exists := seenAngles[key]; exists {
			return fmt.Errorf("angles must be distinct")
		}
		seenAngles[key] = struct{}{}
	}
	if len(strategy.NarrativeFlow) < 3 || len(strategy.NarrativeFlow) > 10 {
		return fmt.Errorf("narrativeFlow must contain 3 to 10 items")
	}
	for index, item := range strategy.NarrativeFlow {
		if err := validateWorkflowText(fmt.Sprintf("narrativeFlow[%d]", index), item, 1_000); err != nil {
			return err
		}
	}
	if len(strategy.EvidenceUse) > len(brief.Evidence) || len(strategy.EvidenceUse) > 30 {
		return fmt.Errorf("evidenceUse contains too many items")
	}
	validSources := make(map[string]string, len(brief.Evidence))
	for _, source := range brief.Evidence {
		validSources[source.ID] = source.Notes
	}
	seenSources := make(map[string]struct{}, len(strategy.EvidenceUse))
	for index, item := range strategy.EvidenceUse {
		notes, exists := validSources[item.SourceID]
		if !exists {
			return fmt.Errorf("evidenceUse[%d].sourceId is not present in the brief", index)
		}
		if strings.TrimSpace(notes) == "" {
			return fmt.Errorf("evidenceUse[%d].sourceId has no supplied evidence notes", index)
		}
		if _, exists := seenSources[item.SourceID]; exists {
			return fmt.Errorf("evidenceUse sourceId is duplicated")
		}
		seenSources[item.SourceID] = struct{}{}
		if err := validateWorkflowText(fmt.Sprintf("evidenceUse[%d].allowedClaim", index), item.AllowedClaim, 2_000); err != nil {
			return err
		}
	}
	if len(strategy.ComplianceRisks) > 12 {
		return fmt.Errorf("complianceRisks contains too many items")
	}
	for index, item := range strategy.ComplianceRisks {
		if err := validateWorkflowText(fmt.Sprintf("complianceRisks[%d]", index), item, 1_500); err != nil {
			return err
		}
	}
	encoded, err := json.Marshal(strategy)
	if err != nil {
		return err
	}
	if len(encoded) > maximumStrategyBytes {
		return fmt.Errorf("strategy exceeds %d bytes", maximumStrategyBytes)
	}
	return nil
}

func validateWorkflowText(name, value string, maximum int) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must be non-empty text", name)
	}
	if len([]rune(value)) > maximum {
		return fmt.Errorf("%s exceeds %d characters", name, maximum)
	}
	return nil
}
