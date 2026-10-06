package services

import (
	"context"
	"errors"
	"image"
)

var ErrPDFUnreadable = errors.New("the PDF could not be read")

type PDFTextLine struct {
	Text   string
	Left   float64
	Top    float64
	Width  float64
	Height float64
}

type PDFPageLayout struct {
	Width  float64
	Height float64
	Lines  []PDFTextLine
}

type PDFDocument interface {
	PageCount() int
	RenderPage(ctx context.Context, page, dpi int) (*image.RGBA, error)
	PageText(ctx context.Context, page int) (string, error)
	PageLayout(ctx context.Context, page int) (*PDFPageLayout, error)
	Close() error
}

type PDFReader interface {
	Open(ctx context.Context, data []byte) (PDFDocument, error)
}
