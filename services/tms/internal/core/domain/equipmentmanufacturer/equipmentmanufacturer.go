/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package equipmentmanufacturer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*EquipmentManufacturer)(nil)
	_ domain.Validatable        = (*EquipmentManufacturer)(nil)
)

type EquipmentManufacturer struct {
	bun.BaseModel `bun:"table:equipment_manufacturers,alias:em" json:"-"`

	// Primary identifiers
	ID             pulid.ID      `bun:"id,type:VARCHAR(100),pk,notnull"                                                      json:"id"`
	BusinessUnitID pulid.ID      `bun:"business_unit_id,type:VARCHAR(100),notnull,pk"                                        json:"businessUnitId"`
	OrganizationID pulid.ID      `bun:"organization_id,type:VARCHAR(100),notnull,pk"                                         json:"organizationId"`
	Status         domain.Status `bun:"status,type:status_enum,notnull,default:'Active'"                                     json:"status"`
	Name           string        `bun:"name,type:VARCHAR(100),notnull"                                                       json:"name"`
	Description    string        `bun:"description,type:TEXT,nullzero"                                                       json:"description"`
	SearchVector   string        `bun:"search_vector,type:TSVECTOR,scanonly"                                                 json:"-"`
	Rank           string        `bun:"rank,type:VARCHAR(100),scanonly"                                                      json:"-"`
	Version        int64         `bun:"version,type:BIGINT"                                                                  json:"version"`
	CreatedAt      int64         `bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt      int64         `bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (em *EquipmentManufacturer) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, em,
		// Name is required and must be between 1 and 100 characters
		validation.Field(&em.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
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
func (em *EquipmentManufacturer) GetID() string {
	return em.ID.String()
}

func (em *EquipmentManufacturer) GetTableName() string {
	return "equipment_manufacturers"
}

func (em *EquipmentManufacturer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if em.ID.IsNil() {
			em.ID = pulid.MustNew("em_")
		}

		em.CreatedAt = now
	case *bun.UpdateQuery:
		em.UpdatedAt = now
	}

	return nil
}
