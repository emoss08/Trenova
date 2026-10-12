package triagetemplaterepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

const templateEntity = "TriageReplyTemplate"

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) inboundmessageports.TriageReplyTemplateRepository {
	return &repository{db: p.DB}
}

func (r *repository) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*triage.ReplyTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triage.ReplyTemplate, error) {
		cols := triagecol.ReplyTemplateColumns
		entities := make([]*triage.ReplyTemplate, 0, triage.MaxTemplatesPerTenant)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Apply(triagecol.ReplyTemplateApplyTenant(tenantInfo)).
			OrderExpr(cols.Name.Expr("lower({})") + " ASC").
			Order(cols.ID.OrderAsc()).
			Limit(triage.MaxTemplatesPerTenant).
			Scan(ctx); err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) Count(ctx context.Context, tenantInfo pagination.TenantInfo) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*triage.ReplyTemplate)(nil)).
			Apply(triagecol.ReplyTemplateApplyTenant(tenantInfo)).
			Count(ctx)

		return int(count), err
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req inboundmessageports.GetTriageReplyTemplateRequest,
) (*triage.ReplyTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.ReplyTemplate, error) {
		entity := new(triage.ReplyTemplate)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(triagecol.ReplyTemplateColumns.ID.Eq(), req.ID).
			Apply(triagecol.ReplyTemplateApplyTenant(req.TenantInfo)).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, templateEntity)
		}

		return entity, nil
	})
}

func (r *repository) NameTaken(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	name string,
	exceptID pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		cols := triagecol.ReplyTemplateColumns
		query := r.db.DBForContext(ctx).
			NewSelect().
			Model((*triage.ReplyTemplate)(nil)).
			Where(cols.Name.Expr("lower({})")+" = ?", strings.ToLower(strings.TrimSpace(name))).
			Apply(triagecol.ReplyTemplateApplyTenant(tenantInfo))
		if exceptID.IsNotNil() {
			query = query.Where(cols.ID.Ne(), exceptID)
		}

		return query.Exists(ctx)
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *triage.ReplyTemplate,
) (*triage.ReplyTemplate, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.ReplyTemplate, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *triage.ReplyTemplate,
) (*triage.ReplyTemplate, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.ReplyTemplate, error) {
		previous := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(triagecol.ReplyTemplateColumns.Version.Eq(), previous).
			Returning("*").
			Exec(ctx)
		if err != nil {
			entity.Version = previous

			return nil, err
		}
		if err = dberror.CheckRowsAffected(
			results,
			templateEntity,
			entity.ID.String(),
		); err != nil {
			entity.Version = previous

			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req inboundmessageports.GetTriageReplyTemplateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		result, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*triage.ReplyTemplate)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return triagecol.ReplyTemplateScopeTenantDelete(dq, req.TenantInfo).
					Where(triagecol.ReplyTemplateColumns.ID.Eq(), req.ID)
			}).
			Exec(ctx)
		if err != nil {
			return err
		}

		return dberror.CheckFound(result, templateEntity)
	})
}
