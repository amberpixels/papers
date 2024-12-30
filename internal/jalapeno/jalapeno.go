// Package jalapeno is a library that provides Markdown -> Notion conversion
package jalapeno

import (
	"fmt"
	"regexp"
	"strings"

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

// ParseBlocks parses the given markdown source into Notion Blocks
func (p *Parser) ParseBlocks(source []byte) (nt.Blocks, error) {
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
		return nil, fmt.Errorf("failed to walk parsed Markdown AST: %w", err)
	}

	return blockBuilders.Build(source), nil
}

func PrepareNotionPageProperties(blocks nt.Blocks) (nt.Blocks, nt.Properties) {
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

	return blocks, nt.Properties{
		string(nt.PropertyConfigTypeTitle): nt.TitleProperty{
			Title: pageTitle,
		},
	}
}

// IsConvertableToRichText returns true if given Markdown AST node is convertable directly into notion RichText
func IsConvertableToRichText(node mdast.Node) bool {
	switch node.(type) {
	case *mdast.Text,
		*mdast.CodeBlock, *mdast.FencedCodeBlock,
		*mdast.ListItem, *mdast.AutoLink,
		*mdast.RawHTML, *mdast.HTMLBlock, *mdast.Paragraph,
		*mdast.Emphasis, *mdastx.Strikethrough, *mdast.CodeSpan:
		return true
	case *mdast.Link:
		// TODO: not yet working in full manner
		if node.FirstChild() != nil && node.FirstChild().Kind() == mdast.KindImage {
			return false
		}
		return true
	case *mdast.TextBlock:
		if node.FirstChild() != nil && node.FirstChild().Kind() == mdastx.KindTaskCheckBox {
			// NOCOV??
			return false
		}
		return true
	default:
		return false
	}
}

// ExtractRichTexts extract all richtexts for a given node
// It does work ONLY for nodes that can be handled purely via Notion's RichTexts
// Use HandledViaRichTexts to check it.
func ExtractRichTexts(node mdast.Node) NtRichTextBuilders {
	if node.ChildCount() == 0 {
		return NtRichTextBuilders{ToRichText(node)}
	}

	richTexts := make(NtRichTextBuilders, 0)
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		richTexts = append(richTexts, decorateRichTexts(
			node,
			ExtractRichTexts(child),
		)...)
	}
	return richTexts
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
	case *mdast.FencedCodeBlock, *mdast.CodeBlock:
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
		return nil
	}
}

