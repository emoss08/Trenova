package pgfield_test

import (
	"testing"

	pgtimeonly "github.com/emoss08/trenova/pkg/models/pgfield"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeOnlyScan(t *testing.T) {
	time := &pgtimeonly.TimeOnly{}

	err := time.Scan("12:34:56")
	require.NoError(t, err)
	assert.Equal(t, "12:34:56", time.Time.Format("15:04:05"))
}

func TestTimeOnlyScanError(t *testing.T) {
	time := &pgtimeonly.TimeOnly{}

	err := time.Scan(123)
	require.Error(t, err)
	assert.Equal(t, "unsupported type int, expected string", err.Error())
}

func TestTimeOnlyMarshalJSON(t *testing.T) {
	time := &pgtimeonly.TimeOnly{}
	if err := time.Scan("12:34:56"); err != nil {
		t.Fatal(err)
	}

	b, err := time.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"12:34:56"`, string(b))
}

func TestTimeOnlyUnmarshalJSON(t *testing.T) {
	time := &pgtimeonly.TimeOnly{}

	err := time.UnmarshalJSON([]byte(`"12:34:56"`))
	require.NoError(t, err)
	assert.Equal(t, "12:34:56", time.Time.Format("15:04:05"))
}

func TestTimeOnlyUnmarshalJSONEmpty(t *testing.T) {
	time := &pgtimeonly.TimeOnly{}

	err := time.UnmarshalJSON([]byte(`""`))
	require.NoError(t, err)
	assert.True(t, time.Time.IsZero())
}

func TestTimeOnlyUnmarshalJSONError(t *testing.T) {
	time := &pgtimeonly.TimeOnly{}

	err := time.UnmarshalJSON([]byte(`123`))
	require.Error(t, err)
}

func TestTimeOnlyValue(t *testing.T) {
	time := &pgtimeonly.TimeOnly{}
	if err := time.Scan("12:34:56"); err != nil {
		t.Fatal(err)
	}

	v, err := time.Value()
	require.NoError(t, err)
	assert.Equal(t, "12:34:56", v)
}

func TestTimeOnlyValueZero(t *testing.T) {
	time := &pgtimeonly.TimeOnly{}

	v, err := time.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestTimeOnlyValueError(t *testing.T) {
	time := &pgtimeonly.TimeOnly{}

	v, err := time.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}
