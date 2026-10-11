package tenant

import (
	"context"
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
		{"valid entity passes", validRetention(120, DefaultAIAuditRetentionDays), false},
		{"zero retention period fails", validRetention(0, DefaultAIAuditRetentionDays), true},
		{"negative retention period fails", validRetention(-1, DefaultAIAuditRetentionDays), true},
		{"retention period of 1 passes", validRetention(1, DefaultAIAuditRetentionDays), false},
		{"missing AI audit trail retention fails", validRetention(120, 0), true},
		{"AI audit trail kept under a year fails", validRetention(120, 364), true},
		{"AI audit trail kept a year passes", validRetention(120, MinAIAuditRetentionDays), false},
		{"AI audit trail kept ten years passes", validRetention(120, 3650), false},
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

func validRetention(auditDays, aiAuditDays int) *DataRetention {
	return &DataRetention{AuditRetentionPeriod: auditDays, AIAuditRetentionPeriod: aiAuditDays}
}

func TestDataRetention_AIAuditRetentionDays(t *testing.T) {
	t.Parallel()

	var missing *DataRetention
	assert.Equal(t, DefaultAIAuditRetentionDays, missing.AIAuditRetentionDays())
	assert.Equal(t, DefaultAIAuditRetentionDays, (&DataRetention{}).AIAuditRetentionDays())
	assert.Equal(t, 400, (&DataRetention{AIAuditRetentionPeriod: 400}).AIAuditRetentionDays())
}

func TestDataRetention_AIAuditRetentionError_NamesTheField(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	validRetention(120, 30).Validate(multiErr)

	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "aiAuditRetentionPeriod", multiErr.Errors[0].Field)
}

func TestDataRetention_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()
		dr := &DataRetention{}
		err := dr.BeforeAppendModel(context.TODO(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.False(t, dr.ID.IsNil())
		assert.NotZero(t, dr.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()
		existingID := pulid.MustNew("dr_")
		dr := &DataRetention{ID: existingID}
		err := dr.BeforeAppendModel(context.TODO(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.Equal(t, existingID, dr.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()
		dr := &DataRetention{}
		err := dr.BeforeAppendModel(context.TODO(), (*bun.UpdateQuery)(nil))
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
