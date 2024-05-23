package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type TableChangeAlertHandler struct {
	Service           *services.TableChangeAlertService
	PermissionService *services.PermissionService
}

func NewTableChangeAlertHandler(s *api.Server) *TableChangeAlertHandler {
	return &TableChangeAlertHandler{
		Service:           services.NewTableChangeAlertService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the TableChangeAlertHandler.
func (h *TableChangeAlertHandler) RegisterRoutes(r fiber.Router) {
	tableChangeAlertAPI := r.Group("/table-change-alerts")
	tableChangeAlertAPI.Get("/", h.getTableChangeAlerts())
	tableChangeAlertAPI.Post("/", h.createTableChangeAlert())
	tableChangeAlertAPI.Put("/:tableChangeAlertID", h.updateTableChangeAlert())
	tableChangeAlertAPI.Get("/table-names", h.getTableNames())
	tableChangeAlertAPI.Get("/topic-names", h.getTopicNames())
}

// getTableChangeAlerts is a handler that returns a list of table change alerts.
//
// GET /table-change-alerts
func (h *TableChangeAlertHandler) getTableChangeAlerts() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "tablechangealert.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		offset, limit, err := util.PaginationParams(c)
		if err != nil {
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

		entities, count, err := h.Service.GetTableChangeAlerts(c.UserContext(), limit, offset, orgID, buID)
		if err != nil {
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

// createTableChangeAlert is a handler that creates a table change alert.
//
// POST /table-change-alerts
func (h *TableChangeAlertHandler) createTableChangeAlert() fiber.Handler {
	return func(c *fiber.Ctx) error {
		newEntity := new(ent.TableChangeAlert)

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
		err := h.PermissionService.CheckUserPermission(c, "tablechangealert.add")
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

		entity, err := h.Service.CreateTableChangeAlert(c.UserContext(), newEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusBadRequest).JSON(errorResponse)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

// updateTableChangeAlert is a handler that updates a table change alert.
//
// PUT /table-change-alerts/:tableChangeAlertID
func (h *TableChangeAlertHandler) updateTableChangeAlert() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableChangeAlertID := c.Params("tableChangeAlertID")
		if tableChangeAlertID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Email Profile ID is required",
						Attr:   "tableChangeAlertID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "tablechangealert.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		updatedEntity := new(ent.TableChangeAlert)

		if err = util.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return err
		}

		updatedEntity.ID = uuid.MustParse(tableChangeAlertID)

		entity, err := h.Service.UpdateTableChangeAlert(c.UserContext(), updatedEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusBadRequest).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h *TableChangeAlertHandler) getTableNames() fiber.Handler {
	return func(c *fiber.Ctx) error {
		entities, count, err := h.Service.GetTableNames(c.UserContext())
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results: entities,
			Count:   count,
		})
	}
}

func (h *TableChangeAlertHandler) getTopicNames() fiber.Handler {
	return func(c *fiber.Ctx) error {
		entities, count, err := h.Service.GetTopicNames()
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results: entities,
			Count:   count,
		})
	}
}
