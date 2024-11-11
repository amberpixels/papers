package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/amberpixels/papers/internal/md2nt"
	"github.com/joho/godotenv"
	"github.com/jomei/notionapi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// Scaffold for input args
var (
	ParamFileName           string
	ParamNotionAPIToken     string
	ParamNotionParentPageID string
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	if err := godotenv.Load(".env"); err != nil {
		panic(err)
	}

	ParamFileName = os.Getenv("FILE_NAME")
	ParamNotionAPIToken = os.Getenv("NOTION_API_TOKEN")
	ParamNotionParentPageID = os.Getenv("NOTION_PARENT_PAGE_ID")

	slog.Info("Params: ",
		"filename", ParamFileName,
		"token", ParamNotionAPIToken,
		"page_id", ParamNotionParentPageID)

	notionTest()
}

func notionTest() {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	p := md2nt.NewParser(md)
	if err := p.ParseFile(ParamFileName); err != nil {
		panic(err)
	}

	notionBlocks := make([]notionapi.Block, 0)

	err := p.Walk(func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node.Kind() == ast.KindDocument {
			return ast.WalkContinue, nil
		}

		blocks := p.ToNotionBlocks(node)

		notionBlocks = append(notionBlocks, blocks...)
		return ast.WalkSkipChildren, nil
	})

	if err != nil {
		fmt.Println("Walrking error = ", err)
		panic("failed to walk AST")
	}

	var docTitle []notionapi.RichText
	if len(notionBlocks) > 0 {
		for i, block := range notionBlocks {
			if block.GetType() == notionapi.BlockTypeHeading1 {
				docTitle = block.(*notionapi.Heading1Block).Heading1.RichText
				// delete the i block
				notionBlocks = append(notionBlocks[:i], notionBlocks[i+1:]...)
				break

			}
		}
		// TODO(amberpixels): handle headings equality spread (H1-H6 of markdown) spread into H1-H3 of notion
	}
	if len(docTitle) == 0 {
		docTitle = []notionapi.RichText{{Text: &notionapi.Text{Content: fmt.Sprintf("Unnamed Document (Generated from %s)", ParamFileName)}}}
	}

	properties := notionapi.Properties{
		string(notionapi.PropertyConfigTypeTitle): notionapi.TitleProperty{
			Title: docTitle,
		},
	}

	// Create a new client
	client := notionapi.NewClient(notionapi.Token(ParamNotionAPIToken))

	// Define the new page request with template
	newPage := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(ParamNotionParentPageID),
		},
		Properties: properties,
		Children:   notionBlocks,
	}

	// Create the new page
	p1, err := client.Page.Create(context.Background(), newPage)
	if err != nil {
		log.Fatalf("Failed to create Notion page: %v", err)
	}
	fmt.Println("OK: ", p1.URL)
	fmt.Println("Notion page created successfully!")
}
