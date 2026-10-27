package tableinsight

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"go.uber.org/fx"
)

type BaseQuery func(ctx context.Context, scope *permissionports.TableInsightScope) (Base, error)

type Column struct {
	Col       keel.Col
	Facetable bool
	Summable  bool
	Timeline  bool
}

type Columns map[string]Column

func Facet(col keel.Col) Column {
	return Column{Col: col, Facetable: true}
}

func Sum(col keel.Col) Column {
	return Column{Col: col, Summable: true}
}

func Timeline(col keel.Col) Column {
	return Column{Col: col, Timeline: true}
}

func (c Columns) Fields() []permissionports.TableInsightField {
	fields := make([]permissionports.TableInsightField, 0, len(c))
	for name := range c {
		column := c[name]
		fields = append(fields, permissionports.TableInsightField{
			Name:      name,
			Facetable: column.Facetable,
			Summable:  column.Summable,
			Timeline:  column.Timeline,
		})
	}
	slices.SortFunc(fields, func(a, b permissionports.TableInsightField) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return fields
}

type Config struct {
	Resource permission.Resource
	DB       *postgres.Connection
	IDColumn keel.Col
	Columns  Columns
	Base     BaseQuery
}

type Result struct {
	fx.Out

	Source permissionports.TableInsightSource `group:"table_insight_sources"`
}

type source struct {
	cfg *Config
}

func New(cfg *Config) Result {
	return Result{Source: &source{cfg: cfg}}
}

func Filtered[T any](table keel.Handle[*T]) BaseQuery {
	return func(
		_ context.Context,
		scope *permissionports.TableInsightScope,
	) (Base, error) {
		if scope.Filter == nil {
			return nil, errortypes.NewValidationError(
				"filter",
				errortypes.ErrRequired,
				"Counts and totals need the table's filters",
			)
		}

		return Typed(listquery.Filter(table.Query(), scope.Filter)), nil
	}
}

func (s *source) Resource() permission.Resource {
	return s.cfg.Resource
}

func (s *source) InsightFields() []permissionports.TableInsightField {
	return s.cfg.Columns.Fields()
}

func (s *source) Facet(
	ctx context.Context,
	req *permissionports.TableFacetRequest,
) (*permissionports.TableFacetResult, error) {
	return dbtx.Read(
		ctx,
		s.cfg.DB,
		func(ctx context.Context) (*permissionports.TableFacetResult, error) {
			base, err := s.cfg.Base(ctx, &req.Scope)
			if err != nil {
				return nil, err
			}
			return base.facet(ctx, s.cfg.DB.Keel(ctx), s.cfg.Columns, req)
		},
	)
}

func (s *source) Aggregate(
	ctx context.Context,
	req *permissionports.TableAggregateRequest,
) (*permissionports.TableAggregateResult, error) {
	return dbtx.Read(
		ctx,
		s.cfg.DB,
		func(ctx context.Context) (*permissionports.TableAggregateResult, error) {
			base, err := s.cfg.Base(ctx, &req.Scope)
			if err != nil {
				return nil, err
			}
			return base.aggregate(ctx, s.cfg.DB.Keel(ctx), s.cfg.Columns, req)
		},
	)
}

func (s *source) MatchingIDs(
	ctx context.Context,
	req *permissionports.TableMatchRequest,
) (*permissionports.TableMatchResult, error) {
	return dbtx.Read(
		ctx,
		s.cfg.DB,
		func(ctx context.Context) (*permissionports.TableMatchResult, error) {
			base, err := s.cfg.Base(ctx, &req.Scope)
			if err != nil {
				return nil, err
			}
			return base.matchingIDs(ctx, s.cfg.DB.Keel(ctx), s.cfg.IDColumn, req.Limit)
		},
	)
}

func (s *source) Series(
	ctx context.Context,
	req *permissionports.TableSeriesRequest,
) (*permissionports.TableSeriesResult, error) {
	return dbtx.Read(
		ctx,
		s.cfg.DB,
		func(ctx context.Context) (*permissionports.TableSeriesResult, error) {
			base, err := s.cfg.Base(ctx, &req.Scope)
			if err != nil {
				return nil, err
			}
			return base.series(ctx, s.cfg.DB.Keel(ctx), s.cfg.Columns, req)
		},
	)
}
