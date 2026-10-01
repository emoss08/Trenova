package db

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const rlsGeneratedKeyBytes = 48

var (
	errRLSPostgresOnly = errors.New("row-level security is only available on PostgreSQL")
	errRLSMigratorRole = errors.New(
		"database.migrator must name a different role from database.user so the application role " +
			"does not own the tables it is isolated from",
	)
)

var dbRLSCmd = &cobra.Command{
	Use:   "rls",
	Short: "Manage PostgreSQL row-level security",
	Long: `Manage the roles, signing keys and policies that isolate tenants in PostgreSQL.

Every tenant table carries a row-level security policy that admits only rows of
the organization named by the transaction's signed scope. The application role
(database.user) is a member of trenova_tenant and is bound by those policies; the
system role (database.system.user) is a member of trenova_rls_bypass and serves
only code paths that declare dbscope.WithSystem. Migrations run as
database.migrator.`,
}

var dbRLSGenerateKeyCmd = &cobra.Command{
	Use:   "generate-key",
	Short: "Generate a scope signing key for database.rls",
	Long: `Print a new random key id and key for database.rls.scopeKeyId and
database.rls.scopeKey. Store the key as a secret; the next "trenova db migrate"
installs it.`,
	RunE: runRLSGenerateKey,
}

var dbRLSProvisionRolesCmd = &cobra.Command{
	Use:   "provision-roles",
	Short: "Create or update the application and system login roles",
	Long: `Create (or update the password and attributes of) the application login role
named by database.user and the system login role named by database.system.user,
and grant them trenova_tenant and trenova_rls_bypass respectively.

Run it as database.migrator, a role with CREATEROLE. Passwords are sent as
SCRAM-SHA-256 verifiers, never in clear text.`,
	RunE: runRLSProvisionRoles,
}

var dbRLSRetireKeyCmd = &cobra.Command{
	Use:   "retire-key [key-id]",
	Short: "Stop accepting scopes signed with a key",
	Long: `Retire a scope signing key after every running instance has moved to a newer
database.rls.scopeKeyId. Transactions signed with a retired key are refused.`,
	Args: cobra.ExactArgs(1),
	RunE: runRLSRetireKey,
}

var dbRLSStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report row-level security coverage",
	RunE:  runRLSStatus,
}

func runRLSGenerateKey(_ *cobra.Command, _ []string) error {
	key := make([]byte, rlsGeneratedKeyBytes)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("generate scope key: %w", err)
	}

	color.Cyan("database.rls.scopeKeyId: k%s", time.Now().UTC().Format("20060102"))
	color.Cyan("database.rls.scopeKey:   %s", base64.StdEncoding.EncodeToString(key))

	return nil
}

func runRLSProvisionRoles(_ *cobra.Command, _ []string) error {
	if !cfg.Database.GetDialect().IsPostgres() {
		return errRLSPostgresOnly
	}

	if !cfg.Database.Migrator.Configured() ||
		strings.EqualFold(strings.TrimSpace(cfg.Database.Migrator.User), strings.TrimSpace(cfg.Database.User)) {
		return errRLSMigratorRole
	}

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	ctx := context.Background()
	roles, err := postgres.ProvisionRLSRoles(ctx, manager.GetDB(), postgres.ProvisionRLSRolesParams{
		App: postgres.RLSLoginRole{
			User:     cfg.Database.User,
			Password: cfg.Database.Password,
		},
		System: postgres.RLSLoginRole{
			User:     cfg.Database.System.User,
			Password: cfg.Database.System.Password,
		},
	})
	if err != nil {
		return err
	}

	color.Green("✓ Application role %q is bound by tenant isolation", roles[0])
	color.Green("✓ System role %q serves dbscope.WithSystem code paths", roles[1])

	return nil
}

func runRLSRetireKey(_ *cobra.Command, args []string) error {
	if !cfg.Database.GetDialect().IsPostgres() {
		return errRLSPostgresOnly
	}

	keyID := strings.TrimSpace(args[0])
	if strings.EqualFold(keyID, strings.TrimSpace(cfg.Database.RLS.ScopeKeyID)) && !force {
		return fmt.Errorf(
			"key %q is the configured database.rls.scopeKeyId; pass --force to retire it anyway",
			keyID,
		)
	}

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	retired, err := postgres.RetireRLSScopeKey(context.Background(), manager.GetDB(), keyID)
	if err != nil {
		return err
	}
	if !retired {
		color.Yellow("Key %q was not active", keyID)
		return nil
	}

	color.Green("✓ Key %q is retired", keyID)

	return nil
}

func runRLSStatus(_ *cobra.Command, _ []string) error {
	if !cfg.Database.GetDialect().IsPostgres() {
		return errRLSPostgresOnly
	}

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	status, err := postgres.ReadRLSStatus(context.Background(), manager.GetDB())
	if err != nil {
		return err
	}

	color.Cyan("Mode:                 %s", cfg.Database.RLS.GetMode())
	color.Cyan("Configured key:       %s", cfg.Database.RLS.ScopeKeyID)
	color.Cyan("Active keys:          %s", strings.Join(status.ActiveKeys, ", "))
	color.Cyan("Protected tables:     %d", status.ProtectedTables)
	color.Cyan("Global tables:        %d", status.GlobalTables)

	if len(status.UnprotectedTables) == 0 {
		color.Green("✓ Every non-global table has row-level security")
		return nil
	}

	color.Red("✗ Tables without row-level security:")
	for _, table := range status.UnprotectedTables {
		color.Red("  - %s", table)
	}

	return fmt.Errorf("%d table(s) lack row-level security", len(status.UnprotectedTables))
}

func init() {
	dbRLSCmd.AddCommand(dbRLSGenerateKeyCmd)
	dbRLSCmd.AddCommand(dbRLSProvisionRolesCmd)
	dbRLSCmd.AddCommand(dbRLSRetireKeyCmd)
	dbRLSCmd.AddCommand(dbRLSStatusCmd)
	DbCmd.AddCommand(dbRLSCmd)
}
