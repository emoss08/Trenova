/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package equipmenttype

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
	_ bun.BeforeAppendModelHook = (*EquipmentType)(nil)
	_ domain.Validatable        = (*EquipmentType)(nil)
	_ infra.PostgresSearchable  = (*EquipmentType)(nil)
)

type EquipmentType struct {
	bun.BaseModel `bun:"table:equipment_types,alias:et" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull"               json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),notnull,pk" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),notnull,pk"  json:"organizationId"`

	// Core Fields
	Status      domain.Status `json:"status"      bun:"status,type:status_enum,notnull,default:'Active'"`
	Code        string        `json:"code"        bun:"code,type:VARCHAR(10),notnull"`
	Description string        `json:"description" bun:"description,type:TEXT,nullzero"`
	Class       Class         `json:"class"       bun:"class,type:equipment_class_enum,notnull"`
	Color       string        `json:"color"       bun:"color,type:VARCHAR(10),nullzero"`

	// Metadata
	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (et *EquipmentType) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, et,
		// Code is required and must be between 1 and 100 characters
		validation.Field(&et.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 100).Error("Code must be between 1 and 100 characters"),
		),

		// Class is required and must be a valid class
		validation.Field(
			&et.Class,
			validation.Required.Error("Class is required"),
			validation.In(ClassTractor, ClassTrailer, ClassContainer, ClassOther).
				Error("Class must be a valid class"),
		),

		// Color must be a valid hex color
		validation.Field(&et.Color,
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
func (et *EquipmentType) GetID() string {
	return et.ID.String()
}

func (et *EquipmentType) GetTableName() string {
	return "equipment_types"
}

func (et *EquipmentType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if et.ID.IsNil() {
			et.ID = pulid.MustNew("et_")
		}

		et.CreatedAt = now
	case *bun.UpdateQuery:
		et.UpdatedAt = now
	}

	return nil
}

func (et *EquipmentType) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "et",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "code",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "description",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "class",
				Weight: "C",
				Type:   infra.PostgresSearchTypeEnum,
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}
