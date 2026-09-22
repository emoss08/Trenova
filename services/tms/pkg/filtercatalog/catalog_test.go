package filtercatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Every refusal here exists because the alternative is a wrong answer that looks
like a right one.

The query builder drops a field it cannot map and returns the page unfiltered,
which reads to whoever asked as a filtered answer. An enum value outside the
set matches nothing, which reads as "there are none" — that one reached
production, where a driver search that matched nobody became "there are no
driver records currently in the Trenova system".

So none of these are style: each is a class of confident lie, and the test is
that the caller is stopped and told what would have worked.
*/
func shipments() Resource {
	return Resource{
		Tool:     "list_shipments",
		Resource: permission.ResourceShipment,
		Entity:   "shipments",
		Fields: []Field{
			{Name: "status", Kind: KindEnum, Values: []string{"InTransit", "Completed"}},
			{Name: "proNumber", Kind: KindText, Sortable: true},
			{Name: "actualShipDate", Kind: KindDate, Sortable: true},
			{Name: "totalChargeAmount", Kind: KindNumber},
			{Name: "readyToBill", Kind: KindBoolean},
		},
	}.Prepare()
}

func utc() *Criteria {
	return NewCriteria("shipments").At(Clock{Now: 1789776000, Timezone: "UTC"})
}

func TestCompile_NarrowsOnACataloguedField(t *testing.T) {
	t.Parallel()

	criteria := utc()
	filters, err := shipments().Compile([]Condition{
		{Field: "status", Operator: dbtype.OpEqual, Value: "InTransit"},
	}, criteria)

	require.NoError(t, err)
	require.Len(t, filters, 1)
	assert.Equal(t, "status", filters[0].Field)
	assert.Equal(t, "InTransit", filters[0].Value)
	assert.Equal(t, []string{"status equals InTransit"}, criteria.Terms())
}

// The enum is matched case-insensitively but stored as the catalogue spells
// it, because the column is case-sensitive and "intransit" matches nothing.
func TestCompile_NormalisesAnEnumToTheCataloguedSpelling(t *testing.T) {
	t.Parallel()

	filters, err := shipments().Compile([]Condition{
		{Field: "status", Operator: dbtype.OpEqual, Value: "intransit"},
	}, utc())

	require.NoError(t, err)
	assert.Equal(t, "InTransit", filters[0].Value)
}

func TestCompile_RefusesAFieldTheResourceDoesNotHave(t *testing.T) {
	t.Parallel()

	_, err := shipments().Compile([]Condition{
		{Field: "driverMood", Operator: dbtype.OpEqual, Value: "cheerful"},
	}, utc())

	require.Error(t, err)
	// The refusal names what would have worked, so one round trip fixes it.
	assert.Contains(t, err.Error(), "proNumber")
}

func TestCompile_RefusesAnOperatorTheKindDoesNotAccept(t *testing.T) {
	t.Parallel()

	_, err := shipments().Compile([]Condition{
		{Field: "status", Operator: dbtype.OpContains, Value: "trans"},
	}, utc())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "enum")
	assert.Contains(t, err.Error(), string(dbtype.OpEqual))
}

func TestCompile_RefusesAValueOutsideTheEnum(t *testing.T) {
	t.Parallel()

	_, err := shipments().Compile([]Condition{
		{Field: "status", Operator: dbtype.OpEqual, Value: "Teleporting"},
	}, utc())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "InTransit, Completed")
}

// A curated field that stops mapping to a column is the one drift the catalog
// cannot afford: the builder would skip it and return the page unfiltered.
func TestCompile_RefusesAFieldTheEntityNoLongerMaps(t *testing.T) {
	t.Parallel()

	resource := shipments()
	resource.Config = &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"proNumber": true},
	}
	resource = resource.Prepare()

	_, err := resource.Compile([]Condition{
		{Field: "status", Operator: dbtype.OpEqual, Value: "InTransit"},
	}, utc())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "can no longer be queried")
}

