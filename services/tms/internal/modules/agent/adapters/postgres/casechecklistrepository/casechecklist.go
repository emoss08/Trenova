package casechecklistrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/deskcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/deskcase/deskcasedb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxTemplateRows = 1000

var templateErrors = dberror.Mapping{Entity: "Checklist template"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) deskcaseports.CaseChecklistRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.case-checklist-repository")}
}

func withCustomerName() *keel.Query[deskcase.ChecklistTemplate] {
	t := deskcasedb.ChecklistTemplates
	cus := customerdb.Customers.As("cus")

	return t.LeftJoin(cus, cus.ID.EqCol(t.CustomerID)).
		Compute(t.CustomerName, keel.DynOf(cus.Name).Coalesce(""))
}

func listTemplatesQuery(
	req *deskcaseports.ListCaseChecklistTemplatesRequest,
) *keel.Query[deskcase.ChecklistTemplate] {
	t := deskcasedb.ChecklistTemplates

	return withCustomerName().
		Where(t.Kind.Eq(req.Kind)).
		Order(t.CustomerID.Asc().NullsFirst(), t.CustomerName.Asc()).
		Limit(maxTemplateRows)
}

func templateForQuery(
	req *deskcaseports.GetCaseChecklistTemplateForRequest,
) *keel.Query[deskcase.ChecklistTemplate] {
	t := deskcasedb.ChecklistTemplates

	return t.Where(
		t.Kind.Eq(req.Kind),
		keel.Or(
			t.CustomerID.IsNull(),
			keel.When(req.CustomerID.IsNotNil(), t.CustomerID.Eq(req.CustomerID)),
		),
	).
		Order(t.CustomerID.Asc().NullsLast()).
		Limit(1)
}

func listTicksQuery(
	req *deskcaseports.ListCaseChecklistTicksRequest,
) *keel.Query[deskcase.ChecklistTick] {
	tk := deskcasedb.ChecklistTicks
	usr := tenantdb.Users.As("usr")

	return tk.LeftJoin(usr, usr.ID.EqCol(tk.TickedByID)).
		Compute(tk.TickedByName, keel.DynOf(usr.Name).Coalesce("")).
		Where(tk.SubjectType.Eq(req.SubjectType), tk.SubjectID.Eq(req.SubjectID)).
		Limit(maxTemplateRows)
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *deskcaseports.ListCaseChecklistTemplatesRequest,
) ([]*deskcase.ChecklistTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*deskcase.ChecklistTemplate, error) {
		items, err := listTemplatesQuery(req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list checklist templates: %w", err)
		}

		return items, nil
	})
}

func (r *repository) TemplateFor(
	ctx context.Context,
	req *deskcaseports.GetCaseChecklistTemplateForRequest,
) (*deskcase.ChecklistTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*deskcase.ChecklistTemplate, error) {
		items, err := templateForQuery(req).List(ctx, r.db.Keel(ctx))
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
	req *deskcaseports.GetCaseChecklistTemplateRequest,
) (*deskcase.ChecklistTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*deskcase.ChecklistTemplate, error) {
		t := deskcasedb.ChecklistTemplates
		entity, err := withCustomerName().Where(t.ID.Eq(req.ID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, templateErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) SaveTemplate(
	ctx context.Context,
	entity *deskcase.ChecklistTemplate,
) (*deskcase.ChecklistTemplate, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*deskcase.ChecklistTemplate, error) {
		t := deskcasedb.ChecklistTemplates
		db := r.db.Keel(ctx)
		if entity.ID.IsNil() {
			entity.Version = 0
			if err := t.Create(ctx, db, entity); err != nil {
				if dberror.IsUniqueConstraintViolation(err) {
					return nil, deskcaseports.ErrCaseChecklistTemplateStale
				}
				return nil, fmt.Errorf("insert checklist template: %w", err)
			}
			return entity, nil
		}

		if err := t.Save(ctx, db, entity, t.Items, t.UpdatedByID); err != nil {
			if errors.Is(err, keel.ErrConflict) || errors.Is(err, keel.ErrNotFound) {
				return nil, deskcaseports.ErrCaseChecklistTemplateStale
			}
			return nil, fmt.Errorf("update checklist template: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) DeleteTemplate(
	ctx context.Context,
	req *deskcaseports.GetCaseChecklistTemplateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		t := deskcasedb.ChecklistTemplates
		n, err := t.Where(t.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return fmt.Errorf("delete checklist template: %w", err)
		}
		return dberror.CheckAffected(n, templateErrors.Entity, req.ID.String())
	})
}

func (r *repository) CountDocumentTypes(
	ctx context.Context,
	req *deskcaseports.CountCaseChecklistDocumentTypesRequest,
) (int, error) {
	if len(req.IDs) == 0 {
		return 0, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		dt := documenttypedb.DocumentTypes
		count, err := dt.Where(dt.ID.In(req.IDs...)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count document types: %w", err)
		}

		return int(count), nil
	})
}

func (r *repository) ListTicks(
	ctx context.Context,
	req *deskcaseports.ListCaseChecklistTicksRequest,
) ([]*deskcase.ChecklistTick, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*deskcase.ChecklistTick, error) {
		items, err := listTicksQuery(req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list checklist ticks: %w", err)
		}

		return items, nil
	})
}

func (r *repository) Tick(ctx context.Context, entity *deskcase.ChecklistTick) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		tk := deskcasedb.ChecklistTicks
		_, err := tk.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*deskcase.ChecklistTick{entity},
			tk.OrganizationID,
			tk.BusinessUnitID,
			tk.SubjectType,
			tk.SubjectID,
			tk.ItemKey,
		)
		if err != nil {
			return fmt.Errorf("tick checklist step: %w", err)
		}

		return nil
	})
}

func (r *repository) Untick(
	ctx context.Context,
	req *deskcaseports.UntickCaseChecklistItemRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		tk := deskcasedb.ChecklistTicks
		_, err := tk.Where(
			tk.SubjectType.Eq(req.SubjectType),
			tk.SubjectID.Eq(req.SubjectID),
			tk.ItemKey.Eq(req.ItemKey),
		).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return fmt.Errorf("untick checklist step: %w", err)
		}

		return nil
	})
}
