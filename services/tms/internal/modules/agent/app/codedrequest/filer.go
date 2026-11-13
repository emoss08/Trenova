package codedrequest

import (
	"context"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const maxReasonRunes = 500

type Params struct {
	fx.In

	Definitions   agentdefinitionports.AgentDefinitionRepository
	Runs          agentruntimeports.AgentRunService
	RunRepo       agentruntimeports.AgentRunRepository
	Runtime       *agentruntime.Service
	Recorder      *proposalrecorder.Service
	Shadow        *agentshadow.Resolver
	Users         tenantports.UserRepository
	Organizations tenantports.OrganizationRepository
	Plans         platformplanports.PlanService
	Logger        *zap.Logger
}

// Filer files a request that code wrote for an agent: a run of the agent
// about the record, its one tool call put through the runtime's policy as a
// model's call would be, and the proposal it makes recorded like a run's.
// The tier, holds, egress and shadow are the runtime's own, so a coded
// request is held, approved and audited exactly as a model's is. No model
// is called. It is for an agent acting on its own, as a routine does when
// an event reaches it; a person's own request from the Watchtower is sent
// under their name and does not come through here.
type Filer struct {
	definitions   agentdefinitionports.AgentDefinitionRepository
	runs          agentruntimeports.AgentRunService
	runRepo       agentruntimeports.AgentRunRepository
	runtime       *agentruntime.Service
	recorder      *proposalrecorder.Service
	shadow        *agentshadow.Resolver
	users         tenantports.UserRepository
	organizations tenantports.OrganizationRepository
	plans         platformplanports.PlanService
	l             *zap.Logger
}

func New(p Params) *Filer { //nolint:gocritic // fx requires an fx.In parameter struct by value
	return &Filer{
		definitions:   p.Definitions,
		runs:          p.Runs,
		runRepo:       p.RunRepo,
		runtime:       p.Runtime,
		recorder:      p.Recorder,
		shadow:        p.Shadow,
		users:         p.Users,
		organizations: p.Organizations,
		plans:         p.Plans,
		l:             p.Logger.Named("coded-request"),
	}
}

// Filing is what requests are filed with for one agent: the agent, who it
// acts as, the clock its dates read in and whether it is in shadow, so its
// request is recorded as what it would have asked and reaches nobody.
type Filing struct {
	Tenant     pagination.TenantInfo
	Definition *agentdefinition.Definition
	Timezone   string
	// Texts says the organization's plan includes text messages, so a
	// driver with a phone number and no login can still be reached.
	Texts  bool
	actor  *permissionports.RequestActor
	shadow bool
}

// For loads the agent and what its requests need. Nil when the agent is
// gone or off, so nothing is filed for it.
func (f *Filer) For(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definitionID pulid.ID,
) (*Filing, error) {
	definition, err := f.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         definitionID,
			TenantInfo: tenant,
		},
	)
	if errortypes.IsNotFoundError(err) {
		return nil, nil //nolint:nilnil // no agent is an answer, not a failure
	}
	if err != nil {
		return nil, err
	}

	return f.with(ctx, tenant, definition)
}

func (f *Filer) with(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definition *agentdefinition.Definition,
) (*Filing, error) {
	if definition == nil || !definition.Enabled || !definition.IsBackground() {
		return nil, nil //nolint:nilnil // an agent that is off files nothing
	}

	system, err := f.users.GetSystemUser(ctx, "id")
	if err != nil {
		return nil, fmt.Errorf("read the system user: %w", err)
	}
	org, err := f.organizations.GetByID(
		ctx,
		tenantports.GetOrganizationByIDRequest{TenantInfo: tenant},
	)
	if err != nil {
		return nil, fmt.Errorf("read the organization's clock: %w", err)
	}
	verdict, err := f.shadow.ForDefinition(ctx, tenant, definition)
	if err != nil {
		return nil, fmt.Errorf("read whether the agent is in shadow: %w", err)
	}

	texts, err := planservice.Allows(ctx, f.plans, tenant, platformplan.CapabilitySMS)
	if err != nil {
		return nil, fmt.Errorf("read whether the plan includes text messages: %w", err)
	}

	return &Filing{
		Tenant:     tenant,
		Definition: definition,
		Timezone:   org.Timezone,
		Texts:      texts,
		actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeAgent,
			PrincipalID:    definition.ID,
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			UserID:         system.ID,
		},
		shadow: verdict.Shadow(),
	}, nil
}

