package measurement

import (
	"net/url"
	"testing"
)

func TestExperimentRatesAreSafeDirectionalAndNeverAutoWinner(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	view, err := store.Save(Experiment{
		Title: "Hook test", Hypothesis: "A specific hook may improve qualified clicks.", Variable: "hook",
		VariantA: "Problem hook", VariantB: "Outcome hook", PrimaryMetric: "qualified lead rate", Comparable: false,
		MetricsA: VariantMetrics{}, MetricsB: VariantMetrics{Impressions: 100, Clicks: 10, Leads: 2, Sales: 1, RevenueSatang: 50_000},
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.RatesA.CTR != 0 || view.RatesA.LeadRate != 0 || view.RatesA.CloseRate != 0 {
		t.Fatalf("zero denominator rates = %#v", view.RatesA)
	}
	if view.RatesB.CTR != 0.1 || view.RatesB.LeadRate != 0.2 || view.RatesB.CloseRate != 0.5 {
		t.Fatalf("derived rates = %#v", view.RatesB)
	}
	if view.AnalysisLabel != "directional" || view.Winner != "" {
		t.Fatalf("analysis metadata = %#v", view)
	}
	view.Experiment.Comparable = true
	view, err = store.Save(view.Experiment)
	if err != nil || view.AnalysisLabel != "comparable" || view.Winner != "" {
		t.Fatalf("comparable view = %#v, %v", view, err)
	}
}

func TestUTMBuilderPreservesNonUTMOverwritesEscapesAndRejectsUnsafeURL(t *testing.T) {
	result, err := BuildUTM(UTMRequest{
		DestinationURL: "https://Example.com/landing?ref=owner&utm_source=old&utm_term=old#section",
		Source:         " Facebook ", Medium: "Organic Social", Campaign: "August Launch", Content: "Teacher's Video", Term: "ร้าน ออนไลน์",
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(result.URL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	for key, want := range map[string]string{
		"ref": "owner", "utm_source": "facebook", "utm_medium": "organic-social", "utm_campaign": "august-launch", "utm_content": "teacher-s-video", "utm_term": "ร้าน-ออนไลน์",
	} {
		if query.Get(key) != want {
			t.Errorf("%s = %q, want %q", key, query.Get(key), want)
		}
	}
	if parsed.Fragment != "section" || result.CampaignID != "facebook-organic-social-august-launch" {
		t.Fatalf("UTM result = %#v", result)
	}
	for _, destination := range []string{"ftp://example.com/file", "https://user:secret@example.com/", "//example.com/path", "https://example.com\\path"} {
		if _, err := BuildUTM(UTMRequest{DestinationURL: destination, Source: "facebook", Medium: "social", Campaign: "test"}); err == nil {
			t.Errorf("unsafe destination %q was accepted", destination)
		}
	}
}
