package tableinsight

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type BaseQuery func(
	ctx context.Context,
	dba bun.IDB,
	scope *repositories.TableInsightScope,
) (*bun.SelectQuery, error)

type Config struct {
	Resource permission.Resource
	DB       *postgres.Connection
	IDColumn buncolgen.Column
	Columns  dbhelper.InsightColumns
	Base     BaseQuery
}

type Result struct {
	fx.Out

	Source repositories.TableInsightSource `group:"table_insight_sources"`
}

type source struct {
	cfg *Config
}

func New(cfg *Config) Result {
	return Result{Source: &source{cfg: cfg}}
}

func Filtered[T domaintypes.PostgresSearchable](model T, alias string) BaseQuery {
	return func(
		_ context.Context,
		dba bun.IDB,
		scope *repositories.TableInsightScope,
	) (*bun.SelectQuery, error) {
		if scope.Filter == nil {
			return nil, errortypes.NewValidationError(
				"filter",
				errortypes.ErrRequired,
				"Counts and totals need the table's filters",
			)
		}
		return dba.NewSelect().
			Model(model).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return querybuilder.ApplyFiltersWithoutSort(sq, alias, scope.Filter, model)
			}), nil
	}
}

func (s *source) Resource() permission.Resource {
	return s.cfg.Resource
}

func (s *source) InsightFields() []repositories.TableInsightField {
	return s.cfg.Columns.Fields()
}

func (s *source) Facet(
	ctx context.Context,
	req *repositories.TableFacetRequest,
) (*repositories.TableFacetResult, error) {
	return dbtx.Read(
		ctx,
		s.cfg.DB,
		func(ctx context.Context) (*repositories.TableFacetResult, error) {
			base, err := s.cfg.Base(ctx, s.cfg.DB.DBForContext(ctx), &req.Scope)
			if err != nil {
				return nil, err
			}
			return dbhelper.Facet(ctx, base, s.cfg.Columns, req)
		},
	)
}

func (s *source) Aggregate(
	ctx context.Context,
	req *repositories.TableAggregateRequest,
) (*repositories.TableAggregateResult, error) {
	return dbtx.Read(
		ctx,
		s.cfg.DB,
		func(ctx context.Context) (*repositories.TableAggregateResult, error) {
			base, err := s.cfg.Base(ctx, s.cfg.DB.DBForContext(ctx), &req.Scope)
			if err != nil {
				return nil, err
			}
			return dbhelper.Aggregate(ctx, base, s.cfg.Columns, req)
		},
	)
}

func (s *source) MatchingIDs(
	ctx context.Context,
	req *repositories.TableMatchRequest,
) (*repositories.TableMatchResult, error) {
	return dbtx.Read(
		ctx,
		s.cfg.DB,
		func(ctx context.Context) (*repositories.TableMatchResult, error) {
			base, err := s.cfg.Base(ctx, s.cfg.DB.DBForContext(ctx), &req.Scope)
			if err != nil {
				return nil, err
			}
			return dbhelper.MatchingIDs(ctx, base, &s.cfg.IDColumn, req.Limit)
		},
	)
}

func (s *source) Series(
	ctx context.Context,
	req *repositories.TableSeriesRequest,
) (*repositories.TableSeriesResult, error) {
	return dbtx.Read(
		ctx,
		s.cfg.DB,
		func(ctx context.Context) (*repositories.TableSeriesResult, error) {
			base, err := s.cfg.Base(ctx, s.cfg.DB.DBForContext(ctx), &req.Scope)
			if err != nil {
				return nil, err
			}
			return dbhelper.Series(ctx, base, s.cfg.Columns, req)
		},
	)
}
