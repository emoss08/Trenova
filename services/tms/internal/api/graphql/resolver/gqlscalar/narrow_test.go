package gqlscalar

import (
	"math"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireRangeError(t *testing.T, err error, field string, minimum, maximum int64) {
	t.Helper()

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, field, valErr.Field())
	assert.Equal(t, errortypes.ErrInvalid, valErr.Code)
	assert.Equal(t, "Must be between {0} and {1}", valErr.Message)
	assert.Equal(t, []any{minimum, maximum}, valErr.Args)
}

func TestNarrowInt_Int32(t *testing.T) {
	t.Parallel()

	got, err := NarrowInt[int32]("waitingPeriodDays", 15)
	require.NoError(t, err)
	assert.Equal(t, int32(15), got)

	got, err = NarrowInt[int32]("waitingPeriodDays", math.MaxInt32)
	require.NoError(t, err)
	assert.Equal(t, int32(math.MaxInt32), got)

	got, err = NarrowInt[int32]("waitingPeriodDays", math.MinInt32)
	require.NoError(t, err)
	assert.Equal(t, int32(math.MinInt32), got)

	_, err = NarrowInt[int32]("waitingPeriodDays", math.MaxInt32+1)
	requireRangeError(t, err, "waitingPeriodDays", math.MinInt32, math.MaxInt32)

	_, err = NarrowInt[int32]("waitingPeriodDays", 4294967311)
	requireRangeError(t, err, "waitingPeriodDays", math.MinInt32, math.MaxInt32)

	_, err = NarrowInt[int32]("waitingPeriodDays", math.MinInt32-1)
	requireRangeError(t, err, "waitingPeriodDays", math.MinInt32, math.MaxInt32)
}

func TestNarrowInt_Int16(t *testing.T) {
	t.Parallel()

	got, err := NarrowInt[int16]("tiers[2].minMonths", math.MaxInt16)
	require.NoError(t, err)
	assert.Equal(t, int16(math.MaxInt16), got)

	_, err = NarrowInt[int16]("tiers[2].minMonths", math.MaxInt16+1)
	requireRangeError(t, err, "tiers[2].minMonths", math.MinInt16, math.MaxInt16)

	_, err = NarrowInt[int16]("tiers[2].minMonths", math.MinInt16-1)
	requireRangeError(t, err, "tiers[2].minMonths", math.MinInt16, math.MaxInt16)
}

func TestNarrowInt_Int8AndInt64(t *testing.T) {
	t.Parallel()

	_, err := NarrowInt[int8]("level", math.MaxInt8+1)
	requireRangeError(t, err, "level", math.MinInt8, math.MaxInt8)

	got, err := NarrowInt[int64]("version", math.MaxInt64)
	require.NoError(t, err)
	assert.Equal(t, int64(math.MaxInt64), got)
}

func TestNarrowInt_NamedType(t *testing.T) {
	t.Parallel()

	type minutes int16

	got, err := NarrowInt[minutes]("graceMinutes", 30)
	require.NoError(t, err)
	assert.Equal(t, minutes(30), got)

	_, err = NarrowInt[minutes]("graceMinutes", math.MaxInt16+1)
	requireRangeError(t, err, "graceMinutes", math.MinInt16, math.MaxInt16)
}

func TestNarrowIntValueAndPtr_TreatNilAsUnset(t *testing.T) {
	t.Parallel()

	value, err := NarrowIntValue[int32]("breakMinutes", nil)
	require.NoError(t, err)
	assert.Equal(t, int32(0), value)

	ptr, err := NarrowIntPtr[int32]("breakMinutes", nil)
	require.NoError(t, err)
	assert.Nil(t, ptr)

	input := 45
	ptr, err = NarrowIntPtr[int32]("breakMinutes", &input)
	require.NoError(t, err)
	require.NotNil(t, ptr)
	assert.Equal(t, int32(45), *ptr)

	tooLarge := math.MaxInt32 + 1
	ptr, err = NarrowIntPtr[int32]("breakMinutes", &tooLarge)
	requireRangeError(t, err, "breakMinutes", math.MinInt32, math.MaxInt32)
	assert.Nil(t, ptr)
}

func TestApplyInt_LeavesTheDestinationOnNilOrError(t *testing.T) {
	t.Parallel()

	dest := int16(7)
	require.NoError(t, ApplyInt("priority", nil, &dest))
	assert.Equal(t, int16(7), dest)

	next := 9
	require.NoError(t, ApplyInt("priority", &next, &dest))
	assert.Equal(t, int16(9), dest)

	tooLarge := math.MaxInt16 + 1
	requireRangeError(
		t,
		ApplyInt("priority", &tooLarge, &dest),
		"priority",
		math.MinInt16,
		math.MaxInt16,
	)
	assert.Equal(t, int16(9), dest)
}

func TestApplyIntPtr_LeavesTheDestinationOnNilOrError(t *testing.T) {
	t.Parallel()

	var dest *int32
	require.NoError(t, ApplyIntPtr("eligibilityHoursWorked", nil, &dest))
	assert.Nil(t, dest)

	next := 1250
	require.NoError(t, ApplyIntPtr("eligibilityHoursWorked", &next, &dest))
	require.NotNil(t, dest)
	assert.Equal(t, int32(1250), *dest)

	tooSmall := math.MinInt32 - 1
	requireRangeError(
		t,
		ApplyIntPtr("eligibilityHoursWorked", &tooSmall, &dest),
		"eligibilityHoursWorked",
		math.MinInt32,
		math.MaxInt32,
	)
	assert.Equal(t, int32(1250), *dest)
}

func TestApplyRequiredInt_RefusesAValueThatWouldWrap(t *testing.T) {
	t.Parallel()

	var version int32 = 3
	wrapping := 4294967311
	err := ApplyRequiredInt("version", "Version", graphql.OmittableOf(&wrapping), &version)
	requireRangeError(t, err, "version", math.MinInt32, math.MaxInt32)
	assert.Equal(t, int32(3), version)

	maximum := math.MaxInt32
	require.NoError(
		t,
		ApplyRequiredInt("version", "Version", graphql.OmittableOf(&maximum), &version),
	)
	assert.Equal(t, int32(math.MaxInt32), version)
}
