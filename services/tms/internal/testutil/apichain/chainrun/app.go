package chainrun

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/loggingconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/objectstoretest"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
	"go.uber.org/fx"
)

const (
	testConfigPath = "config/config.test.yaml"
	logDir         = "build/apichains"
	startTimeout   = 2 * time.Minute
	stopTimeout    = 30 * time.Second
)

type App struct {
	URL     string
	Admin   *keel.DB
	Effects *Effects
	LogPath string

	trips *tripCounter
}

func Boot(t *testing.T, name string) *App {
	t.Helper()

	_, admin, dsn, cleanup := seededtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	logPath, err := filepath.Abs(filepath.Join(logDir, name+".log"))
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(logPath), 0o755))
	_ = os.Remove(logPath)

	redisOptions := sharedtestutil.SetupTestRedis(t).Options()
	redisHost, rawPort, err := net.SplitHostPort(redisOptions.Addr)
	require.NoError(t, err)
	redisPort, err := strconv.Atoi(rawPort)
	require.NoError(t, err)

	cfg := loadConfig(t, logPath)
	cfg.Cache.Host = redisHost
	cfg.Cache.Port = redisPort
	cfg.Cache.DB = redisOptions.DB
	cfg.Cache.Username = ""
	cfg.Cache.Password = ""
	rls := dbtest.RLSDatabaseConfig(t, admin, dsn)
	cfg.Database.Host = rls.Host
	cfg.Database.Port = rls.Port
	cfg.Database.Name = rls.Name
	cfg.Database.User = rls.User
	cfg.Database.Password = rls.Password
	cfg.Database.SSLMode = rls.SSLMode
	cfg.Database.RLS = rls.RLS
	cfg.Database.System = rls.System

	effects := &Effects{}
	var server *api.Server
	app := fx.New(
		bootstrap.Options(),
		bootstrap.APIOptions(),
		fx.Replace(cfg),
		fx.Decorate(func() kernelports.Client { return objectstoretest.New() }),
		fx.Decorate(func() converter.StorageDriver { return refusingPayloads{} }),
		fx.Decorate(func() kernelports.WorkflowStarter {
			return workflowRecorder{effects: effects}
		}),
		fx.Decorate(func() kernelports.WorkflowSignalStarter {
			return workflowRecorder{effects: effects}
		}),
		fx.Decorate(func() kernelports.WorkflowResults {
			return workflowRecorder{effects: effects}
		}),
		fx.Decorate(
			func(inner realtimeports.RealtimePublisher) realtimeports.RealtimePublisher {
				return realtimeRecorder{inner: inner, effects: effects}
			},
		),
		fx.Decorate(func(inner auditports.AuditService) auditports.AuditService {
			return auditRecorder{AuditService: inner, effects: effects}
		}),
		fx.Populate(&server),
		fx.NopLogger,
	)
	require.NoError(t, app.Err(), "build the API graph (log: %s)", logPath)

	startCtx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()
	require.NoError(t, app.Start(startCtx), "start the API (log: %s)", logPath)
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
		defer stopCancel()
		_ = app.Stop(stopCtx)
	})

	trips := &tripCounter{}
	httpServer := httptest.NewServer(trips.wrap(server.Handler()))
	t.Cleanup(httpServer.Close)

	return &App{
		URL:     httpServer.URL,
		Admin:   admin,
		Effects: effects,
		LogPath: logPath,
		trips:   trips,
	}
}

func (a *App) RoundTrips() int {
	return a.trips.take()
}

func loadConfig(t *testing.T, logPath string) *config.Config {
	t.Helper()

	raw, err := os.ReadFile(testConfigPath)
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.test.yaml"), raw, 0o600))

	cfg, err := config.NewLoader(
		config.WithConfigPath(dir),
		config.WithEnvironment(appconfig.EnvTest),
	).Load()
	require.NoError(t, err)

	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 0
	cfg.Logging.Output = "file"
	cfg.Logging.Format = "json"
	cfg.Logging.Sampling = false
	cfg.Logging.File = &loggingconfig.FileConfig{Path: logPath, MaxSize: 100, MaxBackups: 1}

	return cfg
}

type tripCounter struct {
	mu    sync.Mutex
	trips int
}

func (c *tripCounter) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := keel.Budget(r.Context())
		next.ServeHTTP(w, r.WithContext(ctx))
		if report, ok := keel.BudgetReportFrom(ctx); ok {
			c.mu.Lock()
			c.trips += report.RoundTrips
			c.mu.Unlock()
		}
	})
}

func (c *tripCounter) take() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := c.trips
	c.trips = 0

	return out
}
