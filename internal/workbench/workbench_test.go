package workbench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ContentBlueprint/internal/domain"
)

func TestCatalogHasExactlyTenValidTrustedDefinitions(t *testing.T) {
	playbooks := Catalog()
	if len(playbooks) != 10 {
		t.Fatalf("catalog length = %d, want 10", len(playbooks))
	}
	seen := map[string]bool{}
	for _, playbook := range playbooks {
		if seen[playbook.ID] || playbook.ID == "" || playbook.Category == "" || playbook.Title == "" || playbook.Summary == "" || playbook.Outcome == "" || len(playbook.Fields) == 0 {
			t.Fatalf("invalid playbook definition: %#v", playbook)
		}
		seen[playbook.ID] = true
		instructions, ok := TrustedInstructions(playbook.ID)
		if !ok || strings.TrimSpace(instructions) == "" {
			t.Fatalf("playbook %q has no trusted instructions", playbook.ID)
		}
		fieldKeys := map[string]bool{}
		brief := GrowthBrief{PlaybookID: playbook.ID, Language: "Thai", Inputs: map[string]string{}, Evidence: []domain.EvidenceSource{}}
		for _, field := range playbook.Fields {
			if fieldKeys[field.Key] || field.Key == "" || field.Label == "" || field.Help == "" || field.Placeholder == "" || field.MaxChars < 1 {
				t.Fatalf("invalid field in %q: %#v", playbook.ID, field)
			}
			if field.InputType != "text" && field.InputType != "textarea" && field.InputType != "url" {
				t.Fatalf("invalid input type in %q: %q", playbook.ID, field.InputType)
			}
			fieldKeys[field.Key] = true
			if field.Required {
				brief.Inputs[field.Key] = "verified input"
				if field.InputType == "url" {
					brief.Inputs[field.Key] = "https://example.com/page"
				}
			}
		}
		if err := ValidateBrief(brief); err != nil {
			t.Fatalf("default valid brief for %q: %v", playbook.ID, err)
		}
	}
	wantIDs := []string{"offer-audience", "facebook-campaign", "sales-reply", "seo-topic-map", "seo-content-brief", "seo-onpage-review", "seo-internal-links", "seo-structured-data", "seo-search-console-opportunities", "cross-channel-repurpose"}
	for _, id := range wantIDs {
		if !seen[id] {
			t.Errorf("catalog missing %q", id)
		}
	}
	facebook, _ := LookupPlaybook("facebook-campaign")
	fieldKeys := map[string]bool{}
	for _, field := range facebook.Fields {
		fieldKeys[field.Key] = true
	}
	if !fieldKeys["competitorAdsNotes"] || !fieldKeys["ownedAudienceAssets"] {
		t.Fatalf("Facebook campaign fields do not expose safe user-supplied competitive/audience inputs: %#v", fieldKeys)
	}
	instructions, _ := TrustedInstructions("facebook-campaign")
	for _, guardrail := range []string{"Do not scrape", "identify people", "another page's followers", "lawful basis", "Audience Source Plan", "creative gap"} {
		if !strings.Contains(instructions, guardrail) {
			t.Errorf("Facebook trusted prompt missing guardrail %q", guardrail)
		}
	}
	encoded, err := json.Marshal(playbooks)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "TRUSTED_PLAYBOOK_INSTRUCTIONS") || strings.Contains(string(encoded), "another page's followers") {
		t.Fatal("trusted prompt instructions leaked through the public catalog DTO")
	}
}

