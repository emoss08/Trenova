package accessorialcharge

import (
	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// var (
// 	_ bun.BeforeAppendModelHook = (*AccessorialCharge)(nil)
// 	_ domain.Validatable        = (*AccessorialCharge)(nil)
// 	_ infra.PostgresSearchable  = (*AccessorialCharge)(nil)
// )

type AccessorialCharge struct {
	bun.BaseModel `bun:"table:accessorial_charges,alias:acc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id" bun:",pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	// Core Fields
	Status      domain.Status   `json:"status" bun:"status,type:status_enum,notnull,default:'Active'"`
	Code        string          `json:"code" bun:"code,type:VARCHAR(10),notnull"`
	Description string          `json:"description" bun:"description,type:TEXT,notnull"`
	Unit        int16           `json:"unit" bun:"unit,type:INTEGER,notnull"`
	Method      Method          `json:"method" bun:"method,type:accessorial_method_enum,notnull"`
	Amount      decimal.Decimal `json:"amount" bun:"amount,type:NUMERIC(19,4),notnull"`

	// Metadata
	Version      int64  `json:"version" bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}
