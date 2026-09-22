package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
)

type AssistantArtifactSeed struct {
	seedhelpers.BaseSeed
}

// AssistantArtifactSeed fills the Desk's artifacts pane.
//
// An artifact is the product of a turn — a table, a draft, a plan, a ledger —
// and the pane is the half of the Desk that is not conversation. Empty, it
// reads as a feature that does not work; these give it one of each kind it
// can draw, so the renderers, the pin and the download can all be tried.
//
// Every payload is written in the shape the pane's readers expect, because a
// payload that parses to nothing renders as an empty card and would teach a
// developer the wrong thing about what the kind looks like.
//
// run_diff is seeded on Postgres only. The SQLite schema's kind CHECK
// predates that kind — the dialect converter never emits a CHECK alteration —
// so a run_diff row there would fail the whole seed run.
//
// Depends on:
//   - AssistantConversation: the threads the artifacts hang off
//   - AgentActivity: the plan an artifact is a view over
func NewAssistantArtifactSeed() *AssistantArtifactSeed {
	seed := &AssistantArtifactSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"AssistantArtifact",
		"1.0.0",
		"Seeds Desk artifacts of every kind the pane can render",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(
		seedhelpers.SeedAssistantConversation,
		seedhelpers.SeedAgentActivity,
	)

	return seed
}

type artifactSeedRefs struct {
	org       *tenant.Organization
	threads   []*conversation.Thread
	shipments []*shipment.Shipment
	plan      *agent.AgentPlan
	steps     []*agent.AgentProposal
	// draft is the proposal the email artifact is a view over. An email draft
	// without one is refused by the domain, and rightly: a draft that is not
	// tied to the decision it belongs to is a second copy of that decision.
	draft *agent.AgentProposal
	now   int64
	// includeRunDiff is whether the schema accepts a run_diff row. SQLite's
	// kind CHECK predates the kind, so it is seeded on Postgres only.
	includeRunDiff bool
}

func (s *AssistantArtifactSeed) Run(ctx context.Context, tx bun.Tx) error {
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

			cols := buncolgen.ArtifactColumns
			count, err := tx.NewSelect().
				Model((*assistantartifact.Artifact)(nil)).
				Where(cols.OrganizationID.Eq(), refs.org.ID).
				Where(cols.BusinessUnitID.Eq(), refs.org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("count existing artifacts: %w", err)
			}
			if count > 0 {
				return s.backfillRunDiff(ctx, tx, refs)
			}

			artifacts := s.artifacts(refs)
			if _, err = tx.NewInsert().Model(&artifacts).Exec(ctx); err != nil {
				return fmt.Errorf("insert artifacts: %w", err)
			}

			return nil
		},
	)
}

// backfillRunDiff adds the run_diff artifact to a database seeded before the
// kind existed, so re-running the seed shows it without a reset. The rest of
// the artifacts are left as they are.
func (s *AssistantArtifactSeed) backfillRunDiff(
	ctx context.Context,
	tx bun.Tx,
	refs *artifactSeedRefs,
) error {
	if !refs.includeRunDiff {
		return nil
	}

	cols := buncolgen.ArtifactColumns
	exists, err := tx.NewSelect().
		Model((*assistantartifact.Artifact)(nil)).
		Where(cols.OrganizationID.Eq(), refs.org.ID).
		Where(cols.BusinessUnitID.Eq(), refs.org.BusinessUnitID).
		Where(cols.Kind.Eq(), assistantartifact.KindRunDiff).
		Exists(ctx)
	if err != nil {
		return fmt.Errorf("look for a seeded run diff: %w", err)
	}
	if exists {
		return nil
	}

	for _, artifact := range s.artifacts(refs) {
		if artifact.Kind != assistantartifact.KindRunDiff {
			continue
		}
		if _, err = tx.NewInsert().Model(artifact).Exec(ctx); err != nil {
			return fmt.Errorf("insert run diff artifact: %w", err)
		}
	}

	return nil
}

