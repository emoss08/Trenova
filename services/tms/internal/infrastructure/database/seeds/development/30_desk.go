package development

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
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
// A handful of rows are written directly, for the kinds whose sources are not
// seeded at all — hours of service, weather, a credential sweep. Those carry
// no subject on purpose: offering to hand one to an agent would be offering
// to work on nothing.
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
	org        *tenant.Organization
	messages   []*inboundmessage.InboundMessage
	proposals  []*agent.AgentProposal
	plans      []*agent.AgentPlan
	failedRuns []*agent.AgentRun
	exceptions []*agent.AgentException
	insights   []*insight.Insight
	agentNames map[pulid.ID]string
	now        int64
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

			cols := buncolgen.ItemColumns
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

	cols := buncolgen.InboundMessageColumns
	refs.messages = make([]*inboundmessage.InboundMessage, 0, 4)
	if err = tx.NewSelect().
		Model(&refs.messages).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.Status.In(), bun.In([]inboundmessage.Status{
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

	return refs, nil
}

// loadAgentWork reads the records the tower stands in for.
//
// The items are built from these rather than written by hand, which is what
// makes "Hand off" and "Ask about this" do anything: both need the subject of
// a real record, and an invented one has none.
func (r *deskSeedRefs) loadAgentWork(ctx context.Context, tx bun.Tx) error {
	proposalCols := buncolgen.AgentProposalColumns
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

	planCols := buncolgen.AgentPlanColumns
	r.plans = make([]*agent.AgentPlan, 0, 2)
	if err := tx.NewSelect().
		Model(&r.plans).
		Where(planCols.OrganizationID.Eq(), r.org.ID).
		Where(planCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Where(planCols.Status.Eq(), agent.PlanStatusPending).
		Scan(ctx); err != nil {
		return fmt.Errorf("load pending plans: %w", err)
	}

	runCols := buncolgen.AgentRunColumns
	r.failedRuns = make([]*agent.AgentRun, 0, 2)
	if err := tx.NewSelect().
		Model(&r.failedRuns).
		Where(runCols.OrganizationID.Eq(), r.org.ID).
		Where(runCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Where(runCols.Status.Eq(), agent.RunStatusFailed).
		Scan(ctx); err != nil {
		return fmt.Errorf("load failed runs: %w", err)
	}

	exceptionCols := buncolgen.AgentExceptionColumns
	r.exceptions = make([]*agent.AgentException, 0, 2)
	if err := tx.NewSelect().
		Model(&r.exceptions).
		Where(exceptionCols.OrganizationID.Eq(), r.org.ID).
		Where(exceptionCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Where(exceptionCols.ResolutionState.Eq(), agent.ResolutionStateOpen).
		Scan(ctx); err != nil {
		return fmt.Errorf("load open exceptions: %w", err)
	}

	insightCols := buncolgen.InsightColumns
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
	defCols := buncolgen.DefinitionColumns
	if err := tx.NewSelect().
		Model(&definitions).
		Where(defCols.OrganizationID.Eq(), r.org.ID).
		Where(defCols.BusinessUnitID.Eq(), r.org.BusinessUnitID).
		Scan(ctx); err != nil {
		return fmt.Errorf("load agent definitions: %w", err)
	}

	byDefinition := make(map[pulid.ID]string, len(definitions))
	for _, definition := range definitions {
		byDefinition[definition.ID] = definition.Name
	}

	runs := make([]*agent.AgentRun, 0, 16)
	runCols := buncolgen.AgentRunColumns
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
	inputs := make([]services.WatchtowerItemInput, 0, 24)
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

	items := make([]*watchtower.Item, 0, len(inputs)+5)
	for _, input := range inputs {
		items = append(items, &watchtower.Item{
			ID:             pulid.MustNew("wt_"),
			OrganizationID: refs.org.ID,
			BusinessUnitID: refs.org.BusinessUnitID,
			SourceKind:     input.SourceKind,
			SourceID:       input.SourceID,
			Severity:       input.Severity,
			Title:          input.Title,
			Summary:        input.Summary,
			SubjectType:    input.SubjectType,
			SubjectID:      input.SubjectID,
			EventKind:      input.EventKind,
			Path:           input.Path,
			OccurredAt:     input.OccurredAt,
		})
	}

	items = append(items, s.standingItems(refs)...)

	if _, err := tx.NewInsert().Model(&items).Exec(ctx); err != nil {
		return fmt.Errorf("insert watchtower items: %w", err)
	}

	return nil
}

// standingItems round out the feed so every severity and several kinds are on
// it. They deliberately carry no subject: the records they stand for are not
// seeded, so offering to hand one to an agent would offer to work on nothing.
func (s *DeskSeed) standingItems(refs *deskSeedRefs) []*watchtower.Item {
	resolved := refs.now - 2*deskSeedHour
	build := func(
		kind watchtower.SourceKind,
		sourceID string,
		severity watchtower.Severity,
		title string,
		summary string,
		occurredAt int64,
	) *watchtower.Item {
		return &watchtower.Item{
			ID:             pulid.MustNew("wt_"),
			OrganizationID: refs.org.ID,
			BusinessUnitID: refs.org.BusinessUnitID,
			SourceKind:     kind,
			SourceID:       sourceID,
			Severity:       severity,
			Title:          title,
			Summary:        summary,
			OccurredAt:     occurredAt,
		}
	}

	closed := build(
		watchtower.SourceServiceFailure,
		"seed-watchtower-service-failure",
		watchtower.SeverityWarning,
		"Service failure closed: late delivery on the Columbus lane",
		"Root cause recorded as receiver congestion; no charge raised.",
		refs.now-deskSeedDay,
	)
	closed.ResolvedAt = &resolved

	return []*watchtower.Item{
		build(
			watchtower.SourceHOSViolation,
			"seed-watchtower-hos",
			watchtower.SeverityCritical,
			"Hours of service: driver over the 14-hour window",
			"The clock ran out 22 minutes before the appointment; the move needs reassigning.",
			refs.now-45*60,
		),
		build(
			watchtower.SourceWorkerCredential,
			"seed-watchtower-credential",
			watchtower.SeverityWarning,
			"Three credentials expiring for one driver",
			"Medical card, hazmat endorsement and annual review all fall inside 30 days.",
			refs.now-5*deskSeedHour,
		),
		build(
			watchtower.SourceBillingException,
			"seed-watchtower-billing",
			watchtower.SeverityWarning,
			"Billing blocked: accessorial has no rate",
			"A detention charge was added by hand and matches no agreement line.",
			refs.now-7*deskSeedHour,
		),
		build(
			watchtower.SourceWeatherAlert,
			"seed-watchtower-weather",
			watchtower.SeverityInfo,
			"Winter storm warning across I-80 in Nebraska",
			"Two moves are routed through the warned counties tomorrow morning.",
			refs.now-deskSeedHour,
		),
		closed,
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

	cols := buncolgen.BriefingColumns
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
