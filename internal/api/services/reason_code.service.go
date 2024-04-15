package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/reasoncode"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
)

type ReasonCodeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewReasonCodeService creates a new reason code service.
func NewReasonCodeService(s *api.Server) *ReasonCodeService {
	return &ReasonCodeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetReasonCodes gets the reason codes for an organization.
func (r *ReasonCodeService) GetReasonCodes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.ReasonCode, int, error) {
	entityCount, countErr := r.Client.ReasonCode.Query().Where(
		reasoncode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.ReasonCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			reasoncode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateReasonCode creates a new reason code.
func (r *ReasonCodeService) CreateReasonCode(
	ctx context.Context, entity *ent.ReasonCode,
) (*ent.ReasonCode, error) {
	newEntity := new(ent.ReasonCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createReasonCodeEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return newEntity, nil
}

func (r *ReasonCodeService) createReasonCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.ReasonCode,
) (*ent.ReasonCode, error) {
	createdEntity, err := tx.ReasonCode.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetCodeType(entity.CodeType).
		SetDescription(entity.Description).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create entity")
	}

	return createdEntity, nil
}

// UpdateReasonCode updates a reason code.
func (r *ReasonCodeService) UpdateReasonCode(
	ctx context.Context, entity *ent.ReasonCode,
) (*ent.ReasonCode, error) {
	updatedEntity := new(ent.ReasonCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateReasonCodeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *ReasonCodeService) updateReasonCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.ReasonCode,
) (*ent.ReasonCode, error) {
	current, err := tx.ReasonCode.Get(ctx, entity.ID)
	if err != nil {
		return nil, eris.Wrap(err, "failed to retrieve requested entity")
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.ReasonCode.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetCodeType(entity.CodeType).
		SetDescription(entity.Description).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
