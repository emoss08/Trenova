package listquery

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const truckSelect = `SELECT t."id", t."organization_id", t."business_unit_id", t."owner_id", t."code", ` +
	`t."status", t."year", t."weight", t."description", t."active", t."token", t."created_at" ` +
	`FROM "test_trucks" t `

const truckTenant = `WHERE t."organization_id" = $1 AND t."business_unit_id" = $2`

func filtered(t *testing.T, filter *pagination.QueryOptions) (string, []any) {
	t.Helper()

	q := testTrucks.Query()
	p := newPlan(q)
	q.Where(p.where(filter)...).Limit(10)

	return render(t, q)
}

func filters(fs ...domaintypes.FieldFilter) *pagination.QueryOptions {
	return &pagination.QueryOptions{FieldFilters: fs}
}

func TestFilters_RenderTypedPredicates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		filter domaintypes.FieldFilter
		where  string
		args   []any
	}{
		{
			name: "equality on a text column",
			filter: domaintypes.FieldFilter{
				Field:    "code",
				Operator: dbtype.OpEqual,
				Value:    "TRK-1",
			},
			where: `(t."code" = $3)`,
			args:  []any{"TRK-1"},
		},
		{
			name: "inequality on a text column",
			filter: domaintypes.FieldFilter{
				Field:    "code",
				Operator: dbtype.OpNotEqual,
				Value:    "TRK-1",
			},
			where: `(t."code" <> $3)`,
			args:  []any{"TRK-1"},
		},
		{
			name: "equality on an enum compares its text",
			filter: domaintypes.FieldFilter{
				Field:    "status",
				Operator: dbtype.OpEqual,
				Value:    "Available",
			},
			where: `(CAST(t."status" AS text) = $3)`,
			args:  []any{"Available"},
		},
		{
			name: "a JSON number on an integer column binds an integer",
			filter: domaintypes.FieldFilter{
				Field:    "year",
				Operator: dbtype.OpGreaterThan,
				Value:    2001.0,
			},
			where: `(t."year" > $3)`,
			args:  []any{int64(2001)},
		},
		{
			name: "a numeric string on an integer column binds an integer",
			filter: domaintypes.FieldFilter{
				Field:    "year",
				Operator: dbtype.OpLessThanOrEqual,
				Value:    "2024",
			},
			where: `(t."year" <= $3)`,
			args:  []any{int64(2024)},
		},
		{
			name: "an integer on a float column binds a float",
			filter: domaintypes.FieldFilter{
				Field:    "weight",
				Operator: dbtype.OpGreaterThanOrEqual,
				Value:    int64(5),
			},
			where: `(t."weight" >= $3)`,
			args:  []any{float64(5)},
		},
		{
			name:   "a boolean",
			filter: domaintypes.FieldFilter{Field: "active", Operator: dbtype.OpEqual, Value: true},
			where:  `(t."active" = $3)`,
			args:   []any{true},
		},
		{
			name: "contains escapes the pattern",
			filter: domaintypes.FieldFilter{
				Field:    "code",
				Operator: dbtype.OpContains,
				Value:    "50%_off",
			},
			where: `(t."code" ILIKE $3)`,
			args:  []any{`%50\%\_off%`},
		},
		{
			name: "starts with ignores case",
			filter: domaintypes.FieldFilter{
				Field:    "code",
				Operator: dbtype.OpStartsWith,
				Value:    "trk",
			},
			where: `(t."code" ILIKE $3)`,
			args:  []any{"trk%"},
		},
		{
			name: "ends with ignores case",
			filter: domaintypes.FieldFilter{
				Field:    "code",
				Operator: dbtype.OpEndsWith,
				Value:    "01",
			},
			where: `(t."code" ILIKE $3)`,
			args:  []any{"%01"},
		},
		{
			name:   "like keeps the pattern",
			filter: domaintypes.FieldFilter{Field: "code", Operator: dbtype.OpLike, Value: "T_K%"},
			where:  `(t."code" LIKE $3)`,
			args:   []any{"T_K%"},
		},
		{
			name: "contains on a number compares its text",
			filter: domaintypes.FieldFilter{
				Field:    "year",
				Operator: dbtype.OpContains,
				Value:    "20",
			},
			where: `(CAST(t."year" AS text) ILIKE $3)`,
			args:  []any{"%20%"},
		},
		{
			name: "in on an enum binds one text array",
			filter: domaintypes.FieldFilter{
				Field:    "status",
				Operator: dbtype.OpIn,
				Value:    []any{"Available", "OutOfService"},
			},
			where: `(CAST(t."status" AS text) = ANY($3))`,
			args:  []any{[]string{"Available", "OutOfService"}},
		},
		{
			name: "not in on an integer binds one integer array",
			filter: domaintypes.FieldFilter{
				Field:    "year",
				Operator: dbtype.OpNotIn,
				Value:    []any{2001.0, int64(2002)},
			},
			where: `(t."year" <> ALL($3))`,
			args:  []any{[]int64{2001, 2002}},
		},
		{
			name:   "in with one value",
			filter: domaintypes.FieldFilter{Field: "code", Operator: dbtype.OpIn, Value: "TRK-1"},
			where:  `(t."code" = ANY($3))`,
			args:   []any{[]string{"TRK-1"}},
		},
		{
			name:   "is null",
			filter: domaintypes.FieldFilter{Field: "year", Operator: dbtype.OpIsNull},
			where:  `(t."year" IS NULL)`,
		},
		{
			name:   "is not null",
			filter: domaintypes.FieldFilter{Field: "year", Operator: dbtype.OpIsNotNull},
			where:  `(t."year" IS NOT NULL)`,
		},
		{
			name: "a date range includes its last day",
			filter: domaintypes.FieldFilter{
				Field:    "createdAt",
				Operator: dbtype.OpDateRange,
				Value:    map[string]any{"from": 1700000000.0, "to": 1700086400.0},
			},
			where: `((t."created_at" >= $3) AND (t."created_at" < $4))`,
			args:  []any{int64(1700000000), int64(1700172800)},
		},
		{
			name: "a relation path joins the relation",
			filter: domaintypes.FieldFilter{
				Field:    "owner.name",
				Operator: dbtype.OpContains,
				Value:    "acme",
			},
			where: `(ow."name" ILIKE $3)`,
			args:  []any{"%acme%"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sql, args := filtered(t, filters(tc.filter))

			assert.Contains(t, sql, truckTenant+" AND "+tc.where+" LIMIT 10")
			assert.Equal(t, append([]any{"org_1", "bu_1"}, tc.args...), args)
		})
	}
}

