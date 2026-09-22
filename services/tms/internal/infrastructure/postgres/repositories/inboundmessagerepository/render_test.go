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