func (s *AssistantArtifactSeed) loadRefs(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
) (*artifactSeedRefs, error) {
	org, err := sc.GetDefaultOrganization(ctx)
	if err != nil {
		return nil, err
	}

	refs := &artifactSeedRefs{
		org:            org,
		now:            timeutils.NowUnix(),
		includeRunDiff: tx.Dialect().Name() == dialect.PG,
	}

	threadCols := buncolgen.ThreadColumns
	refs.threads = make([]*conversation.Thread, 0, 2)
	if err = tx.NewSelect().
		Model(&refs.threads).
		Where(threadCols.OrganizationID.Eq(), org.ID).
		Where(threadCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Order(threadCols.CreatedAt.OrderAsc()).
		Limit(2).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load threads: %w", err)
	}
	if len(refs.threads) == 0 {
		return nil, fmt.Errorf("need a seeded thread: %w", seedhelpers.ErrEntityNotFound)
	}

	shipmentCols := buncolgen.ShipmentColumns
	refs.shipments = make([]*shipment.Shipment, 0, 3)
	if err = tx.NewSelect().
		Model(&refs.shipments).
		Where(shipmentCols.OrganizationID.Eq(), org.ID).
		Where(shipmentCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Order(shipmentCols.ProNumber.OrderAsc()).
		Limit(3).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load shipments: %w", err)
	}
	if len(refs.shipments) < 3 {
		return nil, fmt.Errorf("need three seeded shipments: %w", seedhelpers.ErrEntityNotFound)
	}

	planCols := buncolgen.AgentPlanColumns
	refs.plan = new(agent.AgentPlan)
	if err = tx.NewSelect().
		Model(refs.plan).
		Where(planCols.OrganizationID.Eq(), org.ID).
		Where(planCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load plan: %w", err)
	}

	proposalCols := buncolgen.AgentProposalColumns
	refs.steps = make([]*agent.AgentProposal, 0, 3)
	if err = tx.NewSelect().
		Model(&refs.steps).
		Where(proposalCols.OrganizationID.Eq(), org.ID).
		Where(proposalCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(proposalCols.PlanID.Eq(), refs.plan.ID).
		Order(proposalCols.PlanStep.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load plan steps: %w", err)
	}

	refs.draft = new(agent.AgentProposal)
	if err = tx.NewSelect().
		Model(refs.draft).
		Where(proposalCols.OrganizationID.Eq(), org.ID).
		Where(proposalCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(proposalCols.ToolName.Eq(), "send_detention_notice").
		Where(proposalCols.Status.Eq(), agent.ProposalStatusPending).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load the proposal behind the email draft: %w", err)
	}

	return refs, nil
}

// thread picks which conversation an artifact hangs off. The pane is scoped
// to one thread, so spreading them over two shows both a full pane and the
// switch between them.
func (r *artifactSeedRefs) thread(index int) pulid.ID {
	if index < len(r.threads) {
		return r.threads[index].ID
	}

	return r.threads[0].ID
}

//nolint:funlen // One literal per artifact kind; splitting hides the set.
func (s *AssistantArtifactSeed) artifacts(
	refs *artifactSeedRefs,
) []*assistantartifact.Artifact {
	build := func(
		thread pulid.ID,
		kind assistantartifact.Kind,
		title string,
		payload map[string]any,
		age int64,
	) *assistantartifact.Artifact {
		return &assistantartifact.Artifact{
			OrganizationID:   refs.org.ID,
			BusinessUnitID:   refs.org.BusinessUnitID,
			ThreadID:         thread,
			Kind:             kind,
			Status:           assistantartifact.StatusReady,
			Title:            title,
			Payload:          payload,
			SourceToolCallID: "seed_" + string(kind),
			CreatedAt:        refs.now - age,
			UpdatedAt:        refs.now - age,
		}
	}

	steps := make([]map[string]any, 0, len(refs.steps))
	for _, step := range refs.steps {
		steps = append(steps, map[string]any{
			"step":       step.PlanStep,
			"proposalId": step.ID.String(),
			"toolName":   step.ToolName,
			"rationale":  step.Rationale,
		})
	}

	preview := build(refs.thread(0), assistantartifact.KindReportPreview,
		"Revenue by customer, last 30 days",
		map[string]any{
			"name":    "Revenue by customer",
			"dataset": "shipments",
			"columns": []map[string]any{
				{"id": "customer", "label": "Customer", "type": "string"},
				{"id": "loads", "label": "Loads", "type": "int"},
				{"id": "revenue", "label": "Revenue", "type": "decimal", "format": "currency"},
			},
			"rows": []map[string]any{
				{"Customer": "Northwind Foods", "Loads": 48, "Revenue": "142850.00"},
				{"Customer": "Harborline Grocers", "Loads": 31, "Revenue": "98420.50"},
				{"Customer": "Cascade Building Supply", "Loads": 19, "Revenue": "61230.00"},
				{"Customer": "Pinnacle Paper", "Loads": 12, "Revenue": "38915.75"},
			},
			"totals":   map[string]any{"Customer": "Total", "Loads": 110, "Revenue": "341416.25"},
			"rowCount": 4,
		}, 2*activitySeedHour)
	preview.Pinned = true

	// Both of these are views over decision state rather than copies of it,
	// which is why the domain refuses them without the record they stand for.
	draft := build(refs.thread(0), assistantartifact.KindEmailDraft,
		"Detention notice to Northwind Foods",
		map[string]any{
			"tool":    "send_detention_notice",
			"to":      []string{"dispatch@northwindfoods.example"},
			"subject": "Detention accruing on " + refs.shipments[0].ProNumber,
			"body": "Our driver arrived at 09:12 and free time expired at 13:12. " +
				"Detention is accruing at the hourly rate in the agreement on file. " +
				"Please let us know when the trailer can be released.",
			"rationale": "The agreement bills detention hourly from the end of free time, " +
				"and the notice is what starts the clock they agreed to.",
		}, 2*activitySeedHour)
	draft.ProposalID = refs.draft.ID
	draft.RunID = refs.draft.RunID

	planned := build(refs.thread(0), assistantartifact.KindPlan,
		refs.plan.Title,
		map[string]any{
			"title":     refs.plan.Title,
			"summary":   refs.plan.Summary,
			"stepCount": refs.plan.StepCount,
			"steps":     steps,
		}, 5*activitySeedHour)
	planned.PlanID = refs.plan.ID
	planned.RunID = refs.plan.RunID

	artifacts := []*assistantartifact.Artifact{
		preview,

		build(refs.thread(0), assistantartifact.KindTableView,
			"Shipments delivering today",
			map[string]any{
				"tool":        "list_shipments",
				"entity":      "shipment",
				"searchedFor": []string{"delivering today", "not yet delivered"},
				"columns":     []string{"proNumber", "customer", "status", "destination"},
				"rows": []map[string]any{
					{
						"proNumber":   refs.shipments[0].ProNumber,
						"customer":    "Northwind Foods",
						"status":      "InTransit",
						"destination": "Columbus, OH",
					},
					{
						"proNumber":   refs.shipments[1].ProNumber,
						"customer":    "Harborline Grocers",
						"status":      "Delivered",
						"destination": "Toledo, OH",
					},
					{
						"proNumber":   refs.shipments[2].ProNumber,
						"customer":    "Northwind Foods",
						"status":      "InTransit",
						"destination": "Fort Wayne, IN",
					},
				},
				"rowCount": 3,
			}, 3*activitySeedHour),

		draft,
		planned,

		build(refs.thread(0), assistantartifact.KindEntityCard,
			refs.shipments[0].ProNumber,
			map[string]any{
				"entity": "shipment",
				"record": map[string]any{
					"id":          refs.shipments[0].ID.String(),
					"proNumber":   refs.shipments[0].ProNumber,
					"status":      string(refs.shipments[0].Status),
					"bol":         refs.shipments[0].BOL,
					"customer":    "Northwind Foods",
					"origin":      "Chicago, IL",
					"destination": "Columbus, OH",
					"weight":      "42,000 lb",
				},
			}, 4*activitySeedHour),

		build(refs.thread(1), assistantartifact.KindRateExplanation,
			"Why "+refs.shipments[1].ProNumber+" priced at $2,840",
			map[string]any{
				"shipmentId": refs.shipments[1].ID.String(),
				"side":       "Customer",
				"currency":   "USD",
				"winner": map[string]any{
					"agreementCode": "NWF-2026",
					"agreementName": "Northwind Foods master",
					"ruleLabel":     "Chicago → Columbus, dry van",
				},
				"tieBreak": "Two rules matched the lane; the more specific one won.",
				"rejected": []map[string]any{{
					"agreementCode": "SPOT-DEFAULT",
					"ruleLabel":     "Any lane, mileage",
					"reason":        "Less specific",
					"detail":        "A lane rule beats a mileage fallback.",
				}},
				"components": []map[string]any{
					{
						"label":        "Linehaul",
						"basis":        "355 mi at $6.80/mi",
						"amount":       "2414.00",
						"runningTotal": "2414.00",
					},
					{
						"label":        "Fuel surcharge",
						"basis":        "DOE midwest, $0.48/mi",
						"amount":       "170.40",
						"runningTotal": "2584.40",
					},
					{
						"label":        "Detention",
						"basis":        "2 h at $65.00/h",
						"amount":       "130.00",
						"runningTotal": "2714.40",
					},
					{
						"label":        "Stop-off",
						"basis":        "1 extra stop",
						"amount":       "125.60",
						"runningTotal": "2840.00",
					},
				},
				"guardrails": []map[string]any{{
					"kind":   "Minimum charge",
					"bound":  "1800.00",
					"raw":    "2840.00",
					"result": "2840.00",
				}},
				"totals": map[string]any{
					"linehaul":    "2414.00",
					"fuel":        "170.40",
					"accessorial": "255.60",
					"total":       "2840.00",
				},
				"warnings": []string{
					"The detention hours came from the dwell clock, not from a signed bill.",
				},
			}, 6*activitySeedHour),
	}

	if refs.includeRunDiff {
		artifacts = append(artifacts, build(refs.thread(0), assistantartifact.KindRunDiff,
			"Revenue by customer — what changed",
			runDiffPayload(refs), 2*activitySeedHour))
	}

	return artifacts
}

// runDiffPayload is two weekly runs of the revenue report compared: one
// customer grew, one shrank, one is new and one dropped out, so every kind of
// movement the renderer draws has a row. Decimals are exact strings, as the
// diff writes them.
func runDiffPayload(refs *artifactSeedRefs) map[string]any {
	week := int64(7 * 24 * 3600)
	side := func(runID string, generatedAt int64, rows int) map[string]any {
		return map[string]any{
			"runId":       runID,
			"reportName":  "Revenue by customer",
			"generatedAt": generatedAt,
			"rowCount":    rows,
			"truncated":   false,
		}
	}
	measure := func(before, after, delta string) []map[string]any {
		return []map[string]any{{
			"column": "revenue", "label": "Revenue", "before": before, "after": after, "delta": delta,
		}}
	}

	return map[string]any{
		"before":   side("rrun_seed_last_week", refs.now-week, 4),
		"after":    side("rrun_seed_this_week", refs.now-3600, 4),
		"keys":     []string{"customer"},
		"measures": []string{"revenue"},
		"summary": map[string]any{
			"added": 1, "removed": 1, "changed": 2, "unchanged": 1, "duplicate": 0,
		},
		"changes": []map[string]any{
			{
				"kind": "Changed", "key": "Acme Manufacturing",
				"keyValues": []string{"Acme Manufacturing"},
				"measures":  measure("18240.00", "22915.50", "4675.50"),
			},
			{
				"kind": "Changed", "key": "Harborline Grocers",
				"keyValues": []string{"Harborline Grocers"},
				"measures":  measure("9120.00", "6480.00", "-2640.00"),
			},
			{
				"kind": "Added", "key": "Northwind Foods",
				"keyValues": []string{"Northwind Foods"},
				"measures":  measure("", "3150.00", "3150.00"),
			},
			{
				"kind": "Removed", "key": "Blue Ridge Paper",
				"keyValues": []string{"Blue Ridge Paper"},
				"measures":  measure("2210.00", "", "-2210.00"),
			},
		},
		"totals": []map[string]any{{
			"column": "revenue", "label": "Revenue",
			"before": "41870.00", "after": "44845.50", "delta": "2975.50",
		}},
		"truncated": false,
		"note":      "",
	}
}
