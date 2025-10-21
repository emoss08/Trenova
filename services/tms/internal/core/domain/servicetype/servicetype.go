package servicetype

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
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*ServiceType)(nil)
	_ domaintypes.PostgresSearchable = (*ServiceType)(nil)
	_ domain.Validatable             = (*ServiceType)(nil)
	_ framework.TenantedEntity       = (*ServiceType)(nil)
)

type ServiceType struct {
	bun.BaseModel `bun:"table:service_types,alias:st" json:"-"`

	ID             pulid.ID      `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID      `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Status         domain.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	Code           string        `json:"code"           bun:"code,type:VARCHAR(10),notnull"`
	Description    string        `json:"description"    bun:"description,type:TEXT,nullzero"`
	Color          string        `json:"color"          bun:"color,type:VARCHAR(10)"`
	Version        int64         `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64         `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64         `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string        `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string        `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (st *ServiceType) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(st,
		validation.Field(&st.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 100).Error("Code must be between 1 and 100 characters"),
		),
		validation.Field(&st.Color,
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

func (st *ServiceType) GetID() string {
	return st.ID.String()
}

func (st *ServiceType) GetTableName() string {
	return "service_types"
}

func (st *ServiceType) GetOrganizationID() pulid.ID {
	return st.OrganizationID
}

func (st *ServiceType) GetBusinessUnitID() pulid.ID {
	return st.BusinessUnitID
}

func (st *ServiceType) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "st",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Weight: domaintypes.SearchWeightA, Type: domaintypes.FieldTypeText},
			{
				Name:   "description",
				Weight: domaintypes.SearchWeightB,
				Type:   domaintypes.FieldTypeText,
			},
		},
	}
}

func (st *ServiceType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if st.ID.IsNil() {
			st.ID = pulid.MustNew("st_")
		}

		st.CreatedAt = now
	case *bun.UpdateQuery:
		st.UpdatedAt = now
	}

	return nil
}
