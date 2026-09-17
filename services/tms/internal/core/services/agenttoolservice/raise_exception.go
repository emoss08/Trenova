package agenttoolservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

var ErrMissingRun = errors.New("an exception can only be raised from within an agent run")

type raiseExceptionTool struct {
	exceptions serviceports.AgentExceptionService
}

func newRaiseExceptionTool(exceptions serviceports.AgentExceptionService) serviceports.AgentTool {
	return &raiseExceptionTool{exceptions: exceptions}
}

func (t *raiseExceptionTool) Name() string { return "raise_exception" }

func (t *raiseExceptionTool) Description() string {
	return "Hand a case to a person when you cannot resolve it: say what you tried, how " +
		"severe it is and which record it concerns. Use this instead of guessing."
}

func (t *raiseExceptionTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subjectType": map[string]any{
				"type":        "string",
				"enum":        subjectTypeNames(),
				"description": "The kind of record the case is about.",
			},
			"subjectId": map[string]any{
				"type":        "string",
				"description": "The id of that record.",
			},
			"category": map[string]any{
				"type":        "string",
				"enum":        exceptionCategoryNames(),
				"description": "What kind of problem it is.",
			},
			"severity": map[string]any{
				"type": "string",
				"enum": []string{"Low", "Medium", "High", "Critical"},
			},
			"attemptSummary": map[string]any{
				"type":        "string",
				"description": "What you tried and why it was not enough, for the person who takes over.",
			},
			"blastRadius": map[string]any{
				"type":        "integer",
				"description": "How many records are affected.",
			},
		},
		"required":             []string{"subjectType", "subjectId", "category", "severity", "attemptSummary"},
		"additionalProperties": false,
	}
}

func (t *raiseExceptionTool) Reversible() bool { return false }

func (t *raiseExceptionTool) PermissionResource() permission.Resource {
	return permission.ResourceAgentException
}

func (t *raiseExceptionTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

func (t *raiseExceptionTool) RequiresIdempotencyKey() bool { return false }

func (t *raiseExceptionTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierAutoExecute
}

func (t *raiseExceptionTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if params.RunID.IsNil() {
		return ErrMissingRun
	}

	subjectTypeRaw, err := requireString(params.Params, "subjectType")
	if err != nil {
		return err
	}
	subjectType := agent.SubjectType(subjectTypeRaw)
	if !subjectType.IsValid() {
		return errors.New("parameter \"subjectType\" is not a known record kind")
	}
	subjectID, err := requirePulid(params.Params, "subjectId")
	if err != nil {
		return err
	}
	summary, err := requireString(params.Params, "attemptSummary")
	if err != nil {
		return err
	}

	category := agent.ExceptionCategory(optionalString(params.Params, "category"))
	if !category.IsValid() {
		category = agent.CategoryOther
	}
	severity := agent.Severity(optionalString(params.Params, "severity"))
	if !severity.IsValid() {
		severity = agent.SeverityMedium
	}

	_, err = t.exceptions.Flag(ctx, &serviceports.FlagAgentExceptionRequest{
		RunID:          params.RunID,
		Category:       category,
		Severity:       severity,
		SubjectType:    subjectType,
		SubjectID:      subjectID,
		AttemptSummary: summary,
		Evidence: []agent.EvidenceRef{{
			Type: "agent_run",
			ID:   params.RunID.String(),
			Note: "raised by the agent",
		}},
		BlastRadius: int(optionalInt64(params.Params, "blastRadius")),
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
	}, params.Actor)

	return err
}

func subjectTypeNames() []string {
	kinds := agent.AllSubjectTypes()
	names := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		names = append(names, string(kind))
	}

	return names
}

func exceptionCategoryNames() []string {
	categories := agent.AllExceptionCategories()
	names := make([]string, 0, len(categories))
	for _, category := range categories {
		names = append(names, string(category))
	}

	return names
}
