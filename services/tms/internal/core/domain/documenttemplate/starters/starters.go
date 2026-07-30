// Package starters holds the built-in templates every organization starts from.
//
// The assets are real .html, .css, and .txt files rather than Go string literals.
// A 200-line invoice layout written as a quoted string is unreviewable in a diff,
// unformattable by any tool, and impossible to open in an editor that understands
// HTML. The EDI designer keeps its starters in Go, but those are structured segment
// builders, not documents.
//
// Because a BuiltIn resolution reads the embedded asset at render time, shipping a
// new binary updates every organization that has not customized a kind — no
// migration, no seed, and no way to clobber an organization that has.
package starters

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
)

//go:embed assets/*
var assetFS embed.FS

const assetDir = "assets"

// File extensions, one per channel. The stem is the kind, so a kind and its assets
// cannot drift apart without a missing-file failure.
const (
	extHTML    = ".html"
	extCSS     = ".css"
	extText    = ".txt"
	extSubject = ".subject"
)

// Starter is the built-in content for one kind.
type Starter struct {
	Kind documenttemplate.Kind

	// BodyHTML is the PDF body or the HTML email body.
	BodyHTML string
	// CSS dresses a PDF body. Empty for message kinds, where mail clients strip
	// stylesheets and the starters use inline styles instead.
	CSS string
	// BodyText is the plain-text alternative for a message.
	BodyText string
	// Subject is the message subject line.
	Subject string

	// PageSize, Orientation, and Margins are the page setup a document starts with.
	PageSize    documenttemplate.PageSize
	Orientation documenttemplate.Orientation
	Margins     documenttemplate.Margins
}

// ContentHash identifies the starter's content.
//
// A seeded draft records it, so a later deploy can tell an administrator that the
// built-in has changed since they started from it.
func (s *Starter) ContentHash() string {
	// A separator between fields so moving content from one channel to another
	// changes the hash rather than producing the same concatenation.
	joined := strings.Join([]string{s.BodyHTML, s.CSS, s.BodyText, s.Subject}, "\x00")
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:])
}

// Channel returns the source for one channel.
func (s *Starter) Channel(channel documenttemplate.Channel) string {
	switch channel {
	case documenttemplate.ChannelPDF, documenttemplate.ChannelEmailHTML:
		return s.BodyHTML
	case documenttemplate.ChannelEmailText:
		return s.BodyText
	case documenttemplate.ChannelSubject:
		return s.Subject
	case documenttemplate.ChannelNotificationTitle:
		return s.Subject
	case documenttemplate.ChannelNotificationBody:
		return s.BodyText
	default:
		return ""
	}
}

// reportMarginsMillimeters is the 0.4in box the report layout has always used.
const reportMarginMillimeters = 10.16

// For returns the built-in starter for a kind.
//
// A missing asset is an error rather than an empty template: rendering an invoice
// from nothing would produce a blank page and send it to a customer.
func For(kind documenttemplate.Kind) (*Starter, error) {
	registry := documenttemplate.NewRegistry()
	def, ok := registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown template kind %q", kind)
	}

	starter := &Starter{
		Kind:        kind,
		PageSize:    documenttemplate.PageSizeLetter,
		Orientation: documenttemplate.OrientationPortrait,
		Margins:     documenttemplate.DefaultMargins(),
	}

	stem := def.StarterKey()

	// Only a kind with an HTML channel has an HTML asset. An in-app notification
	// is a title and a body of text; the client renders both into its own layout,
	// and markup would show up as characters.
	if def.HasChannel(documenttemplate.ChannelPDF) ||
		def.HasChannel(documenttemplate.ChannelEmailHTML) {
		body, bodyErr := readAsset(stem + extHTML)
		if bodyErr != nil {
			return nil, bodyErr
		}
		starter.BodyHTML = body
	}

	if def.HasChannel(documenttemplate.ChannelPDF) {
		css, cssErr := readAsset(stem + extCSS)
		if cssErr != nil {
			return nil, cssErr
		}
		starter.CSS = css
	}

	if def.HasChannel(documenttemplate.ChannelEmailText) ||
		def.HasChannel(documenttemplate.ChannelNotificationBody) {
		text, textErr := readAsset(stem + extText)
		if textErr != nil {
			return nil, textErr
		}
		starter.BodyText = text
	}

	if def.HasChannel(documenttemplate.ChannelSubject) ||
		def.HasChannel(documenttemplate.ChannelNotificationTitle) {
		subject, subjectErr := readAsset(stem + extSubject)
		if subjectErr != nil {
			return nil, subjectErr
		}
		// A subject is one line. Trailing newlines from the file would otherwise
		// travel into a mail header.
		starter.Subject = strings.TrimSpace(subject)
	}

	applyPageSetup(kind, starter)

	return starter, nil
}

// applyPageSetup overrides the defaults for kinds whose layout needs it.
func applyPageSetup(kind documenttemplate.Kind, starter *Starter) {
	if kind != documenttemplate.KindReportPDF {
		return
	}

	// Reports are wide, and a report squeezed into portrait loses its rightmost
	// columns to the page edge.
	starter.Orientation = documenttemplate.OrientationLandscape
	starter.Margins = documenttemplate.Margins{
		Top:    reportMarginMillimeters,
		Bottom: reportMarginMillimeters,
		Left:   reportMarginMillimeters,
		Right:  reportMarginMillimeters,
	}
}

// All returns every built-in starter, ordered by kind.
func All() ([]*Starter, error) {
	kinds := documenttemplate.AllKinds()
	out := make([]*Starter, 0, len(kinds))

	for _, kind := range kinds {
		starter, err := For(kind)
		if err != nil {
			return nil, err
		}
		out = append(out, starter)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })

	return out, nil
}

// Hash returns the content hash for a kind's starter.
func Hash(kind documenttemplate.Kind) (string, error) {
	starter, err := For(kind)
	if err != nil {
		return "", err
	}
	return starter.ContentHash(), nil
}

func readAsset(name string) (string, error) {
	body, err := assetFS.ReadFile(path.Join(assetDir, name))
	if err != nil {
		return "", fmt.Errorf("read built-in template asset %q: %w", name, err)
	}
	return string(body), nil
}

// AssetNames lists the embedded files, for the test that asserts nothing is
// orphaned. A stale asset is dead weight; a missing one breaks a kind.
func AssetNames() ([]string, error) {
	entries, err := fs.ReadDir(assetFS, assetDir)
	if err != nil {
		return nil, fmt.Errorf("list built-in template assets: %w", err)
	}

	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		out = append(out, entry.Name())
	}
	sort.Strings(out)

	return out, nil
}
