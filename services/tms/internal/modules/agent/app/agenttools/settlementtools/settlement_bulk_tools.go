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
	description func(text *lifecycleText) string
	rationale   func(text *lifecycleText) string
}

var approveSettlements = settlementBulk{
	decision: approveDecision,
	verb:     common.VerbApprove,
	past:     common.PastApproved,
	description: func(text *lifecycleText) string {
		return "Propose approving pending " + text.noun + "s, one or up to 50 in one " +
			"proposal the person decides together. Approving commits the organization to " +
			"paying each " + text.payee + " what it comes to. Propose only those " +
			text.getTool + " shows with no open exception or dispute you cannot explain; " +
			"the person may untick settlements. Name every settlement that is ready in one " +
			"call; one approving would refuse is reported, not approved, and the rest still go."
	},
	rationale: func(text *lifecycleText) string {
		return "Commits the organization to what " + text.payee + "s are paid; only a " +
			"person approves, and may untick any of them, and hands-off approval is the " +
			"settlement control's rule."
	},
}

var postSettlements = settlementBulk{
	decision: postDecision,
	verb:     "post",
	past:     common.PastPosted,
	description: func(text *lifecycleText) string {
		return "Propose posting approved " + text.noun + "s to the general ledger, one or " +
			"up to 50 in one proposal the person decides together. Each is " + text.postEach +
			". It cannot be undone except by voiding; the person may untick settlements. " +
			"Name every approved settlement in one call; one posting would refuse is " +
			"reported, not posted, and the rest still go."
	},
	rationale: func(*lifecycleText) string {
		return "Books settlements' payables to the ledger and queues each for the accounting " +
			"system; only a person posts, and may untick any of them."
	},
}

// settlementBulkRecipes are the reads that hand each tool its settlements,
// ending at the tool. They once ran on through the period's other writes,
// which told a model asked only to approve to submit, post and record
// payments as well.
var settlementBulkRecipes = map[string][]string{
	"settlement_approve_driver": {
		toolListDriverSettlements,
		"settlement_get_driver",
		"settlement_approve_driver",
	},
	"settlement_post_driver": {
		toolListDriverSettlements,
		"settlement_post_driver",
	},
	"settlement_approve_carrier": {
		"settlement_list_carrier",
		"settlement_get_carrier",
		"settlement_approve_carrier",
	},
}

type SettlementBulkTool[E any] struct {
	name        string
	description string
	rationale   string
	single      *SettlementDecisionTool[E]
	batch       *common.RecordBatch
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
	return &SettlementBulkTool[E]{
		name:        single.Name(),
		description: bulk.description(spec.text),
		rationale:   bulk.rationale(spec.text),
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

func (t *SettlementBulkTool[E]) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return t.batch.Target(params)
}

func (t *SettlementBulkTool[E]) Description() string { return t.description }

func (t *SettlementBulkTool[E]) SearchTerms() []string { return t.single.SearchTerms() }

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

func (t *SettlementBulkTool[E]) SplitRecords(params map[string]any) ([]map[string]any, error) {
	return t.batch.Split(params)
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
	return t.batch.Execute(ctx, t, &params)
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
