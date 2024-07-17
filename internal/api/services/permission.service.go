// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package services

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type PermissionService struct {
	enforcer *casbin.Enforcer
}

func NewPermissionService(enforcer *casbin.Enforcer) *PermissionService {
	return &PermissionService{
		enforcer: enforcer,
	}
}

func (s *PermissionService) CheckUserPermission(c *fiber.Ctx, resource string, action string) error {
	userID, ok := c.Locals(utils.CTXUserID).(uuid.UUID)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "User ID not found in context")
	}

	permission := fmt.Sprintf("%s:%s", resource, action)

	allowed, err := s.enforcer.Enforce(userID.String(), permission, "allow")
	if err != nil {
		return fmt.Errorf("Error checking permission: %w", err) //nolint:stylecheck // This is an error message
	}

	if !allowed {
		return fmt.Errorf("You do not have permission to `%s:%s`, please contact your administrator.", resource, action) //nolint:revive,stylecheck // This is an error message
	}

	return nil
}

func (s *PermissionService) CheckOwnershipPermission(c *fiber.Ctx, resource string, action string, ownerID string) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "User ID not found in context")
	}

	// If the user is updating their own profile, allow it
	if userID.String() == ownerID && resource == "user" && action == "update" {
		return nil
	}

	// Otherwise, check for the regular permission
	return s.CheckUserPermission(c, resource, action)
}

func (s *PermissionService) AddRoleForUser(userID uuid.UUID, role string) error {
	_, err := s.enforcer.AddGroupingPolicy(userID.String(), role)

	return err
}

func (s *PermissionService) AddPermissionForRole(role, resource, action string) error {
	_, err := s.enforcer.AddPolicy(role, resource, action)
	return err
}

func (s *PermissionService) AddCustomPermissionForUser(userID uuid.UUID, resource, action string) error {
	_, err := s.enforcer.AddPolicy(userID.String(), resource, action)
	return err
}
