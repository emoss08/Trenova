package workerchecklistrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.WorkerChecklistRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-checklist-repository"),
	}
}

func orderTemplateItems(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.WorkerChecklistTemplateItemColumns
	return sq.Order(cols.SortOrder.OrderAsc()).
		Relation(buncolgen.WorkerChecklistTemplateItemRelations.CredentialType).
		Relation(buncolgen.WorkerChecklistTemplateItemRelations.DocumentType)
}

func orderChecklistItems(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.WorkerChecklistItemColumns
	return sq.Order(cols.SortOrder.OrderAsc()).
		Relation(buncolgen.WorkerChecklistItemRelations.CompletedBy).
		Relation(buncolgen.WorkerChecklistItemRelations.EvidenceDocument).
		Relation(buncolgen.WorkerChecklistItemRelations.EvidenceCredential).
		Relation(buncolgen.WorkerChecklistItemRelations.CredentialType)
}

func (r *repository) applyTemplateFilters(
	q *bun.SelectQuery,
	req *repositories.ListChecklistTemplatesRequest,
) *bun.SelectQuery {
	cols := buncolgen.WorkerChecklistTemplateColumns
	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}
	if req.Kind != "" {
		q = q.Where(cols.Kind.Eq(), req.Kind)
	}
	if req.Trigger != "" {
		q = q.Where(cols.Trigger.Eq(), req.Trigger)
	}
	return q
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *repositories.ListChecklistTemplatesRequest,
) (*pagination.CursorListResult[*worker.WorkerChecklistTemplate], error) {
	log := r.l.With(zap.String("operation", "ListTemplates"))

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*worker.WorkerChecklistTemplate)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.WorkerChecklistTemplateTable.Alias,
					req.Filter,
					(*worker.WorkerChecklistTemplate)(nil),
				)
				return r.applyTemplateFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count checklist templates", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*worker.WorkerChecklistTemplate]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*worker.WorkerChecklistTemplate) *bun.SelectQuery {
				return dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.WorkerChecklistTemplateTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.WorkerChecklistTemplateTable.Alias,
					req.Filter,
					req.Cursor,
					(*worker.WorkerChecklistTemplate)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				sq = r.applyTemplateFilters(sq, req)
				if req.IncludeItems {
					sq = sq.Relation(
						buncolgen.WorkerChecklistTemplateRelations.Items,
						orderTemplateItems,
					)
				}
				return sq, nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list checklist templates", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) ListActiveTemplates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.WorkerChecklistTemplate, error) {
	cols := buncolgen.WorkerChecklistTemplateColumns
	entities := make([]*worker.WorkerChecklistTemplate, 0, 8)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.WorkerChecklistTemplateRelations.Items, orderTemplateItems).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerChecklistTemplateScopeTenant(sq, tenantInfo).
				Where(cols.Status.Eq(), domaintypes.StatusActive)
		}).
		Order(cols.Name.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list active checklist templates", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetTemplateByID(
	ctx context.Context,
	req *repositories.GetChecklistTemplateByIDRequest,
) (*worker.WorkerChecklistTemplate, error) {
	entity := new(worker.WorkerChecklistTemplate)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerChecklistTemplateScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerChecklistTemplateColumns.ID.Eq(), req.ID)
		})
	if req.IncludeItems {
		q = q.Relation(buncolgen.WorkerChecklistTemplateRelations.Items, orderTemplateItems)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerChecklistTemplate")
	}

	return entity, nil
}

func (r *repository) GetDefaultTemplate(
	ctx context.Context,
	req *repositories.GetDefaultChecklistTemplateRequest,
) (*worker.WorkerChecklistTemplate, error) {
	cols := buncolgen.WorkerChecklistTemplateColumns
	entity := new(worker.WorkerChecklistTemplate)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.WorkerChecklistTemplateRelations.Items, orderTemplateItems).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerChecklistTemplateScopeTenant(sq, req.TenantInfo).
				Where(cols.Trigger.Eq(), req.Trigger).
				Where(cols.IsDefault.Eq(), true).
				Where(cols.Status.Eq(), domaintypes.StatusActive)
		}).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerChecklistTemplate")
	}

	return entity, nil
}

