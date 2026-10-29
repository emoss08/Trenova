package billingtransferjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func automaticRun(
	status billingtransfer.RunStatus,
	transferred, notTransferred int,
) *billingtransfer.BillingTransferRun {
	return &billingtransfer.BillingTransferRun{
		ID:                  pulid.MustNew("btr_"),
		OrganizationID:      pulid.MustNew("org_"),
		BusinessUnitID:      pulid.MustNew("bu_"),
		Scope:               billingtransfer.RunScopeAutomatic,
		Status:              status,
		TotalCount:          transferred + notTransferred,
		TransferredCount:    transferred,
		NotTransferredCount: notTransferred,
	}
}

func TestAutomaticWordingAlwaysReportsShipmentsLeftBehind(t *testing.T) {
	t.Parallel()

	a := &Activities{l: zap.NewNop()}

	eventType, priority, title, message, ok := a.automaticWording(
		t.Context(), automaticRun(billingtransfer.RunStatusCompleted, 8, 2),
	)
	assert.True(t, ok)
	assert.Equal(t, eventRunCompleted, eventType)
	assert.Equal(t, notification.PriorityHigh, priority)
	assert.Equal(t, "Automatic transfer to billing needs attention", title)
	assert.Contains(t, message, "8 of 10 ready shipments")
	assert.Contains(t, message, "2 could not be moved")

	eventType, _, _, _, ok = a.automaticWording(
		t.Context(),
		automaticRun(billingtransfer.RunStatusFailed, 3, 0),
	)
	assert.True(t, ok)
	assert.Equal(t, eventRunFailed, eventType)

	_, _, _, _, ok = a.automaticWording(
		t.Context(),
		automaticRun(billingtransfer.RunStatusCanceled, 3, 4),
	)
	assert.False(t, ok, "whoever stopped it already knows")
}

func TestAutomaticWordingReportsACleanRunOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	run := automaticRun(billingtransfer.RunStatusCompleted, 5, 0)
	for _, notify := range []bool{false, true} {
		controls := tenantportsmock.NewMockBillingControlRepository(t)
		controls.EXPECT().
			GetByOrgID(mock.Anything, run.OrganizationID).
			Return(&billingcontrol.BillingControl{NotifyOnAutoInvoiceCreation: notify}, nil).
			Once()
		a := &Activities{l: zap.NewNop(), controls: controls}

		_, priority, title, _, ok := a.automaticWording(t.Context(), run)

		assert.Equal(t, notify, ok)
		if notify {
			assert.Equal(t, notification.PriorityLow, priority)
			assert.Equal(t, "Shipments moved into billing automatically", title)
		}
	}
}
