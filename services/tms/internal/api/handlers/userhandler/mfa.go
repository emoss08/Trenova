package userhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func tenantInfoOf(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func (h *Handler) getMyMFA(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	status, err := h.mfa.Status(
		c.Request.Context(),
		tenantInfoOf(authCtx),
		authCtx.AuthenticatorAAL,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, status)
}

func (h *Handler) beginTOTPEnrollment(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req services.BeginTOTPEnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoOf(authCtx)

	enrollment, err := h.mfa.BeginTOTPEnrollment(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, enrollment)
}

func (h *Handler) confirmTOTPEnrollment(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req services.ConfirmTOTPEnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoOf(authCtx)
	req.SessionID = authCtx.SessionID

	resp, err := h.mfa.ConfirmTOTPEnrollment(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) disableTOTP(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req services.DisableTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoOf(authCtx)

	if err := h.mfa.DisableTOTP(c.Request.Context(), &req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) regenerateRecoveryCodes(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req services.RegenerateRecoveryCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoOf(authCtx)

	resp, err := h.mfa.RegenerateRecoveryCodes(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) resetUserMFA(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	targetUserID, err := pulid.MustParse(c.Param("userID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.mfa.ResetUserMFA(c.Request.Context(), &services.ResetUserMFARequest{
		TenantInfo:   tenantInfoOf(authCtx),
		TargetUserID: targetUserID,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
