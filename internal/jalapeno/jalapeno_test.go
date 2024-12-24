package jalapeno_test

import (
	"fmt"
	"testing"

	"github.com/amberpixels/peppers/internal/jalapeno"
	"github.com/amberpixels/peppers/internal/testhelpers"
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
	type AssertFunc = func(t *testing.T, source string, expectedBlocks nt.Blocks)
	type TestFunc = func(name string, source string, expectedBlocks nt.Blocks)

	f, ff, xf, run := testhelpers.GenerateCases[TestFunc, AssertFunc](t, func(t *testing.T, source string, expectedBlocks nt.Blocks) {
		blocks, err := parserInstance.ParseBlocks([]byte(source))

		require.NoError(t, err, "Parsing failed")
		assert.Len(t, blocks, len(expectedBlocks), "Generated blocks do not match expected blocks")
		for i, b := range blocks {
			assert.Equal(t, expectedBlocks[i].GetType(), b.GetType(),
				fmt.Sprintf("Generated block[%d] do not match expected block", i))
			assert.Equal(t, expectedBlocks[i], b,
				fmt.Sprintf("Generated block[%d] do not match expected block", i))
		}
	})
	_, _, _ = f, ff, xf

	f("Single Heading1", "# Heading", nt.Blocks{
		nt.NewHeading1Block(nt.Heading{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Heading"),
			},
		}),
	})
	f("Multiple-words Heading2", "## Heading Foobar", nt.Blocks{
		nt.NewHeading2Block(nt.Heading{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Heading"),
				*nt.NewTextRichText(" Foobar"),
			},
		}),
	})
	f("Headings H1 to H4", `# Heading 1
## Heading 2
### Heading 3
#### Heading 4`,
		nt.Blocks{
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
		})

	f("Just a paragraph", `Hello Foobar`, nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Hello"),
				*nt.NewTextRichText(" Foobar"),
			},
			Children: nt.Blocks{},
		}),
	})

	f("Paragraph with emphasis", `Hello **foobar**`, nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Hello "),
				*nt.NewTextRichText("foobar").AnnotateBold(),
			},
			Children: nt.Blocks{},
		}),
	})

	f("Paragraph with italic", `Hello *foobar*`, nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Hello "),
				*nt.NewTextRichText("foobar").AnnotateItalic(),
			},
			Children: nt.Blocks{},
		}),
	})

	f("Paragraph with strikethrough", `Hello ~~no foobar~~`, nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Hello "),
				*nt.NewTextRichText("no foobar").AnnotateStrikethrough(),
			},
			Children: nt.Blocks{},
		}),
	})

	f("Paragraph with different annotations", `This is a **bold text** 1, *italic text* 2, and ~~strikethrough text~~ 3.`, nt.Blocks{
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
	})

	f("Paragraph with one link", `Visit [OpenAI](https://openai.com)`, nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Visit "),
				*nt.NewLinkRichText("OpenAI", "https://openai.com"),
			},
			Children: nt.Blocks{},
		}),
	})

	f("Paragraph with an Inline link", `Hello https://openai.com`, nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Hello "),
				*nt.NewLinkRichText("https://openai.com", "https://openai.com"),
			},
			Children: nt.Blocks{},
		}),
	})

	f("Heading with non-inline link", `# Hello [OpenAI](https://openai.com)`, nt.Blocks{
		nt.NewHeading1Block(nt.Heading{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Hello "),
				*nt.NewLinkRichText("OpenAI", "https://openai.com"),
			},
		}),
	})

	f("Heading with inline link", `# Hello https://openai.com`, nt.Blocks{
		nt.NewHeading1Block(nt.Heading{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Hello "),
				*nt.NewLinkRichText("https://openai.com", "https://openai.com"),
			},
		}),
	})

	f("Heading2 with an inline link in brackets", `## Hello (https://openai.com)`, nt.Blocks{
		nt.NewHeading2Block(nt.Heading{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Hello ("),
				*nt.NewLinkRichText("https://openai.com", "https://openai.com"),
				*nt.NewTextRichText(")"),
			},
		}),
	})

	f("Heading with annotations + link", `# **ULID** Wrapper for *PostgreSQL* and *GORM* [link inside](https://github.com/oklog/ulid)`, nt.Blocks{
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
	})

	f("Paragraph with inline code", "This is `inline code`", nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("This is "),
				*nt.NewTextRichText("inline code").AnnotateCode(),
			},
			Children: nt.Blocks{},
		}),
	})

	f("Heading with inline code", "# This is `inline code`", nt.Blocks{
		nt.NewHeading1Block(nt.Heading{
			RichText: []nt.RichText{
				*nt.NewTextRichText("This is "),
				*nt.NewTextRichText("inline code").AnnotateCode(),
			},
		}),
	})

	f("Headings + Paragraph with code inline", `# Your Readme Package name

## Overview
`+"The `packageName` package provides a set of function for doing something useful.",
		nt.Blocks{
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
		})

	f("Simple Bulleted List", `- Item 1
- Item 2
- Item 3`,
		nt.Blocks{
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 1"),
				},
				Children: nt.Blocks{},
			}),
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 2"),
				},
				Children: nt.Blocks{},
			}),
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 3"),
				},
				Children: nt.Blocks{},
			}),
		})

	f("Simple Numbered List", `1. Item 1
2. Item 2
3. Item 3`,
		nt.Blocks{
			nt.NewNumberedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 1"),
				},
				Children: nt.Blocks{},
			}),
			nt.NewNumberedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 2"),
				},
				Children: nt.Blocks{},
			}),
			nt.NewNumberedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 3"),
				},
				Children: nt.Blocks{},
			}),
		})

	f("Nested Bulleted List", `- Item 1
  - Subitem 1.1
  - Subitem 1.2
  - Subitem 1.3
- Item 2
  - Subitem 2.1
  - Subitem 2.2
  - Subitem 2.3`,
		nt.Blocks{
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 1"),
				},
				Children: nt.Blocks{
					nt.NewBulletedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 1.1"),
						},
						Children: nt.Blocks{},
					}),
					nt.NewBulletedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 1.2"),
						},
						Children: nt.Blocks{},
					}),
					nt.NewBulletedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 1.3"),
						},
						Children: nt.Blocks{},
					}),
				},
			}),
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 2"),
				},
				Children: nt.Blocks{
					nt.NewBulletedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 2.1"),
						},
						Children: nt.Blocks{},
					}),
					nt.NewBulletedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 2.2"),
						},
						Children: nt.Blocks{},
					}),
					nt.NewBulletedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 2.3"),
						},
						Children: nt.Blocks{},
					}),
				},
			}),
		})

	f("Nested Numbered List Inside Bulleted List", `- Item 1
  1. Subitem 1.1
  2. Subitem 1.2
  3. Subitem 1.3
- Item 2
  1. Subitem 2.1
  2. Subitem 2.2
  3. Subitem 2.3`,
		nt.Blocks{
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 1"),
				},
				Children: nt.Blocks{
					nt.NewNumberedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 1.1"),
						},
						Children: nt.Blocks{},
					}),
					nt.NewNumberedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 1.2"),
						},
						Children: nt.Blocks{},
					}),
					nt.NewNumberedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 1.3"),
						},
						Children: nt.Blocks{},
					}),
				},
			}),
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 2"),
				},
				Children: nt.Blocks{
					nt.NewNumberedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 2.1"),
						},
						Children: nt.Blocks{},
					}),
					nt.NewNumberedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 2.2"),
						},
						Children: nt.Blocks{},
					}),
					nt.NewNumberedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Subitem"),
							*nt.NewTextRichText(" 2.3"),
						},
						Children: nt.Blocks{},
					}),
				},
			}),
		})

	f("3-Level Deep Nested List with Mixed Bulleted and Numbered Items",

		`- **Top Level 1**
  1. Second Level 1.1 with [a link](https://example.com)
  2. *Second Level 1.2*
      - Third Level 1.2.1
      - Third Level 1.2.2
- Top Level 2
  1. Second Level 2.1
      - Third Level 2.1.1
      - **Third Level 2.1.2** with *italic text*
  2. Second Level 2.2`,
		nt.Blocks{
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Top Level 1").AnnotateBold(),
				},
				Children: nt.Blocks{
					nt.NewNumberedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Second Level 1.1 with "),
							*nt.NewLinkRichText("a link", "https://example.com"),
						},
						Children: nt.Blocks{},
					}),
					nt.NewNumberedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Second Level 1.2").AnnotateItalic(),
						},
						Children: nt.Blocks{
							nt.NewBulletedListItemBlock(nt.ListItem{
								RichText: []nt.RichText{
									*nt.NewTextRichText("Third Level"),
									*nt.NewTextRichText(" 1.2.1"),
								},
								Children: nt.Blocks{},
							}),
							nt.NewBulletedListItemBlock(nt.ListItem{
								RichText: []nt.RichText{
									*nt.NewTextRichText("Third Level"),
									*nt.NewTextRichText(" 1.2.2"),
								},
								Children: nt.Blocks{},
							}),
						},
					}),
				},
			}),

			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Top Level"),
					*nt.NewTextRichText(" 2"),
				},
				Children: nt.Blocks{
					nt.NewNumberedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Second Level"),
							*nt.NewTextRichText(" 2.1"),
						},
						Children: nt.Blocks{
							nt.NewBulletedListItemBlock(nt.ListItem{
								RichText: []nt.RichText{
									*nt.NewTextRichText("Third Level"),
									*nt.NewTextRichText(" 2.1.1"),
								},
								Children: nt.Blocks{},
							}),
							nt.NewBulletedListItemBlock(nt.ListItem{
								RichText: []nt.RichText{
									*nt.NewTextRichText("Third Level 2.1.2").AnnotateBold(),
									*nt.NewTextRichText(" with "),
									*nt.NewTextRichText("italic text").AnnotateItalic(),
								},
								Children: nt.Blocks{},
							}),
						},
					}),
					nt.NewNumberedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Second Level"),
							*nt.NewTextRichText(" 2.2"),
						},
						Children: nt.Blocks{},
					}),
				},
			}),
		})

	f("Simple TODO List", `- [ ] Item 1
- [ ] Item 2
- [x] Item 3`,
		nt.Blocks{
			nt.NewToDoBlock(nt.ToDo{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 1"),
				},
			}),
			nt.NewToDoBlock(nt.ToDo{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 2"),
				},
			}),
			nt.NewToDoBlock(nt.ToDo{
				Checked: true,
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 3"),
				},
			}),
		})

	f("Simple TODO List + emphasis", `- [ ] Item 1
- [ ] _Item 2_
- [x] **Item 3**`,
		nt.Blocks{
			nt.NewToDoBlock(nt.ToDo{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 1"),
				},
			}),
			nt.NewToDoBlock(nt.ToDo{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item 2").AnnotateItalic(),
				},
			}),
			nt.NewToDoBlock(nt.ToDo{
				Checked: true,
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item 3").AnnotateBold(),
				},
			}),
		})

	ff("Bulleted List with Nested TODO List", `- Item 1
  - [ ] TODO 1
  - [x] TODO 2
- Item 2`,
		nt.Blocks{
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 1"),
				},
				Children: nt.Blocks{
					nt.NewToDoBlock(nt.ToDo{
						RichText: []nt.RichText{
							*nt.NewTextRichText("TODO"),
							*nt.NewTextRichText(" 1"),
						},
					}),
					nt.NewToDoBlock(nt.ToDo{
						Checked: true,
						RichText: []nt.RichText{
							*nt.NewTextRichText("TODO"),
							*nt.NewTextRichText(" 2"),
						},
					}),
				},
			}),
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("Item"),
					*nt.NewTextRichText(" 2"),
				},
				Children: nt.Blocks{},
			}),
		})

	run()
}
