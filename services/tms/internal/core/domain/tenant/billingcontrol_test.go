package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func newValidBillingControl() *BillingControl {
	return &BillingControl{
		ID:                                    pulid.MustNew("bc_"),
		BusinessUnitID:                        pulid.MustNew("bu_"),
		OrganizationID:                        pulid.MustNew("org_"),
		DefaultPaymentTerm:                    PaymentTermNet30,
		ShowDueDateOnInvoice:                  true,
		ShowBalanceDueOnInvoice:               true,
		ReadyToBillAssignmentMode:             ReadyToBillAssignmentModeManualOnly,
		BillingQueueTransferMode:              BillingQueueTransferModeManualOnly,
		InvoiceDraftCreationMode:              InvoiceDraftCreationModeManualOnly,
		InvoicePostingMode:                    InvoicePostingModeManualReviewRequired,
		ShipmentBillingRequirementEnforcement: EnforcementLevelBlock,
		RateValidationEnforcement:             EnforcementLevelRequireReview,
		BillingExceptionDisposition:           BillingExceptionDispositionRouteToBillingReview,
		NotifyOnBillingExceptions:             true,
		RateVarianceTolerancePercent:          decimal.Zero,
		RateVarianceAutoResolutionMode:        RateVarianceAutoResolutionModeDisabled,
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
		{"missing payment term fails", func(bc *BillingControl) { bc.DefaultPaymentTerm = "" }, true},
		{"missing ready to bill assignment mode fails", func(bc *BillingControl) { bc.ReadyToBillAssignmentMode = "" }, true},
		{"missing billing queue transfer mode fails", func(bc *BillingControl) { bc.BillingQueueTransferMode = "" }, true},
		{"missing invoice posting mode fails", func(bc *BillingControl) { bc.InvoicePostingMode = "" }, true},
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

func TestBillingControl_Getters(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("bc_")
	orgID := pulid.MustNew("org_")
	bc := &BillingControl{ID: id, OrganizationID: orgID}

	assert.Equal(t, "billing_controls", bc.GetTableName())
	assert.Equal(t, id, bc.GetID())
	assert.Equal(t, orgID, bc.GetOrganizationID())
}
