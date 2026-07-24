package agent

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AgentDecision)(nil)
	_ validationframework.TenantedEntity = (*AgentDecision)(nil)
	_ pagination.CursorEntity            = (*AgentDecision)(nil)
)

type AgentDecision struct {
	bun.BaseModel `bun:"table:agent_decisions,alias:ad" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	ProposalID      *pulid.ID      `json:"proposalId"      bun:"proposal_id,type:VARCHAR(100),nullzero"`
	ExceptionID     *pulid.ID      `json:"exceptionId"     bun:"exception_id,type:VARCHAR(100),nullzero"`
	DecidedByUserID pulid.ID       `json:"decidedByUserId" bun:"decided_by_user_id,type:VARCHAR(100),notnull"`
	Decision        DecisionType   `json:"decision"        bun:"decision,type:agent_decision_type_enum,notnull"`
	Modifications   map[string]any `json:"modifications"   bun:"modifications,type:JSONB,nullzero"`
	ReasonCode      string         `json:"reasonCode"      bun:"reason_code,type:VARCHAR(100),notnull"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	DecidedBy    *tenant.User         `bun:"rel:belongs-to,join:decided_by_user_id=id" json:"-"`
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id"    json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"     json:"-"`
}

func (d *AgentDecision) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		d,
		validation.Field(&d.DecidedByUserID,
			validation.Required.Error("Decisions must be attributed to a human user"),
		),
		validation.Field(&d.Decision,
			validation.Required.Error("Decision is required"),
			validation.By(isValidEnum(d.Decision.IsValid, "Invalid decision")),
		),
		validation.Field(&d.ReasonCode, validation.Required.Error("Reason code is required")),
	)

	var validationErrs validation.Errors
	if errors.As(err, &validationErrs) {
		errortypes.FromOzzoErrors(validationErrs, multiErr)
	}

	if d.subjectCount() != 1 {
		multiErr.Add(
			"proposalId",
			errortypes.ErrInvalid,
			"A decision must reference exactly one proposal or exception",
		)
	}

	if d.Decision == DecisionModified && len(d.Modifications) == 0 {
		multiErr.Add(
			"modifications",
			errortypes.ErrRequired,
			"Modifications are required when the decision is Modified",
		)
	}
}

func (d *AgentDecision) subjectCount() int {
	count := 0
	if d.ProposalID != nil && d.ProposalID.IsNotNil() {
		count++
	}
	if d.ExceptionID != nil && d.ExceptionID.IsNotNil() {
		count++
	}
	return count
}

func (d *AgentDecision) GetID() pulid.ID {
	return d.ID
}

func (d *AgentDecision) GetCreatedAt() int64 {
	return d.CreatedAt
}

func (d *AgentDecision) GetOrganizationID() pulid.ID {
	return d.OrganizationID
}

func (d *AgentDecision) GetBusinessUnitID() pulid.ID {
	return d.BusinessUnitID
}

func (d *AgentDecision) GetTableName() string {
	return "agent_decisions"
}

func (d *AgentDecision) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("ad_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}
