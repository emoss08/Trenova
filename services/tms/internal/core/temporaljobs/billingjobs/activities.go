package billingjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	InvoiceService services.InvoiceService
	Logger         *zap.Logger
}

type Activities struct {
	invoiceService services.InvoiceService
	logger         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		invoiceService: p.InvoiceService,
		logger:         p.Logger.Named("billing-activities"),
	}
}

func (a *Activities) AutoPostInvoiceActivity(
	ctx context.Context,
	payload *AutoPostInvoicePayload,
) (*AutoPostInvoiceResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	current, err := a.invoiceService.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         payload.InvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if current.Status == invoice.StatusPosted {
		return &AutoPostInvoiceResult{
			InvoiceID:     current.ID,
			PostedAt:      derefInt64(current.PostedAt),
			CompletedAt:   timeutils.NowUnix(),
			AlreadyPosted: true,
		}, nil
	}

	posted, err := a.invoiceService.Post(ctx, &services.PostInvoiceRequest{
		InvoiceID:   payload.InvoiceID,
		TenantInfo:  tenantInfo,
		TriggeredBy: "auto-post-workflow",
	}, &services.RequestActor{
		PrincipalType:  payload.PrincipalType,
		PrincipalID:    payload.PrincipalID,
		UserID:         payload.UserID,
		APIKeyID:       payload.APIKeyID,
		BusinessUnitID: payload.BusinessUnitID,
		OrganizationID: payload.OrganizationID,
	})
	if err != nil {
		return nil, err
	}

	return &AutoPostInvoiceResult{
		InvoiceID:   posted.ID,
		PostedAt:    derefInt64(posted.PostedAt),
		CompletedAt: timeutils.NowUnix(),
	}, nil
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}

	return *value
}
