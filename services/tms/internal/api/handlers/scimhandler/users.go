package scimhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/shared/scimproto"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listUsers(c *gin.Context) {
	dir := directoryOf(c)
	page, err := h.provisioner.ListUsers(c.Request.Context(), listRequest(c, dir))
	if err != nil {
		h.fail(c, err)
		return
	}

	base := h.baseURL(c)
	resources := make([]any, 0, len(page.Users))
	for _, user := range page.Users {
		resources = append(resources, locateUser(base, user))
	}

	h.respond(c, http.StatusOK, scimproto.NewListResponse(resources, page.Total, page.StartIndex))
}

func (h *Handler) getUser(c *gin.Context) {
	user, err := h.provisioner.GetUser(c.Request.Context(), directoryOf(c), pathID(c))
	if err != nil {
		h.fail(c, err)
		return
	}

	h.respond(c, http.StatusOK, locateUser(h.baseURL(c), user))
}

func (h *Handler) createUser(c *gin.Context) {
	body, err := h.readBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	resource, err := scimproto.DecodeUser(body)
	if err != nil {
		h.fail(c, err)
		return
	}

	user, err := h.provisioner.CreateUser(c.Request.Context(), directoryOf(c), resource)
	if err != nil {
		h.fail(c, err)
		return
	}

	located := locateUser(h.baseURL(c), user)
	c.Header("Location", located.Meta.Location)
	h.respond(c, http.StatusCreated, located)
}

func (h *Handler) replaceUser(c *gin.Context) {
	body, err := h.readBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	resource, err := scimproto.DecodeUser(body)
	if err != nil {
		h.fail(c, err)
		return
	}

	user, err := h.provisioner.ReplaceUser(c.Request.Context(), directoryOf(c), pathID(c), resource)
	if err != nil {
		h.fail(c, err)
		return
	}

	h.respond(c, http.StatusOK, locateUser(h.baseURL(c), user))
}

func (h *Handler) patchUser(c *gin.Context) {
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

	user, err := h.provisioner.PatchUser(
		c.Request.Context(),
		directoryOf(c),
		pathID(c),
		patch.Operations,
	)
	if err != nil {
		h.fail(c, err)
		return
	}

	h.respond(c, http.StatusOK, locateUser(h.baseURL(c), user))
}

func (h *Handler) deleteUser(c *gin.Context) {
	if err := h.provisioner.DeleteUser(c.Request.Context(), directoryOf(c), pathID(c)); err != nil {
		h.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func locateUser(base string, user *scimproto.User) *scimproto.User {
	if user.Meta == nil {
		user.Meta = &scimproto.Meta{ResourceType: scimproto.ResourceTypeUser}
	}
	user.Meta.Location = base + "/Users/" + user.ID

	return user
}
