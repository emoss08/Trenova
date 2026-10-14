// Package pdfassembly joins, splits and rotates PDFs in memory with pdfcpu.
//
// Everything here works on bytes. pdfcpu can read and write files, and by
// default it keeps a configuration directory under the user's home; neither
// is wanted in a server that holds tenant documents, so the configuration is
// built in memory and nothing touches the disk.
package pdfassembly

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var (
	ErrNoPages      = errors.New("there are no pages to assemble")
	ErrUnreadable   = errors.New("the file is not a readable PDF")
	ErrTooManyPages = errors.New("the PDF has more pages than a capture may hold")
	ErrNoImage      = errors.New("there is no image to convert")
	ErrImageTooBig  = errors.New("the image is too large to convert")
)

const (
	MaxImageBytes = 25 << 20
	imagePageSpec = "form:Letter, pos:full, sc:0.95"
)

func init() {
	model.ConfigPath = "disable"
}

type Assembler struct{}

func New() captureports.CapturePDFAssembler {
	return &Assembler{}
}

func NewImageConverter() captureports.ImagePDFConverter {
	return &Assembler{}
}

func configuration() *model.Configuration {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	return conf
}

func (a *Assembler) PageCount(ctx context.Context, pdf []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	count, err := api.PageCount(ctx, bytes.NewReader(pdf), configuration())
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	return count, nil
}

func (a *Assembler) Assemble(
	ctx context.Context,
	pages []captureports.CaptureAssemblyPage,
) ([]byte, error) {
	if len(pages) == 0 {
		return nil, ErrNoPages
	}

	if len(pages) == 1 && capture.NormalizeRotation(pages[0].Rotation) == 0 {
		return pages[0].PDF, nil
	}

	readers := make([]io.ReadSeeker, 0, len(pages))
	for i, page := range pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		rotated, err := rotate(ctx, page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", i+1, err)
		}
		readers = append(readers, bytes.NewReader(rotated))
	}

	if len(readers) == 1 {
		return io.ReadAll(readers[0])
	}

	out := new(bytes.Buffer)
	if err := api.MergeRaw(ctx, readers, out, false, configuration()); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	return out.Bytes(), nil
}

func rotate(ctx context.Context, page captureports.CaptureAssemblyPage) ([]byte, error) {
	rotation := capture.NormalizeRotation(page.Rotation)
	if rotation == 0 {
		return page.PDF, nil
	}

	out := new(bytes.Buffer)
	if err := api.Rotate(
		ctx,
		bytes.NewReader(page.PDF),
		out,
		rotation,
		nil,
		configuration(),
	); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	return out.Bytes(), nil
}

func (a *Assembler) Split(ctx context.Context, pdf []byte) ([][]byte, error) {
	count, err := a.PageCount(ctx, pdf)
	if err != nil {
		return nil, err
	}
	if count > capture.MaxBatchPages {
		return nil, ErrTooManyPages
	}
	if count == 1 {
		return [][]byte{pdf}, nil
	}

	spans, err := api.SplitRaw(ctx, bytes.NewReader(pdf), 1, configuration())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	pages := make([][]byte, 0, len(spans))
	for _, span := range spans {
		if err = ctx.Err(); err != nil {
			return nil, err
		}

		data, readErr := io.ReadAll(span.Reader)
		if readErr != nil {
			return nil, readErr
		}
		pages = append(pages, data)
	}

	return pages, nil
}

func (a *Assembler) ImageToPDF(ctx context.Context, image []byte) ([]byte, error) {
	if len(image) == 0 {
		return nil, ErrNoImage
	}
	if len(image) > MaxImageBytes {
		return nil, ErrImageTooBig
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	imp, err := api.Import(imagePageSpec, types.POINTS)
	if err != nil {
		return nil, fmt.Errorf("configure image import: %w", err)
	}
	out := new(bytes.Buffer)
	if err = api.ImportImages(
		ctx,
		nil,
		out,
		[]io.Reader{bytes.NewReader(image)},
		imp,
		configuration(),
	); err != nil {
		return nil, fmt.Errorf("convert image to PDF: %w", err)
	}

	return out.Bytes(), nil
}
