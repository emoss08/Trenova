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

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

var (
	ErrNoPages      = errors.New("there are no pages to assemble")
	ErrUnreadable   = errors.New("the file is not a readable PDF")
	ErrTooManyPages = errors.New("the PDF has more pages than a capture may hold")
)

func init() {
	model.ConfigPath = "disable"
}

type Assembler struct{}

func New() services.CapturePDFAssembler {
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

	count, err := api.PageCount(bytes.NewReader(pdf), configuration())
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	return count, nil
}

func (a *Assembler) Assemble(
	ctx context.Context,
	pages []services.CaptureAssemblyPage,
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

		rotated, err := rotate(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", i+1, err)
		}
		readers = append(readers, bytes.NewReader(rotated))
	}

	if len(readers) == 1 {
		return io.ReadAll(readers[0])
	}

	out := new(bytes.Buffer)
	if err := api.MergeRaw(readers, out, false, configuration()); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	return out.Bytes(), nil
}

func rotate(page services.CaptureAssemblyPage) ([]byte, error) {
	rotation := capture.NormalizeRotation(page.Rotation)
	if rotation == 0 {
		return page.PDF, nil
	}

	out := new(bytes.Buffer)
	if err := api.Rotate(
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

	spans, err := api.SplitRaw(bytes.NewReader(pdf), 1, configuration())
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
