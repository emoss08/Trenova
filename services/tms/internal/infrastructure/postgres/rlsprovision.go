package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/uptrace/bun"
)

var (
	ErrRLSProvisionRoleRequired = errors.New("a login role and password are required")
	ErrRLSProvisionSameRole     = errors.New(
		"the application and system roles must be different database roles",
	)
)

type RLSProvisionResult struct {
	KeyInstalled  bool
	PoliciesAdded int
}

func ProvisionRLS(ctx context.Context, db bun.IDB, cfg *config.Config) (*RLSProvisionResult, error) {
	result := new(RLSProvisionResult)
	if cfg == nil || !cfg.Database.GetDialect().IsPostgres() {
		return result, nil
	}

	added, err := ReconcileRLS(ctx, db)
	if err != nil {
		return nil, err
	}
	result.PoliciesAdded = added

	if !cfg.Database.RLS.Enabled() {
		return result, nil
	}

	key, err := cfg.Database.RLS.DecodeScopeKey()
	if err != nil {
		return nil, err
	}

	if err = InstallRLSScopeKey(ctx, db, strings.TrimSpace(cfg.Database.RLS.ScopeKeyID), key); err != nil {
		return nil, err
	}
	result.KeyInstalled = true

	return result, nil
}

func ReconcileRLS(ctx context.Context, db bun.IDB) (int, error) {
	var added int
	if err := db.NewRaw("SELECT trenova_rls.reconcile()").Scan(ctx, &added); err != nil {
		return 0, fmt.Errorf("reconcile row-level security policies and grants: %w", err)
	}

	return added, nil
}

func InstallRLSScopeKey(ctx context.Context, db bun.IDB, keyID string, key []byte) error {
	if !scopeKeyIDPattern.MatchString(keyID) || len(key) < minScopeKeyBytes {
		return ErrInvalidScopeKey
	}

	pads := deriveScopePads(key)

	_, err := db.NewRaw(
		`INSERT INTO trenova_rls.scope_keys (key_id, inner_pad, outer_pad)
		VALUES (?, ?, ?)
		ON CONFLICT (key_id) DO UPDATE
		SET inner_pad = EXCLUDED.inner_pad,
			outer_pad = EXCLUDED.outer_pad,
			retired_at = NULL
		WHERE trenova_rls.scope_keys.inner_pad IS DISTINCT FROM EXCLUDED.inner_pad
			OR trenova_rls.scope_keys.outer_pad IS DISTINCT FROM EXCLUDED.outer_pad
			OR trenova_rls.scope_keys.retired_at IS NOT NULL`,
		keyID,
		pads.Inner,
		pads.Outer,
	).Exec(ctx)
	if err != nil {
		return fmt.Errorf("install row-level security scope key %q: %w", keyID, err)
	}

	return nil
}

func RetireRLSScopeKey(ctx context.Context, db bun.IDB, keyID string) (bool, error) {
	if !scopeKeyIDPattern.MatchString(keyID) {
		return false, ErrInvalidScopeKey
	}

	res, err := db.NewRaw(
		`UPDATE trenova_rls.scope_keys
		SET retired_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
		WHERE key_id = ? AND retired_at IS NULL`,
		keyID,
	).Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("retire row-level security scope key %q: %w", keyID, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return rows > 0, nil
}

type RLSLoginRole struct {
	User     string
	Password string
}

type ProvisionRLSRolesParams struct {
	App    RLSLoginRole
	System RLSLoginRole
}

func ProvisionRLSRoles(ctx context.Context, db bun.IDB, params ProvisionRLSRolesParams) ([]string, error) {
	app := strings.TrimSpace(params.App.User)
	system := strings.TrimSpace(params.System.User)

	if app == "" || params.App.Password == "" || system == "" || params.System.Password == "" {
		return nil, ErrRLSProvisionRoleRequired
	}
	if strings.EqualFold(app, system) {
		return nil, ErrRLSProvisionSameRole
	}

	appVerifier, err := scramSHA256Verifier(params.App.Password)
	if err != nil {
		return nil, err
	}
	systemVerifier, err := scramSHA256Verifier(params.System.Password)
	if err != nil {
		return nil, err
	}

	roles := []struct {
		name     string
		verifier string
		member   string
		excluded string
	}{
		{name: app, verifier: appVerifier, member: tenantRoleName, excluded: bypassRoleName},
		{name: system, verifier: systemVerifier, member: bypassRoleName, excluded: tenantRoleName},
	}

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, role := range roles {
			if _, err := tx.NewRaw(
				provisionLoginRoleSQL,
				role.name,
				role.verifier,
				role.member,
				role.excluded,
			).Exec(ctx); err != nil {
				return fmt.Errorf("provision database role %q: %w", role.name, err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return []string{app, system}, nil
}

const provisionLoginRoleSQL = `
DO $provision$
DECLARE
	role_name text := ?;
	role_verifier text := ?;
	member_of text := ?;
	excluded_from text := ?;
	attributes text := 'LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS INHERIT';
BEGIN
	IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = role_name) THEN
		EXECUTE format('ALTER ROLE %I %s PASSWORD %L', role_name, attributes, role_verifier);
	ELSE
		EXECUTE format('CREATE ROLE %I %s PASSWORD %L', role_name, attributes, role_verifier);
	END IF;

	IF pg_has_role(role_name, excluded_from, 'MEMBER') THEN
		EXECUTE format('REVOKE %I FROM %I', excluded_from, role_name);
	END IF;

	IF NOT pg_has_role(role_name, member_of, 'MEMBER') THEN
		EXECUTE format('GRANT %I TO %I', member_of, role_name);
	END IF;
END
$provision$`

type RLSStatus struct {
	ActiveKeys        []string
	ProtectedTables   int
	GlobalTables      int
	UnprotectedTables []string
}

func ReadRLSStatus(ctx context.Context, db bun.IDB) (*RLSStatus, error) {
	status := new(RLSStatus)

	if err := db.NewRaw(
		`SELECT key_id FROM trenova_rls.scope_keys WHERE retired_at IS NULL ORDER BY created_at`,
	).Scan(ctx, &status.ActiveKeys); err != nil {
		return nil, fmt.Errorf("read row-level security scope keys: %w", err)
	}

	if err := db.NewRaw(
		`SELECT
			count(*) FILTER (WHERE c.relrowsecurity AND c.relforcerowsecurity),
			count(*) FILTER (WHERE g.table_name IS NOT NULL)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN trenova_rls.global_tables g ON g.table_name = c.relname
		WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p')`,
	).Scan(ctx, &status.ProtectedTables, &status.GlobalTables); err != nil {
		return nil, fmt.Errorf("count row-level security tables: %w", err)
	}

	if err := db.NewRaw(
		`SELECT c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
			AND c.relkind IN ('r', 'p')
			AND NOT (c.relrowsecurity AND c.relforcerowsecurity)
			AND NOT EXISTS (SELECT 1 FROM trenova_rls.global_tables g WHERE g.table_name = c.relname)
		ORDER BY c.relname`,
	).Scan(ctx, &status.UnprotectedTables); err != nil {
		return nil, fmt.Errorf("list tables without row-level security: %w", err)
	}

	return status, nil
}
