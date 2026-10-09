package tablelayout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validEntity() *TableLayout {
	return &TableLayout{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		Resource:       "shipment",
		Layout: &Layout{
			ColumnVisibility: map[string]bool{"proNumber": true, "status": false},
			ColumnOrder:      []string{"proNumber", "status"},
			ColumnSizing:     map[string]float64{"proNumber": 160},
			ColumnPinning:    &tableconfiguration.ColumnPinning{Left: []string{"select"}},
			Density:          DensityCompact,
		},
	}
}

func fieldsOf(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}
	return fields
}

func TestTableLayout_Validate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*TableLayout)
		field  string
	}{
		{name: "accepts a full layout", mutate: func(*TableLayout) {}},
		{name: "accepts an empty layout", mutate: func(tl *TableLayout) { tl.Layout = &Layout{} }},
		{name: "requires a resource", mutate: func(tl *TableLayout) { tl.Resource = "" }, field: "resource"},
		{
			name:   "accepts a table named in words, as every multi-word table is",
			mutate: func(tl *TableLayout) { tl.Resource = "Service Type" },
		},
		{
			name:   "accepts a name with digits and punctuation",
			mutate: func(tl *TableLayout) { tl.Resource = "AI Audit Event: v2.1_beta-3" },
		},
		{
			name:   "rejects a leading space, which would split one table's layout in two",
			mutate: func(tl *TableLayout) { tl.Resource = " Service Type" },
			field:  "resource",
		},
		{
			name:   "rejects a trailing space",
			mutate: func(tl *TableLayout) { tl.Resource = "Service Type " },
			field:  "resource",
		},
		{
			name:   "rejects characters outside a table name",
			mutate: func(tl *TableLayout) { tl.Resource = "Service/Type" },
			field:  "resource",
		},
		{
			name:   "rejects a resource that is too long",
			mutate: func(tl *TableLayout) { tl.Resource = strings.Repeat("a", MaxResourceLength+1) },
			field:  "resource",
		},
		{name: "requires a layout", mutate: func(tl *TableLayout) { tl.Layout = nil }, field: "layout"},
		{
			name:   "rejects a width below the minimum",
			mutate: func(tl *TableLayout) { tl.Layout.ColumnSizing["proNumber"] = MinColumnWidth - 1 },
			field:  "layout.columnSizing",
		},
		{
			name:   "rejects a width above the maximum",
			mutate: func(tl *TableLayout) { tl.Layout.ColumnSizing["proNumber"] = MaxColumnWidth + 1 },
			field:  "layout.columnSizing",
		},
		{
			name:   "rejects a column ordered twice",
			mutate: func(tl *TableLayout) { tl.Layout.ColumnOrder = []string{"status", "status"} },
			field:  "layout.columnOrder[1]",
		},
		{
			name:   "rejects an empty column ID",
			mutate: func(tl *TableLayout) { tl.Layout.ColumnOrder = []string{" "} },
			field:  "layout.columnOrder[0]",
		},
		{
			name: "rejects a column pinned to both sides",
			mutate: func(tl *TableLayout) {
				tl.Layout.ColumnPinning = &tableconfiguration.ColumnPinning{
					Left:  []string{"status"},
					Right: []string{"status"},
				}
			},
			field: "layout.columnPinning",
		},
		{
			name:   "rejects an unknown density",
			mutate: func(tl *TableLayout) { tl.Layout.Density = "spacious" },
			field:  "layout.density",
		},
		{
			name: "rejects too many format rules",
			mutate: func(tl *TableLayout) {
				tl.Layout.FormatRules = make([]tableconfiguration.FormatRule, MaxFormatRules+1)
			},
			field: "layout.formatRules",
		},
		{
			name:   "accepts pinned rows",
			mutate: func(tl *TableLayout) { tl.Layout.PinnedRowIDs = []string{"shp_1", "shp_2"} },
		},
		{
			name:   "rejects a row pinned twice",
			mutate: func(tl *TableLayout) { tl.Layout.PinnedRowIDs = []string{"shp_1", "shp_1"} },
			field:  "layout.pinnedRowIds[1]",
		},
		{
			name: "rejects too many pinned rows",
			mutate: func(tl *TableLayout) {
				ids := make([]string, 0, MaxPinnedRows+1)
				for i := range MaxPinnedRows + 1 {
					ids = append(ids, fmt.Sprintf("shp_%d", i))
				}
				tl.Layout.PinnedRowIDs = ids
			},
			field: "layout.pinnedRowIds",
		},
		{
			name: "rejects too many columns",
			mutate: func(tl *TableLayout) {
				order := make([]string, 0, MaxColumns+1)
				for i := range MaxColumns + 1 {
					order = append(order, fmt.Sprintf("c%d", i))
				}
				tl.Layout.ColumnOrder = order
			},
			field: "layout.columnOrder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := validEntity()
			tt.mutate(entity)

			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)

			if tt.field == "" {
				assert.False(t, multiErr.HasErrors(), "unexpected errors: %v", fieldsOf(multiErr))
				return
			}
			assert.Contains(t, fieldsOf(multiErr), tt.field)
		})
	}
}
