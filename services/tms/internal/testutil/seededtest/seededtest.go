

package seededtest

import (
	"context"
	"testing"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/seeds/fixtures"
	"github.com/emoss08/trenova/internal/seeds/reference"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/uptrace/bun"
)

var template = seedtest.Template{Name: "fixtures", Seeds: seed.Merge(reference.Seeds, fixtures.Seeds)}

func SetupTestDB(t *testing.T) (context.Context, *bun.DB, func()) {
	t.Helper()

	ctx, db, _, cleanup := SetupTestDBWithDSN(t)

	return ctx, db, cleanup
}

func SetupTestDBWithDSN(t *testing.T) (context.Context, *bun.DB, string, func()) {
	t.Helper()

	return seedtest.SetupFromTemplate(t, template)
}
