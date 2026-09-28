package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
)

type flagManualReviewTool struct {
	exceptions serviceports.AgentExceptionService
}

func newFlagManualReviewTool(exceptions serviceports.AgentExceptionService) serviceports.AgentTool {
	return &flagManualReviewTool{exceptions: exceptions}
}

func (t *flagManualReviewTool) Name() string { return "flag_for_manual_review" }

func (t *flagManualReviewTool) Description() string {
	return "Flag a billing queue item for manual review by raising an agent exception " +
		"with a category and evidence. Use it on a billing run when the item cannot be " +
		"cleared without a person: missing paperwork, a rate that does not match, a " +
		"charge in dispute. Not for any other kind of record (use raise_exception)."
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
				fieldType: stringProperty(
					"What kind of record it is: document, charge, rate, shipment.", 0,
				),
				evidenceID: stringProperty("The record's id.", 0),
				fieldNote:  stringProperty("What the record shows, in a sentence.", 0),
			},
			toolschema.KeyRequired:             []string{fieldType, evidenceID},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

const evidenceID = "id"

func (t *flagManualReviewTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"runId": map[string]any{
				"type": "string",
				"description": "The id of the run you are in, the one working this run's " +
					"subject; never invent one.",
			},
			"subjectId": map[string]any{
				"type":        "string",
				"description": "The billing queue item being flagged: this run's subject.",
			},
			fieldCategory: agenttoolschema.Enum(
				"What kind of problem it is, such as MissingDocumentation, IncorrectRates, "+
					"WeightDiscrepancy, AccessorialDispute, DuplicateCharge or RateNotOnFile.",
				agenttoolschema.ExceptionCategories,
			),
			fieldSeverity: agenttoolschema.Enum(
				"How much it holds up billing.",
				agenttoolschema.Severities,
			),
			"attemptSummary": map[string]any{
				"type": "string",
				"description": "What you checked and why it was not enough, for the biller who " +
					"takes over.",
			},
			"blastRadius": map[string]any{
				"type":        "integer",
				"description": "How many records the problem affects, when more than this one.",
			},
			fieldEvidence: evidenceItemsProperty(
				"At least one record that shows the problem, each as {type, id, " +
					"note}: a document, a charge, a rate, the shipment.",
			),
		},
		"required": []string{
			"runId",
			"subjectId",
			"category",
			"severity",
			"attemptSummary",
			"evidence",
		},
		"additionalProperties": false,
	}
}

func (t *flagManualReviewTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAgentException,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeRun,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Records an exception on the run for a person inside the organization to " +
			"work.",
	}
}

func (t *flagManualReviewTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
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
func (t *flagManualReviewTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return previewValidates(ctx, t, &params)
}

func (t *flagManualReviewTool) request(
	params *serviceports.ToolExecuteParams,
) (*serviceports.FlagAgentExceptionRequest, error) {
	runID, err := requirePulid(params.Params, "runId")
	if err != nil {
		return nil, err
	}

	subjectID, err := requirePulid(params.Params, "subjectId")
	if err != nil {
		return nil, err
	}

	category, err := requireEnum(
		params.Params,
		fieldCategory,
		agenttoolschema.ExceptionCategories.Values,
	)
	if err != nil {
		return nil, err
	}

	severity, err := requireEnum(params.Params, fieldSeverity, agenttoolschema.Severities.Values)
	if err != nil {
		return nil, err
	}

	attemptSummary, err := requireString(params.Params, "attemptSummary")
	if err != nil {
		return nil, err
	}

	var evidence []agent.EvidenceRef
	if err = decodeParam(params.Params, "evidence", &evidence); err != nil {
		return nil, err
	}

	return &serviceports.FlagAgentExceptionRequest{
		RunID:          runID,
		Category:       category,
		Severity:       severity,
		SubjectType:    agent.SubjectBillingQueueItem,
		SubjectID:      subjectID,
		AttemptSummary: attemptSummary,
		Evidence:       evidence,
		BlastRadius:    int(optionalInt64(params.Params, "blastRadius")),
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
	}, nil
}
