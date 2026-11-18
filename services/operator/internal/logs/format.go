package logs

import (
	"strings"
	"time"
)

var escaper = strings.NewReplacer("\r\n", `\n`, "\n", `\n`, "\r", `\r`)

func message(e *Entry, redact func(string) string) string {
	return escaper.Replace(redact(strings.TrimRight(e.Line, "\r\n")))
}

func bundleLine(e *Entry, redact func(string) string) string {
	var b strings.Builder
	msg := message(e, redact)
	b.Grow(len(msg) + len(e.Pod) + len(e.Container) + 40)
	b.WriteString(e.Time.UTC().Format(time.RFC3339Nano))
	b.WriteByte(' ')
	b.WriteString(orDash(e.Pod))
	b.WriteByte(' ')
	b.WriteString(orDash(e.Container))
	b.WriteByte(' ')
	b.WriteString(msg)
	b.WriteByte('\n')
	return b.String()
}

func viewLine(e *Entry, redact func(string) string) string {
	return orDash(e.Pod) + "/" + orDash(e.Container) + ": " +
		e.Time.UTC().Format(time.RFC3339Nano) + " " + message(e, redact) + "\n"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