func (r *repository) TemplateCodeExists(
	ctx context.Context,
	req *repositories.ChecklistTemplateCodeExistsRequest,
) (bool, error) {
	cols := buncolgen.WorkerChecklistTemplateColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerChecklistTemplate)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerChecklistTemplateScopeTenant(sq, req.TenantInfo).
				Where("LOWER("+cols.Code.String()+") = ?", strings.ToLower(req.Code))
		})
	if !req.ExcludeID.IsNil() {
		q = q.Where(cols.ID.Ne(), req.ExcludeID)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		r.l.Error("failed to check checklist template code", zap.Error(err))
		return false, err
	}

	return exists, nil
}

func stampTemplateItems(entity *worker.WorkerChecklistTemplate, resetIDs bool) {
	for i, item := range entity.Items {
		if item == nil {
			continue
		}
		if resetIDs {
			item.ID = pulid.Nil
		}
		item.TemplateID = entity.ID
		item.OrganizationID = entity.OrganizationID
		item.BusinessUnitID = entity.BusinessUnitID
		item.SortOrder = int32(i) //nolint:gosec // item counts are tiny
	}
}

func (r *repository) insertTemplateItems(
	ctx context.Context,
	entity *worker.WorkerChecklistTemplate,
) error {
	if len(entity.Items) == 0 {
		return nil
	}
	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Items).
		Returning("*").
		Exec(ctx)
	return err
}

func (r *repository) CreateTemplate(
	ctx context.Context,
	entity *worker.WorkerChecklistTemplate,
) (*worker.WorkerChecklistTemplate, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if entity.IsDefault {
			if cErr := r.ClearDefaultTemplate(c, &repositories.ClearDefaultChecklistTemplateRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				Trigger: entity.Trigger,
			}); cErr != nil {
				return cErr
			}
		}
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}
		stampTemplateItems(entity, false)
		return r.insertTemplateItems(c, entity)
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"code",
				errortypes.ErrDuplicate,
				"A checklist template with this code already exists",
			)
		}
		r.l.Error("failed to create checklist template", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Checklist template is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) UpdateTemplate(
	ctx context.Context,
	entity *worker.WorkerChecklistTemplate,
) (*worker.WorkerChecklistTemplate, error) {
	ov := entity.Version
	entity.Version++
	itemCols := buncolgen.WorkerChecklistTemplateItemColumns
	tenant := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if entity.IsDefault {
			if cErr := r.ClearDefaultTemplate(c, &repositories.ClearDefaultChecklistTemplateRequest{
				TenantInfo: tenant,
				Trigger:    entity.Trigger,
				ExceptID:   entity.ID,
			}); cErr != nil {
				return cErr
			}
		}

		results, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(buncolgen.WorkerChecklistTemplateColumns.Version.Eq(), ov).
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}
		if uErr = dberror.CheckRowsAffected(results, "WorkerChecklistTemplate", entity.ID.String()); uErr != nil {
			return uErr
		}

		if _, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*worker.WorkerChecklistTemplateItem)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.WorkerChecklistTemplateItemScopeTenantDelete(dq, tenant).
					Where(itemCols.TemplateID.Eq(), entity.ID)
			}).
			Exec(c); dErr != nil {
			return dErr
		}

		stampTemplateItems(entity, true)
		return r.insertTemplateItems(c, entity)
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"code",
				errortypes.ErrDuplicate,
				"A checklist template with this code already exists",
			)
		}
		r.l.Error("failed to update checklist template", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Checklist template is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) ClearDefaultTemplate(
	ctx context.Context,
	req *repositories.ClearDefaultChecklistTemplateRequest,
) error {
	cols := buncolgen.WorkerChecklistTemplateColumns
	q := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*worker.WorkerChecklistTemplate)(nil)).
		Set(cols.IsDefault.Set(), false).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.WorkerChecklistTemplateScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.Trigger.Eq(), req.Trigger).
				Where(cols.IsDefault.Eq(), true)
		})
	if !req.ExceptID.IsNil() {
		q = q.Where(cols.ID.Ne(), req.ExceptID)
	}

	if _, err := q.Exec(ctx); err != nil {
		r.l.Error("failed to clear default checklist template", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) CountOpenChecklists(
	ctx context.Context,
	req *repositories.CountOpenChecklistsRequest,
) (int, error) {
	cols := buncolgen.WorkerChecklistColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerChecklist)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerChecklistScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), worker.ChecklistStatusOpen)
		})
	if !req.WorkerID.IsNil() {
		q = q.Where(cols.WorkerID.Eq(), req.WorkerID)
	}
	if !req.TemplateID.IsNil() {
		q = q.Where(cols.TemplateID.Eq(), req.TemplateID)
	}

	count, err := q.Count(ctx)
	if err != nil {
		r.l.Error("failed to count open checklists", zap.Error(err))
		return 0, err
	}

	return count, nil
}

