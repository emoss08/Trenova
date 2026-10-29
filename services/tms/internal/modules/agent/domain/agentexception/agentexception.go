package agentexception

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevidence"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*AgentException)(nil)
	_ pagination.CursorEntity            = (*AgentException)(nil)
	_ domaintypes.PostgresSearchable     = (*AgentException)(nil)
)

type AgentException struct {
	keel.Table `keel:"agent_exceptions"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:ax_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RunID           pulid.ID                    `json:"runId"           keel:"run_id"`
	Category        ExceptionCategory           `json:"category"        keel:"category"`
	Severity        Severity                    `json:"severity"        keel:"severity"`
	SubjectType     agent.SubjectType           `json:"subjectType"     keel:"subject_type"`
	SubjectID       pulid.ID                    `json:"subjectId"       keel:"subject_id"`
	AttemptSummary  string                      `json:"attemptSummary"  keel:"attempt_summary"`
	Evidence        []agentevidence.EvidenceRef `json:"evidence"        keel:"evidence,jsonb"`
	BlastRadius     int                         `json:"blastRadius"     keel:"blast_radius"`
	ResolutionState ResolutionState             `json:"resolutionState" keel:"resolution_state"`
	ResolutionNotes string                      `json:"resolutionNotes" keel:"resolution_notes,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Run          *agent.AgentRun      `json:"-" keel:"belongs:run_id"`
	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (e *AgentException) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		e,
		validation.Field(&e.RunID, validation.Required.Error("Run id is required")),
		validation.Field(&e.Category,
			validation.Required.Error("Category is required"),
			domainvalidation.ValidEnum[ExceptionCategory]("Invalid category"),
		),
		validation.Field(&e.Severity,
			validation.Required.Error("Severity is required"),
			domainvalidation.ValidEnum[Severity]("Invalid severity"),
		),
		validation.Field(&e.SubjectType,
			validation.Required.Error("Subject type is required"),
			domainvalidation.ValidEnum[agent.SubjectType]("Invalid subject type"),
		),
		validation.Field(&e.SubjectID, validation.Required.Error("Subject id is required")),
		validation.Field(&e.AttemptSummary,
			validation.Required.Error("Attempt summary is required"),
		),
		validation.Field(&e.ResolutionState,
			validation.Required.Error("Resolution state is required"),
			domainvalidation.ValidEnum[ResolutionState]("Invalid resolution state"),
		),
	))

	agentevidence.ValidateRefs("evidence", e.Evidence, multiErr)
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

func (e *AgentException) BeforeCreate(ctx context.Context) error {
	if e.Severity == "" {
		e.Severity = SeverityMedium
	}
	if e.ResolutionState == "" {
		e.ResolutionState = ResolutionStateOpen
	}

	return nil
}
