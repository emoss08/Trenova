package tenant

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AgentControl)(nil)
	_ validationframework.TenantedEntity = (*AgentControl)(nil)
)

type AgentControl struct {
	bun.BaseModel `json:"-" bun:"table:agent_controls,alias:agc"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`

	ShadowMode             bool `json:"shadowMode"             bun:"shadow_mode,type:BOOLEAN,notnull,default:true"`
	BillingAgentEnabled    bool `json:"billingAgentEnabled"    bun:"billing_agent_enabled,type:BOOLEAN,notnull,default:false"`
	DecisionTimeoutSeconds int  `json:"decisionTimeoutSeconds" bun:"decision_timeout_seconds,type:INTEGER,notnull,default:86400"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (ac *AgentControl) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		ac,
		validation.Field(
			&ac.DecisionTimeoutSeconds,
			validation.Min(60).Error("Decision timeout must be at least 60 seconds"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (ac *AgentControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ac.ID.IsNil() {
			ac.ID = pulid.MustNew("agc_")
		}
		ac.CreatedAt = now
	case *bun.UpdateQuery:
		ac.UpdatedAt = now
	}

	return nil
}

func (ac *AgentControl) GetID() pulid.ID {
	return ac.ID
}

func (ac *AgentControl) GetTableName() string {
	return "agent_controls"
}

func (ac *AgentControl) GetOrganizationID() pulid.ID {
	return ac.OrganizationID
}

func (ac *AgentControl) GetBusinessUnitID() pulid.ID {
	return ac.BusinessUnitID
}
