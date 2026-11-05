package documentupload

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestAConversationUploadIsPrivateToWhoeverIsUploadingIt(t *testing.T) {
	t.Parallel()

	uploader := pulid.MustNew("usr_")
	session := &DocumentUploadSession{
		ResourceType: document.ConversationResourceType,
		UploadedByID: uploader,
	}

	assert.True(t, session.VisibleToPerson(uploader))
	assert.False(t, session.VisibleToPerson(pulid.MustNew("usr_")))
	assert.False(t, session.VisibleToPerson(pulid.Nil))

	shipment := &DocumentUploadSession{ResourceType: "shipment", UploadedByID: uploader}
	assert.True(t, shipment.VisibleToPerson(pulid.MustNew("usr_")))
}