func TestFilters_ARelationPathJoinsItsTableUnderTheTenant(t *testing.T) {
	t.Parallel()

	sql, _ := filtered(t, filters(domaintypes.FieldFilter{
		Field:    "owner.name",
		Operator: dbtype.OpEqual,
		Value:    "Acme",
	}))

	assert.Equal(t, truckSelect+
		`LEFT JOIN "test_owners" ow ON ow."id" = t."owner_id" AND ow."organization_id" = $1 AND ow."business_unit_id" = $2 `+
		truckTenant+` AND (ow."name" = $3) LIMIT 10`, sql)
}

func TestFilters_SkipWhatTheyCannotApply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		filter domaintypes.FieldFilter
	}{
		{
			"an unknown field",
			domaintypes.FieldFilter{Field: "nope", Operator: dbtype.OpEqual, Value: "x"},
		},
		{
			"a field hidden from JSON",
			domaintypes.FieldFilter{Field: "token", Operator: dbtype.OpEqual, Value: "x"},
		},
		{
			"the search vector",
			domaintypes.FieldFilter{Field: "searchVector", Operator: dbtype.OpEqual, Value: "x"},
		},
		{
			"the snake case column name",
			domaintypes.FieldFilter{
				Field:    "created_at",
				Operator: dbtype.OpGreaterThan,
				Value:    1.0,
			},
		},
		{
			"an empty equality",
			domaintypes.FieldFilter{Field: "code", Operator: dbtype.OpEqual, Value: ""},
		},
		{
			"an empty inequality",
			domaintypes.FieldFilter{Field: "status", Operator: dbtype.OpNotEqual, Value: ""},
		},
		{
			"a fraction on an integer column",
			domaintypes.FieldFilter{Field: "year", Operator: dbtype.OpGreaterThan, Value: 2001.5},
		},
		{
			"a word on an integer column",
			domaintypes.FieldFilter{Field: "year", Operator: dbtype.OpEqual, Value: "soon"},
		},
		{
			"an ordering comparison on an enum",
			domaintypes.FieldFilter{Field: "status", Operator: dbtype.OpGreaterThan, Value: "A"},
		},
		{
			"a field of the related row hidden from JSON",
			domaintypes.FieldFilter{Field: "owner.secret", Operator: dbtype.OpEqual, Value: "x"},
		},
		{
			"a to-many relation path",
			domaintypes.FieldFilter{Field: "loads.name", Operator: dbtype.OpEqual, Value: "x"},
		},
		{
			"a path deeper than one relation",
			domaintypes.FieldFilter{
				Field:    "owner.owner.name",
				Operator: dbtype.OpEqual,
				Value:    "x",
			},
		},
		{
			"an unsupported operator",
			domaintypes.FieldFilter{Field: "createdAt", Operator: dbtype.OpThisWeek},
		},
		{
			"a count operator",
			domaintypes.FieldFilter{Field: "year", Operator: dbtype.OpCountGt, Value: 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sql, args := filtered(t, filters(tc.filter))

			assert.Equal(t, truckSelect+truckTenant+" LIMIT 10", sql)
			assert.Equal(t, []any{"org_1", "bu_1"}, args)
		})
	}
}

