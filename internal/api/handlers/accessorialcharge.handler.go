package handlers

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type AccessorialChargeHandler struct {
	logger            *zerolog.Logger
	service           *services.AccessorialChargeService
	permissionService *services.PermissionService
}

func NewAccessorialChargeHandler(s *server.Server) *AccessorialChargeHandler {
	return &AccessorialChargeHandler{
		logger:            s.Logger,
		service:           services.NewAccessorialChargeService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
	}
}

func (h AccessorialChargeHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/accessorial-charges")
	api.Get("/", h.Get())
	api.Get("/:accessorialchargeID", h.GetByID())
	api.Post("/", h.Create())
	api.Put("/:accessorialchargeID", h.Update())
}

func (h AccessorialChargeHandler) Get() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("AccessorialChargeHandler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		offset, limit, err := utils.PaginationParams(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ProblemDetail{
				Type:     "invalid",
				Title:    "Invalid Request",
				Status:   fiber.StatusBadRequest,
				Detail:   err.Error(),
				Instance: fmt.Sprintf("%s/probs/validation-error", c.BaseURL()),
				InvalidParams: []types.InvalidParam{
					{
						Name:   "limit",
						Reason: "Limit must be a positive integer",
					},
					{
						Name:   "offset",
						Reason: "Offset must be a positive integer",
					},
				},
			})
		}

		if err = h.permissionService.CheckUserPermission(c, "accessorial_charge", "view"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		filter := &services.AccessorialChargeQueryFilter{
			Query:          c.Query("search", ""),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Limit:          limit,
			Offset:         offset,
		}

		entities, cnt, err := h.service.GetAll(c.UserContext(), filter)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get AccessorialCharges",
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.AccessorialCharge]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}

func (h AccessorialChargeHandler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		createdEntity := new(models.AccessorialCharge)

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "accessorial_charge", "create"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		createdEntity.BusinessUnitID = buID
		createdEntity.OrganizationID = orgID

		if err := utils.ParseBodyAndValidate(c, createdEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		entity, err := h.service.Create(c.UserContext(), createdEntity)
		if err != nil {
			resp := utils.CreateServiceError(c, err)
			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

func (h AccessorialChargeHandler) GetByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		accessorialchargeID := c.Params("accessorialchargeID")
		if accessorialchargeID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "AccessorialCharge ID is required",
			})
		}

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("AccessorialChargeHandler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "accessorial_charge", "view"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		entity, err := h.service.Get(c.UserContext(), uuid.MustParse(accessorialchargeID), orgID, buID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get AccessorialCharge",
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h AccessorialChargeHandler) Update() fiber.Handler {
	return func(c *fiber.Ctx) error {
		accessorialchargeID := c.Params("accessorialchargeID")
		if accessorialchargeID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "AccessorialCharge ID is required",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "accessorial_charge", "update"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		updatedEntity := new(models.AccessorialCharge)

		if err := utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		updatedEntity.ID = uuid.MustParse(accessorialchargeID)

		entity, err := h.service.UpdateOne(c.UserContext(), updatedEntity)
		if err != nil {
			resp := utils.CreateServiceError(c, err)
			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
