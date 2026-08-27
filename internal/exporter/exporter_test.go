package exporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/htmlutil"
)

func TestRenderHTMLProducesDocumentAndRemovesActiveContent(t *testing.T) {
	project := exportProjectFixture()
	project.Content.MainContentHTML = `<h2>Overview</h2><p onclick="steal()">Safe text <a href="javascript:bad()">bad link</a>.</p><script>alert(1)</script>`

	data, extension, err := Render(project, "html")
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)
	if extension != ".html" || !strings.Contains(result, "<!doctype html>") || !strings.Contains(result, "<h1>Export title</h1>") {
		t.Errorf("unexpected HTML export: %s", result)
	}
	for _, forbidden := range []string{"<script", "onclick", "javascript:"} {
		if strings.Contains(strings.ToLower(result), forbidden) {
			t.Errorf("HTML export contains %q: %s", forbidden, result)
		}
	}
	if !strings.Contains(result, `href="https://example.com/source"`) {
		t.Errorf("HTML export is missing evidence link: %s", result)
	}
	if !strings.Contains(result, "Grounding sources") || !strings.Contains(result, `href="https://grounded.example/article"`) {
		t.Errorf("HTML export is missing grounding sources: %s", result)
	}
}

func TestRenderMarkdownPreservesUsefulStructure(t *testing.T) {
	project := exportProjectFixture()
	project.Content.MainContentHTML = `<h2>Overview</h2><p>Hello <strong>reader</strong>.</p><ul><li>First</li></ul>`
	data, extension, err := Render(project, "md")
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)
	if extension != ".md" || !strings.Contains(result, "# Export title") || !strings.Contains(result, "## Overview") || !strings.Contains(result, "**reader**") {
		t.Errorf("unexpected Markdown export: %s", result)
	}
	if !strings.Contains(result, "## Grounding sources") || !strings.Contains(result, `[Grounded article](https://grounded.example/article)`) {
		t.Errorf("Markdown export is missing grounding sources: %s", result)
	}
}

func TestRenderJSONRoundTripsProject(t *testing.T) {
	project := exportProjectFixture()
	data, extension, err := Render(project, "json")
	if err != nil {
		t.Fatal(err)
	}
	var decoded domain.Project
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if extension != ".json" || decoded.ID != project.ID || decoded.Content == nil || decoded.Content.Title != project.Content.Title || len(decoded.GroundingSources) != 1 {
		t.Errorf("decoded export = %#v", decoded)
	}
	if strings.Contains(strings.ToLower(string(data)), "apikey") {
		t.Errorf("JSON project export contains an API-key field: %s", data)
	}
}

func TestHTMLToMarkdownPreservesInlineWhitespace(t *testing.T) {
	got := htmlToMarkdown(htmlutil.Sanitize(`<p>Hello <strong>reader</strong>.</p>`))
	if got != "Hello **reader**.\n" {
		t.Errorf("htmlToMarkdown() = %q, want inline whitespace preserved", got)
	}
}

func TestHTMLToMarkdownExtractsPlainPreformattedText(t *testing.T) {
	got := htmlToMarkdown(htmlutil.Sanitize(`<pre><code>&lt;tag attr="x"&gt;&amp; value</code></pre>`))
	if !strings.Contains(got, "```\n<tag attr=\"x\">& value\n```") {
		t.Errorf("htmlToMarkdown() pre block = %q", got)
	}
	if strings.Contains(got, "<code>") || strings.Contains(got, "&lt;") {
		t.Errorf("htmlToMarkdown() leaked markup/entities in pre block: %q", got)
	}
}

func TestHTMLToMarkdownUsesFenceLongerThanCodeBackticks(t *testing.T) {
	got := htmlToMarkdown(htmlutil.Sanitize("<pre><code>before\n```\nafter</code></pre>"))
	if !strings.HasPrefix(got, "````\nbefore\n```\nafter\n````") {
		t.Errorf("htmlToMarkdown() fence = %q, want four-backtick fence around triple backticks", got)
	}
}

