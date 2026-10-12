package inboundmessagerepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
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

	sqlText := countBreakdownQuery(renderDB(), inboundmessageports.CountInboundMessagesRequest{}).String()

	assert.Contains(t, sqlText, "COALESCE(imsg.classification, '') AS classification", sqlText)
	assert.Contains(t, sqlText,
		"GROUP BY imsg.status, COALESCE(imsg.classification, ''), imsg.mailbox_id", sqlText)
}

// Retention reads only what somebody has finished with, oldest first, inside
// one tenant: a message still waiting on a person is never old enough to go.
func TestRetentionReadsOnlySettledMailInsideOneTenant(t *testing.T) {
	t.Parallel()

	req := inboundmessageports.ListSettledInboundMessagesRequest{
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

	sqlText := deleteByIDsQuery(renderDB(), inboundmessageports.DeleteInboundMessagesRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		IDs:        []pulid.ID{"imsg_1", "imsg_2"},
	}).String()

	assert.Contains(t, sqlText, "imsg.id IN ('imsg_1', 'imsg_2')", sqlText)
	assert.Contains(t, sqlText, "imsg.organization_id = 'org_1'", sqlText)
	assert.Contains(t, sqlText, "imsg.business_unit_id = 'bu_1'", sqlText)
}

func renderFolder(req *inboundmessageports.ListInboundMessagesRequest) string {
	entities := make([]*inboundmessage.InboundMessage, 0)
	db := renderDB()

	return narrow(db, db.NewSelect().Model(&entities), req).String()
}

func TestTheLanesAskTheQuestionTheirFolderNames(t *testing.T) {
	t.Parallel()

	viewer := pulid.ID("usr_01J00000000000000000000VW")
	cases := map[inboundmessageports.Lane][]string{
		inboundmessageports.LaneWaiting: {
			"imsg.status IN ('InReview', 'Quarantined')", "imsg.parked IS NULL",
		},
		inboundmessageports.LaneAssignedToMe: {
			"imsg.status IN ('InReview', 'Quarantined')", "imsg.parked IS NULL",
			"imsg.owner_id = 'usr_01J00000000000000000000VW'",
		},
		inboundmessageports.LaneWaitingOnThem: {"imsg.parked = 'AwaitingReply'"},
		inboundmessageports.LaneSnoozed:       {"imsg.parked = 'Snoozed'"},
		inboundmessageports.LaneHandled:       {"imsg.status = 'Actioned'"},
		inboundmessageports.LaneIgnored:       {"imsg.status = 'Ignored'"},
	}
	for lane, fragments := range cases {
		sqlText := renderFolder(&inboundmessageports.ListInboundMessagesRequest{
			Lane:     lane,
			ViewerID: viewer,
		})
		for _, fragment := range fragments {
			assert.Contains(t, sqlText, fragment, "%s: %s", lane, sqlText)
		}
	}

	everything := renderFolder(&inboundmessageports.ListInboundMessagesRequest{
		Lane: inboundmessageports.LaneEverything,
	})
	assert.NotContains(t, everything, "WHERE", everything)
}

func TestCollapsingKeepsTheNewestOfEachThreadInTheFolder(t *testing.T) {
	t.Parallel()

	sqlText := renderFolder(&inboundmessageports.ListInboundMessagesRequest{
		Filter:          &pagination.QueryOptions{Query: "detention"},
		Lane:            inboundmessageports.LaneWaiting,
		Classification:  inboundmessage.ClassificationDetentionDispute,
		CollapseThreads: true,
	})

	assert.Contains(t, sqlText, "NOT EXISTS (SELECT 1 FROM inbound_messages AS nwr", sqlText)
	assert.Contains(t, sqlText, "nwr.thread_key = imsg.thread_key", sqlText)
	assert.Contains(t, sqlText,
		"(nwr.received_at > imsg.received_at OR (nwr.received_at = imsg.received_at AND nwr.id > imsg.id))",
		sqlText)
	assert.Contains(t, sqlText, "nwr.status IN ('InReview', 'Quarantined')", sqlText)
	assert.Contains(t, sqlText, "nwr.parked IS NULL", sqlText)
	assert.Contains(t, sqlText, "nwr.classification = 'DetentionDispute'", sqlText)
	assert.Contains(t, sqlText, "nwr.subject ILIKE '%detention%'", sqlText,
		"the newer message has to be in the same folder, search included")

	thread := renderFolder(&inboundmessageports.ListInboundMessagesRequest{
		ThreadKey:       "abc@acme.example",
		CollapseThreads: true,
	})
	assert.Contains(t, thread, "imsg.thread_key = 'abc@acme.example'", thread)
	assert.NotContains(t, thread, "NOT EXISTS", "one thread is never collapsed")
}

