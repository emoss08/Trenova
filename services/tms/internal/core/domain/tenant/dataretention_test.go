package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestDataRetention_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dr      *DataRetention
		wantErr bool
	}{
		{"valid entity passes", &DataRetention{AuditRetentionPeriod: 120}, false},
		{"zero retention period fails", &DataRetention{AuditRetentionPeriod: 0}, true},
		{"negative retention period fails", &DataRetention{AuditRetentionPeriod: -1}, true},
		{"retention period of 1 passes", &DataRetention{AuditRetentionPeriod: 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			multiErr := errortypes.NewMultiError()
			tt.dr.Validate(multiErr)
			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestDataRetention_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()
		dr := &DataRetention{}
		err := dr.BeforeAppendModel(nil, (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.False(t, dr.ID.IsNil())
		assert.NotZero(t, dr.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()
		existingID := pulid.MustNew("dr_")
		dr := &DataRetention{ID: existingID}
		err := dr.BeforeAppendModel(nil, (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.Equal(t, existingID, dr.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()
		dr := &DataRetention{}
		err := dr.BeforeAppendModel(nil, (*bun.UpdateQuery)(nil))
		require.NoError(t, err)
		assert.NotZero(t, dr.UpdatedAt)
	})
}

func TestDataRetention_GetTableName(t *testing.T) {
	t.Parallel()
	dr := &DataRetention{}
	assert.Equal(t, "data_retention", dr.GetTableName())
}

func TestDataRetention_GetID(t *testing.T) {
	t.Parallel()
	id := pulid.MustNew("dr_")
	dr := &DataRetention{ID: id}
	assert.Equal(t, id, dr.GetID())
}

func TestDataRetention_GetOrganizationID(t *testing.T) {
	t.Parallel()
	orgID := pulid.MustNew("org_")
	dr := &DataRetention{OrganizationID: orgID}
	assert.Equal(t, orgID, dr.GetOrganizationID())
}

func TestDataRetention_GetBusinessUnitID(t *testing.T) {
	t.Parallel()
	buID := pulid.MustNew("bu_")
	dr := &DataRetention{BusinessUnitID: buID}
	assert.Equal(t, buID, dr.GetBusinessUnitID())
}
