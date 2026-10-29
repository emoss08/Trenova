package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevidence"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	activitySeedHour = int64(3600)
	activitySeedDay  = 24 * activitySeedHour
	// seedActivityModel matches the local provider the AI provider seed writes,
	// so a run opened in AI Control names a model the organization actually has.
	seedActivityModel = "qwen2.5-14b-instruct"
)

type AgentActivitySeed struct {
}

// AgentActivitySeed fills what the desks have been doing: runs in every
// state, decisions waiting and decisions already taken, a plan whose steps
// are one approval, the exceptions a run raised when it could not finish, and
// the trust ledger behind the promotion story.
//
// Between them the Decisions queue has something to approve, reject and batch;
// AI Control's activity tab has runs and exceptions to open; and the scorecard
// panel has a record to draw, because a scorecard is computed from these rows
// rather than stored — seed the rows and the panel fills itself.
//
// The proposals are spread across agents and tools on purpose. A queue where
// every row is the same tool cannot show what batching does, and a queue where
// every row is pending cannot show what a decided one looks like.
//
// Depends on:
//   - AgentDefinition: the agents the runs belong to
//   - Shipment: the loads and customers the proposals name
//   - Worker: the driver the credential proposal names
var AgentActivity = Seeds.Add(seed.Seed{
	Name:        "agent_activity",
	Kind:        seed.Fixture,
	Description: "Seeds agent runs, decisions, a plan, exceptions and the trust ledger",
	After:       []*seed.Seed{AgentDefinition, Shipment, Worker},
	Run:         (&AgentActivitySeed{}).Run,
})

type activitySeedRefs struct {
	org       *tenant.Organization
	admin     *tenant.User
	dispatch  *agentdefinition.Definition
	billing   *agentdefinition.Definition
	shipments []*shipment.Shipment
	customers []*customer.Customer
	worker    *worker.Worker
	now       int64
}

func (s *AgentActivitySeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	refs, err := s.loadRefs(ctx, db, sr)
	if err != nil {
		return err
	}

	// The conversation seed writes one run of its own, so an existing
	// row is not enough to say this seed has run. Counting the
	// scheduled runs — which only this seed writes — is.

	if err = s.insertDecisions(ctx, db, refs); err != nil {
		return err
	}
	if err = s.insertPlan(ctx, db, refs); err != nil {
		return err
	}
	if err = s.insertFailures(ctx, db, refs); err != nil {
		return err
	}

	return s.insertTrust(ctx, db, refs)
}

func (s *AgentActivitySeed) loadRefs(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
) (*activitySeedRefs, error) {
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return nil, err
	}
	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return nil, fmt.Errorf("get admin user: %w", err)
	}

	refs := &activitySeedRefs{org: org, admin: admin, now: timeutils.NowUnix()}

	d := agentdefinitiondb.Definitions
	loadAgent := func(name string) (*agentdefinition.Definition, error) {
		entity, dErr := d.AllTenants().Where(
			d.OrganizationID.Eq(org.ID),
			d.BusinessUnitID.Eq(org.BusinessUnitID),
			d.Name.Eq(name),
		).First(ctx, db)
		if dErr != nil {
			return nil, fmt.Errorf("load agent %q: %w", name, dErr)
		}

		return entity, nil
	}
	if refs.dispatch, err = loadAgent(SeedAgentDispatchName); err != nil {
		return nil, err
	}
	if refs.billing, err = loadAgent(SeedAgentBillingName); err != nil {
		return nil, err
	}

	refs.shipments = make([]*shipment.Shipment, 0, 4)
	if refs.shipments, err = shipmentdb.Shipments.AllTenants().Where(
		shipmentdb.Shipments.OrganizationID.Eq(org.ID),
		shipmentdb.Shipments.BusinessUnitID.Eq(org.BusinessUnitID),
	).Order(shipmentdb.Shipments.ProNumber.Asc()).Limit(4).List(ctx, db); err != nil {
		return nil, fmt.Errorf("load shipments: %w", err)
	}
	if len(refs.shipments) < 4 {
		return nil, fmt.Errorf("need four seeded shipments: %w", errFixtureMissing)
	}

	refs.customers = make([]*customer.Customer, 0, 2)
	if refs.customers, err = customerdb.Customers.AllTenants().Where(
		customerdb.Customers.OrganizationID.Eq(org.ID),
		customerdb.Customers.BusinessUnitID.Eq(org.BusinessUnitID),
	).Order(customerdb.Customers.Name.Asc()).Limit(2).List(ctx, db); err != nil {
		return nil, fmt.Errorf("load customers: %w", err)
	}
	if len(refs.customers) < 2 {
		return nil, fmt.Errorf("need two seeded customers: %w", errFixtureMissing)
	}

	refs.worker = new(worker.Worker)
	if refs.worker, err = workerdb.Workers.AllTenants().Where(
		workerdb.Workers.OrganizationID.Eq(org.ID),
		workerdb.Workers.BusinessUnitID.Eq(org.BusinessUnitID),
	).Order(workerdb.Workers.LastName.Asc()).First(ctx, db); err != nil {
		return nil, fmt.Errorf("load worker: %w", err)
	}

	return refs, nil
}

