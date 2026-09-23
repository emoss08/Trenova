package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
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
			"category": map[string]any{
				"type": "string",
				"description": "What kind of problem it is, such as MissingDocumentation, " +
					"IncorrectRates, WeightDiscrepancy, AccessorialDispute, DuplicateCharge or " +
					"RateNotOnFile.",
			},
			"severity": map[string]any{
				"type":        "string",
				"description": "Low, Medium, High or Critical: how much it holds up billing.",
			},
			"attemptSummary": map[string]any{
				"type": "string",
				"description": "What you checked and why it was not enough, for the biller who " +
					"takes over.",
			},
			"blastRadius": map[string]any{
				"type":        "integer",
				"description": "How many records the problem affects, when more than this one.",
			},
			"evidence": map[string]any{
				"type": "array",
				"description": "At least one record that shows the problem, each as {type, id, " +
					"note}: a document, a charge, a rate, the shipment.",
				"items": map[string]any{"type": "object"},
			},
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

	runID, err := requirePulid(params.Params, "runId")
	if err != nil {
		return err
	}

	subjectID, err := requirePulid(params.Params, "subjectId")
	if err != nil {
		return err
	}

	category, err := requireString(params.Params, "category")
	if err != nil {
		return err
	}

	severity, err := requireString(params.Params, "severity")
	if err != nil {
		return err
	}

	attemptSummary, err := requireString(params.Params, "attemptSummary")
	if err != nil {
		return err
	}

	var evidence []agent.EvidenceRef
	if err = decodeParam(params.Params, "evidence", &evidence); err != nil {
		return err
	}

	_, err = t.exceptions.Flag(ctx, &serviceports.FlagAgentExceptionRequest{
		RunID:          runID,
		Category:       agent.ExceptionCategory(category),
		Severity:       agent.Severity(severity),
		SubjectType:    agent.SubjectBillingQueueItem,
		SubjectID:      subjectID,
		AttemptSummary: attemptSummary,
		Evidence:       evidence,
		BlastRadius:    int(optionalInt64(params.Params, "blastRadius")),
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
	}, params.Actor)

	return err
}
