package agent

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentfingerprint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
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
	_ validationframework.TenantedEntity = (*AgentRun)(nil)
	_ pagination.CursorEntity            = (*AgentRun)(nil)
	_ domaintypes.PostgresSearchable     = (*AgentRun)(nil)
)

type AgentRun struct {
	keel.Table `keel:"agent_runs"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:ar_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	AgentDefinitionID pulid.ID                      `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	Trigger           RunTrigger                    `json:"trigger"           keel:"trigger"`
	Summary           string                        `json:"summary"           keel:"summary,nullzero"`
	AgentType         Type                          `json:"agentType"         keel:"agent_type"`
	SubjectType       SubjectType                   `json:"subjectType"       keel:"subject_type"`
	SubjectID         pulid.ID                      `json:"subjectId"         keel:"subject_id"`
	Status            RunStatus                     `json:"status"            keel:"status"`
	WorkflowID        string                        `json:"workflowId"        keel:"workflow_id,nullzero"`
	ModelIdentifier   string                        `json:"modelIdentifier"   keel:"model_identifier,nullzero"`
	PromptVersion     string                        `json:"promptVersion"     keel:"prompt_version"`
	InputContextHash  string                        `json:"inputContextHash"  keel:"input_context_hash"`
	StartedAt         int64                         `json:"startedAt"         keel:"started_at,nullzero"`
	CompletedAt       *int64                        `json:"completedAt"       keel:"completed_at"`
	ErrorMessage      string                        `json:"errorMessage"      keel:"error_message,nullzero"`
	Tainted           bool                          `json:"tainted"           keel:"tainted"`
	Taint             *RunTaint                     `json:"taint"             keel:"taint,jsonb"`
	TaintedAt         *int64                        `json:"taintedAt"         keel:"tainted_at"`
	Fingerprint       *agentfingerprint.Fingerprint `json:"fingerprint"       keel:"fingerprint,jsonb"`

	Transcript *RunTranscript `json:"-" keel:"transcript,jsonb"`

	// TraceID is the run's trace. TurnID is the conversation turn a run was
	// opened for. ParentOwnerKind, ParentOwnerID and DelegateCallID are set on
	// a delegate's run: the run or turn that handed it the task, and the call
	// that did.
	TraceID         string       `json:"traceId"         keel:"trace_id,nullzero"`
	TurnID          pulid.ID     `json:"turnId"          keel:"turn_id,nullzero"`
	ParentOwnerKind RunOwnerKind `json:"parentOwnerKind" keel:"parent_owner_kind,nullzero"`
	ParentOwnerID   pulid.ID     `json:"parentOwnerId"   keel:"parent_owner_id,nullzero"`
	DelegateCallID  string       `json:"delegateCallId"  keel:"delegate_call_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
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

func (r *AgentRun) TaintedRecords() []agentrecord.RecordRef {
	if r == nil || !r.Tainted {
		return nil
	}

	return []agentrecord.RecordRef{{EntityType: TaintEntityAgentRun, ID: r.ID.String()}}
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

func (r *AgentRun) BeforeCreate(ctx context.Context) error {
	if r.Trigger == "" {
		r.Trigger = RunTriggerManual
	}
	if r.Status == "" {
		r.Status = RunStatusPending
	}

	return nil
}
