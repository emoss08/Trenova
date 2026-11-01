package userhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/gin-gonic/gin"
)

func respondWithSelf(c *gin.Context, user *tenant.User) {
	authCtx := authctx.GetAuthContext(c)
	user.CurrentOrganizationID = authCtx.OrganizationID
	user.BusinessUnitID = authCtx.BusinessUnitID
	helpers.JSON(c, http.StatusOK, user)
}
