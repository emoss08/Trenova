package billingqueueservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func liveDocument(id pulid.ID) *document.Document {
	return &document.Document{ID: id, IsCurrentVersion: true, Status: document.StatusActive}
}

func TestADocumentTheShipmentNoLongerHoldsDropsOutOfThePacketItWasIn(t *testing.T) {
	deleted, uploaded := pulid.MustNew("doc_"), pulid.MustNew("doc_")

	kept, err := carriedPacket(
		[]pulid.ID{deleted, uploaded},
		[]pulid.ID{deleted},
		[]*document.Document{liveDocument(uploaded)},
	)

	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{uploaded}, kept)
}

func TestADocumentTheShipmentDoesNotHoldCannotBeAddedToThePacket(t *testing.T) {
	stranger, pod := pulid.MustNew("doc_"), pulid.MustNew("doc_")

	_, err := carriedPacket(
		[]pulid.ID{pod, stranger},
		[]pulid.ID{pod},
		[]*document.Document{liveDocument(pod)},
	)

	var invalid *errortypes.Error
	require.True(t, errors.As(err, &invalid), "got %v", err)
	assert.Equal(t, "packetDocumentIds", invalid.Field())
}

func TestARejectedOrSupersededDocumentIsNotCurrent(t *testing.T) {
	rejected, superseded := pulid.MustNew("doc_"), pulid.MustNew("doc_")
	docs := []*document.Document{
		{ID: rejected, IsCurrentVersion: true, Status: document.StatusRejected},
		{ID: superseded, IsCurrentVersion: false, Status: document.StatusActive},
	}

	kept, err := carriedPacket(
		[]pulid.ID{rejected, superseded},
		[]pulid.ID{rejected, superseded},
		docs,
	)

	require.NoError(t, err)
	assert.Empty(t, kept)
}
