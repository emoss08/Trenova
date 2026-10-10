package agentquerytoolservice

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The absence that got read as "fine".

Three of the eight drivers in the incident had no medical card date. The agent
listed them under "drivers with no medical card expiry" and concluded nobody
was expiring — when a driver with no certificate on file is the one who cannot
be dispatched at all, and the one whose card expires in three weeks still can.

The field was int64 with omitempty, so an unrecorded certificate produced no
key. There was nothing in the result for a reader to notice.
*/
func TestOptionalDate_SaysWhenNothingIsOnFile(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.Marshal(recordedDate(0))
	require.NoError(t, err)

	assert.JSONEq(t, `"none on file"`, string(encoded))
}

// A set date goes out as a bare number so the runtime's date pass renders it.
// Anything else would arrive as an epoch the reader cannot use.
func TestOptionalDate_EmitsTheInstantForTheDatePassToRender(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.Marshal(recordedDate(1791591001))
	require.NoError(t, err)

	assert.JSONEq(t, `1791591001`, string(encoded))
}

/*
"Nobody wrote it down" and "it has not happened yet" are different facts.

A shipment with no delivery date is in transit. A driver with no medical card
is a gap in the record. Rendering both as the same phrase would tell a reader
that an in-transit load is missing paperwork.
*/
func TestOptionalDate_DistinguishesUnrecordedFromNotYetHappened(t *testing.T) {
	t.Parallel()

	unrecorded, err := sonic.Marshal(recordedDate(0))
	require.NoError(t, err)
	pending, err := sonic.Marshal(expectedDate(0, "not delivered yet"))
	require.NoError(t, err)

	assert.JSONEq(t, `"none on file"`, string(unrecorded))
	assert.JSONEq(t, `"not delivered yet"`, string(pending))
	assert.NotEqual(t, string(unrecorded), string(pending))
}

func TestPointerDate_TreatsNilAsNothingOnFile(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.Marshal(pointerDate(nil))
	require.NoError(t, err)
	assert.JSONEq(t, `"none on file"`, string(encoded))

	seconds := int64(1791591001)
	set, err := sonic.Marshal(pointerDate(&seconds))
	require.NoError(t, err)
	assert.JSONEq(t, `1791591001`, string(set))
}

// A negative or zero instant is not a date anybody set. Emitting it would put
// 1970 on the page and invent a lapsed credential for every driver who has not
// filed one.
func TestOptionalDate_RefusesAnInstantNobodySet(t *testing.T) {
	t.Parallel()

	for _, seconds := range []int64{0, -1} {
		encoded, err := sonic.Marshal(recordedDate(seconds))
		require.NoError(t, err)
		assert.JSONEq(t, `"none on file"`, string(encoded))
	}
}

// The compliance fields are the reason the row was asked for, so they are
// never omitted — that is the whole change.
func TestWorkerRow_AlwaysCarriesTheCredentialDates(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.Marshal(workerRow{})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &decoded))

	for _, key := range []string{"medicalCardExpiry", "licenseExpiry", "hazmatExpiry"} {
		value, present := decoded[key]
		assert.True(t, present, "%s must be present even when nothing is on file", key)
		assert.Equal(t, "none on file", value)
	}
}

/*
The zero value has to be safe.

applyProfile returns early for a worker with no qualification profile, so the
row's dates are never constructed and reach the encoder as the zero value. That
produced an empty string, which reads as a blank field rather than a missing
certificate — the same silence the omitempty tag produced, arrived at a
different way.
*/
func TestOptionalDate_ZeroValueStillSaysNothingIsOnFile(t *testing.T) {
	t.Parallel()

	var unconstructed optionalDate

	encoded, err := sonic.Marshal(unconstructed)
	require.NoError(t, err)

	assert.JSONEq(t, `"none on file"`, string(encoded))
}
