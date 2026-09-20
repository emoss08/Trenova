package agent

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AgentProposal)(nil)
	_ validationframework.TenantedEntity = (*AgentProposal)(nil)
	_ pagination.CursorEntity            = (*AgentProposal)(nil)
	_ domaintypes.PostgresSearchable     = (*AgentProposal)(nil)
)

type AgentProposal struct {
	bun.BaseModel `bun:"table:agent_proposals,alias:ap" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	RunID        pulid.ID        `json:"runId"        bun:"run_id,type:VARCHAR(100),notnull"`
	ToolName     string          `json:"toolName"     bun:"tool_name,type:VARCHAR(100),notnull"`
	ToolParams   map[string]any  `json:"toolParams"   bun:"tool_params,type:JSONB,notnull,default:'{}'"`
	Confidence   decimal.Decimal `json:"confidence"   bun:"confidence,type:NUMERIC(5,4),notnull,default:0"`
	Rationale    string          `json:"rationale"    bun:"rationale,type:TEXT,notnull"`
	Evidence     []EvidenceRef   `json:"evidence"     bun:"evidence,type:JSONB,notnull,default:'[]'"`
	AutonomyTier AutonomyTier    `json:"autonomyTier" bun:"autonomy_tier,type:agent_autonomy_tier_enum,notnull,default:'Propose'"`
	Status       ProposalStatus  `json:"status"       bun:"status,type:agent_proposal_status_enum,notnull,default:'Pending'"`

	// ExecutedAt and ExecutionError record what happened after approval. An
	// accepted proposal with neither set has been approved but has not run.
	ExecutedAt      *int64   `json:"executedAt"      bun:"executed_at,type:BIGINT,nullzero"`
	ExecutionError  string   `json:"executionError"  bun:"execution_error,type:TEXT,nullzero"`
	SourceMessageID pulid.ID `json:"sourceMessageId" bun:"source_message_id,type:VARCHAR(100),nullzero"`

	// ExpiresAt is when a pending proposal stops being decidable. A proposal
	// is a judgement about the world as it was; past this point that world is
	// gone and the judgement with it.
	ExpiresAt int64 `json:"expiresAt" bun:"expires_at,type:BIGINT,nullzero"`

	// TargetResource, TargetID and TargetVersion pin the record the proposal
	// would change, as it was when proposed. The executor refuses to run
	// against a different version: a hold proposed on a shipment that has
	// since been delivered is not the change anyone approved.
	TargetResource string   `json:"targetResource" bun:"target_resource,type:VARCHAR(100),nullzero"`
	TargetID       pulid.ID `json:"targetId"       bun:"target_id,type:VARCHAR(100),nullzero"`
	TargetVersion  int64    `json:"targetVersion"  bun:"target_version,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Run          *AgentRun            `bun:"rel:belongs-to,join:run_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" json:"-"`
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id"                                                                   json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"                                                                    json:"-"`
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
	))

	validateEvidence("evidence", p.Evidence, multiErr)
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

func (p *AgentProposal) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("ap_")
		}
		p.CreatedAt = now
		if p.ExpiresAt == 0 {
			p.ExpiresAt = now + int64(DefaultProposalTTL.Seconds())
		}
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
