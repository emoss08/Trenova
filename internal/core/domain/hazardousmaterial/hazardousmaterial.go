package hazardousmaterial

import (
	"github.com/trenova-app/transport/internal/core/domain"
	"github.com/trenova-app/transport/internal/core/domain/businessunit"
	"github.com/trenova-app/transport/internal/core/domain/organization"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
)

// HazardousMaterial represents the structure of a hazardous material entity in the database.
type HazardousMaterial struct {
	bun.BaseModel `bun:"table:hazardous_materials,alias:hm" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:",pk,type:VARCHAR(50)" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,notnull,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,notnull,type:VARCHAR(100)" json:"organizationId"`

	// Core Fields
	Status               domain.Status  `bun:"status,type:status,default:'Active'" json:"status"`
	Code                 string         `bun:"code,notnull,type:VARCHAR(100)" json:"code"`
	Name                 string         `bun:"name,notnull,type:VARCHAR(100)" json:"name"`
	Description          string         `bun:"description,type:TEXT,notnull" json:"description"`
	Class                HazardousClass `bun:"class,type:VARCHAR(16),notnull" json:"class"`
	UNNumber             string         `bun:"un_number,type:VARCHAR(100)" json:"unNumber"`
	ERGNumber            string         `bun:"erg_number,type:VARCHAR(100)" json:"ergNumber"`
	PackingGroup         PackingGroup   `bun:"packing_group,type:VARCHAR(3)" json:"packingGroup"`
	ProperShippingName   string         `bun:"proper_shipping_name,type:TEXT" json:"properShippingName"`
	HandlingInstructions string         `bun:"handling_instructions,type:TEXT" json:"handlingInstructions"`
	EmergencyContact     string         `bun:"emergency_contact,type:TEXT" json:"emergencyContact"`
	PlacardRequired      bool           `bun:"placard_required,type:BOOLEAN,default:false" json:"placardRequired"`
	IsReportableQuantity bool           `bun:"is_reportable_quantity,type:BOOLEAN,default:false" json:"isReportableQuantity"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `bun:"created_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}
