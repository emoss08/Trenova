package agent

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AgentRunStep)(nil)
	_ validationframework.TenantedEntity = (*AgentRunStep)(nil)
)

// RunOwnerKind names the unit of work a step, a usage row or a delegate run
// belongs to: a background agent run or a conversation turn.
type RunOwnerKind string

const (
	RunOwnerAgentRun      = RunOwnerKind("AgentRun")
	RunOwnerAssistantTurn = RunOwnerKind("AssistantTurn")
)

func (k RunOwnerKind) IsValid() bool {
	switch k {
	case RunOwnerAgentRun, RunOwnerAssistantTurn:
		return true
	default:
		return false
	}
}

func AllRunOwnerKinds() []RunOwnerKind {
	return []RunOwnerKind{RunOwnerAgentRun, RunOwnerAssistantTurn}
}

// AgentRunStep is one thing a run did, or began to do.
//
// The rows exist so an activity that is retried does not redo the work of the
// attempt before it. A step is claimed before its tool runs and settled after,
// so a row left Started names a write whose outcome nobody recorded — the one
// case the run has to report rather than repeat.
type AgentRunStep struct {
	bun.BaseModel `bun:"table:agent_run_steps,alias:ars" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	// OwnerKind and OwnerID name the run or conversation turn this belongs to.
	// There is no foreign key, because one column cannot point at two tables;
	// the rows are swept on a retention schedule instead of cascaded.
	OwnerKind string   `json:"ownerKind" bun:"owner_kind,type:VARCHAR(20),notnull"`
	OwnerID   pulid.ID `json:"ownerId"   bun:"owner_id,type:VARCHAR(100),notnull"`

	Attempt  int    `json:"attempt"  bun:"attempt,type:INTEGER,notnull,default:1"`
	Kind     string `json:"kind"     bun:"kind,type:VARCHAR(20),notnull"`
	Status   string `json:"status"   bun:"status,type:VARCHAR(20),notnull"`
	StepKey  string `json:"stepKey"  bun:"step_key,type:VARCHAR(120),notnull"`
	ToolName string `json:"toolName" bun:"tool_name,type:VARCHAR(200),notnull,default:''"`
	CallID   string `json:"callId"   bun:"call_id,type:VARCHAR(200),notnull,default:''"`

	Arguments map[string]any `json:"arguments" bun:"arguments,type:JSONB,notnull,default:'{}'::jsonb"`
	Outcome   map[string]any `json:"outcome"   bun:"outcome,type:JSONB,notnull,default:'{}'::jsonb"`

	// TraceID and SpanID are the span the step ran in. AgentDefinitionID and
	// AgentDefinitionVersion are the agent that made the call as it was then,
	// which for a delegate's step is not the agent the run belongs to, and
	// DelegateCallID the task it was working on.
	TraceID                string   `json:"traceId"                bun:"trace_id,type:VARCHAR(32),nullzero"`
	SpanID                 string   `json:"spanId"                 bun:"span_id,type:VARCHAR(16),nullzero"`
	AgentDefinitionID      pulid.ID `json:"agentDefinitionId"      bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	AgentDefinitionVersion *int64   `json:"agentDefinitionVersion" bun:"agent_definition_version,type:BIGINT,nullzero"`
	DelegateCallID         string   `json:"delegateCallId"         bun:"delegate_call_id,type:VARCHAR(200),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (s *AgentRunStep) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		s,
		validation.Field(&s.OwnerKind,
			validation.Required.Error("Owner kind is required"),
			validation.In("AgentRun", "AssistantTurn").Error("Invalid owner kind"),
		),
		validation.Field(&s.OwnerID, validation.Required.Error("Owner id is required")),
		validation.Field(&s.Kind,
			validation.Required.Error("Kind is required"),
			validation.In("Tool", "Completion").Error("Invalid kind"),
		),
		validation.Field(&s.Status,
			validation.Required.Error("Status is required"),
			validation.In("Started", "Completed", "Failed").Error("Invalid status"),
		),
		validation.Field(&s.StepKey, validation.Required.Error("Step key is required")),
		validation.Field(&s.TraceID, domainvalidation.TraceID("Trace id is invalid")),
		validation.Field(&s.SpanID, domainvalidation.SpanID("Span id is invalid")),
		validation.Field(&s.AgentDefinitionVersion,
			validation.Min(int64(0)).Error("Agent definition version cannot be negative"),
		),
	))
}

func (s *AgentRunStep) GetID() pulid.ID {
	return s.ID
}

func (s *AgentRunStep) GetOrganizationID() pulid.ID {
	return s.OrganizationID
}

func (s *AgentRunStep) GetBusinessUnitID() pulid.ID {
	return s.BusinessUnitID
}

func (s *AgentRunStep) GetTableName() string {
	return "agent_run_steps"
}

func (s *AgentRunStep) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("ars_")
		}
		s.CreatedAt = now
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}
