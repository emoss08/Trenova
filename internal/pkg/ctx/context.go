package ctx

import (
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
)

// ContextKey type for context keys
type ContextKey string

const (
	CTXUserID         = ContextKey("userID")
	CTXOrganizationID = ContextKey("organizationID")
	CTXBusinessUnitID = ContextKey("businessUnitID")
	CTXSessionID      = ContextKey("sessionID")
	CTXUserRole       = ContextKey("userRole")
)

var (
	ErrMissingContextData = eris.New("required data missing from context")
	ErrInvalidContextData = eris.New("invalid context data type")
)

// RequestContext contains all the context information needed for request processing
type RequestContext struct {
	UserID pulid.ID `json:"userId"`
	OrgID  pulid.ID `json:"organizationId"`
	BuID   pulid.ID `json:"businessUnitId"`
}

// GetRequestContext extracts all relevant information from the Fiber context
func GetRequestContext(c *fiber.Ctx) (*RequestContext, error) {
	ctx := &RequestContext{}

	// Get user ID
	if userID, ok := c.Locals(CTXUserID).(pulid.ID); ok {
		ctx.UserID = userID
	} else {
		log.Debug().Interface("userID", c.Locals(CTXUserID)).Msg("invalid user ID type")
		return nil, eris.Wrap(ErrMissingContextData, "user ID not found or invalid type")
	}

	// Get organization ID
	if orgID, ok := c.Locals(CTXOrganizationID).(pulid.ID); ok {
		ctx.OrgID = orgID
	} else {
		log.Debug().Interface("orgID", c.Locals(CTXOrganizationID)).Msg("invalid org ID type")
		return nil, eris.Wrap(ErrMissingContextData, "organization ID not found or invalid type")
	}

	// Get business unit ID
	if buID, ok := c.Locals(CTXBusinessUnitID).(pulid.ID); ok {
		ctx.BuID = buID
	} else {
		log.Debug().Interface("buID", c.Locals(CTXBusinessUnitID)).Msg("invalid business unit ID type")
		return nil, eris.Wrap(ErrMissingContextData, "business unit ID not found or invalid type")
	}

	return ctx, nil
}

// HandleRequestError handles errors from GetRequestContext
func HandleRequestError(c *fiber.Ctx, err error) error {
	if eris.Is(err, ErrMissingContextData) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Unauthorized",
			"message": err.Error(),
		})
	}

	if eris.Is(err, ErrInvalidContextData) {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal Server Error",
			"message": "Invalid context data type",
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error":   "Internal Server Error",
		"message": "An unexpected error occurred",
	})
}

// WithRequestContext combines getting the request context and error handling
func WithRequestContext(c *fiber.Ctx) (*RequestContext, error) {
	ctx, err := GetRequestContext(c)
	if err != nil {
		return nil, HandleRequestError(c, err)
	}

	return ctx, nil
}
