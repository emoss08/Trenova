package fixtures

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
)

func LoadFixtures(ctx context.Context, fixture *dbfixture.Fixture, db *bun.DB) error {
	if _, err := LoadAdminAccount(ctx, db, fixture); err != nil {
		return eris.Wrap(err, "load admin account")
	}

	// Load permissions and roles
	if err := LoadPermissions(ctx, db, fixture); err != nil {
		return eris.Wrap(err, "load permissions")
	}

	// Load fake accounts
	if err := LoadFakeAccounts(ctx, db, fixture); err != nil {
		return eris.Wrap(err, "load fake accounts")
	}

	return nil
}
