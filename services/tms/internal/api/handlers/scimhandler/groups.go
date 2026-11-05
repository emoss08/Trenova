package scimhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/shared/scimproto"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listGroups(c *gin.Context) {
	dir := directoryOf(c)
	page, err := h.provisioner.ListGroups(c.Request.Context(), listRequest(c, dir))
	if err != nil {
		h.fail(c, err)
		return
	}

	base := h.baseURL(c)
	resources := make([]any, 0, len(page.Groups))
	for _, group := range page.Groups {
		resources = append(resources, locateGroup(base, group))
	}

	h.respond(c, http.StatusOK, scimproto.NewListResponse(resources, page.Total, page.StartIndex))
}

func (h *Handler) getGroup(c *gin.Context) {
	group, err := h.provisioner.GetGroup(c.Request.Context(), &scimports.GetGroupRequest{
		Directory:      directoryOf(c),
		GroupID:        pathID(c),
		IncludeMembers: !excludesMembers(c),
	})
	if err != nil {
		h.fail(c, err)
		return
	}

	h.respond(c, http.StatusOK, locateGroup(h.baseURL(c), group))
}

func (h *Handler) createGroup(c *gin.Context) {
	body, err := h.readBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	resource, err := scimproto.DecodeGroup(body)
	if err != nil {
		h.fail(c, err)
		return
	}

	group, err := h.provisioner.CreateGroup(c.Request.Context(), directoryOf(c), resource)
	if err != nil {
		h.fail(c, err)
		return
	}

	located := locateGroup(h.baseURL(c), group)
	c.Header("Location", located.Meta.Location)
	h.respond(c, http.StatusCreated, located)
}

func (h *Handler) replaceGroup(c *gin.Context) {
	body, err := h.readBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	resource, err := scimproto.DecodeGroup(body)
	if err != nil {
		h.fail(c, err)
		return
	}

	group, err := h.provisioner.ReplaceGroup(
		c.Request.Context(),
		directoryOf(c),
		pathID(c),
		resource,
	)
	if err != nil {
		h.fail(c, err)
		return
	}

	h.respond(c, http.StatusOK, locateGroup(h.baseURL(c), group))
}

func (h *Handler) patchGroup(c *gin.Context) {
	body, err := h.readBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	patch, err := scimproto.DecodePatch(body, scimports.MaxPatchOperations)
	if err != nil {
		h.fail(c, err)
		return
	}

	if err = h.provisioner.PatchGroup(
		c.Request.Context(),
		directoryOf(c),
		pathID(c),
		patch.Operations,
	); err != nil {
		h.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) deleteGroup(c *gin.Context) {
	if err := h.provisioner.DeleteGroup(c.Request.Context(), directoryOf(c), pathID(c)); err != nil {
		h.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func locateGroup(base string, group *scimproto.Group) *scimproto.Group {
	if group.Meta == nil {
		group.Meta = &scimproto.Meta{ResourceType: scimproto.ResourceTypeGroup}
	}
	group.Meta.Location = base + "/Groups/" + group.ID

	return group
}
