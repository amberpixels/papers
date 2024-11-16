// Package jalapeno is a library that provides Markdown -> Notion conversion
package jalapeno

import (
	"errors"
	"fmt"
	"log/slog"

	nt "github.com/jomei/notionapi"
	md "github.com/yuin/goldmark"
	mdast "github.com/yuin/goldmark/ast"
	mdastx "github.com/yuin/goldmark/extension/ast"
	mdtext "github.com/yuin/goldmark/text"
)

// Parser stands for an instance
type Parser struct {
	mdParser md.Markdown
}

func NewParser(mdParser md.Markdown) *Parser {
	return &Parser{mdParser: mdParser}
}

// ParsePage parses the given markdown source into blocks and properties of a Notion page
func (p *Parser) ParsePage(source []byte) (nt.Blocks, nt.Properties, error) {
	tree := p.mdParser.Parser().Parse(mdtext.NewReader(source))

	blockBuilders := make(NtBlockBuilders, 0)
	err := mdast.Walk(tree, func(node mdast.Node, entering bool) (mdast.WalkStatus, error) {
		if !entering || node.Kind() == mdast.KindDocument {
			return mdast.WalkContinue, nil
		}

		blockBuilders = append(blockBuilders, MdNode2NtBlocks(node)...)

		return mdast.WalkSkipChildren, nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to walk parsed Markdown AST: %w", err)
	}

	blocks := blockBuilders.Build(source)

	// TODO(amberpixels): handle headings equality spread (H1-H6 of markdown) spread into H1-H3 of notion
	//                   The thing should be configurable

	var pageTitle []nt.RichText
	if len(blocks) > 0 {
		for i, block := range blocks {
			if block.GetType() == nt.BlockTypeHeading1 {
				pageTitle = block.(*nt.Heading1Block).Heading1.RichText // nolint:errcheck
				// delete this block
				blocks = append(blocks[:i], blocks[i+1:]...)
				break
			}
		}
	}
	if len(pageTitle) == 0 {
		// TODO(amberpixels): default title should be configurable
		pageTitle = []nt.RichText{{Text: &nt.Text{Content: "Unnamed Document"}}}
	}

	properties := nt.Properties{
		string(nt.PropertyConfigTypeTitle): nt.TitleProperty{
			Title: pageTitle,
		},
	}

	return blocks, properties, nil
}

var (
	// ErrMustBeNtBlock is returned when a given node can't be parsed as RichText but is a separate notion block
	ErrMustBeNtBlock = errors.New("given node must be a separate notion block")

	// ErrMdNodeNotSupported is returned when a given markdown node is not supported
	ErrMdNodeNotSupported = errors.New("given markdown node is not supported")
)

// ToRichText returns a NtRichTextBuilder for a given node
// RichTextConstructor then can be called with a given source to construct a ready-to-use notion RichText object
// When given node can't be constructed as a RichText, ErrMustBeNtBlock is returned
func ToRichText(node mdast.Node) (*NtRichTextBuilder, error) {
	switch v := node.(type) {
	case *mdast.Heading:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			return &nt.RichText{
				Type: nt.ObjectTypeText,
				Text: &nt.Text{Content: string(contentFromLines(v, source))},
			}
		}), nil

	case *mdast.Text:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			return &nt.RichText{
				Type: nt.ObjectTypeText,
				Text: &nt.Text{Content: string(v.Value(source))},
			}
		}), nil
	case *mdast.FencedCodeBlock:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			return &nt.RichText{
				Type: nt.ObjectTypeText,
				Text: &nt.Text{Content: string(contentFromLines(v, source))},
			}
		}), nil
	case *mdast.AutoLink:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			link := string(v.URL(source))
			label := string(v.Label(source))

			return &nt.RichText{
				Type: nt.ObjectTypeText,
				Text: &nt.Text{
					Content: label,
					Link:    &nt.Link{Url: link},
				}}
		}), nil
	case *mdast.RawHTML:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			content := html2notion(
				string(contentFromSegments(v.Segments, source)),
			)

			return &nt.RichText{
				Type: nt.ObjectTypeText,
				Text: &nt.Text{Content: content},
			}
		}), nil
	case *mdast.HTMLBlock:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			content := html2notion(
				string(contentFromLines(v, source)),
			)

			return &nt.RichText{
				Type: nt.ObjectTypeText,
				Text: &nt.Text{Content: content},
			}
		}), nil
	case *mdast.Image:
		return nil, ErrMustBeNtBlock
	default:
		return nil, fmt.Errorf("%w: %s", ErrMdNodeNotSupported, node.Kind().String())
	}
}

