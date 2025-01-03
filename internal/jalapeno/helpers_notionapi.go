package jalapeno

import (
	nt "github.com/jomei/notionapi"
)

func sanitizeBlockLanguage(language string) string {
	if language == "" {
		language = "plain text"
	}
	return language
}

func nonEmptyRichTexts(rts []nt.RichText) []nt.RichText {
	for i, rt := range rts {
		if rt.PlainText == "" {
			rts = append(rts[:i], rts[i+1:]...)
		}
	}
	return rts
}
