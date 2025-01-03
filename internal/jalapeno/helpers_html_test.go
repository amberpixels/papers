package jalapeno

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHtml2Notion_Paragraph(t *testing.T) {
	blocks, rts, err := html2notion(`<p>Hello <strong>World</strong></p>`)
	assert.Empty(t, rts)
	require.NoError(t, err)

	assert.Len(t, blocks, 1)
	_ = blocks
	//nt.NewParagraphBlock(nt.Paragraph{
	//	RichText: []nt.RichText{},
	//})
	//assert.Equal(t, nt.ParagraphBlock{}, blocks[0])
	//fmt.Printf("%#v", blocks[0])
}
