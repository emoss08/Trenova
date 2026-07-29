package templateengine

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"
	texttemplate "text/template"
)

// Channel names one output slot of a template version. It mirrors the domain's
// channel enum without importing it, so this package stays free of domain deps.
type Channel string

const (
	ChannelPDF               = Channel("PDF")
	ChannelEmailHTML         = Channel("EmailHTML")
	ChannelEmailText         = Channel("EmailText")
	ChannelSubject           = Channel("Subject")
	ChannelNotificationTitle = Channel("NotificationTitle")
	ChannelNotificationBody  = Channel("NotificationBody")
)

func (c Channel) String() string { return string(c) }

// IsHTML reports whether the channel produces markup, and therefore needs
// contextual auto-escaping and the HTML sanitizer.
func (c Channel) IsHTML() bool {
	switch c {
	case ChannelPDF, ChannelEmailHTML:
		return true
	case ChannelEmailText, ChannelSubject, ChannelNotificationTitle, ChannelNotificationBody:
		return false
	default:
		return false
	}
}

// Compiled is a parsed, validated template ready to execute.
//
// Exactly one of htmlTmpl or textTmpl is set. Published versions are immutable,
// so a Compiled can be cached against a version id and reused across renders.
type Compiled struct {
	htmlTmpl *template.Template
	textTmpl *texttemplate.Template

	Channel Channel

	// Paths are the variables the template actually reads. The service records
	// them so a change to a kind's catalog can report which templates it affects.
	Paths []string

	// ContentHash identifies the source this was compiled from.
	ContentHash string
}

// Execute renders into dst under a size budget and the caller's deadline.
func (c *Compiled) Execute(
	ctx context.Context,
	dst io.Writer,
	data any,
	maxBytes int64,
) (int64, error) {
	writer := newLimitedWriter(ctx, dst, maxBytes)

	var err error
	switch {
	case c.htmlTmpl != nil:
		err = c.htmlTmpl.Execute(writer, data)
	case c.textTmpl != nil:
		err = c.textTmpl.Execute(writer, data)
	default:
		return 0, errors.New("compiled template has no executable body")
	}

	if err != nil {
		if errors.Is(err, ErrOutputTooLarge) || errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return writer.Written(), err
		}
		return writer.Written(), fmt.Errorf("execute template: %w", err)
	}

	return writer.Written(), nil
}

// Render is Execute into a string, for the channels short enough that streaming
// buys nothing: subjects, notification titles, email bodies.
func (c *Compiled) Render(ctx context.Context, data any, maxBytes int64) (string, error) {
	var b strings.Builder
	if _, err := c.Execute(ctx, &b, data, maxBytes); err != nil {
		return "", err
	}
	return b.String(), nil
}
