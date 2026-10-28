package workerchecklistrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist/workerchecklistdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerchecklistports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	templateBusyMessage  = "Checklist template is busy. Retry the request."
	checklistBusyMessage = "Checklist is busy. Retry the request."
)

var templateErrors = dberror.Mapping{Entity: "WorkerChecklistTemplate"}

var checklistErrors = dberror.Mapping{Entity: "WorkerChecklist"}

var itemErrors = dberror.Mapping{Entity: "WorkerChecklistItem"}

type templateCount struct {
	TemplateID pulid.ID
	Count      int64
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerchecklistports.WorkerChecklistRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-checklist-repository"),
	}
}

func templateItems() keel.Loader {
	it := workerchecklistdb.WorkerChecklistTemplates.Items()

	return it.Order(it.SortOrder.Asc()).Load(it.CredentialType(), it.DocumentType())
}

func checklistItems() keel.Loader {
	it := workerchecklistdb.WorkerChecklists.Items()

	return it.Order(it.SortOrder.Asc()).Load(
		it.CompletedBy(),
		it.EvidenceDocument(),
		it.EvidenceCredential(),
		it.CredentialType(),
	)
}

func templateFilter(req *workerchecklistports.ListChecklistTemplatesRequest) []keel.Pred {
	t := workerchecklistdb.WorkerChecklistTemplates

	return []keel.Pred{
		keel.When(req.Status != "", t.Status.Eq(domaintypes.Status(req.Status))),
		keel.When(req.Kind != "", t.Kind.Eq(workerchecklist.ChecklistKind(req.Kind))),
		keel.When(req.Trigger != "", t.Trigger.Eq(workerchecklist.ChecklistTrigger(req.Trigger))),
	}
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *workerchecklistports.ListChecklistTemplatesRequest,
) (*pagination.CursorListResult[*workerchecklist.WorkerChecklistTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*workerchecklist.WorkerChecklistTemplate], error) {
			t := workerchecklistdb.WorkerChecklistTemplates
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				t.Where(templateFilter(req)...),
				&listquery.Request[workerchecklist.WorkerChecklistTemplate]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*workerchecklist.WorkerChecklistTemplate],
					) *keel.Query[*workerchecklist.WorkerChecklistTemplate] {
						return q.Load(listquery.When(req.IncludeItems, templateItems())...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list checklist templates", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListActiveTemplates(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*workerchecklist.WorkerChecklistTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerchecklist.WorkerChecklistTemplate, error) {
			t := workerchecklistdb.WorkerChecklistTemplates
			entities, err := t.Where(t.Status.Eq(domaintypes.StatusActive)).
				Load(templateItems()).
				Order(t.Name.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list active checklist templates", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) GetTemplateByID(
	ctx context.Context,
	req *workerchecklistports.GetChecklistTemplateByIDRequest,
) (*workerchecklist.WorkerChecklistTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workerchecklist.WorkerChecklistTemplate, error) {
			t := workerchecklistdb.WorkerChecklistTemplates
			entity, err := t.Where(t.ID.Eq(req.ID)).
				Load(listquery.When(req.IncludeItems, templateItems())...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, templateErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetDefaultTemplate(
	ctx context.Context,
	req *workerchecklistports.GetDefaultChecklistTemplateRequest,
) (*workerchecklist.WorkerChecklistTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workerchecklist.WorkerChecklistTemplate, error) {
			t := workerchecklistdb.WorkerChecklistTemplates
			entity, err := t.Where(
				t.Trigger.Eq(req.Trigger),
				t.IsDefault.IsTrue(),
				t.Status.Eq(domaintypes.StatusActive),
			).
				Load(templateItems()).
				Order(t.CreatedAt.Asc()).
				First(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, templateErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *repository) TemplateCodeExists(
	ctx context.Context,
	req *workerchecklistports.ChecklistTemplateCodeExistsRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		t := workerchecklistdb.WorkerChecklistTemplates
		exists, err := t.Where(
			t.Code.Lower().Eq(strings.ToLower(req.Code)),
			keel.When(req.ExcludeID.IsNotNil(), t.ID.Ne(req.ExcludeID)),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check checklist template code", zap.Error(err))
			return false, err
		}

		return exists, nil
	})
}

func stampTemplateItems(
	entity *workerchecklist.WorkerChecklistTemplate,
	resetIDs bool,
) []*workerchecklist.WorkerChecklistTemplateItem {
	items := make([]*workerchecklist.WorkerChecklistTemplateItem, 0, len(entity.Items))
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
		items = append(items, item)
	}

	return items
}

func (r *repository) insertTemplateItems(
	ctx context.Context,
	items []*workerchecklist.WorkerChecklistTemplateItem,
) error {
	if len(items) == 0 {
		return nil
	}

	return workerchecklistdb.WorkerChecklistTemplateItems.CreateMany(ctx, r.db.Keel(ctx), items)
}

func (r *repository) CreateTemplate(
	ctx context.Context,
	entity *workerchecklist.WorkerChecklistTemplate,
) (*workerchecklist.WorkerChecklistTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerchecklist.WorkerChecklistTemplate, error) {
			if err := r.createTemplate(ctx, entity); err != nil {
				r.l.Error("failed to create checklist template", zap.Error(err))
				return nil, dberror.MapRetryableTransactionError(
					templateErrors.Map(err, entity.ID.String()),
					templateBusyMessage,
				)
			}

			return entity, nil
		},
	)
}

