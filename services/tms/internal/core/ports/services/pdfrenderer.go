package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
)

// ErrPDFRendererUnavailable is returned when no rendering backend is configured
// or the configured one is unreachable. Callers turn this into a business error
// rather than a 500 — a preview that cannot render is not a server fault.
var ErrPDFRendererUnavailable = errors.New("pdf renderer unavailable")

// PDFRenderRequest is one HTML document to print.
//
// HeaderHTML and FooterHTML are separate documents rather than markup inside
// HTML because the rendering backend repeats them on every page and exposes its
// own page-number substitutions there.
type PDFRenderRequest struct {
	HTML        string
	HeaderHTML  string
	FooterHTML  string
	Title       string
	TraceID     string
	PageSize    documenttemplate.PageSize
	Orientation documenttemplate.Orientation
	Margins     documenttemplate.Margins
}

type PDFRenderer interface {
	// Render prints HTML to PDF bytes. It returns a wrapped
	// ErrPDFRendererUnavailable when the backend is not configured or not
	// reachable, so callers can distinguish "no renderer" from "bad template".
	Render(ctx context.Context, req *PDFRenderRequest) ([]byte, error)

	// Healthy reports whether the backend can currently accept work.
	Healthy(ctx context.Context) error

	// Enabled reports whether a backend is configured at all, so callers can
	// skip work instead of building HTML that has nowhere to go.
	Enabled() bool
}
