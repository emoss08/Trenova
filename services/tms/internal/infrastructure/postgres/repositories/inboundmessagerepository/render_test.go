package inboundmessagerepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func renderDB() *bun.DB {
	return bun.NewDB(new(sql.DB), pgdialect.New())
}

/*
The service sorts the inbox by arrival. The cursor sort resolves a field name
through the entity's field configuration, and a name it cannot resolve is an
error on every page — so the field the service names must be one the
configuration knows, and it must reach the ORDER BY as the arrival column.
*/
func TestTheArrivalSortResolvesToTheReceivedAtColumn(t *testing.T) {
	t.Parallel()

	filter := &pagination.QueryOptions{
		Sort: []domaintypes.SortField{{Field: "receivedAt", Direction: dbtype.SortDirectionDesc}},
	}
	entities := make([]*inboundmessage.InboundMessage, 0)

	q, err := querybuilder.ApplyCursorFilters(
		renderDB().NewSelect().Model(&entities),
		"imsg",
		filter,
		pagination.CursorInfo{},
		(*inboundmessage.InboundMessage)(nil),
	)
	require.NoError(t, err)

	sqlText := q.String()
	assert.Contains(t, sqlText, `ORDER BY "imsg"."received_at" DESC`, sqlText)
}

func TestTheCensusGroupsByStatusReadingAndMailbox(t *testing.T) {
	t.Parallel()

	sqlText := countBreakdownQuery(renderDB(), repositories.CountInboundMessagesRequest{}).String()

	assert.Contains(t, sqlText, "COALESCE(imsg.classification, '') AS classification", sqlText)
	assert.Contains(t, sqlText,
		"GROUP BY imsg.status, imsg.classification, imsg.mailbox_id", sqlText)
}

// Retention reads only what somebody has finished with, oldest first, inside
// one tenant: a message still waiting on a person is never old enough to go.
func TestRetentionReadsOnlySettledMailInsideOneTenant(t *testing.T) {
	t.Parallel()

	req := repositories.ListSettledInboundMessagesRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		Before:     1_700_000_000,
		Limit:      500,
	}
	entities := make([]*inboundmessage.InboundMessage, 0)
	sqlText := settledBeforeQuery(renderDB(), req, &entities).String()

	assert.Contains(t, sqlText, `imsg.status IN ('Actioned', 'Ignored')`, sqlText)
	assert.Contains(t, sqlText, "imsg.received_at < 1700000000", sqlText)
	assert.Contains(t, sqlText, "'org_1'", sqlText)
	assert.Contains(t, sqlText, "'bu_1'", sqlText)
	assert.Contains(t, sqlText, `ORDER BY "imsg"."received_at" ASC`, sqlText)
	assert.Contains(t, sqlText, "LIMIT 500", sqlText)
	assert.NotContains(t, sqlText, "Quarantined", sqlText)
}

func TestRetentionDeletesOnlyInsideOneTenant(t *testing.T) {
	t.Parallel()

	sqlText := deleteByIDsQuery(renderDB(), repositories.DeleteInboundMessagesRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		IDs:        []pulid.ID{"imsg_1", "imsg_2"},
	}).String()

	assert.Contains(t, sqlText, "imsg.id IN ('imsg_1', 'imsg_2')", sqlText)
	assert.Contains(t, sqlText, "imsg.organization_id = 'org_1'", sqlText)
	assert.Contains(t, sqlText, "imsg.business_unit_id = 'bu_1'", sqlText)
}
