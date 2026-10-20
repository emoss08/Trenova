package receivabletools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
)

type FlagManualReviewTool struct {
	exceptions agentports.AgentExceptionService
}

func NewFlagManualReviewTool(exceptions agentports.AgentExceptionService) agentports.AgentTool {
	return &FlagManualReviewTool{exceptions: exceptions}
}

func (t *FlagManualReviewTool) Name() string { return "agent_request_review" }

func (t *FlagManualReviewTool) Description() string {
	return "Flag a billing queue item for manual review by raising an agent exception " +
		"with a category and evidence. Use it on a billing run when the item cannot be " +
		"cleared without a person: missing paperwork, a rate that does not match, a " +
		"charge in dispute. Not for any other kind of record (use agent_raise_exception)."
}

// evidenceItemsProperty is a list of the records that show a problem, in
// the shape an exception keeps them.
func evidenceItemsProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				common.FieldType: common.StringProperty(
					"What kind of record it is: document, charge, rate, shipment.", 0,
				),
				evidenceID:       common.StringProperty("The record's id.", 0),
				common.FieldNote: common.StringProperty("What the record shows, in a sentence.", 0),
			},
			toolschema.KeyRequired:             []string{common.FieldType, evidenceID},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

const evidenceID = "id"

const schemaType = "type"

func (t *FlagManualReviewTool) ParamSchema() map[string]any {
	return map[string]any{
		schemaType: "object",
		"properties": map[string]any{
			"runId": agenttoolschema.RecordIDText(permission.ResourceAgentRun,
				"The id of the run you are in, the one working this run's subject; never "+
					"invent one."),
			common.FieldSubjectID: agenttoolschema.KindID(
				"The billing queue item being flagged: this run's subject.",
				permission.KindBillingQueueItem,
			),
			common.FieldCategory: agenttoolschema.Enum(
				"What kind of problem it is, such as MissingDocumentation, IncorrectRates, "+
					"WeightDiscrepancy, AccessorialDispute, DuplicateCharge or RateNotOnFile.",
				agenttoolschema.ExceptionCategories,
			),
			common.FieldSeverity: agenttoolschema.Enum(
				"How much it holds up billing.",
				agenttoolschema.Severities,
			),
			"attemptSummary": map[string]any{
				schemaType: "string",
				"description": "What you checked and why it was not enough, for the biller who " +
					"takes over.",
			},
			"blastRadius": map[string]any{
				schemaType:    "integer",
				"description": "How many records the problem affects, when more than this one.",
			},
			common.FieldEvidence: evidenceItemsProperty(
				"At least one record that shows the problem, each as {type, id, " +
					"note}: a document, a charge, a rate, the shipment.",
			),
		},
		"required": []string{
			"runId",
			common.FieldSubjectID,
			"category",
			"severity",
			"attemptSummary",
			"evidence",
		},
		"additionalProperties": false,
	}
}

func (t *FlagManualReviewTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAgentException,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeRun,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Escalates:     true,
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Records an exception on the run for a person inside the organization to " +
			"work.",
	}
}

func (t *FlagManualReviewTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.exceptions.Flag(ctx, request, params.Actor)

	return err
}

// Validate builds the exception the preview builds, so a call that names
// no run, a category the domain does not have or evidence in the wrong
// shape is refused to the model before it is proposed.
func (t *FlagManualReviewTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *FlagManualReviewTool) request(
	params *agentports.ToolExecuteParams,
) (*agentports.FlagAgentExceptionRequest, error) {
	runID, err := common.RequirePulid(params.Params, "runId")
	if err != nil {
		return nil, err
	}

	subjectID, err := common.RequirePulid(params.Params, common.FieldSubjectID)
	if err != nil {
		return nil, err
	}

	category, err := common.RequireEnum(
		params.Params,
		common.FieldCategory,
		agenttoolschema.ExceptionCategories.Values,
	)
	if err != nil {
		return nil, err
	}

	severity, err := common.RequireEnum(
		params.Params,
		common.FieldSeverity,
		agenttoolschema.Severities.Values,
	)
	if err != nil {
		return nil, err
	}

	attemptSummary, err := common.RequireString(params.Params, "attemptSummary")
	if err != nil {
		return nil, err
	}

	var evidence []agent.EvidenceRef
	if err = common.DecodeParam(params.Params, "evidence", &evidence); err != nil {
		return nil, err
	}

	return &agentports.FlagAgentExceptionRequest{
		RunID:          runID,
		Category:       category,
		Severity:       severity,
		SubjectType:    agent.SubjectBillingQueueItem,
		SubjectID:      subjectID,
		AttemptSummary: attemptSummary,
		Evidence:       evidence,
		BlastRadius:    int(common.OptionalInt64(params.Params, "blastRadius")),
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
	}, nil
}
