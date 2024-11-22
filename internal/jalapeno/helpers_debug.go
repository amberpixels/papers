package jalapeno

import (
	"fmt"
	"log/slog"
)

var (
	DebugSource []byte
)

func SetDebugSource(s []byte) { DebugSource = s }

func DebugRichTexts(rts NtRichTextBuilders, prefix string) {
	if len(DebugSource) == 0 {
		return
	}

	slog.Debug(fmt.Sprintf("%s as %d rich texts: ", prefix, len(rts)))
	for i, rt := range rts {
		built := rt.Build(DebugSource)
		slog.Debug(fmt.Sprintf("%d: %s", i, built.Text.Content))
	}
}
func DebugBlock(b *NtBlockBuilder, prefix string) {
	if len(DebugSource) == 0 {
		return
	}

	slog.Debug(fmt.Sprintf("%s as block ", prefix))

	built := b.Build(DebugSource)
	slog.Debug(fmt.Sprintf("-> %s", built.GetRichTextString()))
}
