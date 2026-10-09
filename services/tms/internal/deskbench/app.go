package deskbench

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	temporallog "go.temporal.io/sdk/log"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

const (
	startTimeout = 2 * time.Minute
	stopTimeout  = 30 * time.Second
	logFileName  = "worker.log"
	sdkLogName   = "temporal.log"
)

type OpenOptions struct {
	Namespace string
	OutDir    string
	LivePaths []string
}

type deps struct {
	fx.In

	Config           *config.Config
	Logger           *zap.Logger
	DB               ports.DBConnection
	Assistant        serviceports.AssistantService
	Pages            serviceports.PageAssistant
	AgentDefinitions serviceports.AgentDefinitionService
	Turns            *assistantturnservice.Service
	Users            repositories.UserRepository
	Definitions      repositories.AgentDefinitionRepository
	Providers        repositories.AIProviderRepository
	Proposals        repositories.AgentProposalRepository
	Previews         serviceports.ProposalPreviewService
	Decisions        serviceports.AgentDecisionService
	Plans            serviceports.AgentPlanService
	Committer        serviceports.ApprovalCommitter
}

type Bench struct {
	deps
	recorder *Recorder
	watcher  *TurnWatcher
	app      *fx.App
	logPath  string
	sdkLog   *os.File
	live     *LiveFeed
	locks    sync.Map
}

func (b *Bench) exclusive(key string) func() {
	value, _ := b.locks.LoadOrStore(key, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()

	return lock.Unlock
}

func Open(ctx context.Context, opts OpenOptions) (*Bench, error) {
	bench := &Bench{logPath: filepath.Join(opts.OutDir, logFileName)}

	sdkLog, err := os.OpenFile(
		filepath.Join(opts.OutDir, sdkLogName),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		fileMode,
	)
	if err != nil {
		return nil, fmt.Errorf("open the temporal log: %w", err)
	}
	bench.sdkLog = sdkLog

	live, err := OpenLiveFeed(opts.LivePaths...)
	if err != nil {
		_ = sdkLog.Close()
		return nil, err
	}
	bench.live = live
	sdkLogger := temporallog.NewStructuredLogger(slog.New(slog.NewJSONHandler(sdkLog, nil)))

	app := fx.New(
		fx.Provide(func() temporallog.Logger { return sdkLogger }),
		bootstrap.Options(),
		bootstrap.UnscheduledWorkerOptions(),
		fx.Decorate(func(cfg *config.Config) *config.Config {
			return isolate(cfg, opts.Namespace, bench.logPath)
		}),
		fx.Decorate(func(inner serviceports.CompletionService) serviceports.CompletionService {
			bench.recorder = NewRecorder(inner)
			return bench.recorder
		}),
		fx.Decorate(func(inner serviceports.WorkflowStarter) serviceports.WorkflowStarter {
			bench.watcher = NewTurnWatcher(inner)
			return bench.watcher
		}),
		fx.Invoke(func(d deps) { bench.deps = d }),
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger.Named("fx")}
		}),
	)
	if err = app.Err(); err != nil {
		_ = sdkLog.Close()
		live.Close()
		return nil, fmt.Errorf("build the bench (see %s): %w", bench.logPath, err)
	}

	startCtx, cancel := context.WithTimeout(ctx, startTimeout)
	defer cancel()
	if err = app.Start(startCtx); err != nil {
		_ = sdkLog.Close()
		live.Close()
		return nil, fmt.Errorf("start the bench (see %s): %w", bench.logPath, err)
	}
	if bench.recorder == nil || bench.watcher == nil {
		_ = app.Stop(context.Background())
		_ = sdkLog.Close()
		live.Close()
		return nil, errors.New("the bench could not attach to the model router and the workflow starter")
	}
	bench.app = app

	return bench, nil
}

func (b *Bench) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()

	stopErr := b.app.Stop(ctx)
	b.live.Close()
	if closeErr := b.sdkLog.Close(); closeErr != nil && stopErr == nil {
		return closeErr
	}

	return stopErr
}

func (b *Bench) LogPath() string {
	return b.logPath
}

func isolate(cfg *config.Config, namespace, logPath string) *config.Config {
	cfg.App.Debug = false
	cfg.Temporal.Namespace = namespace
	cfg.Temporal.Worker.Queues = nil
	cfg.Logging.Output = "file"
	cfg.Logging.Format = "json"
	cfg.Logging.Sampling = false
	cfg.Logging.File = &config.LogFileConfig{
		Path:       logPath,
		MaxSize:    500,
		MaxAge:     7,
		MaxBackups: 1,
	}

	return cfg
}
