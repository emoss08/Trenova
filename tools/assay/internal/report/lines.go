package report

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

// Lines is a minimal coloured line writer for commands whose output is a short
// narrative rather than a table.
type Lines struct {
	out   io.Writer
	plain func(...any) string
	warn  func(...any) string
	muted func(...any) string
	good  func(...any) string
}

func NewLines(out io.Writer, useColor bool) Lines {
	tint := func(attrs ...color.Attribute) func(...any) string {
		c := color.New(attrs...)
		if useColor {
			c.EnableColor()
		} else {
			c.DisableColor()
		}

		return c.SprintFunc()
	}

	return Lines{
		out:   out,
		plain: tint(),
		warn:  tint(color.FgYellow),
		muted: tint(color.Faint),
		good:  tint(color.FgGreen),
	}
}

func (l Lines) Line(format string, args ...any)  { l.write(l.plain, format, args...) }
func (l Lines) Warn(format string, args ...any)  { l.write(l.warn, format, args...) }
func (l Lines) Muted(format string, args ...any) { l.write(l.muted, format, args...) }
func (l Lines) Good(format string, args ...any)  { l.write(l.good, format, args...) }

func (l Lines) write(paint func(...any) string, format string, args ...any) {
	fmt.Fprintln(l.out, paint(fmt.Sprintf(format, args...)))
}