func TestTheDueSortOrdersOverdueFirstAndUndatedLast(t *testing.T) {
	t.Parallel()

	filter := &pagination.QueryOptions{
		Sort: []domaintypes.SortField{
			{Field: "dueAt", Direction: dbtype.SortDirectionAsc},
			{Field: "receivedAt", Direction: dbtype.SortDirectionDesc},
		},
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
	assert.Contains(t, sqlText,
		`ORDER BY "imsg"."due_at" ASC NULLS LAST, "imsg"."received_at" DESC, "imsg"."id" DESC`,
		sqlText)
}

func TestTheCensusAlsoGroupsByParkingOwnerAndLateness(t *testing.T) {
	t.Parallel()

	sqlText := countBreakdownQuery(renderDB(), inboundmessageports.CountInboundMessagesRequest{
		ViewerID: "usr_01J00000000000000000000VW",
		Now:      1_700_000_000,
	}).String()

	assert.Contains(t, sqlText, "COALESCE(imsg.parked, '') AS parked", sqlText)
	assert.Contains(t, sqlText,
		"COALESCE(imsg.owner_id = 'usr_01J00000000000000000000VW', FALSE) AS mine", sqlText)
	assert.Contains(t, sqlText, "COALESCE(imsg.due_at < 1700000000, FALSE) AS overdue", sqlText)
	assert.Contains(t, sqlText, "GROUP BY imsg.status, COALESCE(imsg.classification, ''), imsg.mailbox_id, "+
		"COALESCE(imsg.parked, ''), COALESCE(imsg.owner_id = 'usr_01J00000000000000000000VW', FALSE), "+
		"COALESCE(imsg.due_at < 1700000000, FALSE)", sqlText)
}

func renderUpdate(patch inboundmessageports.QueuePatch) string {
	return updateQueueStateQuery(renderDB(), inboundmessageports.UpdateQueueStateRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		IDs:        []pulid.ID{"imsg_1", "imsg_2"},
		Patch:      patch,
		Now:        1_700_000_000,
	}).String()
}

func TestAQueueChangeIsOneTenantScopedStatement(t *testing.T) {
	t.Parallel()

	settle := renderUpdate(inboundmessageports.QueuePatch{
		Status: inboundmessage.StatusActioned,
		Review: &inboundmessageports.ReviewPatch{ReviewerID: "usr_1", Note: "done"},
		Unpark: true,
		Due:    inboundmessageports.DueClear,
	})
	for _, fragment := range []string{
		"imsg.organization_id = 'org_1'",
		"imsg.business_unit_id = 'bu_1'",
		"imsg.id IN ('imsg_1', 'imsg_2')",
		"status = 'Actioned'",
		"reviewed_by = 'usr_1'",
		"review_note = COALESCE(NULLIF('done', ''), review_note)",
		"parked = NULL",
		"due_at = NULL",
		"version = version + 1",
		"RETURNING id, organization_id",
	} {
		assert.Contains(t, settle, fragment, settle)
	}
	assert.NotContains(t, settle, "text_body", "a bulk change never ships bodies back")

	snooze := renderUpdate(inboundmessageports.QueuePatch{Park: &inboundmessageports.ParkPatch{
		Parking:       inboundmessage.ParkingSnoozed,
		SnoozeEvent:   inboundmessage.SnoozeEventShipmentDelivered,
		WatchShipment: true,
		By:            "usr_1",
	}})
	assert.Contains(t, snooze, "parked = 'Snoozed'", snooze)
	assert.Contains(t, snooze, "snooze_event = 'ShipmentDelivered'", snooze)
	assert.Contains(t, snooze, "snooze_watch_id = matched_shipment_id", snooze)
	assert.Contains(t, snooze, "snoozed_until = NULL", snooze)

	reopen := renderUpdate(inboundmessageports.QueuePatch{
		Due:     inboundmessageports.DueFromTargets,
		Targets: triage.ResolveTargets(nil),
	})
	assert.Contains(t, reopen,
		"due_at = received_at + 60 * CASE COALESCE(classification, 'Other') WHEN 'Tender' THEN 120::bigint",
		reopen)

	handOff := renderUpdate(inboundmessageports.QueuePatch{
		HandOff: &inboundmessageports.HandOffPatch{To: "usr_2"},
	})
	assert.Contains(t, handOff,
		"action_state = action_state || jsonb_build_object('handedOffTo', 'usr_2'::text, 'handedOffAt', 1700000000::bigint)",
		handOff)
}

