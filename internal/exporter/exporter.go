package exporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/htmlutil"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var filenameCharacter = regexp.MustCompile(`[^a-z0-9-]+`)

func Render(project domain.Project, format string) ([]byte, string, error) {
	switch NormalizeFormat(format) {
	case "html":
		return []byte(renderHTML(project)), ".html", nil
	case "markdown":
		return []byte(renderMarkdown(project)), ".md", nil
	case "json":
		data, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			return nil, "", fmt.Errorf("encode project JSON: %w", err)
		}
		return append(data, '\n'), ".json", nil
	default:
		return nil, "", fmt.Errorf("unsupported export format %q; use html, markdown, or json", format)
	}
}

func NormalizeFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(strings.TrimPrefix(format, "."))) {
	case "htm", "html":
		return "html"
	case "md", "markdown":
		return "markdown"
	case "json":
		return "json"
	default:
		return strings.ToLower(strings.TrimSpace(format))
	}
}

func DefaultFilename(project domain.Project, format string) string {
	content := projectContent(project)
	base := strings.ToLower(strings.TrimSpace(content.Slug))
	if base == "" {
		base = strings.ToLower(strings.TrimSpace(project.Name))
	}
	base = strings.ReplaceAll(base, "_", "-")
	base = strings.ReplaceAll(base, " ", "-")
	base = filenameCharacter.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "content-blueprint"
	}
	extension := map[string]string{"html": ".html", "markdown": ".md", "json": ".json"}[NormalizeFormat(format)]
	return base + extension
}

func EnsureExtension(path, extension string) string {
	if strings.EqualFold(filepath.Ext(path), extension) {
		return path
	}
	return path + extension
}

func Write(path string, data []byte) error {
	return writeAtomic(path, data, os.CreateTemp)
}

type createTempFunc func(directory, pattern string) (*os.File, error)

func writeAtomic(path string, data []byte, createTemp createTempFunc) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("export path is empty")
	}
	directory := filepath.Dir(path)
	temporary, err := createTemp(directory, ".content-blueprint-export-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary export file: %w", err)
	}
	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		_ = os.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("secure temporary export file: %w", err)
	}
	written, err := temporary.Write(data)
	if err != nil {
		return fmt.Errorf("write temporary export file: %w", err)
	}
	if written != len(data) {
		return fmt.Errorf("write temporary export file: %w", io.ErrShortWrite)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary export file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		closed = true
		return fmt.Errorf("close temporary export file: %w", err)
	}
	closed = true
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace export file: %w", err)
	}
	return nil
}

func renderHTML(project domain.Project) string {
	content := projectContent(project)
	language, sectionLabels := exportLanguage(project.Brief.Language)
	title := firstNonEmpty(content.MetaTitle, content.Title, project.Name, "Content Blueprint")
	var output strings.Builder
	output.WriteString("<!doctype html>\n<html lang=\"")
	output.WriteString(stdhtml.EscapeString(language))
	output.WriteString("\">\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <title>")
	output.WriteString(stdhtml.EscapeString(title))
	output.WriteString("</title>\n")
	if strings.TrimSpace(content.MetaDescription) != "" {
		output.WriteString("  <meta name=\"description\" content=\"")
		output.WriteString(stdhtml.EscapeString(content.MetaDescription))
		output.WriteString("\">\n")
	}
	output.WriteString("</head>\n<body>\n<article>\n")
	if strings.TrimSpace(content.Title) != "" {
		output.WriteString("  <h1>")
		output.WriteString(stdhtml.EscapeString(content.Title))
		output.WriteString("</h1>\n")
	}
	if strings.TrimSpace(content.SummaryBox) != "" {
		output.WriteString("  <aside aria-label=\"")
		output.WriteString(stdhtml.EscapeString(sectionLabels.summary))
		output.WriteString("\"><p>")
		output.WriteString(stdhtml.EscapeString(content.SummaryBox))
		output.WriteString("</p></aside>\n")
	}
	if body := htmlutil.Sanitize(content.MainContentHTML); body != "" {
		output.WriteString(body)
		output.WriteByte('\n')
	}
	if len(content.KeyTakeaways) > 0 {
		output.WriteString("  <section>\n    <h2>")
		output.WriteString(stdhtml.EscapeString(sectionLabels.takeaways))
		output.WriteString("</h2>\n    <ul>\n")
		for _, item := range content.KeyTakeaways {
			output.WriteString("      <li>")
			output.WriteString(stdhtml.EscapeString(item.Statement))
			writeHTMLSourceIDs(&output, item.SourceIDs)
			output.WriteString("</li>\n")
		}
		output.WriteString("    </ul>\n  </section>\n")
	}
	if len(content.FAQData) > 0 {
		output.WriteString("  <section>\n    <h2>")
		output.WriteString(stdhtml.EscapeString(sectionLabels.faq))
		output.WriteString("</h2>\n")
		for _, item := range content.FAQData {
			output.WriteString("    <h3>")
			output.WriteString(stdhtml.EscapeString(item.Question))
			output.WriteString("</h3>\n    <p>")
			output.WriteString(stdhtml.EscapeString(item.Answer))
			writeHTMLSourceIDs(&output, item.SourceIDs)
			output.WriteString("</p>\n")
		}
		output.WriteString("  </section>\n")
	}
	writeHTMLEvidence(&output, project.Brief.Evidence, sectionLabels.sources)
	writeHTMLGroundingSources(&output, project.GroundingSources, sectionLabels.groundingSources)
	output.WriteString("</article>\n</body>\n</html>\n")
	return output.String()
}

