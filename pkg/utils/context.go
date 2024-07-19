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

package utils

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ContextKey string

const (
	CTXKeyDisableLogger = ContextKey("disableLogger")
	CTXOrganizationID   = ContextKey("organizationID")
	CTXBusinessUnitID   = ContextKey("businessUnitID")
	CTXUserID           = ContextKey("userID")
	CTXDB               = ContextKey("db")
)

var ErrMissingContextData = errors.New("required data missing from context")

// ContextIDs represents the important IDs extracted from the Fiber context
type ContextIDs struct {
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	UserID         uuid.UUID
}

// ExtractContextIDs extracts and validates OrganizationID, BusinessUnitID, and UserID from the Fiber context
func ExtractContextIDs(c *fiber.Ctx) (ContextIDs, error) {
	var ids ContextIDs
	var ok bool

	// Extract OrganizationID
	ids.OrganizationID, ok = c.Locals(CTXOrganizationID).(uuid.UUID)
	if !ok || ids.OrganizationID == uuid.Nil {
		return ids, formatError("OrganizationID")
	}

	// Extract BusinessUnitID
	ids.BusinessUnitID, ok = c.Locals(CTXBusinessUnitID).(uuid.UUID)
	if !ok || ids.BusinessUnitID == uuid.Nil {
		return ids, formatError("BusinessUnitID")
	}

	// Extract UserID
	ids.UserID, ok = c.Locals(CTXUserID).(uuid.UUID)
	if !ok || ids.UserID == uuid.Nil {
		return ids, formatError("UserID")
	}

	return ids, nil
}

// formatError creates a formatted error message for missing or invalid IDs
func formatError(missingField string) error {
	return fmt.Errorf("%w: %s is missing or invalid", ErrMissingContextData, missingField)
}

// HandleContextError handles errors from ExtractContextIDs and returns an appropriate Fiber error
func HandleContextError(c *fiber.Ctx, err error) error {
	if errors.Is(err, ErrMissingContextData) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
			Code:    fiber.StatusUnauthorized,
			Message: err.Error(),
		})
	}
	// Handle other types of errors if needed
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
		Code:    fiber.StatusInternalServerError,
		Message: "An unexpected error occurred",
	})
}

// ExtractAndHandleContextIDs combines extraction and error handling
func ExtractAndHandleContextIDs(c *fiber.Ctx) (ContextIDs, error) {
	ids, err := ExtractContextIDs(c)
	if err != nil {
		return ids, HandleContextError(c, err)
	}
	return ids, nil
}
