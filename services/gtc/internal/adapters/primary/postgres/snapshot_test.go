package postgres

import (
	"testing"

	"github.com/emoss08/gtc/internal/core/domain"
)

func TestBuildSnapshotQueryCompositeCursor(t *testing.T) {
	t.Parallel()

	query, args := buildSnapshotQuery(
		domain.SnapshotBinding{
			Schema:      "public",
			Table:       "shipments",
			PrimaryKeys: []string{"id", "organization_id", "business_unit_id"},
		},
		domain.Cursor{Values: []any{"shp_1", "org_1", "bu_1"}},
		250,
	)

	expected := `SELECT * FROM "public"."shipments" WHERE ("id" > $1) OR ("id" = $2 AND "organization_id" > $3) OR ("id" = $4 AND "organization_id" = $5 AND "business_unit_id" > $6) ORDER BY "id", "organization_id", "business_unit_id" LIMIT $7`
	if query != expected {
		t.Fatalf("unexpected query:\n%s", query)
	}

	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d", len(args))
	}

	if args[len(args)-1] != 250 {
		t.Fatalf("expected batch size to be last arg, got %v", args[len(args)-1])
	}
}
