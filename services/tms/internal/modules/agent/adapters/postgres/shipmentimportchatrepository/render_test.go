package shipmentimportchatrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/shipmentimportchat"
	"github.com/emoss08/trenova/internal/modules/agent/ports/shipmentimportchatports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestConversationByDocumentQuery(t *testing.T) {
	t.Parallel()

	active, activeArgs := keeltest.Render(t, conversationByDocumentQuery(
		shipmentimportchatports.GetShipmentImportConversationRequest{DocumentID: pulid.ID("doc_1")},
	))
	assert.Contains(t, active, `ORDER BY sicc."status" = $4 DESC, sicc."updated_at" DESC`)
	assert.Equal(t, string(shipmentimportchat.ConversationStatusActive), activeArgs[3])
	assert.Contains(t, active, "LIMIT 1")

	byStatus, args := keeltest.Render(t, conversationByDocumentQuery(
		shipmentimportchatports.GetShipmentImportConversationRequest{
			DocumentID: pulid.ID("doc_1"),
			Status:     shipmentimportchat.ConversationStatusCompleted,
		},
	))
	assert.Contains(t, byStatus, `ORDER BY sicc."updated_at" DESC`)
	assert.Contains(t, args, string(shipmentimportchat.ConversationStatusCompleted))
}

func TestListTurnsQuery(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, listTurnsQuery(
		shipmentimportchatports.ListShipmentImportTurnsRequest{ConversationID: pulid.ID("sic_1")},
	))
	assert.Contains(t, sql, `"turn_index" ASC`)
}

func TestArchiveActiveQuery(t *testing.T) {
	t.Parallel()

	_, args := keeltest.Render(t, archiveActiveQuery(pulid.ID("doc_1")))
	assert.Contains(t, args, string(shipmentimportchat.ConversationStatusActive))
	assert.Contains(t, args, "doc_1")
}
