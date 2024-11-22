// Package jalapeno is a library that provides Markdown -> Notion conversion
package jalapeno

import (
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

		blockBuilders = append(blockBuilders, ToBlocks(node)...)

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
		pageTitle = []nt.RichText{
			*nt.NewTextRichText("Unnamed Document"),
		}
	}

	properties := nt.Properties{
		string(nt.PropertyConfigTypeTitle): nt.TitleProperty{
			Title: pageTitle,
		},
	}

	return blocks, properties, nil
}

// IsConvertableToBlock returns if given Markdown AST node is convertable to notionapi Block
// If not, it means it might be converted into RichText directly (and used as contents of Paragraph block for example)
func IsConvertableToBlock(node mdast.Node) bool {
	switch node.(type) {
	case *mdast.Image, *mdastx.TaskCheckBox:
		return true
	default:
		return false
	}
}

// IsConvertableToRichText returns true if given Markdown AST node is convertable directly into notion RichText
func IsConvertableToRichText(node mdast.Node) bool {
	switch node.(type) {
	case *mdast.Heading, *mdast.Text, *mdast.TextBlock,
		*mdast.FencedCodeBlock, *mdast.ListItem, *mdast.AutoLink,
		*mdast.RawHTML, *mdast.HTMLBlock:
		return true
	default:
		return false
	}
}

// ToRichText returns a NtRichTextBuilder for a given node
// RichTextConstructor then can be called with a given source to construct a ready-to-use notion RichText object
func ToRichText(node mdast.Node) *NtRichTextBuilder {
	switch v := node.(type) {
	case *mdast.Heading:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			return nt.NewTextRichText(string(contentFromLines(v, source)))
		})
	case *mdast.Text:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			return nt.NewTextRichText(string(v.Value(source)))
		})
	case *mdast.TextBlock:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			return nt.NewTextRichText(string(contentFromLines(v, source)))
		})
	case *mdast.FencedCodeBlock:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			return nt.NewTextRichText(string(contentFromLines(v, source)))
		})
	case *mdast.ListItem:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			return nt.NewTextRichText(string(contentFromLines(v, source)))
		})
	case *mdast.AutoLink:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			link := string(v.URL(source))
			label := string(v.Label(source))

			return nt.NewLinkRichText(label, link)
		})
	case *mdast.RawHTML:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			content := html2notion(
				string(contentFromSegments(v.Segments, source)),
			)
			return nt.NewTextRichText(content)
		})
	case *mdast.HTMLBlock:
		return NewNtRichTextBuilder(func(source []byte) *nt.RichText {
			content := html2notion(
				string(contentFromLines(v, source)),
			)
			return nt.NewTextRichText(content)
		})
	default:
		// Use IsConvertableToRichText to prevent such panics
		panic("ToRichText: unsupported markdown node type " + node.Kind().String())
	}
}

