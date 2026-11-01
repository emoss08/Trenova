package ediladinghandler

import (
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const maxDraftBodyBytes = 2 * (edilading.MaxProjectBytes + edilading.MaxSamplesBytes)

func (h *Handler) registerLadingRoutes(lading *gin.RouterGroup) {
	lading.PUT(
		"/versions/:versionID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.saveLadingDraft,
	)
}

func (h *Handler) saveLadingDraft(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	body, err := helpers.ReadBoundedRequestBody(c, maxDraftBodyBytes)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(edilading.SaveDraftRequest)
	if err = sonic.Unmarshal(body, req); err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"body",
			errortypes.ErrInvalid,
			"The draft is not valid JSON",
		))
		return
	}
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.projects.SaveDraft(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, version)
}
