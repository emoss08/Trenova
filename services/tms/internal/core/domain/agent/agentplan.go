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
	_ bun.BeforeAppendModelHook          = (*AgentPlan)(nil)
	_ validationframework.TenantedEntity = (*AgentPlan)(nil)
	_ pagination.CursorEntity            = (*AgentPlan)(nil)
	_ domaintypes.PostgresSearchable     = (*AgentPlan)(nil)
)

// AgentPlan is several proposals from one run as one decision.
//
// A run that needs five writes used to produce five cards, each approved on
// its own, in whatever order the approver clicked. A plan is approved once
// and its steps run in the order the agent asked for them; the first step
// that fails stops the plan, and the steps after it are skipped rather than
// run against a world the failed step was meant to change.
type AgentPlan struct {
	bun.BaseModel `bun:"table:agent_plans,alias:apl" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	RunID   pulid.ID   `json:"runId"   bun:"run_id,type:VARCHAR(100),notnull"`
	Title   string     `json:"title"   bun:"title,type:VARCHAR(200),notnull"`
	Summary string     `json:"summary" bun:"summary,type:TEXT,nullzero"`
	Status  PlanStatus `json:"status"  bun:"status,type:VARCHAR(20),notnull,default:'Pending'"`

	StepCount      int    `json:"stepCount"      bun:"step_count,type:INTEGER,notnull"`
	CompletedSteps int    `json:"completedSteps" bun:"completed_steps,type:INTEGER,notnull"`
	FailedStep     *int   `json:"failedStep"     bun:"failed_step,type:INTEGER,nullzero"`
	FailureError   string `json:"failureError"   bun:"failure_error,type:TEXT,nullzero"`

	DecidedByUserID *pulid.ID `json:"decidedByUserId" bun:"decided_by_user_id,type:VARCHAR(100),nullzero"`
	DecidedAt       *int64    `json:"decidedAt"       bun:"decided_at,type:BIGINT,nullzero"`
	// ExpiresAt is the latest expiry among the plan's steps: while any step
	// can still be decided, so can the plan.
	ExpiresAt int64 `json:"expiresAt" bun:"expires_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Run          *AgentRun            `bun:"rel:belongs-to,join:run_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" json:"-"`
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id"                                                                   json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"                                                                    json:"-"`
}

const maxPlanTitleLength = 200

func (p *AgentPlan) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&p.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&p.RunID, validation.Required.Error("Run is required")),
		validation.Field(&p.Title,
			validation.Required.Error("Title is required"),
			validation.Length(1, maxPlanTitleLength).Error("Title cannot be longer than 200 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[PlanStatus]("Status is invalid"),
		),
		validation.Field(&p.StepCount, validation.Min(2).Error("A plan needs at least two steps")),
	))
}

// Expired reports whether the plan's window has closed as of now.
func (p *AgentPlan) Expired(now int64) bool {
	return p.ExpiresAt > 0 && p.ExpiresAt <= now
}

func (p *AgentPlan) GetID() pulid.ID { return p.ID }

func (p *AgentPlan) GetCreatedAt() int64 { return p.CreatedAt }

func (p *AgentPlan) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *AgentPlan) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *AgentPlan) GetTableName() string { return "agent_plans" }

func (p *AgentPlan) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "apl",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "title", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (p *AgentPlan) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("apl_")
		}
		if p.Status == "" {
			p.Status = PlanStatusPending
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

func (*AgentPlan) IsPendingDecision() {}