// flattenRichTexts flattens given MD as node into series of Notion RichTexts
// Should be only used when we know that we can't build a nested block structure with the given node
func flattenRichTexts(node mdast.Node) NtRichTextBuilders {
	// Final point: If no has no children, try to get its content via Lines, Segment, etc
	if node.ChildCount() == 0 {
		return NtRichTextBuilders{ToRichText(node)}
	}

	richTexts := make(NtRichTextBuilders, 0)

	// If has children: recursively iterate and flatten results
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {

		// Special handling:
		switch child.Kind() {
		case mdast.KindImage, mdastx.KindTaskCheckBox:
			panic(fmt.Sprintf("flattenedRichTexts is not possible for %s. nestedObjects must be use", child.Kind()))
		}

		deeperRichTexts := flattenRichTexts(child)
		DebugRichTexts(deeperRichTexts, fmt.Sprintf("   --> Flattening children of %s", child.Kind()))

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

		case *mdast.Text, *mdast.TextBlock, *mdast.RawHTML, *mdast.AutoLink:
		default:
			slog.Warn(fmt.Sprintf("Unhandled child's type: %s", v.Kind().String()))
		}

		richTexts = append(richTexts, deeperRichTexts...)
	}

	return richTexts
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
func flatten(node mdast.Node, levelArg ...int) (richTexts NtRichTextBuilders, blocks NtBlockBuilders) {
	var level int
	if len(levelArg) > 0 {
		level = levelArg[0]
	}

	// Final point: If no has no children, try to get its content via Lines, Segment, etc
	if node.ChildCount() == 0 {
		if IsConvertableToRichText(node) {
			richText := ToRichText(node)
			return NtRichTextBuilders{richText}, nil
		}

		return
	}

	richTexts = make(NtRichTextBuilders, 0)
	blocks = make(NtBlockBuilders, 0)

	// If has children: recursively iterate and flatten results
	iSibling := -1
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		iSibling++

		// Special handling:
		switch v := child.(type) {
		case *mdast.Image:
			/*
						// make a copy of the rich texts inside, as they will become Image Caption
				// but we nil-ify the original rich texts as to prevent them from duplicating
				captionRichTexts := append(NtRichTextBuilders{}, deeperRichTexts...)
				deeperRichTexts = nil
				blocks = append(blocks, NewNtBlockBuilder(func(source []byte) nt.Block {
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
				}))
			*/
		case *mdastx.TaskCheckBox:

			// [ x ] Some text
			//   ^---  ---------child
			//       ^---- next
			// So we need to create a block for checkbox with RichTexts taken from the rest of the siblings:
			labels := make(NtRichTextBuilders, 0)
			for next := child.NextSibling(); next != nil; next = next.NextSibling() {
				tabLog(level, fmt.Sprintf("flattening %d TASK CHECK sibling", iSibling))
				siblingTexts, _ := flatten(next, level+1)
				labels = append(labels, siblingTexts...)
			}
			block := NewNtBlockBuilder(func(source []byte) nt.Block {
				return &nt.ToDoBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeToDo,
					},
					ToDo: nt.ToDo{
						Checked:  v.IsChecked,
						RichText: labels.Build(source),
					},
				}
			})

			DebugBlock(block, "TODO Task")

			return nil, NtBlockBuilders{block}
		}

		// Rest are fine: simply flatten and maybe decorate

		// Flatten children of current child
		tabLog(level, fmt.Sprintf("flattening %d general sibling", iSibling))

		deeperRichTexts, deeperBlocks := flatten(child, level+1)

		DebugRichTexts(deeperRichTexts, fmt.Sprintf("Flattening children of %s", child.Kind()))

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

		case *mdast.Text, *mdast.TextBlock, *mdast.RawHTML, *mdast.AutoLink:
		// we're fine here doing nothing
		case *mdast.Image, *mdastx.TaskCheckBox:
			// something is really broken. First case should have handled this already
			panic("something is really broken")
		default:
			slog.Warn(fmt.Sprintf("Unhandled child's type: %s", v.Kind().String()))
		}

		blocks = append(blocks, deeperBlocks...)
		richTexts = append(richTexts, deeperRichTexts...)
	}

	return richTexts, blocks
}

