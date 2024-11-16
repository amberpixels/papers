package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/alecthomas/kong"
	"github.com/amberpixels/peppers/internal/jalapeno"
	"github.com/joho/godotenv"
	"github.com/jomei/notionapi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

var in struct {
	NotionAPIToken string `help:"Notion API token." env:"NOTION_API_TOKEN"`
	NotionParentID string `help:"Parent page ID in Notion." env:"NOTION_PARENT_PAGE_ID"`
	FileName       string `help:"Path to the local README.md file." env:"FILE_NAME"`

	DevMode bool `help:"Dev mode (verbose logging, etc)" env:"DEV_MODE"`
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load(".env")
	if os.IsNotExist(err) {
		// having `.env` is optional, so we're OK here
	} else if err != nil {
		slog.Warn("failed to read .env: " + err.Error())
	}

	// for now we do not need result of Kong. It will be needed later, when we have commands
	_ = kong.Parse(&in)

	// Create a context that is canceled when an interrupt or termination signal is received
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	if in.DevMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	source, err := os.ReadFile(in.FileName)
	if err != nil {
		ExitWithError("Couldn't read the source file", err)
	}

	// Display the parsed parameters
	fmt.Printf("Converting Markdown File [%s] into Notion [%s]\n", in.FileName, in.NotionParentID)

	p := jalapeno.NewParser(goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	))

	blocks, props, err := p.ParsePage(source)
	if err != nil {
		ExitWithError("Couldn't parse the given file", err)
	}

	slog.Debug("Using Notion API with the given token: " + in.NotionAPIToken)

	pageReq := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(in.NotionParentID),
		},
		Properties: props,
		Children:   blocks,
	}

	client := notionapi.NewClient(notionapi.Token(in.NotionAPIToken))

	notionPageResult, err := client.Page.Create(ctx, pageReq)
	if err != nil {
		ExitWithError("failed to create the Notion page", err)
	}

	fmt.Printf("Successfully created Notion page: %s\n", notionPageResult.URL)
}

// ExitWithError outputs an error message and exits the program with a non-zero status code.
func ExitWithError(msg string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", msg, err)
	os.Exit(1)
}
