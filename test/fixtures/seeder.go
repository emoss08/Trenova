// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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

	if _, err := LoadSystemAccount(ctx, db, fixture); err != nil {
		return err
	}

	if _, err := LoadBasicAccount(ctx, db, fixture); err != nil {
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

	// Load fake workers
	if err := LoadWorkers(ctx, db, fixture); err != nil {
		return err
	}

	// Load resource definitions
	if err := LoadResourceDefinition(ctx, db); err != nil {
		return err
	}

	return nil
}
