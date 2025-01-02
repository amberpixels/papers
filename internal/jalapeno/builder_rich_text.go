package jalapeno

import nt "github.com/jomei/notionapi"

// NtRichTextBuilder is a builder for nt.RichText
// It builds a nt.RichText from a given source and optionally can decorate it aftew
type NtRichTextBuilder struct {
	build      func(source []byte) *nt.RichText
	decorators []RichTextDecorator
}

type RichTextDecorator func(*nt.RichText)

type NtRichTextBuilders []*NtRichTextBuilder

func NewNtRichTextBuilder(build func(source []byte) *nt.RichText) *NtRichTextBuilder {
	return &NtRichTextBuilder{
		build:      build,
		decorators: make([]RichTextDecorator, 0),
	}
}

func (b *NtRichTextBuilder) DecorateWith(d RichTextDecorator) {
	b.decorators = append(b.decorators, d)
}

func (b *NtRichTextBuilder) Build(source []byte) *nt.RichText {
	richText := b.build(source)
	for _, d := range b.decorators {
		d(richText)
	}
	return richText
}

func (builders NtRichTextBuilders) Build(source []byte) []nt.RichText {
	result := make([]nt.RichText, 0)
	for _, builder := range builders {
		result = append(result, *builder.Build(source))
	}
	return result
}

//
// RichText Decorators
//

var (
	boldDecorator          = func(t *nt.RichText) { t.AnnotateBold() }
	italicDecorator        = func(t *nt.RichText) { t.AnnotateItalic() }
	strikethroughDecorator = func(t *nt.RichText) { t.AnnotateStrikethrough() }
	codeDecorator          = func(t *nt.RichText) { t.AnnotateCode() }

	linkDecorator = func(urlDestination string) func(*nt.RichText) {
		return func(t *nt.RichText) { t.MakeLink(urlDestination) }
	}
)

var (
	_ RichTextDecorator = boldDecorator
	_ RichTextDecorator = italicDecorator
	_ RichTextDecorator = strikethroughDecorator
	_ RichTextDecorator = codeDecorator
	_ RichTextDecorator = linkDecorator("google.com")
)
