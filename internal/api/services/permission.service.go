package services

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/permission"
	ps "github.com/emoss08/trenova/internal/ent/permission"
	"github.com/emoss08/trenova/internal/ent/user"
	"github.com/emoss08/trenova/internal/util"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type PermissionService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

func NewPermissionService(s *api.Server) *PermissionService {
	return &PermissionService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// HasPermission checks if a user has a specific permission.
func (s *PermissionService) hasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	// Query user with roles and permissions
	user, err := s.Client.User.Query().
		Where(user.IDEQ(userID)).
		WithRoles(func(q *ent.RoleQuery) {
			q.WithPermissions(func(pq *ent.PermissionQuery) {
				pq.Where(ps.Codename(permission))
			})
		}).Only(ctx)
	if err != nil {
		s.Logger.Error().Err(err).Str("userID", userID.String()).Msg("Failed to query user")
		return false, fmt.Errorf("failed to query user: %w", err)
	}

	// If the user is admin, return true
	if user.IsAdmin || user.IsSuperAdmin {
		return true, nil
	}

	// Check if the user has the requested permission
	for _, role := range user.Edges.Roles {
		for _, perm := range role.Edges.Permissions {
			if perm.Codename == permission {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *PermissionService) CheckUserPermission(c *fiber.Ctx, permission string) error {
	userID, ok := c.Locals(util.CTXUserID).(uuid.UUID)
	if !ok {
		return errors.New("user ID not found in the request context")
	}

	hasPermission, err := s.hasPermission(c.UserContext(), userID, permission)
	if err != nil {
		s.Logger.Error().Err(err).Str("userID", userID.String()).Msg("Failed to check user permissions")
		return fmt.Errorf("failed to check user permissions: %w", err)
	}

	if !hasPermission {
		s.Logger.Warn().Str("userID", userID.String()).Msg("User does not have permission")
		return errors.New("user does not have permission")
	}

	return err
}

// GetPermissions gets the permissions for an organization.
func (s *PermissionService) GetPermissions(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.Permission, int, error) {
	entityCount, countErr := s.Client.Permission.Query().Where(
		permission.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := s.Client.Permission.Query().
		Limit(limit).
		Offset(offset).
		WithRoles().
		Where(
			permission.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}
