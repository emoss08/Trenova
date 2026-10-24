package agent

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Evaluation)(nil)
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

// ReplayAction is one write the replay would have made: the tool, what it
// was given, and the preview of what it would have changed.
type ReplayAction struct {
	ToolName   string          `json:"toolName"`
	Arguments  map[string]any  `json:"arguments"`
	Rationale  string          `json:"rationale"`
	Tier       AutonomyTier    `json:"tier"`
	Simulation *ToolSimulation `json:"simulation,omitempty"`
}

// Evaluation is one recorded run replayed against its agent as it is now.
//
// The replay runs with every write simulated, so nothing it does reaches a
// record, and the original's proposals are matched against what the replay
// would have proposed. The decisions people made on the original are the
// yardstick: a change the replay drops that a person rejected is an
// improvement, one it drops that a person approved is a regression.
type Evaluation struct {
	bun.BaseModel `bun:"table:agent_evaluations,alias:aeval" json:"-"`
	keel.Table    `keel:"agent_evaluations"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:aeval_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)" keel:"organization_id,pk,tenant"`

	AgentDefinitionID pulid.ID         `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),notnull" keel:"agent_definition_id"`
	SourceRunID       pulid.ID         `json:"sourceRunId"       bun:"source_run_id,type:VARCHAR(100),nullzero" keel:"source_run_id,nullzero"`
	EvalCaseID        *pulid.ID        `json:"evalCaseId"        bun:"eval_case_id,type:VARCHAR(100),nullzero" keel:"eval_case_id"`
	Status            EvaluationStatus `json:"status"            bun:"status,type:VARCHAR(20),notnull,default:'Pending'" keel:"status"`
	Trigger           RunTrigger       `json:"trigger"           bun:"trigger,type:VARCHAR(20),notnull" keel:"trigger"`
	SubjectType       SubjectType      `json:"subjectType"       bun:"subject_type,type:VARCHAR(50),nullzero" keel:"subject_type,nullzero"`
	SubjectID         pulid.ID         `json:"subjectId"         bun:"subject_id,type:VARCHAR(100),nullzero" keel:"subject_id,nullzero"`

	Input             string   `json:"input"             bun:"input,type:TEXT,nullzero" keel:"input,nullzero"`
	DefinitionVersion int64    `json:"definitionVersion" bun:"definition_version,type:BIGINT,notnull" keel:"definition_version"`
	PromptVersion     string   `json:"promptVersion"     bun:"prompt_version,type:VARCHAR(100),nullzero" keel:"prompt_version,nullzero"`
	Model             string   `json:"model"             bun:"model,type:VARCHAR(255),nullzero" keel:"model,nullzero"`
	ProviderID        pulid.ID `json:"providerId"        bun:"provider_id,type:VARCHAR(100),nullzero" keel:"provider_id,nullzero"`
	Reply             string   `json:"reply"             bun:"reply,type:TEXT,nullzero" keel:"reply,nullzero"`

	Actions           []ReplayAction    `json:"actions"           bun:"actions,type:JSONB,notnull,default:'[]'" keel:"actions,jsonb"`
	Comparison        *ReplayComparison `json:"comparison"        bun:"comparison,type:JSONB,nullzero" keel:"comparison,jsonb"`
	OriginalProposals int               `json:"originalProposals" bun:"original_proposals,type:INTEGER,notnull" keel:"original_proposals"`
	ToolCallsUsed     int               `json:"toolCallsUsed"     bun:"tool_calls_used,type:INTEGER,notnull" keel:"tool_calls_used"`

	Checks      *CaseChecks   `json:"checks"      bun:"checks,type:JSONB,nullzero" keel:"checks,jsonb"`
	Judge       *JudgeVerdict `json:"judge"       bun:"judge,type:JSONB,nullzero" keel:"judge,jsonb"`
	CaseScore   *float64      `json:"caseScore"   bun:"case_score,type:DOUBLE PRECISION,nullzero" keel:"case_score"`
	Fingerprint *Fingerprint  `json:"fingerprint" bun:"fingerprint,type:JSONB,nullzero" keel:"fingerprint,jsonb"`

	SuiteRunID   *pulid.ID `json:"suiteRunId"   bun:"suite_run_id,type:VARCHAR(100),nullzero" keel:"suite_run_id"`
	SuiteOrdinal *int      `json:"suiteOrdinal" bun:"suite_ordinal,type:INTEGER,nullzero" keel:"suite_ordinal"`

	WorkflowID        string    `json:"workflowId"        bun:"workflow_id,type:VARCHAR(255),nullzero" keel:"workflow_id,nullzero"`
	ErrorMessage      string    `json:"errorMessage"      bun:"error_message,type:TEXT,nullzero" keel:"error_message,nullzero"`
	RequestedByUserID *pulid.ID `json:"requestedByUserId" bun:"requested_by_user_id,type:VARCHAR(100),nullzero" keel:"requested_by_user_id"`
	StartedAt         *int64    `json:"startedAt"         bun:"started_at,type:BIGINT,nullzero" keel:"started_at"`
	CompletedAt       *int64    `json:"completedAt"       bun:"completed_at,type:BIGINT,nullzero" keel:"completed_at"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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
			domainvalidation.ValidEnum[RunTrigger]("Trigger is invalid"),
		),
		validation.Field(&e.SubjectType,
			validation.When(e.ReplaysRun(), validation.Required.Error("Subject type is required")),
			domainvalidation.ValidEnum[SubjectType]("Subject type is invalid"),
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

func (e *Evaluation) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew(EvaluationIDPrefix)
		}
		if e.Status == "" {
			e.Status = EvaluationStatusPending
		}
		if e.Actions == nil {
			e.Actions = []ReplayAction{}
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}
