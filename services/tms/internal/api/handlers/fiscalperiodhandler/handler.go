package fiscalperiodhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *fiscalperiodservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *fiscalperiodservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/fiscal-periods")
	api.PUT(
		"/:fiscalPeriodID/close/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpClose),
		h.close,
	)
	api.GET(
		"/:fiscalPeriodID/close-blockers/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpRead),
		h.closeBlockers,
	)
	api.PUT(
		"/:fiscalPeriodID/reopen/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpReopen),
		h.reopen,
	)
	api.PUT(
		"/:fiscalPeriodID/lock/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpLock),
		h.lock,
	)
	api.PUT(
		"/:fiscalPeriodID/unlock/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpUnlock),
		h.unlock,
	)
	api.PUT(
		"/:fiscalPeriodID/activate/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpActivate),
		h.activate,
	)
}

func (h *Handler) closeBlockers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.GetCloseBlockers(
		c.Request.Context(),
		fiscalperiodports.GetFiscalPeriodByIDRequest{
			ID:         fiscalPeriodID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

// @Summary Close a fiscal period
// @ID closeFiscalPeriod
// @Tags Fiscal Periods
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/close/ [put]
func (h *Handler) close(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Close(c.Request.Context(), fiscalperiodports.CloseFiscalPeriodRequest{
		ID:         fiscalPeriodID,
		TenantInfo: pagination.FromAuthAsUser(authCtx),
	}, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}

type reopenFiscalPeriodPayload struct {
	ReopenReason string `json:"reopenReason"`
}

// @Summary Reopen a fiscal period
// @ID reopenFiscalPeriod
// @Tags Fiscal Periods
// @Accept json
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Param request body reopenFiscalPeriodPayload true "Reopen payload"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/reopen/ [put]
func (h *Handler) reopen(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	payload := new(reopenFiscalPeriodPayload)
	if err = c.ShouldBindJSON(payload); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Reopen(
		c.Request.Context(),
		fiscalperiodports.ReopenFiscalPeriodRequest{
			ID:           fiscalPeriodID,
			TenantInfo:   pagination.FromAuthAsUser(authCtx),
			ReopenReason: payload.ReopenReason,
		},
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}

// @Summary Lock a fiscal period
// @ID lockFiscalPeriod
// @Tags Fiscal Periods
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/lock/ [put]
func (h *Handler) lock(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Lock(c.Request.Context(), &fiscalperiodports.LockFiscalPeriodRequest{
		ID:         fiscalPeriodID,
		TenantInfo: pagination.FromAuthAsUser(authCtx),
	}, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}

// @Summary Unlock a fiscal period
// @ID unlockFiscalPeriod
// @Tags Fiscal Periods
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/unlock/ [put]
func (h *Handler) unlock(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Unlock(
		c.Request.Context(),
		fiscalperiodports.UnlockFiscalPeriodRequest{
			ID:         fiscalPeriodID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}

// @Summary Open an inactive fiscal period
// @ID activateFiscalPeriod
// @Tags Fiscal Periods
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/activate/ [put]
func (h *Handler) activate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Activate(
		c.Request.Context(),
		fiscalperiodports.ActivateFiscalPeriodRequest{
			ID:         fiscalPeriodID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}
