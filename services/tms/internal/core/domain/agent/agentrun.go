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

	AgentDefinitionID pulid.ID     `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	Trigger           RunTrigger   `json:"trigger"           bun:"trigger,type:VARCHAR(20),notnull,default:'Manual'"`
	Summary           string       `json:"summary"           bun:"summary,type:TEXT,nullzero"`
	AgentType         Type         `json:"agentType"         bun:"agent_type,type:agent_type_enum,notnull"`
	SubjectType       SubjectType  `json:"subjectType"       bun:"subject_type,type:agent_subject_type_enum,notnull"`
	SubjectID         pulid.ID     `json:"subjectId"         bun:"subject_id,type:VARCHAR(100),notnull"`
	Status            RunStatus    `json:"status"            bun:"status,type:agent_run_status_enum,notnull,default:'Pending'"`
	WorkflowID        string       `json:"workflowId"        bun:"workflow_id,type:VARCHAR(255),nullzero"`
	ModelIdentifier   string       `json:"modelIdentifier"   bun:"model_identifier,type:VARCHAR(255),nullzero"`
	PromptVersion     string       `json:"promptVersion"     bun:"prompt_version,type:VARCHAR(100),notnull"`
	InputContextHash  string       `json:"inputContextHash"  bun:"input_context_hash,type:VARCHAR(64),notnull"`
	StartedAt         int64        `json:"startedAt"         bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt       *int64       `json:"completedAt"       bun:"completed_at,type:BIGINT,nullzero"`
	ErrorMessage      string       `json:"errorMessage"      bun:"error_message,type:TEXT,nullzero"`
	Tainted           bool         `json:"tainted"           bun:"tainted,type:BOOLEAN,notnull,default:false"`
	Taint             *RunTaint    `json:"taint"             bun:"taint,type:JSONB,nullzero"`
	TaintedAt         *int64       `json:"taintedAt"         bun:"tainted_at,type:BIGINT,nullzero"`
	Fingerprint       *Fingerprint `json:"fingerprint"       bun:"fingerprint,type:JSONB,nullzero"`

	// TraceID is the run's trace. TurnID is the conversation turn a run was
	// opened for. ParentOwnerKind, ParentOwnerID and DelegateCallID are set on
	// a delegate's run: the run or turn that handed it the task, and the call
	// that did.
	TraceID         string       `json:"traceId"         bun:"trace_id,type:VARCHAR(32),nullzero"`
	TurnID          pulid.ID     `json:"turnId"          bun:"turn_id,type:VARCHAR(100),nullzero"`
	ParentOwnerKind RunOwnerKind `json:"parentOwnerKind" bun:"parent_owner_kind,type:VARCHAR(20),nullzero"`
	ParentOwnerID   pulid.ID     `json:"parentOwnerId"   bun:"parent_owner_id,type:VARCHAR(100),nullzero"`
	DelegateCallID  string       `json:"delegateCallId"  bun:"delegate_call_id,type:VARCHAR(200),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (r *AgentRun) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		r,
		validation.Field(&r.AgentType,
			validation.Required.Error("Agent type is required"),
			domainvalidation.ValidEnum[Type]("Invalid agent type"),
		),
		validation.Field(&r.SubjectType,
			validation.Required.Error("Subject type is required"),
			domainvalidation.ValidEnum[SubjectType]("Invalid subject type"),
		),
		validation.Field(&r.SubjectID, validation.Required.Error("Subject id is required")),
		validation.Field(&r.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[RunTrigger]("Invalid trigger"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[RunStatus]("Invalid status"),
		),
		validation.Field(&r.PromptVersion,
			validation.Required.Error("Prompt version is required"),
		),
		validation.Field(&r.InputContextHash,
			validation.Required.Error("Input context hash is required"),
		),
		validation.Field(&r.TraceID, domainvalidation.TraceID("Trace id is invalid")),
		validation.Field(&r.ParentOwnerKind,
			domainvalidation.ValidEnum[RunOwnerKind]("Invalid parent owner kind"),
		),
	))

	r.validateParentOwner(multiErr)
}

func (r *AgentRun) validateParentOwner(multiErr *errortypes.MultiError) {
	if (r.ParentOwnerKind == "") != r.ParentOwnerID.IsNil() {
		multiErr.Add(
			"parentOwnerId",
			errortypes.ErrInvalid,
			"Parent owner kind and parent owner id must be set together",
		)
	}

	if r.DelegateCallID != "" && r.ParentOwnerID.IsNil() {
		multiErr.Add(
			"delegateCallId",
			errortypes.ErrInvalid,
			"A delegate call requires the run or turn that made it",
		)
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

func (r *AgentRun) RecordTaint(taint *RunTaint, now int64) bool {
	if !taint.Tainted() {
		return false
	}

	merged := r.Taint.Clone()
	if merged == nil {
		merged = &RunTaint{}
	}
	before := len(merged.Marks)
	merged.Merge(taint)
	changed := !r.Tainted || len(merged.Marks) != before
	r.Tainted = true
	r.Taint = merged
	if r.TaintedAt == nil {
		r.TaintedAt = &now
	}

	return changed
}

func (r *AgentRun) TaintedRecords() []RecordRef {
	if r == nil || !r.Tainted {
		return nil
	}

	return []RecordRef{{EntityType: TaintEntityAgentRun, ID: r.ID.String()}}
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
		if r.Trigger == "" {
			r.Trigger = RunTriggerManual
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}
