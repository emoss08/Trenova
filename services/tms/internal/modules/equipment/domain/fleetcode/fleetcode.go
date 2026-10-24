package fleetcode

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*FleetCode)(nil)
	_ domaintypes.PostgresSearchable     = (*FleetCode)(nil)
)

// Fleet groupings
type FleetCode struct {
	keel.Table `                                          keel:"fleet_codes,report=Equipment"`

	ID             pulid.ID           `json:"id" keel:"id,pk,autoid:fc_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	ManagerID      pulid.ID           `json:"managerId" keel:"manager_id"`
	Status         domaintypes.Status `json:"status" keel:"status"`
	Code           string             `json:"code" keel:"code"`
	Description    string             `json:"description" keel:"description,nullzero"`
	RevenueGoal    *float64           `json:"revenueGoal" keel:"revenue_goal,format=money"`
	DeadheadGoal   *float64           `json:"deadheadGoal" keel:"deadhead_goal,format=percent"`
	MileageGoal    *float64           `json:"mileageGoal" keel:"mileage_goal,format=distance"`
	Color          string             `json:"color" keel:"color,nullzero"`
	SearchVector   string             `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string             `json:"-" keel:"rank,computed"`
	Version        int64              `json:"version" keel:"version,version"`
	CreatedAt      int64              `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" keel:"belongs:business_unit_id,noreport"`
	Organization *tenant.Organization `json:"organization,omitempty" keel:"belongs:organization_id,noreport"`
	Manager      *tenant.User         `json:"manager,omitempty" keel:"belongs:manager_id,noreport"`
}

func (fc *FleetCode) BeforeCreate(_ context.Context) error {
	if fc.Status == "" {
		fc.Status = domaintypes.StatusActive
	}

	return nil
}

func (fc *FleetCode) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		fc,
		validation.Field(&fc.Code, validation.Required),
		validation.Field(
			&fc.Code,
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),
		validation.Field(&fc.ManagerID, validation.Required.Error("Manager is required")),
	))
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

func (fc *FleetCode) GetCreatedAt() int64 {
	return fc.CreatedAt
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
