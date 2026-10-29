package fuelsurchargerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge/fuelsurchargedb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	programBusyMessage     = "Fuel surcharge program is busy. Retry the request."
	defaultFallbackBatch   = 200
	usedFallbackDetailPath = "usedFallback"
)

var programErrors = dberror.Mapping{Entity: "FuelSurchargeProgram"}

type ProgramParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type programRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewProgramRepository(p ProgramParams) fuelsurchargeports.FuelSurchargeProgramRepository {
	return &programRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fuel-surcharge-program-repository"),
	}
}

func orderedTableRows() keel.Loader {
	rows := fuelsurchargedb.FuelSurchargePrograms.TableRows()
	return rows.Order(rows.SortOrder.Asc(), rows.PriceMin.Asc())
}

func (r *programRepository) ListConnection(
	ctx context.Context,
	req *fuelsurchargeports.ListFuelSurchargeProgramConnectionRequest,
) (*pagination.CursorListResult[*fuelsurcharge.FuelSurchargeProgram], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*fuelsurcharge.FuelSurchargeProgram], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				fuelsurchargedb.FuelSurchargePrograms.Query(),
				&listquery.Request[fuelsurcharge.FuelSurchargeProgram]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.FuelSurchargeProgramColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list fuel surcharge programs", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func activeProgramsQuery() *keel.Query[*fuelsurcharge.FuelSurchargeProgram] {
	p := fuelsurchargedb.FuelSurchargePrograms

	return p.Where(p.Status.Eq(fuelsurcharge.ProgramStatusActive)).
		Load(p.FuelIndex(), orderedTableRows()).
		Order(p.Name.Asc())
}

func (r *programRepository) ListActive(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*fuelsurcharge.FuelSurchargeProgram, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*fuelsurcharge.FuelSurchargeProgram, error) {
			entities, err := activeProgramsQuery().All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list active fuel surcharge programs", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func programQuery(
	req *fuelsurchargeports.GetFuelSurchargeProgramByIDRequest,
) *keel.Query[*fuelsurcharge.FuelSurchargeProgram] {
	p := fuelsurchargedb.FuelSurchargePrograms
	q := p.Where(p.ID.Eq(req.ProgramID))
	if req.IncludeRows {
		q = q.Load(orderedTableRows())
	}
	if req.IncludeIndex {
		q = q.Load(p.FuelIndex())
	}
	if req.IncludeCharge {
		q = q.Load(p.AccessorialCharge())
	}

	return q
}

func (r *programRepository) GetByID(
	ctx context.Context,
	req *fuelsurchargeports.GetFuelSurchargeProgramByIDRequest,
) (*fuelsurcharge.FuelSurchargeProgram, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*fuelsurcharge.FuelSurchargeProgram, error) {
			entity, err := programQuery(req).One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get fuel surcharge program",
					zap.String("id", req.ProgramID.String()),
					zap.Error(err),
				)
				return nil, programErrors.Map(err, req.ProgramID.String())
			}

			return entity, nil
		},
	)
}

func stampTableRows(entity *fuelsurcharge.FuelSurchargeProgram, resetIDs bool) {
	for _, row := range entity.TableRows {
		if row == nil {
			continue
		}

		if resetIDs {
			row.ID = pulid.Nil
		}

		row.FuelSurchargeProgramID = entity.ID
		row.OrganizationID = entity.OrganizationID
		row.BusinessUnitID = entity.BusinessUnitID
	}
}

