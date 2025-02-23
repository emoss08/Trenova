package servicetype

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*ServiceType)(nil)
	_ domain.Validatable        = (*ServiceType)(nil)
	_ infra.PostgresSearchable  = (*ServiceType)(nil)
)

type ServiceType struct {
	bun.BaseModel `bun:"table:service_types,alias:st" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id" bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`

	// Core Fields
	Status      domain.Status `json:"status" bun:"status,type:status_enum,notnull,default:'Active'"`
	Code        string        `json:"code" bun:"code,type:VARCHAR(100),notnull"`
	Description string        `json:"description" bun:"description,type:VARCHAR(255)"`
	Color       string        `json:"color" bun:"color,type:VARCHAR(10)"`

	// Metadata
	Version      int64  `json:"version" bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`
	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (st *ServiceType) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, st,
		// Code is required and must be between 1 and 100 characters
		validation.Field(&st.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 100).Error("Code must be between 1 and 100 characters"),
		),
		// Color must be a valid hex color
		validation.Field(&st.Color,
			is.HexColor.Error("Color must be a valid hex color. Please try again."),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Pagination Configuration
func (st *ServiceType) GetID() string {
	return st.ID.String()
}

func (st *ServiceType) GetTableName() string {
	return "service_types"
}

func (st *ServiceType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

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

func (st *ServiceType) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "st",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "code",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "description",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
		// CustomRank:      "ts_rank(search_vector, query)",
	}
}
