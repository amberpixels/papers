package jalapeno

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	nt "github.com/jomei/notionapi"
	"golang.org/x/net/html"
)

// html2notion converts HTML into Notion blocks
// Old comment for reference:
//
//	TODO(amberpixels): add support HTML
//	  Note: we want to support basic HTML that is usually used in Markdown:
//	  <p> (for centering), <img> (for images), <br> (for line breaks)
//	  Also we can support <b>, <i>, <s>, <code> tags
func html2notion(rawHTML string) (nt.Blocks, []nt.RichText, error) {
	// sanitizing first
	rawHTML = strings.TrimSpace(rawHTML)
	rawHTML = strings.ToLower(rawHTML)

	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return nil, nil, err
	}

	var theBody *html.Node
	htmlwalk(doc, func(node *html.Node) {
		if theBody != nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "body" {
			theBody = node
			return
		}
	})

	var blocksExist bool
	htmlwalk(theBody, func(node *html.Node) {
		if blocksExist {
			return
		}
		if node.Type != html.ElementNode || node.Data == "body" {
			return
		}
		if !isInlineTag(node.Data) {
			blocksExist = true
			return
		}
	})

	if !blocksExist {
		richTexts := make([]nt.RichText, 0)
		htmlwalk(theBody, func(node *html.Node) {
			if node.Type != html.ElementNode || node.Data == "body" {
				return
			}
			rt := htmlNodeToRichTexts(node)
			if rt == nil {
				return
			}

			richTexts = append(richTexts, rt...)
		})

		return nil, richTexts, nil
	}

	blocks := nt.Blocks{}
	htmlwalk(theBody, func(node *html.Node) {
		if node.Type != html.ElementNode || node.Data == "body" {
			return
		}

		block := htmlNodeToBlock(node)
		if block == nil {
			return
		}

		blocks = append(blocks, block)
	})

	return blocks, nil, nil
}

func htmlwalk(node *html.Node, process func(*html.Node)) {
	process(node)

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		htmlwalk(child, process)
	}
}

// htmlNodeToBlock maps HTML elements to Notion blocks
func htmlNodeToBlock(node *html.Node) nt.Block {
	switch node.Data {
	case "p", "div":
		rts := make([]nt.RichText, 0)
		htmlwalk(node, func(n *html.Node) {
			if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "div") {
				return
			}
			if n.Type != html.ElementNode && n.Type != html.TextNode {
				return
			}

			rts = append(rts, htmlNodeToRichTexts(n)...)
		})
		return &nt.ParagraphBlock{
			Paragraph: nt.Paragraph{
				RichText: rts,
			},
		}
	case "h1", "h2", "h3", "h4", "h5", "h6":
		lvl, _ := strconv.Atoi(strings.TrimPrefix(node.Data, "h"))

		// Handle headers (same logic as paragraphs for alignment)
		return nt.NewHeadingBlock(nt.Heading{
			RichText: htmlNodeToRichTexts(node),
		}, lvl)
	default:
		return nil
	}
}

func htmlNodeToRichText(node *html.Node) *nt.RichText {
	switch node.Type {
	case html.TextNode:
		text := strings.TrimSpace(node.Data)
		if text == "" {
			return nil
		}

		return nt.NewTextRichText(text)
	case html.ElementNode:
		// Very simple logic for now: todo: support styling via attributes and css
		switch node.Data {
		case "strong", "b":
			return nt.NewTextRichText(extractText(node)).AnnotateBold()
		case "em", "i":
			return nt.NewTextRichText(extractText(node)).AnnotateItalic()
		case "span":
			return nt.NewTextRichText(extractText(node))
		default:
			fmt.Println("unspported HTML data ", node.Data)
			return nil
		}
	default:
		fmt.Println("Unsupported HTML type ", node.Type)
		return nil
	}
}

func htmlNodeToRichTexts(node *html.Node) []nt.RichText {
	v := htmlNodeToRichText(node)
	return []nt.RichText{*v}
}

// extractText extracts all plain text from a node
func extractText(node *html.Node) string {
	var buffer bytes.Buffer
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			buffer.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(node)
	return buffer.String()
}

// isInlineTag checks if an HTML tag is an inline element
func isInlineTag(tag string) bool {
	switch tag {
	case "strong", "b", "em", "i", "span":
		return true
	}
	return false
}
