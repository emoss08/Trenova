package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/revenuecode"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
)

// RevenueCodeService is the service for revenue code.
type RevenueCodeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewRevenueCodeService creates a new revenue code service.
func NewRevenueCodeService(s *api.Server) *RevenueCodeService {
	return &RevenueCodeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetRevenueCodes gets the revenue codes for an organization.
func (r *RevenueCodeService) GetRevenueCodes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.RevenueCode, int, error) {
	entityCount, countErr := r.Client.RevenueCode.Query().Where(
		revenuecode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.RevenueCode.Query().
		Limit(limit).
		Offset(offset).
		WithExpenseAccount().
		WithRevenueAccount().
		Where(
			revenuecode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateRevenueCode creates a new revenue code.
func (r *RevenueCodeService) CreateRevenueCode(
	ctx context.Context, entity *ent.RevenueCode,
) (*ent.RevenueCode, error) {
	updatedEntity := new(ent.RevenueCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createRevenueCodeEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *RevenueCodeService) createRevenueCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.RevenueCode,
) (*ent.RevenueCode, error) {
	createdEntity, err := tx.RevenueCode.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetNillableExpenseAccountID(entity.ExpenseAccountID).
		SetNillableRevenueAccountID(entity.RevenueAccountID).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create email profile")
	}

	return createdEntity, nil
}

// UpdateRevenueCode updates a revenue code.
func (r *RevenueCodeService) UpdateRevenueCode(
	ctx context.Context, entity *ent.RevenueCode,
) (*ent.RevenueCode, error) {
	updatedEntity := new(ent.RevenueCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateRevenueCodeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *RevenueCodeService) updateRevenueCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.RevenueCode,
) (*ent.RevenueCode, error) {
	current, err := tx.RevenueCode.Get(ctx, entity.ID)
	if err != nil {
		return nil, eris.Wrap(err, "failed to retrieve requested entity")
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"name")
	}

	// Start building the update operation
	updateOp := tx.RevenueCode.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetNillableExpenseAccountID(entity.ExpenseAccountID).
		SetNillableRevenueAccountID(entity.RevenueAccountID).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
