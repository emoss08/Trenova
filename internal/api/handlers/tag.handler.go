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

type TagHandler struct {
	Service           *services.TagService
	PermissionService *services.PermissionService
}

func NewTagHandler(s *api.Server) *TagHandler {
	return &TagHandler{
		Service:           services.NewTagService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the TagHandler.
func (h *TagHandler) RegisterRoutes(r fiber.Router) {
	tagAPI := r.Group("/tags")
	tagAPI.Get("/", h.GetTags())
	tagAPI.Post("/", h.CreateTag())
	tagAPI.Put("/:tagID", h.UpdateTag())
}

// GetTags is a handler that returns a list of tags.
//
// GET /tags
func (h *TagHandler) GetTags() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "tag.view")
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

		entities, count, err := h.Service.GetTags(c.UserContext(), limit, offset, orgID, buID)
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

// CreateTag is a handler that creates a new tag.
//
// POST /tags
func (h *TagHandler) CreateTag() fiber.Handler {
	return func(c *fiber.Ctx) error {
		newEntity := new(ent.Tag)

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
		err := h.PermissionService.CheckUserPermission(c, "tag.add")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		newEntity.BusinessUnitID = buID
		newEntity.OrganizationID = orgID

		if err := util.ParseBodyAndValidate(c, newEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "body",
					},
				},
			})
		}

		entity, err := h.Service.CreateTag(c.UserContext(), newEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

// UpdateTag is a handler that updates an tag.
//
// PUT /tags/:tagID
func (h *TagHandler) UpdateTag() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tagID := c.Params("tagID")
		if tagID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "tag ID is required",
						Attr:   "tagID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "tag.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		updatedEntity := new(ent.Tag)

		if err := util.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "request body",
					},
				},
			})
		}

		updatedEntity.ID = uuid.MustParse(tagID)

		entity, err := h.Service.UpdateTag(c.UserContext(), updatedEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
