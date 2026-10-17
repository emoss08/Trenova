package kernelports

import (
	"context"
	"errors"
	"io"
)

var (
	ErrDocumentConverterUnavailable   = errors.New("document converter unavailable")
	ErrDocumentConverterMisconfigured = errors.New("document converter refused its credentials")
	ErrConversionTaskNotFound         = errors.New("conversion task not found")
	ErrDocumentNotConvertible         = errors.New("document could not be converted")
)

type ConversionTaskID string

type ConversionState string

const (
	ConversionStatePending   ConversionState = "pending"
	ConversionStateRunning   ConversionState = "running"
	ConversionStateSucceeded ConversionState = "succeeded"
	ConversionStateFailed    ConversionState = "failed"
)

type ConversionStatus struct {
	State   ConversionState
	Message string
}

func (s ConversionStatus) Done() bool {
	return s.State == ConversionStateSucceeded || s.State == ConversionStateFailed
}

type DocumentConverter interface {
	Enabled() bool
	Healthy(ctx context.Context) error
	Submit(ctx context.Context, req *ConvertRequest) (ConversionTaskID, error)
	Status(ctx context.Context, id ConversionTaskID) (ConversionStatus, error)
	Result(ctx context.Context, id ConversionTaskID) (*ConvertedDocument, error)
}

type ConvertRequest struct {
	FileName    string
	ContentType string
	Body        io.Reader
	Size        int64
	OCR         bool
	Languages   []string
}

type ConvertedDocument struct {
	Converter        string
	ConverterVersion string
	Pages            []ConvertedPage
	Raw              []byte
}

type ConvertedPage struct {
	Number        int
	OCR           bool
	Native        bool
	OCRConfidence float64
	Width         float64
	Height        float64
	Text          string
	Markdown      string
	Lines         []ConvertedLine
}

type ConvertedLine struct {
	Text       string
	X, Y, W, H float64
}
