package podchasejobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/podchasejobs/podchasecontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// chasePageSize is the shipments one activity asks about: each is a run
	// record, a dispatched call and a recorded proposal, no model call.
	chasePageSize   = 25
	chaseRetryAfter = 5 * time.Minute
	promptVersion   = "routine.pod-chase.v1"
)

type ActivitiesParams struct {
	fx.In

	Queue         workqueueports.Queue
	Facts         podchaseports.FactsReader
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Runs          agentruntimeports.AgentRunService
	RunRepo       agentruntimeports.AgentRunRepository
	EventRuns     agenteventports.EventRunQueue
	Runtime       *agentruntime.Service
	Recorder      *proposalrecorder.Service
	Users         tenantports.UserRepository
	Organizations tenantports.OrganizationRepository
	Logger        *zap.Logger
}

type Activities struct {
	queue         workqueueports.Queue
	facts         podchaseports.FactsReader
	definitions   agentdefinitionports.AgentDefinitionRepository
	runs          agentruntimeports.AgentRunService
	runRepo       agentruntimeports.AgentRunRepository
	eventRuns     agenteventports.EventRunQueue
	runtime       *agentruntime.Service
	recorder      *proposalrecorder.Service
	users         tenantports.UserRepository
	organizations tenantports.OrganizationRepository
	l             *zap.Logger
}

func NewActivities(
	p ActivitiesParams, //nolint:gocritic // fx requires an fx.In parameter struct by value
) *Activities {
	return &Activities{
		queue:         p.Queue,
		facts:         p.Facts,
		definitions:   p.Definitions,
		runs:          p.Runs,
		runRepo:       p.RunRepo,
		eventRuns:     p.EventRuns,
		runtime:       p.Runtime,
		recorder:      p.Recorder,
		users:         p.Users,
		organizations: p.Organizations,
		l:             p.Logger.Named("job.pod-chase"),
	}
}

// ChasePODsActivity asks for a page of the tenant's queued shipments' proofs
// of delivery. A shipment whose request could not be filed is due again
// after chaseRetryAfter; the rest are settled.
func (a *Activities) ChasePODsActivity(
	ctx context.Context,
	payload *temporaljobs.DrainPayload,
) (*temporaljobs.DrainPage, error) {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return temporaljobs.DrainDue(ctx, &temporaljobs.DrainDueParams{
		Queue:      a.queue,
		Kind:       podchasecontract.DrainKind,
		Tenant:     tenant,
		PageSize:   chasePageSize,
		RetryAfter: chaseRetryAfter,
		Handle: func(ctx context.Context, shipments []pulid.ID, agents map[pulid.ID]string) map[pulid.ID]error {
			failed := make(map[pulid.ID]error)
			for definitionID, group := range byAgent(shipments, agents) {
				for id, err := range a.chase(ctx, tenant, definitionID, group) {
					failed[id] = err
				}
			}

			return failed
		},
	})
}

func byAgent(shipments []pulid.ID, agents map[pulid.ID]string) map[pulid.ID][]pulid.ID {
	groups := make(map[pulid.ID][]pulid.ID)
	for _, id := range shipments {
		definitionID := pulid.ID(agents[id])
		groups[definitionID] = append(groups[definitionID], id)
	}

	return groups
}

// chasing is what a page asks with for one agent: the agent, who it acts as,
// the clock its dates read in and the billing sender for email.
type chasing struct {
	tenant     pagination.TenantInfo
	definition *agentdefinition.Definition
	actor      *permissionports.RequestActor
	timezone   string
	sender     string
}

// chase asks about one agent's shipments. A shipment no longer missing its
// proof of delivery, or already in an open run of the agent, is settled as
// it is; one nobody can be asked about goes to the agent's model.
func (a *Activities) chase(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definitionID pulid.ID,
	shipments []pulid.ID,
) map[pulid.ID]error {
	failed := make(map[pulid.ID]error)
	failAll := func(err error) map[pulid.ID]error {
		for _, id := range shipments {
			failed[id] = err
		}
		return failed
	}

	c, settled, err := a.chasingFor(ctx, tenant, definitionID)
	if err != nil {
		return failAll(err)
	}
	if settled {
		return failed
	}

	facts, err := a.facts.ListPODChaseFacts(ctx, &podchaseports.ListFactsRequest{
		TenantInfo:      tenant,
		ShipmentIDs:     shipments,
		DeliveredBefore: timeutils.NowUnix(),
	})
	if err != nil {
		return failAll(err)
	}
	open, err := a.runRepo.ListOpenSubjects(ctx, &agentruntimeports.ListOpenSubjectsRequest{
		TenantInfo:   tenant,
		DefinitionID: definitionID,
		SubjectIDs:   shipments,
	})
	if err != nil {
		return failAll(err)
	}
	busy := make(map[pulid.ID]struct{}, len(open))
	for _, id := range open {
		busy[id] = struct{}{}
	}

	for _, fact := range facts {
		if ctx.Err() != nil {
			failed[fact.ShipmentID] = ctx.Err()
			continue
		}
		if _, ok := busy[fact.ShipmentID]; ok {
			continue
		}
		if activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx, fact.ShipmentID.String())
		}
		if cErr := a.ask(ctx, c, fact); cErr != nil {
			a.l.Warn("could not ask for a proof of delivery; it is tried again",
				zap.String("shipmentId", fact.ShipmentID.String()),
				zap.Error(cErr))
			failed[fact.ShipmentID] = cErr
		}
	}

	return failed
}

