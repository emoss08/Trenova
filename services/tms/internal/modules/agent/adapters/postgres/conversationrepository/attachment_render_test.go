package conversationrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

/*
A file counts as sent only when a person's saved message in that thread carries
it: an upload left in the composer is not part of the conversation. The test
asks for that thread, the person's own messages, and JSONB containment of the
document's id, written as a JSON array PostgreSQL can cast.
*/
func TestCarriesAttachmentAsksForThePersonsMessageInThatThread(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	documentID := pulid.MustNew("doc_")
	sql := buildCarriesAttachmentQuery(bun.NewDB(nil, pgdialect.New()),
		&agentports.CarriesAttachmentRequest{
			ThreadID:   threadID,
			DocumentID: documentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
		},
	).String()

	assert.Contains(
		t,
		sql,
		`amsg.attachments @> '[{"documentId":"`+documentID.String()+`"}]'::jsonb`,
		sql,
	)
	assert.Contains(t, sql, threadID.String(), sql)
	assert.Contains(t, sql, `amsg.role = 'User'`, sql)
}