// runSpec is one run and what it produced, so a run and its proposals are
// written together rather than matched up afterwards by id.
type runSpec struct {
	agentID     pulid.ID
	agentType   agent.Type
	trigger     agent.RunTrigger
	subjectType agent.SubjectType
	subjectID   pulid.ID
	status      agent.RunStatus
	summary     string
	startedAt   int64
	completedAt *int64
	errorText   string
	proposals   []*agent.AgentProposal
}

func (s *AgentActivitySeed) writeRun(
	ctx context.Context,
	db *keel.DB,
	refs *activitySeedRefs,
	spec runSpec,
) (*agent.AgentRun, error) {
	run := &agent.AgentRun{
		OrganizationID:    refs.org.ID,
		BusinessUnitID:    refs.org.BusinessUnitID,
		AgentDefinitionID: spec.agentID,
		AgentType:         spec.agentType,
		Trigger:           spec.trigger,
		SubjectType:       spec.subjectType,
		SubjectID:         spec.subjectID,
		Status:            spec.status,
		Summary:           spec.summary,
		ModelIdentifier:   seedActivityModel,
		PromptVersion:     "seed/v1",
		InputContextHash:  "seed-" + spec.subjectID.String(),
		StartedAt:         spec.startedAt,
		CompletedAt:       spec.completedAt,
		ErrorMessage:      spec.errorText,
	}
	if err := agentdb.AgentRuns.Create(ctx, db, run); err != nil {
		return nil, fmt.Errorf("insert agent run: %w", err)
	}

	for _, proposal := range spec.proposals {
		proposal.OrganizationID = refs.org.ID
		proposal.BusinessUnitID = refs.org.BusinessUnitID
		proposal.RunID = run.ID
	}
	if err := agentdb.AgentProposals.CreateMany(ctx, db, spec.proposals); err != nil {
		return nil, fmt.Errorf("insert proposals: %w", err)
	}

	return run, nil
}

// proposal builds one decision card. Everything a reader needs to judge it
// without opening the record — the rationale, the evidence and the tier —
// is on the row, because that is what the queue shows.
func proposal(
	tool string,
	tier agent.AutonomyTier,
	status agent.ProposalStatus,
	confidence float64,
	rationale string,
	params map[string]any,
	evidence []agentevidence.EvidenceRef,
) *agent.AgentProposal {
	return &agent.AgentProposal{
		ToolName:     tool,
		ToolParams:   params,
		Confidence:   decimal.NewFromFloat(confidence),
		Rationale:    rationale,
		Evidence:     evidence,
		AutonomyTier: tier,
		Status:       status,
		HeldBy:       []string{},
	}
}

