package aiauditjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/aiauditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

const errTypeExportNotFound = "AIAuditExportNotFound"

type ProjectResult struct {
	Inserted int `json:"inserted"`
	Tenants  int `json:"tenants"`
	Read     int `json:"read"`
}

type TenantVerification struct {
	OrganizationID string                     `json:"organizationId"`
	BusinessUnitID string                     `json:"businessUnitId"`
	Status         aiaudit.VerificationStatus `json:"status"`
	VerifiedSeq    int64                      `json:"verifiedSeq"`
	FailedSeq      *int64                     `json:"failedSeq,omitempty"`
	Rows           int64                      `json:"rows"`
}

type VerifyResult struct {
	Tenants []*TenantVerification `json:"tenants"`
	Errors  int                   `json:"errors"`
}

type PruneResult = aiauditservice.PruneResult

type CleanupResult = aiauditservice.CleanupResult

type ExportResult struct {
	ExportID string               `json:"exportId"`
	Status   aiaudit.ExportStatus `json:"status"`
	RowCount int64                `json:"rowCount"`
}

// heartbeat reports progress to Temporal from inside an activity.
func heartbeat(ctx context.Context) aiauditservice.Heartbeat {
	return func(details ...any) {
		activity.RecordHeartbeat(ctx, details...)
	}
}

type ActivitiesParams struct {
	fx.In

	Projector *aiauditservice.Projector
	Verifier  *aiauditservice.Verifier
	Retention *aiauditservice.Retention
	Exports   *aiauditservice.Exports
}

// Activities run on the audit queue: the projector, verification, retention
// and export cleanup.
type Activities struct {
	projector *aiauditservice.Projector
	verifier  *aiauditservice.Verifier
	retention *aiauditservice.Retention
	exports   *aiauditservice.Exports
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		projector: p.Projector,
		verifier:  p.Verifier,
		retention: p.Retention,
		exports:   p.Exports,
	}
}

func (a *Activities) ProjectAIAuditActivity(ctx context.Context) (*ProjectResult, error) {
	result, err := a.projector.RunOnce(ctx, heartbeat(ctx))
	if err != nil {
		return nil, err
	}

	read := 0
	for _, count := range result.Read {
		read += count
	}

	return &ProjectResult{Inserted: result.Inserted, Tenants: result.Tenants, Read: read}, nil
}

func (a *Activities) ListAIAuditTenantsActivity(
	ctx context.Context,
) ([]pagination.TenantInfo, error) {
	return a.verifier.Tenants(ctx)
}

func (a *Activities) VerifyAIAuditTenantActivity(
	ctx context.Context,
	tenantInfo *pagination.TenantInfo,
) (*TenantVerification, error) {
	if tenantInfo == nil || tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return nil, temporal.NewNonRetryableApplicationError(
			"a verification needs a tenant", "InvalidInput", nil,
		)
	}

	result, err := a.verifier.VerifyTenant(ctx, *tenantInfo, heartbeat(ctx))
	if err != nil {
		return nil, err
	}

	return &TenantVerification{
		OrganizationID: tenantInfo.OrgID.String(),
		BusinessUnitID: tenantInfo.BuID.String(),
		Status:         result.Status,
		VerifiedSeq:    result.VerifiedSeq,
		FailedSeq:      result.FailedSeq,
		Rows:           result.Rows,
	}, nil
}

func (a *Activities) PruneAIAuditActivity(ctx context.Context) (*PruneResult, error) {
	return a.retention.PruneAll(ctx, heartbeat(ctx))
}

func (a *Activities) CleanupAIAuditExportsActivity(ctx context.Context) (*CleanupResult, error) {
	return a.exports.Cleanup(ctx, heartbeat(ctx))
}

type ExportActivitiesParams struct {
	fx.In

	Exports *aiauditservice.Exports
}

// ExportActivities run on the report queue, beside the report renderers.
type ExportActivities struct {
	exports *aiauditservice.Exports
}

func NewExportActivities(p ExportActivitiesParams) *ExportActivities {
	return &ExportActivities{exports: p.Exports}
}

func (a *ExportActivities) RunAIAuditExportActivity(
	ctx context.Context,
	payload *serviceports.AIAuditExportPayload,
) (*ExportResult, error) {
	export, err := a.exports.Run(ctx, payload.TenantInfo(), payload.ExportID, heartbeat(ctx))
	if err != nil {
		var notFound *errortypes.NotFoundError
		if errors.As(err, &notFound) {
			return nil, temporal.NewNonRetryableApplicationError(
				err.Error(), errTypeExportNotFound, err,
			)
		}

		return nil, err
	}

	return &ExportResult{
		ExportID: export.ID.String(),
		Status:   export.Status,
		RowCount: export.RowCount,
	}, nil
}
