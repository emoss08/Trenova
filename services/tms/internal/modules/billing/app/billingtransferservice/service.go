package billingtransferservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingtransferports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	RunRepo   billingtransferports.BillingTransferRunRepository
	Workflows kernelports.WorkflowStarter
	Shipments shipmentserviceports.ShipmentBillingTransferService `optional:"true"`
	Logger    *zap.Logger
}

type Service struct {
	runRepo   billingtransferports.BillingTransferRunRepository
	workflows kernelports.WorkflowStarter
	shipments shipmentserviceports.ShipmentBillingTransferService
	l         *zap.Logger
}

func New(p Params) *Service {
	return &Service{
		runRepo:   p.RunRepo,
		workflows: p.Workflows,
		shipments: p.Shipments,
		l:         p.Logger.Named("billing-transfer-service"),
	}
}

type RunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
}

func (s *Service) GetRun(
	ctx context.Context,
	req *RunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	return s.runRepo.GetByID(ctx, &billingtransferports.GetBillingTransferRunRequest{
		TenantInfo: req.TenantInfo,
		RunID:      req.RunID,
	})
}

// GetActiveRun answers with nothing when the caller has no run in flight. That
// is not an error: it is how a freshly opened dialog learns it should show the
// shipment picker rather than reattach to something.
func (s *Service) GetActiveRun(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*billingtransfer.BillingTransferRun, error) {
	if tenant.UserID.IsNil() {
		return nil, errortypes.NewAuthenticationError("A user is required to read a transfer")
	}

	return s.runRepo.GetActive(ctx, &billingtransferports.GetActiveBillingTransferRunRequest{
		TenantInfo:    tenant,
		RequestedByID: tenant.UserID,
	})
}

func (s *Service) ListRunItems(
	ctx context.Context,
	req *billingtransferports.ListBillingTransferRunItemsRequest,
) (*pagination.CursorListResult[*billingtransfer.BillingTransferRunItem], error) {
	return s.runRepo.ListItems(ctx, req)
}
