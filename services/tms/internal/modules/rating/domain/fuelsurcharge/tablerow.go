package fuelsurcharge

import (
	"sort"
	"strconv"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

type FuelSurchargeTableRow struct {
	keel.Table `keel:"fuel_surcharge_table_rows"`

	ID                     pulid.ID            `json:"id"                     keel:"id,pk,autoid:fsr_"`
	BusinessUnitID         pulid.ID            `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID            `json:"organizationId"         keel:"organization_id,pk,tenant"`
	FuelSurchargeProgramID pulid.ID            `json:"fuelSurchargeProgramId" keel:"fuel_surcharge_program_id"`
	PriceMin               decimal.NullDecimal `json:"priceMin"               keel:"price_min"`
	PriceMax               decimal.NullDecimal `json:"priceMax"               keel:"price_max"`
	Value                  decimal.Decimal     `json:"value"                  keel:"value"`
	SortOrder              int32               `json:"sortOrder"              keel:"sort_order"`
	CreatedAt              int64               `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64               `json:"updatedAt"              keel:"updated_at,autoupdate"`
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

func (r *FuelSurchargeTableRow) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&r.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&r.FuelSurchargeProgramID,
			validation.Required.Error("Fuel surcharge program is required"),
		),
		validation.Field(&r.SortOrder,
			validation.Min(int32(0)).Error("Sort order cannot be negative"),
		),
	))

	// The row is selected by where the fuel price falls, so a band whose top is
	// below its bottom can never be reached and the surcharge silently drops to
	// whichever band happens to match instead.
	if r.PriceMin.Valid && r.PriceMax.Valid &&
		r.PriceMax.Decimal.LessThanOrEqual(r.PriceMin.Decimal) {
		multiErr.Add("priceMax", errortypes.ErrInvalid,
			"Maximum price must be greater than the minimum price")
	}

	if r.PriceMin.Valid && r.PriceMin.Decimal.IsNegative() {
		multiErr.Add("priceMin", errortypes.ErrInvalid, "Minimum price cannot be negative")
	}

	if r.Value.IsNegative() {
		multiErr.Add("value", errortypes.ErrInvalid, "Value cannot be negative")
	}
}
