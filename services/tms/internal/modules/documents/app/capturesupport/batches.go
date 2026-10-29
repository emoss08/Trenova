package capturesupport

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	BatchActionCreated  = "created"
	BatchActionUpdated  = "updated"
	BatchActionReceived = "page.received"
)

// publishBatch tells the intake queue a batch changed. It carries no entity:
// who may see a batch depends on their data scope, which the queue's own
// query applies when it refetches.
func (s *Service) PublishBatch(ctx context.Context, batch *capture.CaptureBatch, action string) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  batch.OrganizationID,
		BuID:   batch.BusinessUnitID,
		UserID: batch.UserID,
	}
	s.Publish(ctx, tenantInfo, permission.ResourceCaptureBatch, batch.ID, action, pulid.Nil, nil)
}
