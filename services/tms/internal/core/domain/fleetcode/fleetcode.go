package fleetcode

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*FleetCode)(nil)
	_ validationframework.TenantedEntity = (*FleetCode)(nil)
	_ domaintypes.PostgresSearchable     = (*FleetCode)(nil)
)

type FleetCode struct {
	bun.BaseModel `bun:"table:fleet_codes,alias:fc" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	ManagerID      pulid.ID           `json:"managerId"      bun:"manager_id,type:VARCHAR(100),notnull"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	Code           string             `json:"code"           bun:"code,type:VARCHAR(10),notnull"`
	Description    string             `json:"description"    bun:"description,type:TEXT,nullzero"`
	RevenueGoal    *float64           `json:"revenueGoal"    bun:"revenue_goal,type:NUMERIC(10,2),nullzero"`
	DeadheadGoal   *float64           `json:"deadheadGoal"   bun:"deadhead_goal,type:NUMERIC(10,2),nullzero"`
	MileageGoal    *float64           `json:"mileageGoal"    bun:"mileage_goal,type:NUMERIC(10,2),nullzero"`
	Color          string             `json:"color"          bun:"color,type:VARCHAR(10),nullzero"`
	SearchVector   string             `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string             `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Manager      *tenant.User         `json:"manager,omitempty"      bun:"rel:belongs-to,join:manager_id=id"`
}

func (fc *FleetCode) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		fc,
		validation.Field(&fc.Code, validation.Required),
		validation.Field(
			&fc.Code,
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),
		validation.Field(&fc.ManagerID, validation.Required.Error("Manager is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (fc *FleetCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if fc.ID.IsNil() {
			fc.ID = pulid.MustNew("fc_")
		}
		fc.CreatedAt = now
	case *bun.UpdateQuery:
		fc.UpdatedAt = now
	}

	return nil
}

func (fc *FleetCode) GetID() pulid.ID {
	return fc.ID
}

func (fc *FleetCode) GetOrganizationID() pulid.ID {
	return fc.OrganizationID
}

func (fc *FleetCode) GetBusinessUnitID() pulid.ID {
	return fc.BusinessUnitID
}

func (fc *FleetCode) GetTableName() string {
	return "fleet_codes"
}

func (fc *FleetCode) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "fc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}
