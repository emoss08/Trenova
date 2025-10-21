package fleetcode

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*FleetCode)(nil)
	_ domain.Validatable             = (*FleetCode)(nil)
	_ framework.TenantedEntity       = (*FleetCode)(nil)
	_ domaintypes.PostgresSearchable = (*FleetCode)(nil)
)

type FleetCode struct {
	bun.BaseModel `bun:"table:fleet_codes,alias:fc" json:"-"`

	ID             pulid.ID            `bun:"id,pk,type:VARCHAR(100)"                                                  json:"id"`
	OrganizationID pulid.ID            `bun:"organization_id,pk,type:VARCHAR(100),notnull"                             json:"organizationId"`
	BusinessUnitID pulid.ID            `bun:"business_unit_id,pk,type:VARCHAR(100),notnull"                            json:"businessUnitId"`
	ManagerID      pulid.ID            `bun:"manager_id,type:VARCHAR(100),notnull"                                     json:"managerId"`
	Status         domain.Status       `bun:"status,type:status_enum,notnull,default:'Active'"                         json:"status"`
	Code           string              `bun:"code,type:VARCHAR(10),notnull"                                            json:"code"`
	Description    string              `bun:"description,type:TEXT"                                                    json:"description"`
	RevenueGoal    decimal.NullDecimal `bun:"revenue_goal,type:NUMERIC(10,2),nullzero"                                 json:"revenueGoal"`
	DeadheadGoal   decimal.NullDecimal `bun:"deadhead_goal,type:NUMERIC(10,2),nullzero"                                json:"deadheadGoal"`
	MileageGoal    decimal.NullDecimal `bun:"mileage_goal,type:NUMERIC(10,2),nullzero"                                 json:"mileageGoal"`
	Color          string              `bun:"color,type:VARCHAR(10)"                                                   json:"color"`
	Version        int64               `bun:"version,type:BIGINT"                                                      json:"version"`
	CreatedAt      int64               `bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt      int64               `bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`
	SearchVector   string              `bun:"search_vector,type:TSVECTOR,scanonly"                                     json:"-"`
	Rank           string              `bun:"rank,type:VARCHAR(100),scanonly"                                          json:"-"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Manager      *tenant.User         `json:"manager,omitempty"      bun:"rel:belongs-to,join:manager_id=id"`
}

func (fc *FleetCode) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(fc,
		validation.Field(&fc.Code,
			validation.Required.Error("Code is required. Please try again"),
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),
		validation.Field(&fc.RevenueGoal,
			validation.Min(0).Error("Revenue goal must be greater than or equal to 0"),
		),
		validation.Field(&fc.DeadheadGoal,
			validation.Min(0).Error("Deadhead goal must be greater than or equal to 0"),
		),
		validation.Field(&fc.ManagerID,
			validation.Required.Error("Manager is required"),
		),
		validation.Field(&fc.Color,
			is.HexColor.Error("Color must be a valid hex color. Please try again."),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (fc *FleetCode) GetID() string {
	return fc.ID.String()
}

func (fc *FleetCode) GetTableName() string {
	return "fleet_codes"
}

func (fc *FleetCode) GetOrganizationID() pulid.ID {
	return fc.OrganizationID
}

func (fc *FleetCode) GetBusinessUnitID() pulid.ID {
	return fc.BusinessUnitID
}

func (fc *FleetCode) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "fc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
		Relationships: []*domaintypes.RelationshipDefinition{
			{
				Field:        "Manager",
				Type:         domaintypes.RelationshipTypeBelongsTo,
				TargetTable:  "users",
				TargetEntity: (*tenant.User)(nil),
				ForeignKey:   "manager_id",
				ReferenceKey: "id",
				Alias:        "mgr",
				Queryable:    true,
			},
		},
	}
}

func (fc *FleetCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

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
