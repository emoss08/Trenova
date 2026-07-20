package fuelsurcharge

import (
	"context"
	"sort"
	"strconv"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*FuelSurchargeTableRow)(nil)

type FuelSurchargeTableRow struct {
	bun.BaseModel `bun:"table:fuel_surcharge_table_rows,alias:fsr" json:"-"`

	ID                     pulid.ID            `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID            `json:"businessUnitId"         bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID            `json:"organizationId"         bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	FuelSurchargeProgramID pulid.ID            `json:"fuelSurchargeProgramId" bun:"fuel_surcharge_program_id,type:VARCHAR(100),notnull"`
	PriceMin               decimal.NullDecimal `json:"priceMin"               bun:"price_min,type:NUMERIC(19,4)"`
	PriceMax               decimal.NullDecimal `json:"priceMax"               bun:"price_max,type:NUMERIC(19,4)"`
	Value                  decimal.Decimal     `json:"value"                  bun:"value,type:NUMERIC(19,4),notnull"`
	SortOrder              int32               `json:"sortOrder"              bun:"sort_order,type:INTEGER,notnull,default:0"`
	CreatedAt              int64               `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64               `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (fr *FuelSurchargeTableRow) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if fr.ID.IsNil() {
			fr.ID = pulid.MustNew("fsr_")
		}
		fr.CreatedAt = now
		fr.UpdatedAt = now
	case *bun.UpdateQuery:
		fr.UpdatedAt = now
	}

	return nil
}

func (fr *FuelSurchargeTableRow) Matches(price decimal.Decimal) bool {
	if fr.PriceMin.Valid && price.LessThan(fr.PriceMin.Decimal) {
		return false
	}
	if fr.PriceMax.Valid && price.GreaterThanOrEqual(fr.PriceMax.Decimal) {
		return false
	}
	return true
}

func validateTableRows(rows []*FuelSurchargeTableRow, multiErr *errortypes.MultiError) {
	type band struct {
		index int
		min   decimal.NullDecimal
		max   decimal.NullDecimal
	}

	bands := make([]band, 0, len(rows))

	for i, row := range rows {
		if row == nil {
			continue
		}

		fieldPrefix := tableRowFieldPrefix(i)

		if row.PriceMin.Valid && row.PriceMax.Valid &&
			row.PriceMax.Decimal.LessThanOrEqual(row.PriceMin.Decimal) {
			multiErr.Add(
				fieldPrefix+".priceMax",
				errortypes.ErrInvalid,
				"Price maximum must be greater than price minimum",
			)
			continue
		}

		bands = append(bands, band{index: i, min: row.PriceMin, max: row.PriceMax})
	}

	sort.Slice(bands, func(a, b int) bool {
		if !bands[a].min.Valid {
			return true
		}
		if !bands[b].min.Valid {
			return false
		}
		return bands[a].min.Decimal.LessThan(bands[b].min.Decimal)
	})

	for i := 1; i < len(bands); i++ {
		prev := bands[i-1]
		curr := bands[i]

		if !curr.min.Valid {
			multiErr.Add(
				tableRowFieldPrefix(curr.index)+".priceMin",
				errortypes.ErrInvalid,
				"Only one row may have an open-ended minimum",
			)
			continue
		}

		if !prev.max.Valid || curr.min.Decimal.LessThan(prev.max.Decimal) {
			multiErr.Add(
				tableRowFieldPrefix(curr.index)+".priceMin",
				errortypes.ErrInvalid,
				"Price bands must not overlap",
			)
		}
	}
}

func tableRowFieldPrefix(index int) string {
	return "tableRows[" + strconv.Itoa(index) + "]"
}
