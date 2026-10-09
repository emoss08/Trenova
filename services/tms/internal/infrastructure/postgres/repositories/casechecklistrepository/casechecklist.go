package casechecklistrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	templateEntity  = "Checklist template"
	maxTemplateRows = 1000
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

func New(p Params) repositories.CaseChecklistRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.case-checklist-repository")}
}

func tenantOf(entity *deskcase.ChecklistTemplate) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

// withCustomerName joins the customer's name onto a template read, for the
// settings list.
func withCustomerName(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.ChecklistTemplateColumns
	cus := buncolgen.CustomerColumns
	alias := buncolgen.CustomerTable.Alias

	return sq.
		ColumnExpr(buncolgen.ChecklistTemplateTable.Alias+".*").
		ColumnExpr("COALESCE("+cus.Name.Qualified()+", '') AS customer_name").
		Join("LEFT JOIN "+buncolgen.CustomerTable.Name+" AS "+alias).
		JoinOn(cus.ID.EqColumn(cols.CustomerID)).
		JoinOn(cus.OrganizationID.EqColumn(cols.OrganizationID)).
		JoinOn(cus.BusinessUnitID.EqColumn(cols.BusinessUnitID))
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *repositories.ListCaseChecklistTemplatesRequest,
) ([]*deskcase.ChecklistTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*deskcase.ChecklistTemplate, error) {
		cols := buncolgen.ChecklistTemplateColumns
		items := make([]*deskcase.ChecklistTemplate, 0)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			Apply(withCustomerName).
			Apply(buncolgen.ChecklistTemplateApplyTenant(req.TenantInfo)).
			Where(cols.Kind.Eq(), req.Kind).
			OrderExpr(cols.CustomerID.Qualified() + " NULLS FIRST").
			OrderExpr("customer_name ASC").
			Limit(maxTemplateRows).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list checklist templates: %w", err)
		}

		return items, nil
	})
}

func (r *repository) TemplateFor(
	ctx context.Context,
	req *repositories.GetCaseChecklistTemplateForRequest,
) (*deskcase.ChecklistTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*deskcase.ChecklistTemplate, error) {
		cols := buncolgen.ChecklistTemplateColumns
		items := make([]*deskcase.ChecklistTemplate, 0, 1)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			Apply(buncolgen.ChecklistTemplateApplyTenant(req.TenantInfo)).
			Where(cols.Kind.Eq(), req.Kind).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = sq.Where(cols.CustomerID.IsNull())
				if req.CustomerID.IsNotNil() {
					sq = sq.WhereOr(cols.CustomerID.Eq(), req.CustomerID)
				}
				return sq
			}).
			OrderExpr(cols.CustomerID.Qualified() + " NULLS LAST").
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("read the checklist template: %w", err)
		}
		if len(items) == 0 {
			return nil, nil //nolint:nilnil // no template is the default, not an error
		}

		return items[0], nil
	})
}

func (r *repository) GetTemplate(
	ctx context.Context,
	req *repositories.GetCaseChecklistTemplateRequest,
) (*deskcase.ChecklistTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*deskcase.ChecklistTemplate, error) {
		cols := buncolgen.ChecklistTemplateColumns
		entity := new(deskcase.ChecklistTemplate)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Apply(withCustomerName).
			Apply(buncolgen.ChecklistTemplateApplyTenant(req.TenantInfo)).
			Where(cols.ID.Eq(), req.ID).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, templateEntity)
		}

		return entity, nil
	})
}

