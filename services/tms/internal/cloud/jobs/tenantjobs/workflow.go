package tenantjobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs/tenantjobscontract"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const maxPurgeRounds = 5_000

var ErrPurgeStalled = errors.New(
	"the tenant's rows stopped going down; the next run of the removal resumes from what is left",
)

var retryPolicy = &temporal.RetryPolicy{
	InitialInterval:    5 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    5 * time.Minute,
	MaximumAttempts:    10,
}

var stepOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy:         retryPolicy,
}

var purgeOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    5 * time.Minute,
	RetryPolicy:         retryPolicy,
}

var verifyOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    30 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    10 * time.Minute,
		MaximumAttempts:    6,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        tenantjobscontract.RemovalWorkflowName,
			Fn:          CloudTenantRemovalWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Delete a removed shared-server tenant's rows, documents, search documents and cache keys, then report it removed",
		},
		{
			Name:        tenantjobscontract.SessionsWorkflowName,
			Fn:          CloudTenantSessionsWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Sign a suspended shared-server tenant's people out",
		},
	}
}

type RemovalResult struct {
	Rows     int64                             `json:"rows"`
	Stores   *StoresResult                     `json:"stores"`
	Sessions *tenantlifecycle.RevocationResult `json:"sessions"`
	Left     []cloudtenantports.TableCount     `json:"left"`
	Unit     *cloudtenantports.DeleteResult    `json:"unit"`
	Org      *cloudtenantports.DeleteResult    `json:"organization"`
	Skipped  bool                              `json:"skipped"`
	Payload  *tenantjobscontract.TenantPayload `json:"payload"`
}

func CloudTenantRemovalWorkflow(
	ctx workflow.Context,
	payload *tenantjobscontract.TenantPayload,
) (*RemovalResult, error) {
	var a *Activities
	out := &RemovalResult{Payload: payload}
	step := workflow.WithActivityOptions(ctx, stepOptions)

	var removing bool
	if err := workflow.ExecuteActivity(step, a.CheckRemovalActivity, payload).
		Get(ctx, &removing); err != nil {
		return nil, err
	}
	if !removing {
		out.Skipped = true
		return out, nil
	}

	out.Sessions = new(tenantlifecycle.RevocationResult)
	if err := workflow.ExecuteActivity(step, a.ReleaseMembersActivity, payload).
		Get(ctx, out.Sessions); err != nil {
		return nil, err
	}

	purged, err := PurgeTenant(ctx, payload)
	if purged != nil {
		out.Rows, out.Org, out.Unit, out.Stores, out.Left = purged.Rows, purged.Org, purged.Unit,
			purged.Stores, purged.Left
	}
	if err != nil {
		return out, err
	}
	if err = workflow.ExecuteActivity(step, a.FinishRemovalActivity, &FinishInput{
		Payload:  payload,
		Rows:     out.Rows,
		Stores:   out.Stores,
		Sessions: out.Sessions.Sessions,
	}).Get(ctx, nil); err != nil {
		return nil, err
	}
	return out, nil
}

type PurgeOutcome struct {
	Rows   int64                          `json:"rows"`
	Org    *cloudtenantports.DeleteResult `json:"organization"`
	Unit   *cloudtenantports.DeleteResult `json:"unit"`
	Stores *StoresResult                  `json:"stores"`
	Left   []cloudtenantports.TableCount  `json:"left"`
}

func PurgeTenant(
	ctx workflow.Context,
	payload *tenantjobscontract.TenantPayload,
) (*PurgeOutcome, error) {
	var a *Activities
	out := new(PurgeOutcome)
	step := workflow.WithActivityOptions(ctx, stepOptions)
	rows, err := purgeUntilStalled(ctx, payload)
	out.Rows += rows
	if err != nil {
		return out, err
	}
	out.Org = new(cloudtenantports.DeleteResult)
	if err = workflow.ExecuteActivity(step, a.DeleteOrganizationActivity, payload).
		Get(ctx, out.Org); err != nil {
		return out, err
	}
	if rows, err = purgeUntilStalled(ctx, payload); err != nil {
		return out, err
	}
	out.Rows += rows
	out.Unit = new(cloudtenantports.DeleteResult)
	if err = workflow.ExecuteActivity(step, a.DeleteBusinessUnitActivity, payload).
		Get(ctx, out.Unit); err != nil {
		return out, err
	}
	out.Stores = new(StoresResult)
	if err = workflow.ExecuteActivity(step, a.PurgeStoresActivity, payload).
		Get(ctx, out.Stores); err != nil {
		return out, err
	}
	verify := workflow.WithActivityOptions(ctx, verifyOptions)
	err = workflow.ExecuteActivity(verify, a.VerifyRemovedActivity, payload).Get(ctx, &out.Left)
	return out, err
}

func purgeUntilStalled(
	ctx workflow.Context,
	payload *tenantjobscontract.TenantPayload,
) (int64, error) {
	var a *Activities
	purge := workflow.WithActivityOptions(ctx, purgeOptions)
	var total int64
	for range maxPurgeRounds {
		result := new(cloudtenantports.PurgeRowsResult)
		if err := workflow.ExecuteActivity(purge, a.PurgeRowsActivity, payload).
			Get(ctx, result); err != nil {
			return total, err
		}
		total += result.Deleted
		if result.Complete || result.Deleted == 0 {
			return total, nil
		}
	}
	return total, ErrPurgeStalled
}

func CloudTenantSessionsWorkflow(
	ctx workflow.Context,
	payload *tenantjobscontract.TenantPayload,
) (*tenantlifecycle.RevocationResult, error) {
	var a *Activities
	out := new(tenantlifecycle.RevocationResult)
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, stepOptions),
		a.RevokeSessionsActivity,
		payload,
	).Get(ctx, out)
	return out, err
}
