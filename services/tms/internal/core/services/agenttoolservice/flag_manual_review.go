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
	return "Flag the item for manual review by raising an agent exception with a category and evidence."
}

func (t *flagManualReviewTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"runId":          map[string]any{"type": "string"},
			"subjectId":      map[string]any{"type": "string"},
			"category":       map[string]any{"type": "string"},
			"severity":       map[string]any{"type": "string"},
			"attemptSummary": map[string]any{"type": "string"},
			"blastRadius":    map[string]any{"type": "integer"},
			"evidence": map[string]any{
				"type":  "array",
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

func (t *flagManualReviewTool) Reversible() bool { return true }

func (t *flagManualReviewTool) PermissionResource() permission.Resource {
	return permission.ResourceAgentException
}

func (t *flagManualReviewTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

func (t *flagManualReviewTool) RequiresIdempotencyKey() bool { return false }

func (t *flagManualReviewTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
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