func TestFilters_ARelationThatIsNotQueryableIsSkipped(t *testing.T) {
	t.Parallel()

	q := testPlainTrucks.Query()
	p := newPlan(q)
	q.Where(p.where(filters(domaintypes.FieldFilter{
		Field:    "owner.name",
		Operator: dbtype.OpEqual,
		Value:    "Acme",
	}))...).Limit(10)

	sql, _ := render(t, q)

	assert.NotContains(t, sql, "JOIN")
}

func TestFilterGroups_OrWithinAGroupAndAcrossGroups(t *testing.T) {
	t.Parallel()

	sql, args := filtered(t, &pagination.QueryOptions{FilterGroups: []domaintypes.FilterGroup{
		{Filters: []domaintypes.FieldFilter{
			{Field: "code", Operator: dbtype.OpEqual, Value: "A"},
			{Field: "nope", Operator: dbtype.OpEqual, Value: "x"},
			{Field: "code", Operator: dbtype.OpEqual, Value: "B"},
		}},
		{Filters: []domaintypes.FieldFilter{
			{Field: "active", Operator: dbtype.OpEqual, Value: true},
		}},
		{Filters: []domaintypes.FieldFilter{
			{Field: "nope", Operator: dbtype.OpEqual, Value: "x"},
		}},
		{},
	}})

	assert.Contains(
		t,
		sql,
		truckTenant+` AND ((t."code" = $3) OR (t."code" = $4)) AND ((t."active" = $5)) LIMIT 10`,
	)
	assert.Equal(t, []any{"org_1", "bu_1", "A", "B", true}, args)
}

func TestFilters_AreCappedAtTheLimit(t *testing.T) {
	t.Parallel()

	fs := make([]domaintypes.FieldFilter, MaxFilters+5)
	for i := range fs {
		fs[i] = domaintypes.FieldFilter{
			Field:    "year",
			Operator: dbtype.OpGreaterThan,
			Value:    int64(i),
		}
	}

	_, args := filtered(t, filters(fs...))

	assert.Len(t, args, MaxFilters+2)
}

func TestAggregateFilters_CountTheRelatedRows(t *testing.T) {
	t.Parallel()

	cases := []struct {
		op   dbtype.Operator
		want string
	}{
		{dbtype.OpCountGt, ">"},
		{dbtype.OpCountGte, ">="},
		{dbtype.OpCountLt, "<"},
		{dbtype.OpCountLte, "<="},
		{dbtype.OpCountEq, "="},
	}

	for _, tc := range cases {
		t.Run(string(tc.op), func(t *testing.T) {
			t.Parallel()

			sql, args := filtered(
				t,
				&pagination.QueryOptions{AggregateFilters: []domaintypes.AggregateFilter{
					{Relation: "loads", Operator: tc.op, Value: 2},
					{Relation: "owner", Operator: tc.op, Value: 2},
					{Relation: "nope", Operator: tc.op, Value: 2},
				}},
			)

			assert.Contains(
				t,
				sql,
				truckTenant+` AND ((SELECT count(*) AS c1 FROM "test_owners" o `+
					`WHERE o."organization_id" = $1 AND o."business_unit_id" = $2 AND (o."truck_id" = t."id")) `+
					tc.want+` $3) LIMIT 10`,
			)
			assert.Equal(t, []any{"org_1", "bu_1", int64(2)}, args)
		})
	}
}

