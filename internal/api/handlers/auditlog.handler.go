package handlers

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/models/property"

	"github.com/emoss08/trenova/pkg/models"

	"github.com/google/uuid"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type AuditLogHandler struct {
	logger            *config.ServerLogger
	service           *services.AuditLogService
	permissionService *services.PermissionService
}

func NewAuditLogHandler(s *server.Server) *AuditLogHandler {
	return &AuditLogHandler{
		logger:            s.Logger,
		service:           services.NewAuditLogService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
	}
}

func (h AuditLogHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/audit-logs")
	api.Get("/", h.Get())
}

func (h AuditLogHandler) Get() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
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
						Name:   constants.FieldLimit,
						Reason: constants.ReasonMustBePositiveInteger,
					},
					{
						Name:   constants.FieldOffset,
						Reason: constants.ReasonMustBePositiveInteger,
					},
				},
			})
		}

		if err = h.permissionService.CheckUserPermission(c, constants.EntityAuditLog, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		filter := &services.AuditLogQueryFilter{
			OrganizationID: ids.OrganizationID,
			BusinessUnitID: ids.BusinessUnitID,
			Limit:          limit,
			Offset:         offset,
		}

		if userID := c.Query("userId"); userID != "" {
			if id, err := uuid.Parse(userID); err != nil {
				filter.UserID = id
			}
		}

		if tableName := c.Query("tableName"); tableName != "" {
			filter.TableName = tableName
		}

		if entityID := c.Query("entityId"); entityID != "" {
			filter.EntityID = entityID
		}

		if action := c.Query("action"); action != "" {
			filter.Action = property.AuditLogAction(action)
		}

		if status := c.Query("status"); status != "" {
			filter.Status = property.LogStatus(status)
		}

		entities, cnt, err := h.service.GetAll(c.UserContext(), filter)
		if err != nil {
			h.logger.Error().Err(err).Msg("Error getting audit logs")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: constants.ErrInternalServer,
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.AuditLog]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}