func writeHTMLSourceIDs(output *strings.Builder, ids []string) {
	if len(ids) == 0 {
		return
	}
	output.WriteString(" <small>[Sources: ")
	output.WriteString(stdhtml.EscapeString(strings.Join(ids, ", ")))
	output.WriteString("]</small>")
}

func writeHTMLEvidence(output *strings.Builder, evidence []domain.EvidenceSource, heading string) {
	if len(evidence) == 0 {
		return
	}
	output.WriteString("  <section>\n    <h2>")
	output.WriteString(stdhtml.EscapeString(heading))
	output.WriteString("</h2>\n    <ol>\n")
	for _, source := range evidence {
		output.WriteString("      <li id=\"source-")
		output.WriteString(stdhtml.EscapeString(source.ID))
		output.WriteString("\">")
		if normalizedURL, ok := domain.NormalizeGroundingURL(source.URL); ok {
			output.WriteString("<a href=\"")
			output.WriteString(stdhtml.EscapeString(normalizedURL))
			output.WriteString("\" rel=\"noopener noreferrer\">")
			output.WriteString(stdhtml.EscapeString(source.Title))
			output.WriteString("</a>")
		} else {
			output.WriteString(stdhtml.EscapeString(source.Title))
		}
		if strings.TrimSpace(source.Notes) != "" {
			output.WriteString(" — ")
			output.WriteString(stdhtml.EscapeString(source.Notes))
		}
		output.WriteString("</li>\n")
	}
	output.WriteString("    </ol>\n  </section>\n")
}

func writeHTMLGroundingSources(output *strings.Builder, sources []domain.GroundingSource, heading string) {
	sources = domain.NormalizeGroundingSources(sources)
	if len(sources) == 0 {
		return
	}
	output.WriteString("  <section>\n    <h2>")
	output.WriteString(stdhtml.EscapeString(heading))
	output.WriteString("</h2>\n    <ol>\n")
	for _, source := range sources {
		label := firstNonEmpty(source.Title, source.URL)
		output.WriteString("      <li><a href=\"")
		output.WriteString(stdhtml.EscapeString(source.URL))
		output.WriteString("\" rel=\"noopener noreferrer\">")
		output.WriteString(stdhtml.EscapeString(label))
		output.WriteString("</a></li>\n")
	}
	output.WriteString("    </ol>\n  </section>\n")
}