func TestSearch_MatchesTheVectorUnderBothDictionaries(t *testing.T) {
	t.Parallel()

	sql, args := filtered(t, &pagination.QueryOptions{Query: "dry van"})

	assert.Contains(t, sql, truckTenant+` AND t."search_vector" @@ (`+
		`websearch_to_tsquery('simple'::regconfig, $3) || to_tsquery('simple'::regconfig, $4) || `+
		`websearch_to_tsquery('english'::regconfig, $3) || to_tsquery('english'::regconfig, $4)) LIMIT 10`)
	assert.Equal(t, []any{"org_1", "bu_1", "dry van", "'dry':* & 'van':*"}, args)
}

func TestSearch_WebsearchSyntaxDropsThePrefixQuery(t *testing.T) {
	t.Parallel()

	for _, query := range []string{`"dry van"`, "dry or van", "dry -van", "d"} {
		sql, args := filtered(t, &pagination.QueryOptions{Query: query})

		assert.Contains(
			t,
			sql,
			`t."search_vector" @@ (websearch_to_tsquery('simple'::regconfig, $3) || `+
				`websearch_to_tsquery('english'::regconfig, $3))`,
			query,
		)
		assert.Equal(t, []any{"org_1", "bu_1", query}, args, query)
	}
}

func TestSearch_ConfiguredFieldsWithoutAVector(t *testing.T) {
	t.Parallel()

	q := testPlainTrucks.Query()
	p := newPlan(q)
	q.Where(p.where(&pagination.QueryOptions{Query: "12"})...).Limit(10)

	sql, args := render(t, q)

	assert.Contains(t, sql, `AND ((pt."code" ILIKE $3) OR (CAST(pt."status" AS text) ILIKE $4) OR `+
		`(CAST(pt."year" AS text) = $5)) LIMIT 10`)
	assert.Equal(t, []any{"org_1", "bu_1", "%12%", "%12%", "12"}, args)

	q = testPlainTrucks.Query()
	p = newPlan(q)
	q.Where(p.where(&pagination.QueryOptions{Query: "acme"})...).Limit(10)

	sql, args = render(t, q)

	assert.Contains(
		t,
		sql,
		`AND ((pt."code" ILIKE $3) OR (CAST(pt."status" AS text) ILIKE $4)) LIMIT 10`,
	)
	assert.Equal(t, []any{"org_1", "bu_1", "%acme%", "%acme%"}, args)
}

func TestSearch_AModelWithoutConfigurationSearchesItsTextColumns(t *testing.T) {
	t.Parallel()

	type bare struct {
		ID             string `json:"id"`
		OrganizationID string `json:"organizationId"`
		BusinessUnitID string `json:"businessUnitId"`
		OwnerID        string `json:"ownerId"`
		Code           string `json:"code"`
		Token          string `json:"-"`
	}

	h := keel.Handle[bare]{
		Ref: &keel.Ref{Table: newTrucksTable("test_bare_trucks", "bt", testOwners)},
	}
	q := h.Query()
	p := newPlan(q)
	q.Where(p.where(&pagination.QueryOptions{Query: "x"})...).Limit(10)

	sql, _ := render(t, q)

	assert.Contains(t, sql, `AND ((bt."id" ILIKE $3) OR (bt."organization_id" ILIKE $4) OR `+
		`(bt."business_unit_id" ILIKE $5) OR (bt."owner_id" ILIKE $6) OR (bt."code" ILIKE $7)) LIMIT 10`)
}

func TestOrders_RequestSortsInOrder(t *testing.T) {
	t.Parallel()

	q := testTrucks.Query()
	p := newPlan(q)
	q.Order(p.orders([]domaintypes.SortField{
		{Field: "code", Direction: dbtype.SortDirectionAsc},
		{Field: "nope", Direction: dbtype.SortDirectionAsc},
		{Field: "year", Direction: "sideways"},
		{Field: "owner.name", Direction: dbtype.SortDirectionDesc},
	})...).Limit(10)

	sql, _ := render(t, q)

	assert.Contains(t, sql, `ORDER BY t."code" ASC, ow."name" DESC LIMIT 10`)
}

