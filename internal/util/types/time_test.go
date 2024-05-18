package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/util/types"
)

func TestTimeOnlyScan(t *testing.T) {
	time := &types.TimeOnly{}

	err := time.Scan("12:34:56")
	require.NoError(t, err)
	assert.Equal(t, "12:34:56", time.Time.Format("15:04:05"))
}

func TestTimeOnlyScanError(t *testing.T) {
	time := &types.TimeOnly{}

	err := time.Scan(123)
	require.Error(t, err)
	assert.Equal(t, "unsupported type int, expected string", err.Error())
}

func TestTimeOnlyMarshalJSON(t *testing.T) {
	time := &types.TimeOnly{}
	if err := time.Scan("12:34:56"); err != nil {
		t.Fatal(err)
	}

	b, err := time.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"12:34:56"`, string(b))
}

func TestTimeOnlyUnmarshalJSON(t *testing.T) {
	time := &types.TimeOnly{}

	err := time.UnmarshalJSON([]byte(`"12:34:56"`))
	require.NoError(t, err)
	assert.Equal(t, "12:34:56", time.Time.Format("15:04:05"))
}

func TestTimeOnlyUnmarshalJSONEmpty(t *testing.T) {
	time := &types.TimeOnly{}

	err := time.UnmarshalJSON([]byte(`""`))
	require.NoError(t, err)
	assert.True(t, time.Time.IsZero())
}

func TestTimeOnlyUnmarshalJSONError(t *testing.T) {
	time := &types.TimeOnly{}

	err := time.UnmarshalJSON([]byte(`123`))
	require.Error(t, err)
}

func TestTimeOnlyValue(t *testing.T) {
	time := &types.TimeOnly{}
	if err := time.Scan("12:34:56"); err != nil {
		t.Fatal(err)
	}

	v, err := time.Value()
	require.NoError(t, err)
	assert.Equal(t, "12:34:56", v)
}

func TestTimeOnlyValueZero(t *testing.T) {
	time := &types.TimeOnly{}

	v, err := time.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestTimeOnlyValueError(t *testing.T) {
	time := &types.TimeOnly{}

	v, err := time.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestTimeOnlySchemaType(t *testing.T) {
	time := types.TimeOnly{}

	schemaType := time.SchemaType()
	assert.Equal(t, map[string]string{
		"postgres": "time",
		"sqlite3":  "text",
	}, schemaType)
}
