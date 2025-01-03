package jalapeno

import nt "github.com/jomei/notionapi"

// NtBlockBuilder is func that makes a nt.Block from given []bytes source
type NtBlockBuilder struct {
	build      func(source []byte) nt.Block
	decorators []func([]byte, nt.Block)
}
type NtBlockBuilders []*NtBlockBuilder

func NewNtBlockBuilder(build func(source []byte) nt.Block) *NtBlockBuilder {
	return &NtBlockBuilder{
		build:      build,
		decorators: make([]func([]byte, nt.Block), 0),
	}
}

func (b *NtBlockBuilder) Build(source []byte) nt.Block {
	block := b.build(source)
	for _, d := range b.decorators {
		d(source, block)
	}
	return block
}

func (b *NtBlockBuilder) DecorateWith(d func(source []byte, block nt.Block)) {
	b.decorators = append(b.decorators, d)
}

func (builders NtBlockBuilders) Build(source []byte) []nt.Block {
	result := make([]nt.Block, 0)
	for _, builder := range builders {
		// Some nodes (e.g. markdown hacky comments) can be handled as nil empty blocks
		// let's just filter them out here
		if built := builder.Build(source); built != nil {
			result = append(result, built)
		}
	}

	return result
}
