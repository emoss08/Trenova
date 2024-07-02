package property_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseActionType_String(t *testing.T) {
	dba := property.DatabaseActionAll
	assert.Equal(t, "All", dba.String())
}

func TestDatabaseActionType_Values(t *testing.T) {
	values := property.DatabaseAction("").Values()
	assert.Equal(t, []string{"Insert", "Update", "Delete", "All"}, values)
}

func TestDatabaseActionType_Value(t *testing.T) {
	dba := property.DatabaseActionAll
	val, err := dba.Value()
	require.NoError(t, err)
	assert.Equal(t, "All", val)
}

func TestDatabaseActionType_Scan(t *testing.T) {
	var dba property.OrganizationType

	err := dba.Scan("Insert")
	require.NoError(t, err)
	assert.Equal(t, property.DatabaseActionInsert, dba)

	err = dba.Scan(nil)
	require.Error(t, err)

	err = dba.Scan(123)
	assert.Error(t, err)
}
