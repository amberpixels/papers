package jalapeno_test

import (
	"fmt"
	"testing"

	"github.com/amberpixels/peppers/internal/jalapeno"
	nt "github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// Global parser instance for all tests
var parserInstance = jalapeno.NewParser(goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Table,
		extension.TaskList,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
))

func TestParser_ParseBlocks(t *testing.T) {
	// Define test cases
	tests := []struct {
		name           string
		source         []byte
		expectedBlocks nt.Blocks
	}{
		{
			name:   "Single Heading 1",
			source: []byte(`# Heading 1`),
			expectedBlocks: nt.Blocks{
				nt.NewHeading1Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Heading"),
						*nt.NewTextRichText(" 1"),
					},
				}),
			},
		},
		{
			name: "Headings H1 to H4",
			source: []byte(`# Heading 1
## Heading 2
### Heading 3
#### Heading 4`),
			expectedBlocks: nt.Blocks{
				nt.NewHeading1Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Heading"),
						*nt.NewTextRichText(" 1"),
					},
				}),
				nt.NewHeading2Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Heading"),
						*nt.NewTextRichText(" 2"),
					},
				}),
				nt.NewHeading3Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Heading"),
						*nt.NewTextRichText(" 3"),
					},
				}),
				nt.NewHeading3Block(nt.Heading{ // H4 is converted to Heading3
					RichText: []nt.RichText{
						*nt.NewTextRichText("Heading"),
						*nt.NewTextRichText(" 4"),
					},
				}),
			},
		},
		{
			name:   "Paragraph with emphasis",
			source: []byte(`Hello **foobar**`),
			expectedBlocks: nt.Blocks{
				nt.NewParagraphBlock(nt.Paragraph{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Hello "),
						*nt.NewTextRichText("foobar").AnnotateBold(),
					},
					Children: nt.Blocks{},
				}),
			},
		},
		{
			name:   "Paragraph with emphasis italic",
			source: []byte(`Hello *foobar*`),
			expectedBlocks: nt.Blocks{
				nt.NewParagraphBlock(nt.Paragraph{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Hello "),
						*nt.NewTextRichText("foobar").AnnotateItalic(),
					},
					Children: nt.Blocks{},
				}),
			},
		},
		{
			name:   "Paragraph with emphasis strikethrough",
			source: []byte(`Hello ~~no foobar~~`),
			expectedBlocks: nt.Blocks{
				nt.NewParagraphBlock(nt.Paragraph{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Hello "),
						*nt.NewTextRichText("no foobar").AnnotateStrikethrough(),
					},
					Children: nt.Blocks{},
				}),
			},
		},
		{
			name:   "Paragraph with different emphasis",
			source: []byte(`This is a **bold text** 1, *italic text* 2, and ~~strikethrough text~~ 3.`),
			expectedBlocks: nt.Blocks{
				nt.NewParagraphBlock(nt.Paragraph{
					RichText: []nt.RichText{
						*nt.NewTextRichText("This is a "),
						*nt.NewTextRichText("bold text").AnnotateBold(),
						*nt.NewTextRichText(" 1, "),
						*nt.NewTextRichText("italic text").AnnotateItalic(),
						*nt.NewTextRichText(" 2, and "),
						*nt.NewTextRichText("strikethrough text").AnnotateStrikethrough(),
						*nt.NewTextRichText(" 3."),
					},
					Children: nt.Blocks{},
				}),
			},
		},
		{
			name:   "Paragraph with one link",
			source: []byte(`Visit [OpenAI](https://openai.com)`),
			expectedBlocks: nt.Blocks{
				nt.NewParagraphBlock(nt.Paragraph{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Visit "),
						*nt.NewLinkRichText("OpenAI", "https://openai.com"),
					},
					Children: nt.Blocks{},
				}),
			},
		},
		{
			name:   "Inline link in a sentence",
			source: []byte(`Hello https://openai.com`),
			expectedBlocks: nt.Blocks{
				nt.NewParagraphBlock(nt.Paragraph{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Hello "),
						*nt.NewLinkRichText("https://openai.com", "https://openai.com"),
					},
					Children: nt.Blocks{},
				}),
			},
		},
		{
			name:   "Heading with non-inline link",
			source: []byte(`# Hello [OpenAI](https://openai.com)`),
			expectedBlocks: nt.Blocks{
				nt.NewHeading1Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Hello "),
						*nt.NewLinkRichText("OpenAI", "https://openai.com"),
					},
				}),
			},
		},
		{
			name:   "Heading with inline link",
			source: []byte(`# Hello https://openai.com`),
			expectedBlocks: nt.Blocks{
				nt.NewHeading1Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Hello "),
						*nt.NewLinkRichText("https://openai.com", "https://openai.com"),
					},
				}),
			},
		},
		{
			name:   "Heading2 with an inline link in brackets",
			source: []byte(`## Hello (https://openai.com)`),
			expectedBlocks: nt.Blocks{
				nt.NewHeading2Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Hello ("),
						*nt.NewLinkRichText("https://openai.com", "https://openai.com"),
						*nt.NewTextRichText(")"),
					},
				}),
			},
		},
		{
			name:   "Heading with empahsis + link",
			source: []byte(`# **ULID** Wrapper for *PostgreSQL* and *GORM* [link inside](https://github.com/oklog/ulid)`),
			expectedBlocks: nt.Blocks{
				nt.NewHeading1Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("ULID").AnnotateBold(),
						*nt.NewTextRichText(" Wrapper for "),
						*nt.NewTextRichText("PostgreSQL").AnnotateItalic(),
						*nt.NewTextRichText(" and "),
						*nt.NewTextRichText("GORM").AnnotateItalic(),
						*nt.NewTextRichText(" "),
						*nt.NewLinkRichText("link inside", "https://github.com/oklog/ulid"),
					},
				}),
			},
		},
		{
			name:   "Paragraph with inline code",
			source: []byte("This is `inline code`"),
			expectedBlocks: nt.Blocks{
				nt.NewParagraphBlock(nt.Paragraph{
					RichText: []nt.RichText{
						*nt.NewTextRichText("This is "),
						*nt.NewTextRichText("inline code").AnnotateCode(),
					},
					Children: nt.Blocks{},
				}),
			},
		},
		{
			name:   "Heading with inline code",
			source: []byte("# This is `inline code`"),
			expectedBlocks: nt.Blocks{
				nt.NewHeading1Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("This is "),
						*nt.NewTextRichText("inline code").AnnotateCode(),
					},
				}),
			},
		},
		{
			name: "Headings + Paragraph with code inline",
			source: []byte(`# Your Readme Package name

## Overview
` + "The `packageName` package provides a set of function for doing something useful.",
			),
			expectedBlocks: nt.Blocks{
				nt.NewHeading1Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Your Readme Package"),
						*nt.NewTextRichText(" name"),
					},
				}),
				nt.NewHeading2Block(nt.Heading{
					RichText: []nt.RichText{
						*nt.NewTextRichText("Overview"),
					},
				}),
				nt.NewParagraphBlock(nt.Paragraph{
					RichText: []nt.RichText{
						*nt.NewTextRichText("The "),
						*nt.NewTextRichText("packageName").AnnotateCode(),
						*nt.NewTextRichText(" package provides a set of function for doing something"),
						*nt.NewTextRichText(" useful."),
					},
					Children: nt.Blocks{},
				}),
			},
		},
	}

	// Run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			blocks, err := parserInstance.ParseBlocks(tc.source)

			require.NoError(t, err, "Parsing failed")
			assert.Len(t, blocks, len(tc.expectedBlocks), "Generated blocks do not match expected blocks")
			for i, b := range blocks {
				assert.Equal(t, tc.expectedBlocks[i].GetType(), b.GetType(), fmt.Sprintf("Generated block[%d] do not match expected block", i))
				assert.Equal(t, tc.expectedBlocks[i], b, fmt.Sprintf("Generated block[%d] do not match expected block", i))
			}
		})
	}
}