func TestMarkdownExportEscapesHTMLAndMarkdownFromPlainText(t *testing.T) {
	project := exportProjectFixture()
	project.Content.SummaryBox = `<img src=x onerror=alert(1)> **not bold**`
	project.Content.MainContentHTML = `<p>&lt;script&gt;alert(1)&lt;/script&gt; *not emphasis*</p>`
	project.Content.FAQData = []domain.FAQItem{{Question: `<img src=x>`, Answer: `<script>alert(1)</script>`, SourceIDs: []string{}}}
	data, _, err := Render(project, "markdown")
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)
	for _, rawHTML := range []string{"<img", "<script>"} {
		if strings.Contains(strings.ToLower(result), rawHTML) {
			t.Errorf("Markdown export contains executable raw HTML %q: %s", rawHTML, result)
		}
	}
	for _, escaped := range []string{`&lt;img src=x onerror=alert(1)&gt;`, `\*\*not bold\*\*`, `&lt;script&gt;alert(1)&lt;/script&gt;`, `\*not emphasis\*`} {
		if !strings.Contains(result, escaped) {
			t.Errorf("Markdown export does not contain escaped text %q: %s", escaped, result)
		}
	}
}

func TestHTMLToMarkdownKeepsSanitizedTableAsHTML(t *testing.T) {
	got := htmlToMarkdown(htmlutil.Sanitize(`<table><thead><tr><th>Name</th><th>Value</th></tr></thead><tbody><tr><td>A</td><td>1</td></tr></tbody></table>`))
	if !strings.Contains(got, "<table>") || !strings.Contains(got, "<th>Name</th>") || !strings.Contains(got, "<td>1</td>") {
		t.Errorf("htmlToMarkdown() table = %q, want sanitized raw HTML", got)
	}
}

func TestRenderRejectsUnsupportedFormat(t *testing.T) {
	if _, _, err := Render(exportProjectFixture(), "pdf"); err == nil {
		t.Fatal("Render() accepted unsupported PDF format")
	}
}

func TestWriteAtomicallyReplacesFileAndUsesPrivateMode(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "article.md")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []byte("new content")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content" {
		t.Errorf("export content = %q, want replacement", data)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if permissions := info.Mode().Perm(); permissions != 0o600 {
			t.Errorf("export permissions = %o, want 600", permissions)
		}
	}
	temporaryFiles, err := filepath.Glob(filepath.Join(directory, ".content-blueprint-export-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporaryFiles) != 0 {
		t.Errorf("temporary exports remain after success: %#v", temporaryFiles)
	}
}

func TestWriteAtomicFailurePreservesTargetAndCleansTemporaryFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "article.md")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	temporaryPath := ""
	closedTempCreator := func(directory, pattern string) (*os.File, error) {
		file, err := os.CreateTemp(directory, pattern)
		if err != nil {
			return nil, err
		}
		temporaryPath = file.Name()
		if err := file.Close(); err != nil {
			return nil, err
		}
		return file, nil
	}
	if err := writeAtomic(path, []byte("replacement"), closedTempCreator); err == nil {
		t.Fatal("writeAtomic() with closed temporary file returned nil error")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Errorf("failed atomic write changed target to %q", data)
	}
	if temporaryPath == "" {
		t.Fatal("test temp creator was not called")
	}
	if _, err := os.Stat(temporaryPath); !os.IsNotExist(err) {
		t.Errorf("temporary file was not cleaned up; stat error = %v", err)
	}
}

func exportProjectFixture() domain.Project {
	return domain.Project{
		ID:   "project-1",
		Name: "Export project",
		Brief: domain.ContentBrief{
			Language: "en",
			Evidence: []domain.EvidenceSource{{
				ID: "source-1", Title: "Source", URL: "https://example.com/source", Notes: "Evidence notes",
			}},
		},
		GroundingSources: []domain.GroundingSource{{Title: "Grounded article", URL: "https://grounded.example/article"}},
		Content: &domain.GeneratedContent{
			Title: "Export title", Slug: "export-title", MetaTitle: "Export meta title", MetaDescription: "Export description",
			SummaryBox:   "Summary",
			KeyTakeaways: []domain.KeyTakeaway{{Statement: "Takeaway", SourceIDs: []string{"source-1"}}},
			FAQData:      []domain.FAQItem{{Question: "Question?", Answer: "Answer.", SourceIDs: []string{"source-1"}}},
		},
	}
}
