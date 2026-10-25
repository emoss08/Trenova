package aiproviderrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var providerErrors = dberror.Mapping{Entity: "AIProvider"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.AIProviderRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aiprovider-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *agentports.ListAIProviderRequest,
) (*pagination.ListResult[*aiprovider.Provider], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*aiprovider.Provider], error) {
			p := aiproviderdb.Providers
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				p.Query(),
				&listquery.Request[aiprovider.Provider]{
					Filter: req.Filter,
					Then:   []keel.Order{p.Priority.Asc(), p.Name.Asc()},
				},
			)
			if err != nil {
				r.l.Error("failed to scan and count ai providers", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentports.ListAIProviderConnectionRequest,
) (*pagination.CursorListResult[*aiprovider.Provider], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*aiprovider.Provider], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				aiproviderdb.Providers.Query(),
				&listquery.Request[aiprovider.Provider]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to scan ai providers", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentports.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aiprovider.Provider, error) {
		entity, err := aiproviderdb.Providers.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, providerErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

// ListForTask returns the enabled providers assigned to a task in priority order.
// Ordering is resolved in SQL rather than in the router so the fallback chain is
// stable across calls and visible in the query plan.
func (r *repository) ListForTask(
	ctx context.Context,
	req agentports.ListAIProvidersForTaskRequest,
) ([]*aiprovider.Provider, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiprovider.Provider, error) {
		entities, err := providersForTaskQuery(req.Task).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list ai providers for task %q: %w", req.Task, err)
		}

		return entities, nil
	})
}

func (r *repository) ListEnabled(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*aiprovider.Provider, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiprovider.Provider, error) {
		p := aiproviderdb.Providers
		entities, err := p.Omit(p.APIKey, p.PreviousAPIKey).
			Where(p.Enabled.IsTrue()).
			Order(p.Priority.Asc(), p.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list enabled ai providers: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListOrdered(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*aiprovider.Provider, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiprovider.Provider, error) {
		p := aiproviderdb.Providers
		entities, err := p.Order(p.Priority.Asc(), p.CreatedAt.Asc(), p.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list ai providers in routing order: %w", err)
		}

		for idx, entity := range entities {
			entities[idx] = entity.Redacted()
		}
		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *agentports.AIProviderSelectOptionsRequest,
) (*pagination.ListResult[*aiprovider.Provider], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*aiprovider.Provider], error) {
			p := aiproviderdb.Providers
			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				selectOptionsQuery(req.Task),
				req.SelectQueryRequest,
				func(query string) keel.Pred {
					return keel.Or(
						listquery.TextContainsFold(p.Name, query),
						listquery.TextContainsFold(p.Model, query),
					)
				},
			)
		},
	)
}

func selectOptionsQuery(task aiprovider.Task) *keel.Query[*aiprovider.Provider] {
	p := aiproviderdb.Providers
	q := p.Pick(p.ID, p.Name, p.Model, p.Kind, p.Enabled, p.Tasks, p.Priority, p.CreatedAt).
		Where(p.Enabled.IsTrue())
	if task != "" {
		q.Where(p.Tasks.ContainsAll(string(task)))
	}

	return q.Order(p.Priority.Asc(), p.Name.Asc(), p.ID.Asc())
}

func providersForTaskQuery(task aiprovider.Task) *keel.Query[*aiprovider.Provider] {
	p := aiproviderdb.Providers

	return p.Where(p.Enabled.IsTrue(), p.Tasks.ContainsAll(string(task))).
		Order(p.Priority.Asc(), p.CreatedAt.Asc())
}

func (r *repository) Create(
	ctx context.Context,
	entity *aiprovider.Provider,
) (*aiprovider.Provider, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aiprovider.Provider, error) {
		if err := aiproviderdb.Providers.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateProviderName(entity.Name)
			}
			r.l.Error("failed to create ai provider",
				zap.String("name", entity.Name),
				zap.Error(err),
			)
			return nil, providerErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
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
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aiprovider.Provider, error) {
		p := aiproviderdb.Providers
		if err := p.SaveExcept(ctx, r.db.Keel(ctx), entity, p.LastTest); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateProviderName(entity.Name)
			}
			return nil, providerErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req agentports.DeleteAIProviderRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := aiproviderdb.Providers
		rows, err := p.Where(p.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return providerErrors.Map(err, req.ID.String())
		}

		return dberror.CheckAffected(rows, "AIProvider", req.ID.String())
	})
}

func untouched() []keel.Assignment {
	p := aiproviderdb.Providers

	return []keel.Assignment{p.Version.ToExpr(p.Version), p.UpdatedAt.ToExpr(p.UpdatedAt)}
}

func (r *repository) MarkTested(
	ctx context.Context,
	req agentports.MarkAIProviderTestedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := aiproviderdb.Providers
		rows, err := p.Where(p.ID.Eq(req.ID)).
			Update(ctx, r.db.Keel(ctx), append(untouched(), p.LastTest.To(req.Outcome))...)
		if err != nil {
			return fmt.Errorf("mark ai provider tested: %w", err)
		}

		return dberror.CheckAffected(rows, "AIProvider", req.ID.String())
	})
}
