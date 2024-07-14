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

type CommodityHandler struct {
	logger            *zerolog.Logger
	service           *services.CommodityService
	permissionService *services.PermissionService
}

func NewCommodityHandler(s *server.Server) *CommodityHandler {
	return &CommodityHandler{
		logger:            s.Logger,
		service:           services.NewCommodityService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
	}
}

func (h CommodityHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/commodities")
	api.Get("/", h.Get())
	api.Get("/:commodityID", h.GetByID())
	api.Post("/", h.Create())
	api.Put("/:commodityID", h.Update())
}

func (h CommodityHandler) Get() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("CommodityHandler: Organization & Business Unit ID not found in context")
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

		if err := h.permissionService.CheckUserPermission(c, "commodity", "view"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		filter := &services.CommodityQueryFilter{
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
				Message: "Failed to get Commodities",
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.Commodity]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}

func (h CommodityHandler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		createdEntity := new(models.Commodity)

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "commodity", "create"); err != nil {
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

func (h CommodityHandler) GetByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		commodityID := c.Params("commodityID")
		if commodityID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Commodity ID is required",
			})
		}

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("CommodityHandler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "commodity", "view"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		entity, err := h.service.Get(c.UserContext(), uuid.MustParse(commodityID), orgID, buID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get Commodity",
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h CommodityHandler) Update() fiber.Handler {
	return func(c *fiber.Ctx) error {
		commodityID := c.Params("commodityID")
		if commodityID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Commodity ID is required",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "commodity", "update"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		updatedEntity := new(models.Commodity)

		if err := utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		updatedEntity.ID = uuid.MustParse(commodityID)

		entity, err := h.service.UpdateOne(c.UserContext(), updatedEntity)
		if err != nil {
			resp := utils.CreateServiceError(c, err)
			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
