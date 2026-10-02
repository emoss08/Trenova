package realtimeinvalidation

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPublishInvoiceSendsSnapshot(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Number:         "INV-1",
		SendStatus:     invoice.SendStatusFailed,
		LastSendError:  "Resend refused to send",
	}
	actor := servicesport.AuditActor{UserID: pulid.MustNew("usr_")}

	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(
			func(req *servicesport.PublishResourceInvalidationRequest) bool {
				snapshot, ok := req.Entity.(*InvoiceSnapshot)
				return ok &&
					req.Resource == "invoice" &&
					req.Action == InvoiceActionSendUpdated &&
					req.RecordID == entity.ID &&
					req.OrganizationID == entity.OrganizationID &&
					req.ActorUserID == actor.UserID &&
					snapshot.Number == "INV-1" &&
					snapshot.SendStatus == invoice.SendStatusFailed &&
					snapshot.LastSendError == "Resend refused to send"
			},
		)).
		Return(nil).
		Once()

	require.NoError(t, PublishInvoice(t.Context(), realtime, &InvoiceChange{
		Invoice: entity,
		Actor:   actor,
		Action:  InvoiceActionSendUpdated,
	}))
}

func TestPublishInvoiceDefaultsAndGuards(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, PublishInvoice(t.Context(), nil, nil), ErrPublishParamsRequired)
	require.NoError(t, PublishInvoice(t.Context(), nil, &InvoiceChange{Invoice: &invoice.Invoice{}}))

	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(
			func(req *servicesport.PublishResourceInvalidationRequest) bool {
				return req.Action == InvoiceActionUpdated
			},
		)).
		Return(nil).
		Once()
	require.NoError(t, PublishInvoice(t.Context(), realtime, &InvoiceChange{
		Invoice: &invoice.Invoice{ID: pulid.MustNew("inv_")},
	}))
}
