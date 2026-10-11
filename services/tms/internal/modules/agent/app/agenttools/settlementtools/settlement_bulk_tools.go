package settlementtools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
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
	ledger common.SettlementLedger[E]
	text   *lifecycleText
	book   SettlementBook[E]
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
	verb:     common.VerbApprove,
	past:     common.PastApproved,
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
	past:     common.PastPosted,
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

type SettlementBulkTool[E any] struct {
	name        string
	description string
	rationale   string
	single      *SettlementDecisionTool[E]
	batch       *common.RecordBatch
}

func bulkName(single string) string {
	return single + "s"
}

func newSettlementBulkTool[E any](
	spec settlementBulkSpec[E],
	bulk *settlementBulk,
) agentports.AgentTool {
	single := &SettlementDecisionTool[E]{
		decision: *bulk.decision(spec.text),
		ledger:   spec.ledger,
		book:     spec.book,
	}
	name := bulkName(single.Name())

	return &SettlementBulkTool[E]{
		name:        name,
		description: bulk.description(spec.text, single.Name()),
		rationale:   bulk.rationale(spec.text, single.Name()),
		single:      single,
		batch: &common.RecordBatch{
			Param:       paramSettlementIDs,
			SingleParam: common.ParamSettlementID,
			Resource:    spec.ledger.Resource,
			Noun:        spec.ledger.Noun,
			Nouns:       spec.ledger.Noun + "s",
			Verb:        bulk.verb,
			Past:        bulk.past,
			Unchanged:   "is already " + bulk.past,
			Single:      single,
			NeedsPerson: common.ErrSettlementNeedsAPerson,
			Labels:      settlementNumbers(spec.get, spec.ledger.Facts),
		},
	}
}

func settlementNumbers[E any](
	get settlementGetter[E],
	facts func(*E) *common.SettlementFacts,
) common.RecordLabels {
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
				labels[id] = facts(entity).Number
			}
		}

		return labels
	}
}

func (t *SettlementBulkTool[E]) Name() string { return t.name }

func (t *SettlementBulkTool[E]) Recipe() []string { return settlementBulkRecipes[t.name] }

func (t *SettlementBulkTool[E]) BatchOf() string { return t.single.Name() }

func (t *SettlementBulkTool[E]) Description() string { return t.description }

func (t *SettlementBulkTool[E]) SearchTerms() []string {
	return []string{"every", "several", "many", "together", common.SearchTermBulk, "once"}
}

func (t *SettlementBulkTool[E]) ParamSchema() map[string]any {
	return common.BulkIDsSchema(t.batch,
		"The "+t.batch.Nouns+", by id from "+t.single.ledger.Sources+" or the proposals "+
			"that made them. Never guess one.",
		nil)
}

func (t *SettlementBulkTool[E]) Policy() agentports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = t.single.ledger.Artifact
	policy.Rationale = t.rationale

	return policy
}

func (t *SettlementBulkTool[E]) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *SettlementBulkTool[E]) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *SettlementBulkTool[E]) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *SettlementBulkTool[E]) ExecuteWithResult(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}

func driverBulkSpec(
	s *driversettlementservice.Service,
) settlementBulkSpec[driversettlement.Settlement] {
	spec := settlementBulkSpec[driversettlement.Settlement]{
		ledger: common.DriverSettlementLedger(),
		text:   driverLifecycle(),
		book:   s,
	}
	if s != nil {
		spec.get = func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*driversettlement.Settlement, error) {
			return s.Get(ctx, driversettlementports.GetDriverSettlementByIDRequest{
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
			return s.Get(ctx, carriersettlementports.GetCarrierSettlementByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		}
	}

	return spec
}

func provideApproveDriverSettlementsTool(
	s *driversettlementservice.Service,
) agentports.AgentTool {
	return newSettlementBulkTool(driverBulkSpec(s), &approveSettlements)
}

func providePostDriverSettlementsTool(s *driversettlementservice.Service) agentports.AgentTool {
	return newSettlementBulkTool(driverBulkSpec(s), &postSettlements)
}

func provideApproveCarrierSettlementsTool(
	s *carriersettlementservice.Service,
) agentports.AgentTool {
	return newSettlementBulkTool(carrierBulkSpec(s), &approveSettlements)
}

func providePostCarrierSettlementsTool(
	s *carriersettlementservice.Service,
) agentports.AgentTool {
	return newSettlementBulkTool(carrierBulkSpec(s), &postSettlements)
}

func SettlementBulkProviders() []any {
	return []any{
		provideApproveDriverSettlementsTool,
		providePostDriverSettlementsTool,
		provideApproveCarrierSettlementsTool,
		providePostCarrierSettlementsTool,
	}
}
