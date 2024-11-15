package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/amberpixels/peppers/internal/jalapeno"
	"github.com/joho/godotenv"
	"github.com/jomei/notionapi"
	"github.com/yuin/goldmark"
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

	source, err := os.ReadFile(ParamFileName)
	if err != nil {
		panic(err)
	}

	p := jalapeno.NewParser(goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	))

	blocks, props, err := p.ParsePage(source)
	if err != nil {
		panic(err)
	}

	pageReq := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(ParamNotionParentPageID),
		},
		Properties: props,
		Children:   blocks,
	}

	client := notionapi.NewClient(notionapi.Token(ParamNotionAPIToken))

	p1, err := client.Page.Create(context.Background(), pageReq)
	if err != nil {
		log.Fatalf("Failed to create Notion page: %v", err)
	}
	fmt.Println("OK: ", p1.URL)
	fmt.Println("Notion page created successfully!")
}
