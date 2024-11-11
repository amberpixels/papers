// Package md2nt provides a function to convert markdown (parsed) into a Notion block tree
package md2nt

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jomei/notionapi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	astExt "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type Parser struct {
	source []byte

	md     goldmark.Markdown
	parsed ast.Node
}

func NewParser(md goldmark.Markdown) *Parser {
	return &Parser{md: md}
}

func (p *Parser) Parse(source []byte) {
	p.parsed = p.md.Parser().Parse(text.NewReader(source))
	p.source = source
}

func (p *Parser) ParseFile(filename string) error {
	source, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	p.Parse(source)
	return nil
}

func flattened(node ast.Node, source []byte) ([]notionapi.RichText, notionapi.Blocks) {
	children := make([]notionapi.Block, 0)

	// Final point: If no has no children, try to get its content via Lines, Segment, etc
	if node.ChildCount() == 0 {
		switch v := node.(type) {
		case *ast.Heading:
			return []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{Content: string(contentFromLines(v, source))},
				},
			}, nil
		case *ast.Text:
			return []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{Content: string(v.Value(source))},
				},
			}, nil
		case *ast.FencedCodeBlock:
			return []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{Content: string(contentFromLines(v, source))},
				},
			}, nil
		case *ast.AutoLink:
			link := string(v.URL(source))
			label := string(v.Label(source))
			return []notionapi.RichText{{
				Text: &notionapi.Text{
					Content: label,
					Link:    &notionapi.Link{Url: link},
				},
			}}, nil
		case *ast.RawHTML:
			content := contentFromSegments(v.Segments, source)

			return []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{Content: html2notion(string(content))},
				},
			}, nil
		case *ast.HTMLBlock:
			content := contentFromLines(v, source)

			return []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{Content: html2notion(string(content))},
				},
			}, nil
		case *ast.Image:
			dest := string(v.Destination)

			img := notionapi.Image{
				Type: notionapi.FileTypeExternal,
				External: &notionapi.FileObject{
					URL: dest,
				},
			}
			if v.Title != nil {
				img.Caption = []notionapi.RichText{
					{
						Type: notionapi.ObjectTypeText,
						Text: &notionapi.Text{Content: string(v.Title)},
					},
				}
			}

			return nil, []notionapi.Block{&notionapi.ImageBlock{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeImage,
				},
				Image: img,
			}}

		default:
			panic(fmt.Sprintf("unhandled final node type: %s", node.Kind().String()))
		}
	}

	// If has children: recursively iterate and flatten results
	richTexts := make([]notionapi.RichText, 0)
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {

		// Flatten children of current child
		flattenedRichTexts, grandChildren := flattened(child, source)

		children = append(children, grandChildren...)

		// Special handling depending on the type of the child
		switch v := child.(type) {
		case *astExt.Strikethrough:
			for i := range flattenedRichTexts {
				annotateStrikethrough(&flattenedRichTexts[i])
			}
		case *ast.Emphasis:
			// Adding t.Annotations = code:true for each child
			for i := range flattenedRichTexts {
				if v.Level == 1 {
					annotateItalic(&flattenedRichTexts[i])
				} else {
					annotateBold(&flattenedRichTexts[i])
				}
			}
		case *ast.CodeSpan:
			// Adding t.Annotations = code:true for each child
			for i := range flattenedRichTexts {
				annotateCode(&flattenedRichTexts[i])
			}

		case *ast.Link:
			for i := range flattenedRichTexts {
				attachLink(&flattenedRichTexts[i], string(v.Destination))
			}
		default:
			fmt.Println("Unhandled child's type: ", v.Kind().String())
		}

		// Appending flattened children
		richTexts = append(richTexts, flattenedRichTexts...)
	}

	return richTexts, children
}

func (p *Parser) Walk(fn func(node ast.Node, entering bool) (ast.WalkStatus, error)) error {
	return ast.Walk(p.parsed, fn)
}

