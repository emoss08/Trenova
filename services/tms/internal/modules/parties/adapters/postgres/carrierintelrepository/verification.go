package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const verificationEntityName = "CarrierEquipmentVerification"

var verificationErrors = dberror.Mapping{Entity: verificationEntityName}

type verificationRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewVerificationRepository(p Params) carrierintelports.CarrierEquipmentVerificationRepository {
	return &verificationRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-equipment-verification-repository"),
	}
}

func (r *verificationRepository) Create(
	ctx context.Context,
	entity *carrierintel.CarrierEquipmentVerification,
) (*carrierintel.CarrierEquipmentVerification, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierEquipmentVerification, error) {
			err := carrierinteldb.CarrierEquipmentVerifications.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create carrier equipment verification", zap.Error(err))
				return nil, fmt.Errorf("create carrier equipment verification: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *verificationRepository) GetByID(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*carrierintel.CarrierEquipmentVerification, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierEquipmentVerification, error) {
			entity, err := carrierinteldb.CarrierEquipmentVerifications.Get(ctx, r.db.Keel(ctx), id)
			if err != nil {
				return nil, verificationErrors.Map(err, id.String())
			}

			return entity, nil
		},
	)
}

func (r *verificationRepository) ListByAssignmentIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	assignmentIDs []pulid.ID,
) ([]*carrierintel.CarrierEquipmentVerification, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierEquipmentVerification, error) {
			if len(assignmentIDs) == 0 {
				return []*carrierintel.CarrierEquipmentVerification{}, nil
			}

			v := carrierinteldb.CarrierEquipmentVerifications
			entities, err := v.Where(v.CarrierAssignmentID.In(assignmentIDs...)).
				Order(v.VerifiedAt.Desc(), v.ID.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list carrier equipment verifications", zap.Error(err))
				return nil, fmt.Errorf("list carrier equipment verifications: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *verificationRepository) Update(
	ctx context.Context,
	entity *carrierintel.CarrierEquipmentVerification,
) (*carrierintel.CarrierEquipmentVerification, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierEquipmentVerification, error) {
			err := carrierinteldb.CarrierEquipmentVerifications.Save(ctx, r.db.Keel(ctx), entity)
			if dberror.IsNotFoundError(err) {
				return nil, dberror.CreateVersionMismatchError(
					verificationEntityName,
					entity.ID.String(),
				)
			}
			if err != nil {
				r.l.Error("failed to update carrier equipment verification", zap.Error(err))
				return nil, fmt.Errorf("update carrier equipment verification: %w", err)
			}

			return entity, nil
		},
	)
}
