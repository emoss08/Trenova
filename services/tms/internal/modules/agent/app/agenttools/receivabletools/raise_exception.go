package receivabletools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

// subjectKinds are the kinds of record a case can be about, read from the
// prefix each agent.SubjectType declares, so the id beside a subjectType is
// held to exactly those kinds and the list cannot drift from the enum. When a
// subject's prefix has no kind in permission's table the list is empty and
// the parameter is left unmarked, for the catalog contract test to name,
// rather than refusing every real id of that subject.
var subjectKinds = func() []permission.RecordKind {
	subjects := agent.AllSubjectTypes()
	kinds := make([]permission.RecordKind, 0, len(subjects))
	for _, subject := range subjects {
		kind, ok := permission.RecordKindOfIDPrefix(subject.IDPrefix())
		if !ok {
			return nil
		}
		kinds = append(kinds, kind)
	}

	return kinds
}()

func subjectIDProperty(description string) map[string]any {
	if len(subjectKinds) == 0 {
		return agenttoolschema.IDText(description)
	}

	return agenttoolschema.KindID(description, subjectKinds...)
}

var ErrMissingRun = errors.New("an exception can only be raised from within an agent run")

var ErrSubjectUncheckable = errors.New(
	"the record an exception is about cannot be checked right now",
)

type RaiseExceptionTool struct {
	exceptions agentports.AgentExceptionService
	subjects   agentports.AgentSubjectRepository
}

func NewRaiseExceptionTool(
	exceptions agentports.AgentExceptionService,
	subjects agentports.AgentSubjectRepository,
) agenttoolports.AgentTool {
	return &RaiseExceptionTool{exceptions: exceptions, subjects: subjects}
}

func (t *RaiseExceptionTool) Name() string { return "agent_raise_exception" }

func (t *RaiseExceptionTool) Description() string {
	return "Flag a case for a person to review when you cannot resolve it: say what you tried, how " +
		"severe it is and which record it concerns. Name the records that show the " +
		"problem as evidence. Use this instead of guessing. It is for " +
		"a problem with a record or a run that a person must resolve, not for requesting a " +
		"product feature or reporting that a tool cannot do something; for that, tell the " +
		"person plainly what cannot be done. It only works inside an agent run."
}

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

func (t *RaiseExceptionTool) SearchTerms() []string {
	return []string{"flag for review", "manual review", "escalate", "hand off", "human"}
}

func (t *RaiseExceptionTool) ParamSchema() map[string]any {
	return map[string]any{
		schemaType: "object",
		"properties": map[string]any{
			common.FieldSubjectType: agenttoolschema.Enum(
				"The kind of record the case is about; BillingQueueItem on a billing run.",
				agenttoolschema.SubjectTypes,
			),
			common.FieldSubjectID: subjectIDProperty(
				"The id of that record: usually this run's subject, or an id a " +
					"tool such as shipment_get or shipment_list returned. Its prefix must " +
					"match subjectType: a report's id starts rd_, a dashboard's rdb_, an " +
					"insight's inst_.",
			),
			common.FieldCategory: agenttoolschema.Enum(
				"What kind of problem it is. On a billing queue item that is usually "+
					"MissingDocumentation, IncorrectRates, WeightDiscrepancy, "+
					"AccessorialDispute, DuplicateCharge or RateNotOnFile.",
				agenttoolschema.ExceptionCategories,
			),
			common.FieldSeverity: agenttoolschema.Enum(
				"How urgent it is for the person who takes over. Defaults to Medium.",
				agenttoolschema.Severities,
			),
			"attemptSummary": map[string]any{
				schemaType:    "string",
				"description": "What you tried and why it was not enough, for the person who takes over.",
			},
			"blastRadius": map[string]any{
				schemaType:    "integer",
				"description": "How many records are affected.",
			},
			common.FieldEvidence: evidenceItemsProperty(
				"Optional: the records that show the problem, each as {type, id, note}: a " +
					"document, a charge, a rate, the shipment. The run itself is always kept " +
					"as evidence.",
			),
		},
		"required": []string{
			"subjectType",
			common.FieldSubjectID,
			"category",
			"severity",
			"attemptSummary",
		},
		"additionalProperties": false,
	}
}

