package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryOptionsFromGraphQL_MapsDataTableFilters(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}

	opts := queryOptionsFromGraphQL(gqlListOptions{
		TenantInfo: tenantInfo,
		Limit:      50,
		Offset:     100,
		Query:      "TRK",
		FieldFilters: []*gqlmodel.FieldFilterInput{
			{
				Field:    "status",
				Operator: string(dbtype.OpIn),
				Value:    []any{"Available", "OutOfService"},
			},
			{
				Field:    "year",
				Operator: string(dbtype.OpEqual),
				Value:    2025,
			},
			nil,
		},
		FilterGroups: []*gqlmodel.FilterGroupInput{
			{
				Filters: []*gqlmodel.FieldFilterInput{
					{
						Field:    "registrationExpiry",
						Operator: string(dbtype.OpDateRange),
						Value: map[string]any{
							"from": "2026-01-01",
							"to":   "2026-12-31",
						},
					},
					{
						Field:    "createdAt",
						Operator: string(dbtype.OpLastNDays),
						Value:    "7",
					},
				},
			},
		},
		Sort: []*gqlmodel.SortFieldInput{
			{
				Field:     "code",
				Direction: string(dbtype.SortDirectionAsc),
			},
			nil,
		},
	})

	require.NotNil(t, opts)
	assert.Equal(t, tenantInfo, opts.TenantInfo)
	assert.Equal(t, 50, opts.Pagination.Limit)
	assert.Equal(t, 100, opts.Pagination.Offset)
	assert.Equal(t, "TRK", opts.Query)

	require.Len(t, opts.FieldFilters, 2)
	assert.Equal(t, "status", opts.FieldFilters[0].Field)
	assert.Equal(t, dbtype.OpIn, opts.FieldFilters[0].Operator)
	assert.Equal(t, []string{"Available", "OutOfService"}, opts.FieldFilters[0].Value)
	assert.Equal(t, "year", opts.FieldFilters[1].Field)
	assert.Equal(t, dbtype.OpEqual, opts.FieldFilters[1].Operator)
	assert.Equal(t, int64(2025), opts.FieldFilters[1].Value)

	require.Len(t, opts.FilterGroups, 1)
	require.Len(t, opts.FilterGroups[0].Filters, 2)
	assert.Equal(t, "registrationExpiry", opts.FilterGroups[0].Filters[0].Field)
	assert.Equal(t, dbtype.OpDateRange, opts.FilterGroups[0].Filters[0].Operator)
	assert.Equal(
		t,
		map[string]any{
			"from": "2026-01-01",
			"to":   "2026-12-31",
		},
		opts.FilterGroups[0].Filters[0].Value,
	)
	assert.Equal(t, "createdAt", opts.FilterGroups[0].Filters[1].Field)
	assert.Equal(t, dbtype.OpLastNDays, opts.FilterGroups[0].Filters[1].Operator)
	assert.Equal(t, 7, opts.FilterGroups[0].Filters[1].Value)

	require.Len(t, opts.Sort, 1)
	assert.Equal(t, "code", opts.Sort[0].Field)
	assert.Equal(t, dbtype.SortDirectionAsc, opts.Sort[0].Direction)
}

func TestQueryOptionsFromGraphQL_HandlesEmptyInputs(t *testing.T) {
	t.Parallel()

	opts := queryOptionsFromGraphQL(gqlListOptions{})

	require.NotNil(t, opts)
	assert.Nil(t, opts.FieldFilters)
	assert.Nil(t, opts.FilterGroups)
	assert.Nil(t, opts.Sort)
}