// nolint: gocyclo // WILL be OK after refactor
func ToBlocks(node mdast.Node) NtBlockBuilders {
	// Pure flattening first:
	switch node.Kind() {
	case mdast.KindHeading:
		// Although in MD mdast.Heading is respresented via deeply nested tree of objects
		// In notion it should be a flattened list of RichTexts (With no children)
		// Edge case: Notion's heading.collapseable=true (that supports children) is not supported yet
		//            TODO(amberpixels): create an issue for it
		richTexts := flattenRichTexts(node)

		DebugRichTexts(richTexts, "Heading")

		switch node.(*mdast.Heading).Level { // nolint:errcheck
		case 1:
			return NtBlockBuilders{
				NewNtBlockBuilder(func(source []byte) nt.Block {
					return &nt.Heading1Block{BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeHeading1,
					}, Heading1: nt.Heading{RichText: richTexts.Build(source)}}
				}),
			}
		case 2:
			return NtBlockBuilders{
				NewNtBlockBuilder(func(source []byte) nt.Block {
					return &nt.Heading2Block{BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeHeading2,
					}, Heading2: nt.Heading{RichText: richTexts.Build(source)}}
				}),
			}
		default:
			return NtBlockBuilders{
				NewNtBlockBuilder(func(source []byte) nt.Block {
					return &nt.Heading3Block{BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeHeading3,
					}, Heading3: nt.Heading{RichText: richTexts.Build(source)}}
				}),
			}
		}
	case mdast.KindFencedCodeBlock:
		return NtBlockBuilders{
			NewNtBlockBuilder(func(source []byte) nt.Block {
				codeBlock := node.(*mdast.FencedCodeBlock) // nolint:errcheck
				return &nt.CodeBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeCode,
					},
					Code: nt.Code{
						Language: sanitizeBlockLanguage(string(codeBlock.Language(source))),
						RichText: flattenRichTexts(node).Build(source),
					},
				}
			}),
		}
	case mdast.KindHTMLBlock:
		return NtBlockBuilders{
			NewNtBlockBuilder(func(source []byte) nt.Block {
				return &nt.ParagraphBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeParagraph,
					},
					Paragraph: nt.Paragraph{
						RichText: flattenRichTexts(node).Build(source),
					},
				}
			}),
		}
	case mdast.KindThematicBreak:
		// Create a Notion Divider Block
		return NtBlockBuilders{
			NewNtBlockBuilder(func(_ []byte) nt.Block {
				return &nt.DividerBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeDivider,
					},
				}
			}),
		}
	case mdast.KindImage:
		// make a copy of the rich texts inside, as they will become Image Caption
		// but we nil-ify the original rich texts as to prevent them from duplicating
		//captionRichTexts := append(NtRichTextBuilders{}, deeperRichTexts...)
		//deeperRichTexts = nil
		// TODO caption??
		return NtBlockBuilders{
			NewNtBlockBuilder(func(source []byte) nt.Block {
				return &nt.ImageBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeImage,
					},
					Image: nt.Image{
						Type: nt.FileTypeExternal,
						External: &nt.FileObject{
							URL: string(node.(*mdast.Image).Destination),
						},
						//Caption: captionRichTexts.Build(source),
					},
				}
			}),
		}

	case /* ??? */ mdastx.KindTaskCheckBox:
		// [ x ] Some text
		//   ^---  ---------child
		//       ^---- next
		// So we need to create a block for checkbox with RichTexts taken from the rest of the siblings:
		labels := make(NtRichTextBuilders, 0)
		for next := node.NextSibling(); next != nil; next = next.NextSibling() {
			siblingTexts := flattenRichTexts(next)
			labels = append(labels, siblingTexts...)
		}
		block := NewNtBlockBuilder(func(source []byte) nt.Block {
			return &nt.ToDoBlock{
				BasicBlock: nt.BasicBlock{
					Object: nt.ObjectTypeBlock,
					Type:   nt.BlockTypeToDo,
				},
				ToDo: nt.ToDo{
					Checked:  node.(*mdastx.TaskCheckBox).IsChecked,
					RichText: labels.Build(source),
				},
			}
		})

		DebugBlock(block, "TODO Task")

		return NtBlockBuilders{block}
	case mdastx.KindTable: // Use the extension AST for the Table node
		table := node.(*mdastx.Table) // nolint:errcheck

		// Collect headers and rows
		headers := make([]NtRichTextBuilders, 0)
		rows := make([][]NtRichTextBuilders, 0)

		// Iterate over the table's children to extract headers and rows
		// TODO: move this deeper, as tables can be not first-level as well
		for tr := table.FirstChild(); tr != nil; tr = tr.NextSibling() {
			switch tr.Kind() {
			case mdastx.KindTableHeader:
				// Collect headers
				for th := tr.FirstChild(); th != nil; th = th.NextSibling() {
					richTexts := flattenRichTexts(th)
					headers = append(headers, richTexts)
				}

			case mdastx.KindTableRow:
				// Collect each row's cells
				row := make([]NtRichTextBuilders, 0)
				for td := tr.FirstChild(); td != nil; td = td.NextSibling() {
					richTexts := flattenRichTexts(td)
					row = append(row, richTexts)
				}
				rows = append(rows, row)
			}
		}

		// Create Notion table block
		return NtBlockBuilders{
			NewNtBlockBuilder(func(source []byte) nt.Block {
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
			}),
		}
	}

	if node.ChildCount() == 0 {
		panic("Empty node on top-level ToBlocks call")
	}

	// Nested blocks are required:
	switch node.Kind() {
	case mdast.KindParagraph:
		innerBlocks := make(NtBlockBuilders, 0)
		innerTexts := make(NtRichTextBuilders, 0)
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if IsConvertableToRichText(child) {
				innerTexts = append(innerTexts, flattenRichTexts(child)...)
			} else if IsConvertableToBlock(child) {
				innerBlocks = append(innerBlocks, ToBlocks(child)...)
			}
		}
		return NtBlockBuilders{
			NewNtBlockBuilder(func(source []byte) nt.Block {
				return &nt.ParagraphBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeParagraph,
					},
					Paragraph: nt.Paragraph{
						RichText: innerTexts.Build(source),
						Children: innerBlocks.Build(source),
					},
				}
			}),
		}
	case mdast.KindList:
		list := node.(*mdast.List)
		return handleList(list)
	case mdast.KindListItem:
		return handleListItem(node)
	case mdast.KindTextBlock:

	case mdast.KindBlockquote:
		richTexts, blocks := flatten(node)

		return NtBlockBuilders{
			NewNtBlockBuilder(func(source []byte) nt.Block {
				return &nt.QuoteBlock{
					BasicBlock: nt.BasicBlock{
						Object: nt.ObjectTypeBlock,
						Type:   nt.BlockTypeQuote,
					},
					Quote: nt.Quote{
						RichText: richTexts.Build(source),
						Children: blocks.Build(source),
					},
				}
			}),
		}
	}

	panic(fmt.Sprintf("unhandled node type: %s", node.Kind().String()))
}

