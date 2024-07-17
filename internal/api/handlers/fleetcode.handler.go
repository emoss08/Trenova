// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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

type FleetCodeHandler struct {
	logger            *zerolog.Logger
	service           *services.FleetCodeService
	permissionService *services.PermissionService
}

func NewFleetCodeHandler(s *server.Server) *FleetCodeHandler {
	return &FleetCodeHandler{
		logger:            s.Logger,
		service:           services.NewFleetCodeService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
	}
}

func (h FleetCodeHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/fleet-codes")
	api.Get("/", h.Get())
	api.Get("/:fleetcodeID", h.GetByID())
	api.Post("/", h.Create())
	api.Put("/:fleetcodeID", h.Update())
}

func (h FleetCodeHandler) Get() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("FleetCodeHandler: Organization & Business Unit ID not found in context")
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

		if err = h.permissionService.CheckUserPermission(c, "fleet_code", "view"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		filter := &services.FleetCodeQueryFilter{
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
				Message: "Failed to get FleetCodes",
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.FleetCode]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}

func (h FleetCodeHandler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		createdEntity := new(models.FleetCode)

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "fleet_code", "create"); err != nil {
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

func (h FleetCodeHandler) GetByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		fleetcodeID := c.Params("fleetcodeID")
		if fleetcodeID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "FleetCode ID is required",
			})
		}

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("FleetCodeHandler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "fleet_code", "view"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		entity, err := h.service.Get(c.UserContext(), uuid.MustParse(fleetcodeID), orgID, buID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get FleetCode",
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h FleetCodeHandler) Update() fiber.Handler {
	return func(c *fiber.Ctx) error {
		fleetcodeID := c.Params("fleetcodeID")
		if fleetcodeID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "FleetCode ID is required",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "fleet_code", "update"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		updatedEntity := new(models.FleetCode)

		if err := utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		updatedEntity.ID = uuid.MustParse(fleetcodeID)

		entity, err := h.service.UpdateOne(c.UserContext(), updatedEntity)
		if err != nil {
			resp := utils.CreateServiceError(c, err)
			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
