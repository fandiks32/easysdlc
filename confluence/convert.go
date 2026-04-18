package confluence

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// ConvertStorageToMarkdown converts Confluence XHTML storage format to Markdown.
func ConvertStorageToMarkdown(xhtml string) string {
	doc, err := html.Parse(strings.NewReader(xhtml))
	if err != nil {
		// If parsing fails, return the raw content stripped of tags.
		return stripTags(xhtml)
	}

	var b strings.Builder
	walkNode(&b, doc, &convertState{})
	return strings.TrimSpace(b.String())
}

type convertState struct {
	listDepth int
	ordered   bool
	itemIndex int
	inPre     bool
	inTable   bool
	tableRow  []string
}

func walkNode(b *strings.Builder, n *html.Node, state *convertState) {
	switch n.Type {
	case html.TextNode:
		text := n.Data
		if !state.inPre {
			// Collapse whitespace in non-pre contexts.
			text = collapseWhitespace(text)
		}
		b.WriteString(text)
		return
	case html.ElementNode:
		// handled below
	default:
		// Recurse into children for document/fragment nodes.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkNode(b, c, state)
		}
		return
	}

	tag := normalizeTag(n)

	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		level := int(tag[1] - '0')
		b.WriteString("\n\n")
		b.WriteString(strings.Repeat("#", level))
		b.WriteString(" ")
		walkChildren(b, n, state)
		b.WriteString("\n\n")

	case "p":
		b.WriteString("\n\n")
		walkChildren(b, n, state)
		b.WriteString("\n\n")

	case "br":
		b.WriteString("\n")

	case "strong", "b":
		b.WriteString("**")
		walkChildren(b, n, state)
		b.WriteString("**")

	case "em", "i":
		b.WriteString("*")
		walkChildren(b, n, state)
		b.WriteString("*")

	case "code":
		if !state.inPre {
			b.WriteString("`")
			walkChildren(b, n, state)
			b.WriteString("`")
		} else {
			walkChildren(b, n, state)
		}

	case "pre":
		state.inPre = true
		b.WriteString("\n\n```\n")
		walkChildren(b, n, state)
		b.WriteString("\n```\n\n")
		state.inPre = false

	case "a":
		href := getAttr(n, "href")
		b.WriteString("[")
		walkChildren(b, n, state)
		b.WriteString("](")
		b.WriteString(href)
		b.WriteString(")")

	case "ul":
		prevOrdered := state.ordered
		prevIndex := state.itemIndex
		state.ordered = false
		state.listDepth++
		state.itemIndex = 0
		b.WriteString("\n")
		walkChildren(b, n, state)
		state.listDepth--
		state.ordered = prevOrdered
		state.itemIndex = prevIndex
		b.WriteString("\n")

	case "ol":
		prevOrdered := state.ordered
		prevIndex := state.itemIndex
		state.ordered = true
		state.listDepth++
		state.itemIndex = 0
		b.WriteString("\n")
		walkChildren(b, n, state)
		state.listDepth--
		state.ordered = prevOrdered
		state.itemIndex = prevIndex
		b.WriteString("\n")

	case "li":
		indent := strings.Repeat("  ", state.listDepth-1)
		if state.ordered {
			state.itemIndex++
			b.WriteString(fmt.Sprintf("%s%d. ", indent, state.itemIndex))
		} else {
			b.WriteString(indent + "- ")
		}
		walkChildren(b, n, state)
		b.WriteString("\n")

	case "table":
		state.inTable = true
		b.WriteString("\n\n")
		walkChildren(b, n, state)
		b.WriteString("\n")
		state.inTable = false

	case "thead", "tbody", "tfoot":
		walkChildren(b, n, state)

	case "tr":
		state.tableRow = nil
		walkChildren(b, n, state)
		if len(state.tableRow) > 0 {
			b.WriteString("| ")
			b.WriteString(strings.Join(state.tableRow, " | "))
			b.WriteString(" |\n")
			// Emit separator after header row (first row).
			if isHeaderRow(n) {
				b.WriteString("|")
				for range state.tableRow {
					b.WriteString(" --- |")
				}
				b.WriteString("\n")
			}
		}

	case "th", "td":
		var cell strings.Builder
		walkNode(&cell, wrapChildren(n), state)
		state.tableRow = append(state.tableRow, strings.TrimSpace(cell.String()))

	// Confluence structured macros (code blocks, info panels, etc.)
	case "ac:structured-macro":
		macroName := getAttr(n, "ac:name")
		switch macroName {
		case "code", "noformat":
			lang := getMacroParam(n, "language")
			b.WriteString("\n\n```")
			b.WriteString(lang)
			b.WriteString("\n")
			body := getMacroBody(n)
			b.WriteString(body)
			b.WriteString("\n```\n\n")
		case "info", "note", "warning", "tip":
			b.WriteString("\n\n> **")
			b.WriteString(strings.ToUpper(macroName))
			b.WriteString(":** ")
			body := getMacroRichBody(n)
			b.WriteString(strings.ReplaceAll(body, "\n", "\n> "))
			b.WriteString("\n\n")
		default:
			// For unknown macros, extract any text content.
			walkChildren(b, n, state)
		}

	case "ac:plain-text-body":
		walkChildren(b, n, state)

	case "ac:rich-text-body":
		walkChildren(b, n, state)

	case "ac:parameter", "ac:image", "ac:link":
		// Skip macro parameters and standalone images/links in rendering.

	case "img", "ac:emoticon":
		alt := getAttr(n, "alt")
		if alt == "" {
			alt = getAttr(n, "ac:name")
		}
		if alt != "" {
			b.WriteString(alt)
		}

	case "blockquote":
		var content strings.Builder
		walkChildren(&content, n, state)
		lines := strings.Split(strings.TrimSpace(content.String()), "\n")
		b.WriteString("\n\n")
		for _, line := range lines {
			b.WriteString("> ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")

	case "hr":
		b.WriteString("\n\n---\n\n")

	case "span", "div", "section", "article", "main", "header", "footer",
		"nav", "aside", "figure", "figcaption", "details", "summary":
		walkChildren(b, n, state)

	default:
		walkChildren(b, n, state)
	}
}

