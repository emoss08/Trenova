package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func newValidBillingControl() *BillingControl {
	return &BillingControl{
		ID:                        pulid.MustNew("bc_"),
		BusinessUnitID:            pulid.MustNew("bu_"),
		OrganizationID:            pulid.MustNew("org_"),
		PaymentTerm:               PaymentTermNet30,
		BillingExceptionHandling:  BillingExceptionQueue,
		TransferSchedule:          TransferScheduleContinuous,
		AutoTransfer:              true,
		TransferBatchSize:         100,
		RateDiscrepancyThreshold:  5.00,
		AllowInvoiceConsolidation: true,
		ConsolidationPeriodDays:   7,
	}
}

func TestBillingControl_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(*BillingControl)
		wantErr bool
	}{
		{"valid entity passes", func(_ *BillingControl) {}, false},
		{"missing payment term fails", func(bc *BillingControl) { bc.PaymentTerm = "" }, true},
		{
			"invalid payment term fails",
			func(bc *BillingControl) { bc.PaymentTerm = "Invalid" },
			true,
		},
		{
			"Net15 payment term passes",
			func(bc *BillingControl) { bc.PaymentTerm = PaymentTermNet15 },
			false,
		},
		{
			"DueOnReceipt payment term passes",
			func(bc *BillingControl) { bc.PaymentTerm = PaymentTermDueOnReceipt },
			false,
		},
		{
			"missing exception handling fails",
			func(bc *BillingControl) { bc.BillingExceptionHandling = "" },
			true,
		},
		{
			"invalid exception handling fails",
			func(bc *BillingControl) { bc.BillingExceptionHandling = "Bad" },
			true,
		},
		{
			"Notify exception handling passes",
			func(bc *BillingControl) { bc.BillingExceptionHandling = BillingExceptionNotify },
			false,
		},
		{
			"zero rate threshold fails",
			func(bc *BillingControl) { bc.RateDiscrepancyThreshold = 0 },
			true,
		},
		{"auto transfer requires batch size", func(bc *BillingControl) {
			bc.AutoTransfer = true
			bc.TransferBatchSize = 0
		}, true},
		{"invalid transfer schedule fails", func(bc *BillingControl) {
			bc.AutoTransfer = true
			bc.TransferSchedule = "Invalid"
		}, true},
		{"consolidation requires period days", func(bc *BillingControl) {
			bc.AllowInvoiceConsolidation = true
			bc.ConsolidationPeriodDays = 0
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bc := newValidBillingControl()
			tt.modify(bc)
			multiErr := errortypes.NewMultiError()
			bc.Validate(multiErr)
			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestBillingControl_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()
		bc := &BillingControl{}
		err := bc.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.False(t, bc.ID.IsNil())
		assert.NotZero(t, bc.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()
		existingID := pulid.MustNew("bc_")
		bc := &BillingControl{ID: existingID}
		err := bc.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.Equal(t, existingID, bc.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()
		bc := &BillingControl{}
		err := bc.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)
		assert.NotZero(t, bc.UpdatedAt)
	})
}

func TestBillingControl_GetTableName(t *testing.T) {
	t.Parallel()
	bc := &BillingControl{}
	assert.Equal(t, "billing_controls", bc.GetTableName())
}

func TestBillingControl_GetID(t *testing.T) {
	t.Parallel()
	id := pulid.MustNew("bc_")
	bc := &BillingControl{ID: id}
	assert.Equal(t, id, bc.GetID())
}

func TestBillingControl_GetOrganizationID(t *testing.T) {
	t.Parallel()
	orgID := pulid.MustNew("org_")
	bc := &BillingControl{OrganizationID: orgID}
	assert.Equal(t, orgID, bc.GetOrganizationID())
}