// flatten flattens given MD ast node into series of Notion RichTexts and (optionally) Blocks.
// RichTexts and Blocks are returned as builders, so later they can be built with given source bytes.
// Flattening is a required process because Markdown deeply nested can be shown as flat notion blocks or rich texts.
//
// Examples:
//
//   - Markdown's Header (with all its deep children) can only be flat Notion's Header with rich texts inside. /
//
//   - Markdown's Image (with possible children in its title) can only be Notion's Block
//
// TODO(amberpixels): consider refactoring as this function should be split into two: on for rich text and one for block
//   - this can be achieved if we have a knowledge on how each mdast.Node should be converted.
//
// nolint: gocyclo // Will be OK after refactor
func flatten(node mdast.Node) (richTexts NtRichTextBuilders, blocks NtBlockBuilders) {
	richTexts = make(NtRichTextBuilders, 0)
	blocks = make(NtBlockBuilders, 0)

	// Final point: If no has no children, try to get its content via Lines, Segment, etc
	if node.ChildCount() == 0 {
		richText, err := ToRichText(node)
		if err != nil && !errors.Is(err, ErrMustBeNtBlock) {
			panic(err)
		}

		if richText != nil {
			richTexts = append(richTexts, richText)
		}

		if errors.Is(err, ErrMustBeNtBlock) {
			switch v := node.(type) {
			case *mdast.Image:
				blocks = append(blocks, func(_ []byte) nt.Block {
					// Source is not used here :)
					img := nt.Image{
						Type: nt.FileTypeExternal,
						External: &nt.FileObject{
							URL: string(v.Destination),
						},
					}
					if v.Title != nil { // here, v.Title is probably will always be empty, but anyway
						img.Caption = []nt.RichText{
							{
								Type: nt.ObjectTypeText,
								Text: &nt.Text{Content: string(v.Title)},
							},
						}
					}

					return &nt.ImageBlock{
						BasicBlock: nt.BasicBlock{
							Object: nt.ObjectTypeBlock,
							Type:   nt.BlockTypeImage,
						},
						Image: img,
					}
				})

			default:
				panic(fmt.Sprintf("-> unhandled final node type: %s", node.Kind().String()))
			}

			// handle as image
		}

		return
	}

	// If has children: recursively iterate and flatten results
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {

		// Flatten children of current child
		deeperRichTexts, deeperBlocks := flatten(child)

		blocks = append(blocks, deeperBlocks...)

		// Special handling depending on the type of the child
		switch v := child.(type) {
		case *mdastx.Strikethrough:
			for i := range deeperRichTexts {
				deeperRichTexts[i].DecorateWith(strikethroughDecorator)
			}
		case *mdast.Emphasis:
			for i := range deeperRichTexts {
				if v.Level == 1 {
					deeperRichTexts[i].DecorateWith(italicDecorator)
				} else {
					deeperRichTexts[i].DecorateWith(boldDecorator)
				}
			}
		case *mdast.CodeSpan:
			// Adding t.Annotations = code:true for each child
			for i := range deeperRichTexts {
				deeperRichTexts[i].DecorateWith(codeDecorator)
			}

		case *mdast.Link:
			for i := range deeperRichTexts {
				deeperRichTexts[i].DecorateWith(linkDecorator(string(v.Destination)))
			}

		case *mdast.Image:
			// make a copy of the rich texts inside, as they will become Image Caption
			// but we nil-ify the original rich texts as to prevent them from duplicating
			captionRichTexts := append(NtRichTextBuilders{}, deeperRichTexts...)
			deeperRichTexts = nil
			blocks = append(blocks, func(source []byte) nt.Block {
				return &nt.ImageBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeImage,
					},
					Image: nt.Image{
						Type: nt.FileTypeExternal,
						External: &nt.FileObject{
							URL: string(v.Destination),
						},
						// TODO(amberpixels): in case if image had a link parent, we need to do caption as link
						Caption: captionRichTexts.Build(source),
					},
				}
			})

		case *mdast.Text, *mdast.RawHTML, *mdast.AutoLink:
		// we're fine here
		default:
			slog.Warn(fmt.Sprintf("Unhandled child's type: %s", v.Kind().String()))
		}

		// Appending flattened children
		richTexts = append(richTexts, deeperRichTexts...)
	}

	return richTexts, blocks
}

