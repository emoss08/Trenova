package ratematrixhandler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/responseguard"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratematrixservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *ratematrixservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *ratematrixservice.Service
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
	resource := permission.ResourceRateMatrix.String()
	requireRead := h.pm.RequirePermission(resource, permission.OpRead)
	requireUpdate := h.pm.RequirePermission(resource, permission.OpUpdate)

	api := rg.Group("/rate-matrices")
	api.GET("/", requireRead, h.list)
	api.GET("/:rateMatrixID/", requireRead, h.get)
	api.POST("/", h.pm.RequirePermission(resource, permission.OpCreate), h.create)
	api.PUT("/:rateMatrixID/", requireUpdate, h.update)
	api.DELETE("/:rateMatrixID/", h.pm.RequirePermission(resource, permission.OpDelete), h.delete)

	api.GET("/:rateMatrixID/cells/", requireRead, h.listCells)
	api.PUT("/:rateMatrixID/cells/", requireUpdate, h.replaceCells)

	api.POST(
		"/:rateMatrixID/submit/",
		h.pm.RequirePermission(resource, permission.OpSubmit),
		h.submit,
	)
	api.POST(
		"/:rateMatrixID/approve/",
		h.pm.RequirePermission(resource, permission.OpApprove),
		h.approve,
	)
	api.POST(
		"/:rateMatrixID/reject/",
		h.pm.RequirePermission(resource, permission.OpReject),
		h.reject,
	)
	api.POST(
		"/:rateMatrixID/request-changes/",
		h.pm.RequirePermission(resource, permission.OpReject),
		h.requestChanges,
	)
	api.POST(
		"/:rateMatrixID/archive/",
		h.pm.RequirePermission(resource, permission.OpArchive),
		h.archive,
	)
	api.POST("/:rateMatrixID/restore/", requireUpdate, h.restore)
	api.GET("/:rateMatrixID/reviewers/", requireRead, h.reviewers)

	selectOptions := api.Group("/select-options", responseguard.References())
	selectOptions.GET("/", h.selectOptions)
}

type approvalActionRequest struct {
	Comment string `json:"comment"`
}

type submitRequest struct {
	Comment      string `json:"comment"`
	ReviewerID   string `json:"reviewerId"`
	ReviewerNote string `json:"reviewerNote"`
}

// @Summary Submit a rate matrix draft for review
// @Description A draft that differs from the published version, with cells that fit its axes, waits for a reviewer to publish it. An archived matrix is submitted to bring it back.
// @ID submitRateMatrix
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param request body submitRequest true "Comment, and optionally a reviewer and a note for them"
// @Success 200 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/submit [post]
func (h *Handler) submit(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	var req submitRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	reviewerID := pulid.Nil
	if raw := strings.TrimSpace(req.ReviewerID); raw != "" {
		parsed, parseErr := pulid.Parse(raw)
		if parseErr != nil {
			h.eh.HandleError(c, errortypes.NewValidationError(
				"reviewerId",
				errortypes.ErrInvalid,
				"Choose an active reviewer, other than you, who can publish rate matrices",
			))
			return
		}
		reviewerID = parsed
	}

	submitted, err := h.service.Submit(c.Request.Context(), &ratematrixservice.SubmitRequest{
		ApprovalActionRequest: ratematrixservice.ApprovalActionRequest{
			TenantInfo: tenantOf(authCtx),
			EntityID:   matrixID,
			Comment:    req.Comment,
		},
		ReviewerID:   reviewerID,
		ReviewerNote: req.ReviewerNote,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, submitted)
}

func (h *Handler) handleApprovalAction(
	c *gin.Context,
	action func(
		ctx context.Context,
		req *ratematrixservice.ApprovalActionRequest,
	) (*ratematrix.RateMatrix, error),
) {
	authCtx := authctx.GetAuthContext(c)
	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	var req approvalActionRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := action(c.Request.Context(), &ratematrixservice.ApprovalActionRequest{
		TenantInfo: tenantOf(authCtx),
		EntityID:   matrixID,
		Comment:    req.Comment,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, updated)
}

// @Summary Publish a rate matrix draft
// @Description Copies the submitted draft into the next published version. Tariffs read it once a formula template version that pins it is approved; contracts already rating keep the version they pinned.
// @ID approveRateMatrix
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param request body approvalActionRequest true "An optional comment"
// @Success 200 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/approve [post]
func (h *Handler) approve(c *gin.Context) {
	h.handleApprovalAction(c, h.service.Approve)
}

// @Summary Reject a rate matrix draft
// @ID rejectRateMatrix
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param request body approvalActionRequest true "Why it was rejected"
// @Success 200 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/reject [post]
func (h *Handler) reject(c *gin.Context) {
	h.handleApprovalAction(c, h.service.Reject)
}

// @Summary Request changes to a rate matrix draft
// @ID requestRateMatrixChanges
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param request body approvalActionRequest true "What needs to change"
// @Success 200 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/request-changes [post]
func (h *Handler) requestChanges(c *gin.Context) {
	h.handleApprovalAction(c, h.service.RequestChanges)
}

// @Summary Archive a rate matrix
// @Description An archived matrix is no longer read by new template versions; versions already pinned keep reading it.
// @ID archiveRateMatrix
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param request body ratematrixservice.ArchiveRequest true "Confirm when template versions pin it"
// @Success 200 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/archive [post]
func (h *Handler) archive(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	req := new(ratematrixservice.ArchiveRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(authCtx)
	req.RateMatrixID = matrixID

	archived, err := h.service.Archive(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, archived)
}

// @Summary Restore a rate matrix draft from a published version
// @Description Replaces the draft's axes and cells with a published version's. Restoring the newest published version discards the draft's changes.
// @ID restoreRateMatrixVersion
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param request body ratematrixservice.RestoreVersionRequest true "The published version to restore"
// @Success 200 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/restore [post]
func (h *Handler) restore(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	req := new(ratematrixservice.RestoreVersionRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(authCtx)
	req.RateMatrixID = matrixID

	restored, err := h.service.RestoreVersion(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, restored)
}

// @Summary List who can publish a rate matrix
// @ID listRateMatrixReviewers
// @Tags Rate Matrices
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Success 200 {array} ratematrixservice.Reviewer
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/reviewers [get]
func (h *Handler) reviewers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if _, err := h.matrixID(c); err != nil {
		return
	}

	reviewers, err := h.service.ListReviewers(c.Request.Context(), tenantOf(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, reviewers)
}

// @Summary List rate matrices
// @ID listRateMatrices
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratematrix.RateMatrix]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratematrix.RateMatrix], error) {
			return h.service.List(
				c.Request.Context(),
				&ratematrixports.ListRateMatrixRequest{Filter: req},
			)
		},
	)
}

