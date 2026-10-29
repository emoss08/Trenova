package agentevaluation

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentfingerprint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreplay"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Evaluation)(nil)
	_ domaintypes.PostgresSearchable     = (*Evaluation)(nil)
)

const EvaluationIDPrefix = "aeval_"

type EvaluationStatus string

const (
	EvaluationStatusPending   = EvaluationStatus("Pending")
	EvaluationStatusRunning   = EvaluationStatus("Running")
	EvaluationStatusCompleted = EvaluationStatus("Completed")
	EvaluationStatusFailed    = EvaluationStatus("Failed")
	EvaluationStatusSkipped   = EvaluationStatus("Skipped")
)

func AllEvaluationStatuses() []EvaluationStatus {
	return []EvaluationStatus{
		EvaluationStatusPending,
		EvaluationStatusRunning,
		EvaluationStatusCompleted,
		EvaluationStatusFailed,
		EvaluationStatusSkipped,
	}
}

func (s EvaluationStatus) IsValid() bool {
	switch s {
	case EvaluationStatusPending,
		EvaluationStatusRunning,
		EvaluationStatusCompleted,
		EvaluationStatusFailed,
		EvaluationStatusSkipped:
		return true
	default:
		return false
	}
}

func (s EvaluationStatus) Terminal() bool {
	switch s {
	case EvaluationStatusCompleted, EvaluationStatusFailed, EvaluationStatusSkipped:
		return true
	default:
		return false
	}
}

type CheckKind string

const (
	CheckKindHard = CheckKind("hard")
	CheckKindSoft = CheckKind("soft")
)

type CaseCheck struct {
	Name     string    `json:"name"`
	Kind     CheckKind `json:"kind"`
	Applies  bool      `json:"applies"`
	Passed   bool      `json:"passed"`
	Score    float64   `json:"score"`
	Weight   float64   `json:"weight"`
	Detail   string    `json:"detail,omitempty"`
	Findings []string  `json:"findings,omitempty"`
}

