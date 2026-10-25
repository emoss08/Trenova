package assignmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestGetByMoveID_IgnoresArchivedAssignments(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	sql, args := keeltest.Render(t, liveOnMove(moveID))

	assert.Regexp(t, `"archived_at" IS NULL`, sql)
	assert.Regexp(t, `"shipment_move_id" = \$\d+`, sql)
	assert.Regexp(t, `"organization_id" = \$\d+`, sql)
	assert.Contains(t, args, moveID.String())
}

func TestList_ExcludesArchivedAssignments(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, withResources(liveAssignments()).Limit(10))

	assert.Regexp(t, `"archived_at" IS NULL`, sql)
	assert.Regexp(t, `"organization_id" = \$\d+`, sql)
}

func TestUnassign_OnlyArchivesTheLiveVersionRead(t *testing.T) {
	t.Parallel()

	entity := &shipment.Assignment{
		ID:      pulid.MustNew("asn_"),
		Version: 3,
	}
	sql, args := keeltest.Render(t, unassignable(entity))

	assert.Regexp(t, `"archived_at" IS NULL`, sql)
	assert.Regexp(t, `"version" = \$\d+`, sql)
	assert.Contains(t, args, entity.ID.String())
	assert.Contains(t, args, int64(3))
}