func TestBriefValidationIsExactBoundedAndCanonical(t *testing.T) {
	brief := validBrief()
	first, err := BriefRevision(brief)
	if err != nil {
		t.Fatal(err)
	}
	reordered := validBrief()
	reordered.Inputs = map[string]string{"audience": "Small shop owners", "offer": "Eight-lesson workshop", "problems": "Unclear positioning"}
	second, err := BriefRevision(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || len(first) != 64 {
		t.Fatalf("canonical revisions differ: %q != %q", first, second)
	}

	unknown := validBrief()
	unknown.Inputs["notAllowed"] = "value"
	if err := ValidateBrief(unknown); err == nil {
		t.Fatal("unknown input field was accepted")
	}
	missing := validBrief()
	delete(missing.Inputs, "offer")
	if err := ValidateBrief(missing); err == nil {
		t.Fatal("missing required input was accepted")
	}
	oversized := validBrief()
	oversized.Inputs["offer"] = strings.Repeat("x", 12_001)
	if err := ValidateBrief(oversized); err == nil {
		t.Fatal("oversized input was accepted")
	}
	badSource := validBrief()
	badSource.Evidence[0].ID = "bad id"
	if err := ValidateBrief(badSource); err == nil {
		t.Fatal("invalid evidence source was accepted")
	}
	duplicateSource := validBrief()
	duplicateSource.Evidence = append(duplicateSource.Evidence, duplicateSource.Evidence[0])
	if err := ValidateBrief(duplicateSource); err == nil {
		t.Fatal("duplicate evidence source was accepted")
	}
}

func TestGrowthPackStrictKindsSourcesAndPlaybookSemantics(t *testing.T) {
	brief := validBrief()
	pack := validPack()
	if err := ValidatePack(brief.PlaybookID, brief.Evidence, pack); err != nil {
		t.Fatalf("valid pack: %v", err)
	}

	unknownSource := validPack()
	unknownSource.Blocks[0].SourceIDs = []string{"missing"}
	if err := ValidatePack(brief.PlaybookID, brief.Evidence, unknownSource); err == nil {
		t.Fatal("unknown source ID was accepted")
	}
	wrongShape := validPack()
	wrongShape.Blocks[0].Items = []BlockItem{{Label: "extra", Value: "not allowed"}}
	if err := ValidatePack(brief.PlaybookID, brief.Evidence, wrongShape); err == nil {
		t.Fatal("prose block with items was accepted")
	}
	duplicate := validPack()
	duplicate.Blocks = append(duplicate.Blocks, duplicate.Blocks[0])
	if err := ValidatePack(brief.PlaybookID, brief.Evidence, duplicate); err == nil {
		t.Fatal("duplicate block ID was accepted")
	}
	metric := validPack()
	metric.Blocks[0].EvidenceBasis = BasisImportedMetric
	metric.Blocks[0].SourceIDs = []string{}
	if err := ValidatePack(brief.PlaybookID, brief.Evidence, metric); err == nil {
		t.Fatal("imported metric was accepted outside Search Console playbook")
	}
	metricBrief := GrowthBrief{PlaybookID: "seo-search-console-opportunities", Language: "Thai", Inputs: map[string]string{"importedMetrics": "Imported user data: 10 clicks", "dateRange": "2026-01-01 to 2026-01-31", "property": "example.com", "goal": "Find refresh opportunities"}, Evidence: []domain.EvidenceSource{}}
	if err := ValidateBrief(metricBrief); err != nil {
		t.Fatal(err)
	}
	metric.Blocks[0].Kind = BlockProse
	if err := ValidatePack(metricBrief.PlaybookID, metricBrief.Evidence, metric); err != nil {
		t.Fatalf("Search Console imported metric pack: %v", err)
	}
	raw, _ := json.Marshal(validPack())
	raw = append(raw[:len(raw)-1], []byte(`,"unknown":true}`)...)
	if _, err := DecodePack(raw, brief); err == nil {
		t.Fatal("strict decoder accepted unknown field")
	}
}

func TestStorePersistsAtomicallyMarksStaleAndRequiresReviewNote(t *testing.T) {
	directory := t.TempDir()
	store, err := NewStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	brief := validBrief()
	briefSnapshot, err := store.SaveBrief(brief)
	if err != nil {
		t.Fatal(err)
	}
	packSnapshot, err := store.SavePack(briefSnapshot.BriefRevision, validPack(), "Codex CLI")
	if err != nil {
		t.Fatal(err)
	}
	if packSnapshot.ReviewStatus != "needs_review" || packSnapshot.PlaybookID != brief.PlaybookID {
		t.Fatalf("pack snapshot metadata = %#v", packSnapshot)
	}
	loaded, err := store.LoadPack()
	if err != nil || !reflect.DeepEqual(loaded.Pack, NormalizePack(validPack())) {
		t.Fatalf("LoadPack = %#v, %v", loaded, err)
	}
	if _, err := store.ReviewPack(briefSnapshot.BriefRevision, "approved", ""); err == nil {
		t.Fatal("approval without reviewer note was accepted")
	}
	reviewed, err := store.ReviewPack(briefSnapshot.BriefRevision, "approved", "Owner checked claims and links.")
	if err != nil || reviewed.ReviewStatus != "approved" || reviewed.ReviewUpdatedAt == nil {
		t.Fatalf("ReviewPack = %#v, %v", reviewed, err)
	}
	changed := brief
	changed.Inputs["audience"] = "Marketing teachers"
	current, err := store.SaveBrief(changed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SavePack(briefSnapshot.BriefRevision, validPack(), "Codex CLI"); err == nil {
		t.Fatal("stale pack save was accepted")
	}
	if _, err := store.ReviewPack(briefSnapshot.BriefRevision, "rejected", "Brief changed."); err == nil {
		t.Fatal("stale review was accepted")
	}
	if current.BriefRevision == briefSnapshot.BriefRevision {
		t.Fatal("changed brief revision did not change")
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".growth-workbench-*.tmp"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files remain after atomic writes: %v, %v", matches, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "growth-brief.json")); err != nil {
		t.Fatal(err)
	}
}

func validBrief() GrowthBrief {
	return GrowthBrief{
		PlaybookID: "offer-audience", Language: "Thai", BrandVoice: "Direct and practical",
		Inputs:   map[string]string{"offer": "Eight-lesson workshop", "audience": "Small shop owners", "problems": "Unclear positioning"},
		Evidence: []domain.EvidenceSource{{ID: "outline", Title: "Course outline", URL: "https://example.com/outline", Notes: "The outline lists eight lessons."}},
	}
}

func validPack() GrowthPack {
	return GrowthPack{
		Title: "Offer and audience plan", Summary: "A bounded plan based on supplied facts.",
		Blocks:        []GrowthBlock{{ID: "value-proposition", Title: "Value proposition", Purpose: "Clarify the offer", Kind: BlockProse, Body: "The workshop contains eight lessons.", Items: []BlockItem{}, Columns: []string{}, Rows: [][]string{}, Code: "", EvidenceBasis: BasisSuppliedEvidence, SourceIDs: []string{"outline"}}},
		OpenQuestions: []string{"Confirm the workshop schedule."}, RiskFlags: []string{"Do not promise sales outcomes."},
		ReviewChecks: []ReviewCheck{{Status: "review", Label: "Owner approval", Reason: "The owner must verify price and schedule."}},
	}
}