func renderMarkdown(project domain.Project) string {
	content := projectContent(project)
	_, sectionLabels := exportLanguage(project.Brief.Language)
	var output strings.Builder
	if strings.TrimSpace(content.Title) != "" {
		output.WriteString("# ")
		output.WriteString(markdownText(content.Title))
		output.WriteString("\n\n")
	}
	if strings.TrimSpace(content.SummaryBox) != "" {
		for _, line := range strings.Split(content.SummaryBox, "\n") {
			output.WriteString("> ")
			output.WriteString(markdownText(line))
			output.WriteByte('\n')
		}
		output.WriteByte('\n')
	}
	output.WriteString(htmlToMarkdown(htmlutil.Sanitize(content.MainContentHTML)))
	if len(content.KeyTakeaways) > 0 {
		output.WriteString("\n## ")
		output.WriteString(sectionLabels.takeaways)
		output.WriteString("\n\n")
		for _, item := range content.KeyTakeaways {
			output.WriteString("- ")
			output.WriteString(markdownText(item.Statement))
			writeMarkdownSourceIDs(&output, item.SourceIDs)
			output.WriteByte('\n')
		}
	}
	if len(content.FAQData) > 0 {
		output.WriteString("\n## ")
		output.WriteString(sectionLabels.faq)
		output.WriteString("\n\n")
		for _, item := range content.FAQData {
			output.WriteString("### ")
			output.WriteString(markdownText(item.Question))
			output.WriteString("\n\n")
			output.WriteString(markdownText(item.Answer))
			writeMarkdownSourceIDs(&output, item.SourceIDs)
			output.WriteString("\n\n")
		}
	}
	if len(project.Brief.Evidence) > 0 {
		output.WriteString("## ")
		output.WriteString(sectionLabels.sources)
		output.WriteString("\n\n")
		for _, source := range project.Brief.Evidence {
			output.WriteString("- **")
			output.WriteString(markdownText(source.ID))
			output.WriteString(":** ")
			if normalizedURL, ok := domain.NormalizeGroundingURL(source.URL); ok {
				output.WriteByte('[')
				output.WriteString(markdownText(source.Title))
				output.WriteString("](")
				output.WriteString(strings.ReplaceAll(normalizedURL, ")", "%29"))
				output.WriteByte(')')
			} else {
				output.WriteString(markdownText(source.Title))
			}
			if strings.TrimSpace(source.Notes) != "" {
				output.WriteString(" — ")
				output.WriteString(markdownText(source.Notes))
			}
			output.WriteByte('\n')
		}
	}
	groundingSources := domain.NormalizeGroundingSources(project.GroundingSources)
	if len(groundingSources) > 0 {
		output.WriteString("\n## ")
		output.WriteString(sectionLabels.groundingSources)
		output.WriteString("\n\n")
		for _, source := range groundingSources {
			label := firstNonEmpty(source.Title, source.URL)
			output.WriteString("- [")
			output.WriteString(markdownText(label))
			output.WriteString("](")
			output.WriteString(strings.ReplaceAll(source.URL, ")", "%29"))
			output.WriteString(")\n")
		}
	}
	return strings.TrimSpace(output.String()) + "\n"
}

func writeMarkdownSourceIDs(output *strings.Builder, ids []string) {
	if len(ids) > 0 {
		output.WriteString(" _[Sources: ")
		output.WriteString(markdownText(strings.Join(ids, ", ")))
		output.WriteString("]_")
	}
}

func htmlToMarkdown(value string) string {
	context := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(value), context)
	if err != nil {
		return htmlutil.Text(value) + "\n"
	}
	var output strings.Builder
	for _, node := range nodes {
		renderMarkdownNode(&output, node, 0)
	}
	return strings.TrimSpace(collapseBlankLines(output.String())) + "\n"
}

func renderMarkdownNode(output *strings.Builder, node *html.Node, listDepth int) {
	if node.Type == html.TextNode {
		output.WriteString(markdownInlineText(node.Data))
		return
	}
	if node.Type != html.ElementNode {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderMarkdownNode(output, child, listDepth)
		}
		return
	}
	tag := strings.ToLower(node.Data)
	switch tag {
	case "h2":
		output.WriteString("\n\n## ")
		renderMarkdownChildren(output, node, listDepth)
		output.WriteString("\n\n")
	case "h3":
		output.WriteString("\n\n### ")
		renderMarkdownChildren(output, node, listDepth)
		output.WriteString("\n\n")
	case "h4":
		output.WriteString("\n\n#### ")
		renderMarkdownChildren(output, node, listDepth)
		output.WriteString("\n\n")
	case "p", "figure", "figcaption":
		output.WriteString("\n\n")
		renderMarkdownChildren(output, node, listDepth)
		output.WriteString("\n\n")
	case "strong", "b":
		output.WriteString("**")
		renderMarkdownChildren(output, node, listDepth)
		output.WriteString("**")
	case "em", "i":
		output.WriteByte('_')
		renderMarkdownChildren(output, node, listDepth)
		output.WriteByte('_')
	case "code":
		inlineCode := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "`", "&#96;").Replace(plainNodeText(node))
		output.WriteByte('`')
		output.WriteString(inlineCode)
		output.WriteByte('`')
	case "pre":
		preformatted := strings.Trim(plainNodeText(node), "\r\n")
		fence := codeFence(preformatted)
		output.WriteString("\n\n")
		output.WriteString(fence)
		output.WriteByte('\n')
		output.WriteString(preformatted)
		output.WriteByte('\n')
		output.WriteString(fence)
		output.WriteString("\n\n")
	case "blockquote":
		text := strings.TrimSpace(plainNodeText(node))
		output.WriteString("\n\n")
		for _, line := range strings.Split(text, "\n") {
			output.WriteString("> ")
			output.WriteString(markdownText(line))
			output.WriteByte('\n')
		}
		output.WriteString("\n\n")
	case "ul", "ol":
		output.WriteByte('\n')
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderMarkdownNode(output, child, listDepth+1)
		}
		output.WriteByte('\n')
	case "li":
		output.WriteString(strings.Repeat("  ", max(0, listDepth-1)))
		output.WriteString("- ")
		renderMarkdownChildren(output, node, listDepth)
		output.WriteByte('\n')
	case "a":
		var label strings.Builder
		renderMarkdownChildren(&label, node, listDepth)
		href := attribute(node, "href")
		if href != "" {
			output.WriteByte('[')
			output.WriteString(label.String())
			output.WriteString("](")
			output.WriteString(strings.ReplaceAll(href, ")", "%29"))
			output.WriteByte(')')
		} else {
			output.WriteString(label.String())
		}
	case "br":
		output.WriteString("  \n")
	case "hr":
		output.WriteString("\n\n---\n\n")
	case "table":
		output.WriteString("\n\n")
		output.WriteString(renderHTMLNode(node))
		output.WriteString("\n\n")
	case "tr":
		renderMarkdownChildren(output, node, listDepth)
		output.WriteByte('\n')
	case "th", "td":
		output.WriteString(" | ")
		renderMarkdownChildren(output, node, listDepth)
	default:
		renderMarkdownChildren(output, node, listDepth)
	}
}