// Busy reports whether the agent already has an open run about the record,
// so the same request is not filed twice.
func (f *Filer) Busy(ctx context.Context, filing *Filing, subjectID pulid.ID) (bool, error) {
	open, err := f.runRepo.ListOpenSubjects(ctx, &agentruntimeports.ListOpenSubjectsRequest{
		TenantInfo:   filing.Tenant,
		DefinitionID: filing.Definition.ID,
		SubjectIDs:   []pulid.ID{subjectID},
	})
	if err != nil {
		return false, err
	}

	return len(open) > 0, nil
}

// Request is one coded request: the record it is about, the tool call that
// makes it and why, in the words its proposal carries.
type Request struct {
	AgentType     agent.Type
	SubjectType   agent.SubjectType
	SubjectID     pulid.ID
	PromptVersion string
	// Summary is the run's, as a person reads it in the agent's history.
	Summary   string
	Trigger   agent.RunTrigger
	Tool      string
	Arguments map[string]any
	Rationale string
}

// File opens the run, dispatches the call and records what it made.
func (f *Filer) File(ctx context.Context, filing *Filing, req *Request) (*agent.AgentRun, error) {
	run, err := f.runs.StartInline(ctx, &agentruntimeports.StartInlineAgentRunRequest{
		AgentType:         req.AgentType,
		AgentDefinitionID: filing.Definition.ID,
		SubjectType:       req.SubjectType,
		SubjectID:         req.SubjectID,
		PromptVersion:     req.PromptVersion,
		Summary:           req.Summary,
		Trigger:           req.Trigger,
		TenantInfo:        filing.Tenant,
	}, filing.actor)
	if err != nil {
		return nil, fmt.Errorf("open the run that files the request: %w", err)
	}

	outcome := f.runtime.DispatchStep(ctx, &agentruntimeports.RunRequest{
		Definition: filing.Definition,
		Actor:      filing.actor,
		Context: agentturn.RuntimeContext{
			Now:      timeutils.NowUnix(),
			Trigger:  req.Trigger,
			Timezone: filing.Timezone,
		},
		RunID:      run.ID,
		Unattended: true,
		Shadow:     filing.shadow,
	}, &toolcall.Call{
		Call: llmports.ToolCall{
			ID:        req.Tool + "-" + req.SubjectID.String(),
			Name:      req.Tool,
			Arguments: req.Arguments,
		},
		CompletionText: req.Rationale,
	})
	if outcome.Action == nil {
		f.end(ctx, filing, run, "The request could not be filed: "+outcome.Content)
		return nil, fmt.Errorf("file the request: %s", outcome.Content)
	}

	if _, err = f.recorder.Record(ctx, &proposalrecorder.RecordRequest{
		Actor:      filing.actor,
		Definition: filing.Definition,
		Run:        run,
		Actions:    []agentruntimeports.PendingAction{*outcome.Action},
	}); err != nil {
		f.end(ctx, filing, run, "The request could not be recorded.")
		return nil, fmt.Errorf("record the request: %w", err)
	}

	return run, nil
}

// end fails a run whose request did not get filed, so it neither stays open
// nor keeps the record from its next request.
func (f *Filer) end(ctx context.Context, filing *Filing, run *agent.AgentRun, reason string) {
	completedAt := timeutils.NowUnix()
	run.Status = agent.RunStatusFailed
	run.ErrorMessage = stringutils.Ellipsize(reason, maxReasonRunes)
	run.CompletedAt = &completedAt
	if _, err := f.runRepo.Update(ctx, run); err != nil {
		f.l.Warn("could not close a coded request's run that filed nothing",
			zap.String("run", run.ID.String()),
			zap.String("organizationId", filing.Tenant.OrgID.String()),
			zap.Error(err))
	}
}
