package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carriersettlementservice"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramSettlementIDs = "settlementIds"

type settlementGetter[E any] func(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*E, error)

type settlementBulkSpec[E any] struct {
	ledger settlementLedger[E]
	text   *lifecycleText
	book   settlementBook[E]
	get    settlementGetter[E]
}

type settlementBulk struct {
	decision    func(*lifecycleText) *settlementDecision
	verb        string
	past        string
	description func(text *lifecycleText, single string) string
	rationale   func(text *lifecycleText, single string) string
}

var approveSettlements = settlementBulk{
	decision: approveDecision,
	verb:     verbApprove,
	past:     pastApproved,
	description: func(text *lifecycleText, single string) string {
		return "Propose approving several pending " + text.noun + "s in one proposal the " +
			"person approves together. Use it instead of " + single + " whenever more " +
			"than one is ready. Each is approved exactly as " + single +
			" would, committing the organization to paying the " + text.payee + " what it " +
			"comes to; a person always decides and may untick settlements. Propose only " +
			"those " + text.getTool + " shows with no open exception or dispute you cannot " +
			"explain. A settlement approving would refuse is reported, not approved, and the " +
			"rest still go. Takes 1 to 50 settlements."
	},
	rationale: func(text *lifecycleText, single string) string {
		return "Commits the organization to what several " + text.payee + "s are paid at " +
			"once; only a person approves, settlement by settlement exactly as " + single +
			", and may untick any of them."
	},
}

var postSettlements = settlementBulk{
	decision: postDecision,
	verb:     "post",
	past:     pastPosted,
	description: func(text *lifecycleText, single string) string {
		return "Propose posting several approved " + text.noun + "s to the general ledger " +
			"in one proposal the person approves together. Use it instead of " + single +
			" whenever more than one is ready. Each is posted exactly as " + single +
			" would: " + text.postEach + ". It cannot be undone except by voiding, so a " +
			"person always decides and may untick settlements. A settlement posting would " +
			"refuse is reported, not posted, and the rest still go. Takes 1 to 50 settlements."
	},
	rationale: func(text *lifecycleText, single string) string {
		return "Books several settlements' payables to the ledger at once and queues each " +
			"for the accounting system; only a person posts, settlement by settlement " +
			"exactly as " + single + ", and may untick any of them."
	},
}

// settlementBulkRecipes are the reads that hand each bulk tool its
// settlements, ending at the tool. They once ran on through the period's other
// writes, which told a model asked only to approve to submit, post and record
// payments as well. The bulk tools are named from their single twins, so the
// recipes are kept by name here rather than on each spec.
var settlementBulkRecipes = map[string][]string{
	"approve_driver_settlements": {
		"list_driver_settlements",
		"get_driver_settlement",
		"approve_driver_settlements",
	},
	"post_driver_settlements": {
		"list_driver_settlements",
		"post_driver_settlements",
	},
	"approve_carrier_settlements": {
		"list_carrier_settlements",
		"get_carrier_settlement",
		"approve_carrier_settlements",
	},
}

type settlementBulkTool[E any] struct {
	name        string
	description string
	rationale   string
	single      *settlementDecisionTool[E]
	batch       *recordBatch
}

var (
	_ serviceports.ToolPreviewer      = (*settlementBulkTool[driversettlement.Settlement])(nil)
	_ serviceports.ToolValidator      = (*settlementBulkTool[driversettlement.Settlement])(nil)
	_ serviceports.ToolResultReporter = (*settlementBulkTool[driversettlement.Settlement])(nil)
	_ serviceports.SearchableTool     = (*settlementBulkTool[driversettlement.Settlement])(nil)
)

func bulkName(single string) string {
	return single + "s"
}

