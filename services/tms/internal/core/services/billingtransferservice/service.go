package billingtransferservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	RunRepo   repositories.BillingTransferRunRepository
	Workflows services.WorkflowStarter
	Logger    *zap.Logger
}

type Service struct {
	runRepo   repositories.BillingTransferRunRepository
	workflows services.WorkflowStarter
	l         *zap.Logger
}

func New(p Params) *Service {
	return &Service{
		runRepo:   p.RunRepo,
		workflows: p.Workflows,
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
	return s.runRepo.GetByID(ctx, &repositories.GetBillingTransferRunRequest{
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

	return s.runRepo.GetActive(ctx, &repositories.GetActiveBillingTransferRunRequest{
		TenantInfo:    tenant,
		RequestedByID: tenant.UserID,
	})
}

func (s *Service) ListRunItems(
	ctx context.Context,
	req *repositories.ListBillingTransferRunItemsRequest,
) (*pagination.CursorListResult[*billingtransfer.BillingTransferRunItem], error) {
	return s.runRepo.ListItems(ctx, req)
}
