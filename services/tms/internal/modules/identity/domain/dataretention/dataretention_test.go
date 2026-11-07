package dataretention

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataRetention_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dr      *DataRetention
		wantErr bool
	}{
		{"valid entity passes", validRetention(MinAuditAccessRetentionDays, DefaultAIAuditRetentionDays), false},
		{"unset audit classes take their floors", validRetention(0, DefaultAIAuditRetentionDays), false},
		{"reads kept under 90 days fail", validRetention(89, DefaultAIAuditRetentionDays), true},
		{"reads kept past ten years fail", validRetention(MaxAuditRetentionDays+1, DefaultAIAuditRetentionDays), true},
		{"reads kept ten years pass", validRetention(MaxAuditRetentionDays, DefaultAIAuditRetentionDays), false},
		{"financial entries kept under seven years fail", &DataRetention{
			AuditFinancialRetentionDays: 2554,
			AIAuditRetentionPeriod:      DefaultAIAuditRetentionDays,
		}, true},
		{"operational entries kept under two years fail", &DataRetention{
			AuditOperationalRetentionDays: 729,
			AIAuditRetentionPeriod:        DefaultAIAuditRetentionDays,
		}, true},
		{"security entries kept under two years fail", &DataRetention{
			AuditSecurityRetentionDays: 729,
			AIAuditRetentionPeriod:     DefaultAIAuditRetentionDays,
		}, true},
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

func validRetention(accessDays, aiAuditDays int) *DataRetention {
	return &DataRetention{AuditAccessRetentionDays: accessDays, AIAuditRetentionPeriod: aiAuditDays}
}

func TestDataRetention_AuditRetentionDays_NamesEachClass(t *testing.T) {
	t.Parallel()

	dr := &DataRetention{
		AuditAccessRetentionDays:      100,
		AuditOperationalRetentionDays: 800,
		AuditFinancialRetentionDays:   3000,
		AuditSecurityRetentionDays:    900,
	}
	assert.Equal(t, 100, dr.AuditRetentionDays("Access"))
	assert.Equal(t, 800, dr.AuditRetentionDays("Operational"))
	assert.Equal(t, 3000, dr.AuditRetentionDays("Financial"))
	assert.Equal(t, 900, dr.AuditRetentionDays("Security"))
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
