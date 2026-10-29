package referenceseed

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert/tablechangealertdb"
	"github.com/emoss08/trenova/shared/pulid"
)

type tcaAllowlistEntry struct {
	tableName   string
	displayName string
}

var defaultTCAAllowlist = []tcaAllowlistEntry{
	{"shipments", "Shipments"},
	{"customers", "Customers"},
	{"carriers", "Carriers"},
	{"carrier_contacts", "Carrier Contacts"},
	{"carrier_insurance_policies", "Carrier Insurance Policies"},
	{"carrier_settlements", "Carrier Settlements"},
	{"carrier_cost_events", "Carrier Cost Events"},
	{"carrier_ledger_entries", "Carrier Ledger Entries"},
	{"carrier_assignments", "Carrier Assignments"},
	{"rate_confirmations", "Rate Confirmations"},
	{"carrier_invoice_matches", "Carrier Invoice Matches"},
	{"tenders", "Tenders"},
	{"tender_offers", "Tender Offers"},
	{"routing_guides", "Routing Guides"},
	{"workers", "Workers"},
	{"tractors", "Tractors"},
	{"trailers", "Trailers"},
}

func CreateTCAAllowlist(ctx context.Context, db *keel.DB, scope tenantscope.Scope) (int, error) {
	t := tablechangealertdb.TCAAllowlistedTables
	exists, err := tenantscope.Scoped(ctx, t.Where(
		t.OrganizationID.Eq(scope.OrganizationID),
		t.BusinessUnitID.Eq(scope.BusinessUnitID),
	)).Exists(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("check existing allowlisted tables: %w", err)
	}
	if exists {
		return 0, nil
	}

	now := scope.NowUnix()
	entities := make([]*tablechangealert.TCAAllowlistedTable, 0, len(defaultTCAAllowlist))
	for _, entry := range defaultTCAAllowlist {
		entities = append(entities, &tablechangealert.TCAAllowlistedTable{
			ID:             pulid.MustNew("tcaw_"),
			OrganizationID: scope.OrganizationID,
			BusinessUnitID: scope.BusinessUnitID,
			TableName:      entry.tableName,
			DisplayName:    entry.displayName,
			Enabled:        true,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	if err = t.CreateMany(ctx, db, entities); err != nil {
		return 0, fmt.Errorf("insert allowlisted tables: %w", err)
	}

	return len(entities), nil
}
