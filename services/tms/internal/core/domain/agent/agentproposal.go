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
	ToolParams   map[string]any  `json:"toolParams"   bun:"tool_params,type:JSONB,notnull,default:'{}'::jsonb"`
	Confidence   decimal.Decimal `json:"confidence"   bun:"confidence,type:NUMERIC(5,4),notnull,default:0"`
	Rationale    string          `json:"rationale"    bun:"rationale,type:TEXT,notnull"`
	Evidence     []EvidenceRef   `json:"evidence"     bun:"evidence,type:JSONB,notnull,default:'[]'::jsonb"`
	AutonomyTier AutonomyTier    `json:"autonomyTier" bun:"autonomy_tier,type:agent_autonomy_tier_enum,notnull,default:'Propose'"`
	Status       ProposalStatus  `json:"status"       bun:"status,type:agent_proposal_status_enum,notnull,default:'Pending'"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Run          *AgentRun            `bun:"rel:belongs-to,join:run_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" json:"-"`
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id"                                                                    json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"                                                                     json:"-"`
}

func (p *AgentProposal) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		p,
		validation.Field(&p.RunID, validation.Required.Error("Run id is required")),
		validation.Field(&p.ToolName, validation.Required.Error("Tool name is required")),
		validation.Field(&p.Rationale, validation.Required.Error("Rationale is required")),
		validation.Field(&p.AutonomyTier,
			validation.Required.Error("Autonomy tier is required"),
			validation.By(isValidEnum(p.AutonomyTier.IsValid, "Invalid autonomy tier")),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			validation.By(isValidEnum(p.Status.IsValid, "Invalid status")),
		),
		validation.Field(&p.Confidence,
			validation.By(func(_ any) error {
				if p.Confidence.LessThan(decimal.Zero) || p.Confidence.GreaterThan(decimal.NewFromInt(1)) {
					return errors.New("confidence must be between 0 and 1")
				}
				return nil
			}),
		),
	)

	var validationErrs validation.Errors
	if errors.As(err, &validationErrs) {
		errortypes.FromOzzoErrors(validationErrs, multiErr)
	}

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
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
