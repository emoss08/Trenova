package agent_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

/*
A kind of record is named in many tables, and each is held here to the one
list of kinds the server knows, the permission registry. A table that names a
kind the registry does not know is a record the runtime cannot check an id of,
and a page or a mention that names it is refused.
*/

func TestEveryRecordLinkIsAKindTheServerKnows(t *testing.T) {
	t.Parallel()

	for _, entity := range productguide.Default.RecordEntities() {
		assert.True(t, permission.RecordKind(entity).Known(), entity)
	}
}

func TestEveryAgentSubjectIsAKindOfRecord(t *testing.T) {
	t.Parallel()

	for _, subject := range agent.AllSubjectTypes() {
		kind, ok := subject.RecordKind()
		if !assert.True(t, ok, "%s names no kind of record", subject) {
			continue
		}
		assert.True(t, kind.Known(), "%s is %s, which the registry does not know", subject, kind)
		prefix, _ := kind.IDPrefix()
		assert.Equal(t, prefix, subject.IDPrefix(), subject)
		assert.Equal(t, kind.Noun(), subject.Noun(), subject)
	}
}

func TestEveryMemoryRecordKindIsAKindOfRecord(t *testing.T) {
	t.Parallel()

	for _, kind := range agent.AllMemoryRecordKinds() {
		recordKind := permission.RecordKind(kind)
		if !assert.True(t, recordKind.Known(), kind) {
			continue
		}
		prefix, _ := recordKind.IDPrefix()
		got, ok := agent.MemoryRecordKindOfID(pulid.MustNew(prefix))
		assert.True(t, ok, kind)
		assert.Equal(t, kind, got)
	}
}
