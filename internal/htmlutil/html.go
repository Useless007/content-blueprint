package htmlutil

import (
	"bytes"
	stdhtml "html"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var safeSourceID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

var allowedElements = map[string]bool{
	"a": true, "b": true, "blockquote": true, "br": true, "code": true,
	"em": true, "figcaption": true, "figure": true, "h2": true, "h3": true,
	"h4": true, "hr": true, "i": true, "li": true, "ol": true, "p": true,
	"pre": true, "span": true, "strong": true, "sub": true, "sup": true,
	"table": true, "tbody": true, "td": true, "tfoot": true, "th": true,
	"thead": true, "tr": true, "u": true, "ul": true,
}

var blockedElements = map[string]bool{
	"applet": true, "audio": true, "button": true, "canvas": true,
	"embed": true, "form": true, "frame": true, "frameset": true,
	"iframe": true, "input": true, "link": true, "math": true,
	"meta": true, "noscript": true, "object": true, "script": true,
	"select": true, "style": true, "svg": true, "textarea": true,
	"video": true,
}

func parseFragment(value string) []*html.Node {
	context := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(value), context)
	if err != nil {
		return []*html.Node{{Type: html.TextNode, Data: value}}
	}
	return nodes
}

func Text(value string) string {
	var builder strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && blockedElements[strings.ToLower(node.Data)] {
			return
		}
		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
			builder.WriteByte(' ')
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	for _, node := range parseFragment(value) {
		visit(node)
	}
	return strings.Join(strings.Fields(stdhtml.UnescapeString(builder.String())), " ")
}

// SourceIDs returns de-duplicated evidence IDs from exact inline citation
// markers such as <sup data-source-id="S1">[S1]</sup>. Lookalike elements or
// markers whose visible label does not match the attribute are ignored.
func SourceIDs(value string) []string {
	result := make([]string, 0)
	seen := make(map[string]struct{})
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && blockedElements[strings.ToLower(node.Data)] {
			return
		}
		if node.Type == html.ElementNode && strings.EqualFold(node.Data, "sup") {
			id := ""
			for _, attribute := range node.Attr {
				if strings.EqualFold(attribute.Key, "data-source-id") {
					id = strings.TrimSpace(attribute.Val)
					break
				}
			}
			if id != "" && node.FirstChild != nil && node.FirstChild == node.LastChild &&
				node.FirstChild.Type == html.TextNode && strings.TrimSpace(node.FirstChild.Data) == "["+id+"]" {
				if _, exists := seen[id]; !exists {
					seen[id] = struct{}{}
					result = append(result, id)
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	for _, node := range parseFragment(value) {
		visit(node)
	}
	return result
}

// Sanitize keeps a deliberately small set of article markup. Unsupported
// formatting is unwrapped, while active-content elements and their children
// are removed completely.
func Sanitize(value string) string {
	var output bytes.Buffer
	for _, node := range parseFragment(value) {
		renderSafe(&output, node)
	}
	return strings.TrimSpace(output.String())
}

func renderSafe(output *bytes.Buffer, node *html.Node) {
	switch node.Type {
	case html.TextNode:
		output.WriteString(stdhtml.EscapeString(node.Data))
	case html.ElementNode:
		tag := strings.ToLower(node.Data)
		if blockedElements[tag] {
			return
		}
		if !allowedElements[tag] {
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				renderSafe(output, child)
			}
			return
		}
		output.WriteByte('<')
		output.WriteString(tag)
		for _, attribute := range safeAttributes(tag, node.Attr) {
			output.WriteByte(' ')
			output.WriteString(attribute.Key)
			output.WriteString(`="`)
			output.WriteString(stdhtml.EscapeString(attribute.Val))
			output.WriteByte('"')
		}
		output.WriteByte('>')
		if tag == "br" || tag == "hr" {
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderSafe(output, child)
		}
		output.WriteString("</")
		output.WriteString(tag)
		output.WriteByte('>')
	default:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderSafe(output, child)
		}
	}
}

func safeAttributes(tag string, attributes []html.Attribute) []html.Attribute {
	result := make([]html.Attribute, 0, len(attributes)+1)
	for _, attribute := range attributes {
		key := strings.ToLower(attribute.Key)
		value := strings.TrimSpace(attribute.Val)
		switch {
		case tag == "a" && key == "href" && safeLink(value):
			result = append(result, html.Attribute{Key: "href", Val: value})
		case tag == "a" && key == "title":
			result = append(result, html.Attribute{Key: "title", Val: value})
		case (tag == "sup" || tag == "span") && key == "data-source-id" && safeSourceID.MatchString(value):
			result = append(result, html.Attribute{Key: "data-source-id", Val: value})
		case (tag == "td" || tag == "th") && (key == "colspan" || key == "rowspan") && numericSpan(value):
			result = append(result, html.Attribute{Key: key, Val: value})
		}
	}
	if tag == "a" {
		result = append(result, html.Attribute{Key: "rel", Val: "noopener noreferrer"})
	}
	return result
}

func safeLink(value string) bool {
	if strings.Contains(value, "\\") || strings.HasPrefix(value, "//") {
		return false
	}
	if strings.HasPrefix(value, "#") || strings.HasPrefix(value, "/") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return parsed.Host != "" && parsed.User == nil
	case "mailto":
		return parsed.Opaque != "" && parsed.User == nil
	default:
		return parsed.Scheme == "" && !strings.HasPrefix(value, "//")
	}
}

func numericSpan(value string) bool {
	if value == "" || len(value) > 2 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return value != "0"
}
