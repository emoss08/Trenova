package report

import (
	"fmt"
	"io"

	"github.com/emoss08/assay/internal/ui"
)

// Lines is a minimal coloured line writer for commands whose output is a short
// narrative rather than a table.
type Lines struct {
	out   io.Writer
	paint ui.Painter
}

func NewLines(out io.Writer, useColor bool) Lines {
	return Lines{out: out, paint: ui.New(useColor)}
}

// Painter exposes the styling layer for callers that need more than plain
// lines — badges, bars, tables — so every command shares one look.
func (l Lines) Painter() ui.Painter { return l.paint }

func (l Lines) Line(format string, args ...any) { l.write(nil, format, args...) }

func (l Lines) Warn(format string, args ...any) { l.write(l.paint.Warn, format, args...) }

func (l Lines) Muted(format string, args ...any) { l.write(l.paint.Muted, format, args...) }

func (l Lines) Good(format string, args ...any) { l.write(l.paint.Good, format, args...) }

func (l Lines) Bad(format string, args ...any) { l.write(l.paint.Bad, format, args...) }

func (l Lines) write(paint func(string) string, format string, args ...any) {
	text := fmt.Sprintf(format, args...)
	if paint != nil {
		text = paint(text)
	}
	fmt.Fprintln(l.out, text)
}
