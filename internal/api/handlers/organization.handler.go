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

type OrganizationHandler struct {
	Server            *api.Server
	Service           *services.OrganizationService
	PermissionService *services.PermissionService
}

func NewOrganizationHandler(s *api.Server) *OrganizationHandler {
	return &OrganizationHandler{
		Server:            s,
		Service:           services.NewOrganizationService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

func (h *OrganizationHandler) RegisterRoutes(r fiber.Router) {
	organizationAPI := r.Group("/organizations")
	organizationAPI.Get("/me", h.GetUserOrganization())
	organizationAPI.Post("/logo", h.UploadLogo())
	organizationAPI.Put("/:orgID", h.UpdateOrganization())
}
// GetUserOrganization is a handler that returns the organization of the currently authenticated user.
//
// GET /organizations/me
func (h *OrganizationHandler) GetUserOrganization() fiber.Handler {
	return func(c *fiber.Ctx) error {
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)
		orgID, orgOK := c.Locals(util.CTXOrganizationID).(uuid.UUID)

		if !buOK || !orgOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalServerError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalServerError",
						Detail: "Internal server error",
						Attr:   "session",
					},
				},
			},
			)
		}

		user, err := h.Service.GetUserOrganization(c.UserContext(), buID, orgID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(user)
	}
}

func (h *OrganizationHandler) UploadLogo() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID not found in the request context",
						Attr:   "OrgID",
					},
				},
			})
		}

		// Handle the uploaded file
		logo, err := c.FormFile("logo")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("failed to read file from request")
		}

		entity, err := h.Service.UploadLogo(c.UserContext(), logo, orgID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h *OrganizationHandler) UpdateOrganization() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID := c.Params("orgID")
		if orgID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Organization ID is required",
						Attr:   "orgID",
					},
				},
			})
		}

		// Check if the user has the required permissions or if the user is updating their own profile
		err := h.PermissionService.CheckUserPermission(c, "organization.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		updatedEntity := new(ent.Organization)

		if err = util.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return err
		}

		updatedEntity.ID = uuid.MustParse(orgID)

		entity, err := h.Service.UpdateOrganization(c.UserContext(), updatedEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
