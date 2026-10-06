package tenantbootstrap

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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

func CreateTCAAllowlist(ctx context.Context, db bun.IDB, scope Scope) (int, error) {
	cols := buncolgen.TCAAllowlistedTableColumns
	exists, err := db.NewSelect().
		Model((*tablechangealert.TCAAllowlistedTable)(nil)).
		Where(cols.OrganizationID.Eq(), scope.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), scope.BusinessUnitID).
		Exists(ctx)
	if err != nil {
		return 0, fmt.Errorf("check existing allowlisted tables: %w", err)
	}
	if exists {
		return 0, nil
	}

	now := scope.now()
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

	if _, err = db.NewInsert().Model(&entities).Exec(ctx); err != nil {
		return 0, fmt.Errorf("insert allowlisted tables: %w", err)
	}
	for _, entity := range entities {
		if err = scope.record(ctx, "tca_allowlisted_tables", entity.ID); err != nil {
			return 0, err
		}
	}

	return len(entities), nil
}