func (r *repository) CountOpenChecklistsByTemplateIDs(
	ctx context.Context,
	req *repositories.CountOpenChecklistsByTemplateIDsRequest,
) (map[pulid.ID]int, error) {
	if len(req.TemplateIDs) == 0 {
		return map[pulid.ID]int{}, nil
	}

	cols := buncolgen.WorkerChecklistColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerChecklist)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerChecklistScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), worker.ChecklistStatusOpen).
				Where(cols.TemplateID.In(), bun.In(req.TemplateIDs))
		})

	counts, err := dbhelper.CountByID(ctx, q, cols.TemplateID, len(req.TemplateIDs))
	if err != nil {
		r.l.Error("failed to count open checklists by template", zap.Error(err))
		return nil, err
	}

	return counts, nil
}

func (r *repository) ListForWorker(
	ctx context.Context,
	req *repositories.ListWorkerChecklistsRequest,
) ([]*worker.WorkerChecklist, error) {
	cols := buncolgen.WorkerChecklistColumns
	entities := make([]*worker.WorkerChecklist, 0, 4)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.WorkerChecklistRelations.StartedBy).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerChecklistScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
			if !req.IncludeClosed {
				sq = sq.Where(cols.Status.Eq(), worker.ChecklistStatusOpen)
			}
			return sq
		}).
		Order(cols.Status.OrderAsc()).
		Order(cols.StartedAt.OrderDesc())
	if req.IncludeItems {
		q = q.Relation(buncolgen.WorkerChecklistRelations.Items, orderChecklistItems)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list worker checklists", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetWorkerChecklistByIDRequest,
) (*worker.WorkerChecklist, error) {
	entity := new(worker.WorkerChecklist)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.WorkerChecklistRelations.StartedBy).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerChecklistScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerChecklistColumns.ID.Eq(), req.ID)
		})
	if req.IncludeItems {
		q = q.Relation(buncolgen.WorkerChecklistRelations.Items, orderChecklistItems)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerChecklist")
	}

	return entity, nil
}

func (r *repository) GetItemByID(
	ctx context.Context,
	req *repositories.GetWorkerChecklistItemByIDRequest,
) (*worker.WorkerChecklistItem, error) {
	entity := new(worker.WorkerChecklistItem)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerChecklistItemScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerChecklistItemColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerChecklistItem")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *worker.WorkerChecklist,
) (*worker.WorkerChecklist, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}
		if len(entity.Items) == 0 {
			return nil
		}
		for i, item := range entity.Items {
			item.ChecklistID = entity.ID
			item.OrganizationID = entity.OrganizationID
			item.BusinessUnitID = entity.BusinessUnitID
			item.SortOrder = int32(i) //nolint:gosec // item counts are tiny
		}
		_, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(&entity.Items).
			Returning("*").
			Exec(c)
		return iErr
	})
	if err != nil {
		r.l.Error("failed to create worker checklist", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Checklist is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.WorkerChecklist,
) (*worker.WorkerChecklist, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerChecklistColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update worker checklist", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "WorkerChecklist", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateItems(ctx context.Context, items []*worker.WorkerChecklistItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		for _, item := range items {
			ov := item.Version
			item.Version++
			results, err := r.db.DBForContext(c).
				NewUpdate().
				Model(item).
				WherePK().
				Where(buncolgen.WorkerChecklistItemColumns.Version.Eq(), ov).
				Returning("*").
				Exec(c)
			if err != nil {
				return err
			}
			if err = dberror.CheckRowsAffected(results, "WorkerChecklistItem", item.ID.String()); err != nil {
				return err
			}
		}
		return nil
	})
}
