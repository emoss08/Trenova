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
	_ bun.BeforeAppendModelHook          = (*AgentException)(nil)
	_ validationframework.TenantedEntity = (*AgentException)(nil)
	_ pagination.CursorEntity            = (*AgentException)(nil)
	_ domaintypes.PostgresSearchable     = (*AgentException)(nil)
)

type AgentException struct {
	bun.BaseModel `bun:"table:agent_exceptions,alias:ax" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	RunID           pulid.ID          `json:"runId"           bun:"run_id,type:VARCHAR(100),notnull"`
	Category        ExceptionCategory `json:"category"        bun:"category,type:agent_exception_category_enum,notnull"`
	Severity        Severity          `json:"severity"        bun:"severity,type:agent_severity_enum,notnull,default:'Medium'"`
	SubjectType     SubjectType       `json:"subjectType"     bun:"subject_type,type:agent_subject_type_enum,notnull"`
	SubjectID       pulid.ID          `json:"subjectId"       bun:"subject_id,type:VARCHAR(100),notnull"`
	AttemptSummary  string            `json:"attemptSummary"  bun:"attempt_summary,type:TEXT,notnull"`
	Evidence        []EvidenceRef     `json:"evidence"        bun:"evidence,type:JSONB,notnull,default:'[]'::jsonb"`
	BlastRadius     int               `json:"blastRadius"     bun:"blast_radius,type:INTEGER,notnull,default:0"`
	ResolutionState ResolutionState   `json:"resolutionState" bun:"resolution_state,type:agent_resolution_state_enum,notnull,default:'Open'"`
	ResolutionNotes string            `json:"resolutionNotes" bun:"resolution_notes,type:TEXT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Run          *AgentRun            `bun:"rel:belongs-to,join:run_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" json:"-"`
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id"                                                                    json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"                                                                     json:"-"`
}

func (e *AgentException) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		e,
		validation.Field(&e.RunID, validation.Required.Error("Run id is required")),
		validation.Field(&e.Category,
			validation.Required.Error("Category is required"),
			validation.By(isValidEnum(e.Category.IsValid, "Invalid category")),
		),
		validation.Field(&e.Severity,
			validation.Required.Error("Severity is required"),
			validation.By(isValidEnum(e.Severity.IsValid, "Invalid severity")),
		),
		validation.Field(&e.SubjectType,
			validation.Required.Error("Subject type is required"),
			validation.By(isValidEnum(e.SubjectType.IsValid, "Invalid subject type")),
		),
		validation.Field(&e.SubjectID, validation.Required.Error("Subject id is required")),
		validation.Field(&e.AttemptSummary,
			validation.Required.Error("Attempt summary is required"),
		),
		validation.Field(&e.ResolutionState,
			validation.Required.Error("Resolution state is required"),
			validation.By(isValidEnum(e.ResolutionState.IsValid, "Invalid resolution state")),
		),
	)

	var validationErrs validation.Errors
	if errors.As(err, &validationErrs) {
		errortypes.FromOzzoErrors(validationErrs, multiErr)
	}

	validateEvidence("evidence", e.Evidence, multiErr)
}

func (e *AgentException) GetID() pulid.ID {
	return e.ID
}

func (e *AgentException) GetCreatedAt() int64 {
	return e.CreatedAt
}

func (e *AgentException) GetOrganizationID() pulid.ID {
	return e.OrganizationID
}

func (e *AgentException) GetBusinessUnitID() pulid.ID {
	return e.BusinessUnitID
}

func (e *AgentException) GetTableName() string {
	return "agent_exceptions"
}

func (e *AgentException) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ax",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "subject_id", Type: domaintypes.FieldTypeText},
			{Name: "category", Type: domaintypes.FieldTypeEnum},
			{Name: "severity", Type: domaintypes.FieldTypeEnum},
			{Name: "resolution_state", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (e *AgentException) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("ax_")
		}
		e.CreatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}
