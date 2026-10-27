package inboundmessagerepository

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func renderFolder(
	t *testing.T,
	req *inboundmessageports.ListInboundMessagesRequest,
) (string, []any) {
	t.Helper()

	m := inboundmessagedb.InboundMessages

	return keeltest.Render(t, m.Where(narrow(req)...))
}

func TestTheFolderSortsByArrivalOrSoonestDue(t *testing.T) {
	t.Parallel()

	arrival, _ := keeltest.Render(t, selectIDsQuery(inboundmessageports.SelectInboundMessageIDsRequest{
		Folder: &inboundmessageports.ListInboundMessagesRequest{},
		Limit:  50,
	}))
	assert.Contains(t, arrival, `ORDER BY im."received_at" DESC, im."id" DESC LIMIT 50`, arrival)

	due, _ := keeltest.Render(t, selectIDsQuery(inboundmessageports.SelectInboundMessageIDsRequest{
		Folder: &inboundmessageports.ListInboundMessagesRequest{Sort: inboundmessageports.SortDue},
		Limit:  50,
	}))
	assert.Contains(
		t,
		due,
		`ORDER BY im."due_at" ASC NULLS LAST, im."received_at" DESC, im."id" DESC LIMIT 50`,
		due,
	)
}

func TestTheCensusGroupsOpenMailByParkingOwnerAndLateness(t *testing.T) {
	t.Parallel()

	open, _ := censusStatuses(nil)
	sqlText, args := keeltest.Render(t, openCensusQuery(inboundmessageports.CountInboundMessagesRequest{
		ViewerID: "usr_01J00000000000000000000VW",
		Now:      1_700_000_000,
	}, open))

	assert.Equal(
		t,
		`SELECT im."status", coalesce(im."classification", $3), im."mailbox_id", `+
			`coalesce(im."parked", $4), coalesce(im."owner_id" = $5::text, $6), `+
			`coalesce(im."due_at" < $7, $8), count(*) FROM "inbound_messages" im `+
			`WHERE im."business_unit_id" = $1 AND im."organization_id" = $2 `+
			`AND im."status" IN ('Received', 'Processing', 'Classified', 'InReview', 'Quarantined') `+
			`GROUP BY im."status", 2, im."mailbox_id", 4, 5, 6`,
		sqlText,
	)
	assert.Equal(t, []any{
		"bu_1",
		"org_1",
		"",
		"",
		"usr_01J00000000000000000000VW",
		false,
		int64(1_700_000_000),
		false,
	}, args)
}

func TestTheCensusCountsSettledMailWithoutTheOpenLanesWork(t *testing.T) {
	t.Parallel()

	_, settled := censusStatuses(nil)
	sqlText, args := keeltest.Render(
		t,
		settledCensusQuery(inboundmessageports.CountInboundMessagesRequest{}, settled),
	)

	assert.Equal(
		t,
		`SELECT im."status", coalesce(im."classification", $3), im."mailbox_id", $4::text, $5::boolean, $5::boolean, `+
			`count(*) FROM "inbound_messages" im WHERE im."business_unit_id" = $1 AND im."organization_id" = $2 `+
			`AND im."status" IN ('Actioned', 'Ignored') GROUP BY im."status", im."classification", im."mailbox_id"`,
		sqlText,
	)
	assert.Equal(t, []any{"bu_1", "org_1", "", "", false}, args)

	_, bothHalves := countBreakdownQuery(inboundmessageports.CountInboundMessagesRequest{}).(keel.Compound)
	assert.True(t, bothHalves, "open and settled mail are counted by one UNION ALL")

	settledOnly, oneHalf := countBreakdownQuery(inboundmessageports.CountInboundMessagesRequest{
		Statuses: []inboundmessage.Status{inboundmessage.StatusActioned},
	}).(*keel.Query[*inboundmessage.InboundMessage])
	assert.True(t, oneHalf, "only settled statuses need no union")
	if oneHalf {
		sqlText, _ = keeltest.Render(t, settledOnly)
		assert.Contains(t, sqlText, `$4::text, $5::boolean, $5::boolean, count(*)`, sqlText)
		assert.Contains(t, sqlText, `im."status" IN ('Actioned') GROUP BY`, sqlText)
	}
}

func TestRetentionReadsOnlySettledMailInsideOneTenant(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, settledBeforeQuery(inboundmessageports.ListSettledInboundMessagesRequest{
		Before: 1_700_000_000,
		Limit:  500,
	}))

	assert.Equal(
		t,
		`SELECT im."id", im."organization_id", im."business_unit_id", im."html_key" `+
			`FROM "inbound_messages" im WHERE im."business_unit_id" = $1 AND im."organization_id" = $2 `+
			`AND im."status" IN ('Actioned', 'Ignored') AND im."received_at" < $3 ORDER BY im."received_at" ASC LIMIT 500`,
		sqlText,
	)
	assert.Equal(t, []any{"bu_1", "org_1", int64(1_700_000_000)}, args)
}