func TestAnEventWakesOnlyTheMessagesWatchingIt(t *testing.T) {
	t.Parallel()

	sqlText := wakeByEventQuery(renderDB(), inboundmessageports.WakeByEventRequest{
		TenantInfo:     pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		Event:          inboundmessage.SnoozeEventDocumentAttached,
		WatchID:        "shp_1",
		DocumentTypeID: "dt_1",
		Now:            1_700_000_000,
	}).String()

	for _, fragment := range []string{
		"imsg.organization_id = 'org_1'",
		"imsg.snooze_event = 'DocumentAttached'",
		"imsg.snooze_watch_id = 'shp_1'",
		"imsg.parked = 'Snoozed'",
		"imsg.snooze_document_type_id = 'dt_1'",
		"parked = NULL",
		"RETURNING",
	} {
		assert.Contains(t, sqlText, fragment, sqlText)
	}
}

func TestTheReconcileReadsBothTimersAcrossTenants(t *testing.T) {
	t.Parallel()

	sqlText := overdueParkedQuery(renderDB(), inboundmessageports.ListOverdueParkedRequest{
		Now:   1_700_000_000,
		Limit: 500,
	}).String()

	assert.Contains(t, sqlText, "imsg.parked = 'Snoozed'", sqlText)
	assert.Contains(t, sqlText, "imsg.snoozed_until <= 1700000000", sqlText)
	assert.Contains(t, sqlText, "imsg.parked = 'AwaitingReply'", sqlText)
	assert.Contains(t, sqlText, "imsg.awaiting_until <= 1700000000", sqlText)
	assert.Contains(t, sqlText, "UNION ALL", sqlText)
	assert.NotContains(t, sqlText, "organization_id =", "the sweep spans every tenant")
}

func TestATargetChangeRecomputesOneKindInOneStatement(t *testing.T) {
	t.Parallel()

	other := recomputeDueQuery(renderDB(), inboundmessageports.RecomputeDueRequest{
		TenantInfo:     pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		Classification: inboundmessage.ClassificationOther,
		Minutes:        90,
	}).String()
	assert.Contains(t, other, "due_at = received_at + 5400::bigint", other)
	assert.Contains(t, other, "imsg.status IN ('InReview', 'Quarantined')", other)
	assert.Contains(t, other, "((imsg.classification = 'Other') OR (imsg.classification IS NULL))", other,
		"an unread message waits on the Other target")
}

func TestTheThreadReadsItsNewestMessagesOldestFirst(t *testing.T) {
	t.Parallel()

	entities := make([]*inboundmessage.InboundMessage, 0)
	sqlText := listThreadQuery(renderDB(), inboundmessageports.ListThreadRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		ThreadKey:  "abc@acme.example",
		Limit:      200,
	}, &entities).String()

	assert.Contains(t, sqlText, "imsg.thread_key = 'abc@acme.example'", sqlText)
	assert.Contains(t, sqlText, "LIMIT 200", sqlText)
	assert.Contains(t, sqlText, `ORDER BY "imsg"."received_at" ASC`, sqlText)
}

func TestMarkingAMailboxReceivedNeverMovesItBackwards(t *testing.T) {
	t.Parallel()

	sqlText := markReceivedQuery(renderDB(), inboundmessageports.MailboxReceivedRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		MailboxID:  "imbx_1",
		ReceivedAt: 1_700_000_000,
	}).String()

	assert.Contains(t, sqlText,
		"last_received_at = GREATEST(COALESCE(last_received_at, 0), 1700000000::bigint)", sqlText)
	assert.Contains(t, sqlText, "imbx.id = 'imbx_1'", sqlText)
}