// insertDecisions writes the queue the builder below lays out.
func (s *AgentActivitySeed) insertDecisions(
	ctx context.Context,
	db *keel.DB,
	refs *activitySeedRefs,
) error {
	for _, spec := range decisionSpecs(refs) {
		if _, err := s.writeRun(ctx, db, refs, spec); err != nil {
			return err
		}
	}

	return nil
}

// decisionSpecs lays out the queue: four runs waiting on a person, and two
// that were already decided so the history is not empty.
//
// It builds rather than writes, so what the queue will contain can be
// asserted without a database — which is the only place a seed is otherwise
// ever exercised.
//
//nolint:funlen // One literal per decision; splitting it would hide the set.
func decisionSpecs(refs *activitySeedRefs) []runSpec {
	done := func(at int64) *int64 { return &at }

	detention := proposal(
		"detention_send_notice",
		agent.TierActWithApproval,
		agent.ProposalStatusPending,
		0.88,
		"The trailer has been on the dock four hours past free time and the customer's "+
			"agreement bills detention hourly. The notice is what starts the clock they agreed to.",
		map[string]any{
			"shipmentId": refs.shipments[0].ID.String(),
			"customerId": refs.customers[0].ID.String(),
			"hours":      4,
			"subject":    "Detention accruing on " + refs.shipments[0].ProNumber,
		},
		[]agentevidence.EvidenceRef{
			{
				Type: "Shipment",
				ID:   refs.shipments[0].ID.String(),
				Note: "Arrived 09:12, free time expired 13:12",
			},
			{Type: "Customer", ID: refs.customers[0].ID.String(), Note: "Detention billed hourly"},
		},
	)
	detention.ExpiresAt = refs.now + 2*activitySeedDay
	detention.TargetResource = "shipment"
	detention.TargetID = refs.shipments[0].ID
	detention.TargetVersion = refs.shipments[0].Version

	update := proposal(
		"customer_send_email",
		agent.TierPropose,
		agent.ProposalStatusPending,
		0.79,
		"The load delivered two hours inside the window and the customer asks for a note on "+
			"every delivery. Saying so before they ask is the difference between a service "+
			"record and a complaint.",
		map[string]any{
			"shipmentId": refs.shipments[1].ID.String(),
			"customerId": refs.customers[1].ID.String(),
			"subject":    "Delivered: " + refs.shipments[1].ProNumber,
			"body": "This load delivered at 14:22, inside the appointment window. " +
				"The signed bill is attached to the shipment.",
		},
		[]agentevidence.EvidenceRef{
			{Type: "Shipment", ID: refs.shipments[1].ID.String(), Note: "Delivered 14:22"},
		},
	)
	update.ExpiresAt = refs.now + activitySeedDay

	credential := proposal(
		"worker_request_credential_renewal",
		agent.TierActWithApproval,
		agent.ProposalStatusPending,
		0.93,
		"Three papers on the same driver fall due inside thirty days. One packet now is one "+
			"conversation; three reminders later is three.",
		map[string]any{
			"workerId":    refs.worker.ID.String(),
			"credentials": []string{"Medical card", "Hazmat endorsement", "Annual review"},
			"dueWithin":   30,
		},
		[]agentevidence.EvidenceRef{
			{
				Type: "Worker",
				ID:   refs.worker.ID.String(),
				Note: "Medical card expires in 11 days",
			},
		},
	)
	credential.ExpiresAt = refs.now + 5*activitySeedDay

	// A second detention_send_notice, so the queue can show what selecting two
	// rows of the same tool and deciding them together looks like.
	secondDetention := proposal(
		"detention_send_notice",
		agent.TierActWithApproval,
		agent.ProposalStatusPending,
		0.84,
		"Same dock, same customer, a different load. The free time expired two hours ago and "+
			"nothing on the shipment says the delay was ours.",
		map[string]any{
			"shipmentId": refs.shipments[2].ID.String(),
			"customerId": refs.customers[0].ID.String(),
			"hours":      2,
			"subject":    "Detention accruing on " + refs.shipments[2].ProNumber,
		},
		[]agentevidence.EvidenceRef{
			{Type: "Shipment", ID: refs.shipments[2].ID.String(), Note: "Free time expired 15:40"},
		},
	)
	secondDetention.ExpiresAt = refs.now + 2*activitySeedDay
	secondDetention.TargetResource = "shipment"
	secondDetention.TargetID = refs.shipments[2].ID
	secondDetention.TargetVersion = refs.shipments[2].Version

	executed := proposal(
		"shipment_add_comment",
		agent.TierAutoExecute,
		agent.ProposalStatusExecuted,
		0.97,
		"The customer was told the load was running late. Writing it on the shipment is what "+
			"stops the next person telling them again.",
		map[string]any{
			"shipmentId":  refs.shipments[3].ID.String(),
			"comment":     "Customer notified of the two-hour delay at 11:05.",
			"commentType": "CustomerUpdate",
		},
		[]agentevidence.EvidenceRef{
			{Type: "Shipment", ID: refs.shipments[3].ID.String()},
		},
	)
	executed.ExecutedAt = done(refs.now - 3*activitySeedHour)

	rejected := proposal(
		"worker_place_dispatch_hold",
		agent.TierPropose,
		agent.ProposalStatusRejected,
		0.61,
		"The driver's medical card shows expired on file, so the agent asked to stop "+
			"dispatching them.",
		map[string]any{
			"workerId": refs.worker.ID.String(),
			"reason":   "Medical card expired",
		},
		[]agentevidence.EvidenceRef{
			{
				Type: "Worker",
				ID:   refs.worker.ID.String(),
				Note: "Card renewed, paperwork not filed",
			},
		},
	)

	return []runSpec{
		{
			agentID:     refs.dispatch.ID,
			agentType:   agent.TypeGeneral,
			trigger:     agent.RunTriggerEvent,
			subjectType: agent.SubjectShipment,
			subjectID:   refs.shipments[0].ID,
			status:      agent.RunStatusAwaitingDecision,
			summary:     "Detention is accruing on " + refs.shipments[0].ProNumber + ".",
			startedAt:   refs.now - 2*activitySeedHour,
			proposals:   []*agent.AgentProposal{detention},
		},
		{
			agentID:     refs.dispatch.ID,
			agentType:   agent.TypeGeneral,
			trigger:     agent.RunTriggerEvent,
			subjectType: agent.SubjectShipment,
			subjectID:   refs.shipments[2].ID,
			status:      agent.RunStatusAwaitingDecision,
			summary:     "Detention is accruing on " + refs.shipments[2].ProNumber + ".",
			startedAt:   refs.now - 90*60,
			proposals:   []*agent.AgentProposal{secondDetention},
		},
		{
			agentID:     refs.dispatch.ID,
			agentType:   agent.TypeGeneral,
			trigger:     agent.RunTriggerEvent,
			subjectType: agent.SubjectShipment,
			subjectID:   refs.shipments[1].ID,
			status:      agent.RunStatusAwaitingDecision,
			summary:     refs.shipments[1].ProNumber + " delivered inside the window.",
			startedAt:   refs.now - 4*activitySeedHour,
			proposals:   []*agent.AgentProposal{update},
		},
		{
			agentID:     refs.dispatch.ID,
			agentType:   agent.TypeGeneral,
			trigger:     agent.RunTriggerScheduled,
			subjectType: agent.SubjectWorker,
			subjectID:   refs.worker.ID,
			status:      agent.RunStatusAwaitingDecision,
			summary:     "Three credentials fall due on one driver inside thirty days.",
			startedAt:   refs.now - 6*activitySeedHour,
			proposals:   []*agent.AgentProposal{credential},
		},
		{
			agentID:     refs.dispatch.ID,
			agentType:   agent.TypeGeneral,
			trigger:     agent.RunTriggerEvent,
			subjectType: agent.SubjectShipment,
			subjectID:   refs.shipments[3].ID,
			status:      agent.RunStatusCompleted,
			summary:     "Noted the delay the customer was told about.",
			startedAt:   refs.now - 4*activitySeedHour,
			completedAt: done(refs.now - 3*activitySeedHour),
			proposals:   []*agent.AgentProposal{executed},
		},
		{
			agentID:     refs.dispatch.ID,
			agentType:   agent.TypeGeneral,
			trigger:     agent.RunTriggerScheduled,
			subjectType: agent.SubjectWorker,
			subjectID:   refs.worker.ID,
			status:      agent.RunStatusCompleted,
			summary:     "Asked to hold a driver whose card was already renewed.",
			startedAt:   refs.now - 2*activitySeedDay,
			completedAt: done(refs.now - 2*activitySeedDay + activitySeedHour),
			proposals:   []*agent.AgentProposal{rejected},
		},
	}
}