// ToBlocks converts given MD ast node into series of Notion Blocks
// nolint: gocyclo // Will be OK after further refactor
func ToBlocks(node mdast.Node) NtBlockBuilders {
	// Thoughts: First switch is used when ToBlocks was called from children handling (recursion)
	// can we optimize it somehow?

	// Pure flattening first:
	switch node.Kind() {
	case mdast.KindHeading:
		return handleHeading(node)
	case mdast.KindCodeBlock, mdast.KindFencedCodeBlock:
		return NtBlockBuilders{
			NewNtBlockBuilder(func(source []byte) nt.Block {
				var language string
				if codeBlock, ok := node.(*mdast.FencedCodeBlock); ok {
					language = sanitizeBlockLanguage(string(codeBlock.Language(source)))
				}
				richTexts := ExtractRichTexts(node)

				return nt.NewCodeBlock(nt.Code{
					RichText: richTexts.Build(source),
					Language: language,
				})
			}),
		}
	case mdast.KindThematicBreak:
		return NtBlockBuilders{
			NewNtBlockBuilder(func(_ []byte) nt.Block {
				return nt.NewDividerBlock()
			}),
		}
	case mdast.KindImage:
		captionRichTexts := NtRichTextBuilders{}
		if child := node.FirstChild(); child != nil {
			captionRichTexts = ExtractRichTexts(child)
		}

		return NtBlockBuilders{
			NewNtBlockBuilder(func(source []byte) nt.Block {
				return nt.NewImageBlock(nt.Image{
					Type: nt.FileTypeExternal,
					External: &nt.FileObject{
						URL: string(node.(*mdast.Image).Destination), // nolint:errcheck
					},
					Caption: captionRichTexts.Build(source),
				})
			}),
		}
	case mdastx.KindTable: // Use the extension AST for the Table node
		return handleTable(node)
	case mdast.KindHTMLBlock:
		return handleHTMLBlock(node)
	}

	if node.ChildCount() == 0 {
		panic("Empty node on top-level ToBlocks call")
	}

	// Nested blocks are required:
	switch node.Kind() {
	case mdast.KindParagraph:
		// similar to BlockQuote - should be handled in a shared way
		innerTexts := make(NtRichTextBuilders, 0)
		innerBlocks := make(NtBlockBuilders, 0)
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			// if it's convertable to rich text and we didn't handle any blocks yet, we're OK to flatten
			// as soon as we met an inner block, all further children are considered as blocks as well
			if IsConvertableToRichText(child) && len(innerBlocks) == 0 {
				innerTexts = append(innerTexts, ExtractRichTexts(child)...)
			} else {
				innerBlocks = append(innerBlocks, ToBlocks(child)...)
			}
		}
		return NtBlockBuilders{
			NewNtBlockBuilder(func(source []byte) nt.Block {
				return nt.NewParagraphBlock(nt.Paragraph{
					RichText: innerTexts.Build(source),
					Children: innerBlocks.Build(source),
				})
			}),
		}
	case mdast.KindBlockquote:
		return handleBlockquote(node)
	case mdast.KindList:
		return handleList(node)
	case mdast.KindTextBlock:
		richTexts := ExtractRichTexts(node)
		return NtBlockBuilders{
			NewNtBlockBuilder(func(source []byte) nt.Block {
				return nt.NewQuoteBlock(nt.Quote{
					RichText: richTexts.Build(source),
				})
			}),
		}
	}

	panic(fmt.Sprintf("unhandled node type: %s", node.Kind().String()))
}

// handleHeading handles custom logic of Markdown->Notion Headings
// Although in MD mdast.Heading can have children,
// In notion it's a flattened list of RichTexts
// Edge case: Notion's heading.collapseable=true (that supports children) is not supported yet
// TODO(amberpixels): support collapsable headings with children
func handleHeading(node mdast.Node) NtBlockBuilders {
	heading := node.(*mdast.Heading) // nolint:errcheck
	headingLevel := heading.Level
	richTexts := ExtractRichTexts(node)

	return NtBlockBuilders{NewNtBlockBuilder(func(source []byte) nt.Block {
		return nt.NewHeadingBlock(
			nt.Heading{RichText: richTexts.Build(source)},
			headingLevel,
		)
	})}
}