func renderMarkdownChildren(output *strings.Builder, node *html.Node, listDepth int) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		renderMarkdownNode(output, child, listDepth)
	}
}

func renderHTMLNode(node *html.Node) string {
	var output bytes.Buffer
	if err := html.Render(&output, node); err != nil {
		return ""
	}
	return output.String()
}

func plainNodeText(node *html.Node) string {
	var output strings.Builder
	var visit func(*html.Node)
	visit = func(current *html.Node) {
		if current.Type == html.TextNode {
			output.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return output.String()
}

func attribute(node *html.Node, key string) string {
	for _, item := range node.Attr {
		if strings.EqualFold(item.Key, key) {
			return item.Val
		}
	}
	return ""
}

func collapseBlankLines(value string) string {
	for strings.Contains(value, "\n\n\n") {
		value = strings.ReplaceAll(value, "\n\n\n", "\n\n")
	}
	return value
}

func normalizeInlineWhitespace(value string) string {
	var output strings.Builder
	wasSpace := false
	for _, character := range value {
		if character == '\r' || character == '\n' || character == '\t' || character == ' ' {
			if !wasSpace {
				output.WriteByte(' ')
				wasSpace = true
			}
			continue
		}
		output.WriteRune(character)
		wasSpace = false
	}
	return output.String()
}

func markdownInlineText(value string) string {
	return escapeMarkdownText(normalizeInlineWhitespace(value))
}

func markdownText(value string) string {
	return escapeMarkdownText(strings.TrimSpace(value))
}

func escapeMarkdownText(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\\", "\\\\",
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
		"`", "\\`",
	)
	return replacer.Replace(value)
}

func codeFence(value string) string {
	longest := 0
	current := 0
	for _, character := range value {
		if character == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	return strings.Repeat("`", max(3, longest+1))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func projectContent(project domain.Project) domain.GeneratedContent {
	if project.Content == nil {
		return domain.GeneratedContent{}
	}
	return *project.Content
}

type labels struct {
	summary          string
	takeaways        string
	faq              string
	sources          string
	groundingSources string
}

func exportLanguage(value string) (string, labels) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(normalized, "th") || strings.Contains(normalized, "thai") || strings.Contains(normalized, "ไทย") {
		return "th", labels{summary: "สรุป", takeaways: "ประเด็นสำคัญ", faq: "คำถามที่พบบ่อย", sources: "แหล่งข้อมูล", groundingSources: "แหล่งข้อมูลจากการค้นหา"}
	}
	if strings.HasPrefix(normalized, "en") || strings.Contains(normalized, "english") || strings.Contains(normalized, "อังกฤษ") {
		return "en", labels{summary: "Summary", takeaways: "Key takeaways", faq: "Frequently asked questions", sources: "Sources", groundingSources: "Grounding sources"}
	}
	if normalized == "" {
		normalized = "th"
	}
	return normalized, labels{summary: "Summary", takeaways: "Key takeaways", faq: "Frequently asked questions", sources: "Sources", groundingSources: "Grounding sources"}
}
