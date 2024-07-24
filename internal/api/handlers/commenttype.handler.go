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

package handlers

import (
	"fmt"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CommentTypeHandler struct {
	logger            *config.ServerLogger
	service           *services.CommentTypeService
	permissionService *services.PermissionService
	auditService      *audit.Service
}

func NewCommentTypeHandler(s *server.Server) *CommentTypeHandler {
	return &CommentTypeHandler{
		logger:            s.Logger,
		service:           services.NewCommentTypeService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
		auditService:      s.AuditService,
	}
}

func (h CommentTypeHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/comment-types")
	api.Get("/", h.Get())
	api.Get("/:commenttypeID", h.GetByID())
	api.Post("/", h.Create())
	api.Put("/:commenttypeID", h.Update())
}

func (h CommentTypeHandler) Get() fiber.Handler {
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

		if err = h.permissionService.CheckUserPermission(c, constants.EntityCommentType, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		filter := &services.CommentTypeQueryFilter{
			Query:          c.Query("search", ""),
			OrganizationID: ids.OrganizationID,
			BusinessUnitID: ids.BusinessUnitID,
			Limit:          limit,
			Offset:         offset,
		}

		entities, cnt, err := h.service.GetAll(c.UserContext(), filter)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to get CommentTypes")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.CommentType]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}

func (h CommentTypeHandler) GetByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		commenttypeID := c.Params("commenttypeID")
		if commenttypeID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "CommentType ID is required",
			})
		}

		if err = h.permissionService.CheckUserPermission(c, constants.EntityCommentType, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		entity, err := h.service.Get(c.UserContext(), uuid.MustParse(commenttypeID), ids.OrganizationID, ids.BusinessUnitID)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to get CommentType")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h CommentTypeHandler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		createdEntity := new(models.CommentType)

		if err = h.permissionService.CheckUserPermission(c, constants.EntityCommentType, constants.ActionCreate); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		createdEntity.BusinessUnitID = ids.BusinessUnitID
		createdEntity.OrganizationID = ids.OrganizationID

		if err = utils.ParseBodyAndValidate(c, createdEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		entity, err := h.service.Create(c.UserContext(), createdEntity)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to create CommentType")
			resp := utils.CreateServiceError(c, err)
			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		go h.auditService.LogAction(constants.TableCommentType, entity.ID.String(), property.AuditLogActionCreate, entity, ids.UserID, ids.OrganizationID, ids.BusinessUnitID)

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

func (h CommentTypeHandler) Update() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		commenttypeID := c.Params("commenttypeID")
		if commenttypeID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "CommentType ID is required",
			})
		}

		if err = h.permissionService.CheckUserPermission(c, constants.EntityCommentType, constants.ActionUpdate); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		updatedEntity := new(models.CommentType)

		if err = utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		updatedEntity.ID = uuid.MustParse(commenttypeID)

		entity, err := h.service.UpdateOne(c.UserContext(), updatedEntity)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to update CommentType")
			resp := utils.CreateServiceError(c, err)
			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		go h.auditService.LogAction(constants.TableCommentType, entity.ID.String(), property.AuditLogActionUpdate, entity, ids.UserID, ids.OrganizationID, ids.BusinessUnitID)

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
