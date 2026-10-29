package seedtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/systemconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/keeltables"
	"github.com/emoss08/trenova/internal/seeds/reference"
	"github.com/emoss08/trenova/internal/seeds/seedenv"
)

const (
	SystemUserPassword = "test-system-password"

	keyLength = 16
)

type Template struct {
	Name  string
	Seeds *seed.Registry
}

var referenceTemplate = Template{Name: "reference", Seeds: reference.Seeds}

func testConfig() *config.Config {
	return &config.Config{System: systemconfig.Config{SystemUserPassword: SystemUserPassword}}
}

func seedOptions() seed.Options {
	env := seedenv.Env{Config: testConfig}

	return seed.Options{Environment: seed.Test, Tenants: env.Tenants, System: env.System}
}

func seedKey(ctx context.Context, adminDSN string, kind Template) (string, error) {
	db, err := keel.Open(ctx, keel.Config{DSN: adminDSN, SkipEnums: true})
	if err != nil {
		return "", fmt.Errorf("open the admin database to fingerprint seeds: %w", err)
	}
	defer db.Close()

	plan, err := kind.Seeds.Plan(ctx, db, seedOptions())
	if err != nil {
		return "", fmt.Errorf("fingerprint the %s seeds: %w", kind.Name, err)
	}

	h := sha256.New()
	for _, item := range plan.Seeds {
		_, _ = fmt.Fprintf(h, "%s\x00%s\x00", item.Seed.Name, item.Fingerprint)
	}

	return TemplateKey + ":" + kind.Name + ":" + hex.EncodeToString(h.Sum(nil))[:keyLength], nil
}

func applySeeds(kind Template) func(ctx context.Context, dsn string) error {
	return func(ctx context.Context, dsn string) error {
		cfg := dbpool.KeelConfig(nil, nil)
		cfg.DSN = dsn
		cfg.Tables = keeltables.All()
		db, err := keel.Open(ctx, cfg)
		if err != nil {
			return fmt.Errorf("open the template database: %w", err)
		}
		defer db.Close()

		if _, err = kind.Seeds.Apply(ctx, db, seedOptions()); err != nil {
			return fmt.Errorf("apply the %s seeds to the template: %w", kind.Name, err)
		}

		return nil
	}
}
