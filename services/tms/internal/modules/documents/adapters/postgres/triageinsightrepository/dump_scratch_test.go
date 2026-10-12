package triageinsightrepository

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestDumpScratch(t *testing.T) {
	out := os.Getenv("DUMP_SQL")
	if out == "" {
		t.Skip()
	}
	db := bun.NewDB(new(sql.DB), pgdialect.New())
	tenant := pagination.TenantInfo{OrgID: "org_perf", BuID: "bu_perf"}
	win := triageports.BriefingWindow{TenantInfo: tenant, DayStart: 1760245200, Now: 1760290000, Limit: 200}
	var msgs []*inboundmessage.InboundMessage
	queries := []string{
		briefingCountsQuery(db, win).String(),
		handledQuery(db, win).String(),
		arrivalsQuery(db, win).String(),
		riverQuery(db, win).String(),
		waitingQuery(db, triageports.WaitingRequest{TenantInfo: tenant, Limit: 1000}, &msgs).String(),
		correctionTotalsQuery(db, triageports.CorrectionTotalsRequest{TenantInfo: tenant, LastWeekStart: 1759100000, WeekStart: 1759700000}).String(),
		ruleCandidatesQuery(db, triageports.RuleCandidatesRequest{TenantInfo: tenant, Classification: "ProofOfDelivery", SenderScope: triage.SenderDomain, Sender: "drivers.example", Since: 1757700000, Limit: 5000}).String(),
		ruleCandidatesCountQuery(db, triageports.RuleCandidatesRequest{TenantInfo: tenant, Classification: "Other", SenderScope: triage.SenderAnyone, Since: 1757700000, Limit: 5000}).ColumnExpr("count(*)").String(),
		manualOutcomesQuery(db, triageports.ManualOutcomesRequest{TenantInfo: tenant, Since: 1757700000}).String(),
		senderDomainsQuery(db, triageports.SenderDomainsRequest{TenantInfo: tenant, Query: "dri", Since: 1752500000, Limit: 20}).String(),
		senderKindsQuery(db, triageports.SenderStatsRequest{TenantInfo: tenant, SenderDomain: "drivers.example", Since: 1752500000, RecentLimit: 3}).String(),
		senderSummaryQuery(db, triageports.SenderStatsRequest{TenantInfo: tenant, SenderDomain: "drivers.example", Since: 1752500000, RecentLimit: 3}).String(),
		senderRecentQuery(db, triageports.SenderStatsRequest{TenantInfo: tenant, SenderDomain: "drivers.example", Since: 1752500000, RecentLimit: 3}).String(),
	}
	if err := os.WriteFile(out, []byte(strings.Join(queries, ";\n\n")+";\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