// @Summary Get a rate matrix
// @ID getRateMatrix
// @Tags Rate Matrices
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Success 200 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	entity, err := h.service.GetByID(c.Request.Context(), &ratematrixports.GetRateMatrixByIDRequest{
		RateMatrixID:      matrixID,
		TenantInfo:        tenantOf(authCtx),
		IncludeDimensions: true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}

// @Summary Create a rate matrix
// @ID createRateMatrix
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param request body ratematrix.RateMatrix true "Rate matrix payload"
// @Success 201 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(ratematrix.RateMatrix)
	if err := authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, created)
}

// @Summary Update a rate matrix
// @ID updateRateMatrix
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param request body ratematrix.RateMatrix true "Rate matrix payload"
// @Success 200 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	entity := new(ratematrix.RateMatrix)
	if err = authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity.ID = matrixID

	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updated)
}

// @Summary Delete a rate matrix
// @ID deleteRateMatrix
// @Tags Rate Matrices
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID} [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	if err = h.service.Delete(c.Request.Context(), &ratematrixports.GetRateMatrixByIDRequest{
		RateMatrixID: matrixID,
		TenantInfo:   tenantOf(authCtx),
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary List a rate matrix's cells
// @ID listRateMatrixCells
// @Tags Rate Matrices
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param versionNumber query int false "A published version; 0 or absent reads the draft"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratematrix.RateMatrixCell]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/cells [get]
func (h *Handler) listCells(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	versionNumber := ratematrix.DraftVersion
	if raw := strings.TrimSpace(c.Query("versionNumber")); raw != "" {
		parsed, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || parsed < 0 {
			h.eh.HandleError(c, errortypes.NewValidationError(
				"versionNumber",
				errortypes.ErrInvalid,
				"Version number must be a published version, or 0 for the draft",
			))
			return
		}
		versionNumber = parsed
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
			return h.service.ListCells(
				c.Request.Context(),
				&ratematrixports.ListRateMatrixCellsRequest{
					TenantInfo:    tenantOf(authCtx),
					RateMatrixID:  matrixID,
					VersionNumber: versionNumber,
					Filter:        req,
				},
			)
		},
	)
}

type replaceCellsRequest struct {
	Cells []*ratematrix.RateMatrixCell `json:"cells"`
}

// @Summary Replace a rate matrix's cells
// @Description Swaps the entire grid for the one supplied. A tariff arrives as a whole sheet, and merging would leave behind whatever the new sheet dropped.
// @ID replaceRateMatrixCells
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param request body replaceCellsRequest true "Cells payload"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/cells [put]
func (h *Handler) replaceCells(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	body := new(replaceCellsRequest)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.ReplaceCells(
		c.Request.Context(),
		&ratematrixports.ReplaceRateMatrixCellsRequest{
			TenantInfo:   tenantOf(authCtx),
			RateMatrixID: matrixID,
			Cells:        body.Cells,
		},
		authCtx.UserID,
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary List rate matrix options
// @ID listRateMatrixOptions
// @Tags Rate Matrices
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratematrix.RateMatrix]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratematrix.RateMatrix], error) {
			return h.service.SelectOptions(c.Request.Context(), req)
		},
	)
}

func (h *Handler) matrixID(c *gin.Context) (pulid.ID, error) {
	matrixID, err := pulid.MustParse(c.Param("rateMatrixID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pulid.Nil, err
	}

	return matrixID, nil
}

func tenantOf(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
