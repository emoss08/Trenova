package agent

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AgentRun)(nil)
	_ validationframework.TenantedEntity = (*AgentRun)(nil)
	_ pagination.CursorEntity            = (*AgentRun)(nil)
	_ domaintypes.PostgresSearchable     = (*AgentRun)(nil)
)

type AgentRun struct {
	bun.BaseModel `bun:"table:agent_runs,alias:ar" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	AgentType        Type        `json:"agentType"        bun:"agent_type,type:agent_type_enum,notnull"`
	SubjectType      SubjectType `json:"subjectType"      bun:"subject_type,type:agent_subject_type_enum,notnull"`
	SubjectID        pulid.ID    `json:"subjectId"        bun:"subject_id,type:VARCHAR(100),notnull"`
	Status           RunStatus   `json:"status"           bun:"status,type:agent_run_status_enum,notnull,default:'Pending'"`
	WorkflowID       string      `json:"workflowId"       bun:"workflow_id,type:VARCHAR(255),nullzero"`
	ModelIdentifier  string      `json:"modelIdentifier"  bun:"model_identifier,type:VARCHAR(255),nullzero"`
	PromptVersion    string      `json:"promptVersion"    bun:"prompt_version,type:VARCHAR(100),notnull"`
	InputContextHash string      `json:"inputContextHash" bun:"input_context_hash,type:VARCHAR(64),notnull"`
	StartedAt        int64       `json:"startedAt"        bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt      *int64      `json:"completedAt"      bun:"completed_at,type:BIGINT,nullzero"`
	ErrorMessage     string      `json:"errorMessage"     bun:"error_message,type:TEXT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (r *AgentRun) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		r,
		validation.Field(&r.AgentType,
			validation.Required.Error("Agent type is required"),
			validation.By(isValidEnum(r.AgentType.IsValid, "Invalid agent type")),
		),
		validation.Field(&r.SubjectType,
			validation.Required.Error("Subject type is required"),
			validation.By(isValidEnum(r.SubjectType.IsValid, "Invalid subject type")),
		),
		validation.Field(&r.SubjectID, validation.Required.Error("Subject id is required")),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.By(isValidEnum(r.Status.IsValid, "Invalid status")),
		),
		validation.Field(&r.PromptVersion,
			validation.Required.Error("Prompt version is required"),
		),
		validation.Field(&r.InputContextHash,
			validation.Required.Error("Input context hash is required"),
		),
	)

	var validationErrs validation.Errors
	if errors.As(err, &validationErrs) {
		errortypes.FromOzzoErrors(validationErrs, multiErr)
	}
}

func (r *AgentRun) GetID() pulid.ID {
	return r.ID
}

func (r *AgentRun) GetCreatedAt() int64 {
	return r.CreatedAt
}

func (r *AgentRun) GetOrganizationID() pulid.ID {
	return r.OrganizationID
}

func (r *AgentRun) GetBusinessUnitID() pulid.ID {
	return r.BusinessUnitID
}

func (r *AgentRun) GetTableName() string {
	return "agent_runs"
}

func (r *AgentRun) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ar",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "subject_id", Type: domaintypes.FieldTypeText},
			{Name: "workflow_id", Type: domaintypes.FieldTypeText},
			{Name: "agent_type", Type: domaintypes.FieldTypeEnum},
			{Name: "subject_type", Type: domaintypes.FieldTypeEnum},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (r *AgentRun) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("ar_")
		}
		r.CreatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}
