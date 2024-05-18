package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/divisioncode"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

type DivisionCodeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewDivisionCodeService creates a new division code service.
func NewDivisionCodeService(s *api.Server) *DivisionCodeService {
	return &DivisionCodeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetDivisionCodes gets the division codes for an organization.
func (r *DivisionCodeService) GetDivisionCodes(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.DivisionCode, int, error) {
	entityCount, countErr := r.Client.DivisionCode.Query().Where(
		divisioncode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.DivisionCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			divisioncode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateDivisionCode creates a new division code.
func (r *DivisionCodeService) CreateDivisionCode(
	ctx context.Context, entity *ent.DivisionCode,
) (*ent.DivisionCode, error) {
	updatedEntity := new(ent.DivisionCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createDivisionCodeEntity(ctx, tx, entity)
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

func (r *DivisionCodeService) createDivisionCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.DivisionCode,
) (*ent.DivisionCode, error) {
	createdEntity, err := tx.DivisionCode.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetNillableApAccountID(entity.ApAccountID).
		SetNillableCashAccountID(entity.CashAccountID).
		SetNillableExpenseAccountID(entity.ExpenseAccountID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateDivisionCode updates a division code.
func (r *DivisionCodeService) UpdateDivisionCode(ctx context.Context, entity *ent.DivisionCode) (*ent.DivisionCode, error) {
	updatedEntity := new(ent.DivisionCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateDivisionCodeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *DivisionCodeService) updateDivisionCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.DivisionCode,
) (*ent.DivisionCode, error) {
	current, err := tx.DivisionCode.Get(ctx, entity.ID)
	if err != nil {
		return nil, err
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.DivisionCode.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetNillableApAccountID(entity.ApAccountID).
		SetNillableCashAccountID(entity.CashAccountID).
		SetNillableExpenseAccountID(entity.ExpenseAccountID).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