func walkChildren(b *strings.Builder, n *html.Node, state *convertState) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkNode(b, c, state)
	}
}

// wrapChildren creates a virtual parent that holds n's children for isolated rendering.
func wrapChildren(n *html.Node) *html.Node {
	wrapper := &html.Node{Type: html.DocumentNode}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		clone := *c
		clone.Parent = wrapper
		clone.PrevSibling = nil
		clone.NextSibling = nil
		if wrapper.FirstChild == nil {
			wrapper.FirstChild = &clone
			wrapper.LastChild = &clone
		} else {
			clone.PrevSibling = wrapper.LastChild
			wrapper.LastChild.NextSibling = &clone
			wrapper.LastChild = &clone
		}
	}
	return wrapper
}

func normalizeTag(n *html.Node) string {
	if n.Namespace != "" {
		return n.Namespace + ":" + n.Data
	}
	return n.Data
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		attrKey := a.Key
		if a.Namespace != "" {
			attrKey = a.Namespace + ":" + a.Key
		}
		if attrKey == key {
			return a.Val
		}
	}
	return ""
}

// getMacroParam extracts a parameter value from an ac:structured-macro's ac:parameter children.
func getMacroParam(n *html.Node, name string) string {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && normalizeTag(c) == "ac:parameter" {
			if getAttr(c, "ac:name") == name {
				return textContent(c)
			}
		}
	}
	return ""
}

// getMacroBody extracts text from ac:plain-text-body inside a macro.
func getMacroBody(n *html.Node) string {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && normalizeTag(c) == "ac:plain-text-body" {
			return textContent(c)
		}
	}
	return ""
}

// getMacroRichBody extracts rendered text from ac:rich-text-body inside a macro.
func getMacroRichBody(n *html.Node) string {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && normalizeTag(c) == "ac:rich-text-body" {
			var b strings.Builder
			walkChildren(&b, c, &convertState{})
			return strings.TrimSpace(b.String())
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func isHeaderRow(trNode *html.Node) bool {
	for c := trNode.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "th" {
			return true
		}
	}
	return false
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}

func stripTags(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	return textContent(doc)
}
