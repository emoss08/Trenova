package agent

import (
	"context"
	"errors"
	"slices"
	"time"

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
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*AgentProposal)(nil)
	_ pagination.CursorEntity            = (*AgentProposal)(nil)
	_ domaintypes.PostgresSearchable     = (*AgentProposal)(nil)
)

type AgentProposal struct {
	keel.Table `keel:"agent_proposals"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:ap_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RunID        pulid.ID        `json:"runId"        keel:"run_id"`
	ToolName     string          `json:"toolName"     keel:"tool_name"`
	ToolParams   map[string]any  `json:"toolParams"   keel:"tool_params,jsonb"`
	Confidence   decimal.Decimal `json:"confidence"   keel:"confidence"`
	Rationale    string          `json:"rationale"    keel:"rationale"`
	Evidence     []EvidenceRef   `json:"evidence"     keel:"evidence,jsonb"`
	AutonomyTier AutonomyTier    `json:"autonomyTier" keel:"autonomy_tier"`
	Status       ProposalStatus  `json:"status"       keel:"status"`

	// ExecutedAt and ExecutionError record what happened after approval. An
	// accepted proposal with neither set has been approved but has not run.
	// ExecutionResult is what a successful run made, when the tool says: the
	// record and its id, so later turns refer to it and not to the proposal.
	ExecutedAt      *int64               `json:"executedAt"      keel:"executed_at"`
	ExecutionError  string               `json:"executionError"  keel:"execution_error,nullzero"`
	ExecutionResult *ToolExecutionResult `json:"executionResult" keel:"execution_result,jsonb"`
	SourceMessageID pulid.ID             `json:"sourceMessageId" keel:"source_message_id,nullzero"`

	// ExpiresAt is when a pending proposal stops being decidable. A proposal
	// is a judgement about the world as it was; past this point that world is
	// gone and the judgement with it.
	ExpiresAt int64 `json:"expiresAt" keel:"expires_at,nullzero"`

	// SimulatedAt and Simulation are set instead of an execution when the
	// agent was in simulation: what the write would have changed, and when
	// that was worked out.
	SimulatedAt *int64          `json:"simulatedAt" keel:"simulated_at"`
	Simulation  *ToolSimulation `json:"simulation"  keel:"simulation,jsonb"`

	// PlanID and PlanStep tie a proposal to the plan it is a step of and say
	// where in the order it sits. A proposal decided on its own has neither.
	PlanID   *pulid.ID `json:"planId"   keel:"plan_id"`
	PlanStep int       `json:"planStep" keel:"plan_step"`

	// RemindedAt is when the people who could decide this proposal were told,
	// a second time, that it was still waiting. Null until they are.
	RemindedAt *int64 `json:"remindedAt" keel:"reminded_at"`

	// TargetResource, TargetID and TargetVersion pin the record the proposal
	// would change, as it was when proposed. The executor refuses to run
	// against a different version: a hold proposed on a shipment that has
	// since been delivered is not the change anyone approved.
	TargetResource string   `json:"targetResource" keel:"target_resource,nullzero"`
	TargetID       pulid.ID `json:"targetId"       keel:"target_id,nullzero"`
	TargetVersion  int64    `json:"targetVersion"  keel:"target_version,nullzero"`

	Tainted       bool        `json:"tainted"       keel:"tainted"`
	Taint         *RunTaint   `json:"taint"         keel:"taint,jsonb"`
	EgressClass   EgressClass `json:"egressClass"   keel:"egress_class,nullzero"`
	HeldBy        []string    `json:"heldBy"        keel:"held_by"`
	MoneyMinor    *int64      `json:"moneyMinor"    keel:"money_minor"`
	MoneyCurrency string      `json:"moneyCurrency" keel:"money_currency,nullzero"`

	// TraceID and SpanID are the tool span the write was decided in, so a
	// decision made later links back to it; StepKey is the ledger step that
	// raised it. ExecutedByUserID and ExecutedTargetVersion say who the write
	// ran as and what version it left the record at.
	TraceID               string   `json:"traceId"               keel:"trace_id,nullzero"`
	SpanID                string   `json:"spanId"                keel:"span_id,nullzero"`
	StepKey               string   `json:"stepKey"               keel:"step_key,nullzero"`
	ExecutedByUserID      pulid.ID `json:"executedByUserId"      keel:"executed_by_user_id,nullzero"`
	ExecutedTargetVersion *int64   `json:"executedTargetVersion" keel:"executed_target_version"`

	// ScriptStepKey is the ledger step of the script that wrote this proposal
	// as an intent; empty for a write the model called itself.
	ScriptStepKey string `json:"scriptStepKey" keel:"script_step_key,nullzero"`

	// PendingModifications is the wording a person changed on the proposal
	// and has not yet approved with, keyed by parameter, kept so the edit
	// survives a reload. Set only while the proposal is pending; deciding it
	// clears it, and the decision keeps what was approved.
	PendingModifications map[string]any `json:"pendingModifications" keel:"pending_modifications,jsonb,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Run          *AgentRun            `json:"-" keel:"belongs:run_id"`
	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

// DefaultProposalTTL is how long a proposal waits for a decision before it
// expires on its own. A week is long enough to survive a weekend and a holiday
// and short enough that the record it refers to is usually still the record
// it described.
const DefaultProposalTTL = 7 * 24 * time.Hour

// Expired reports whether a proposal's window has closed as of now.
func (p *AgentProposal) Expired(now int64) bool {
	return p.ExpiresAt > 0 && p.ExpiresAt <= now
}

func (p *AgentProposal) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		p,
		validation.Field(&p.RunID, validation.Required.Error("Run id is required")),
		validation.Field(&p.ToolName, validation.Required.Error("Tool name is required")),
		validation.Field(&p.Rationale, validation.Required.Error("Rationale is required")),
		validation.Field(&p.AutonomyTier,
			validation.Required.Error("Autonomy tier is required"),
			domainvalidation.ValidEnum[AutonomyTier]("Invalid autonomy tier"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ProposalStatus]("Invalid status"),
		),
		validation.Field(&p.Confidence,
			validation.By(func(_ any) error {
				if p.Confidence.LessThan(decimal.Zero) ||
					p.Confidence.GreaterThan(decimal.NewFromInt(1)) {
					return errors.New("confidence must be between 0 and 1")
				}
				return nil
			}),
		),
		validation.Field(&p.TraceID, domainvalidation.TraceID("Trace id is invalid")),
		validation.Field(&p.SpanID, domainvalidation.SpanID("Span id is invalid")),
		validation.Field(&p.ExecutedTargetVersion,
			validation.Min(int64(0)).Error("Executed target version cannot be negative"),
		),
	))

	validateEvidence("evidence", p.Evidence, multiErr)

	if p.EgressClass != "" && !p.EgressClass.IsValid() {
		multiErr.Add("egressClass", errortypes.ErrInvalid, "Egress class is invalid")
	}
}