func (p *Parser) ToNotionBlocks(node ast.Node) []notionapi.Block {
	switch node.Kind() {
	case ast.KindHeading:
		// Although in MD ast.Heading is respresented via deeply nested tree of objects
		// In notion it should be a flattened list of RichTexts
		// Edge case: Notion's heading.collapseable=true (that supports children) is not supported yet
		//            TODO(amberpixels): create an issue for it

		richTexts, _ := flattened(node, p.source)

		slog.Debug(fmt.Sprintf("MD Heading flattened into %d", len(richTexts)))
		for i, rt := range richTexts {
			slog.Debug(fmt.Sprintf("Heading richtext[%d]: %s", i, rt.PlainText))
		}

		switch node.(*ast.Heading).Level {
		case 1:
			return []notionapi.Block{&notionapi.Heading1Block{BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeHeading1,
			}, Heading1: notionapi.Heading{RichText: richTexts}}}
		case 2:
			return []notionapi.Block{&notionapi.Heading2Block{BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeHeading2,
			}, Heading2: notionapi.Heading{RichText: richTexts}}}
		default:
			return []notionapi.Block{&notionapi.Heading3Block{BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeHeading3,
			}, Heading3: notionapi.Heading{RichText: richTexts}}}
		}
	case ast.KindParagraph:
		richTexts, children := flattened(node, p.source)

		slog.Debug(fmt.Sprintf("MD Paragraph flattened into %d", len(richTexts)))
		for i, rt := range richTexts {
			slog.Debug(fmt.Sprintf("MD Pargraphrichtext[%d]: %s", i, rt.PlainText))
		}

		if len(richTexts) == 0 && len(children) > 0 {
			return children
		}

		return []notionapi.Block{&notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeParagraph,
			},
			Paragraph: notionapi.Paragraph{
				RichText: richTexts,
				Children: children, // TODO: NOT SURE IF THIS IS CORRECT
			},
		}}

	case ast.KindFencedCodeBlock:
		codeBlock := node.(*ast.FencedCodeBlock)

		richTexts, _ := flattened(node, p.source)

		return []notionapi.Block{&notionapi.CodeBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeCode,
			},
			Code: notionapi.Code{
				Language: sanitizeBlockLanguage(string(codeBlock.Language(p.source))),
				RichText: richTexts,
			},
		}}
	case ast.KindHTMLBlock:
		richTexts, _ := flattened(node, p.source)

		return []notionapi.Block{&notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeParagraph,
			},
			Paragraph: notionapi.Paragraph{
				RichText: richTexts,
			},
		}}
	case ast.KindList:

		list, _ := node.(*ast.List)
		isBulletedList := list.Marker == '-' || list.Marker == '+'

		result := make([]notionapi.Block, 0)
		for mdItem := node.FirstChild(); mdItem != nil; mdItem = mdItem.NextSibling() {
			flattenedRichTexts, _ := flattened(mdItem, p.source)

			if isBulletedList {
				result = append(result, &notionapi.BulletedListItemBlock{
					BasicBlock: notionapi.BasicBlock{
						Object: notionapi.ObjectTypeBlock,
						Type:   notionapi.BlockTypeBulletedListItem,
					},
					BulletedListItem: notionapi.ListItem{
						RichText: flattenedRichTexts,
					},
				})
			} else {
				result = append(result, &notionapi.NumberedListItemBlock{
					BasicBlock: notionapi.BasicBlock{
						Object: notionapi.ObjectTypeBlock,
						Type:   notionapi.BlockTypeNumberedListItem,
					},
					NumberedListItem: notionapi.ListItem{
						RichText: flattenedRichTexts,
					},
				})
			}
		}
		return result
	default:
		panic(fmt.Sprintf("unhandled node type: %s", node.Kind().String()))
	}
}

/*
	case *ast.Image:
			title := "<no-title>"
			if v.Title != nil {
				title = string(v.Title)
			}
			dest := string(v.Destination)
			return []notionapi.RichText{{
				Text: &notionapi.Text{Content: fmt.Sprintf("image%s_%s", title, dest)},
			}}, nil



		case *ast.Link:
			// For now let's support only links containing simple things, e.g. text
			contentParts := make([]string, 0)
			for _, content := range richChildren {
				contentParts = append(contentParts, content.Text.Content)
			}
			contentStr := strings.Join(contentParts, " ")
			fmt.Println("LINK : ", string(v.Destination))
			link := string(v.Destination)
			if link == "" || strings.HasPrefix(link, "#") {
				link = "https://localhost" + link
			}

			richTexts = append(richTexts, notionapi.RichText{
				Text: &notionapi.Text{
					Link: &notionapi.Link{
						Url: link,
					},
					Content: contentStr,
				},
			})
*/
