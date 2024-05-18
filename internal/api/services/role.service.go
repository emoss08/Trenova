package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/role"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type RoleService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewRoleService creates a new role service.
func NewRoleService(s *api.Server) *RoleService {
	return &RoleService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetRoles gets the roles for an organization and business unit.
func (s *RoleService) GetRoles(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.Role, int, error) {
	entityCount, countErr := s.Client.Role.Query().Where(
		role.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := s.Client.Role.Query().
		Limit(limit).
		Offset(offset).
		WithPermissions().
		Where(
			role.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateRole creates a new role.
func (s *RoleService) CreateRole(
	ctx context.Context, entity *ent.Role,
) (*ent.Role, error) {
	updatedEntity := new(ent.Role)

	err := util.WithTx(ctx, s.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = s.createRoleEntity(ctx, tx, entity)
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

func (s *RoleService) createRoleEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Role,
) (*ent.Role, error) {
	createdEntity, err := tx.Role.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetName(entity.Name).
		SetDescription(entity.Description).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateRole updates a charge type.
func (s *RoleService) UpdateRole(
	ctx context.Context, entity *ent.Role,
) (*ent.Role, error) {
	updatedEntity := new(ent.Role)

	err := util.WithTx(ctx, s.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = s.updateRoleEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (s *RoleService) updateRoleEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Role,
) (*ent.Role, error) {
	current, err := tx.Role.Get(ctx, entity.ID)
	if err != nil {
		return nil, err
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"name")
	}

	// Start building the update operation
	updateOp := tx.Role.UpdateOneID(entity.ID).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
