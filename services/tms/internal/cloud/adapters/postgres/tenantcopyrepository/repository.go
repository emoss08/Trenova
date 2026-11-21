package tenantcopyrepository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/pgxdsn"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/schemaversion"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
)

const (
	sessionReason    = "open the tenant archive's own database session, which copies one tenant's rows of every table in or out under the system role"
	planReason       = "read the catalog, which belongs to no tenant, to plan a tenant archive"
	preflightReason  = "check, before an archive is loaded, that nothing of the tenant it holds is on the server already: its organization, rows, keys, addresses and usernames, which are unique across the server"
	referencesReason = "find the reference rows (states, jurisdictions, catalogs) an archive refers to by their natural keys, which are shared by every tenant"
	partitionsReason = "make the audit partitions an imported tenant's audit trail needs; the partitions belong to no tenant"

	applicationName      = "trenova-tenant-archive"
	keyBatch             = 1_000
	danglingBatch        = 80
	sessionLockTimeout   = "5s"
	sessionIdleTimeout   = "15min"
	archiveLoadSetting   = "trenova.archive_load"
	maxStatementDuration = 6 * time.Hour
)

var (
	ErrUnitShared = errors.New(
		"the tenant's business unit holds other organizations; an archive carries one business unit with one organization",
	)
	ErrTenantMissing = errors.New("the tenant's organization is not on this server")
)

type Params struct {
	fx.In

	DB       *postgres.Connection
	App      *appconfig.Config
	Database *databaseconfig.Config
	AIAudit  *aiauditconfig.Config
}

type repository struct {
	db       *postgres.Connection
	app      *appconfig.Config
	database *databaseconfig.Config
	keys     *trailKeys
}

func New(p Params) archiveports.Rows {
	return &repository{
		db:       p.DB,
		app:      p.App,
		database: p.Database,
		keys:     newTrailKeys(p.AIAudit),
	}
}

func (r *repository) SchemaVersion() string {
	return schemaversion.Head()
}

func (r *repository) connect(ctx context.Context) (*pgx.Conn, error) {
	ctx = dbscope.WithSystem(ctx, sessionReason)
	if !dbscope.IsSystem(ctx) {
		return nil, ErrTenantMissing
	}
	dsn, err := pgxdsn.Normalize(
		r.database.DSNForRole(applicationName, r.database.System),
		pgxdsn.Options{ApplicationName: applicationName},
	)
	if err != nil {
		return nil, err
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("read the archive session's address: %w", err)
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open the archive session: %w", err)
	}
	if err = dbpool.PrepareConn(ctx, conn); err != nil {
		_ = conn.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("prepare the archive session: %w", err)
	}
	return conn, nil
}

func settle(
	ctx context.Context,
	tx pgx.Tx,
	statementTimeout time.Duration,
	extra ...[2]string,
) error {
	settings := make([][2]string, 0, len(archiveplan.SessionSettings)+4+len(extra))
	settings = append(settings, archiveplan.SessionSettings...)
	settings = append(
		settings,
		[2]string{
			"statement_timeout",
			strconv.FormatInt(statementTimeout.Milliseconds(), 10) + "ms",
		},
		[2]string{"lock_timeout", sessionLockTimeout},
		[2]string{"idle_in_transaction_session_timeout", sessionIdleTimeout},
		[2]string{"jit", "off"},
	)
	settings = append(settings, extra...)
	for _, s := range settings {
		if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", s[0], s[1]); err != nil {
			return fmt.Errorf("set %s for the archive session: %w", s[0], err)
		}
	}
	return nil
}

func statementBudget(ctx context.Context) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		if left := time.Until(deadline); left > time.Second {
			return min(left, maxStatementDuration)
		}
		return time.Second
	}
	return maxStatementDuration
}

func (r *repository) PlanFor(
	ctx context.Context,
	purpose tenantarchive.Purpose,
	envelope map[string][]string,
) (*archiveplan.Plan, error) {
	ctx = dbscope.WithSystem(ctx, planReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*archiveplan.Plan, error) {
		cat, err := readCatalog(ctx, keelRunner(r.db.Keel(ctx)))
		if err != nil {
			return nil, err
		}
		return archiveplan.Build(&archiveplan.Input{
			Catalog:  cat,
			Purpose:  purpose,
			Envelope: envelope,
		})
	})
}
