package agent

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
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

type EvaluationStatus string

const (
	EvaluationStatusPending   = EvaluationStatus("Pending")
	EvaluationStatusRunning   = EvaluationStatus("Running")
	EvaluationStatusCompleted = EvaluationStatus("Completed")
	EvaluationStatusFailed    = EvaluationStatus("Failed")
)

func (s EvaluationStatus) IsValid() bool {
	switch s {
	case EvaluationStatusPending,
		EvaluationStatusRunning,
		EvaluationStatusCompleted,
		EvaluationStatusFailed:
		return true
	default:
		return false
	}
}

func (s EvaluationStatus) Terminal() bool {
	return s == EvaluationStatusCompleted || s == EvaluationStatusFailed
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

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	AgentDefinitionID pulid.ID         `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),notnull"`
	SourceRunID       pulid.ID         `json:"sourceRunId"       bun:"source_run_id,type:VARCHAR(100),notnull"`
	Status            EvaluationStatus `json:"status"            bun:"status,type:VARCHAR(20),notnull,default:'Pending'"`
	Trigger           RunTrigger       `json:"trigger"           bun:"trigger,type:VARCHAR(20),notnull"`
	SubjectType       SubjectType      `json:"subjectType"       bun:"subject_type,type:VARCHAR(50),notnull"`
	SubjectID         pulid.ID         `json:"subjectId"         bun:"subject_id,type:VARCHAR(100),notnull"`

	Input             string   `json:"input"             bun:"input,type:TEXT,nullzero"`
	DefinitionVersion int64    `json:"definitionVersion" bun:"definition_version,type:BIGINT,notnull"`
	PromptVersion     string   `json:"promptVersion"     bun:"prompt_version,type:VARCHAR(100),nullzero"`
	Model             string   `json:"model"             bun:"model,type:VARCHAR(255),nullzero"`
	ProviderID        pulid.ID `json:"providerId"        bun:"provider_id,type:VARCHAR(100),nullzero"`
	Reply             string   `json:"reply"             bun:"reply,type:TEXT,nullzero"`

	Actions           []ReplayAction    `json:"actions"           bun:"actions,type:JSONB,notnull,default:'[]'"`
	Comparison        *ReplayComparison `json:"comparison"        bun:"comparison,type:JSONB,nullzero"`
	OriginalProposals int               `json:"originalProposals" bun:"original_proposals,type:INTEGER,notnull"`
	ToolCallsUsed     int               `json:"toolCallsUsed"     bun:"tool_calls_used,type:INTEGER,notnull"`

	WorkflowID        string    `json:"workflowId"        bun:"workflow_id,type:VARCHAR(255),nullzero"`
	ErrorMessage      string    `json:"errorMessage"      bun:"error_message,type:TEXT,nullzero"`
	RequestedByUserID *pulid.ID `json:"requestedByUserId" bun:"requested_by_user_id,type:VARCHAR(100),nullzero"`
	StartedAt         *int64    `json:"startedAt"         bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt       *int64    `json:"completedAt"       bun:"completed_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (e *Evaluation) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&e.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&e.AgentDefinitionID, validation.Required.Error("Agent is required")),
		validation.Field(&e.SourceRunID, validation.Required.Error("A run to replay is required")),
		validation.Field(&e.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[EvaluationStatus]("Status is invalid"),
		),
		validation.Field(&e.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[RunTrigger]("Trigger is invalid"),
		),
		validation.Field(&e.SubjectType,
			validation.Required.Error("Subject type is required"),
			domainvalidation.ValidEnum[SubjectType]("Subject type is invalid"),
		),
		validation.Field(&e.SubjectID, validation.Required.Error("Subject id is required")),
	))
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
			e.ID = pulid.MustNew("aeval_")
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
