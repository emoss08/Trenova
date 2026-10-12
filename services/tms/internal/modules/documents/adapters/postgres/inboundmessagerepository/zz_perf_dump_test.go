package inboundmessagerepository

import (
	"os"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestZZDumpPerfSQL(t *testing.T) {
	out := os.Getenv("PERF_SQL_OUT")
	if out == "" {
		t.Skip()
	}
	tenant := pagination.TenantInfo{
		OrgID: "org_01M4QZRW7QZX0E7CHB0AC04KZ2",
		BuID:  "bu_01M4QZRW7HZ6MA6202G3B3T35E",
	}
	viewer := pulid.ID("usr_01M4QZRWCDSQC1YKG5JX0FE2GP")
	db := renderDB()
	var b strings.Builder
	emit := func(name, sql string) {
		b.WriteString("\\echo ==== " + name + "\nEXPLAIN (ANALYZE, BUFFERS) " + sql + ";\n")
	}
	due := []domaintypes.SortField{
		{Field: "dueAt", Direction: dbtype.SortDirectionAsc},
		{Field: "receivedAt", Direction: dbtype.SortDirectionDesc},
	}
	newest := []domaintypes.SortField{{Field: "receivedAt", Direction: dbtype.SortDirectionDesc}}
	list := func(name string, req inboundmessageports.ListInboundMessagesRequest, sort []domaintypes.SortField, after []any) {
		req.Filter.TenantInfo = tenant
		req.Filter.Sort = sort
		req.ViewerID = viewer
		cursor := pagination.CursorInfo{Limit: 50}
		if after != nil {
			probe := &pagination.QueryOptions{TenantInfo: tenant, Sort: sort}
			entities := make([]*inboundmessage.InboundMessage, 0)
			_, err := querybuilder.ApplyCursorFilters(db.NewSelect().Model(&entities), "imsg", probe,
				pagination.CursorInfo{}, (*inboundmessage.InboundMessage)(nil))
			require.NoError(t, err)
			cursor.After = "x"
			cursor.Cursor = pagination.Cursor{Sort: probe.CursorSort, Values: after}
		}
		entities := make([]*inboundmessage.InboundMessage, 0)
		q, err := querybuilder.ApplyCursorFilters(db.NewSelect().Model(&entities).Relation("Mailbox"),
			"imsg", req.Filter, cursor, (*inboundmessage.InboundMessage)(nil))
		require.NoError(t, err)
		q = narrow(db, q, &req).Limit(51)
		emit(name, q.String())

		count := db.NewSelect().Model((*inboundmessage.InboundMessage)(nil)).ColumnExpr("count(*)")
		count = narrow(db, querybuilder.ApplyFiltersWithoutSort(count, "imsg", req.Filter,
			(*inboundmessage.InboundMessage)(nil)), &req)
		emit(name+" (total count)", count.String())
	}

	list("waiting by due, page 1", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, Lane: inboundmessageports.LaneWaiting,
	}, due, nil)
	list("waiting by due, page 2", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, Lane: inboundmessageports.LaneWaiting,
	}, due, []any{os.Getenv("PERF_DUE"), os.Getenv("PERF_RECEIVED"), os.Getenv("PERF_ID")})
	list("waiting by due, threads collapsed", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, Lane: inboundmessageports.LaneWaiting, CollapseThreads: true,
	}, due, nil)
	list("everything, newest", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, Lane: inboundmessageports.LaneEverything,
	}, newest, nil)
	list("everything, newest, threads collapsed", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, Lane: inboundmessageports.LaneEverything, CollapseThreads: true,
	}, newest, nil)
	list("handled, newest", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, Lane: inboundmessageports.LaneHandled,
	}, newest, nil)
	list("assigned to me, due", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, Lane: inboundmessageports.LaneAssignedToMe,
	}, due, nil)
	list("snoozed, newest", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, Lane: inboundmessageports.LaneSnoozed,
	}, newest, nil)
	list("kind folder (Invoice), waiting by due", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, Lane: inboundmessageports.LaneWaiting,
		Classification: inboundmessage.ClassificationInvoice,
	}, due, nil)
	list("mailbox folder (Billing), everything newest", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, Lane: inboundmessageports.LaneEverything,
		MailboxID: "imbx_01M4QZX1YZH87CCRWQ75PTX4RK",
	}, newest, nil)
	list("search 'invoice', everything newest", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{Query: "invoice"}, Lane: inboundmessageports.LaneEverything,
	}, newest, nil)
	list("one thread", inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{}, ThreadKey: "synthetic-24999@d99.example",
	}, newest, nil)

	emit("census (CountBreakdown)", countBreakdownQuery(db, inboundmessageports.CountInboundMessagesRequest{
		TenantInfo: tenant, ViewerID: viewer, Now: 1_791_900_000,
	}).String())

	keys := make([]string, 0, 50)
	ids := make([]pulid.ID, 0, 50)
	for i := 1; i <= 150; i += 3 {
		n := strings.Repeat("0", 16-len(itoa(i))) + itoa(i)
		keys = append(keys, "synthetic-"+itoa(i-i%3)+"@d"+itoa((i-i%3)%300)+".example")
		ids = append(ids, pulid.ID("imsg_synth"+n))
	}
	threadCounts := db.NewSelect().Model((*inboundmessage.InboundMessage)(nil)).
		ColumnExpr("imsg.thread_key").ColumnExpr("count(*)").
		Where("imsg.organization_id = ?", tenant.OrgID).Where("imsg.business_unit_id = ?", tenant.BuID).
		Where("imsg.thread_key IN (?)", bun.List(keys)).GroupExpr("imsg.thread_key")
	emit("thread counts for a page (50 keys)", threadCounts.String())
	threadEntities := make([]*inboundmessage.InboundMessage, 0)
	emit("thread reader (ListThread)", listThreadQuery(db, inboundmessageports.ListThreadRequest{
		TenantInfo: tenant, ThreadKey: "synthetic-24999@d99.example", Limit: 200,
	}, &threadEntities).String())
	acts := make([]*triage.Activity, 0)
	emit("activity for a page (50 messages)", listActivitiesQuery(db, inboundmessageports.ListActivitiesRequest{
		TenantInfo: tenant, MessageIDs: ids,
	}, &acts).String())
	emit("saved view waiting counts", waitingCountsQuery(db, tenant, []*triage.View{
		{ID: "tview_1", Query: "acme"}, {ID: "tview_2", Query: "SEED-"},
	}).String())
	emit("reconcile overdue parked", overdueParkedQuery(db, inboundmessageports.ListOverdueParkedRequest{
		Now: 1_791_900_000, Limit: 500,
	}).String())

	require.NoError(t, os.WriteFile(out, []byte(b.String()), 0o600))
}

func itoa(i int) string {
	return strings.TrimSpace(strings.Repeat(" ", 0) + func() string {
		digits := ""
		if i == 0 {
			return "0"
		}
		for i > 0 {
			digits = string(rune('0'+i%10)) + digits
			i /= 10
		}
		return digits
	}())
}
