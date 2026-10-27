package seededtest

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/seeds/fixtures"
	"github.com/emoss08/trenova/internal/seeds/reference"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
)

var template = seedtest.Template{Name: "fixtures", Seeds: seed.Merge(reference.Seeds, fixtures.Seeds)}

func SetupTestDB(t *testing.T) (context.Context, *keel.DB, func()) {
	t.Helper()

	ctx, db, _, cleanup := SetupTestDBWithDSN(t)

	return ctx, db, cleanup
}

func SetupTestDBWithDSN(t *testing.T) (context.Context, *keel.DB, string, func()) {
	t.Helper()

	return seedtest.SetupFromTemplate(t, template)
}
