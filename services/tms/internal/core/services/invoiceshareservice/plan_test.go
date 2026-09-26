package invoiceshareservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreviewShareRendersEachEmailAndSendsNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	dana := h.recipient("Dana Whitfield")
	h.expectInvoiceAndSharer()
	h.expectUsers(dana)
	h.allow(dana)

	preview, err := h.svc.PreviewShare(
		h.ctxWithSharerActivation(t),
		h.request([]pulid.ID{dana.ID}, "  Check the detention line  ", invoice.ShareTabCharges),
		h.actor,
	)
	require.NoError(t, err)

	assert.Equal(t, "Check the detention line", preview.Note)
	assert.Equal(t, "Marcus Bell", preview.SharedByName)
	assert.True(t, preview.EmailConfigured)
	require.Len(t, preview.Recipients, 1)
	recipient := preview.Recipients[0]
	assert.True(t, recipient.Emailed)
	assert.Equal(t, dana.EmailAddress, recipient.EmailAddress)
	assert.Equal(t, "subject:invoice.share.email", recipient.Subject)
	assert.Empty(t, h.shares.upserted)
	assert.Empty(t, h.sentEmails)
	assert.Empty(t, h.notifications)
}

func TestPreviewShareRefusesWhatShareRefuses(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.svc.PreviewShare(
		t.Context(),
		h.request([]pulid.ID{h.sharer.ID}, "", invoice.ShareTabOverview),
		h.actor,
	)
	require.Error(t, err)
	requireFieldError(t, err, "userIds[0]")
}

func TestPreviewShareSaysWhenNoEmailWouldGo(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.cfg.App.WebBaseURL = ""
	dana := h.recipient("Dana Whitfield")
	h.expectInvoiceAndSharer()
	h.expectUsers(dana)
	h.allow(dana)

	preview, err := h.svc.PreviewShare(
		h.ctxWithSharerActivation(t),
		h.request([]pulid.ID{dana.ID}, "", invoice.ShareTabOverview),
		h.actor,
	)
	require.NoError(t, err)
	assert.False(t, preview.EmailConfigured)
	require.Len(t, preview.Recipients, 1)
	assert.False(t, preview.Recipients[0].Emailed)
}
