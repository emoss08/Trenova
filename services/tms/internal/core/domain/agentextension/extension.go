package agentextension

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Extension)(nil)

type Extension struct {
	bun.BaseModel `bun:"table:agent_extensions,alias:agext" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Type           Type           `json:"type"           bun:"type,type:VARCHAR(50),notnull"`
	Enabled        bool           `json:"enabled"        bun:"enabled,type:BOOLEAN,notnull"`
	Availability   Availability   `json:"availability"   bun:"availability,type:VARCHAR(20),notnull"`
	Configuration  map[string]any `json:"configuration"  bun:"configuration,type:JSONB,notnull"`
	EnabledByID    pulid.ID       `json:"enabledById"    bun:"enabled_by_id,type:VARCHAR(100),nullzero"`
	EnabledAt      *int64         `json:"enabledAt"      bun:"enabled_at,type:BIGINT,nullzero"`
	Version        int64          `json:"version"        bun:"version,type:BIGINT,notnull"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	EnabledBy *tenant.User `json:"enabledBy,omitempty" bun:"rel:belongs-to,join:enabled_by_id=id"`
}

func (e *Extension) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("aext_")
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}

func (e *Extension) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&e.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&e.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[Type]("Type is not an extension this system supports"),
		),
		validation.Field(&e.Availability,
			validation.Required.Error("Availability is required"),
			domainvalidation.ValidEnum[Availability]("Availability must be AllAgents or SelectedAgents"),
		),
		validation.Field(&e.EnabledByID,
			validation.When(
				e.Enabled,
				validation.Required.Error("An enabled extension must record who enabled it"),
			),
		),
	))
}

func (e *Extension) Ready(spec Spec) bool {
	return e != nil && e.Enabled && spec.Configured(e.Configuration)
}

func (e *Extension) GrantsEveryAgent() bool {
	return e != nil && e.Enabled && e.Availability == AvailabilityAllAgents
}
