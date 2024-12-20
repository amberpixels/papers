package jalapeno_test

import (
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

func TestParser_Headings(t *testing.T) {
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
	}

	// Run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			blocks, err := parserInstance.ParseBlocks(tc.source)
			require.NoError(t, err, "Parsing failed")
			assert.Equal(t, tc.expectedBlocks, blocks, "Generated blocks do not match expected blocks")
		})
	}
}
