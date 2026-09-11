package invoicerunhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoicerunservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *invoicerunservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *invoicerunservice.Service
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
	api := rg.Group("/billing/invoice-runs")

	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceInvoiceRun.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:runID/",
		h.pm.RequirePermission(permission.ResourceInvoiceRun.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/preview/",
		h.pm.RequirePermission(permission.ResourceInvoiceRun.String(), permission.OpCreate),
		h.preview,
	)
	api.PATCH(
		"/:runID/membership/",
		h.pm.RequirePermission(permission.ResourceInvoiceRun.String(), permission.OpUpdate),
		h.adjustMembership,
	)
	// Issuing invoices to customers is a separable authority from building the
	// preview, so committing takes Approve rather than Create.
	api.POST(
		"/:runID/commit/",
		h.pm.RequirePermission(permission.ResourceInvoiceRun.String(), permission.OpApprove),
		h.commit,
	)
	api.POST(
		"/:runID/cancel/",
		h.pm.RequirePermission(permission.ResourceInvoiceRun.String(), permission.OpCancel),
		h.cancel,
	)
}

func tenantInfoFrom(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}
}

type previewRequest struct {
	CustomerIDs []pulid.ID `json:"customerIds"`
	PeriodStart int64      `json:"periodStart"`
	PeriodEnd   int64      `json:"periodEnd"`
	InvoiceDate int64      `json:"invoiceDate"`
}

type exclusionRequest struct {
	ItemID pulid.ID `json:"itemId"`
	Reason string   `json:"reason"`
}

type moveRequest struct {
	ItemID        pulid.ID `json:"itemId"`
	TargetGroupID pulid.ID `json:"targetGroupId"`
}

type membershipRequest struct {
	Exclude []exclusionRequest `json:"exclude"`
	Include []pulid.ID         `json:"include"`
	Moves   []moveRequest      `json:"moves"`
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

// @Summary List invoice runs
// @ID listInvoiceRuns
// @Tags Invoice Run
// @Produce json
// @Success 200 {object} pagination.Response[[]invoicerun.InvoiceRun]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/invoice-runs/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*invoicerun.InvoiceRun], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListInvoiceRunsRequest{Filter: req},
			)
		},
	)
}

// @Summary Get an invoice run
// @ID getInvoiceRun
// @Tags Invoice Run
// @Produce json
// @Param runID path string true "Invoice run ID"
// @Success 200 {object} invoicerun.InvoiceRun
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/invoice-runs/{runID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	runID, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	run, err := h.service.Get(c.Request.Context(), repositories.GetInvoiceRunByIDRequest{
		ID:            runID,
		TenantInfo:    tenantInfoFrom(authCtx),
		IncludeGroups: true,
		IncludeItems:  true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, run)
}

// @Summary Build an invoice run preview
// @ID previewInvoiceRun
// @Tags Invoice Run
// @Accept json
// @Produce json
// @Param request body previewRequest true "Preview request"
// @Success 200 {object} invoicerun.InvoiceRun
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/invoice-runs/preview/ [post]
func (h *Handler) preview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req previewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"request",
			errortypes.ErrInvalid,
			"Invalid request body",
		))
		return
	}

	run, err := h.service.Preview(
		c.Request.Context(),
		&services.PreviewInvoiceRunRequest{
			TenantInfo:  tenantInfoFrom(authCtx),
			CustomerIDs: req.CustomerIDs,
			PeriodStart: req.PeriodStart,
			PeriodEnd:   req.PeriodEnd,
			InvoiceDate: req.InvoiceDate,
			Source:      invoicerun.SourceManual,
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, run)
}

// @Summary Adjust invoice run membership
// @ID adjustInvoiceRunMembership
// @Tags Invoice Run
// @Accept json
// @Produce json
// @Param runID path string true "Invoice run ID"
// @Param request body membershipRequest true "Membership changes"
// @Success 200 {object} invoicerun.InvoiceRun
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/invoice-runs/{runID}/membership/ [patch]
func (h *Handler) adjustMembership(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	runID, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req membershipRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"request",
			errortypes.ErrInvalid,
			"Invalid request body",
		))
		return
	}

	exclusions := make([]services.ItemExclusion, 0, len(req.Exclude))
	for _, exclusion := range req.Exclude {
		exclusions = append(exclusions, services.ItemExclusion{
			ItemID: exclusion.ItemID,
			Reason: exclusion.Reason,
		})
	}
	moves := make([]services.ItemMove, 0, len(req.Moves))
	for _, move := range req.Moves {
		moves = append(moves, services.ItemMove{
			ItemID:        move.ItemID,
			TargetGroupID: move.TargetGroupID,
		})
	}

	run, err := h.service.AdjustMembership(
		c.Request.Context(),
		&services.AdjustInvoiceRunMembershipRequest{
			TenantInfo: tenantInfoFrom(authCtx),
			RunID:      runID,
			Exclude:    exclusions,
			Include:    req.Include,
			Moves:      moves,
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, run)
}

// @Summary Commit an invoice run
// @ID commitInvoiceRun
// @Tags Invoice Run
// @Produce json
// @Param runID path string true "Invoice run ID"
// @Success 200 {object} services.CommitInvoiceRunResult
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/invoice-runs/{runID}/commit/ [post]
func (h *Handler) commit(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	runID, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.Commit(
		c.Request.Context(),
		&services.CommitInvoiceRunRequest{
			TenantInfo: tenantInfoFrom(authCtx),
			RunID:      runID,
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Cancel an invoice run
// @ID cancelInvoiceRun
// @Tags Invoice Run
// @Accept json
// @Produce json
// @Param runID path string true "Invoice run ID"
// @Param request body cancelRequest true "Cancel reason"
// @Success 200 {object} invoicerun.InvoiceRun
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/invoice-runs/{runID}/cancel/ [post]
func (h *Handler) cancel(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	runID, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req cancelRequest
	_ = c.ShouldBindJSON(&req)

	run, err := h.service.Cancel(
		c.Request.Context(),
		&services.CancelInvoiceRunRequest{
			TenantInfo: tenantInfoFrom(authCtx),
			RunID:      runID,
			Reason:     req.Reason,
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, run)
}
