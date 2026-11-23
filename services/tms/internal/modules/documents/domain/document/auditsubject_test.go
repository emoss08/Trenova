package document

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestAuditSubject_NamesTheRecordTheDocumentIsAttachedTo(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	resource, id := (&Document{ResourceType: "worker", ResourceID: workerID.String()}).AuditSubject()
	assert.Equal(t, permission.ResourceWorker, resource)
	assert.Equal(t, workerID, id)

	resource, id = (&Document{}).AuditSubject()
	assert.Empty(t, resource)
	assert.True(t, id.IsNil())

	resource, id = (*Document)(nil).AuditSubject()
	assert.Empty(t, resource)
	assert.True(t, id.IsNil())
}
