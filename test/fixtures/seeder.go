package fixtures

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
)

func LoadFixtures(ctx context.Context, fixture *dbfixture.Fixture, db *bun.DB) error {
	if _, err := LoadAdminAccount(ctx, db, fixture); err != nil {
		return err
	}

	// Load permissions and roles
	if err := LoadPermissions(ctx, db, fixture); err != nil {
		return err
	}
	// Load fake accounts
	if err := LoadFakeAccounts(ctx, db, fixture); err != nil {
		return err
	}

	// Load resource definitions
	if err := LoadResourceDefinition(ctx, db); err != nil {
		return err
	}

	return nil
}
