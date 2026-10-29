package agentrunstep

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*AgentRunStep)(nil)
)

// AgentRunStep is one thing a run did, or began to do.
//
// The rows exist so an activity that is retried does not redo the work of the
// attempt before it. A step is claimed before its tool runs and settled after,
// so a row left Started names a write whose outcome nobody recorded — the one
// case the run has to report rather than repeat.
type AgentRunStep struct {
	keel.Table `keel:"agent_run_steps"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:ars_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// OwnerKind and OwnerID name the run or conversation turn this belongs to.
	// There is no foreign key, because one column cannot point at two tables;
	// the rows are swept on a retention schedule instead of cascaded.
	OwnerKind string   `json:"ownerKind" keel:"owner_kind"`
	OwnerID   pulid.ID `json:"ownerId"   keel:"owner_id"`

	Attempt  int    `json:"attempt"  keel:"attempt"`
	Kind     string `json:"kind"     keel:"kind"`
	Status   string `json:"status"   keel:"status"`
	StepKey  string `json:"stepKey"  keel:"step_key"`
	ToolName string `json:"toolName" keel:"tool_name"`
	CallID   string `json:"callId"   keel:"call_id"`

	Arguments map[string]any `json:"arguments" keel:"arguments,jsonb"`
	Outcome   map[string]any `json:"outcome"   keel:"outcome,jsonb"`

	// TraceID and SpanID are the span the step ran in. AgentDefinitionID and
	// AgentDefinitionVersion are the agent that made the call as it was then,
	// which for a delegate's step is not the agent the run belongs to, and
	// DelegateCallID the task it was working on.
	TraceID                string   `json:"traceId"                keel:"trace_id,nullzero"`
	SpanID                 string   `json:"spanId"                 keel:"span_id,nullzero"`
	AgentDefinitionID      pulid.ID `json:"agentDefinitionId"      keel:"agent_definition_id,nullzero"`
	AgentDefinitionVersion *int64   `json:"agentDefinitionVersion" keel:"agent_definition_version"`
	DelegateCallID         string   `json:"delegateCallId"         keel:"delegate_call_id,nullzero"`

	StartedAtMs *int64 `json:"startedAtMs" keel:"started_at_ms,readonly"`
	SettledAtMs *int64 `json:"settledAtMs" keel:"settled_at_ms"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (s *AgentRunStep) BeforeCreate(_ context.Context) error {
	if s.Attempt == 0 {
		s.Attempt = 1
	}

	return nil
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