func (r *programRepository) insertTableRows(
	ctx context.Context,
	db *keel.DB,
	entity *fuelsurcharge.FuelSurchargeProgram,
) error {
	rows := make([]*fuelsurcharge.FuelSurchargeTableRow, 0, len(entity.TableRows))
	for _, row := range entity.TableRows {
		if row != nil {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return nil
	}

	return fuelsurchargedb.FuelSurchargeTableRows.CreateMany(ctx, db, rows)
}

func mapProgramError(err error, id string) error {
	return dberror.MapRetryableTransactionError(programErrors.Map(err, id), programBusyMessage)
}

func (r *programRepository) GetByIDs(
	ctx context.Context,
	req *fuelsurchargeports.GetFuelSurchargeProgramsByIDsRequest,
) ([]*fuelsurcharge.FuelSurchargeProgram, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*fuelsurcharge.FuelSurchargeProgram, error) {
			if len(req.ProgramIDs) == 0 {
				return []*fuelsurcharge.FuelSurchargeProgram{}, nil
			}

			p := fuelsurchargedb.FuelSurchargePrograms
			entities, err := p.Where(p.ID.In(req.ProgramIDs...)).
				Limit(len(req.ProgramIDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get fuel surcharge programs", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *programRepository) Create(
	ctx context.Context,
	entity *fuelsurcharge.FuelSurchargeProgram,
) (*fuelsurcharge.FuelSurchargeProgram, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*fuelsurcharge.FuelSurchargeProgram, error) {
			db := r.db.Keel(ctx)
			err := fuelsurchargedb.FuelSurchargePrograms.Create(ctx, db, entity)
			if err == nil {
				stampTableRows(entity, false)
				err = r.insertTableRows(ctx, db, entity)
			}
			if err != nil {
				r.l.Error("failed to create fuel surcharge program", zap.Error(err))
				return nil, mapProgramError(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *programRepository) Update(
	ctx context.Context,
	entity *fuelsurcharge.FuelSurchargeProgram,
) (*fuelsurcharge.FuelSurchargeProgram, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*fuelsurcharge.FuelSurchargeProgram, error) {
			db := r.db.Keel(ctx)
			rows := fuelsurchargedb.FuelSurchargeTableRows

			err := fuelsurchargedb.FuelSurchargePrograms.Save(ctx, db, entity)
			if err == nil {
				_, err = rows.Where(rows.FuelSurchargeProgramID.Eq(entity.ID)).Delete(ctx, db)
			}
			if err == nil {
				stampTableRows(entity, true)
				err = r.insertTableRows(ctx, db, entity)
			}
			if err != nil {
				r.l.Error("failed to update fuel surcharge program",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, mapProgramError(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *programRepository) Delete(
	ctx context.Context,
	req *fuelsurchargeports.GetFuelSurchargeProgramByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := fuelsurchargedb.FuelSurchargePrograms.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&fuelsurcharge.FuelSurchargeProgram{ID: req.ProgramID},
		)
		if err != nil {
			r.l.Error("failed to delete fuel surcharge program",
				zap.String("id", req.ProgramID.String()),
				zap.Error(err),
			)
			return programErrors.Map(err, req.ProgramID.String())
		}

		return nil
	})
}

func fallbackShipmentsQuery(limit int) *keel.Query[*shipment.AdditionalCharge] {
	ac := shipmentdb.AdditionalCharges
	sp := ac.Shipment()

	return ac.Group(ac.ShipmentID).
		Pick(ac.ShipmentID).
		Where(
			ac.IsSystemGenerated.IsTrue(),
			ac.FuelSurchargeProgramID.IsNotNull(),
			ac.FuelSurchargeDetail.Text(usedFallbackDetailPath).Eq("true"),
			sp.Status.NotIn(shipmentvocab.StatusInvoiced, shipmentvocab.StatusCanceled),
		).
		Limit(limit)
}

func (r *programRepository) ListFallbackShipmentIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	limit int,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if limit <= 0 {
			limit = defaultFallbackBatch
		}

		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), fallbackShipmentsQuery(limit))
		if err != nil {
			r.l.Error("failed to list fuel surcharge fallback shipments", zap.Error(err))
			return nil, err
		}

		return ids, nil
	})
}

func programOptionsQuery() *keel.Query[*fuelsurcharge.FuelSurchargeProgram] {
	p := fuelsurchargedb.FuelSurchargePrograms

	return p.Pick(p.ID, p.Name, p.Code, p.Description, p.Method, p.Status).
		Where(p.Status.Eq(fuelsurcharge.ProgramStatusActive)).
		Order(p.Name.Asc())
}

func (r *programRepository) SelectOptions(
	ctx context.Context,
	req *fuelsurchargeports.FuelSurchargeProgramSelectOptionsRequest,
) (*pagination.ListResult[*fuelsurcharge.FuelSurchargeProgram], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*fuelsurcharge.FuelSurchargeProgram], error) {
			p := fuelsurchargedb.FuelSurchargePrograms

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				programOptionsQuery(),
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						p.Name.ContainsFold(s),
						p.Code.ContainsFold(s),
						p.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}
