package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap/zaptest"
)

const rlsTestScopeKeyID = "ktest"

func NewRLSConnection(t *testing.T, admin *bun.DB, dsn string) *postgres.Connection {
	t.Helper()

	ctx := t.Context()
	suffix := strings.ToLower(pulid.MustNew("r").String())
	appRole := "rls_app_" + suffix
	systemRole := "rls_sys_" + suffix
	password := base64.RawURLEncoding.EncodeToString(randomBytes(t, 24))

	_, err := postgres.ProvisionRLSRoles(ctx, admin, postgres.ProvisionRLSRolesParams{
		App:    postgres.RLSLoginRole{User: appRole, Password: password},
		System: postgres.RLSLoginRole{User: systemRole, Password: password},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		dropCtx := context.WithoutCancel(ctx)
		for _, role := range []string{appRole, systemRole} {
			_, _ = admin.ExecContext(dropCtx, fmt.Sprintf("DROP ROLE IF EXISTS %q", role))
		}
	})

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	port, err := strconv.Atoi(parsed.Port())
	require.NoError(t, err)

	cfg := &config.Config{
		App: config.AppConfig{Name: "trenova-rls-test"},
		Database: config.DatabaseConfig{
			Host:         parsed.Hostname(),
			Port:         port,
			Name:         strings.TrimPrefix(parsed.Path, "/"),
			User:         appRole,
			Password:     password,
			SSLMode:      "disable",
			MaxOpenConns: 16,
			RLS: config.RLSConfig{
				Mode:       "enforce",
				ScopeKeyID: rlsTestScopeKeyID,
				ScopeKey:   base64.StdEncoding.EncodeToString(randomBytes(t, 48)),
			},
			System: config.DatabaseRole{User: systemRole, Password: password},
		},
	}

	_, err = postgres.ProvisionRLS(ctx, admin, cfg)
	require.NoError(t, err)

	lifecycle := fxtest.NewLifecycle(t)
	conn, err := postgres.NewConnection(postgres.ConnectionParams{
		Lifecycle: lifecycle,
		Config:    cfg,
		Logger:    zaptest.NewLogger(t),
	})
	require.NoError(t, err)
	lifecycle.RequireStart()
	t.Cleanup(lifecycle.RequireStop)

	return conn
}

func randomBytes(t *testing.T, size int) []byte {
	t.Helper()

	raw := make([]byte, size)
	_, err := rand.Read(raw)
	require.NoError(t, err)

	return raw
}
