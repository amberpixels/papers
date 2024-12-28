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

	f("Empty document", "", nt.Blocks{})

	// -----------------
	// --- HEADINGS ----
	// -----------------

	f("Empty Heading1", "#", nt.Blocks{
		nt.NewHeading1Block(nt.Heading{
			RichText: []nt.RichText{
				*nt.NewTextRichText(""),
			},
		}),
	})
	f("Empty Heading1 with whitespace", "#   ", nt.Blocks{
		nt.NewHeading1Block(nt.Heading{
			RichText: []nt.RichText{
				*nt.NewTextRichText(""), // whitespace is trimmed
			},
		}),
	})
	f("Single Heading1", "# Heading", nt.Blocks{
		nt.NewHeading1Block(nt.Heading{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Heading"),
			},
		}),
	})
	f("Alternative heading 1", "Heading\n===", nt.Blocks{
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

	f("Alternative Heading2", "Heading Foobar\n--------------", nt.Blocks{
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

	// -------------------
	// --- PARAGRAPHS ----
	// -------------------

	f("Empty paragram (with whitespace)", " \t\t ", nt.Blocks{})
	f("Empty paragram (multiline with whitespace)", " \t\n\n\t ", nt.Blocks{})

	f("Simple oneline paragraph", `Hello Foobar`, nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Hello"),
				*nt.NewTextRichText(" Foobar"),
			},
			Children: nt.Blocks{},
		}),
	})

	f("Simple multiline paragraph", `
This is the first line
this is the second
and the third.
`,
		nt.Blocks{
			nt.NewParagraphBlock(nt.Paragraph{
				RichText: []nt.RichText{
					*nt.NewTextRichText("This is the first"),
					*nt.NewTextRichText(" line"),
					*nt.NewTextRichText("this is the"),
					*nt.NewTextRichText(" second"),
					*nt.NewTextRichText("and the"),
					*nt.NewTextRichText(" third."),
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

	f("Paragraph with inline code", "This is `inline code`", nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("This is "),
				*nt.NewTextRichText("inline code").AnnotateCode(),
			},
			Children: nt.Blocks{},
		}),
	})

	// --------------
	// --- LINKS ----
	// --------------

	f("Simple link", "[OpenAI](https://openai.com)", nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewLinkRichText("OpenAI", "https://openai.com"),
			},
			Children: nt.Blocks{},
		}),
	})

	f("Simple inline link", "https://openai.com", nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewLinkRichText("https://openai.com", "https://openai.com"),
			},
			Children: nt.Blocks{},
		}),
	})

	f("Simple explicit link", "<fake@gmail.com>", nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewLinkRichText("fake@gmail.com", "fake@gmail.com"),
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

	f("Formatting links", `
I love supporting the **[EFF](https://eff.org)**.
This is the *[Markdown Guide](https://www.markdownguide.org)*.
See the section on [`+"`code`"+`](#code).`, nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("I love supporting the "),
				*nt.NewLinkRichText("EFF", "https://eff.org").AnnotateBold(),
				*nt.NewTextRichText("."),
				*nt.NewTextRichText("This is the "),
				*nt.NewLinkRichText("Markdown Guide", "https://www.markdownguide.org").AnnotateItalic(),
				*nt.NewTextRichText("."),
				*nt.NewTextRichText("See the section on "),
				*nt.NewLinkRichText("code", "#code").AnnotateCode(),
				*nt.NewTextRichText("."),
			},
			Children: nt.Blocks{},
		}),
	})

	// --------------
	// --- LISTS ----
	// --------------

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

	f("Unordered List Items With Numbers", `
- 1968\. A great year!
- I think 1969 was second best.`,
		nt.Blocks{
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("1968\\. A great year"),
					*nt.NewTextRichText("!"),
				},
				Children: nt.Blocks{},
			}),
			nt.NewBulletedListItemBlock(nt.ListItem{
				RichText: []nt.RichText{
					*nt.NewTextRichText("I think 1969 was second"),
					*nt.NewTextRichText(" best."),
				},
				Children: nt.Blocks{},
			}),
		})

	// --------------
	// --- TASKS ----
	// --------------

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

	f("Bulleted List with Nested TODO List", `
- Item 1
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

	// --------------
	// --- CODE -----
	// --------------

	f("Simple Non-Fenced Code Block", `
	package main
	func main() {
		fmt.Println("Hello, World!")
	}`,
		nt.Blocks{
			nt.NewCodeBlock(nt.Code{
				RichText: []nt.RichText{
					*nt.NewTextRichText("package main\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}"),
				},
			}),
		})

	f("Simple Fenced Code Block", "```go\npackage main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n```",
		nt.Blocks{
			nt.NewCodeBlock(nt.Code{
				RichText: []nt.RichText{
					*nt.NewTextRichText("package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}"),
				},
				Language: "go",
			}),
		})

	f("Heading H1 + Heading H2 + Fenced Code Block", fmt.Sprintf(`# Heading 1
## Heading 2

%sgo
package main

func main() {
		fmt.Println("Hello, World!")
}
%s`, "```", "```"),
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
			nt.NewCodeBlock(nt.Code{
				RichText: []nt.RichText{
					*nt.NewTextRichText("package main\n\nfunc main() {\n\t\tfmt.Println(\"Hello, World!\")\n}"),
				},
				Language: "go",
			}),
		})

	// --------------
	// --- TABLES ---
	// --------------

	f("Simple Markdown Table", `| Column1 | Column2 |
|---------|---------|
| Value1  | Value2  |
| Value3  | Value4  |`,
		nt.Blocks{
			nt.NewTableBlock(nt.Table{
				TableWidth:      2,
				HasColumnHeader: true,
				Children: nt.Blocks{
					nt.NewTableRowBlock(nt.TableRow{
						Cells: [][]nt.RichText{
							{*nt.NewTextRichText("Column1")},
							{*nt.NewTextRichText("Column2")},
						},
					}),
					nt.NewTableRowBlock(nt.TableRow{
						Cells: [][]nt.RichText{
							{*nt.NewTextRichText("Value1")},
							{*nt.NewTextRichText("Value2")},
						},
					}),
					nt.NewTableRowBlock(nt.TableRow{
						Cells: [][]nt.RichText{
							{*nt.NewTextRichText("Value3")},
							{*nt.NewTextRichText("Value4")},
						},
					}),
				},
			}),
		})

	f("Markdown Table with Emphasis in Cells", `| Column1    | Column2    |
|------------|------------|
| *Italic*   | **Bold**   |
| ~~Strike~~ | `+"`Code`"+` |`,
		nt.Blocks{
			nt.NewTableBlock(nt.Table{
				TableWidth:      2,
				HasColumnHeader: true,
				Children: nt.Blocks{
					nt.NewTableRowBlock(nt.TableRow{
						Cells: [][]nt.RichText{
							{*nt.NewTextRichText("Column1")},
							{*nt.NewTextRichText("Column2")},
						},
					}),
					nt.NewTableRowBlock(nt.TableRow{
						Cells: [][]nt.RichText{
							{*nt.NewTextRichText("Italic").AnnotateItalic()},
							{*nt.NewTextRichText("Bold").AnnotateBold()},
						},
					}),
					nt.NewTableRowBlock(nt.TableRow{
						Cells: [][]nt.RichText{
							{*nt.NewTextRichText("Strike").AnnotateStrikethrough()},
							{*nt.NewTextRichText("Code").AnnotateCode()},
						},
					}),
				},
			}),
		})

	f("Markdown Table with URLs in Cells", `| Column1       | Column2        |
|---------------|----------------|
| [Google](https://google.com) | [OpenAI](https://openai.com) |
| Plain URL     | https://example.com |`,
		nt.Blocks{
			nt.NewTableBlock(nt.Table{
				TableWidth:      2,
				HasColumnHeader: true,
				Children: nt.Blocks{
					nt.NewTableRowBlock(nt.TableRow{
						Cells: [][]nt.RichText{
							{*nt.NewTextRichText("Column1")},
							{*nt.NewTextRichText("Column2")},
						},
					}),
					nt.NewTableRowBlock(nt.TableRow{
						Cells: [][]nt.RichText{
							{*nt.NewLinkRichText("Google", "https://google.com")},
							{*nt.NewLinkRichText("OpenAI", "https://openai.com")},
						},
					}),
					nt.NewTableRowBlock(nt.TableRow{
						Cells: [][]nt.RichText{
							{*nt.NewTextRichText("Plain"), *nt.NewTextRichText(" URL")},
							{*nt.NewLinkRichText("https://example.com", "https://example.com")},
						},
					}),
				},
			}),
		})

	// -------------------
	// --- BLOCKQUOTES ---
	// -------------------

	f("Simple Blockquote", `> This is a block quote`,
		nt.Blocks{
			nt.NewQuoteBlock(nt.Quote{
				RichText: []nt.RichText{
					*nt.NewTextRichText("This is a block"),
					*nt.NewTextRichText(" quote"),
				},
				Children: nt.Blocks{},
			}),
		})

	f("Multiline Blockquote", "> This is a block quote\n>\n> This is the last line",
		nt.Blocks{
			nt.NewQuoteBlock(nt.Quote{
				RichText: []nt.RichText{
					*nt.NewTextRichText("This is a block"),
					*nt.NewTextRichText(" quote"),
					*nt.NewTextRichText("This is the last"),
					*nt.NewTextRichText(" line"),
				},
				Children: nt.Blocks{},
			}),
		})

	f("Nested Blockquotes", `> This is a blockquote
> > This is a nested blockquote`,
		nt.Blocks{
			nt.NewQuoteBlock(nt.Quote{
				RichText: []nt.RichText{
					*nt.NewTextRichText("This is a"),
					*nt.NewTextRichText(" blockquote"),
				},
				Children: nt.Blocks{
					nt.NewQuoteBlock(nt.Quote{
						RichText: []nt.RichText{
							*nt.NewTextRichText("This is a nested"),
							*nt.NewTextRichText(" blockquote"),
						},
						Children: nt.Blocks{},
					}),
				},
			}),
		})

	f("Blockquotes with nested elements", `
> #### The quarterly results look great!
> Second line
`,
		nt.Blocks{
			nt.NewQuoteBlock(nt.Quote{
				RichText: []nt.RichText{},
				Children: nt.Blocks{
					nt.NewHeading3Block(nt.Heading{
						RichText: []nt.RichText{
							*nt.NewTextRichText("The quarterly results look great"),
							*nt.NewTextRichText("!"),
						},
					}),
					nt.NewParagraphBlock(nt.Paragraph{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Second"),
							*nt.NewTextRichText(" line"),
						},
						Children: nt.Blocks{},
					}),
				},
			}),
		},
	)

	f("Blockquotes with more nested elements", `
> #### The quarterly results look great!
>
> - Revenue was off the chart.
> - Profits were higher than ever.
>
>  *Everything* is going according to **plan**.
`,
		nt.Blocks{
			nt.NewQuoteBlock(nt.Quote{
				RichText: []nt.RichText{},
				Children: nt.Blocks{
					nt.NewHeading3Block(nt.Heading{
						RichText: []nt.RichText{
							*nt.NewTextRichText("The quarterly results look great"),
							*nt.NewTextRichText("!"),
						},
					}),

					nt.NewBulletedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Revenue was off the"),
							*nt.NewTextRichText(" chart."),
						},
						Children: nt.Blocks{},
					}),
					nt.NewBulletedListItemBlock(nt.ListItem{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Profits were higher than"),
							*nt.NewTextRichText(" ever."),
						},
						Children: nt.Blocks{},
					}),

					nt.NewParagraphBlock(nt.Paragraph{
						RichText: []nt.RichText{
							*nt.NewTextRichText("Everything").AnnotateItalic(),
							*nt.NewTextRichText(" is going according to "),
							*nt.NewTextRichText("plan").AnnotateBold(),
							*nt.NewTextRichText("."),
						},
						Children: nt.Blocks{},
					}),
				},
			}),
		},
	)

	// --------------
	// --- IMAGES ---
	// --------------

	f("Image Without Caption", `![](https://example.com/image.png)`,
		nt.Blocks{
			nt.NewParagraphBlock(nt.Paragraph{
				RichText: []nt.RichText{},
				Children: nt.Blocks{
					nt.NewImageBlock(nt.Image{
						Type: "external",
						External: &nt.FileObject{
							URL: "https://example.com/image.png",
						},
						Caption: []nt.RichText{}, // No caption
					}),
				},
			}),
		})

	f("Image With Caption", `![Alt text](https://example.com/image.png)`,
		nt.Blocks{
			nt.NewParagraphBlock(nt.Paragraph{
				RichText: []nt.RichText{},
				Children: nt.Blocks{
					nt.NewImageBlock(nt.Image{
						Type: "external",
						External: &nt.FileObject{
							URL: "https://example.com/image.png",
						},
						Caption: []nt.RichText{
							*nt.NewTextRichText("Alt text"),
						},
					}),
				},
			}),
		})

	// --------------
	// --- MISC -----
	// --------------

	f("Horizontal Rule", `---`, nt.Blocks{
		nt.NewDividerBlock(),
	})

	// --------------
	// --- HTML -----
	// --------------

	f("Basic HTML", `Hello<br>World`, nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("Hello"),
				*nt.NewTextRichText("\n"),
				*nt.NewTextRichText("World"),
			},
			Children: nt.Blocks{},
		}),
	})

	// FOR NOW: we're OK with simply Paragraph with raw HTML
	f("HTML Block", `<div>
  <p>This is an HTML block</p>
</div>`, nt.Blocks{
		nt.NewParagraphBlock(nt.Paragraph{
			RichText: []nt.RichText{
				*nt.NewTextRichText("<div>\n  <p>this is an html block</p>\n</div>"),
			},
			Children: nil,
		}),
	})

	run()
}
