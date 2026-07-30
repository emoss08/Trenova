package detentionjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	DetentionService    services.DetentionNoticeService
	ShipmentControlRepo repositories.ShipmentControlRepository
	Logger              *zap.Logger
}

type Activities struct {
	detentionService    services.DetentionNoticeService
	shipmentControlRepo repositories.ShipmentControlRepository
	logger              *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		detentionService:    p.DetentionService,
		shipmentControlRepo: p.ShipmentControlRepo,
		logger:              p.Logger.Named("temporal.detention"),
	}
}

func (a *Activities) ListDetentionTenantsActivity(
	ctx context.Context,
	payload *ListDetentionTenantsPayload,
) (*ListDetentionTenantsResult, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, temporaljobs.DefaultTenantScanLimit)

	tenants, err := a.shipmentControlRepo.ListDetentionEngineTenants(ctx, limit)
	if err != nil {
		return nil, err
	}

	return &ListDetentionTenantsResult{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, 1),
	}, nil
}

func (a *Activities) SweepTenantNoticesActivity(
	ctx context.Context,
	payload *SweepTenantNoticesPayload,
) (*SweepTenantNoticesResult, error) {
	tenantInfo := payload.TenantInfo()
	recordActivityHeartbeat(ctx, "sweeping-detention-notices", tenantInfo.OrgID.String())

	result, err := a.detentionService.SweepNoticesDue(ctx, tenantInfo)
	if err != nil {
		a.logger.Error("Failed to sweep detention notices for tenant",
			zap.String("orgID", tenantInfo.OrgID.String()),
			zap.String("buID", tenantInfo.BuID.String()),
			zap.Error(err))
		return nil, err
	}

	return &SweepTenantNoticesResult{
		Due:     result.Due,
		Sent:    result.Sent,
		Skipped: result.Skipped,
		Failed:  result.Failed,
	}, nil
}

// AttachNoticePDFDocumentActivity points a notice at the document its PDF became
// and logs the render against the template version behind it.
//
// The document id arrives from the finalize child workflow rather than being
// looked up, so this activity cannot file the wrong document if a newer version of
// the same notice is uploaded while it retries.
func (a *Activities) AttachNoticePDFDocumentActivity(
	ctx context.Context,
	payload *StoreNoticePDFPayload,
	documentID pulid.ID,
) (*StoreNoticePDFResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	if err := a.detentionService.AttachNoticePDFDocument(
		ctx,
		&services.AttachNoticePDFDocumentParams{
			TenantInfo:   tenantInfo,
			NoticeID:     payload.NoticeID,
			OccurrenceID: payload.OccurrenceID,
			DocumentID:   documentID,
			FileName:     payload.FileName,
			FileSize:     payload.FileSize,
			Provenance:   payload.Provenance,
			UserID:       payload.UserID,
		},
	); err != nil {
		a.logger.Error("failed to file detention notice PDF",
			zap.String("noticeId", payload.NoticeID.String()),
			zap.String("documentId", documentID.String()),
			zap.Error(err))
		return nil, err
	}

	return &StoreNoticePDFResult{
		NoticeID:    payload.NoticeID,
		DocumentID:  &documentID,
		CompletedAt: timeutils.NowUnix(),
	}, nil
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	if activity.IsActivity(ctx) {
		activity.RecordHeartbeat(ctx, details...)
	}
}
