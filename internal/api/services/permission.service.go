package services

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
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
				pq.Where(ps.NameEQ(permission))
			})
		}).Only(ctx)
	if err != nil {
		s.Logger.Error().Err(err).Str("userID", userID.String()).Msg("Failed to query user")
		return false, fmt.Errorf("failed to query user: %w", err)
	}

	// Check if the user has the requested permission
	for _, role := range user.Edges.Roles {
		for _, perm := range role.Edges.Permissions {
			if perm.Name == permission {
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
