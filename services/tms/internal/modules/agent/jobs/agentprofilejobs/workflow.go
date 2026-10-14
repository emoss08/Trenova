package agentprofilejobs

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentprofilejobs/agentprofilecontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const secondsPerDay = 24 * 60 * 60

var stepRetry = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2,
	MaximumInterval:    30 * time.Second,
	MaximumAttempts:    5,
}

var listOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	Summary:             "List what to write profiles for",
	RetryPolicy:         stepRetry,
}

var prepareOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 3 * time.Minute,
	Summary:             "Work out the profile's facts",
	RetryPolicy:         stepRetry,
}

var modelOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	Summary:             "Write the profile",
	RetryPolicy:         modelcall.RetryPolicy(3),
}

var finishOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	Summary:             "Keep the profile",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    10,
	},
}

var pruneOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	Summary:             "Prune the profile's old versions",
	RetryPolicy:         stepRetry,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        SweepWorkflowName,
			Fn:          SweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Write the agent profiles of every organization and the people in it",
		},
		{
			Name:        OrganizationWorkflowName,
			Fn:          OrganizationWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Write one organization's agent profile and its people's",
		},
		{
			Name:        ProfileWorkflowName,
			Fn:          ProfileWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Write one agent profile from its facts and guidance",
		},
		{
			Name:        agentprofilecontract.RequestWorkflowName,
			Fn:          RequestWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Write an agent profile again once a person asks or guidance settles",
		},
	}
}

// SweepWorkflow writes every organization's profiles, one child workflow per
// organization, all as of one instant carried across continue-as-new.
func SweepWorkflow(ctx workflow.Context, input *SweepInput) (*SweepResult, error) {
	if input == nil {
		input = &SweepInput{}
	}
	now := input.Now
	if now == 0 {
		now = workflow.Now(ctx).Unix()
	}

	var a *Activities
	result := &SweepResult{}
	_, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "Agent profiles",
		Concurrency: organizationConcurrency,
		ListPage: func(
			wctx workflow.Context,
			after *temporaljobs.TenantWorkItem,
		) (*temporaljobs.TenantPage, error) {
			if after == nil {
				after = input.After
			}
			listCtx := workflow.WithActivityOptions(wctx, listOptions)
			var page temporaljobs.TenantPage
			err := workflow.ExecuteActivity(
				listCtx,
				a.ListOrganizationsActivity,
				&ListOrganizationsInput{
					After: after,
					Limit: temporaljobs.DefaultOrganizationPageSize,
				},
			).Get(listCtx, &page)

			return &page, err
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			childCtx := workflow.WithChildOptions(wctx, workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("agent-profiles/%s/%d", tenant.OrganizationID, now),
				TaskQueue:  temporaltype.TaskQueueSystem.String(),
				Priority:   priorityFor(tenant.TenantInfo()),
			})

			var written OrganizationResult
			if err := workflow.ExecuteChildWorkflow(childCtx, OrganizationWorkflow,
				&OrganizationInput{TenantWorkItem: tenant, Now: now},
			).Get(childCtx, &written); err != nil {
				result.FailedOrganizations = append(result.FailedOrganizations,
					tenant.OrganizationID.String())

				return 0, err
			}
			result.Organizations++
			result.People += written.People
			result.add(&written.ProfileResult)

			return written.Written, nil
		},
	})
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(ctx, SweepWorkflow,
			&SweepInput{After: next, Now: now})
	}

	return result, nil
}

