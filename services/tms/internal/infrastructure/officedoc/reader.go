package officedoc

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const (
	maxArchiveEntries = 20_000
	maxPartBytes      = 64 << 20
	maxTotalBytes     = 256 << 20
	maxTextBytes      = 8 << 20
	maxImageBytes     = 20 << 20
	maxImagesPerPage  = 16
	maxPages          = 5_000
)

var errUnsupportedFormat = errors.New("unsupported office format")

type Reader struct{}

var _ kernelports.OfficeDocumentReader = (*Reader)(nil)

func New() kernelports.OfficeDocumentReader {
	return &Reader{}
}

func (r *Reader) Read(
	ctx context.Context,
	data []byte,
	format kernelports.OfficeFormat,
) (pages []kernelports.OfficePage, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			pages = nil
			err = fmt.Errorf("read %s: %w: parser panic: %v",
				format, kernelports.ErrOfficeDocumentUnreadable, recovered)
		}
	}()

	pages, err = r.read(ctx, data, format)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("read %s: %w", format, ctxErr)
		}
		return nil, fmt.Errorf("read %s: %w: %w", format, kernelports.ErrOfficeDocumentUnreadable, err)
	}

	return pages, nil
}

func (r *Reader) read(
	ctx context.Context,
	data []byte,
	format kernelports.OfficeFormat,
) ([]kernelports.OfficePage, error) {
	switch format {
	case kernelports.OfficeFormatXLSX:
		return readXLSX(ctx, data)
	case kernelports.OfficeFormatDOCX:
		return withArchive(ctx, data, readDOCX)
	case kernelports.OfficeFormatPPTX:
		return withArchive(ctx, data, readPPTX)
	case kernelports.OfficeFormatEPUB:
		return withArchive(ctx, data, readEPUB)
	default:
		return nil, fmt.Errorf("%w: %q", errUnsupportedFormat, format)
	}
}

func withArchive(
	ctx context.Context,
	data []byte,
	read func(context.Context, *archive) ([]kernelports.OfficePage, error),
) ([]kernelports.OfficePage, error) {
	pkg, err := openArchive(data)
	if err != nil {
		return nil, err
	}
	return read(ctx, pkg)
}