// chasingFor loads the agent and what its requests need. settled is true when
// the agent is gone or off, so the queued shipments are dropped.
func (a *Activities) chasingFor(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definitionID pulid.ID,
) (*chasing, bool, error) {
	definition, err := a.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         definitionID,
			TenantInfo: tenant,
		},
	)
	if errortypes.IsNotFoundError(err) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !definition.Enabled || !definition.IsBackground() {
		return nil, true, nil
	}

	system, err := a.users.GetSystemUser(ctx, "id")
	if err != nil {
		return nil, false, fmt.Errorf("read the system user: %w", err)
	}
	org, err := a.organizations.GetByID(
		ctx,
		tenantports.GetOrganizationByIDRequest{TenantInfo: tenant},
	)
	if err != nil {
		return nil, false, fmt.Errorf("read the organization's clock: %w", err)
	}
	sender, err := a.facts.BillingSenderProfile(ctx, tenant)
	if err != nil {
		return nil, false, err
	}

	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    definition.ID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		UserID:         system.ID,
	}

	return &chasing{
		tenant:     tenant,
		definition: definition,
		actor:      actor,
		timezone:   org.Timezone,
		sender:     stringOf(sender),
	}, false, nil
}

func stringOf(id pulid.ID) string {
	if id.IsNil() {
		return ""
	}

	return id.String()
}

// ask files one shipment's request: a run of the agent about the shipment,
// started by the event, its one call put through the runtime's policy as a
// model's call would be, and the proposal it makes recorded like a run's.
// A shipment nobody can be asked about, or whose call the runtime refuses,
// is queued for the agent's model instead.
func (a *Activities) ask(ctx context.Context, c *chasing, fact *podchaseports.Facts) error {
	call := planCall(fact, c.sender, c.timezone)
	if call == nil {
		return a.toModel(ctx, c, fact.ShipmentID)
	}

	run, err := a.runs.StartInline(ctx, &agentruntimeports.StartInlineAgentRunRequest{
		AgentType:         agent.TypeBillingException,
		AgentDefinitionID: c.definition.ID,
		SubjectType:       agent.SubjectShipment,
		SubjectID:         fact.ShipmentID,
		PromptVersion:     promptVersion,
		Summary:           "Asked for the signed proof of delivery",
		Trigger:           agent.RunTriggerEvent,
		TenantInfo:        c.tenant,
	}, c.actor)
	if err != nil {
		return fmt.Errorf("open the run that asks for the proof of delivery: %w", err)
	}

	outcome := a.runtime.DispatchStep(ctx, a.runRequest(c, run), &toolcall.Call{
		Call: llmports.ToolCall{
			ID:        "pod-chase-" + fact.ShipmentID.String(),
			Name:      call.tool,
			Arguments: call.arguments,
		},
		CompletionText: call.rationale,
	})
	if outcome.Action == nil {
		a.end(ctx, c, run, "The request could not be filed: "+outcome.Content)
		return a.toModel(ctx, c, fact.ShipmentID)
	}

	if _, err = a.recorder.Record(ctx, &proposalrecorder.RecordRequest{
		Actor:      c.actor,
		Definition: c.definition,
		Run:        run,
		Actions:    []agentruntimeports.PendingAction{*outcome.Action},
	}); err != nil {
		a.end(ctx, c, run, "The request could not be recorded.")
		return fmt.Errorf("record the proof of delivery request: %w", err)
	}

	return nil
}

func (a *Activities) runRequest(c *chasing, run *agent.AgentRun) *agentruntimeports.RunRequest {
	return &agentruntimeports.RunRequest{
		Definition: c.definition,
		Actor:      c.actor,
		Context: agentturn.RuntimeContext{
			Now:      timeutils.NowUnix(),
			Trigger:  agent.RunTriggerEvent,
			Timezone: c.timezone,
		},
		RunID:      run.ID,
		Unattended: true,
	}
}

// end fails a run whose request did not get filed, so it neither stays open
// nor keeps the shipment from its next request.
func (a *Activities) end(ctx context.Context, c *chasing, run *agent.AgentRun, reason string) {
	completedAt := timeutils.NowUnix()
	run.Status = agent.RunStatusFailed
	run.ErrorMessage = stringutils.Ellipsize(reason, maxReasonRunes)
	run.CompletedAt = &completedAt
	if _, err := a.runRepo.Update(ctx, run); err != nil {
		a.l.Warn("could not close a proof of delivery run that filed nothing",
			zap.String("run", run.ID.String()),
			zap.String("organizationId", c.tenant.OrgID.String()),
			zap.Error(err))
	}
}

const maxReasonRunes = 500

// toModel queues the shipment for the agent's own run, which reads it and
// decides what to do when the routine cannot.
func (a *Activities) toModel(ctx context.Context, c *chasing, shipmentID pulid.ID) error {
	return a.eventRuns.Enqueue(ctx, &agenteventports.EventRunRequest{
		TenantInfo:   c.tenant,
		DefinitionID: c.definition.ID,
		Kind:         agenttrigger.EventShipmentPODMissing,
		SubjectID:    shipmentID,
	})
}