func (r *repository) createTemplate(
	ctx context.Context,
	entity *workerchecklist.WorkerChecklistTemplate,
) error {
	if entity.IsDefault {
		err := r.ClearDefaultTemplate(
			ctx,
			&workerchecklistports.ClearDefaultChecklistTemplateRequest{
				Trigger: entity.Trigger,
			},
		)
		if err != nil {
			return err
		}
	}

	if err := workerchecklistdb.WorkerChecklistTemplates.Create(ctx, r.db.Keel(ctx), entity); err != nil {
		return err
	}

	return r.insertTemplateItems(ctx, stampTemplateItems(entity, false))
}

func (r *repository) UpdateTemplate(
	ctx context.Context,
	entity *workerchecklist.WorkerChecklistTemplate,
) (*workerchecklist.WorkerChecklistTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerchecklist.WorkerChecklistTemplate, error) {
			if err := r.updateTemplate(ctx, entity); err != nil {
				r.l.Error("failed to update checklist template", zap.Error(err))
				return nil, dberror.MapRetryableTransactionError(
					templateErrors.Map(err, entity.ID.String()),
					templateBusyMessage,
				)
			}

			return entity, nil
		},
	)
}

func (r *repository) updateTemplate(
	ctx context.Context,
	entity *workerchecklist.WorkerChecklistTemplate,
) error {
	if entity.IsDefault {
		err := r.ClearDefaultTemplate(
			ctx,
			&workerchecklistports.ClearDefaultChecklistTemplateRequest{
				Trigger:  entity.Trigger,
				ExceptID: entity.ID,
			},
		)
		if err != nil {
			return err
		}
	}

	db := r.db.Keel(ctx)
	if err := workerchecklistdb.WorkerChecklistTemplates.Save(ctx, db, entity); err != nil {
		return err
	}

	it := workerchecklistdb.WorkerChecklistTemplateItems
	if _, err := it.Where(it.TemplateID.Eq(entity.ID)).Delete(ctx, db); err != nil {
		return err
	}

	return r.insertTemplateItems(ctx, stampTemplateItems(entity, true))
}