func TestTheLanesAskTheQuestionTheirFolderNames(t *testing.T) {
	t.Parallel()

	viewer := pulid.ID("usr_01J00000000000000000000VW")
	cases := map[inboundmessageports.Lane]struct {
		where string
		args  []any
	}{
		inboundmessageports.LaneWaiting: {
			where: `(im."status" IN ('InReview', 'Quarantined') AND im."parked" IS NULL)`,
			args:  []any{"bu_1", "org_1"},
		},
		inboundmessageports.LaneAssignedToMe: {
			where: `(im."status" IN ('InReview', 'Quarantined') AND im."parked" IS NULL AND im."owner_id" = $3)`,
			args:  []any{"bu_1", "org_1", string(viewer)},
		},
		inboundmessageports.LaneWaitingOnThem: {
			where: `im."parked" = 'AwaitingReply'`,
			args:  []any{"bu_1", "org_1"},
		},
		inboundmessageports.LaneSnoozed: {
			where: `im."parked" = 'Snoozed'`,
			args:  []any{"bu_1", "org_1"},
		},
		inboundmessageports.LaneHandled: {
			where: `im."status" = 'Actioned'`,
			args:  []any{"bu_1", "org_1"},
		},
		inboundmessageports.LaneIgnored: {
			where: `im."status" = 'Ignored'`,
			args:  []any{"bu_1", "org_1"},
		},
	}
	for lane, want := range cases {
		sqlText, args := renderFolder(t, &inboundmessageports.ListInboundMessagesRequest{
			Lane:     lane,
			ViewerID: viewer,
		})
		assert.Contains(t, sqlText, want.where, "%s: %s", lane, sqlText)
		assert.Equal(t, want.args, args, lane)
	}

	everything, args := renderFolder(t, &inboundmessageports.ListInboundMessagesRequest{
		Lane: inboundmessageports.LaneEverything,
	})
	assert.Contains(
		t,
		everything,
		`WHERE im."business_unit_id" = $1 AND im."organization_id" = $2`,
		everything,
	)
	assert.Equal(t, []any{"bu_1", "org_1"}, args)
}

func TestCollapsingKeepsTheNewestOfEachThreadInTheFolder(t *testing.T) {
	t.Parallel()

	sqlText, args := renderFolder(t, &inboundmessageports.ListInboundMessagesRequest{
		Filter:          &pagination.QueryOptions{Query: "detention"},
		Lane:            inboundmessageports.LaneWaiting,
		Classification:  inboundmessage.ClassificationDetentionDispute,
		CollapseThreads: true,
	})

	assert.Contains(t, sqlText, `NOT EXISTS (SELECT 1 FROM "inbound_messages" nwr`, sqlText)
	assert.Contains(t, sqlText, `nwr."thread_key" = im."thread_key"`, sqlText)
	assert.Contains(
		t,
		sqlText,
		`(ROW(nwr."received_at", nwr."id") > ROW(im."received_at", im."id"))`,
		sqlText,
	)
	assert.Contains(t, sqlText, `(nwr."status" IN ('InReview', 'Quarantined') AND nwr."parked" IS NULL)`, sqlText)
	assert.Contains(t, sqlText, `nwr."classification" = 'DetentionDispute'`, sqlText)
	assert.Contains(t, sqlText, `nwr."subject" ILIKE $3`, sqlText,
		"the newer message has to be in the same folder, search included")
	assert.Equal(t, "%detention%", args[2])

	thread, args := renderFolder(t, &inboundmessageports.ListInboundMessagesRequest{
		ThreadKey:       "abc@acme.example",
		CollapseThreads: true,
	})
	assert.Contains(t, thread, `im."thread_key" = $3`, thread)
	assert.NotContains(t, thread, "NOT EXISTS", "one thread is never collapsed")
	assert.Equal(t, []any{"bu_1", "org_1", "abc@acme.example"}, args)
}

func TestTheThreadReadsItsNewestMessagesOldestFirst(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, listThreadQuery(inboundmessageports.ListThreadRequest{
		ThreadKey: "abc@acme.example",
		Limit:     200,
	}))

	assert.Contains(
		t,
		sqlText,
		`im."id" IN (SELECT newest."id" FROM "inbound_messages" newest `+
			`WHERE newest."business_unit_id" = $1 AND newest."organization_id" = $2 `+
			`AND newest."thread_key" = $3 ORDER BY newest."received_at" DESC, newest."id" DESC LIMIT 200) `+
			`ORDER BY im."received_at" ASC, im."id" ASC LIMIT 200`,
		sqlText,
	)
	assert.Equal(t, []any{"bu_1", "org_1", "abc@acme.example"}, args)
}

func TestMarkingAMailboxReceivedTouchesOnlyThatMailbox(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, markReceivedQuery(inboundmessageports.MailboxReceivedRequest{
		MailboxID:  "imbx_1",
		ReceivedAt: 1_700_000_000,
	}))

	assert.Contains(
		t,
		sqlText,
		`FROM "inbound_mailboxes" im WHERE im."business_unit_id" = $1 `+
			`AND im."organization_id" = $2 AND im."id" = $3`,
		sqlText,
	)
	assert.Equal(t, []any{"bu_1", "org_1", "imbx_1"}, args)
}
