package archivejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/jobs/archivejobs/archivejobscontract"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs/tenantjobscontract"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const nightlyPages = 50

var runOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 7 * time.Hour,
	HeartbeatTimeout:    5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    30 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    10 * time.Minute,
		MaximumAttempts:    3,
	},
}

var stepOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    2 * time.Minute,
		MaximumAttempts:    10,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        archivejobscontract.ArchiveWorkflowName,
			Fn:          CloudTenantArchiveWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Take a shared-server tenant's archive to the location the control plane named",
		},
		{
			Name:        archivejobscontract.RestoreWorkflowName,
			Fn:          CloudTenantRestoreWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Restore a suspended shared-server tenant from one of its archives",
		},
		{
			Name:        archivejobscontract.ImportWorkflowName,
			Fn:          CloudTenantImportWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Make a shared-server tenant from an archive taken on another server",
		},
		{
			Name:        archivejobscontract.NightlyWorkflowName,
			Fn:          CloudTenantNightlyArchiveWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Archive every active shared-server tenant once a night and prune its old nightly archives",
		},
	}
}

func failed(
	ctx workflow.Context,
	payload *archivejobscontract.CommandPayload,
	kind tenantcommand.Kind,
	cause error,
) error {
	return settleFailure(
		ctx,
		&FailInput{Payload: payload, Kind: kind, Reason: cause.Error()},
		cause,
	)
}

func failedAfterPurge(
	ctx workflow.Context,
	payload *archivejobscontract.CommandPayload,
	cause error,
) error {
	return settleFailure(ctx, &FailInput{
		Payload: payload,
		Kind:    tenantcommand.KindRestore,
		Reason:  cause.Error(),
		Touched: true,
	}, cause)
}

func settleFailure(ctx workflow.Context, input *FailInput, cause error) error {
	var a *Activities
	step := workflow.WithActivityOptions(ctx, stepOptions)
	if err := workflow.ExecuteActivity(step, a.FailCommandActivity, input).
		Get(ctx, nil); err != nil {
		return err
	}
	return cause
}

func CloudTenantArchiveWorkflow(
	ctx workflow.Context,
	payload *archivejobscontract.CommandPayload,
) (*Outcome, error) {
	var a *Activities
	out := new(Outcome)
	run := workflow.WithActivityOptions(ctx, runOptions)
	if err := workflow.ExecuteActivity(run, a.RunArchiveActivity, payload).
		Get(ctx, out); err != nil {
		return nil, failed(ctx, payload, tenantcommand.KindArchive, err)
	}
	return out, nil
}

func CloudTenantImportWorkflow(
	ctx workflow.Context,
	payload *archivejobscontract.CommandPayload,
) (*Outcome, error) {
	var a *Activities
	out := new(Outcome)
	run := workflow.WithActivityOptions(ctx, runOptions)
	if err := workflow.ExecuteActivity(run, a.RunImportActivity, payload).
		Get(ctx, out); err != nil {
		return nil, failed(ctx, payload, tenantcommand.KindImport, err)
	}
	return out, nil
}

type RestoreOutcome struct {
	Skipped  bool                              `json:"skipped"`
	Sessions *tenantlifecycle.RevocationResult `json:"sessions,omitempty"`
	Purged   *tenantjobs.PurgeOutcome          `json:"purged,omitempty"`
	Loaded   *Outcome                          `json:"loaded,omitempty"`
}

func CloudTenantRestoreWorkflow(
	ctx workflow.Context,
	payload *archivejobscontract.CommandPayload,
) (*RestoreOutcome, error) {
	var (
		a      *Activities
		purger *tenantjobs.Activities
	)
	out := new(RestoreOutcome)
	step := workflow.WithActivityOptions(ctx, stepOptions)
	var restoring bool
	if err := workflow.ExecuteActivity(step, a.CheckRestoreActivity, payload).
		Get(ctx, &restoring); err != nil {
		return nil, err
	}
	if !restoring {
		out.Skipped = true
		return out, nil
	}
	run := workflow.WithActivityOptions(ctx, runOptions)
	verified := new(Outcome)
	if err := workflow.ExecuteActivity(run, a.VerifyRestoreActivity, payload).
		Get(ctx, verified); err != nil {
		return out, failed(ctx, payload, tenantcommand.KindRestore, err)
	}
	if verified.Failed || verified.Settled {
		out.Loaded = verified
		return out, nil
	}
	tenant := &tenantjobscontract.TenantPayload{
		InstanceID:     payload.InstanceID,
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
	}
	out.Sessions = new(tenantlifecycle.RevocationResult)
	if err := workflow.ExecuteActivity(step, purger.RevokeSessionsActivity, tenant).
		Get(ctx, out.Sessions); err != nil {
		return out, failed(ctx, payload, tenantcommand.KindRestore, err)
	}
	purged, err := tenantjobs.PurgeTenant(ctx, tenant)
	out.Purged = purged
	if err != nil {
		return out, failedAfterPurge(ctx, payload, err)
	}
	out.Loaded = new(Outcome)
	if err = workflow.ExecuteActivity(run, a.RunRestoreLoadActivity, payload).
		Get(ctx, out.Loaded); err != nil {
		return out, failedAfterPurge(ctx, payload, err)
	}
	return out, nil
}

type NightlyArgs struct {
	Window      time.Duration                `json:"window"`
	Concurrency int                          `json:"concurrency"`
	Day         string                       `json:"day,omitempty"`
	Started     time.Time                    `json:"started,omitzero"`
	After       *temporaljobs.TenantWorkItem `json:"after,omitempty"`
}

func CloudTenantNightlyArchiveWorkflow(
	ctx workflow.Context,
	args NightlyArgs,
) (*temporaljobs.TenantRunResult, error) {
	var a *Activities
	if args.Day == "" {
		args.Started = workflow.Now(ctx).UTC()
		args.Day = args.Started.Format(time.DateOnly)
	}
	deadline := args.Started.Add(args.Window)
	list := workflow.WithActivityOptions(ctx, stepOptions)
	run := workflow.WithActivityOptions(ctx, runOptions)
	first := args.After
	result, after, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "nightly tenant archive",
		Need:        entitlement.AnyActive(),
		Concurrency: args.Concurrency,
		MaxPages:    nightlyPages,
		ListPage: func(ctx workflow.Context, after *temporaljobs.TenantWorkItem) (*temporaljobs.TenantPage, error) {
			if after == nil {
				after = first
			}
			page := new(temporaljobs.TenantPage)
			err := workflow.ExecuteActivity(list, a.ListNightlyTenantsActivity, after).
				Get(ctx, page)
			return page, err
		},
		RunTenant: func(ctx workflow.Context, item temporaljobs.TenantWorkItem) (int, error) {
			if args.Window > 0 && workflow.Now(ctx).After(deadline) {
				return 0, nil
			}
			var archived int
			err := workflow.ExecuteActivity(run, a.NightlyArchiveActivity, &NightlyInput{
				InstanceID:     item.Cursor,
				OrganizationID: item.OrganizationID,
				BusinessUnitID: item.BusinessUnitID,
				Day:            args.Day,
			}).Get(ctx, &archived)
			return archived, err
		},
	})
	if err != nil {
		return result, err
	}
	if after != nil && (args.Window <= 0 || workflow.Now(ctx).Before(deadline)) {
		args.After = after
		return result, workflow.NewContinueAsNewError(ctx, CloudTenantNightlyArchiveWorkflow, args)
	}
	return result, nil
}