type ObservedCall struct {
	ToolName  string         `json:"toolName"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type CaseChecks struct {
	Checks        []CaseCheck    `json:"checks"`
	Calls         []ObservedCall `json:"calls,omitempty"`
	Refused       bool           `json:"refused"`
	HardFailure   bool           `json:"hardFailure"`
	Deterministic float64        `json:"deterministic"`
	Final         float64        `json:"final"`
	Passed        bool           `json:"passed"`
}

func (c *CaseChecks) FailedHard() []CaseCheck {
	if c == nil {
		return nil
	}

	failed := make([]CaseCheck, 0, len(c.Checks))
	for _, check := range c.Checks {
		if check.Kind == CheckKindHard && check.Applies && !check.Passed {
			failed = append(failed, check)
		}
	}

	return failed
}

type JudgeVerdict struct {
	Score     float64 `json:"score"`
	Rationale string  `json:"rationale,omitempty"`
	Model     string  `json:"model,omitempty"`
	JudgedAt  int64   `json:"judgedAt,omitempty"`
}

// Evaluation is one recorded run replayed against its agent as it is now.
//
// The replay runs with every write simulated, so nothing it does reaches a
// record, and the original's proposals are matched against what the replay
// would have proposed. The decisions people made on the original are the
// yardstick: a change the replay drops that a person rejected is an
// improvement, one it drops that a person approved is a regression.
type Evaluation struct {
	keel.Table `keel:"agent_evaluations"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:aeval_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	AgentDefinitionID pulid.ID          `json:"agentDefinitionId" keel:"agent_definition_id"`
	SourceRunID       pulid.ID          `json:"sourceRunId"       keel:"source_run_id,nullzero"`
	EvalCaseID        *pulid.ID         `json:"evalCaseId"        keel:"eval_case_id"`
	Status            EvaluationStatus  `json:"status"            keel:"status"`
	Trigger           agent.RunTrigger  `json:"trigger"           keel:"trigger"`
	SubjectType       agent.SubjectType `json:"subjectType"       keel:"subject_type,nullzero"`
	SubjectID         pulid.ID          `json:"subjectId"         keel:"subject_id,nullzero"`

	Input             string   `json:"input"             keel:"input,nullzero"`
	DefinitionVersion int64    `json:"definitionVersion" keel:"definition_version"`
	PromptVersion     string   `json:"promptVersion"     keel:"prompt_version,nullzero"`
	Model             string   `json:"model"             keel:"model,nullzero"`
	ProviderID        pulid.ID `json:"providerId"        keel:"provider_id,nullzero"`
	Reply             string   `json:"reply"             keel:"reply,nullzero"`

	Actions           []agentreplay.ReplayAction    `json:"actions"           keel:"actions,jsonb"`
	Comparison        *agentreplay.ReplayComparison `json:"comparison"        keel:"comparison,jsonb"`
	OriginalProposals int                           `json:"originalProposals" keel:"original_proposals"`
	ToolCallsUsed     int                           `json:"toolCallsUsed"     keel:"tool_calls_used"`

	Checks      *CaseChecks                   `json:"checks"      keel:"checks,jsonb"`
	Judge       *JudgeVerdict                 `json:"judge"       keel:"judge,jsonb"`
	CaseScore   *float64                      `json:"caseScore"   keel:"case_score"`
	Fingerprint *agentfingerprint.Fingerprint `json:"fingerprint" keel:"fingerprint,jsonb"`

	SuiteRunID   *pulid.ID `json:"suiteRunId"   keel:"suite_run_id"`
	SuiteOrdinal *int      `json:"suiteOrdinal" keel:"suite_ordinal"`

	WorkflowID        string    `json:"workflowId"        keel:"workflow_id,nullzero"`
	ErrorMessage      string    `json:"errorMessage"      keel:"error_message,nullzero"`
	RequestedByUserID *pulid.ID `json:"requestedByUserId" keel:"requested_by_user_id"`
	StartedAt         *int64    `json:"startedAt"         keel:"started_at"`
	CompletedAt       *int64    `json:"completedAt"       keel:"completed_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (e *Evaluation) BeforeCreate(_ context.Context) error {
	if e.Status == "" {
		e.Status = EvaluationStatusPending
	}
	if e.Actions == nil {
		e.Actions = []agentreplay.ReplayAction{}
	}

	return nil
}

func (e *Evaluation) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&e.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&e.AgentDefinitionID, validation.Required.Error("Agent is required")),
		validation.Field(&e.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[EvaluationStatus]("Status is invalid"),
		),
		validation.Field(&e.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[agent.RunTrigger]("Trigger is invalid"),
		),
		validation.Field(&e.SubjectType,
			validation.When(e.ReplaysRun(), validation.Required.Error("Subject type is required")),
			domainvalidation.ValidEnum[agent.SubjectType]("Subject type is invalid"),
		),
		validation.Field(&e.SubjectID,
			validation.When(e.ReplaysRun(), validation.Required.Error("Subject id is required")),
		),
	))

	if e.ReplaysRun() == e.ReplaysCase() {
		multiErr.Add(
			"sourceRunId",
			errortypes.ErrInvalid,
			"An evaluation replays exactly one recorded run or one evaluation case",
		)
	}
}

func (e *Evaluation) ReplaysRun() bool {
	return e.SourceRunID.IsNotNil()
}

func (e *Evaluation) ReplaysCase() bool {
	return e.EvalCaseID != nil && e.EvalCaseID.IsNotNil()
}

func (e *Evaluation) Skip(reason string, at int64) {
	e.Status = EvaluationStatusSkipped
	e.ErrorMessage = reason
	e.CompletedAt = &at
}

func (e *Evaluation) GetID() pulid.ID { return e.ID }

func (e *Evaluation) GetCreatedAt() int64 { return e.CreatedAt }

func (e *Evaluation) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *Evaluation) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *Evaluation) GetTableName() string { return "agent_evaluations" }

func (e *Evaluation) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "aeval",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "reply", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "model", Type: domaintypes.FieldTypeText},
		},
	}
}
