package jalapeno

import (
	"strings"
)

func sanitizeBlockLanguage(language string) string {
	if language == "" {
		language = "plain text"
	}
	return language
}

// html2notion is a hacky function that converts HTML to Notion-compatible text
// It's very simple, and in future is considered to be more complex
// Deprecated: don't tend to use it very often, it's subject to change
//
//	TODO(amberpixels): add support HTML
//	  Note: we want to support basic HTML that is usually used in Markdown:
//	  <p> (for centering), <img> (for images), <br> (for line breaks)
//	  Also we can support <b>, <i>, <s>, <code> tags
func html2notion(contentHTML string) string {
	// sanitizing first
	contentHTML = strings.TrimSpace(contentHTML)
	contentHTML = strings.ToLower(contentHTML)

	// Handling edge cases:
	switch contentHTML {
	case "<br>":
		return "\n"
	default:
		return contentHTML // simply return raw html back
	}
}
