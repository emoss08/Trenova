package handlers

import (
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type RoleHandler struct {
	Service           *services.RoleService
	PermissionService *services.PermissionService
}

func NewRoleHandler(s *api.Server) *RoleHandler {
	return &RoleHandler{
		Service:           services.NewRoleService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the RoleHandler.
func (h *RoleHandler) RegisterRoutes(r fiber.Router) {
	rolesAPI := r.Group("/roles")
	rolesAPI.Get("/", h.GetRoles())
	rolesAPI.Post("/", h.CreateRole())
	rolesAPI.Put("/:roleID", h.UpdateRole())
}

// RoleResponse is the response payload for the role entity.
type RoleResponse struct {
	ID             uuid.UUID         `json:"id,omitempty"`
	BusinessUnitID uuid.UUID         `json:"businessUnitId"`
	OrganizationID uuid.UUID         `json:"organizationId"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	Version        int               `json:"version"`
	Name           string            `json:"name,omitempty"`
	Description    string            `json:"description,omitempty"`
	Permissions    []*ent.Permission `json:"permissions,omitempty"`
}

// GetRoles is a handler that returns a list of roles.
//
// GET /roles
func (h *RoleHandler) GetRoles() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "role.view")
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

		entities, count, err := h.Service.GetRoles(c.UserContext(), limit, offset, orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		nextURL := util.GetNextPageURL(c, limit, offset, count)
		prevURL := util.GetPrevPageURL(c, limit, offset)

		response := make([]RoleResponse, len(entities))
		for i, role := range entities {
			response[i] = RoleResponse{
				ID:             role.ID,
				BusinessUnitID: role.BusinessUnitID,
				OrganizationID: role.OrganizationID,
				CreatedAt:      role.CreatedAt,
				UpdatedAt:      role.UpdatedAt,
				Version:        role.Version,
				Name:           role.Name,
				Description:    role.Description,
				Permissions:    role.Edges.Permissions,
			}
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results:  response,
			Count:    count,
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

// CreateRole is a handler that creates a charge type.
//
// POST /charge-types
func (h *RoleHandler) CreateRole() fiber.Handler {
	return func(c *fiber.Ctx) error {
		newEntity := new(ent.Role)

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
		err := h.PermissionService.CheckUserPermission(c, "role.create")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		newEntity.BusinessUnitID = buID
		newEntity.OrganizationID = orgID

		if err := util.ParseBodyAndValidate(c, newEntity); err != nil {
			return err
		}

		entity, err := h.Service.CreateRole(c.UserContext(), newEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

// UpdateRole is a handler that updates a role.
//
// PUT /roles/:roleID
func (h *RoleHandler) UpdateRole() fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleID := c.Params("roleID")
		if roleID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Email Profile ID is required",
						Attr:   "roleID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "role.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		updatedEntity := new(ent.Role)

		if err := util.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return err
		}

		updatedEntity.ID = uuid.MustParse(roleID)

		entity, err := h.Service.UpdateRole(c.UserContext(), updatedEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