// OrganizationWorkflow writes the organization's profile, then the profile of
// each person who talked to an agent lately, a few at a time. Each profile's
// failure is its own: the organization's does not cost its people theirs, and
// one person's does not fail the organization. An organization that writes no
// profiles at all (learning off, or a plan without agents) writes none of its
// people's either, without asking each one.
func OrganizationWorkflow(
	ctx workflow.Context,
	input *OrganizationInput,
) (*OrganizationResult, error) {
	tenantInfo := input.TenantInfo()
	result := &OrganizationResult{}

	var organization ProfileResult
	if err := runChild(ctx, input.Now, &agentprofileports.GenerateRequest{
		Ref:     agentprofileports.OrganizationSubject(tenantInfo),
		Trigger: agentprofile.TriggerScheduled,
	}, &organization); err != nil {
		result.Failed++
		workflow.GetLogger(ctx).Error("could not write an organization's agent profile",
			"organizationId", tenantInfo.OrgID.String(),
			"error", err.Error(),
		)
	} else {
		result.add(&organization)
		if organization.Gated {
			return result, nil
		}
	}

	var a *Activities
	listCtx := withPriority(ctx, &listOptions, tenantInfo)
	var users []pulid.ID
	if err := workflow.ExecuteActivity(listCtx, a.ListActiveUsersActivity, &ActiveUsersInput{
		TenantWorkItem: input.TenantWorkItem,
		Since:          input.Now - ActiveUserDays*secondsPerDay,
	}).Get(listCtx, &users); err != nil {
		return result, err
	}
	result.People = len(users)

	for start := 0; start < len(users); start += userConcurrency {
		end := min(start+userConcurrency, len(users))
		futures := make([]workflow.ChildWorkflowFuture, 0, end-start)
		for _, userID := range users[start:end] {
			futures = append(futures, startChild(ctx, input.Now, &agentprofileports.GenerateRequest{
				Ref:     agentprofileports.UserSubject(tenantInfo, userID),
				Trigger: agentprofile.TriggerScheduled,
			}))
		}
		for idx, future := range futures {
			var written ProfileResult
			if err := future.Get(ctx, &written); err != nil {
				result.Failed++
				result.FailedUsers = append(result.FailedUsers, users[start+idx])
				workflow.GetLogger(ctx).Error("could not write a person's agent profile",
					"userId", users[start+idx].String(),
					"error", err.Error(),
				)

				continue
			}
			result.add(&written)
		}
	}

	return result, nil
}

// ProfileWorkflow writes one profile: it works out the facts, has the model
// write them up when a version is due, keeps the result and prunes old
// versions.
func ProfileWorkflow(
	ctx workflow.Context,
	req *agentprofileports.GenerateRequest,
) (*ProfileResult, error) {
	result := &ProfileResult{}
	if err := generate(ctx, req, result); err != nil {
		return result, err
	}

	return result, nil
}

// RequestWorkflow writes a profile again on demand. A person asking writes at
// once; guidance waits until it has been quiet for a moment, so several lines
// given together cost one write. A request heard while writing starts another
// round.
func RequestWorkflow(
	ctx workflow.Context,
	input *agentprofilecontract.RequestInput,
) (*ProfileResult, error) {
	signals := workflow.GetSignalChannel(ctx, agentprofilecontract.RequestedSignalName)
	result := &ProfileResult{}
	req := input.Request

	for {
		// The request that started the workflow was also signalled to it, so
		// a person asking once is heard once: what is already waiting is
		// folded in before the write, never after it.
		drain(signals, &req)
		if req.Trigger != agentprofile.TriggerRequested {
			waitForQuiet(ctx, signals, &req)
		}

		if err := generate(ctx, &req, result); err != nil {
			workflow.GetLogger(ctx).Error("could not write an agent profile on request",
				"organizationId", req.Ref.OrganizationID.String(),
				"subject", string(req.Ref.Subject),
				"error", err.Error(),
			)
			result.Failed++
		}

		if !drain(signals, &req) {
			return result, nil
		}
		input.Rounds++
		if input.Rounds >= agentprofilecontract.RoundsPerRun {
			return result, workflow.NewContinueAsNewError(ctx, RequestWorkflow,
				&agentprofilecontract.RequestInput{Request: req})
		}
	}
}

