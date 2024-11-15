package md2nt

import nt "github.com/jomei/notionapi"

// NtRichTextBuilder is a builder for nt.RichText
// It builds a nt.RichText from a given source and optionally can decorate it aftew
type NtRichTextBuilder struct {
	build      func(source []byte) *nt.RichText
	decorators []func(*nt.RichText)
}

type NtRichTextBuilders []*NtRichTextBuilder

func NewNtRichTextBuilder(build func(source []byte) *nt.RichText) *NtRichTextBuilder {
	return &NtRichTextBuilder{
		build:      build,
		decorators: make([]func(*nt.RichText), 0),
	}
}

func (b *NtRichTextBuilder) DecorateWith(d func(*nt.RichText)) {
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
	boldDecorator = func(t *nt.RichText) {
		if t.Annotations == nil {
			t.Annotations = &nt.Annotations{}
		}
		t.Annotations.Bold = true
	}

	italicDecorator = func(t *nt.RichText) {
		if t.Annotations == nil {
			t.Annotations = &nt.Annotations{}
		}
		t.Annotations.Italic = true
	}

	strikethroughDecorator = func(t *nt.RichText) {
		if t.Annotations == nil {
			t.Annotations = &nt.Annotations{}
		}
		t.Annotations.Strikethrough = true
	}

	codeDecorator = func(t *nt.RichText) {
		if t.Annotations == nil {
			t.Annotations = &nt.Annotations{}
		}
		t.Annotations.Code = true
	}

	linkDecorator = func(urlDestination string) func(t *nt.RichText) {
		return func(t *nt.RichText) {
			if t.Text == nil {
				t.Text = &nt.Text{}
			}
			t.Text.Link = &nt.Link{Url: urlDestination}
		}
	}
)

// NtBlockBuilder is func that makes a nt.Block from given []bytes source
type NtBlockBuilder func(source []byte) nt.Block

type NtBlockBuilders []NtBlockBuilder

func (builders NtBlockBuilders) Build(source []byte) []nt.Block {
	result := make([]nt.Block, 0)
	for _, builder := range builders {
		result = append(result, builder(source))
	}
	return result
}
