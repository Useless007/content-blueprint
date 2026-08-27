package htmlutil

import (
	"strings"
	"testing"
)

func TestSanitizeRejectsProtocolRelativeAndActiveLinks(t *testing.T) {
	input := `<p><a href="//evil.example/steal">protocol relative</a> <a href="javascript:steal()">script</a> <a href="/local">local</a> <a href="https://safe.example/page">safe</a></p>`
	result := Sanitize(input)

	if strings.Contains(result, `href="//evil.example/steal"`) || strings.Contains(strings.ToLower(result), "javascript:") {
		t.Errorf("Sanitize() retained an unsafe link: %s", result)
	}
	if !strings.Contains(result, `href="/local"`) || !strings.Contains(result, `href="https://safe.example/page"`) {
		t.Errorf("Sanitize() removed safe links: %s", result)
	}
}

func TestSafeLinkRejectsBackslashNetworkPaths(t *testing.T) {
	for _, value := range []string{"//evil.example", `\evil.example`, `\\evil.example`, `/\evil.example`, `relative\path`, "https:////evil.example", "javascript:alert(1)"} {
		if safeLink(value) {
			t.Errorf("safeLink(%q) = true, want false", value)
		}
	}
}

func TestSourceIDsAcceptsOnlyExactSupMarkers(t *testing.T) {
	value := `<p>Claim <sup data-source-id="S1">[S1]</sup> and another
<sup data-source-id="S2">[wrong]</sup><span data-source-id="S3">[S3]</span>
	<sup data-source-id="S1">[S1]</sup><sup data-source-id="source-4"><b>[source-4]</b></sup></p>
	<form><sup data-source-id="hidden">[hidden]</sup></form><script><sup data-source-id="script">[script]</sup></script>`
	got := SourceIDs(value)
	if len(got) != 1 || got[0] != "S1" {
		t.Errorf("SourceIDs() = %#v, want [S1]", got)
	}
}
