package invoiceadjustmentengine

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func writeOffRun() *adjustmentRun {
	orgID := pulid.MustNew("org_")
	return &adjustmentRun{
		adjustment: &invoiceadjustment.InvoiceAdjustment{
			ID:             pulid.MustNew("iadj_"),
			OrganizationID: orgID,
			BusinessUnitID: pulid.MustNew("bu_"),
			Kind:           invoiceadjustment.KindWriteOff,
		},
		source: &invoice.Invoice{ID: pulid.MustNew("inv_"), Number: "INV-1001"},
		computation: &PreviewComputation{
			Preview: &invoiceadjustmentports.InvoiceAdjustmentPreview{
				CreditTotalAmount: decimal.NewFromInt(-250),
			},
		},
		actor: &permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	}
}

func TestRecordWriteOff_WritesNoJournalWithoutAccounting(t *testing.T) {
	t.Parallel()

	accounting := tenantportsmock.NewMockAccountingControlRepository(t)
	svc := &Engine{accountingRepo: accounting, features: plantest.BaseOnly()}
	run := writeOffRun()

	require.NoError(t, svc.recordWriteOff(t.Context(), run))

	assert.True(t, run.writeOffJournalID.IsNil())
	assert.NotContains(t, run.adjustment.Metadata, "writeOffJournalEntryId")
	accounting.AssertNotCalled(t, "GetByOrgID")
}

func TestRecordWriteOff_PostsTheJournalWithAccounting(t *testing.T) {
	t.Parallel()

	run := writeOffRun()
	accounting := tenantportsmock.NewMockAccountingControlRepository(t)
	accounting.EXPECT().
		GetByOrgID(mock.Anything, run.adjustment.OrganizationID).
		Return(&accountingcontrol.AccountingControl{}, nil).
		Once()
	svc := &Engine{accountingRepo: accounting, features: plantest.AllFeatures()}

	err := svc.recordWriteOff(t.Context(), run)

	require.Error(t, err, "the journal path runs and asks for its accounts")
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "defaultWriteOffAccountId", validation.Field())
}