// insertPlan writes the several writes that are one approval.
//
// A plan is the case a queue of single proposals cannot show: three steps that
// only make sense together, in order, decided once. The steps are ordinary
// proposals carrying the plan's id, which is how the executor runs them.
func (s *AgentActivitySeed) insertPlan(
	ctx context.Context,
	db *keel.DB,
	refs *activitySeedRefs,
) error {
	run, err := s.writeRun(ctx, db, refs, runSpec{
		agentID:     refs.billing.ID,
		agentType:   agent.TypeBillingException,
		trigger:     agent.RunTriggerScheduled,
		subjectType: agent.SubjectShipment,
		subjectID:   refs.shipments[1].ID,
		status:      agent.RunStatusAwaitingDecision,
		summary: "Three things have to happen to " + refs.shipments[1].ProNumber +
			" before it can be invoiced.",
		startedAt: refs.now - 5*activitySeedHour,
	})
	if err != nil {
		return err
	}

	plan := &agentplan.AgentPlan{
		OrganizationID: refs.org.ID,
		BusinessUnitID: refs.org.BusinessUnitID,
		RunID:          run.ID,
		Title:          "Ready " + refs.shipments[1].ProNumber + " for invoicing",
		Summary: "The rate disagrees with the agreement, the accessorial has no code, and " +
			"nobody has asked the customer for the missing reference. Doing any one of " +
			"them alone leaves the load still unbillable.",
		Status:    agentplan.PlanStatusPending,
		StepCount: 3,
		ExpiresAt: refs.now + 3*activitySeedDay,
	}
	if err = agentdb.AgentPlans.Create(ctx, db, plan); err != nil {
		return fmt.Errorf("insert plan: %w", err)
	}

	steps := []*agent.AgentProposal{
		proposal(
			"billing_correct_charge_codes",
			agent.TierActWithApproval,
			agent.ProposalStatusPending,
			0.9,
			"The accessorial was entered without a code, so it prices at zero and the "+
				"invoice is short by the amount of it.",
			map[string]any{
				"shipmentId": refs.shipments[1].ID.String(),
				"chargeCode": "DETENTION",
			},
			[]agentevidence.EvidenceRef{{Type: "Shipment", ID: refs.shipments[1].ID.String()}},
		),
		proposal(
			"customer_send_email",
			agent.TierPropose,
			agent.ProposalStatusPending,
			0.74,
			"The customer's purchase order is not on the load and their portal will refuse "+
				"an invoice without it.",
			map[string]any{
				"shipmentId": refs.shipments[1].ID.String(),
				"customerId": refs.customers[1].ID.String(),
				"subject":    "PO number needed for " + refs.shipments[1].ProNumber,
				"body": "We are short the purchase order for this load and cannot invoice " +
					"without it. Could you send it over?",
			},
			[]agentevidence.EvidenceRef{{Type: "Customer", ID: refs.customers[1].ID.String()}},
		),
		proposal(
			"shipment_add_comment",
			agent.TierAutoExecute,
			agent.ProposalStatusPending,
			0.95,
			"Writing down what was corrected and what was asked for is what stops the next "+
				"person redoing both.",
			map[string]any{
				"shipmentId":  refs.shipments[1].ID.String(),
				"comment":     "Charge code corrected; PO requested from the customer.",
				"commentType": "Billing",
			},
			[]agentevidence.EvidenceRef{{Type: "Shipment", ID: refs.shipments[1].ID.String()}},
		),
	}

	for i, step := range steps {
		step.OrganizationID = refs.org.ID
		step.BusinessUnitID = refs.org.BusinessUnitID
		step.RunID = run.ID
		step.PlanID = &plan.ID
		step.PlanStep = i + 1
		step.ExpiresAt = plan.ExpiresAt
		if err = agentdb.AgentProposals.Create(ctx, db, step); err != nil {
			return fmt.Errorf("insert plan step %d: %w", i+1, err)
		}
	}

	return nil
}

