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

package property_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_String(t *testing.T) {
	status := property.StatusActive
	assert.Equal(t, "Active", status.String())
}

func TestStatus_Values(t *testing.T) {
	values := property.Status("").Values()
	assert.Equal(t, []string{"Active", "Inactive"}, values)
}

func TestStatus_Value(t *testing.T) {
	status := property.StatusActive
	val, err := status.Value()
	require.NoError(t, err)
	assert.Equal(t, "Active", val)
}

func TestStatus_Scan(t *testing.T) {
	var status property.Status

	err := status.Scan("Active")
	require.NoError(t, err)
	assert.Equal(t, property.StatusActive, status)

	err = status.Scan(nil)
	require.Error(t, err)

	err = status.Scan(123)
	assert.Error(t, err)
}
