package auditjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/app/auditalertservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditexportservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditintegrity"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditretention"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditreviewservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Retention *auditretention.Retention
	Sealer    *auditintegrity.Sealer
	Verifier  *auditintegrity.Verifier
	Exports   *auditexportservice.Service
	Alerts    *auditalertservice.Service
	Reviews   *auditreviewservice.Service
}

type Activities struct {
	retention *auditretention.Retention
	sealer    *auditintegrity.Sealer
	verifier  *auditintegrity.Verifier
	exports   *auditexportservice.Service
	alerts    *auditalertservice.Service
	reviews   *auditreviewservice.Service
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		retention: p.Retention,
		sealer:    p.Sealer,
		verifier:  p.Verifier,
		exports:   p.Exports,
		alerts:    p.Alerts,
		reviews:   p.Reviews,
	}
}

func heartbeat(ctx context.Context) func(string) {
	return func(detail string) { activity.RecordHeartbeat(ctx, detail) }
}

func retryable(message string, err error) error {
	if err == nil {
		return nil
	}
	return temporaltype.NewRetryableError(message, err).ToTemporalError()
}

func (a *Activities) SweepRetentionActivity(
	ctx context.Context,
) (*auditretention.SweepResult, error) {
	result, err := a.retention.Sweep(ctx, heartbeat(ctx))
	return result, retryable("Failed to apply audit retention", err)
}

func (a *Activities) CheckPartitionsActivity(
	ctx context.Context,
) (*auditretention.PartitionResult, error) {
	result, err := a.retention.CheckPartitions(ctx)
	return result, retryable("Failed to check audit partitions", err)
}

func (a *Activities) SealActivity(ctx context.Context) (*SealResult, error) {
	result, err := a.sealer.SealAll(ctx, heartbeat(ctx))
	if err != nil {
		return nil, retryable("Failed to seal audit entries", err)
	}
	return sealResult(result), nil
}

func (a *Activities) VerifyActivity(ctx context.Context) (*VerifyResult, error) {
	result, err := a.verifier.VerifyAll(ctx, heartbeat(ctx))
	if err != nil {
		return nil, retryable("Failed to verify the audit chain", err)
	}
	return verifyResult(result), nil
}

func (a *Activities) RunExportActivity(
	ctx context.Context,
	payload *auditexportservice.ExportPayload,
) (*ExportResult, error) {
	tenant := pagination.TenantInfo{OrgID: payload.OrganizationID, BuID: payload.BusinessUnitID}
	ctx = dbscope.WithValidTenant(ctx, dbscope.Tenant{
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
	})
	export, err := a.exports.Run(ctx, tenant, payload.ExportID, heartbeat(ctx))
	if err != nil {
		return nil, retryable("Failed to write the audit export", err)
	}
	return &ExportResult{Status: string(export.Status), RowCount: export.RowCount}, nil
}

func (a *Activities) ExpireExportsActivity(ctx context.Context) (int, error) {
	expired, err := a.exports.ExpireDue(ctx)
	return expired, retryable("Failed to expire audit exports", err)
}

func (a *Activities) RecordReviewActivity(
	ctx context.Context,
	payload *auditports.AlertPayload,
) error {
	ctx = dbscope.WithValidTenant(ctx, dbscope.Tenant{
		OrganizationID: payload.TenantInfo.OrgID,
		BusinessUnitID: payload.TenantInfo.BuID,
	})
	return retryable(
		"Failed to record the security alert for review",
		a.reviews.Raise(ctx, payload),
	)
}

func (a *Activities) NotifyAlertActivity(
	ctx context.Context,
	payload *auditports.AlertPayload,
) (int, error) {
	ctx = dbscope.WithValidTenant(ctx, dbscope.Tenant{
		OrganizationID: payload.TenantInfo.OrgID,
		BusinessUnitID: payload.TenantInfo.BuID,
	})
	sent, err := a.alerts.Notify(ctx, payload)
	return sent, retryable("Failed to notify security reviewers", err)
}
