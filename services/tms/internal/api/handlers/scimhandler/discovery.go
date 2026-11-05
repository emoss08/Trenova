package scimhandler

import (
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/shared/scimproto"
	"github.com/gin-gonic/gin"
)

func (h *Handler) serviceProviderConfig(c *gin.Context) {
	h.respond(
		c,
		http.StatusOK,
		scimproto.NewServiceProviderConfig(h.baseURL(c), scimports.MaxPageSize),
	)
}

func (h *Handler) resourceTypes(c *gin.Context) {
	types := scimproto.NewResourceTypes(h.baseURL(c))
	resources := make([]any, 0, len(types))
	for _, resourceType := range types {
		resources = append(resources, resourceType)
	}

	h.respond(c, http.StatusOK, scimproto.NewListResponse(resources, len(resources), 1))
}

func (h *Handler) resourceType(c *gin.Context) {
	id := c.Param(idParam)
	for _, resourceType := range scimproto.NewResourceTypes(h.baseURL(c)) {
		if strings.EqualFold(resourceType.ID, id) {
			h.respond(c, http.StatusOK, resourceType)
			return
		}
	}

	h.fail(c, scimproto.NotFound("The resource type was not found"))
}

func (h *Handler) schemas(c *gin.Context) {
	schemas := scimproto.NewSchemas(h.baseURL(c))
	resources := make([]any, 0, len(schemas))
	for _, schema := range schemas {
		resources = append(resources, schema)
	}

	h.respond(c, http.StatusOK, scimproto.NewListResponse(resources, len(resources), 1))
}

func (h *Handler) schema(c *gin.Context) {
	id := c.Param(idParam)
	for _, schema := range scimproto.NewSchemas(h.baseURL(c)) {
		if strings.EqualFold(schema.ID, id) {
			h.respond(c, http.StatusOK, schema)
			return
		}
	}

	h.fail(c, scimproto.NotFound("The schema was not found"))
}
