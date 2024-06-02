package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/queries"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type RateHandler struct {
	Logger            *zerolog.Logger
	Service           *services.RateService
	PermissionService *services.PermissionService
}

func NewRateHandler(s *api.Server) *RateHandler {
	return &RateHandler{
		Logger: s.Logger,
		Service: &services.RateService{
			Client: s.Client,
			Logger: s.Logger,
			QueryService: &queries.RateQueryService{
				Client: s.Client,
				Logger: s.Logger,
			},
		},
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the RateHandler.
func (h *RateHandler) RegisterRoutes(r fiber.Router) {
	rateAPI := r.Group("/rates")
	rateAPI.Get("/", h.getRates())
	rateAPI.Post("/", h.createRate())
	// Analytic routes
	rateAPI.Get("/analytics/expired-rates", h.getExpiredRates())
	// rateAPI.Put("/:customerID", h.updateCustomer())
}

// GetRates is a handler that returns a list of rates.
//
// GET /rates
func (h *RateHandler) getRates() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !buOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID or Business Unit ID not found in the request context",
						Attr:   "orgID, buID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "rate.view")
		if err != nil {
			h.Logger.Error().Err(err).Msg("User does not have permission to view rates")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		offset, limit, err := util.PaginationParams(c)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Error parsing pagination parameters")
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "offset, limit",
					},
				},
			})
		}

		// Represents the statuses of the rates to be retrieved ex. "A,I"
		statusStr := c.Query("statuses")

		entities, count, err := h.Service.GetRates(c.UserContext(), limit, offset, orgID, buID, statusStr)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Error retrieving rates")
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		nextURL := util.GetNextPageURL(c, limit, offset, count)
		prevURL := util.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results:  entities,
			Count:    count,
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

// createRate is a handler that creates a new rate.
//
// POST /rates
func (h *RateHandler) createRate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		newEntity := new(ent.Rate)

		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !buOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID or Business Unit ID not found in the request context",
						Attr:   "orgID, buID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "rate.add")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		newEntity.BusinessUnitID = buID
		newEntity.OrganizationID = orgID

		if err = util.ParseBodyAndValidate(c, newEntity); err != nil {
			return err
		}

		entity, err := h.Service.CreateRate(c.UserContext(), newEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

func (h *RateHandler) getExpiredRates() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !buOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID or Business Unit ID not found in the request context",
						Attr:   "orgID, buID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "rate.view")
		if err != nil {
			h.Logger.Error().Err(err).Msg("User does not have permission to view rates")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		entities, count, err := h.Service.GetsRatesNearExpiration(c.UserContext(), orgID, buID)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Error retrieving rates")
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results: entities,
			Count:   count,
		})
	}
}