// insertFailures writes what it looks like when a run does not finish.
//
// One run failed outright, and one finished but raised an exception: those are
// different things, and the activity tab shows them in different places.
func (s *AgentActivitySeed) insertFailures(
	ctx context.Context,
	db *keel.DB,
	refs *activitySeedRefs,
) error {
	done := func(at int64) *int64 { return &at }

	failed, err := s.writeRun(ctx, db, refs, runSpec{
		agentID:     refs.billing.ID,
		agentType:   agent.TypeBillingException,
		trigger:     agent.RunTriggerScheduled,
		subjectType: agent.SubjectShipment,
		subjectID:   refs.shipments[2].ID,
		status:      agent.RunStatusFailed,
		summary:     "Could not read the rate agreement.",
		startedAt:   refs.now - 9*activitySeedHour,
		completedAt: done(refs.now - 9*activitySeedHour + 42),
		errorText: "The provider returned no completion after four attempts: " +
			"context deadline exceeded.",
	})
	if err != nil {
		return err
	}
	_ = failed

	raised, err := s.writeRun(ctx, db, refs, runSpec{
		agentID:     refs.billing.ID,
		agentType:   agent.TypeBillingException,
		trigger:     agent.RunTriggerScheduled,
		subjectType: agent.SubjectShipment,
		subjectID:   refs.shipments[3].ID,
		status:      agent.RunStatusCompleted,
		summary:     "Stopped: the rate on the load matches no agreement on file.",
		startedAt:   refs.now - 7*activitySeedHour,
		completedAt: done(refs.now - 7*activitySeedHour + 61),
	})
	if err != nil {
		return err
	}

	exceptions := []*agentexception.AgentException{
		{
			OrganizationID: refs.org.ID,
			BusinessUnitID: refs.org.BusinessUnitID,
			RunID:          raised.ID,
			Category:       agentexception.CategoryRateNotOnFile,
			Severity:       agentexception.SeverityHigh,
			SubjectType:    agent.SubjectShipment,
			SubjectID:      refs.shipments[3].ID,
			AttemptSummary: "The load prices at $2,840 and no agreement for this customer " +
				"covers the lane. Guessing a rate is how a customer gets billed twice.",
			Evidence: []agentevidence.EvidenceRef{
				{Type: "Shipment", ID: refs.shipments[3].ID.String()},
				{Type: "Customer", ID: refs.customers[0].ID.String(), Note: "No lane agreement"},
			},
			BlastRadius:     1,
			ResolutionState: agentexception.ResolutionStateOpen,
		},
		{
			OrganizationID: refs.org.ID,
			BusinessUnitID: refs.org.BusinessUnitID,
			RunID:          raised.ID,
			Category:       agentexception.CategoryMissingBOL,
			Severity:       agentexception.SeverityMedium,
			SubjectType:    agent.SubjectShipment,
			SubjectID:      refs.shipments[2].ID,
			AttemptSummary: "No signed bill is attached, and the customer will not pay an " +
				"invoice without one.",
			Evidence: []agentevidence.EvidenceRef{
				{Type: "Shipment", ID: refs.shipments[2].ID.String()},
			},
			BlastRadius:     1,
			ResolutionState: agentexception.ResolutionStateResolved,
			ResolutionNotes: "Driver sent the photo; billing attached it by hand.",
		},
	}
	for _, exception := range exceptions {
		if err = agentdb.AgentExceptions.Create(ctx, db, exception); err != nil {
			return fmt.Errorf("insert exception %s: %w", exception.Category, err)
		}
	}

	return nil
}

