package invoicecreationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprovingCarriesEachDocumentsTurnsOntoItsAttachment(t *testing.T) {
	pod, bol := pulid.MustNew("doc_"), pulid.MustNew("doc_")
	item := &billingqueue.BillingQueueItem{
		PacketPageRotations: []billingqueue.PacketPageRotation{
			{DocumentID: pod, Page: 2, Rotation: 90},
			{DocumentID: pod, Page: 4, Rotation: 180},
		},
	}

	attachments := packetAttachments([]pulid.ID{pod, bol}, item)

	require.Len(t, attachments, 2)
	assert.Equal(t, pod, attachments[0].DocumentID)
	assert.True(t, attachments[0].Selected)
	assert.Equal(t, []billingvocab.PageRotation{
		{Page: 2, Rotation: 90},
		{Page: 4, Rotation: 180},
	}, attachments[0].PageRotations)
	assert.Equal(t, []billingvocab.PageRotation{}, attachments[1].PageRotations)
}
