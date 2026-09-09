package userhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/services/passwordresetservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// sendPasswordReset mails a reset link to another user, on an administrator's behalf.
//
// Unlike the unauthenticated /auth/forgot-password, this reports what happened: the
// caller has already been authenticated and permission-checked, so there is no
// account-existence oracle to protect, and an admin watching a button succeed while
// nothing was sent is worse than an error.
//
// It mints a link and nothing more. An administrator cannot set, see or choose the
// password that results, and clicking it cannot lock the user out — their current
// password keeps working until they redeem the link themselves.
func (h *Handler) sendPasswordReset(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	targetUserID, err := pulid.MustParse(c.Param("userID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.passwordReset.RequestResetForUser(
		c.Request.Context(),
		passwordresetservice.AdminResetRequest{
			TargetUserID: targetUserID,
			Actor: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		},
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "A password reset link has been sent."})
}
