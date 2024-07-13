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

type TableChangeAlertHandler struct {
	logger            *zerolog.Logger
	service           *services.TableChangeAlertService
	permissionService *services.PermissionService
}

func NewTableChangeAlertHandler(s *server.Server) *TableChangeAlertHandler {
	return &TableChangeAlertHandler{
		logger:            s.Logger,
		service:           services.NewTableChangeAlertService(s),
		permissionService: services.NewPermissionService(s),
	}
}

func (h TableChangeAlertHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/table-change-alerts")
	api.Get("/", h.Get())
	api.Get("/topics", h.getTopicNames())
	api.Post("/", h.Create())
	api.Put("/:tableChangeAlertID", h.Update())
}

func (h TableChangeAlertHandler) Get() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("TableChangeAlertHandler: Organization & Business Unit ID not found in context")
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

		if err = h.permissionService.CheckUserPermission(c, models.PermissionTableChangeAlertView.String()); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "You do not have permission to perform this action.",
			})
		}

		entities, cnt, err := h.service.GetTableChangeAlerts(c.UserContext(), limit, offset, orgID, buID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get table change alerts",
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.TableChangeAlert]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}

func (h TableChangeAlertHandler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		createdEntity := new(models.TableChangeAlert)

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("WorkerHandler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, models.PermissionTableChangeAlertAdd.String()); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "You do not have permission to perform this action.",
			})
		}

		createdEntity.BusinessUnitID = buID
		createdEntity.OrganizationID = orgID

		if err := utils.ParseBodyAndValidate(c, createdEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		entity, err := h.service.CreateTableChangeAlert(c.UserContext(), createdEntity)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

func (h TableChangeAlertHandler) Update() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableChangeAlertID := c.Params("tableChangeAlertID")
		if tableChangeAlertID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Table Change Alert ID is required",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, models.PermissionTableChangeAlertAdd.String()); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "You do not have permission to perform this action.",
			})
		}

		updatedEntity := new(models.TableChangeAlert)

		if err := utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		updatedEntity.ID = uuid.MustParse(tableChangeAlertID)

		entity, err := h.service.UpdateTableChangeAlert(c.UserContext(), updatedEntity)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to update table change alert",
			})
		}
		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h TableChangeAlertHandler) getTopicNames() fiber.Handler {
	return func(c *fiber.Ctx) error {
		entities, cnt, err := h.service.GetTopicNames()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get topic names",
			})
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]types.TopicName]{
			Results: entities,
			Count:   cnt,
		})
	}
}