func TestKeysetOrder_DefaultsToNewestFirst(t *testing.T) {
	t.Parallel()

	p := newPlan(testTrucks.Query())
	orders, shape, err := p.keysetOrder(nil)
	require.NoError(t, err)

	q := testTrucks.Query().Order(orders...).Limit(10)
	sql, _ := render(t, q)

	assert.Contains(t, sql, `ORDER BY t."created_at" DESC, t."id" DESC LIMIT 10`)
	assert.Equal(t, []pagination.CursorSortField{
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}, shape)
}

func TestKeysetOrder_NullableColumnsSortTheirNullsLast(t *testing.T) {
	t.Parallel()

	p := newPlan(testTrucks.Query())
	orders, shape, err := p.keysetOrder([]domaintypes.SortField{
		{Field: "year", Direction: dbtype.SortDirectionAsc},
		{Field: "description", Direction: dbtype.SortDirectionDesc},
	})
	require.NoError(t, err)

	q := testTrucks.Query().Order(orders...).Limit(10)
	sql, _ := render(t, q)

	assert.Contains(
		t,
		sql,
		`ORDER BY t."year" ASC NULLS LAST, t."description" DESC NULLS LAST, t."id" DESC LIMIT 10`,
	)
	assert.Equal(t, "id", shape[len(shape)-1].Field)
}

func TestKeysetOrder_RefusesWhatAKeysetCannotOrderBy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		sort  domaintypes.SortField
		field string
	}{
		{
			"a relation path",
			domaintypes.SortField{Field: "owner.name", Direction: dbtype.SortDirectionAsc},
			"sort.field",
		},
		{
			"an unknown field",
			domaintypes.SortField{Field: "nope", Direction: dbtype.SortDirectionAsc},
			"sort.field",
		},
		{
			"a direction",
			domaintypes.SortField{Field: "code", Direction: "sideways"},
			"sort.direction",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := newPlan(testTrucks.Query()).keysetOrder([]domaintypes.SortField{tc.sort})

			var validation *errortypes.Error
			require.ErrorAs(t, err, &validation)
			assert.Equal(t, tc.field, validation.Field)
		})
	}
}

func TestNormalizeKeysetSort_AlwaysEndsWithTheID(t *testing.T) {
	t.Parallel()

	sorts := make([]domaintypes.SortField, MaxSortFields+2)
	for i := range sorts {
		sorts[i] = domaintypes.SortField{
			Field:     "field" + string(rune('A'+i)),
			Direction: dbtype.SortDirectionAsc,
		}
	}
	sorts = append(
		sorts,
		domaintypes.SortField{Field: "fieldA", Direction: dbtype.SortDirectionDesc},
	)

	normalized := normalizeKeysetSort(sorts)

	require.Len(t, normalized, MaxSortFields)
	assert.Equal(
		t,
		domaintypes.SortField{Field: "id", Direction: dbtype.SortDirectionAsc},
		normalized[MaxSortFields-1],
	)

	explicit := normalizeKeysetSort([]domaintypes.SortField{
		{Field: "id", Direction: dbtype.SortDirectionAsc},
		{Field: "code", Direction: dbtype.SortDirectionDesc},
	})
	assert.Equal(t, []domaintypes.SortField{
		{Field: "id", Direction: dbtype.SortDirectionAsc},
		{Field: "code", Direction: dbtype.SortDirectionDesc},
	}, explicit)
}

func TestPick_NarrowsToTheNamedColumns(t *testing.T) {
	t.Parallel()

	q, err := Pick(testTrucks.Query(), []string{"code", "status"})
	require.NoError(t, err)

	sql, _ := render(t, q.Limit(10))
	assert.Equal(
		t,
		`SELECT t."code", t."status", t."id", t."organization_id", t."business_unit_id" `+
			`FROM "test_trucks" t `+truckTenant+` LIMIT 10`,
		sql,
	)

	_, err = Pick(testTrucks.Query(), []string{"code", "nope"})
	require.Error(t, err)
}

