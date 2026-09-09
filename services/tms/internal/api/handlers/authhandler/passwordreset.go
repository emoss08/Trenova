package authhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type forgotPasswordRequest struct {
	EmailAddress string `json:"emailAddress" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"       binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// forgotPassword always answers 202, whatever happened underneath.
//
// The service returns nil for an unknown address, a deactivated account and an account
// that has already had its hourly allowance of links, precisely so this handler cannot
// tell them apart either. Any response that varies with whether an address is
// registered turns this endpoint into an account-existence oracle.
func (h *Handler) forgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err := h.passwordReset.RequestReset(c.Request.Context(), req.EmailAddress); err != nil {
		// An infrastructure failure is logged by the service. Surfacing it here would
		// leak the same distinction the flow is built to hide, so the caller is told
		// what every caller is told.
		h.l.Error("password reset request failed")
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "If that address has an account, a reset link is on its way.",
	})
}

func (h *Handler) resetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err := h.passwordReset.ResetPassword(
		c.Request.Context(),
		req.Token,
		req.NewPassword,
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Your password has been changed. Sign in with your new password.",
	})
}
