package dbcli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
)

func scopeKeySeed(rls func() databaseconfig.RLSConfig) seed.Seed {
	return seed.Seed{
		Name:        "rls_scope_key",
		Kind:        seed.Reference,
		Description: "Installs the configured row-level security scope signing key",
		Fingerprint: func() string { return scopeKeyFingerprint(rls()) },
		Run: func(ctx context.Context, r *seed.Run) error {
			installed, err := postgres.InstallConfiguredRLSScopeKey(ctx, r.DB, rls())
			if installed {
				r.Count(1)
			}
			return err
		},
	}
}

func scopeKeyFingerprint(rls databaseconfig.RLSConfig) string {
	if !rls.Enabled() {
		return "disabled"
	}

	sum := sha256.Sum256([]byte(strings.TrimSpace(rls.ScopeKeyID) + "\x00" + rls.ScopeKey))
	return hex.EncodeToString(sum[:])
}
