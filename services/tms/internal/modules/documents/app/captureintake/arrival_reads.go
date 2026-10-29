package captureintake

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

// inspectOnArrival reads a page as soon as it is stored, so its thumbnail is
// there while the rest of the batch is still arriving. It is best effort: a
// page not read now, because the readers are busy or the read failed, is
// read when the batch is sealed.
func (s *Service) inspectOnArrival(tenantInfo pagination.TenantInfo, page *capture.CapturePage) {
	arrived := *page
	s.arrivals.submit(func(ctx context.Context) {
		ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())
		pages := []*capture.CapturePage{&arrived}
		if err := s.processing.ReadPages(ctx, tenantInfo, pages, []int{0}, kernelports.ImagingBatch,
			func(int) {}); err != nil {
			s.l.Debug("a page was not read on arrival", zap.Error(err))
			return
		}
		if arrived.Status != capture.PageProcessed {
			return
		}
		if _, err := s.pages.RecordInspection(ctx, &arrived); err != nil {
			s.l.Warn("could not record a page read on arrival",
				zap.String("pageId", arrived.ID.String()), zap.Error(err))
		}
	})
}
