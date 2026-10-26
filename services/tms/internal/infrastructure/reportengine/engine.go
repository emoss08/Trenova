package reportengine

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/cache"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	trenovaredis "github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	cachePrefix = "report:analytics:"
	slotWait    = 2 * time.Second
)

var (
	ErrNotStarted = errors.New("report engine: the reporting database is not connected")

	_ reportports.ReportEngine = (*Engine)(nil)
)

type Params struct {
	fx.In

	LC      fx.Lifecycle
	DB      *postgres.ReportingConnection
	Catalog *analytics.Catalog
	Redis   *redis.Client `optional:"true"`
	Config  *config.Config
	Logger  *zap.Logger
}

type Engine struct {
	catalog *analytics.Catalog
	db      *postgres.ReportingConnection
	redis   *redis.Client
	cfg     *config.ReportingConfig
	l       *zap.Logger

	mu          sync.RWMutex
	interactive *analytics.Runner
	cached      *analytics.Runner
}

func New(p Params) *Engine {
	e := &Engine{
		catalog: p.Catalog,
		db:      p.DB,
		redis:   p.Redis,
		cfg:     p.Config.GetReportingConfig(),
		l:       p.Logger.Named("reporting.engine"),
	}
	p.LC.Append(fx.Hook{OnStart: e.start})
	return e
}

func (e *Engine) start(ctx context.Context) error {
	db := e.db.Keel(context.WithoutCancel(ctx))
	if db == nil {
		return ErrNotStarted
	}

	interactive, err := analytics.NewRunner(db, e.catalog, e.options(nil))
	if err != nil {
		return err
	}

	cached := interactive
	if store := e.resultStore(); store != nil {
		if cached, err = analytics.NewRunner(db, e.catalog, e.options(store)); err != nil {
			return err
		}
	}

	e.mu.Lock()
	e.interactive, e.cached = interactive, cached
	e.mu.Unlock()
	return nil
}

func (e *Engine) resultStore() analytics.Cache {
	cfg := &e.cfg.ResultCache
	if !cfg.Enabled {
		return nil
	}
	local := cache.NewMemory(cache.MemoryOptions{MaxBytes: cfg.GetLocalBytes()})
	if !cfg.SharedEnabled() || e.redis == nil {
		return local
	}
	return &tieredStore{
		local: local,
		shared: cache.NewRedis(
			trenovaredis.NewBytesClient(e.redis),
			cache.RedisOptions{Prefix: cachePrefix},
		),
		ttl: cfg.GetTTL(),
	}
}

func (e *Engine) options(store analytics.Cache) analytics.RunnerOptions {
	cfg := e.cfg
	opts := analytics.RunnerOptions{
		Limits:      DefinitionLimits(cfg),
		Complexity:  ComplexityLimits(cfg),
		MaxCost:     cfg.GetExplainCostLimit(),
		MaxPlanRows: cfg.GetExplainRowLimit(),
		PreviewRows: cfg.GetPreviewRowLimit(),
		MaxRows:     int(cfg.GetMaxRows()),
		Slots:       cfg.GetMaxConcurrentStatementsPerOrg(),
		SlotWait:    slotWait,
		Timeouts: analytics.Timeouts{
			Preview: cfg.GetPreviewStatementTimeout(),
			Run:     cfg.GetStatementTimeout(),
			Page:    cfg.GetPreviewStatementTimeout(),
		},
		Primary: true,
		OnError: e.report,
	}
	if store != nil {
		opts.Cache = store
		opts.CacheTTL = cfg.ResultCache.GetTTL()
	}
	return opts
}

func DefinitionLimits(cfg *config.ReportingConfig) ir.Limits {
	return ir.Limits{
		MaxDimensions:   cfg.GetMaxDimensions(),
		MaxLimit:        cfg.GetMaxDefinitionLimit(),
		MaxPivotColumns: cfg.GetMaxPivotColumns(),
		MaxPathDepth:    cfg.GetMaxPathDepth(),
	}
}

func ComplexityLimits(cfg *config.ReportingConfig) analytics.ComplexityLimits {
	return analytics.ComplexityLimits{
		MaxJoins:  cfg.GetMaxToOneJoins(),
		MaxToMany: cfg.GetMaxToManySubqueries(),
	}
}

func (e *Engine) report(err error) {
	if errors.Is(err, cache.ErrTooLarge) {
		return
	}
	e.l.Warn("report engine", zap.Error(err))
}

func (e *Engine) runners() (interactive, cached *analytics.Runner, err error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.interactive == nil {
		return nil, nil, ErrNotStarted
	}
	return e.interactive, e.cached, nil
}

func (e *Engine) Catalog() *analytics.Catalog { return e.catalog }

func (e *Engine) Validate(def *ir.Definition) error {
	_, err := ir.Resolve(def, e.catalog, DefinitionLimits(e.cfg))
	return err
}

func (e *Engine) Preview(ctx context.Context, req *analytics.Request) (*analytics.Result, error) {
	interactive, _, err := e.runners()
	if err != nil {
		return nil, err
	}
	return interactive.Preview(ctx, *req)
}

func (e *Engine) Page(
	ctx context.Context,
	req *analytics.Request,
	cursor string,
	size int,
) (*analytics.Result, error) {
	interactive, _, err := e.runners()
	if err != nil {
		return nil, err
	}
	return interactive.Page(ctx, *req, cursor, size)
}

func (e *Engine) Drill(
	ctx context.Context,
	req *analytics.Request,
	row []ir.DrillValue,
	column string,
	opts *analytics.DrillOptions,
) (*analytics.Result, error) {
	interactive, _, err := e.runners()
	if err != nil {
		return nil, err
	}
	return interactive.Drill(ctx, *req, row, column, *opts)
}

func (e *Engine) Compile(ctx context.Context, req *analytics.Request) (*analytics.Compiled, error) {
	interactive, _, err := e.runners()
	if err != nil {
		return nil, err
	}
	return interactive.Compile(ctx, *req)
}

func (e *Engine) Run(ctx context.Context, req *analytics.Request) (*analytics.Stream, error) {
	interactive, cached, err := e.runners()
	if err != nil {
		return nil, err
	}
	scope, err := readscope.Encode(ctx)
	if err != nil {
		return nil, err
	}
	if scope != "" {
		return interactive.Run(ctx, *req)
	}
	return cached.Run(ctx, *req)
}