func (r *repository) ClearDefaultTemplate(
	ctx context.Context,
	req *workerchecklistports.ClearDefaultChecklistTemplateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		t := workerchecklistdb.WorkerChecklistTemplates
		_, err := t.Where(
			t.Trigger.Eq(req.Trigger),
			t.IsDefault.IsTrue(),
			keel.When(req.ExceptID.IsNotNil(), t.ID.Ne(req.ExceptID)),
		).Update(ctx, r.db.Keel(ctx), t.IsDefault.To(false))
		if err != nil {
			r.l.Error("failed to clear default checklist template", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *repository) CountOpenChecklists(
	ctx context.Context,
	req *workerchecklistports.CountOpenChecklistsRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		c := workerchecklistdb.WorkerChecklists
		count, err := c.Where(
			c.Status.Eq(workerchecklist.ChecklistStatusOpen),
			keel.When(req.WorkerID.IsNotNil(), c.WorkerID.Eq(req.WorkerID)),
			keel.When(req.TemplateID.IsNotNil(), c.TemplateID.Eq(req.TemplateID)),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count open checklists", zap.Error(err))
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) CountOpenChecklistsByTemplateIDs(
	ctx context.Context,
	req *workerchecklistports.CountOpenChecklistsByTemplateIDsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		if len(req.TemplateIDs) == 0 {
			return map[pulid.ID]int{}, nil
		}

		c := workerchecklistdb.WorkerChecklists
		rows, err := keel.Into[templateCount](ctx, r.db.Keel(ctx), c.
			Group(c.TemplateID).
			Pick(c.TemplateID, keel.Count()).
			Where(c.Status.Eq(workerchecklist.ChecklistStatusOpen), c.TemplateID.In(req.TemplateIDs...)),
		)
		if err != nil {
			r.l.Error("failed to count open checklists by template", zap.Error(err))
			return nil, err
		}

		counts := make(map[pulid.ID]int, len(rows))
		for _, row := range rows {
			counts[row.TemplateID] = int(row.Count)
		}

		return counts, nil
	})
}

func (r *repository) ListForWorker(
	ctx context.Context,
	req *workerchecklistports.ListWorkerChecklistsRequest,
) ([]*workerchecklist.WorkerChecklist, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerchecklist.WorkerChecklist, error) {
			c := workerchecklistdb.WorkerChecklists
			entities, err := c.Where(
				c.WorkerID.Eq(req.WorkerID),
				keel.When(!req.IncludeClosed, c.Status.Eq(workerchecklist.ChecklistStatusOpen)),
			).
				Load(c.StartedBy()).
				Load(listquery.When(req.IncludeItems, checklistItems())...).
				Order(c.Status.Asc(), c.StartedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list worker checklists", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *workerchecklistports.GetWorkerChecklistByIDRequest,
) (*workerchecklist.WorkerChecklist, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workerchecklist.WorkerChecklist, error) {
			c := workerchecklistdb.WorkerChecklists
			entity, err := c.Where(c.ID.Eq(req.ID)).
				Load(c.StartedBy()).
				Load(listquery.When(req.IncludeItems, checklistItems())...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, checklistErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetItemByID(
	ctx context.Context,
	req *workerchecklistports.GetWorkerChecklistItemByIDRequest,
) (*workerchecklist.WorkerChecklistItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workerchecklist.WorkerChecklistItem, error) {
			entity, err := workerchecklistdb.WorkerChecklistItems.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, itemErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *workerchecklist.WorkerChecklist,
) (*workerchecklist.WorkerChecklist, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerchecklist.WorkerChecklist, error) {
			if err := r.create(ctx, entity); err != nil {
				r.l.Error("failed to create worker checklist", zap.Error(err))
				return nil, dberror.MapRetryableTransactionError(
					checklistErrors.Map(err, entity.ID.String()),
					checklistBusyMessage,
				)
			}

			return entity, nil
		},
	)
}

func (r *repository) create(ctx context.Context, entity *workerchecklist.WorkerChecklist) error {
	db := r.db.Keel(ctx)
	if err := workerchecklistdb.WorkerChecklists.Create(ctx, db, entity); err != nil {
		return err
	}

	if len(entity.Items) == 0 {
		return nil
	}

	for i, item := range entity.Items {
		item.ChecklistID = entity.ID
		item.OrganizationID = entity.OrganizationID
		item.BusinessUnitID = entity.BusinessUnitID
		item.SortOrder = int32(i)
	}

	return workerchecklistdb.WorkerChecklistItems.CreateMany(ctx, db, entity.Items)
}

func (r *repository) Update(
	ctx context.Context,
	entity *workerchecklist.WorkerChecklist,
) (*workerchecklist.WorkerChecklist, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerchecklist.WorkerChecklist, error) {
			if err := workerchecklistdb.WorkerChecklists.Save(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error("failed to update worker checklist", zap.Error(err))
				return nil, checklistErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateItems(
	ctx context.Context,
	items []*workerchecklist.WorkerChecklistItem,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		for _, item := range items {
			if err := workerchecklistdb.WorkerChecklistItems.Save(ctx, db, item); err != nil {
				return itemErrors.Map(err, item.ID.String())
			}
		}

		return nil
	})
}