/*
A window is both ends of a range, never one.

OpNextNDays alone applies only `column <= now + N days`, so "shipping in the
next 7 days" would also match everything that shipped years ago. That is
survivable in a grid somebody sorts; it is not survivable in a sentence that
reports the count.
*/
func TestCompile_BoundsARelativeWindowAtBothEnds(t *testing.T) {
	t.Parallel()

	criteria := utc()
	filters, err := shipments().Compile([]Condition{
		{Field: "actualShipDate", Operator: dbtype.OpNextNDays, Days: 7},
	}, criteria)

	require.NoError(t, err)
	require.Len(t, filters, 2)
	assert.Equal(t, dbtype.OpGreaterThanOrEqual, filters[0].Operator)
	assert.Equal(t, dbtype.OpLessThanOrEqual, filters[1].Operator)
	assert.Equal(t, int64(1789776000), filters[0].Value, "opens at midnight today")
	assert.Equal(t, []string{"actualShipDate within the next 7 days"}, criteria.Terms())
}

func TestCompile_RefusesAWindowWithNoWidth(t *testing.T) {
	t.Parallel()

	for _, days := range []int{0, -1, MaxRelativeDays + 1} {
		_, err := shipments().Compile([]Condition{
			{Field: "actualShipDate", Operator: dbtype.OpNextNDays, Days: days},
		}, utc())

		require.Error(t, err, "days=%d", days)
	}
}

func TestCompile_ReadsAListOfValues(t *testing.T) {
	t.Parallel()

	filters, err := shipments().Compile([]Condition{
		{Field: "status", Operator: dbtype.OpIn, Values: []string{"InTransit", "Completed"}},
	}, utc())

	require.NoError(t, err)
	require.Len(t, filters, 1)
	assert.Equal(t, []any{"InTransit", "Completed"}, filters[0].Value)
}

func TestCompile_RefusesAnEmptyValueList(t *testing.T) {
	t.Parallel()

	_, err := shipments().Compile([]Condition{
		{Field: "status", Operator: dbtype.OpIn},
	}, utc())

	require.Error(t, err)
}

// An operator that takes no operand is not a missing value.
func TestCompile_TakesAnOperatorThatNeedsNoOperand(t *testing.T) {
	t.Parallel()

	criteria := utc()
	filters, err := shipments().Compile([]Condition{
		{Field: "actualShipDate", Operator: dbtype.OpIsNull},
	}, criteria)

	require.NoError(t, err)
	require.Len(t, filters, 1)
	assert.Nil(t, filters[0].Value)
	assert.Equal(t, []string{"actualShipDate is empty"}, criteria.Terms())
}

func TestCompile_RefusesAnOperandItNeeds(t *testing.T) {
	t.Parallel()

	_, err := shipments().Compile([]Condition{
		{Field: "proNumber", Operator: dbtype.OpContains},
	}, utc())

	require.Error(t, err)
}

// Compiling without criteria is legitimate — a caller that only wants filters
// should not have to build a terms accumulator to get them.
func TestCompile_WorksWithoutCriteria(t *testing.T) {
	t.Parallel()

	filters, err := shipments().Compile([]Condition{
		{Field: "proNumber", Operator: dbtype.OpContains, Value: "SEED"},
	}, nil)

	require.NoError(t, err)
	require.Len(t, filters, 1)
}

func TestSort_RefusesAFieldTheResourceCannotOrderBy(t *testing.T) {
	t.Parallel()

	_, err := shipments().Sort("totalChargeAmount", "asc")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "proNumber")
}

func TestSort_DefaultsToNewestFirst(t *testing.T) {
	t.Parallel()

	sort, err := shipments().Sort("actualShipDate", "")

	require.NoError(t, err)
	require.Len(t, sort, 1)
	assert.Equal(t, dbtype.SortDirectionDesc, sort[0].Direction)
}

// No ordering is not an error; it is the default ordering.
func TestSort_TakesNoFieldAsNoOrdering(t *testing.T) {
	t.Parallel()

	sort, err := shipments().Sort("", "asc")

	require.NoError(t, err)
	assert.Empty(t, sort)
}

func TestCatalog_FindsAResourceByPermissionAndByName(t *testing.T) {
	t.Parallel()

	catalog := New(shipments())

	byResource, ok := catalog.For(permission.ResourceShipment)
	require.True(t, ok)
	assert.Equal(t, "shipments", byResource.Entity)

	byEntity, ok := catalog.ByEntity("Shipments")
	require.True(t, ok)
	assert.Equal(t, permission.ResourceShipment, byEntity.Resource)

	_, ok = catalog.For(permission.ResourceWorker)
	assert.False(t, ok)
}
