package captureprocessing

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturecoversheet"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// coverSheetByID reads a cover sheet, reporting whether it still exists.
func (s *Service) coverSheetByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*capturecoversheet.CaptureCoverSheet, bool, error) {
	sheet, err := s.coverSheets.GetByID(ctx, captureports.GetCaptureCoverSheetByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return sheet, true, nil
}

const maxAssembledBytes = 256 << 20

// assembleItem joins an item's pages, rotated as a person left them, into
// one PDF.
func (s *Service) AssembleItem(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *capture.CaptureItem,
	pages []*capture.CapturePage,
) ([]byte, error) {
	byID := make(map[pulid.ID]*capture.CapturePage, len(pages))
	for _, page := range pages {
		byID[page.ID] = page
	}

	parts := make([]imagingkit.Part, 0, len(item.PageIDs))
	for _, id := range item.PageIDs {
		page, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("page %s is not in the batch", id)
		}

		data, err := s.support.GetObject(ctx, tenantInfo, page.StoragePath)
		if err != nil {
			return nil, err
		}
		parts = append(parts, imagingkit.Part{
			Source:   kernelports.ImagingBytes("application/pdf", data),
			Rotation: capture.NormalizeRotation(page.Rotation),
		})
	}

	assembled, _, err := imagingkit.Merge(
		ctx,
		s.imaging,
		parts,
		maxAssembledBytes,
		kernelports.ImagingBatch,
	)
	return assembled, err
}