// handleList processes a markdown list and returns appropriate Notion blocks
func handleList(list *mdast.List) NtBlockBuilders {
	blocks := make(NtBlockBuilders, 0)

	// Group task items together
	var taskItems NtBlockBuilders
	var currentListItem *NtBlockBuilder

	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		// Each child is a ListItem
		if list.Marker == '-' || list.Marker == '+' || list.Marker == '*' {
			// For bulleted lists, check if this item contains task items
			hasTaskItems := false
			for textBlock := child.FirstChild(); textBlock != nil; textBlock = textBlock.NextSibling() {
				if textBlock.Kind() == mdast.KindTextBlock {
					for grandChild := textBlock.FirstChild(); grandChild != nil; grandChild = grandChild.NextSibling() {
						if grandChild.Kind() == mdastx.KindTaskCheckBox {
							hasTaskItems = true
							taskItems = append(taskItems, createTodoItem(grandChild))
						}
					}
				}
			}

			if hasTaskItems {
				// If this is a new list item with task items, add the previous one if it exists
				if currentListItem != nil {
					blocks = append(blocks, currentListItem)
				}
				// Create new list item with the collected task items as children
				currentListItem = createBulletedListItemWithChildren(child, taskItems)
				taskItems = make(NtBlockBuilders, 0) // Reset for next group
			} else {
				// Regular bulleted list item
				if currentListItem != nil {
					blocks = append(blocks, currentListItem)
					currentListItem = nil
				}
				blocks = append(blocks, createBulletedListItem(child))
			}
		} else {
			// Numbered list items (same logic as above)
			blocks = append(blocks, createNumberedListItem(child))
		}
	}

	// Don't forget to add the last list item if it exists
	if currentListItem != nil {
		blocks = append(blocks, currentListItem)
	}

	return blocks
}

// Helper function to create a list item with predefined children
func createBulletedListItemWithChildren(node mdast.Node, children NtBlockBuilders) *NtBlockBuilder {
	mainContent := extractMainContent(node)

	return NewNtBlockBuilder(func(source []byte) nt.Block {
		return &nt.BulletedListItemBlock{
			BasicBlock: nt.BasicBlock{
				Object: nt.ObjectTypeBlock,
				Type:   nt.BlockTypeBulletedListItem,
			},
			BulletedListItem: nt.ListItem{
				RichText: mainContent.Build(source),
				Children: children.Build(source),
			},
		}
	})
}

// handleListItem processes a single list item and its children
func handleListItem(node mdast.Node) NtBlockBuilders {
	// First, get the text content of this item (before any nested lists)
	mainContent := extractMainContent(node)

	// Then, handle any nested content (including nested lists)
	var children NtBlockBuilders
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case mdast.KindList:
			// Recursive handling of nested lists
			children = append(children, handleList(child.(*mdast.List))...)
		case mdastx.KindTaskCheckBox:
			// Handle checkboxes
			children = append(children, createTodoItem(child))
			// ... handle other types of nested content
		}
	}

	// Create the appropriate block type
	return NtBlockBuilders{
		NewNtBlockBuilder(func(source []byte) nt.Block {
			return &nt.BulletedListItemBlock{
				BasicBlock: nt.BasicBlock{
					Object: nt.ObjectTypeBlock,
					Type:   nt.BlockTypeBulletedListItem,
				},
				BulletedListItem: nt.ListItem{
					RichText: mainContent.Build(source),
					Children: children.Build(source),
				},
			}
		}),
	}
}