func TestKeysetAfter_RefusesACursorOfAnotherSort(t *testing.T) {
	t.Parallel()

	shape := []pagination.CursorSortField{
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}
	encoded, err := pagination.EncodeKeysetCursor(shape, "abc", 0)
	require.NoError(t, err)
	info, err := pagination.NewCursorInfo(20, encoded)
	require.NoError(t, err)

	after, err := keysetAfter(&info, shape)
	require.NoError(t, err)
	assert.Equal(t, "abc", after.Keyset)

	_, err = keysetAfter(&info, []pagination.CursorSortField{{Field: "code", Direction: "asc"}})
	require.Error(t, err)

	none, err := keysetAfter(&pagination.CursorInfo{Limit: 20}, shape)
	require.NoError(t, err)
	assert.False(t, none.IsKeyset())
}

func TestConnectionResult_GivesEveryEdgeACursor(t *testing.T) {
	t.Parallel()

	shape := []pagination.CursorSortField{{Field: "id", Direction: "desc"}}
	rows := []*testTruck{{ID: "a"}, {ID: "b"}, {ID: "c"}}

	result, err := connectionResult(
		&keel.Page[testTruck]{Items: rows, Next: "next"},
		&pagination.Cursor{},
		shape,
		nil,
	)
	require.NoError(t, err)

	assert.True(t, result.HasNextPage)
	require.Len(t, result.EdgeCursors, 3)
	cursors := decodeAll(t, result.EdgeCursors)
	assert.Equal(t, pagination.Cursor{Sort: shape, Skip: 1}, cursors[0])
	assert.Equal(t, pagination.Cursor{Sort: shape, Skip: 2}, cursors[1])
	assert.Equal(t, pagination.Cursor{Sort: shape, Keyset: "next"}, cursors[2])

	next, err := result.NextCursor()
	require.NoError(t, err)
	assert.Equal(t, result.EdgeCursors[2], next)
}

func TestConnectionResult_AfterASkipCursorDropsTheSkippedRows(t *testing.T) {
	t.Parallel()

	shape := []pagination.CursorSortField{{Field: "id", Direction: "desc"}}
	rows := []*testTruck{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	after := pagination.Cursor{Sort: shape, Keyset: "prev", Skip: 2}

	result, err := connectionResult(&keel.Page[testTruck]{Items: rows}, &after, shape, new(int))
	require.NoError(t, err)

	assert.False(t, result.HasNextPage)
	assert.Equal(t, []*testTruck{{ID: "c"}, {ID: "d"}}, result.Items)
	cursors := decodeAll(t, result.EdgeCursors)
	assert.Equal(t, pagination.Cursor{Sort: shape, Keyset: "prev", Skip: 3}, cursors[0])
	assert.Equal(t, pagination.Cursor{Sort: shape, Keyset: "prev", Skip: 4}, cursors[1])
	require.NotNil(t, result.TotalCount)
}

func TestConnectionResult_ASkipPastThePageIsEmpty(t *testing.T) {
	t.Parallel()

	result, err := connectionResult(
		&keel.Page[testTruck]{Items: []*testTruck{{ID: "a"}}},
		&pagination.Cursor{Skip: 3},
		nil,
		nil,
	)
	require.NoError(t, err)

	assert.Empty(t, result.Items)
	assert.Empty(t, result.EdgeCursors)
}

func decodeAll(t *testing.T, encoded []string) []pagination.Cursor {
	t.Helper()

	out := make([]pagination.Cursor, 0, len(encoded))
	for _, e := range encoded {
		c, err := pagination.DecodeCursor(e)
		require.NoError(t, err)
		out = append(out, c)
	}

	return out
}

func TestCatalog_ResolvesJSONNamesThroughTheSchema(t *testing.T) {
	t.Parallel()

	cat := catalogFor[testTruck](testTrucks.Ref.Table)

	assert.Contains(t, cat.fields, "createdAt")
	assert.Contains(t, cat.fields, "ownerId")
	assert.NotContains(t, cat.fields, "token")
	assert.NotContains(t, cat.fields, "created_at")
	assert.NotContains(t, cat.fields, "searchVector")
	assert.Equal(t, int32(10), cat.vector)

	owner, ok := cat.relation("owner")
	require.True(t, ok)
	assert.True(t, owner.toOne())
	assert.Contains(t, owner.fields, "name")
	assert.NotContains(t, owner.fields, "secret")

	loads, ok := cat.relation("loads")
	require.True(t, ok)
	assert.True(t, loads.countable())
}