// insertTrust writes the ledger the trust ladder and the scorecard read.
//
// One tool is most of the way to its next tier, one has just been promoted,
// and one was demoted after a failure — so the panel has a promotion to
// explain rather than an empty table. The scorecard itself is computed from
// these rows and the runs above; there is no scorecard table to seed.
func (s *AgentActivitySeed) insertTrust(
	ctx context.Context,
	db *keel.DB,
	refs *activitySeedRefs,
) error {
	rows := trustRows(refs)
	if err := agentdb.ToolTrusts.CreateMany(ctx, db, rows); err != nil {
		return fmt.Errorf("insert tool trust: %w", err)
	}

	return nil
}

func trustRows(refs *activitySeedRefs) []*agenttooltrust.ToolTrust {
	at := func(offset int64) *int64 { value := refs.now - offset; return &value }

	return []*agenttooltrust.ToolTrust{
		{
			OrganizationID:    refs.org.ID,
			BusinessUnitID:    refs.org.BusinessUnitID,
			AgentDefinitionID: refs.dispatch.ID,
			ToolName:          "shipment_add_comment",
			Streak:            14,
			Approvals:         16,
			Modifications:     1,
			Rejections:        0,
			EarnedTier:        agent.TierAutoExecute,
			LastDecisionAt:    at(3 * activitySeedHour),
			PromotedAt:        at(6 * activitySeedDay),
		},
		{
			OrganizationID:    refs.org.ID,
			BusinessUnitID:    refs.org.BusinessUnitID,
			AgentDefinitionID: refs.dispatch.ID,
			ToolName:          "detention_send_notice",
			Streak:            7,
			Approvals:         9,
			Modifications:     2,
			Rejections:        1,
			EarnedTier:        agent.TierActWithApproval,
			LastDecisionAt:    at(activitySeedDay),
			PromotedAt:        at(11 * activitySeedDay),
		},
		{
			OrganizationID:    refs.org.ID,
			BusinessUnitID:    refs.org.BusinessUnitID,
			AgentDefinitionID: refs.dispatch.ID,
			ToolName:          "customer_send_email",
			Streak:            3,
			Approvals:         5,
			Modifications:     4,
			Rejections:        1,
			LastDecisionAt:    at(2 * activitySeedDay),
		},
		{
			OrganizationID:    refs.org.ID,
			BusinessUnitID:    refs.org.BusinessUnitID,
			AgentDefinitionID: refs.dispatch.ID,
			ToolName:          "worker_place_dispatch_hold",
			Streak:            0,
			Approvals:         2,
			Modifications:     0,
			Rejections:        2,
			ExecutionFailures: 1,
			EarnedTier:        agent.TierPropose,
			LastDecisionAt:    at(2 * activitySeedDay),
			DemotedAt:         at(2 * activitySeedDay),
		},
		{
			OrganizationID:    refs.org.ID,
			BusinessUnitID:    refs.org.BusinessUnitID,
			AgentDefinitionID: refs.billing.ID,
			ToolName:          "billing_correct_charge_codes",
			Streak:            5,
			Approvals:         6,
			Modifications:     1,
			Rejections:        0,
			EarnedTier:        agent.TierActWithApproval,
			LastDecisionAt:    at(5 * activitySeedHour),
			PromotedAt:        at(4 * activitySeedDay),
		},
	}
}
