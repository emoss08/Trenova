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
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantturnservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	LockDir   string
}

type deps struct {
	fx.In

	Config           *config.Config
	Logger           *zap.Logger
	DB               kernelports.DBConnection
	Assistant        agentports.AssistantService
	Pages            agentports.PageAssistant
	AgentDefinitions agentdefinitionports.AgentDefinitionService
	Turns            *assistantturnservice.Service
	Users            tenantports.UserRepository
	Definitions      agentdefinitionports.AgentDefinitionRepository
	Providers        agentports.AIProviderRepository
	Proposals        agentports.AgentProposalRepository
	Previews         agentports.ProposalPreviewService
	Decisions        agentports.AgentDecisionService
	Plans            agentports.AgentPlanService
	Committer        agentports.ApprovalCommitter
	Documents        *documentservice.Service
	Contents         documentports.DocumentContentService
	Recommender      agentports.AgentRecommender
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
	held     *namespaceLock
	tidied   sync.Once
}

func (b *Bench) exclusive(key string) func() {
	value, _ := b.locks.LoadOrStore(key, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()

	return lock.Unlock
}

func Open(ctx context.Context, opts OpenOptions) (*Bench, error) {
	held, err := lockNamespace(opts.LockDir, opts.Namespace)
	if err != nil {
		return nil, err
	}
	bench, err := open(ctx, opts)
	if err != nil {
		return nil, errors.Join(err, held.release())
	}
	bench.held = held

	return bench, nil
}

func open(ctx context.Context, opts OpenOptions) (*Bench, error) {
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
		fx.Decorate(func(inner agentports.CompletionService) agentports.CompletionService {
			bench.recorder = NewRecorder(inner)
			return bench.recorder
		}),
		fx.Decorate(func(agentports.AgentReflectionScheduler) agentports.AgentReflectionScheduler {
			return unlearned{}
		}),
		fx.Decorate(func(inner agentports.AgentTrustService) agentports.AgentTrustService {
			return untrusted{AgentTrustService: inner}
		}),
		fx.Decorate(func(inner kernelports.RateLimitStore) kernelports.RateLimitStore {
			return unthrottledMatches{RateLimitStore: inner}
		}),
		fx.Decorate(func(inner kernelports.WorkflowStarter) kernelports.WorkflowStarter {
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
		return nil, errors.New(
			"the bench could not attach to the model router and the workflow starter",
		)
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
		stopErr = closeErr
	}

	return errors.Join(stopErr, b.held.release())
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
