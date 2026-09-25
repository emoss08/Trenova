package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrMissingRun = errors.New("an exception can only be raised from within an agent run")

var ErrSubjectUncheckable = errors.New(
	"the record an exception is about cannot be checked right now",
)

type raiseExceptionTool struct {
	exceptions serviceports.AgentExceptionService
	subjects   repositories.AgentSubjectRepository
}

func newRaiseExceptionTool(
	exceptions serviceports.AgentExceptionService,
	subjects repositories.AgentSubjectRepository,
) serviceports.AgentTool {
	return &raiseExceptionTool{exceptions: exceptions, subjects: subjects}
}

func (t *raiseExceptionTool) Name() string { return "raise_exception" }

func (t *raiseExceptionTool) Description() string {
	return "Hand a case to a person when you cannot resolve it: say what you tried, how " +
		"severe it is and which record it concerns. Use this instead of guessing. It is for " +
		"a problem with a record or a run that a person must resolve, not for requesting a " +
		"product feature or reporting that a tool cannot do something; for that, tell the " +
		"person plainly what cannot be done. It only works inside an agent run."
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
				"type": "string",
				"description": "The id of that record: usually this run's subject, or an id a " +
					"tool such as get_shipment or list_shipments returned. Its prefix must " +
					"match subjectType: a report's id starts rd_, a dashboard's rdb_, an " +
					"insight's inst_.",
			},
			"category": map[string]any{
				"type":        "string",
				"enum":        exceptionCategoryNames(),
				"description": "What kind of problem it is.",
			},
			"severity": map[string]any{
				"type": "string",
				"enum": []string{"Low", "Medium", "High", "Critical"},
				"description": "How urgent it is for the person who takes over. Defaults to " +
					"Medium.",
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
		"required": []string{
			"subjectType",
			"subjectId",
			"category",
			"severity",
			"attemptSummary",
		},
		"additionalProperties": false,
	}
}

func (t *raiseExceptionTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAgentException,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeRun,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Records an exception against the run itself for a person inside the " +
			"organization.",
	}
}

func (t *raiseExceptionTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(ctx, params)
	if err != nil {
		return err
	}

	_, err = t.exceptions.Flag(ctx, request, params.Actor)

	return err
}

func (t *raiseExceptionTool) request(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*serviceports.FlagAgentExceptionRequest, error) {
	if params.RunID.IsNil() {
		return nil, ErrMissingRun
	}

	subjectType, subjectID, err := t.subject(ctx, params)
	if err != nil {
		return nil, err
	}
	summary, err := requireString(params.Params, "attemptSummary")
	if err != nil {
		return nil, err
	}

	category := agent.ExceptionCategory(optionalString(params.Params, "category"))
	if !category.IsValid() {
		category = agent.CategoryOther
	}
	severity := agent.Severity(optionalString(params.Params, "severity"))
	if !severity.IsValid() {
		severity = agent.SeverityMedium
	}

	return &serviceports.FlagAgentExceptionRequest{
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
	}, nil
}

func (t *raiseExceptionTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	_, _, err := t.subject(ctx, params)

	return err
}

// subject reads the record a case is about and checks that it is one: the
// id's prefix must match the kind named, and the record must be the tenant's.
// A report's id filed as an insight used to be recorded as given, approved,
// and left an exception pointing at nothing. Execute checks again, because an
// approval runs it long after the proposal was validated.
func (t *raiseExceptionTool) subject(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (agent.SubjectType, pulid.ID, error) {
	subjectTypeRaw, err := requireString(params.Params, "subjectType")
	if err != nil {
		return "", pulid.Nil, err
	}
	subjectType := agent.SubjectType(strings.TrimSpace(subjectTypeRaw))
	if !subjectType.IsValid() {
		return "", pulid.Nil, fmt.Errorf(
			"parameter \"subjectType\" is not a known record kind; use one of %s",
			strings.Join(subjectTypeNames(), ", "),
		)
	}
	subjectID, err := requirePulid(params.Params, "subjectId")
	if err != nil {
		return "", pulid.Nil, err
	}
	if err = subjectType.CheckID(subjectID); err != nil {
		return "", pulid.Nil, err
	}

	if t.subjects == nil {
		return "", pulid.Nil, ErrSubjectUncheckable
	}
	exists, err := t.subjects.Exists(ctx, repositories.AgentSubjectExistsRequest{
		TenantInfo:  tenantFrom(params),
		SubjectType: subjectType,
		SubjectID:   subjectID,
	})
	if err != nil {
		return "", pulid.Nil, fmt.Errorf("check the %s: %w", subjectType.Noun(), err)
	}
	if !exists {
		return "", pulid.Nil, fmt.Errorf(
			"there is no %s %s in this organization; pass the id of a record a tool returned",
			subjectType.Noun(), subjectID,
		)
	}

	return subjectType, subjectID, nil
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