func MdNode2NtBlocks(node mdast.Node) NtBlockBuilders {
	switch node.Kind() {
	case mdast.KindHeading:
		// Although in MD mdast.Heading is respresented via deeply nested tree of objects
		// In notion it should be a flattened list of RichTexts (With no children)
		// Edge case: Notion's heading.collapseable=true (that supports children) is not supported yet
		//            TODO(amberpixels): create an issue for it
		richTexts, _ := flatten(node)

		slog.Debug(fmt.Sprintf("MD mdast.Heading flattened into %d nt-rich-texts", len(richTexts)))

		switch node.(*mdast.Heading).Level { // nolint:errcheck
		case 1:
			return []NtBlockBuilder{
				func(source []byte) nt.Block {
					return &nt.Heading1Block{BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeHeading1,
					}, Heading1: nt.Heading{RichText: richTexts.Build(source)}}
				},
			}
		case 2:
			return []NtBlockBuilder{
				func(source []byte) nt.Block {
					return &nt.Heading2Block{BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeHeading2,
					}, Heading2: nt.Heading{RichText: richTexts.Build(source)}}
				},
			}
		default:
			return []NtBlockBuilder{
				func(source []byte) nt.Block {
					return &nt.Heading3Block{BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeHeading3,
					}, Heading3: nt.Heading{RichText: richTexts.Build(source)}}
				},
			}
		}
	case mdast.KindParagraph:
		richTexts, blocks := flatten(node)

		slog.Debug(fmt.Sprintf("MD mdast.Heading flattened into %d nt-rich-texts and %d nt-blocks", len(richTexts), len(blocks)))

		if len(richTexts) == 0 && len(blocks) > 0 {
			return blocks
		}

		return []NtBlockBuilder{
			func(source []byte) nt.Block {
				return &nt.ParagraphBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeParagraph,
					},
					Paragraph: nt.Paragraph{
						RichText: richTexts.Build(source),
						Children: blocks.Build(source), // TODO: NOT SURE IF THIS IS CORRECT
					},
				}
			},
		}
	case mdast.KindFencedCodeBlock:
		richTexts, _ := flatten(node)
		return []NtBlockBuilder{
			func(source []byte) nt.Block {
				codeBlock := node.(*mdast.FencedCodeBlock) // nolint:errcheck
				return &nt.CodeBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeCode,
					},
					Code: nt.Code{
						Language: sanitizeBlockLanguage(string(codeBlock.Language(source))),
						RichText: richTexts.Build(source),
					},
				}
			},
		}
	case mdast.KindHTMLBlock:
		richTexts, _ := flatten(node)

		return []NtBlockBuilder{
			func(source []byte) nt.Block {
				return &nt.ParagraphBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeParagraph,
					},
					Paragraph: nt.Paragraph{
						RichText: richTexts.Build(source),
					},
				}
			},
		}
	case mdast.KindList:
		list := node.(*mdast.List) // // nolint:errcheck
		isBulletedList := list.Marker == '-' || list.Marker == '+'

		result := make(NtBlockBuilders, 0)
		for mdItem := node.FirstChild(); mdItem != nil; mdItem = mdItem.NextSibling() {
			flattenedRichTexts, _ := flatten(mdItem)

			if isBulletedList {
				result = append(result, func(source []byte) nt.Block {
					return &nt.BulletedListItemBlock{
						BasicBlock: nt.BasicBlock{
							Object: nt.ObjectTypeBlock,
							Type:   nt.BlockTypeBulletedListItem,
						},
						BulletedListItem: nt.ListItem{
							RichText: flattenedRichTexts.Build(source),
						},
					}
				})
			} else {
				result = append(result, func(source []byte) nt.Block {
					return &nt.NumberedListItemBlock{
						BasicBlock: nt.BasicBlock{
							Object: nt.ObjectTypeBlock,
							Type:   nt.BlockTypeNumberedListItem,
						},
						NumberedListItem: nt.ListItem{
							RichText: flattenedRichTexts.Build(source),
						},
					}
				})
			}
		}
		return result
	case mdastx.KindTable: // Use the extension AST for the Table node
		table := node.(*mdastx.Table) // nolint:errcheck

		// Collect headers and rows
		headers := make([]NtRichTextBuilders, 0)
		rows := make([][]NtRichTextBuilders, 0)

		// Iterate over the table's children to extract headers and rows
		for tr := table.FirstChild(); tr != nil; tr = tr.NextSibling() {
			switch tr.Kind() {
			case mdastx.KindTableHeader:
				// Collect headers
				for th := tr.FirstChild(); th != nil; th = th.NextSibling() {
					richTexts, _ := flatten(th)
					headers = append(headers, richTexts)
				}

			case mdastx.KindTableRow:
				// Collect each row's cells
				row := make([]NtRichTextBuilders, 0)
				for td := tr.FirstChild(); td != nil; td = td.NextSibling() {
					richTexts, _ := flatten(td)
					row = append(row, richTexts)
				}
				rows = append(rows, row)
			}
		}

		// Create Notion table block
		return []NtBlockBuilder{
			func(source []byte) nt.Block {
				// Construct table block
				tableBlock := &nt.TableBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeTableBlock,
					},
					Table: nt.Table{
						TableWidth:      len(headers),
						HasColumnHeader: true,
						Children:        nt.Blocks{}, // will be populated below

						//HasRowHeader:  false, // TODO(amberpixels) is this possible to be known from markdown?
					},
				}

				// Populate header row
				if len(headers) > 0 {
					headerRow := nt.TableRow{
						Cells: make([][]nt.RichText, len(headers)),
					}
					for i, header := range headers {
						headerRow.Cells[i] = header.Build(source)
					}
					tableBlock.Table.Children = append(tableBlock.Table.Children, &nt.TableRowBlock{
						BasicBlock: nt.BasicBlock{
							Object: nt.ObjectTypeBlock,
							Type:   nt.BlockTypeTableRowBlock,
						},
						TableRow: headerRow,
					})
				}

				// Populate the rest of the rows
				for _, row := range rows {
					tableRow := nt.TableRow{
						Cells: make([][]nt.RichText, len(row)),
					}
					for i, cell := range row {
						tableRow.Cells[i] = cell.Build(source)
					}
					tableBlock.Table.Children = append(tableBlock.Table.Children, &nt.TableRowBlock{
						BasicBlock: nt.BasicBlock{
							Object: nt.ObjectTypeBlock,
							Type:   nt.BlockTypeTableRowBlock,
						},
						TableRow: tableRow,
					})
				}

				return tableBlock
			},
		}
	default:
		panic(fmt.Sprintf("unhandled node type: %s", node.Kind().String()))
	}
}
