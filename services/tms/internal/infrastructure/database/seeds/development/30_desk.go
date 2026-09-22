package development

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
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
// The watchtower items for inbound mail are built with the same describer the
// live projection uses, rather than written by hand. That is the point of
// having one: a seeded row and a projected row are the same row, so what a
// developer sees on the feed is what a real message will look like there.
//
// The rest of the feed is written directly, because the records those kinds
// project from are not all seeded and an item standing in for nothing would
// resolve itself the first time the nightly reconcile ran. Those carry no
// subject, so nothing offers to hand them to an agent.
//
// The briefing is deterministic and un-narrated, which is exactly what a
// briefing looks like on an installation with no provider reachable — a
// complete page in its own wording.
//
// Depends on:
//   - InboundMessage: the mail the inbox items stand for
func NewDeskSeed() *DeskSeed {
	seed := &DeskSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"Desk",
		"1.0.0",
		"Seeds the watchtower feed and today's briefing",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedInboundMessage)

	return seed
}

type deskSeedRefs struct {
	org      *tenant.Organization
	messages []*inboundmessage.InboundMessage
	now      int64
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

	return refs, nil
}

func (s *DeskSeed) insertItems(
	ctx context.Context,
	tx bun.Tx,
	refs *deskSeedRefs,
) error {
	items := make([]*watchtower.Item, 0, len(refs.messages)+4)

	for _, message := range refs.messages {
		input := watchtowersources.DescribeInboundMessage(message)
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
	today := time.Unix(refs.now, 0).UTC().Format(time.DateOnly)

	cols := buncolgen.BriefingColumns
	count, err := tx.NewSelect().
		Model((*briefing.Briefing)(nil)).
		Where(cols.OrganizationID.Eq(), refs.org.ID).
		Where(cols.BusinessUnitID.Eq(), refs.org.BusinessUnitID).
		Where(cols.BriefingDate.Eq(), today).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("count existing briefings: %w", err)
	}
	if count > 0 {
		return nil
	}

	waiting := len(refs.messages)
	entity := &briefing.Briefing{
		ID:             pulid.MustNew("brf_"),
		OrganizationID: refs.org.ID,
		BusinessUnitID: refs.org.BusinessUnitID,
		RoleKey:        briefing.RoleDispatch,
		BriefingDate:   today,
		Status:         briefing.StatusReady,
		Headline: fmt.Sprintf(
			"14 pickups and 11 deliveries today, 2 moves uncovered, %d messages waiting.",
			waiting,
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
					"%d messages are waiting on a person.", waiting,
				),
				Items: []briefing.Item{
					{Label: "Messages in review", Value: fmt.Sprint(waiting), Path: "/inbox"},
					{Label: "Proposals pending", Value: "1", Path: "/desk/decisions"},
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
			"proposalsPending":    1,
			"expiringCredentials": 3,
			"seeded":              true,
		},
	}

	if _, err = tx.NewInsert().Model(entity).Exec(ctx); err != nil {
		return fmt.Errorf("insert briefing: %w", err)
	}

	return nil
}
