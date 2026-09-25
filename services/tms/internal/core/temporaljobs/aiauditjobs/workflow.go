package aiauditjobs

import (
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ProjectAIAuditWorkflowName         = "ProjectAIAuditWorkflow"
	PruneAIAuditWorkflowName           = "PruneAIAuditWorkflow"
	CleanupAIAuditExportsWorkflowName  = "CleanupAIAuditExportsWorkflow"
	VerifyAIAuditChainWorkflowName     = serviceports.AIAuditVerifyWorkflowName
	VerifyAIAuditTenantsWorkflowName   = "VerifyAIAuditTenantsWorkflow"
	AIAuditExportWorkflowName          = serviceports.AIAuditExportWorkflowName
	verifyTenantsPerRun                = 200
	projectorActivityTimeout           = 15 * time.Minute
	verifyActivityTimeout              = 2 * time.Hour
	pruneActivityTimeout               = 2 * time.Hour
	exportActivityTimeout              = 2 * time.Hour
	cleanupActivityTimeout             = 30 * time.Minute
	longHeartbeatTimeout               = 2 * time.Minute
	shortHeartbeatTimeout              = time.Minute
	defaultActivityRetryMaximumAttempt = 3
)

func retryPolicy() *temporal.RetryPolicy {
	return &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    defaultActivityRetryMaximumAttempt,
	}
}

func activityOptions(timeout, heartbeat time.Duration) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: timeout,
		HeartbeatTimeout:    heartbeat,
		RetryPolicy:         retryPolicy(),
	}
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        ProjectAIAuditWorkflowName,
			Fn:          ProjectAIAuditWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Write new agent activity to the AI audit trail",
		},
		{
			Name:        VerifyAIAuditChainWorkflowName,
			Fn:          VerifyAIAuditChainWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Verify the AI audit trail's hash chain for one tenant or every tenant",
		},
		{
			Name:        VerifyAIAuditTenantsWorkflowName,
			Fn:          VerifyAIAuditTenantsWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Continue verifying the AI audit trail's hash chain for the tenants left",
		},
		{
			Name:        PruneAIAuditWorkflowName,
			Fn:          PruneAIAuditWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Remove AI audit trail rows past each organization's retention period",
		},
		{
			Name:        CleanupAIAuditExportsWorkflowName,
			Fn:          CleanupAIAuditExportsWorkflow,
			TaskQueue:   temporaltype.AuditTaskQueue,
			Description: "Delete AI audit export files past their download window",
		},
	}
}

func RegisterExportWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        AIAuditExportWorkflowName,
			Fn:          AIAuditExportWorkflow,
			TaskQueue:   temporaltype.ReportTaskQueue,
			Description: "Write a large AI audit trail export to object storage",
		},
	}
}

// ProjectAIAuditWorkflow is one projector pass. Its schedule skips a firing
// while a pass is still running, which is what keeps the projector the
// trail's only writer.
func ProjectAIAuditWorkflow(ctx workflow.Context) (*ProjectResult, error) {
	ctx = workflow.WithActivityOptions(
		ctx,
		activityOptions(projectorActivityTimeout, shortHeartbeatTimeout),
	)

	var a *Activities
	var result *ProjectResult
	if err := workflow.ExecuteActivity(ctx, a.ProjectAIAuditActivity).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// VerifyAIAuditChainWorkflow checks one tenant's chain, or every tenant's
// when none is named, a tenant per activity. A long sweep continues as new
// every so many tenants so its history stays small.
func VerifyAIAuditChainWorkflow(
	ctx workflow.Context,
	payload *serviceports.AIAuditVerifyPayload,
) (*VerifyResult, error) {
	ctx = workflow.WithActivityOptions(
		ctx,
		activityOptions(verifyActivityTimeout, longHeartbeatTimeout),
	)
	var a *Activities
	if payload == nil {
		payload = &serviceports.AIAuditVerifyPayload{}
	}

	if payload.OrganizationID.IsNotNil() {
		var tenantResult *TenantVerification
		if err := workflow.ExecuteActivity(ctx, a.VerifyAIAuditTenantActivity,
			&pagination.TenantInfo{OrgID: payload.OrganizationID, BuID: payload.BusinessUnitID},
		).Get(ctx, &tenantResult); err != nil {
			return nil, err
		}

		return &VerifyResult{Tenants: []*TenantVerification{tenantResult}}, nil
	}

	var tenants []pagination.TenantInfo
	if err := workflow.ExecuteActivity(ctx, a.ListAIAuditTenantsActivity).
		Get(ctx, &tenants); err != nil {
		return nil, err
	}

	return verifyTenants(ctx, tenants)
}

// VerifyAIAuditTenantsWorkflow is where a long sweep continues, with the
// tenants still to check.
func VerifyAIAuditTenantsWorkflow(
	ctx workflow.Context,
	tenants []pagination.TenantInfo,
) (*VerifyResult, error) {
	ctx = workflow.WithActivityOptions(
		ctx,
		activityOptions(verifyActivityTimeout, longHeartbeatTimeout),
	)

	return verifyTenants(ctx, tenants)
}

func verifyTenants(ctx workflow.Context, tenants []pagination.TenantInfo) (*VerifyResult, error) {
	var a *Activities
	result := &VerifyResult{Tenants: make([]*TenantVerification, 0, len(tenants))}

	for i, tenant := range tenants {
		if i == verifyTenantsPerRun {
			return nil, workflow.NewContinueAsNewError(
				ctx, VerifyAIAuditTenantsWorkflow, tenants[i:],
			)
		}

		var tenantResult *TenantVerification
		if err := workflow.ExecuteActivity(ctx, a.VerifyAIAuditTenantActivity, &tenant).
			Get(ctx, &tenantResult); err != nil {
			workflow.GetLogger(ctx).Error("an AI audit chain could not be verified",
				"organizationId", tenant.OrgID.String(), "error", err)
			result.Errors++

			continue
		}
		result.Tenants = append(result.Tenants, tenantResult)
	}

	return result, nil
}

func PruneAIAuditWorkflow(ctx workflow.Context) (*PruneResult, error) {
	ctx = workflow.WithActivityOptions(
		ctx,
		activityOptions(pruneActivityTimeout, longHeartbeatTimeout),
	)

	var a *Activities
	var result *PruneResult
	if err := workflow.ExecuteActivity(ctx, a.PruneAIAuditActivity).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func CleanupAIAuditExportsWorkflow(ctx workflow.Context) (*CleanupResult, error) {
	ctx = workflow.WithActivityOptions(
		ctx,
		activityOptions(cleanupActivityTimeout, shortHeartbeatTimeout),
	)

	var a *Activities
	var result *CleanupResult
	if err := workflow.ExecuteActivity(ctx, a.CleanupAIAuditExportsActivity).
		Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// AIAuditExportWorkflow writes an export too large to write while its
// requester waits.
func AIAuditExportWorkflow(
	ctx workflow.Context,
	payload *serviceports.AIAuditExportPayload,
) (*ExportResult, error) {
	ctx = workflow.WithActivityOptions(
		ctx,
		activityOptions(exportActivityTimeout, longHeartbeatTimeout),
	)

	var a *ExportActivities
	var result *ExportResult
	if err := workflow.ExecuteActivity(ctx, a.RunAIAuditExportActivity, payload).
		Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}
