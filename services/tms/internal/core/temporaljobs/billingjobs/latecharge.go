package billingjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	LateChargeAssessmentWorkflowName = "LateChargeAssessmentWorkflow"
	lateChargeTenantBatch            = 1000
)

// LateChargeAssessmentResult is what one nightly run did across tenants.
type LateChargeAssessmentResult struct {
	TenantsDue       int   `json:"tenantsDue"`
	TenantsFailed    int   `json:"tenantsFailed"`
	MemosCreated     int   `json:"memosCreated"`
	MemosPosted      int   `json:"memosPosted"`
	CustomersSkipped int   `json:"customersSkipped"`
	TotalChargeMinor int64 `json:"totalChargeMinor"`
	CompletedAt      int64 `json:"completedAt"`
}

// LateChargeTenantResult is what one tenant's assessment produced.
type LateChargeTenantResult struct {
	OrganizationID   string `json:"organizationId"`
	BusinessUnitID   string `json:"businessUnitId"`
	Preview          bool   `json:"preview"`
	MemosCreated     int    `json:"memosCreated"`
	MemosPosted      int    `json:"memosPosted"`
	CustomersSkipped int    `json:"customersSkipped"`
	TotalChargeMinor int64  `json:"totalChargeMinor"`
}

// A tenant whose assessment fails is retried a few times, then left for the
// next night: the unique (invoice, period) key means a partial run charges
// nothing twice when it is picked up again.
var lateChargeRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    10 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    2 * time.Minute,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
		temporaltype.ErrorTypeNonRetryable.String(),
		temporaltype.ErrorTypePermissionDenied.String(),
	},
}

var lateChargeActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy:         lateChargeRetryPolicy,
}

// LateChargeAssessmentWorkflow fans out one activity per tenant that has late
// charge assessment switched on. Tenants run one after another so a large
// tenant cannot starve the billing queue of workers.
func LateChargeAssessmentWorkflow(ctx workflow.Context) (*LateChargeAssessmentResult, error) {
	ctx = workflow.WithActivityOptions(ctx, lateChargeActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	tenants := make([]pagination.TenantInfo, 0)
	if err := workflow.ExecuteActivity(ctx, a.ListLateChargeTenantsActivity).Get(ctx, &tenants); err != nil {
		logger.Error("Late charge tenant listing failed", "error", err)
		return nil, err
	}

	result := &LateChargeAssessmentResult{TenantsDue: len(tenants)}
	for _, tenantInfo := range tenants {
		tenantResult := new(LateChargeTenantResult)
		if err := workflow.ExecuteActivity(
			ctx,
			a.AssessTenantLateChargesActivity,
			tenantInfo,
		).Get(ctx, tenantResult); err != nil {
			logger.Error(
				"Late charge assessment failed for tenant",
				"organizationId", tenantInfo.OrgID.String(),
				"error", err,
			)
			result.TenantsFailed++
			continue
		}
		result.MemosCreated += tenantResult.MemosCreated
		result.MemosPosted += tenantResult.MemosPosted
		result.CustomersSkipped += tenantResult.CustomersSkipped
		result.TotalChargeMinor += tenantResult.TotalChargeMinor
	}
	result.CompletedAt = workflow.Now(ctx).Unix()

	return result, nil
}

func (a *Activities) ListLateChargeTenantsActivity(
	ctx context.Context,
) ([]pagination.TenantInfo, error) {
	if a.lateChargeRepo == nil {
		return []pagination.TenantInfo{}, nil
	}

	return a.lateChargeRepo.ListLateChargeTenants(ctx, lateChargeTenantBatch)
}

// AssessTenantLateChargesActivity runs one tenant's assessment. A tenant in
// Preview mode computes the run and writes nothing, so operators can see what
// switching to Automatic would raise.
func (a *Activities) AssessTenantLateChargesActivity(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*LateChargeTenantResult, error) {
	result := &LateChargeTenantResult{
		OrganizationID: tenantInfo.OrgID.String(),
		BusinessUnitID: tenantInfo.BuID.String(),
	}
	if a.lateChargeService == nil {
		return result, nil
	}

	control, err := a.billingControlRepo.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return nil, err
	}
	preview := control.LateChargeAssessmentMode != tenant.LateChargeAssessmentModeAutomatic
	assessed, err := a.lateChargeService.Assess(ctx, &services.LateChargeAssessmentRequest{
		TenantInfo: tenantInfo,
		AsOfDate:   timeutils.NowUnix(),
		Preview:    preview,
	}, consolidationSystemActor())
	if err != nil {
		a.logger.Error(
			"late charge assessment failed",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	result.Preview = preview
	result.MemosCreated = assessed.MemosCreated
	result.MemosPosted = assessed.MemosPosted
	result.CustomersSkipped = assessed.CustomersSkipped
	result.TotalChargeMinor = assessed.TotalChargeMinor

	return result, nil
}
