package quality

import (
	"strings"
	"testing"

	"ContentBlueprint/internal/domain"
)

func TestEvaluateHighQualityContent(t *testing.T) {
	brief := domain.ContentBrief{
		Keyword: "content blueprint",
		Evidence: []domain.EvidenceSource{
			{ID: "a", Title: "A"},
			{ID: "b", Title: "B"},
		},
	}
	body := `<h2>Content blueprint overview</h2><p>Evidence-backed claim <sup data-source-id="a">[a]</sup> and another claim <sup data-source-id="b">[b]</sup>. ` +
		strings.Repeat("content blueprint explains useful evidence clearly. ", 90) + "</p><h3>Practical steps</h3><p>Use the evidence.</p>"
	content := domain.GeneratedContent{
		Title:           "Content blueprint for evidence-first articles",
		Slug:            "content-blueprint",
		MetaTitle:       "Content Blueprint for Better Evidence-Based SEO",
		MetaDescription: strings.Repeat("Clear evidence-first guidance for practical content planning. ", 2),
		SummaryBox:      "A concise explanation.",
		MainContentHTML: body,
		KeyTakeaways: []domain.KeyTakeaway{
			{Statement: "One", SourceIDs: []string{"a"}},
			{Statement: "Two", SourceIDs: []string{"b"}},
			{Statement: "Three", SourceIDs: []string{}},
		},
		FAQData: []domain.FAQItem{
			{Question: "Q1?", Answer: "A1", SourceIDs: []string{"a"}},
			{Question: "Q2?", Answer: "A2", SourceIDs: []string{"b"}},
			{Question: "Q3?", Answer: "A3", SourceIDs: []string{}},
		},
	}

	report := Evaluate(brief, content)
	if report.SourceCoverage != 100 {
		t.Errorf("sourceCoverage = %d, want 100", report.SourceCoverage)
	}
	if report.WordCount < 350 {
		t.Errorf("wordCount = %d, want at least 350", report.WordCount)
	}
	if report.Score < 85 {
		t.Errorf("score = %d, want >= 85; checks = %#v", report.Score, report.Checks)
	}
	for _, item := range report.Checks {
		if item.Status != StatusPass {
			t.Errorf("check %q status = %q, want pass (%s)", item.ID, item.Status, item.Message)
		}
	}
}

func TestEvaluateFlagsUnknownSourcesAndUnsafeHTML(t *testing.T) {
	brief := domain.ContentBrief{Keyword: "keyword", Evidence: []domain.EvidenceSource{{ID: "known", Title: "Known"}}}
	content := domain.GeneratedContent{
		Title: "keyword", Slug: "keyword", MetaTitle: "keyword", MetaDescription: "short", SummaryBox: "summary",
		MainContentHTML: `<h2>keyword</h2><script>alert(1)</script><p onclick="bad()">text</p>`,
		KeyTakeaways:    []domain.KeyTakeaway{{Statement: "Claim", SourceIDs: []string{"invented"}}},
		FAQData:         []domain.FAQItem{{Question: "Question?", Answer: "Answer", SourceIDs: []string{}}},
	}

	report := Evaluate(brief, content)
	assertCheckStatus(t, report, "source_coverage", StatusFail)
	assertCheckStatus(t, report, "body_citations", StatusFail)
	assertCheckStatus(t, report, "safe_html", StatusFail)
	if report.Score >= 70 {
		t.Errorf("score = %d, want below 70 for unsafe unsupported content", report.Score)
	}
}

func TestEvaluateRequiresBodyCitationEvenWhenStructuredItemsCoverEvidence(t *testing.T) {
	brief := domain.ContentBrief{
		Keyword:  "keyword",
		Evidence: []domain.EvidenceSource{{ID: "S1", Title: "Source"}},
	}
	content := domain.GeneratedContent{
		Title: "keyword title", Slug: "keyword", MetaTitle: "keyword title", MetaDescription: "description", SummaryBox: "summary",
		MainContentHTML: "<h2>keyword</h2><p>A factual claim without an inline marker.</p>",
		KeyTakeaways: []domain.KeyTakeaway{
			{Statement: "One", SourceIDs: []string{"S1"}},
			{Statement: "Two", SourceIDs: []string{"S1"}},
			{Statement: "Three", SourceIDs: []string{"S1"}},
		},
		FAQData: []domain.FAQItem{
			{Question: "Q1?", Answer: "A1", SourceIDs: []string{"S1"}},
			{Question: "Q2?", Answer: "A2", SourceIDs: []string{"S1"}},
			{Question: "Q3?", Answer: "A3", SourceIDs: []string{"S1"}},
		},
	}
	report := Evaluate(brief, content)
	if report.SourceCoverage != 100 {
		t.Errorf("sourceCoverage = %d, want 100 from structured references", report.SourceCoverage)
	}
	assertCheckStatus(t, report, "body_citations", StatusFail)
}

func TestEvaluateIncludesExactBodyMarkersInSourceCoverage(t *testing.T) {
	brief := domain.ContentBrief{
		Keyword: "keyword",
		Evidence: []domain.EvidenceSource{
			{ID: "S1", Title: "One"},
			{ID: "S2", Title: "Two"},
		},
	}
	content := domain.GeneratedContent{
		Title: "keyword title", Slug: "keyword", MetaTitle: "keyword title", MetaDescription: "description", SummaryBox: "summary",
		MainContentHTML: `<h2>keyword</h2><p>Claim <sup data-source-id="S1">[S1]</sup>. Another <sup data-source-id="S2">[S2]</sup>.</p>`,
		KeyTakeaways:    []domain.KeyTakeaway{},
		FAQData:         []domain.FAQItem{},
	}
	report := Evaluate(brief, content)
	if report.SourceCoverage != 100 {
		t.Errorf("sourceCoverage = %d, want 100 from body markers", report.SourceCoverage)
	}
	assertCheckStatus(t, report, "body_citations", StatusPass)
}

func TestCountWordsEstimatesUnspacedThai(t *testing.T) {
	if got := CountWords("การเขียนบทความภาษาไทยให้ชัดเจน"); got < 3 {
		t.Errorf("CountWords() = %d, want a useful Thai estimate", got)
	}
}

func assertCheckStatus(t *testing.T, report domain.QualityReport, id, want string) {
	t.Helper()
	for _, item := range report.Checks {
		if item.ID == id {
			if item.Status != want {
				t.Errorf("check %q status = %q, want %q", id, item.Status, want)
			}
			return
		}
	}
	t.Errorf("check %q was not present", id)
}
