package aiproviderrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AIProviderRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aiprovider-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListAIProviderRequest,
) *bun.SelectQuery {
	cols := buncolgen.ProviderColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.ProviderTable.Alias,
		req.Filter,
		(*aiprovider.Provider)(nil),
	)

	return q.Apply(buncolgen.ProviderApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.Priority.OrderAsc(), cols.Name.OrderAsc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAIProviderRequest,
) (*pagination.ListResult[*aiprovider.Provider], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*aiprovider.Provider, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count ai providers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*aiprovider.Provider]{Items: entities, Total: total}, nil
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAIProviderConnectionRequest,
) (*pagination.CursorListResult[*aiprovider.Provider], error) {
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*aiprovider.Provider)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.ProviderTable.Alias,
					req.Filter,
					(*aiprovider.Provider)(nil),
				)

				return sq.Apply(buncolgen.ProviderApplyTenant(req.Filter.TenantInfo))
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count ai providers", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*aiprovider.Provider]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*aiprovider.Provider) *bun.SelectQuery {
				sq := dba.NewSelect().Model(entities)
				if len(req.Columns) == 0 {
					return sq.ColumnExpr(buncolgen.ProviderTable.All())
				}

				return sq.Column(req.Columns...)
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.ProviderTable.Alias,
					req.Filter,
					req.Cursor,
					(*aiprovider.Provider)(nil),
				)
			},
		})
	if err != nil {
		r.l.Error("failed to scan ai providers", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	entity := new(aiprovider.Provider)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ProviderScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ProviderColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AIProvider")
	}

	return entity, nil
}

// ListForTask returns the enabled providers assigned to a task in priority order.
// Ordering is resolved in SQL rather than in the router so the fallback chain is
// stable across calls and visible in the query plan.
func (r *repository) ListForTask(
	ctx context.Context,
	req repositories.ListAIProvidersForTaskRequest,
) ([]*aiprovider.Provider, error) {
	entities := make([]*aiprovider.Provider, 0, defaultTaskCandidates)

	if err := buildProvidersForTaskQuery(
		r.db.DBForContext(ctx), &entities, req,
	).Scan(ctx); err != nil {
		return nil, fmt.Errorf("list ai providers for task %q: %w", req.Task, err)
	}

	return entities, nil
}

// buildProvidersForTaskQuery is split out so the rendered SQL can be asserted
// without a live database. The task argument has to reach PostgreSQL as an
// array literal, and a Go slice and a pgdialect.Array are indistinguishable to
// the compiler — the difference only shows up in the statement.
func buildProvidersForTaskQuery(
	db bun.IDB,
	entities *[]*aiprovider.Provider,
	req repositories.ListAIProvidersForTaskRequest,
) *bun.SelectQuery {
	cols := buncolgen.ProviderColumns

	return db.
		NewSelect().
		Model(entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ProviderScopeTenant(sq, req.TenantInfo).
				Where(cols.Enabled.IsTrue()).
				Where(
					cols.Tasks.Expr("{} @> ?::text[]"),
					pgdialect.Array([]string{string(req.Task)}),
				)
		}).
		Order(cols.Priority.OrderAsc(), cols.CreatedAt.OrderAsc())
}

func (r *repository) Create(
	ctx context.Context,
	entity *aiprovider.Provider,
) (*aiprovider.Provider, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateProviderName(entity.Name)
		}
		log.Error("failed to create ai provider", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// duplicateProviderName surfaces the unique index on (organization, business
// unit, lower(name)) as a field error, since the name is how an operator tells
// two endpoints apart when assigning tasks.
func duplicateProviderName(name string) error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"name",
		errortypes.ErrDuplicate,
		fmt.Sprintf("An AI provider named %q already exists", name),
	)

	return multiErr
}

func (r *repository) Update(
	ctx context.Context,
	entity *aiprovider.Provider,
) (*aiprovider.Provider, error) {
	cols := buncolgen.ProviderColumns
	ov := entity.Version
	entity.Version++

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ProviderScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), ov)
		}).
		Set(cols.Name.Set(), entity.Name).
		Set(cols.Description.Set(), entity.Description).
		Set(cols.Kind.Set(), entity.Kind).
		Set(cols.BaseURL.Set(), entity.BaseURL).
		Set(cols.Model.Set(), entity.Model).
		Set(cols.APIKey.Set(), entity.APIKey).
		Set(cols.AllowPrivateNetwork.Set(), entity.AllowPrivateNetwork).
		Set(cols.StructuredOutputMode.Set(), entity.StructuredOutputMode).
		Set(cols.MaxTokens.Set(), entity.MaxTokens).
		Set(cols.Tasks.Set(), dbhelper.TextArray(entity.Tasks)).
		Set(cols.Priority.Set(), entity.Priority).
		Set(cols.Trusted.Set(), entity.Trusted).
		Set(cols.Enabled.Set(), entity.Enabled).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Set(), entity.Version).
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateProviderName(entity.Name)
		}
		return nil, fmt.Errorf("update ai provider: %w", err)
	}

	if err = dberror.CheckRowsAffected(res, "AIProvider", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteAIProviderRequest,
) error {
	res, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*aiprovider.Provider)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.ProviderScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.ProviderColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete ai provider: %w", err)
	}

	return dberror.CheckRowsAffected(res, "AIProvider", req.ID.String())
}

const defaultTaskCandidates = 4

func (r *repository) MarkTested(
	ctx context.Context,
	req repositories.MarkAIProviderTestedRequest,
) error {
	cols := buncolgen.ProviderColumns

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*aiprovider.Provider)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ProviderScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Set(cols.LastTest.Set(), req.Outcome).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("mark ai provider tested: %w", err)
	}

	return dberror.CheckRowsAffected(res, "AIProvider", req.ID.String())
}
