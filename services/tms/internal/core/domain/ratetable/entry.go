package ratetable

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

var _ bun.BeforeAppendModelHook = (*RateTableEntry)(nil)

type RateTableEntry struct {
	bun.BaseModel `bun:"table:rate_table_entries,alias:rte" json:"-"`

	ID             pulid.ID            `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID            `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	RateTableID    pulid.ID            `json:"rateTableId"    bun:"rate_table_id,type:VARCHAR(100),notnull"`
	MatchKey       *string             `json:"matchKey"       bun:"match_key,type:VARCHAR(100)"`
	RangeMin       decimal.NullDecimal `json:"rangeMin"       bun:"range_min,type:NUMERIC(19,4)"`
	RangeMax       decimal.NullDecimal `json:"rangeMax"       bun:"range_max,type:NUMERIC(19,4)"`
	Value          decimal.Decimal     `json:"value"          bun:"value,type:NUMERIC(19,4),notnull"`
	SortOrder      int32               `json:"sortOrder"      bun:"sort_order,type:INTEGER,notnull,default:0"`
	CreatedAt      int64               `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64               `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (rte *RateTableEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rte.ID.IsNil() {
			rte.ID = pulid.MustNew("rte_")
		}
		rte.CreatedAt = now
		rte.UpdatedAt = now
	case *bun.UpdateQuery:
		rte.UpdatedAt = now
	}

	return nil
}

func validateExactEntries(entries []*RateTableEntry, multiErr *errortypes.MultiError) {
	seen := make(map[string]struct{}, len(entries))

	for i, entry := range entries {
		if entry == nil {
			continue
		}

		fieldPrefix := entryFieldPrefix(i)

		if entry.MatchKey == nil || *entry.MatchKey == "" {
			multiErr.Add(
				fieldPrefix+".matchKey",
				errortypes.ErrRequired,
				"Match key is required for exact lookup tables",
			)
			continue
		}

		if _, dup := seen[*entry.MatchKey]; dup {
			multiErr.Add(
				fieldPrefix+".matchKey",
				errortypes.ErrInvalid,
				"Match keys must be unique",
			)
		}
		seen[*entry.MatchKey] = struct{}{}
	}
}

func validateRangeEntries(entries []*RateTableEntry, multiErr *errortypes.MultiError) {
	type band struct {
		index int
		min   decimal.Decimal
		max   decimal.NullDecimal
	}

	bands := make([]band, 0, len(entries))

	for i, entry := range entries {
		if entry == nil {
			continue
		}

		fieldPrefix := entryFieldPrefix(i)

		if !entry.RangeMin.Valid {
			multiErr.Add(
				fieldPrefix+".rangeMin",
				errortypes.ErrRequired,
				"Range minimum is required for range lookup tables",
			)
			continue
		}

		if entry.RangeMax.Valid && entry.RangeMax.Decimal.LessThanOrEqual(entry.RangeMin.Decimal) {
			multiErr.Add(
				fieldPrefix+".rangeMax",
				errortypes.ErrInvalid,
				"Range maximum must be greater than range minimum",
			)
			continue
		}

		bands = append(bands, band{index: i, min: entry.RangeMin.Decimal, max: entry.RangeMax})
	}

	sort.Slice(bands, func(a, b int) bool {
		return bands[a].min.LessThan(bands[b].min)
	})

	for i := 1; i < len(bands); i++ {
		prev := bands[i-1]
		if !prev.max.Valid || bands[i].min.LessThan(prev.max.Decimal) {
			multiErr.Add(
				entryFieldPrefix(bands[i].index)+".rangeMin",
				errortypes.ErrInvalid,
				"Range bands must not overlap",
			)
		}
	}
}

func entryFieldPrefix(index int) string {
	return "entries[" + strconv.Itoa(index) + "]"
}