func (p *AgentProposal) RequiresPerson(class EgressClass) bool {
	return p.Tainted && (class.Leaves() || slices.Contains(p.HeldBy, HeldByTaintKey))
}

func (p *AgentProposal) GetID() pulid.ID {
	return p.ID
}

func (p *AgentProposal) GetCreatedAt() int64 {
	return p.CreatedAt
}

func (p *AgentProposal) GetOrganizationID() pulid.ID {
	return p.OrganizationID
}

func (p *AgentProposal) GetBusinessUnitID() pulid.ID {
	return p.BusinessUnitID
}

func (p *AgentProposal) GetTableName() string {
	return "agent_proposals"
}

func (p *AgentProposal) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ap",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "tool_name", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "autonomy_tier", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (p *AgentProposal) BeforeCreate(ctx context.Context) error {
	if p.AutonomyTier == "" {
		p.AutonomyTier = TierPropose
	}
	if p.Status == "" {
		p.Status = ProposalStatusPending
	}
	now := timeutils.NowUnix()
	if p.ExpiresAt == 0 {
		p.ExpiresAt = now + int64(DefaultProposalTTL.Seconds())
	}
	if p.HeldBy == nil {
		p.HeldBy = []string{}
	}

	return nil
}

func (*AgentProposal) IsPendingDecision() {}
