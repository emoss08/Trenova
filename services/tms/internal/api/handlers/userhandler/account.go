package userhandler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const maxAvatarIDsParam = 100

type UpdateMyProfileRequest struct {
	JobTitle    *string             `json:"jobTitle"`
	PhoneNumber *string             `json:"phoneNumber"`
	AvatarStyle *tenant.AvatarStyle `json:"avatarStyle"`
}

type UpdateMyAvailabilityRequest struct {
	Status         tenant.AvailabilityStatus `json:"status"`
	FocusUntil     *int64                    `json:"focusUntil"`
	BackOn         *int64                    `json:"backOn"`
	DelegateUserID *string                   `json:"delegateUserId"`
	AwayNote       string                    `json:"awayNote"`
}

type UpdateMyAssistantVisibilityRequest struct {
	AssistantHidden *bool `json:"assistantHidden"`
}

type RevokeSessionsResponse struct {
	Revoked int `json:"revoked"`
}

func selfTenant(c *gin.Context) pagination.TenantInfo {
	authCtx := authctx.GetAuthContext(c)
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func sessionScope(c *gin.Context) userservice.SessionScope {
	return userservice.SessionScope{
		TenantInfo:       selfTenant(c),
		CurrentSessionID: authctx.GetAuthContext(c).SessionID,
	}
}

// @Summary Update current user profile
// @ID updateCurrentUserProfile
// @Tags Users
// @Accept json
// @Produce json
// @Param request body UpdateMyProfileRequest true "Profile fields to change"
// @Success 200 {object} tenant.User
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/profile/ [patch]
func (h *Handler) updateMyProfile(c *gin.Context) {
	var req UpdateMyProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.UpdateMyProfile(c.Request.Context(), selfTenant(c), tenant.ProfileUpdate{
		JobTitle:    req.JobTitle,
		PhoneNumber: req.PhoneNumber,
		AvatarStyle: req.AvatarStyle,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Show or hide the assistant for the current user
// @ID updateCurrentUserAssistantVisibility
// @Tags Users
// @Accept json
// @Produce json
// @Param request body UpdateMyAssistantVisibilityRequest true "Whether the assistant is hidden"
// @Success 200 {object} tenant.User
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/assistant-visibility/ [put]
func (h *Handler) updateMyAssistantVisibility(c *gin.Context) {
	var req UpdateMyAssistantVisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if req.AssistantHidden == nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"assistantHidden",
			errortypes.ErrRequired,
			"Whether the assistant is hidden is required",
		))
		return
	}

	updated, err := h.service.SetMyAssistantHidden(
		c.Request.Context(),
		selfTenant(c),
		*req.AssistantHidden,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Update current user availability
// @ID updateCurrentUserAvailability
// @Tags Users
// @Accept json
// @Produce json
// @Param request body UpdateMyAvailabilityRequest true "Availability"
// @Success 200 {object} tenant.User
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/availability/ [put]
func (h *Handler) updateMyAvailability(c *gin.Context) {
	var req UpdateMyAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	update := tenant.AvailabilityUpdate{
		Status:     req.Status,
		FocusUntil: req.FocusUntil,
		BackOn:     req.BackOn,
		AwayNote:   req.AwayNote,
	}
	if req.DelegateUserID != nil && strings.TrimSpace(*req.DelegateUserID) != "" {
		delegateID, err := pulid.MustParse(strings.TrimSpace(*req.DelegateUserID))
		if err != nil {
			h.eh.HandleError(c, errortypes.NewValidationError(
				"delegateUserId",
				errortypes.ErrInvalid,
				"Hand off to a person in your organization",
			))
			return
		}
		update.DelegateUserID = &delegateID
	}

	updated, err := h.service.UpdateMyAvailability(c.Request.Context(), selfTenant(c), update)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary List avatar styles and availability for teammates
// @ID listUserAvatars
// @Tags Users
// @Produce json
// @Param ids query string true "Comma-separated user IDs (at most 100)"
// @Success 200 {array} userservice.AvatarSummary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/avatars/ [get]
func (h *Handler) listAvatars(c *gin.Context) {
	raw := strings.Split(c.Query("ids"), ",")
	if len(raw) > maxAvatarIDsParam {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"ids",
			errortypes.ErrInvalid,
			"Ask for at most 100 people at a time",
		))
		return
	}

	ids := make([]pulid.ID, 0, len(raw))
	for _, value := range raw {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		id, err := pulid.MustParse(trimmed)
		if err != nil {
			h.eh.HandleError(c, errortypes.NewValidationError(
				"ids",
				errortypes.ErrInvalid,
				"Every id must be a user id",
			))
			return
		}
		ids = append(ids, id)
	}

	avatars, err := h.service.ListAvatars(c.Request.Context(), selfTenant(c), ids)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, avatars)
}

// @Summary List the current user's sessions
// @ID listCurrentUserSessions
// @Tags Users
// @Produce json
// @Success 200 {array} userservice.SessionSummary
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/sessions/ [get]
func (h *Handler) listMySessions(c *gin.Context) {
	sessions, err := h.service.ListMySessions(c.Request.Context(), sessionScope(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// @Summary Sign out one of the current user's other sessions
// @ID revokeCurrentUserSession
// @Tags Users
// @Param sessionID path string true "Session ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/sessions/{sessionID}/ [delete]
func (h *Handler) revokeMySession(c *gin.Context) {
	sessionID, err := pulid.MustParse(c.Param("sessionID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewNotFoundError("Session not found"))
		return
	}

	if err = h.service.RevokeMySession(c.Request.Context(), sessionScope(c), sessionID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Sign out every session but the current one
// @ID revokeCurrentUserOtherSessions
// @Tags Users
// @Produce json
// @Success 200 {object} userhandler.RevokeSessionsResponse
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/sessions/revoke-others/ [post]
func (h *Handler) revokeMyOtherSessions(c *gin.Context) {
	revoked, err := h.service.RevokeMyOtherSessions(c.Request.Context(), sessionScope(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, RevokeSessionsResponse{Revoked: revoked})
}

// @Summary List the current user's recent sign-ins
// @ID listCurrentUserSignIns
// @Tags Users
// @Produce json
// @Param limit query int false "How many (default 5, at most 50)"
// @Success 200 {array} userservice.SignInSummary
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/sign-ins/ [get]
func (h *Handler) listMySignIns(c *gin.Context) {
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			h.eh.HandleError(c, errortypes.NewValidationError(
				"limit",
				errortypes.ErrInvalid,
				"Limit must be a positive number",
			))
			return
		}
		limit = parsed
	}

	signIns, err := h.service.ListMySignIns(c.Request.Context(), selfTenant(c), limit)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, signIns)
}
