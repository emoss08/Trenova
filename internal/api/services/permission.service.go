package services

import (
	"context"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type PermissionService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

func NewPermissionService(s *server.Server) *PermissionService {
	return &PermissionService{
		db:     s.DB,
		logger: s.Logger,
	}
}

func (s *PermissionService) hasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	// Query the user with roles and permissions.
	user := new(models.User)

	err := s.db.NewSelect().
		Model(user).
		Relation("Roles.Permissions").
		Where("u.id = ?", userID).
		Scan(ctx)
	if err != nil {
		return false, err
	}

	// If the user is admin, return true.
	if user.IsAdmin {
		return true, nil
	}

	// Check if the user has the permission.
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			if perm.Codename == permission {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *PermissionService) CheckUserPermission(c *fiber.Ctx, permission string) error {
	userID, ok := c.Locals(utils.CTXUserID).(uuid.UUID)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "user id not found in context")
	}

	hasPermission, err := s.hasPermission(c.Context(), userID, permission)
	if err != nil {
		return err
	}

	if !hasPermission {
		return fiber.NewError(fiber.StatusForbidden, "user does not have permission")
	}

	return nil
}

func (s *PermissionService) GetPermissions(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*models.Permission, int, error) {
	var permissions []*models.Permission
	count, err := s.db.NewSelect().
		Model(&permissions).
		Relation("Roles").
		Where("p.organization_id = ?", orgID).
		Where("p.business_unit_id = ?", buID).
		Order("p.created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return permissions, count, nil
}

func (s *PermissionService) CheckOwnershipPermission(c *fiber.Ctx, permission string, userID string) error {
	uid, ok := c.Locals(utils.CTXUserID).(uuid.UUID)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "user id not found in context")
	}

	// Check if the user is updating their own profile.
	if uid.String() != userID {
		// if the user not updating their own profile, check if the user has the required permission.
		return s.CheckUserPermission(c, permission)
	}

	return nil
}