func generate(
	ctx workflow.Context,
	req *agentprofileports.GenerateRequest,
	result *ProfileResult,
) error {
	tenantInfo := req.Ref.Tenant()

	var a *Activities
	var plan agentprofileports.GenerationPlan
	if err := workflow.ExecuteActivity(
		withPriority(ctx, &prepareOptions, tenantInfo),
		a.PrepareProfileActivity, req,
	).Get(ctx, &plan); err != nil {
		return err
	}
	if !plan.Ready() {
		result.Skipped++
		result.Gated = plan.Skip == agentprofileports.SkipNotAllowed ||
			plan.Skip == agentprofileports.SkipLearningOff

		return nil
	}

	finish := &agentprofileports.FinishGenerationRequest{Plan: &plan}
	var reply ModelReply
	if err := workflow.ExecuteActivity(
		withPriority(ctx, &modelOptions, tenantInfo),
		a.ProfileModelActivity, &plan,
	).Get(ctx, &reply); err != nil {
		finish.Error = "The model could not write the profile: " + err.Error()
	} else if reply.NoProvider {
		result.Skipped++

		return nil
	} else {
		finish.Result = reply.Result
	}

	var outcome agentprofileports.GenerationOutcome
	if err := workflow.ExecuteActivity(
		withPriority(ctx, &finishOptions, tenantInfo),
		a.FinishProfileActivity, finish,
	).Get(ctx, &outcome); err != nil {
		return err
	}
	result.absorb(&outcome)

	if err := workflow.ExecuteActivity(
		withPriority(ctx, &pruneOptions, tenantInfo),
		a.PruneProfileActivity, &plan.Ref,
	).Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Warn("could not prune an agent profile's old versions",
			"organizationId", tenantInfo.OrgID.String(),
			"error", err.Error(),
		)
	}

	return nil
}

func childOptions(
	ctx workflow.Context,
	now int64,
	req *agentprofileports.GenerateRequest,
) workflow.Context {
	key := "organization"
	if req.Ref.Subject == agentprofile.SubjectUser {
		key = "user/" + req.Ref.PersonID.String()
	}

	return workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: fmt.Sprintf("agent-profile/%s/%s/%d", req.Ref.OrganizationID, key, now),
		TaskQueue:  temporaltype.TaskQueueSystem.String(),
		Priority:   priorityFor(req.Ref.Tenant()),
	})
}

func startChild(
	ctx workflow.Context,
	now int64,
	req *agentprofileports.GenerateRequest,
) workflow.ChildWorkflowFuture {
	return workflow.ExecuteChildWorkflow(childOptions(ctx, now, req), ProfileWorkflow, req)
}

func runChild(
	ctx workflow.Context,
	now int64,
	req *agentprofileports.GenerateRequest,
	into *ProfileResult,
) error {
	return startChild(ctx, now, req).Get(ctx, into)
}

func waitForQuiet(
	ctx workflow.Context,
	signals workflow.ReceiveChannel,
	req *agentprofileports.GenerateRequest,
) {
	deadline := workflow.Now(ctx).Add(agentprofilecontract.LongestWait)
	drain(signals, req)

	for req.Trigger != agentprofile.TriggerRequested {
		wait := min(agentprofilecontract.GuidanceQuietPeriod, deadline.Sub(workflow.Now(ctx)))
		if wait <= 0 {
			return
		}

		timerCtx, cancel := workflow.WithCancel(ctx)
		heard := false
		selector := workflow.NewSelector(ctx)
		selector.AddReceive(signals, func(channel workflow.ReceiveChannel, _ bool) {
			var signal agentprofileports.GenerateRequest
			channel.Receive(ctx, &signal)
			merge(req, &signal)
			heard = true
		})
		selector.AddFuture(workflow.NewTimer(timerCtx, wait), func(workflow.Future) {})
		selector.Select(ctx)
		cancel()

		if !heard {
			return
		}
		drain(signals, req)
	}
}

// drain folds every request already waiting into req and reports whether
// there was one.
func drain(signals workflow.ReceiveChannel, req *agentprofileports.GenerateRequest) bool {
	heard := false
	for {
		var signal agentprofileports.GenerateRequest
		if !signals.ReceiveAsync(&signal) {
			return heard
		}
		merge(req, &signal)
		heard = true
	}
}

// merge folds a later request into a waiting one: a person asking outranks
// guidance, and the latest person to ask is named.
func merge(current, next *agentprofileports.GenerateRequest) {
	if next.Trigger == agentprofile.TriggerRequested ||
		current.Trigger == agentprofile.TriggerScheduled {
		current.Trigger = next.Trigger
	}
	if next.RequestedByID.IsNotNil() {
		current.RequestedByID = next.RequestedByID
	}
}

func priorityFor(tenantInfo pagination.TenantInfo) temporal.Priority {
	return agentprofilecontract.Priority(tenantInfo)
}

func withPriority(
	ctx workflow.Context,
	options *workflow.ActivityOptions,
	tenantInfo pagination.TenantInfo,
) workflow.Context {
	prioritized := *options
	prioritized.Priority = priorityFor(tenantInfo)

	return workflow.WithActivityOptions(ctx, prioritized)
}
