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

//nolint:gocritic // fx.In parameter structs must be passed by value
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

	runners := []*analytics.Runner{interactive, interactive}
	if e.redis != nil {
		store := cache.NewRedis(redisClient{e.redis}, cache.RedisOptions{Prefix: cachePrefix})
		cached, cacheErr := analytics.NewRunner(db, e.catalog, e.options(store))
		if cacheErr != nil {
			return cacheErr
		}
		runners[1] = cached
	}

	e.mu.Lock()
	e.interactive, e.cached = runners[0], runners[1]
	e.mu.Unlock()
	return nil
}

func (e *Engine) options(store analytics.Cache) analytics.RunnerOptions {
	cfg := e.cfg
	opts := analytics.RunnerOptions{
		Limits: ir.Limits{
			MaxDimensions:   cfg.GetMaxDimensions(),
			MaxLimit:        cfg.GetMaxDefinitionLimit(),
			MaxPivotColumns: cfg.GetMaxPivotColumns(),
			MaxPathDepth:    cfg.GetMaxPathDepth(),
		},
		Complexity: analytics.ComplexityLimits{
			MaxJoins:  cfg.GetMaxToOneJoins(),
			MaxToMany: cfg.GetMaxToManySubqueries(),
		},
		MaxCost:     cfg.GetExplainCostLimit(),
		MaxPlanRows: cfg.GetExplainRowLimit(),
		PreviewRows: cfg.GetPreviewRowLimit(),
		MaxRows:     int(cfg.GetMaxRows()),
		Slots:       cfg.GetMaxConcurrentStatementsPerOrg(),
		SlotWait:    slotWait,
		Timeouts: analytics.Timeouts{
			Preview: cfg.GetPreviewStatementTimeout(),
			Run:     cfg.GetStatementTimeout(),
			Count:   cfg.GetPreviewStatementTimeout(),
			Page:    cfg.GetPreviewStatementTimeout(),
		},
		Primary: true,
		OnError: e.report,
	}
	if store != nil {
		opts.Cache = store
		opts.CacheTTL = cfg.GetResultCacheTTL()
	}
	return opts
}

func (e *Engine) report(err error) {
	e.l.Warn("report engine", zap.Error(err))
}

func (e *Engine) runners() (*analytics.Runner, *analytics.Runner, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.interactive == nil {
		return nil, nil, ErrNotStarted
	}
	return e.interactive, e.cached, nil
}

func (e *Engine) Catalog() *analytics.Catalog { return e.catalog }

func (e *Engine) Preview(ctx context.Context, req analytics.Request) (*analytics.Result, error) {
	interactive, _, err := e.runners()
	if err != nil {
		return nil, err
	}
	return interactive.Preview(ctx, req)
}

func (e *Engine) Page(
	ctx context.Context,
	req analytics.Request,
	cursor string,
	size int,
) (*analytics.Result, error) {
	interactive, _, err := e.runners()
	if err != nil {
		return nil, err
	}
	return interactive.Page(ctx, req, cursor, size)
}

func (e *Engine) Count(ctx context.Context, req analytics.Request) (int64, error) {
	interactive, _, err := e.runners()
	if err != nil {
		return 0, err
	}
	return interactive.Count(ctx, req)
}

func (e *Engine) Drill(
	ctx context.Context,
	req analytics.Request,
	row []ir.DrillValue,
	column string,
	opts analytics.DrillOptions,
) (*analytics.Result, error) {
	interactive, _, err := e.runners()
	if err != nil {
		return nil, err
	}
	return interactive.Drill(ctx, req, row, column, opts)
}

func (e *Engine) Compile(ctx context.Context, req analytics.Request) (*analytics.Compiled, error) {
	interactive, _, err := e.runners()
	if err != nil {
		return nil, err
	}
	return interactive.Compile(ctx, req)
}

func (e *Engine) Run(ctx context.Context, req analytics.Request) (*analytics.Stream, error) {
	interactive, cached, err := e.runners()
	if err != nil {
		return nil, err
	}
	scope, err := readscope.Encode(ctx)
	if err != nil {
		return nil, err
	}
	if scope != "" {
		return interactive.Run(ctx, req)
	}
	return cached.Run(ctx, req)
}

type redisClient struct {
	client *redis.Client
}

func (c redisClient) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (c redisClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c redisClient) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
