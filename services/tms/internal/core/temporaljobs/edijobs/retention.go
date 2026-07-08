package edijobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	PurgeEDIRawPayloadsWorkflowName = "PurgeEDIRawPayloadsWorkflow"

	ediRetentionPurgeBatchSize = 500
	secondsPerDay              = int64(24 * 60 * 60)
)

type EDIRetentionTenant struct {
	OrganizationID           pulid.ID `json:"organizationId"`
	BusinessUnitID           pulid.ID `json:"businessUnitId"`
	InboundFileRetentionDays int      `json:"inboundFileRetentionDays"`
	MessageRetentionDays     int      `json:"messageRetentionDays"`
}

func (t EDIRetentionTenant) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: t.OrganizationID,
		BuID:  t.BusinessUnitID,
	}
}

type PurgeEDIRawPayloadsTenantResult struct {
	InboundFilesPurged int64 `json:"inboundFilesPurged"`
	MessagesPurged     int64 `json:"messagesPurged"`
}

type PurgeEDIRawPayloadsResult struct {
	TenantsProcessed   int   `json:"tenantsProcessed"`
	FailedTenants      int   `json:"failedTenants"`
	InboundFilesPurged int64 `json:"inboundFilesPurged"`
	MessagesPurged     int64 `json:"messagesPurged"`
}

var retentionActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

func PurgeEDIRawPayloadsWorkflow(ctx workflow.Context) (*PurgeEDIRawPayloadsResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, retentionActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	result := new(PurgeEDIRawPayloadsResult)
	var tenants []EDIRetentionTenant
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListEDIRetentionTenantsActivity,
	).Get(activityCtx, &tenants); err != nil {
		logger.Error("failed to list EDI retention tenants", "error", err)
		return nil, err
	}

	for _, tenantItem := range tenants {
		var tenantResult PurgeEDIRawPayloadsTenantResult
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.PurgeEDIRawPayloadsTenantActivity,
			tenantItem,
		).Get(activityCtx, &tenantResult); err != nil {
			result.FailedTenants++
			logger.Error(
				"failed to purge EDI raw payloads for tenant",
				"organizationId", tenantItem.OrganizationID.String(),
				"error", err,
			)
			continue
		}
		result.TenantsProcessed++
		result.InboundFilesPurged += tenantResult.InboundFilesPurged
		result.MessagesPurged += tenantResult.MessagesPurged
	}
	return result, nil
}

func (a *Activities) ListEDIRetentionTenantsActivity(
	ctx context.Context,
) ([]EDIRetentionTenant, error) {
	retentions, err := a.dataRetentionRepo.List(ctx)
	if err != nil {
		a.logger.Error("failed to list data retention settings", zap.Error(err))
		return nil, err
	}
	tenants := make([]EDIRetentionTenant, 0, len(retentions.Items))
	for _, retention := range retentions.Items {
		if retention.EDIInboundFileRetentionPeriod <= 0 &&
			retention.EDIMessageRetentionPeriod <= 0 {
			continue
		}
		tenants = append(tenants, EDIRetentionTenant{
			OrganizationID:           retention.OrganizationID,
			BusinessUnitID:           retention.BusinessUnitID,
			InboundFileRetentionDays: retention.EDIInboundFileRetentionPeriod,
			MessageRetentionDays:     retention.EDIMessageRetentionPeriod,
		})
	}
	return tenants, nil
}

func (a *Activities) PurgeEDIRawPayloadsTenantActivity(
	ctx context.Context,
	tenantItem EDIRetentionTenant,
) (*PurgeEDIRawPayloadsTenantResult, error) {
	now := timeutils.NowUnix()
	result := new(PurgeEDIRawPayloadsTenantResult)

	if tenantItem.InboundFileRetentionDays > 0 {
		purged, err := a.purgeInBatches(ctx, repositories.PurgeEDIRawPayloadsRequest{
			TenantInfo: tenantItem.TenantInfo(),
			Before:     now - int64(tenantItem.InboundFileRetentionDays)*secondsPerDay,
			PurgedAt:   now,
			Limit:      ediRetentionPurgeBatchSize,
		}, a.inboundFileRepo.PurgeRawContentBefore)
		if err != nil {
			a.logger.Error(
				"failed to purge EDI inbound file raw content",
				zap.String("organizationId", tenantItem.OrganizationID.String()),
				zap.Error(err),
			)
			return nil, err
		}
		result.InboundFilesPurged = purged
	}

	if tenantItem.MessageRetentionDays > 0 {
		purged, err := a.purgeInBatches(ctx, repositories.PurgeEDIRawPayloadsRequest{
			TenantInfo: tenantItem.TenantInfo(),
			Before:     now - int64(tenantItem.MessageRetentionDays)*secondsPerDay,
			PurgedAt:   now,
			Limit:      ediRetentionPurgeBatchSize,
		}, a.messageRepo.PurgeRawX12Before)
		if err != nil {
			a.logger.Error(
				"failed to purge EDI message raw payloads",
				zap.String("organizationId", tenantItem.OrganizationID.String()),
				zap.Error(err),
			)
			return nil, err
		}
		result.MessagesPurged = purged
	}

	a.logger.Info(
		"EDI raw payload retention purge completed for tenant",
		zap.String("organizationId", tenantItem.OrganizationID.String()),
		zap.Int64("inboundFilesPurged", result.InboundFilesPurged),
		zap.Int64("messagesPurged", result.MessagesPurged),
	)
	return result, nil
}

func (a *Activities) purgeInBatches(
	ctx context.Context,
	req repositories.PurgeEDIRawPayloadsRequest,
	purge func(context.Context, repositories.PurgeEDIRawPayloadsRequest) (int64, error),
) (int64, error) {
	var total int64
	for {
		purged, err := purge(ctx, req)
		if err != nil {
			return total, err
		}
		total += purged
		if activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx, total)
		}
		if purged < int64(req.Limit) {
			return total, nil
		}
	}
}
