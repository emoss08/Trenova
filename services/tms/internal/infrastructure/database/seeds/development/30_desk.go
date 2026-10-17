package development

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/buncolgen/agentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/agentdefinitioncol"
	"github.com/emoss08/trenova/pkg/buncolgen/briefingcol"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/insightcol"
	"github.com/emoss08/trenova/pkg/buncolgen/watchtowercol"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowerservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	deskSeedHour = int64(3600)
	deskSeedDay  = 24 * deskSeedHour
)

type DeskSeed struct {
	seedhelpers.BaseSeed
}

// DeskSeed fills the Desk's two reading surfaces: the watchtower feed and
// this morning's briefing.
//
// Most of the feed is built with the same describers the live projection
// uses, over records that really exist — inbound mail, insights, the
// proposals and plans waiting in the queue, the run that failed and the
// exceptions it raised. That is the point of having describers: a seeded row
// and a projected row are the same row, so the nightly reconcile leaves them
// alone, "Ask about this" opens a thread on a real subject, and "Hand off"
// has something to hand off.
//
// The queue the Watchtower page works through is written directly: an item in
// each deadline lane, agents working, waiting on a person and done, an overdue
// detention clock, items with a trail and items with no deadline. Those about
// a seeded shipment or driver carry it as their subject, so they can be handed
// off; the rest carry none, because offering to hand one to an agent would be
// offering to work on nothing.
//
// The briefing is deterministic and un-narrated, which is exactly what a
// briefing looks like on an installation with no provider reachable — a
// complete page in its own wording.
//
// Depends on:
//   - InboundMessage: the mail the inbox items stand for
//   - AgentActivity: the decisions, failures and exceptions the feed carries
//   - Insight: the findings the feed carries
func NewDeskSeed() *DeskSeed {
	seed := &DeskSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"Desk",
		"1.0.0",
		"Seeds the watchtower feed and today's briefing",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(
		seedhelpers.SeedInboundMessage,
		seedhelpers.SeedAgentActivity,
		seedhelpers.SeedInsight,
	)

	return seed
}

type deskSeedRefs struct {
	org         *tenant.Organization
	readerID    pulid.ID
	messages    []*inboundmessage.InboundMessage
	proposals   []*agent.AgentProposal
	plans       []*agent.AgentPlan
	failedRuns  []*agent.AgentRun
	exceptions  []*agent.AgentException
	insights    []*insight.Insight
	agentNames  map[pulid.ID]string
	definitions map[agentdefinition.Template]*agentdefinition.Definition
	shipments   map[string]*shipment.Shipment
	workers     []*worker.Worker
	now         int64
}

func (s *DeskSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			refs, err := s.loadRefs(ctx, tx, sc)
			if err != nil {
				return err
			}

			cols := watchtowercol.ItemColumns
			count, err := tx.NewSelect().
				Model((*watchtower.Item)(nil)).
				Where(cols.OrganizationID.Eq(), refs.org.ID).
				Where(cols.BusinessUnitID.Eq(), refs.org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("count existing watchtower items: %w", err)
			}
			if count == 0 {
				if err = s.insertItems(ctx, tx, refs); err != nil {
					return err
				}
			}

			return s.insertBriefing(ctx, tx, refs)
		},
	)
}

func (s *DeskSeed) loadRefs(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
) (*deskSeedRefs, error) {
	org, err := sc.GetDefaultOrganization(ctx)
	if err != nil {
		return nil, err
	}

	refs := &deskSeedRefs{org: org, now: timeutils.NowUnix()}
	if admin, adminErr := sc.GetUserByUsername(ctx, "admin"); adminErr == nil {
		refs.readerID = admin.ID
	}

	cols := inboundmessagecol.InboundMessageColumns
	refs.messages = make([]*inboundmessage.InboundMessage, 0, 4)
	if err = tx.NewSelect().
		Model(&refs.messages).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.Status.In(), bun.List([]inboundmessage.Status{
			inboundmessage.StatusInReview,
			inboundmessage.StatusQuarantined,
		})).
		Order(cols.ReceivedAt.OrderDesc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load inbound messages awaiting review: %w", err)
	}

	if err = refs.loadAgentWork(ctx, tx); err != nil {
		return nil, err
	}
	if err = refs.loadQueueRefs(ctx, tx); err != nil {
		return nil, err
	}

	return refs, nil
}

