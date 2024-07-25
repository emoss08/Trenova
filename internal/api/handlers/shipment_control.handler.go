package handlers

import (
	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type ShipmentControlHandler struct {
	logger            *config.ServerLogger
	service           *services.ShipmentControlService
	permissionService *services.PermissionService
	auditService      *audit.Service
}

func NewShipmentControlHandler(s *server.Server) *ShipmentControlHandler {
	return &ShipmentControlHandler{
		logger:            s.Logger,
		service:           services.NewShipmentControlService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
		auditService:      s.AuditService,
	}
}

func (sh ShipmentControlHandler) RegisterRoutes(r fiber.Router) {
	shControlAPI := r.Group("/shipment-control")
	shControlAPI.Get("/", sh.getShipmentControlDetails())
	shControlAPI.Put("/", sh.updateShipmentControl())
}

func (sh ShipmentControlHandler) getShipmentControlDetails() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		if err = sh.permissionService.CheckUserPermission(c, constants.EntityShipmentControl, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		entity, err := sh.service.GetShipmentControl(c.UserContext(), ids.BusinessUnitID, ids.OrganizationID)
		if err != nil {
			sh.logger.Error().Str("organizationID", ids.OrganizationID.String()).Err(err).Msg("Error getting shipment control details")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (sh ShipmentControlHandler) updateShipmentControl() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		if err = sh.permissionService.CheckUserPermission(c, constants.EntityShipmentControl, constants.ActionUpdate); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		updatedEntity := new(models.ShipmentControl)
		updatedEntity.OrganizationID = ids.OrganizationID
		updatedEntity.BusinessUnitID = ids.BusinessUnitID

		if err = utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		entity, err := sh.service.UpdateShipmentControl(c.UserContext(), updatedEntity)
		if err != nil {
			sh.logger.Error().Interface("entity", updatedEntity).Err(err).Msg("Failed to update ShipmentControl")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		go sh.auditService.LogAction(constants.TableShipmentControl, entity.ID.String(), property.AuditLogActionUpdate, entity, ids.UserID, ids.OrganizationID, ids.BusinessUnitID)

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
