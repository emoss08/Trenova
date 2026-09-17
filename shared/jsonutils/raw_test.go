package jsonutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawJSONValueWritesTheDocumentItself(t *testing.T) {
	t.Parallel()

	value, err := jsonutils.RawJSON(`{"dot_number":"265752"}`).Value()
	require.NoError(t, err)
	assert.Equal(t, `{"dot_number":"265752"}`, value)

	value, err = jsonutils.RawJSON(`not json`).Value()
	require.NoError(t, err)
	assert.Equal(t, `"not json"`, value)

	value, err = jsonutils.RawJSON(nil).Value()
	require.NoError(t, err)
	assert.Nil(t, value)
}

func TestRawJSONScanUnwrapsLegacyStringScalars(t *testing.T) {
	t.Parallel()

	var raw jsonutils.RawJSON
	require.NoError(t, raw.Scan([]byte(`{"a":1}`)))
	assert.JSONEq(t, `{"a":1}`, raw.String())

	require.NoError(t, raw.Scan(`"{\"a\":1}"`))
	assert.JSONEq(t, `{"a":1}`, raw.String())

	require.NoError(t, raw.Scan(`"plain text"`))
	assert.Equal(t, `"plain text"`, raw.String())

	require.NoError(t, raw.Scan(nil))
	assert.Nil(t, raw)

	assert.Error(t, raw.Scan(42))
}

func TestRawJSONMarshal(t *testing.T) {
	t.Parallel()

	encoded, err := jsonutils.RawJSON(`[1,2]`).MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `[1,2]`, string(encoded))

	encoded, err = jsonutils.RawJSON(nil).MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `null`, string(encoded))
}