// loadAgentWork reads the records the tower stands in for.
//
// The items are built from these rather than written by hand, which is what
// makes "Hand off" and "Ask about this" do anything: both need the subject of
// a real record, and an invented one has none.
func (r *deskSeedRefs) loadAgentWork(ctx context.Context, tx bun.Tx) error {
	proposalCols := agentcol.AgentProposalColumns
	r.proposals = make([]*agent.AgentProposal, 0, 4)
	if err := tx.NewSelect().
		Model(&r.proposals).
		Where(proposalCols.OrganizationID.Eq(), r.org.ID).
		Where(proposalCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Where(proposalCols.Status.Eq(), agent.ProposalStatusPending).
		Where(proposalCols.PlanID.IsNull()).
		Order(proposalCols.CreatedAt.OrderDesc()).
		Scan(ctx); err != nil {
		return fmt.Errorf("load pending proposals: %w", err)
	}

	planCols := agentcol.AgentPlanColumns
	r.plans = make([]*agent.AgentPlan, 0, 2)
	if err := tx.NewSelect().
		Model(&r.plans).
		Where(planCols.OrganizationID.Eq(), r.org.ID).
		Where(planCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Where(planCols.Status.Eq(), agent.PlanStatusPending).
		Scan(ctx); err != nil {
		return fmt.Errorf("load pending plans: %w", err)
	}

	runCols := agentcol.AgentRunColumns
	r.failedRuns = make([]*agent.AgentRun, 0, 2)
	if err := tx.NewSelect().
		Model(&r.failedRuns).
		Where(runCols.OrganizationID.Eq(), r.org.ID).
		Where(runCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Where(runCols.Status.Eq(), agent.RunStatusFailed).
		Scan(ctx); err != nil {
		return fmt.Errorf("load failed runs: %w", err)
	}

	exceptionCols := agentcol.AgentExceptionColumns
	r.exceptions = make([]*agent.AgentException, 0, 2)
	if err := tx.NewSelect().
		Model(&r.exceptions).
		Where(exceptionCols.OrganizationID.Eq(), r.org.ID).
		Where(exceptionCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Where(exceptionCols.ResolutionState.Eq(), agent.ResolutionStateOpen).
		Scan(ctx); err != nil {
		return fmt.Errorf("load open exceptions: %w", err)
	}

	insightCols := insightcol.InsightColumns
	r.insights = make([]*insight.Insight, 0, 5)
	if err := tx.NewSelect().
		Model(&r.insights).
		Where(insightCols.OrganizationID.Eq(), r.org.ID).
		Where(insightCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Where(insightCols.Status.Eq(), insight.StatusActive).
		Order(insightCols.DetectedAt.OrderDesc()).
		Limit(5).
		Scan(ctx); err != nil {
		return fmt.Errorf("load active insights: %w", err)
	}

	return r.loadAgentNames(ctx, tx)
}

// loadAgentNames is what lets a tower row say which desk asked. The proposal
// carries only its run, and the run only its definition, so the names are read
// once here rather than per row.
func (r *deskSeedRefs) loadAgentNames(ctx context.Context, tx bun.Tx) error {
	definitions := make([]*agentdefinition.Definition, 0, 8)
	defCols := agentdefinitioncol.DefinitionColumns
	if err := tx.NewSelect().
		Model(&definitions).
		Where(defCols.OrganizationID.Eq(), r.org.ID).
		Where(defCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Scan(ctx); err != nil {
		return fmt.Errorf("load agent definitions: %w", err)
	}

	byDefinition := make(map[pulid.ID]string, len(definitions))
	r.definitions = make(map[agentdefinition.Template]*agentdefinition.Definition, len(definitions))
	for _, definition := range definitions {
		byDefinition[definition.ID] = definition.Name
		if definition.Template != "" && definition.Enabled {
			if _, taken := r.definitions[definition.Template]; !taken {
				r.definitions[definition.Template] = definition
			}
		}
	}

	runs := make([]*agent.AgentRun, 0, 16)
	runCols := agentcol.AgentRunColumns
	if err := tx.NewSelect().
		Model(&runs).
		Where(runCols.OrganizationID.Eq(), r.org.ID).
		Where(runCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Scan(ctx); err != nil {
		return fmt.Errorf("load runs for agent names: %w", err)
	}

	r.agentNames = make(map[pulid.ID]string, len(runs))
	for _, run := range runs {
		if name, ok := byDefinition[run.AgentDefinitionID]; ok {
			r.agentNames[run.ID] = name
		}
	}

	return nil
}

func (s *DeskSeed) insertItems(
	ctx context.Context,
	tx bun.Tx,
	refs *deskSeedRefs,
) error {
	inputs := make([]watchtowerports.WatchtowerItemInput, 0, 24)
	for _, message := range refs.messages {
		inputs = append(inputs, watchtowersources.DescribeInboundMessage(message))
	}
	for _, entity := range refs.insights {
		inputs = append(inputs, watchtowersources.DescribeInsight(entity))
	}
	for _, entity := range refs.proposals {
		inputs = append(
			inputs,
			watchtowersources.DescribeProposal(entity, refs.agentNames[entity.RunID]),
		)
	}
	for _, entity := range refs.plans {
		inputs = append(
			inputs,
			watchtowersources.DescribePlan(entity, refs.agentNames[entity.RunID]),
		)
	}
	for _, entity := range refs.failedRuns {
		inputs = append(
			inputs,
			watchtowersources.DescribeFailedRun(entity, refs.agentNames[entity.ID]),
		)
	}
	for _, entity := range refs.exceptions {
		inputs = append(inputs, watchtowersources.DescribeException(entity))
	}

	policy := (*dispatchcontrol.DispatchControl)(nil).ExceptionPolicy()
	queue := s.queueItems(refs, refs.readerID)
	items := make([]*watchtower.Item, 0, len(inputs)+len(queue.items)+1)
	for _, input := range inputs {
		input.TenantInfo = pagination.TenantInfo{OrgID: refs.org.ID, BuID: refs.org.BusinessUnitID}
		item := watchtowerservice.ItemFromInput(input)
		item.ID = pulid.MustNew("wt_")
		item.Status = watchtower.StatusOpen
		if item.DueAt == nil {
			due := watchtowerservice.DueAt(policy, item.Severity, refs.now)
			item.DueAt = &due
			item.DueSource = watchtower.DueSourcePolicy
		}
		items = append(items, item)
	}
	items = append(items, queue.items...)
	items = append(items, s.standingItems(refs)...)

	if len(queue.runs) > 0 {
		if _, err := tx.NewInsert().Model(&queue.runs).Exec(ctx); err != nil {
			return fmt.Errorf("insert watchtower agent runs: %w", err)
		}
	}
	if _, err := tx.NewInsert().Model(&items).Exec(ctx); err != nil {
		return fmt.Errorf("insert watchtower items: %w", err)
	}
	if len(queue.reads) > 0 {
		if _, err := tx.NewInsert().Model(&queue.reads).Exec(ctx); err != nil {
			return fmt.Errorf("insert watchtower item reads: %w", err)
		}
	}

	return nil
}

func (s *DeskSeed) standingItems(refs *deskSeedRefs) []*watchtower.Item {
	resolved := refs.now - 2*deskSeedHour

	return []*watchtower.Item{
		{
			ID:             pulid.MustNew("wt_"),
			OrganizationID: refs.org.ID,
			BusinessUnitID: refs.org.BusinessUnitID,
			SourceKind:     watchtower.SourceServiceFailure,
			SourceID:       "seed-watchtower-service-failure",
			Severity:       watchtower.SeverityWarning,
			Title:          "Service failure closed: late delivery on the Columbus lane",
			Summary:        "Root cause recorded as receiver congestion; no charge raised.",
			OccurredAt:     refs.now - deskSeedDay,
			ResolvedAt:     &resolved,
			Status:         watchtower.StatusResolved,
			ResolutionCode: watchtower.ResolutionCleared,
		},
	}
}

// insertBriefing writes this morning's page, un-narrated.
//
// Every figure here is written rather than gathered, which is why the briefing
// is marked as not narrated: the guard that checks a model's prose against the
// facts has nothing to check, and a page claiming to be narrated when no model
// wrote it would be the one lie this table must not tell.
func (s *DeskSeed) insertBriefing(
	ctx context.Context,
	tx bun.Tx,
	refs *deskSeedRefs,
) error {
	entity := briefingFor(refs)

	cols := briefingcol.BriefingColumns
	count, err := tx.NewSelect().
		Model((*briefing.Briefing)(nil)).
		Where(cols.OrganizationID.Eq(), refs.org.ID).
		Where(cols.BusinessUnitID.Eq(), refs.org.BusinessUnitID).
		Where(cols.BriefingDate.Eq(), entity.BriefingDate).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("count existing briefings: %w", err)
	}
	if count > 0 {
		return nil
	}

	if _, err = tx.NewInsert().Model(entity).Exec(ctx); err != nil {
		return fmt.Errorf("insert briefing: %w", err)
	}

	return nil
}

// briefingFor lays out the morning page. It builds rather than writes, so the
// figures and the sections can be checked without a database.
func briefingFor(refs *deskSeedRefs) *briefing.Briefing {
	today := time.Unix(refs.now, 0).UTC().Format(time.DateOnly)
	waiting := len(refs.messages)
	decisions := len(refs.proposals) + len(refs.plans)

	return &briefing.Briefing{
		ID:             pulid.MustNew("brf_"),
		OrganizationID: refs.org.ID,
		BusinessUnitID: refs.org.BusinessUnitID,
		RoleKey:        briefing.RoleDispatch,
		BriefingDate:   today,
		Status:         briefing.StatusReady,
		Headline: fmt.Sprintf(
			"14 pickups and 11 deliveries today, 2 moves uncovered, "+
				"%d messages and %d decisions waiting.",
			waiting, decisions,
		),
		Narrated: false,
		Sections: []briefing.Section{
			{
				Key:     briefing.SectionToday,
				Title:   "Today",
				Summary: "14 pickups and 11 deliveries are scheduled.",
				Items: []briefing.Item{
					{Label: "Pickups", Value: "14", Path: "/dispatch/console"},
					{Label: "Deliveries", Value: "11", Path: "/dispatch/console"},
					{Label: "In transit", Value: "9", Path: "/shipment-management/shipments"},
				},
				Path: "/dispatch/console",
			},
			{
				Key:     briefing.SectionCoverage,
				Title:   "Coverage",
				Summary: "2 moves inside the planning window have no driver.",
				Items: []briefing.Item{
					{Label: "Uncovered moves", Value: "2", Path: "/dispatch/console"},
					{Label: "Drivers available", Value: "6", Path: "/hr/workers"},
				},
				Path: "/dispatch/console",
			},
			{
				Key:     briefing.SectionExceptions,
				Title:   "Needs attention",
				Summary: "1 hours-of-service violation and 1 open service failure.",
				Items: []briefing.Item{
					{Label: "HOS violations", Value: "1", Path: "/desk/watchtower"},
					{
						Label: "Open service failures",
						Value: "1",
						Path:  "/shipment-management/service-failures",
					},
				},
				Path: "/desk/watchtower",
			},
			{
				Key:   briefing.SectionDecisions,
				Title: "Waiting on you",
				Summary: fmt.Sprintf(
					"%d messages and %d decisions are waiting on a person.",
					waiting, decisions,
				),
				Items: []briefing.Item{
					{Label: "Messages in review", Value: fmt.Sprint(waiting), Path: "/inbox"},
					{
						Label: "Decisions pending",
						Value: fmt.Sprint(decisions),
						Path:  "/desk/decisions",
					},
				},
				Path: "/desk/decisions",
			},
			{
				Key:     briefing.SectionCompliance,
				Title:   "Compliance",
				Summary: "3 credentials expire within 30 days, all on one driver.",
				Items: []briefing.Item{
					{Label: "Expiring credentials", Value: "3", Path: "/hr/workers"},
				},
				Path: "/hr/workers",
			},
		},
		Facts: map[string]any{
			"pickupsToday":        14,
			"deliveriesToday":     11,
			"inTransit":           9,
			"uncoveredMoves":      2,
			"driversAvailable":    6,
			"hosViolations":       1,
			"openServiceFailures": 1,
			"messagesInReview":    waiting,
			"decisionsPending":    decisions,
			"expiringCredentials": 3,
			"seeded":              true,
		},
	}
}
