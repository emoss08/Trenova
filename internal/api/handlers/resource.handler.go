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

type ResourceHandler struct {
	Service           *services.ResourceService
	PermissionService *services.PermissionService
}

func NewResourceHandler(s *api.Server) *ResourceHandler {
	return &ResourceHandler{
		Service: services.NewResourceService(s),
	}
}

// RegisterRoutes registers the routes for the ResourceHandler.
func (h *ResourceHandler) RegisterRoutes(r fiber.Router) {
	resourcesAPI := r.Group("/resources")
	resourcesAPI.Get("/", h.GetResources())
}

// ResourceResponse is the response payload for the role entity.
type ResourceResponse struct {
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// The time that this entity was created.
	CreatedAt time.Time `json:"createdAt" validate:"omitempty"`
	// The last time that this entity was updated.
	UpdatedAt time.Time `json:"updatedAt" validate:"omitempty"`
	// Type holds the value of the "type" field.
	Type string `json:"type,omitempty"`
	// Description holds the value of the "description" field.
	Description string `json:"description,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	Permissions []*ent.Permission `json:"permissions,omitempty"`
}

// GetResources is a handler that returns a list of resources.
//
// GET /resources
func (h *ResourceHandler) GetResources() fiber.Handler {
	return func(c *fiber.Ctx) error {
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

		entities, count, err := h.Service.GetResources(c.UserContext(), limit, offset)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		nextURL := util.GetNextPageURL(c, limit, offset, count)
		prevURL := util.GetPrevPageURL(c, limit, offset)

		response := make([]ResourceResponse, len(entities))
		for i, resource := range entities {
			response[i] = ResourceResponse{
				ID:          resource.ID,
				Type:        resource.Type,
				Description: resource.Description,
				Permissions: resource.Edges.Permissions,
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
