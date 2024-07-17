// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