func newSettlementBulkTool[E any](
	spec settlementBulkSpec[E],
	bulk *settlementBulk,
) serviceports.AgentTool {
	single := &settlementDecisionTool[E]{
		decision: *bulk.decision(spec.text),
		ledger:   spec.ledger,
		book:     spec.book,
	}
	name := bulkName(single.Name())

	return &settlementBulkTool[E]{
		name:        name,
		description: bulk.description(spec.text, single.Name()),
		rationale:   bulk.rationale(spec.text, single.Name()),
		single:      single,
		batch: &recordBatch{
			param:       paramSettlementIDs,
			singleParam: paramSettlementID,
			resource:    spec.ledger.resource,
			noun:        spec.ledger.noun,
			nouns:       spec.ledger.noun + "s",
			verb:        bulk.verb,
			past:        bulk.past,
			unchanged:   "is already " + bulk.past,
			single:      single,
			needsPerson: ErrSettlementNeedsAPerson,
			labels:      settlementNumbers(spec.get, spec.ledger.facts),
		},
	}
}

func settlementNumbers[E any](
	get settlementGetter[E],
	facts func(*E) *settlementFacts,
) recordLabels {
	return func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
	) map[pulid.ID]string {
		if get == nil {
			return nil
		}
		labels := make(map[pulid.ID]string, len(ids))
		for _, id := range ids {
			entity, err := get(ctx, tenant, id)
			if err == nil && entity != nil {
				labels[id] = facts(entity).number
			}
		}

		return labels
	}
}

func (t *settlementBulkTool[E]) Name() string { return t.name }

func (t *settlementBulkTool[E]) Recipe() []string { return settlementBulkRecipes[t.name] }

func (t *settlementBulkTool[E]) BatchOf() string { return t.single.Name() }

func (t *settlementBulkTool[E]) Description() string { return t.description }

func (t *settlementBulkTool[E]) SearchTerms() []string {
	return []string{"every", "several", "many", "together", searchTermBulk, "once"}
}

func (t *settlementBulkTool[E]) ParamSchema() map[string]any {
	return bulkIDsSchema(t.batch,
		"The "+t.batch.nouns+", by id from "+t.single.ledger.sources+" or the proposals "+
			"that made them. Never guess one.",
		nil)
}

func (t *settlementBulkTool[E]) Policy() serviceports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = t.single.ledger.artifact
	policy.Rationale = t.rationale

	return policy
}

func (t *settlementBulkTool[E]) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *settlementBulkTool[E]) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *settlementBulkTool[E]) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *settlementBulkTool[E]) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}

func driverBulkSpec(
	s *driversettlementservice.Service,
) settlementBulkSpec[driversettlement.Settlement] {
	spec := settlementBulkSpec[driversettlement.Settlement]{
		ledger: driverSettlementLedger(),
		text:   driverLifecycle(),
		book:   s,
	}
	if s != nil {
		spec.get = func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*driversettlement.Settlement, error) {
			return s.Get(ctx, repositories.GetDriverSettlementByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		}
	}

	return spec
}

func carrierBulkSpec(
	s *carriersettlementservice.Service,
) settlementBulkSpec[carriersettlement.CarrierSettlement] {
	spec := settlementBulkSpec[carriersettlement.CarrierSettlement]{
		ledger: carrierSettlementLedger(),
		text:   carrierLifecycle(),
		book:   s,
	}
	if s != nil {
		spec.get = func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*carriersettlement.CarrierSettlement, error) {
			return s.Get(ctx, repositories.GetCarrierSettlementByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		}
	}

	return spec
}

func provideApproveDriverSettlementsTool(
	s *driversettlementservice.Service,
) serviceports.AgentTool {
	return newSettlementBulkTool(driverBulkSpec(s), &approveSettlements)
}

func providePostDriverSettlementsTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return newSettlementBulkTool(driverBulkSpec(s), &postSettlements)
}

func provideApproveCarrierSettlementsTool(
	s *carriersettlementservice.Service,
) serviceports.AgentTool {
	return newSettlementBulkTool(carrierBulkSpec(s), &approveSettlements)
}

func providePostCarrierSettlementsTool(
	s *carriersettlementservice.Service,
) serviceports.AgentTool {
	return newSettlementBulkTool(carrierBulkSpec(s), &postSettlements)
}

func settlementBulkProviders() []any {
	return []any{
		provideApproveDriverSettlementsTool,
		providePostDriverSettlementsTool,
		provideApproveCarrierSettlementsTool,
		providePostCarrierSettlementsTool,
	}
}