// handleTable handles custom logic of Markdown->Notion tables
// Nothing special here, just custom defining of rows, headers, and cells
func handleTable(node mdast.Node) NtBlockBuilders {
	table := node.(*mdastx.Table) // nolint:errcheck

	// Collect headers and rows
	headers := make([]NtRichTextBuilders, 0)
	rows := make([][]NtRichTextBuilders, 0)

	// TODO: support recursive tables?

	// Iterate over the table's children to extract headers and rows
	for tr := table.FirstChild(); tr != nil; tr = tr.NextSibling() {
		switch tr.Kind() {
		case mdastx.KindTableHeader:
			// Collect headers
			for th := tr.FirstChild(); th != nil; th = th.NextSibling() {
				// TODO: is it possible in the Header to have nested blocks?
				headers = append(headers, ExtractRichTexts(th))
			}

		case mdastx.KindTableRow:
			// Collect each row's cells
			row := make([]NtRichTextBuilders, 0)
			for td := tr.FirstChild(); td != nil; td = td.NextSibling() {
				// TODO: we need to handle any nested blocks inside tables as well
				row = append(row, ExtractRichTexts(td))
			}
			rows = append(rows, row)
		}
	}

	// Create Notion table block
	return NtBlockBuilders{
		NewNtBlockBuilder(func(source []byte) nt.Block {
			// Construct table block
			tableBlock := nt.NewTableBlock(nt.Table{
				TableWidth:      len(headers),
				HasColumnHeader: true,
				Children:        nt.Blocks{}, // will be populated below

				//HasRowHeader:  false, // TODO(amberpixels) is this possible to be known from markdown?
			})

			// Populate header row
			if len(headers) > 0 {
				headerRow := nt.TableRow{
					Cells: make([][]nt.RichText, len(headers)),
				}
				for i, header := range headers {
					headerRow.Cells[i] = header.Build(source)
				}

				tableBlock.Table.Children = append(tableBlock.Table.Children, nt.NewTableRowBlock(headerRow))
			}

			// Populate the rest of the rows
			for _, row := range rows {
				tableRow := nt.TableRow{
					Cells: make([][]nt.RichText, len(row)),
				}
				for i, cell := range row {
					tableRow.Cells[i] = cell.Build(source)
				}
				tableBlock.Table.Children = append(tableBlock.Table.Children, nt.NewTableRowBlock(tableRow))
			}

			return tableBlock
		}),
	}
}

// handleHTMLBlock handles custom logic of Markdown->Notion HTML blocks
// Notion doesn't support HTML in rich-text so we have to convert it manually into Notion blocks
// For now we just keep RAW html (no parsing), but it should be fixed
// TODO: support HTML, at least paragraph, better lists + tables?
func handleHTMLBlock(node mdast.Node) NtBlockBuilders {
	richTexts := ExtractRichTexts(node)
	// TODO find out why letter case is not preserved

	return NtBlockBuilders{
		NewNtBlockBuilder(func(source []byte) nt.Block {
			// Weak solution but fine for now
			saneContent := make([]nt.RichText, 0)
			for _, rt := range richTexts.Build(source) {
				cleaned := sanitizeMarkdownLintComments(rt.PlainText)
				if cleaned == "" {
					continue
				}
				rt.PlainText = cleaned
				rt.Text.Content = cleaned
				saneContent = append(saneContent, rt)
			}

			return nt.NewParagraphBlock(nt.Paragraph{
				RichText: saneContent,
			})
		}),
	}
}

// handleBlockquote handles custom logic of Markdown->Notion Blockquotes
// Notion's Blockquote is a container that has both mandatory rich-text content and children
// Mandatory rich-text makes an issue if in Markdown you had a blockquote with a heading as a first child
// (As heading is a block, can't be fully represented in rich-text)
func handleBlockquote(node mdast.Node) NtBlockBuilders {
	// TODO: handle blockquotes better
	innerTexts := make(NtRichTextBuilders, 0)
	innerBlocks := make(NtBlockBuilders, 0)
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		// if it's convertable to rich text and we didn't handle any blocks yet, we're OK to flatten
		// as soon as we met an inner block, all further children are considered as blocks as well
		if IsConvertableToRichText(child) && len(innerBlocks) == 0 {
			innerTexts = append(innerTexts, ExtractRichTexts(child)...)
		} else {
			innerBlocks = append(innerBlocks, ToBlocks(child)...)
		}
	}

	return NtBlockBuilders{
		NewNtBlockBuilder(func(source []byte) nt.Block {
			return nt.NewQuoteBlock(nt.Quote{
				RichText: innerTexts.Build(source),
				Children: innerBlocks.Build(source),
			})
		}),
	}
}

