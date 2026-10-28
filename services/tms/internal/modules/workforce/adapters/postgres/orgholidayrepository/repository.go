package orgholidayrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto/ptodb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var holidayErrors = dberror.Mapping{Entity: "OrgHoliday"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerports.OrgHolidayRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.org-holiday-repository"),
	}
}

func listQuery(req *workerports.ListOrgHolidaysRequest) *keel.Query[*pto.OrgHoliday] {
	h := ptodb.OrgHolidays
	var window keel.Pred
	if req.From > 0 || req.To > 0 {
		window = keel.Or(
			h.RecursAnnually.IsTrue(),
			keel.And(
				keel.When(req.From > 0, h.HolidayDate.Gte(req.From)),
				keel.When(req.To > 0, h.HolidayDate.Lte(req.To)),
			),
		)
	}

	return h.Where(window).Order(h.HolidayDate.Asc(), h.Name.Asc())
}

func (r *repository) List(
	ctx context.Context,
	req *workerports.ListOrgHolidaysRequest,
) ([]*pto.OrgHoliday, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*pto.OrgHoliday, error) {
		entities, err := listQuery(req).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list org holidays", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req *workerports.GetOrgHolidayByIDRequest,
) (*pto.OrgHoliday, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*pto.OrgHoliday, error) {
		entity, err := ptodb.OrgHolidays.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, holidayErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *pto.OrgHoliday,
) (*pto.OrgHoliday, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pto.OrgHoliday, error) {
		if err := ptodb.OrgHolidays.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create org holiday", zap.Error(err))
			return nil, holidayErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *pto.OrgHoliday,
) (*pto.OrgHoliday, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pto.OrgHoliday, error) {
		if err := ptodb.OrgHolidays.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update org holiday", zap.Error(err))
			return nil, holidayErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Delete(ctx context.Context, req *workerports.GetOrgHolidayByIDRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := ptodb.OrgHolidays.DeleteRow(ctx, r.db.Keel(ctx), &pto.OrgHoliday{ID: req.ID})
		if err != nil {
			r.l.Error("failed to delete org holiday", zap.Error(err))
			return holidayErrors.Map(err, req.ID.String())
		}

		return nil
	})
}
