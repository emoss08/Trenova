package officedoc

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
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

var _ services.OfficeDocumentReader = (*Reader)(nil)

func New() services.OfficeDocumentReader {
	return &Reader{}
}

func (r *Reader) Read(
	ctx context.Context,
	data []byte,
	format services.OfficeFormat,
) (pages []services.OfficePage, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			pages = nil
			err = fmt.Errorf("read %s: %w: parser panic: %v",
				format, services.ErrOfficeDocumentUnreadable, recovered)
		}
	}()

	pages, err = r.read(ctx, data, format)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("read %s: %w", format, ctxErr)
		}
		return nil, fmt.Errorf("read %s: %w: %w", format, services.ErrOfficeDocumentUnreadable, err)
	}

	return pages, nil
}

func (r *Reader) read(
	ctx context.Context,
	data []byte,
	format services.OfficeFormat,
) ([]services.OfficePage, error) {
	switch format {
	case services.OfficeFormatXLSX:
		return readXLSX(ctx, data)
	case services.OfficeFormatDOCX:
		return withArchive(ctx, data, readDOCX)
	case services.OfficeFormatPPTX:
		return withArchive(ctx, data, readPPTX)
	case services.OfficeFormatEPUB:
		return withArchive(ctx, data, readEPUB)
	default:
		return nil, fmt.Errorf("%w: %q", errUnsupportedFormat, format)
	}
}

func withArchive(
	ctx context.Context,
	data []byte,
	read func(context.Context, *archive) ([]services.OfficePage, error),
) ([]services.OfficePage, error) {
	pkg, err := openArchive(data)
	if err != nil {
		return nil, err
	}
	return read(ctx, pkg)
}
