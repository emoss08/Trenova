package carrierassignmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var carrierAssignmentErrors = dberror.Mapping{Entity: "Carrier assignment"}

var accessorialErrors = dberror.Mapping{Entity: "Carrier assignment accessorial"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) shipmentports.CarrierAssignmentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-assignment-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req *shipmentports.GetCarrierAssignmentByIDRequest,
) (*shipment.CarrierAssignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipment.CarrierAssignment, error) {
		ca := shipmentdb.CarrierAssignments
		entity, err := ca.Where(ca.ID.Eq(req.CarrierAssignmentID)).
			Load(ca.Carrier(), ca.Accessorials(), ca.ShipmentMove()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get carrier assignment", zap.Error(err))
			return nil, carrierAssignmentErrors.Map(err, req.CarrierAssignmentID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetActiveByMoveID(
	ctx context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.CarrierAssignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipment.CarrierAssignment, error) {
		ca := shipmentdb.CarrierAssignments
		entity, err := ca.Where(
			ca.ShipmentMoveID.Eq(moveID),
			ca.Status.Ne(shipment.CarrierAssignmentStatusCanceled),
		).
			Load(ca.Carrier(), ca.Accessorials()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // nil result represents an optional absence in this API
			}
			r.l.Error("failed to get carrier assignment by move", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) FindForMatching(
	ctx context.Context,
	req *shipmentports.FindCarrierAssignmentForMatchingRequest,
) (*shipment.CarrierAssignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipment.CarrierAssignment, error) {
		if req.ProNumber == "" && req.ShipmentID.IsNil() {
			return nil, nil //nolint:nilnil // nothing to match without a reference
		}

		ca := shipmentdb.CarrierAssignments
		reference := ca.ShipmentMove().ShipmentID.Eq(req.ShipmentID)
		if req.ProNumber != "" {
			reference = ca.ProNumber.Eq(req.ProNumber)
		}

		entity, err := ca.Where(
			ca.Status.Ne(shipment.CarrierAssignmentStatusCanceled),
			keel.When(!req.CarrierID.IsNil(), ca.CarrierID.Eq(req.CarrierID)),
			reference,
		).
			Load(ca.Carrier(), ca.Accessorials(), ca.ShipmentMove()).
			Order(ca.CreatedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // nil assignment represents an optional absence
			}
			r.l.Error("failed to find carrier assignment for matching", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipment.CarrierAssignment,
) (*shipment.CarrierAssignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.CarrierAssignment, error) {
		log := r.l.With(
			zap.String("operation", "Create"),
			zap.String("shipmentMoveId", entity.ShipmentMoveID.String()),
		)

		db := r.db.Keel(ctx)
		if err := shipmentdb.CarrierAssignments.Create(ctx, db, entity); err != nil {
			log.Error("failed to create carrier assignment", zap.Error(err))
			return nil, carrierAssignmentErrors.Map(err, entity.ID.String())
		}

		stampAccessorials(entity)
		if err := shipmentdb.CarrierAssignmentAccessorials.CreateMany(
			ctx,
			db,
			liveAccessorials(entity),
		); err != nil {
			log.Error("failed to insert carrier assignment accessorials", zap.Error(err))
			return nil, accessorialErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipment.CarrierAssignment,
) (*shipment.CarrierAssignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.CarrierAssignment, error) {
		log := r.l.With(
			zap.String("operation", "Update"),
			zap.String("id", entity.ID.String()),
		)

		db := r.db.Keel(ctx)
		if err := shipmentdb.CarrierAssignments.Save(ctx, db, entity); err != nil {
			log.Error("failed to update carrier assignment", zap.Error(err))
			return nil, carrierAssignmentErrors.Map(err, entity.ID.String())
		}

		if err := deleteMissingAccessorials(ctx, db, entity); err != nil {
			log.Error("failed to delete carrier assignment accessorials", zap.Error(err))
			return nil, err
		}

		stampAccessorials(entity)
		if err := upsertAccessorials(ctx, db, entity); err != nil {
			log.Error("failed to upsert carrier assignment accessorials", zap.Error(err))
			return nil, accessorialErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func stampAccessorials(entity *shipment.CarrierAssignment) {
	for _, acc := range entity.Accessorials {
		if acc == nil {
			continue
		}
		acc.CarrierAssignmentID = entity.ID
		acc.OrganizationID = entity.OrganizationID
		acc.BusinessUnitID = entity.BusinessUnitID
	}
}

func liveAccessorials(entity *shipment.CarrierAssignment) []*shipment.CarrierAssignmentAccessorial {
	rows := make([]*shipment.CarrierAssignmentAccessorial, 0, len(entity.Accessorials))
	for _, acc := range entity.Accessorials {
		if acc != nil {
			rows = append(rows, acc)
		}
	}

	return rows
}

func deleteMissingAccessorials(
	ctx context.Context,
	db *keel.DB,
	entity *shipment.CarrierAssignment,
) error {
	kept := make([]pulid.ID, 0, len(entity.Accessorials))
	for _, acc := range entity.Accessorials {
		if acc != nil && !acc.ID.IsNil() {
			kept = append(kept, acc.ID)
		}
	}

	a := shipmentdb.CarrierAssignmentAccessorials
	_, err := a.Where(
		a.CarrierAssignmentID.Eq(entity.ID),
		keel.When(len(kept) > 0, a.ID.NotIn(kept...)),
	).Delete(ctx, db)

	return err
}

func upsertAccessorials(
	ctx context.Context,
	db *keel.DB,
	entity *shipment.CarrierAssignment,
) error {
	rows := liveAccessorials(entity)
	if len(rows) == 0 {
		return nil
	}

	created := make([]int64, len(rows))
	for i, row := range rows {
		created[i] = row.CreatedAt
	}

	a := shipmentdb.CarrierAssignmentAccessorials
	inserted, err := a.Upsert(
		ctx,
		db,
		rows,
		keel.OnConflict(a.ID, a.BusinessUnitID, a.OrganizationID).
			Update(a.AccessorialChargeID, a.Description, a.Amount),
	)
	if err != nil {
		return err
	}

	for i, row := range rows {
		if !inserted[i] && created[i] != 0 {
			row.CreatedAt = created[i]
		}
	}

	return nil
}
