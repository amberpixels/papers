# peppers 🌶️

**pprs** is a CLI tool designed to transform Markdown documentation into Notion pages.
If you're a developer managing multiple repositories, **pprs** aims to keep your README.md files as 
the single source of truth while making them easily searchable and viewable in Notion. 
This tool is ideal for automating the documentation process in CI setups, ensuring your project documentation stays up to date.

> **Note:** This project is in active development and is **NOT READY** for production use. 

## How It Works

**peppers** uses the `jalapeno` library to convert Markdown AST (parsed with [Goldmark](https://github.com/yuin/goldmark)) 
into Notion blocks via the [Notion API](https://developers.notion.com/).

## Current Features

- **Markdown Syntax Supported:**
    - Headings
    - Emphasis (bold, italic, strikethrough)
    - Lists (bulleted and numbered)
    - Code blocks
    - Inline code
    - Links and autolinks
    - Blockquotes
    - Horizontal rules (semantic breaks)
    - Basic images (`![]()` syntax)
    - Basic tables (not well tested with nested things inside)
    - Limited HTML support (`<br>` only)

- **Notion Page Creation:**
    - Converts a `.md` file to Notion blocks and uploads them to a Notion page using environment variables for configuration.

## Limitations (Work in Progress)

- **Markdown Syntax Not Yet Supported:**
    - Advanced tables (tables + things inside)
    - Footnotes
    - Definition lists
    - Task lists
    - Nested lists
    - Escape characters

- **HTML Support:** Only `<br>` is supported. Other HTML elements are not parsed or converted.

## Roadmap

- [x] Add support for tables
  - [ ] Add support for nested things inside tables
- [x] Add support for blockquotes
- [ ] Add horizontal rules
- [ ] Handle footnotes and definition lists
- [ ] Implement task lists and nested lists
- [ ] Improve HTML support
- [ ] Refactor and move `jalapeno` to `pkg` for independent use