func (r *repository) SaveTemplate(
	ctx context.Context,
	entity *deskcase.ChecklistTemplate,
) (*deskcase.ChecklistTemplate, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*deskcase.ChecklistTemplate, error) {
		db := r.db.DBForContext(ctx)
		if entity.ID.IsNil() {
			entity.Version = 0
			if _, err := db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
				if dberror.IsUniqueConstraintViolation(err) {
					return nil, repositories.ErrCaseChecklistTemplateStale
				}
				return nil, fmt.Errorf("insert checklist template: %w", err)
			}
			return entity, nil
		}

		cols := buncolgen.ChecklistTemplateColumns
		read := entity.Version
		entity.Version++
		res, err := db.NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ChecklistTemplateScopeTenantUpdate(uq, tenantOf(entity)).
					Where(cols.ID.Eq(), entity.ID).
					Where(cols.Version.Eq(), read)
			}).
			Set(cols.Items.Set(), entity.Items).
			Set(cols.UpdatedByID.Set(), entity.UpdatedByID).
			Set(cols.Version.Set(), entity.Version).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("update checklist template: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("update checklist template: %w", err)
		}
		if affected == 0 {
			return nil, repositories.ErrCaseChecklistTemplateStale
		}

		return entity, nil
	})
}

func (r *repository) DeleteTemplate(
	ctx context.Context,
	req *repositories.GetCaseChecklistTemplateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.ChecklistTemplateColumns
		res, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*deskcase.ChecklistTemplate)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.ChecklistTemplateScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("delete checklist template: %w", err)
		}

		return dberror.CheckRowsAffected(res, templateEntity, req.ID.String())
	})
}

func (r *repository) CountDocumentTypes(
	ctx context.Context,
	req *repositories.CountCaseChecklistDocumentTypesRequest,
) (int, error) {
	if len(req.IDs) == 0 {
		return 0, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := buncolgen.DocumentTypeColumns
		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*documenttype.DocumentType)(nil)).
			Apply(buncolgen.DocumentTypeApplyTenant(req.TenantInfo)).
			Where(cols.ID.In(), bun.List(req.IDs)).
			Count(ctx)
		if err != nil {
			return 0, fmt.Errorf("count document types: %w", err)
		}

		return count, nil
	})
}

func (r *repository) ListTicks(
	ctx context.Context,
	req *repositories.ListCaseChecklistTicksRequest,
) ([]*deskcase.ChecklistTick, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*deskcase.ChecklistTick, error) {
		cols := buncolgen.ChecklistTickColumns
		usr := buncolgen.UserColumns
		items := make([]*deskcase.ChecklistTick, 0)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			ColumnExpr(buncolgen.ChecklistTickTable.Alias+".*").
			ColumnExpr("COALESCE("+usr.Name.Qualified()+", '') AS ticked_by_name").
			Join("LEFT JOIN "+buncolgen.UserTable.Name+" AS "+buncolgen.UserTable.Alias).
			JoinOn(usr.ID.EqColumn(cols.TickedByID)).
			Apply(buncolgen.ChecklistTickApplyTenant(req.TenantInfo)).
			Where(cols.SubjectType.Eq(), req.SubjectType).
			Where(cols.SubjectID.Eq(), req.SubjectID).
			Limit(maxTemplateRows).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list checklist ticks: %w", err)
		}

		return items, nil
	})
}

func (r *repository) Tick(ctx context.Context, entity *deskcase.ChecklistTick) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.ChecklistTickColumns
		_, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On("CONFLICT (" + cols.OrganizationID.Bare() + ", " + cols.BusinessUnitID.Bare() +
				", " + cols.SubjectType.Bare() + ", " + cols.SubjectID.Bare() + ", " +
				cols.ItemKey.Bare() + ") DO NOTHING").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("tick checklist step: %w", err)
		}

		return nil
	})
}

func (r *repository) Untick(ctx context.Context, req *repositories.UntickCaseChecklistItemRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.ChecklistTickColumns
		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*deskcase.ChecklistTick)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.ChecklistTickScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.SubjectType.Eq(), req.SubjectType).
					Where(cols.SubjectID.Eq(), req.SubjectID).
					Where(cols.ItemKey.Eq(), req.ItemKey)
			}).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("untick checklist step: %w", err)
		}

		return nil
	})
}