// handleList processes a markdown list and returns appropriate Notion blocks
func handleList(node mdast.Node) NtBlockBuilders {
	list := node.(*mdast.List) // nolint:errcheck

	// Check if list is bulleted or numbered
	var bulletted bool
	if list.Marker == '-' || list.Marker == '+' || list.Marker == '*' {
		bulletted = true
	}

	blocks := make(NtBlockBuilders, 0)
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		blocks = append(blocks, handleListItem(child, bulletted))
	}

	return blocks
}

// handleListItem handles MD's list item and its children
// List Item on markdown can have children. For notion - first child is usually a RichText
// Other children are built as nested blocks
// Exception is TaskItem. On Notion it's not a ListItem at all. It's just a ToDoBlock
func handleListItem(node mdast.Node, bulletted bool) *NtBlockBuilder {
	// Extract RichText (from first child)
	mainContent := make(NtRichTextBuilders, 0)
	if child := node.FirstChild(); child != nil {
		// If we get here, it's safe to convert to rich text
		if IsConvertableToRichText(child) {
			mainContent = ExtractRichTexts(child)
		}
	}

	var children NtBlockBuilders
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if len(mainContent) > 0 && child.PreviousSibling() == nil { // skip main content
			continue
		}

		switch child.Kind() {
		case mdast.KindTextBlock: // TASK items are hidden inside text blocks
			for grandChild := child.FirstChild(); grandChild != nil; {
				if grandChild.Kind() == mdastx.KindTaskCheckBox {
					return handleTaskItem(child)
				}
				break
			}

		default:
			children = append(children, ToBlocks(child)...)
		}
	}

	return NewNtBlockBuilder(func(source []byte) nt.Block {
		li := nt.ListItem{
			RichText: mainContent.Build(source),
			Children: children.Build(source),
		}

		if bulletted {
			return nt.NewBulletedListItemBlock(li)
		} else {
			return nt.NewNumberedListItemBlock(li)
		}
	})
}

// handleTaskItem handles given node to ensure it's a markdown task item
// For this it should have first child as a checkbox and then its content
func handleTaskItem(node mdast.Node) *NtBlockBuilder {
	if node == nil || node.FirstChild() == nil {
		return nil
	}
	checkbox, ok := node.FirstChild().(*mdastx.TaskCheckBox)
	if !ok {
		return nil
	}

	// Get the text content that follows the checkbox
	labels := make(NtRichTextBuilders, 0)
	for next := checkbox.NextSibling(); next != nil; next = next.NextSibling() {
		if IsConvertableToRichText(next) {
			labels = append(labels, ExtractRichTexts(next)...)
		}
	}

	return NewNtBlockBuilder(func(source []byte) nt.Block {
		return nt.NewToDoBlock(nt.ToDo{
			Checked:  checkbox.IsChecked,
			RichText: labels.Build(source),
		})
	})
}

func decorateRichTexts(parent mdast.Node, richTexts NtRichTextBuilders) NtRichTextBuilders {
	// TODO" make immutable function
	switch v := parent.(type) {
	case *mdastx.Strikethrough:
		for i := range richTexts {
			richTexts[i].DecorateWith(strikethroughDecorator)
		}
	case *mdast.Emphasis:
		for i := range richTexts {
			if v.Level == 1 {
				richTexts[i].DecorateWith(italicDecorator)
			} else {
				richTexts[i].DecorateWith(boldDecorator)
			}
		}
	case *mdast.CodeSpan:
		// Adding t.Annotations = code:true for each child
		for i := range richTexts {
			richTexts[i].DecorateWith(codeDecorator)
		}

	case *mdast.Link:
		for i := range richTexts {
			richTexts[i].DecorateWith(linkDecorator(string(v.Destination)))
		}
	}

	return richTexts
}

var markdownLintRegex = regexp.MustCompile(`(?i)<!--\s*markdownlint-.*?-->`)

// sanitizeMarkdownLintComments checks if the content is a markdownlint-disable comment
func sanitizeMarkdownLintComments(content string) string {
	return strings.TrimSpace(markdownLintRegex.ReplaceAllString(content, ""))
}