func (t *RaiseExceptionTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAgentException,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeRun,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Escalates:     true,
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Records an exception against the run itself for a person inside the " +
			"organization.",
	}
}

func (t *RaiseExceptionTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.exceptions.Flag(ctx, request, params.Actor)

	return err
}

func (t *RaiseExceptionTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentports.FlagAgentExceptionRequest, error) {
	if params.RunID.IsNil() {
		return nil, ErrMissingRun
	}

	subjectType, subjectID, err := t.subject(ctx, *params)
	if err != nil {
		return nil, err
	}
	summary, err := common.RequireString(params.Params, "attemptSummary")
	if err != nil {
		return nil, err
	}

	category, given, err := common.OptionalEnum(
		params.Params, "category", agenttoolschema.ExceptionCategories.Values,
	)
	if err != nil {
		return nil, err
	}
	if !given {
		category = agent.CategoryOther
	}
	severity, given, err := common.OptionalEnum(
		params.Params, "severity", agenttoolschema.Severities.Values,
	)
	if err != nil {
		return nil, err
	}
	if !given {
		severity = agent.SeverityMedium
	}
	named, err := givenEvidence(params.Params)
	if err != nil {
		return nil, err
	}
	evidence := make([]agent.EvidenceRef, 0, len(named)+1)
	evidence = append(evidence, agent.EvidenceRef{
		Type: "agent_run",
		ID:   params.RunID.String(),
		Note: "raised by the agent",
	})
	evidence = append(evidence, named...)

	return &agentports.FlagAgentExceptionRequest{
		RunID:          params.RunID,
		Category:       category,
		Severity:       severity,
		SubjectType:    subjectType,
		SubjectID:      subjectID,
		AttemptSummary: summary,
		Evidence:       evidence,
		BlastRadius:    int(common.OptionalInt64(params.Params, "blastRadius")),
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
	}, nil
}

func (t *RaiseExceptionTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	if _, err := givenEvidence(params.Params); err != nil {
		return err
	}
	_, _, err := t.subject(ctx, params)

	return err
}

func givenEvidence(params map[string]any) ([]agent.EvidenceRef, error) {
	if raw, ok := params[common.FieldEvidence]; !ok || raw == nil {
		return nil, nil
	}

	var evidence []agent.EvidenceRef
	if err := common.DecodeParam(params, common.FieldEvidence, &evidence); err != nil {
		return nil, err
	}
	for idx := range evidence {
		ref := &evidence[idx]
		ref.Type = strings.TrimSpace(ref.Type)
		ref.ID = strings.TrimSpace(ref.ID)
		ref.Note = strings.TrimSpace(ref.Note)
		if ref.Type == "" || ref.ID == "" {
			return nil, fmt.Errorf(
				"parameter %q item %d needs both a type and an id",
				common.FieldEvidence, idx,
			)
		}
	}

	return evidence, nil
}

// subject reads the record a case is about and checks that it is one: the
// id's prefix must match the kind named, and the record must be the tenant's.
// A report's id filed as an insight used to be recorded as given, approved,
// and left an exception pointing at nothing. Execute checks again, because an
// approval runs it long after the proposal was validated.
func (t *RaiseExceptionTool) subject(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) (agent.SubjectType, pulid.ID, error) {
	subjectTypeRaw, err := common.RequireString(params.Params, "subjectType")
	if err != nil {
		return "", pulid.Nil, err
	}
	subjectType := agent.SubjectType(strings.TrimSpace(subjectTypeRaw))
	if !subjectType.IsValid() {
		return "", pulid.Nil, fmt.Errorf(
			"parameter \"subjectType\" is not a known record kind; use one of %s",
			strings.Join(agenttoolschema.SubjectTypes.Names(), ", "),
		)
	}
	subjectID, err := common.RequirePulid(params.Params, common.FieldSubjectID)
	if err != nil {
		return "", pulid.Nil, err
	}
	if err = subjectType.CheckID(subjectID); err != nil {
		return "", pulid.Nil, err
	}

	if t.subjects == nil {
		return "", pulid.Nil, ErrSubjectUncheckable
	}
	exists, err := t.subjects.Exists(ctx, agentports.AgentSubjectExistsRequest{
		TenantInfo:  common.TenantFrom(params),
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