// createBulletedListItem creates a bulleted list item from a markdown list item node
func createBulletedListItem(node mdast.Node) *NtBlockBuilder {
	// First, get the text content of this item (before any nested lists)
	mainContent := extractMainContent(node)

	// Then, handle any nested content (including nested lists)
	var children NtBlockBuilders
	var taskItems NtBlockBuilders // Collect all task items together

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case mdast.KindList:
			// Recursive handling of nested lists
			children = append(children, handleList(child.(*mdast.List))...)
		case mdast.KindTextBlock:
			// Collect all task items from this text block
			for grandChild := child.FirstChild(); grandChild != nil; grandChild = grandChild.NextSibling() {
				if grandChild.Kind() == mdastx.KindTaskCheckBox {
					taskItems = append(taskItems, createTodoItem(grandChild))
				}
			}
		}
	}

	// If we found any task items, add them all at once as children
	if len(taskItems) > 0 {
		children = append(children, taskItems...)
	}

	return NewNtBlockBuilder(func(source []byte) nt.Block {
		return &nt.BulletedListItemBlock{
			BasicBlock: nt.BasicBlock{
				Object: nt.ObjectTypeBlock,
				Type:   nt.BlockTypeBulletedListItem,
			},
			BulletedListItem: nt.ListItem{
				RichText: mainContent.Build(source),
				Children: children.Build(source),
			},
		}
	})
}

// createNumberedListItem creates a numbered list item from a markdown list item node
func createNumberedListItem(node mdast.Node) *NtBlockBuilder {
	// First, get the text content of this item (before any nested lists)
	mainContent := extractMainContent(node)

	// Then, handle any nested content (including nested lists)
	var children NtBlockBuilders
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case mdast.KindList:
			// Recursive handling of nested lists
			children = append(children, handleList(child.(*mdast.List))...)
		case mdast.KindTextBlock:
			// Check for task items inside the text block
			for grandChild := child.FirstChild(); grandChild != nil; grandChild = grandChild.NextSibling() {
				if grandChild.Kind() == mdastx.KindTaskCheckBox {
					children = append(children, createTodoItem(grandChild))
				}
			}
		}
	}

	return NewNtBlockBuilder(func(source []byte) nt.Block {
		return &nt.NumberedListItemBlock{
			BasicBlock: nt.BasicBlock{
				Object: nt.ObjectTypeBlock,
				Type:   nt.BlockTypeNumberedListItem,
			},
			NumberedListItem: nt.ListItem{
				RichText: mainContent.Build(source),
				Children: children.Build(source),
			},
		}
	})
}

// createTodoItem creates a todo item from a markdown task checkbox node
func createTodoItem(node mdast.Node) *NtBlockBuilder {
	checkbox := node.(*mdastx.TaskCheckBox)

	// Get the text content that follows the checkbox
	labels := make(NtRichTextBuilders, 0)
	for next := node.NextSibling(); next != nil; next = next.NextSibling() {
		if IsConvertableToRichText(next) {
			labels = append(labels, flattenRichTexts(next)...)
		}
	}

	return NewNtBlockBuilder(func(source []byte) nt.Block {
		return &nt.ToDoBlock{
			BasicBlock: nt.BasicBlock{
				Object: nt.ObjectTypeBlock,
				Type:   nt.BlockTypeToDo,
			},
			ToDo: nt.ToDo{
				Checked:  checkbox.IsChecked,
				RichText: labels.Build(source),
			},
		}
	})
}

// extractMainContent extracts text content while being careful about nested blocks
func extractMainContent(node mdast.Node) NtRichTextBuilders {
	texts := make(NtRichTextBuilders, 0)

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		// Skip lists as they should be handled as children
		if child.Kind() == mdast.KindList {
			continue
		}

		// For text blocks, we need to check their children for task items
		if child.Kind() == mdast.KindTextBlock {
			hasTaskItems := false
			// Check if this text block contains any task items
			for grandChild := child.FirstChild(); grandChild != nil; grandChild = grandChild.NextSibling() {
				if grandChild.Kind() == mdastx.KindTaskCheckBox {
					hasTaskItems = true
					break
				}
			}
			// Skip this text block if it contains task items
			if hasTaskItems {
				continue
			}
		}

		// If we get here, it's safe to convert to rich text
		if IsConvertableToRichText(child) {
			texts = append(texts, flattenRichTexts(child)...)
		}
	}

	return texts
}

func tabLog(level int, message string) {
	var tabs = ""
	for i := 0; i < level; i++ {
		tabs += " "
	}

	slog.Debug(tabs + message)
}
