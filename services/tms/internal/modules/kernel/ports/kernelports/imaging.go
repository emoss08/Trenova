package kernelports

import (
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
)

var (
	ErrImagingInvalidDocument = errors.New("the document is not readable")
	ErrImagingEncrypted       = errors.New("the document is encrypted")
	ErrImagingTooLarge        = errors.New("the document is too large")
	ErrImagingPixelBudget     = errors.New("the image is over the pixel budget")
	ErrImagingUnsupported     = errors.New("the format is not supported")
	ErrImagingPageOutOfRange  = errors.New("the page is not in the document")
	ErrImagingTimeout         = errors.New("the imaging work did not finish in time")
	ErrImagingUnavailable     = errors.New("the imaging service is unavailable")
	ErrImagingInternal        = errors.New("the imaging service failed")
)

type ImagingPriority int

const (
	ImagingBatch ImagingPriority = iota
	ImagingInteractive
)

type ImagingFormat string

const (
	ImagingWebP ImagingFormat = "webp"
	ImagingJPEG ImagingFormat = "jpeg"
	ImagingPNG  ImagingFormat = "png"
)

type ImagingSource struct {
	ContentType string
	Size        int64
	Open        func(ctx context.Context) (io.ReadCloser, error)
}

func ImagingBytes(contentType string, data []byte) ImagingSource {
	return ImagingSource{
		ContentType: contentType,
		Size:        int64(len(data)),
		Open: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		},
	}
}

type ImagingPageError struct {
	Reason  error
	Message string
}

func (e *ImagingPageError) Error() string {
	return e.Reason.Error() + ": " + e.Message
}

func (e *ImagingPageError) Unwrap() error {
	return e.Reason
}

type ImagingArtifact struct {
	Index       int
	ContentType string
	Width       int
	Height      int
	Size        int64
	Body        io.Reader
	Err         *ImagingPageError
}

type ImagingPageInfo struct {
	Index        int
	WidthPoints  float64
	HeightPoints float64
	Rotation     int
	HasText      bool
	CharCount    int
}

type ImagingDocumentKind string

const (
	ImagingKindPDF   ImagingDocumentKind = "pdf"
	ImagingKindImage ImagingDocumentKind = "image"
)

type ImagingProbeRequest struct {
	Source      ImagingSource
	TextProfile bool
	Priority    ImagingPriority
}

type ImagingProbe struct {
	Kind        ImagingDocumentKind
	PageCount   int
	Pages       []ImagingPageInfo
	Encrypted   bool
	PDFVersion  string
	Producer    string
	ImageFormat string
	Orientation int
}

type ImagingRenderRequest struct {
	Source    ImagingSource
	Pages     []int
	DPI       int
	FitWidth  int
	FitHeight int
	Format    ImagingFormat
	Quality   int
	Lossless  bool
	MaxPages  int
	Priority  ImagingPriority
}

type ImagingRenderSummary struct {
	PageCount int
	Rendered  int
}

type ImagingCapturePage struct {
	Index  int
	Source ImagingSource
}

type ImagingCaptureRequest struct {
	PageCount int
	Pages     func() iter.Seq2[ImagingCapturePage, error]
	Priority  ImagingPriority
}

type ImagingBarcode struct {
	Format string
	Text   string
}

type ImagingCaptureResult struct {
	Index                int
	Width                int
	Height               int
	Thumbnail            []byte
	ThumbnailContentType string
	InkCoverage          float64
	Codes                []ImagingBarcode
	Err                  *ImagingPageError
}

type ImagingCaptureSummary struct {
	Inspected int
	Failed    int
}

type ImagingPageRange struct {
	Source   int
	First    int
	Last     int
	Rotation int
}

type ImagingAssembleRequest struct {
	Sources  []ImagingSource
	Outputs  [][]ImagingPageRange
	Priority ImagingPriority
}

type ImagingNormalizeRequest struct {
	Source    ImagingSource
	Format    ImagingFormat
	Quality   int
	MaxWidth  int
	MaxHeight int
	Priority  ImagingPriority
}

type ImagingNormalizeSummary struct {
	SourceFormat       string
	SourceWidth        int
	SourceHeight       int
	OrientationApplied int
}

type Imaging interface {
	Probe(ctx context.Context, req *ImagingProbeRequest) (*ImagingProbe, error)
	Render(
		ctx context.Context,
		req *ImagingRenderRequest,
		yield func(*ImagingArtifact) error,
	) (*ImagingRenderSummary, error)
	InspectCapture(
		ctx context.Context,
		req *ImagingCaptureRequest,
		yield func(*ImagingCaptureResult) error,
	) (*ImagingCaptureSummary, error)
	Assemble(
		ctx context.Context,
		req *ImagingAssembleRequest,
		yield func(*ImagingArtifact) error,
	) ([]int, error)
	NormalizeImage(
		ctx context.Context,
		req *ImagingNormalizeRequest,
		yield func(*ImagingArtifact) error,
	) (*ImagingNormalizeSummary, error)
}
