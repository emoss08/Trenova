package formulalibraryhandler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulalibraryservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *formulalibraryservice.Service
	PageAssistant        agentports.PageAssistant `optional:"true"`
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service       *formulalibraryservice.Service
	pageAssistant agentports.PageAssistant
	eh            *helpers.ErrorHandler
	pm            *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:       p.Service,
		pageAssistant: p.PageAssistant,
		eh:            p.ErrorHandler,
		pm:            p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	resource := permission.ResourceFormulaLibrary.String()
	requireRead := h.pm.RequirePermission(resource, permission.OpRead)
	requireCreate := h.pm.RequirePermission(resource, permission.OpCreate)
	requireUpdate := h.pm.RequirePermission(resource, permission.OpUpdate)
	requireDuplicate := h.pm.RequirePermission(resource, permission.OpDuplicate)
	requireSubmit := h.pm.RequirePermission(resource, permission.OpSubmit)
	requireApprove := h.pm.RequirePermission(resource, permission.OpApprove)
	requireReject := h.pm.RequirePermission(resource, permission.OpReject)
	requireArchive := h.pm.RequirePermission(resource, permission.OpArchive)

	requireAssistant := h.pm.RequirePermission(
		permission.ResourceAssistant.String(),
		permission.OpCreate,
	)

	api := rg.Group("/formula-libraries")
	api.POST("/", requireCreate, h.create)
	api.POST("/duplicate", requireDuplicate, h.duplicate)
	api.POST("/ai/thread/", requireRead, requireAssistant, h.openAssistantThread)

	idGroup := api.Group("/:libraryID")
	idGroup.PUT("/", requireUpdate, h.update)
	idGroup.POST("/rollback", requireUpdate, h.rollback)
	idGroup.POST("/archive", requireArchive, h.archive)
	idGroup.POST("/submit", requireSubmit, h.submit)
	idGroup.POST("/approve", requireApprove, h.approve)
	idGroup.POST("/reject", requireReject, h.reject)
	idGroup.POST("/request-changes", requireReject, h.requestChanges)
	idGroup.GET("/readiness", requireRead, h.readiness)
	idGroup.GET("/review-diff", requireRead, h.reviewDiff)
	idGroup.GET("/compare", requireRead, h.compare)
	idGroup.GET("/usages", requireRead, h.usages)
	idGroup.GET("/reviewers", requireRead, h.reviewers)
}

func (h *Handler) libraryID(c *gin.Context) (pulid.ID, bool) {
	id, err := pulid.MustParse(c.Param("libraryID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pulid.Nil, false
	}
	return id, true
}

// @Summary Create a formula library
// @Description Checks the library's source with kind library, runs its tests, and stores it as a draft with its imports and test count.
// @ID createFormulaLibrary
// @Tags Formula Libraries
// @Accept json
// @Produce json
// @Param request body formulalibrary.FormulaLibrary true "Name, description and source"
// @Success 201 {object} formulalibrary.FormulaLibrary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(formulalibrary.FormulaLibrary)
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID
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

// @Summary Update a formula library
// @Description A change to the source or the name makes a new version and returns an approved or in-review library to draft. Approved templates keep the version they pinned.
// @ID updateFormulaLibrary
// @Tags Formula Libraries
// @Accept json
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Param request body formulalibrary.FormulaLibrary true "The library"
// @Success 200 {object} formulalibrary.FormulaLibrary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, ok := h.libraryID(c)
	if !ok {
		return
	}

	entity := new(formulalibrary.FormulaLibrary)
	entity.ID = id
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID
	if err := authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity.ID = id

	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, updated)
}

// @Summary Duplicate formula libraries
// @ID duplicateFormulaLibraries
// @Tags Formula Libraries
// @Accept json
// @Produce json
// @Param request body formulalibraryservice.DuplicateRequest true "The libraries to copy"
// @Success 200 {array} formulalibrary.FormulaLibrary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/duplicate [post]
func (h *Handler) duplicate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(formulalibraryservice.DuplicateRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.FromAuthAsUser(authCtx)

	created, err := h.service.Duplicate(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, created)
}

// @Summary Roll a formula library back to a version
// @ID rollbackFormulaLibrary
// @Tags Formula Libraries
// @Accept json
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Param request body formulalibraryservice.RollbackRequest true "The version to restore"
// @Success 200 {object} formulalibrary.FormulaLibrary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/rollback [post]
func (h *Handler) rollback(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, ok := h.libraryID(c)
	if !ok {
		return
	}

	req := new(formulalibraryservice.RollbackRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.FromAuthAsUser(authCtx)
	req.LibraryID = id

	updated, err := h.service.Rollback(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, updated)
}

// @Summary Archive a formula library
// @Description Stops new template versions importing the library. Approved template versions keep the version they pinned. A library templates import needs confirmInUse.
// @ID archiveFormulaLibrary
// @Tags Formula Libraries
// @Accept json
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Param request body formulalibraryservice.ArchiveRequest true "Confirmation"
// @Success 200 {object} formulalibrary.FormulaLibrary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/archive [post]
func (h *Handler) archive(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, ok := h.libraryID(c)
	if !ok {
		return
	}

	req := new(formulalibraryservice.ArchiveRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.FromAuthAsUser(authCtx)
	req.LibraryID = id

	updated, err := h.service.Archive(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, updated)
}

type approvalActionRequest struct {
	Comment string `json:"comment"`
}

type submitRequest struct {
	Comment      string `json:"comment"`
	ReviewerID   string `json:"reviewerId"`
	ReviewerNote string `json:"reviewerNote"`
}

// @Summary Submit a formula library for review
// @Description Refused unless the source checks clean and every test passes.
// @ID submitFormulaLibrary
// @Tags Formula Libraries
// @Accept json
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Param request body submitRequest true "Submit request"
// @Success 200 {object} formulalibrary.FormulaLibrary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/submit [post]
func (h *Handler) submit(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, ok := h.libraryID(c)
	if !ok {
		return
	}

	var req submitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	reviewerID := pulid.Nil
	if raw := strings.TrimSpace(req.ReviewerID); raw != "" {
		parsed, err := pulid.Parse(raw)
		if err != nil {
			h.eh.HandleError(c, errortypes.NewValidationError(
				"reviewerId",
				errortypes.ErrInvalid,
				"Choose an active reviewer, other than you, who can approve formula libraries",
			))
			return
		}
		reviewerID = parsed
	}

	submitted, err := h.service.Submit(c.Request.Context(), &formulalibraryservice.SubmitRequest{
		ApprovalActionRequest: formulalibraryservice.ApprovalActionRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			EntityID:   id,
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
		req *formulalibraryservice.ApprovalActionRequest,
	) (*formulalibrary.FormulaLibrary, error),
) {
	authCtx := authctx.GetAuthContext(c)
	id, ok := h.libraryID(c)
	if !ok {
		return
	}

	var req approvalActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	library, err := action(c.Request.Context(), &formulalibraryservice.ApprovalActionRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		EntityID:   id,
		Comment:    req.Comment,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, library)
}

// @Summary Approve a formula library
// @Description Approves the library's current content as a new version. Templates pin it the next time one of their versions is approved; nothing already approved reprices.
// @ID approveFormulaLibrary
// @Tags Formula Libraries
// @Accept json
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Param request body approvalActionRequest true "Approval comment"
// @Success 200 {object} formulalibrary.FormulaLibrary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/approve [post]
func (h *Handler) approve(c *gin.Context) {
	h.handleApprovalAction(c, h.service.Approve)
}

// @Summary Reject a formula library
// @ID rejectFormulaLibrary
// @Tags Formula Libraries
// @Accept json
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Param request body approvalActionRequest true "Why it was rejected"
// @Success 200 {object} formulalibrary.FormulaLibrary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/reject [post]
func (h *Handler) reject(c *gin.Context) {
	h.handleApprovalAction(c, h.service.Reject)
}

// @Summary Request changes on a formula library
// @ID requestFormulaLibraryChanges
// @Tags Formula Libraries
// @Accept json
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Param request body approvalActionRequest true "What should change"
// @Success 200 {object} formulalibrary.FormulaLibrary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/request-changes [post]
func (h *Handler) requestChanges(c *gin.Context) {
	h.handleApprovalAction(c, h.service.RequestChanges)
}

// @Summary A formula library's readiness for review
// @ID getFormulaLibraryReadiness
// @Tags Formula Libraries
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Success 200 {object} formulalibraryservice.ReadinessResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/readiness [get]
func (h *Handler) readiness(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, ok := h.libraryID(c)
	if !ok {
		return
	}

	resp, err := h.service.Readiness(c.Request.Context(), &formulalibraryservice.ReadinessRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		LibraryID:  id,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary What a reviewer of a formula library is approving
// @ID getFormulaLibraryReviewDiff
// @Tags Formula Libraries
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Success 200 {object} formulalibraryservice.ReviewDiffResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/review-diff [get]
func (h *Handler) reviewDiff(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, ok := h.libraryID(c)
	if !ok {
		return
	}

	resp, err := h.service.ReviewDiff(c.Request.Context(), &formulalibraryservice.ReviewDiffRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		LibraryID:  id,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary Compare two versions of a formula library
// @ID compareFormulaLibraryVersions
// @Tags Formula Libraries
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Param from query int true "From version"
// @Param to query int true "To version"
// @Success 200 {object} formulalibraryservice.VersionDiff
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/compare [get]
func (h *Handler) compare(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, ok := h.libraryID(c)
	if !ok {
		return
	}

	from, fromErr := strconv.ParseInt(c.Query("from"), 10, 64)
	to, toErr := strconv.ParseInt(c.Query("to"), 10, 64)
	if fromErr != nil || toErr != nil || from < 1 || to < 1 {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"versions",
			errortypes.ErrInvalid,
			"from and to must be version numbers",
		))
		return
	}

	resp, err := h.service.CompareVersions(
		c.Request.Context(),
		&formulalibraryports.GetLibraryVersionRangeRequest{
			TenantInfo:  pagination.FromAuthAsUser(authCtx),
			LibraryID:   id,
			FromVersion: from,
			ToVersion:   to,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary The template versions that import a formula library
// @ID listFormulaLibraryUsages
// @Tags Formula Libraries
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Param versionNumber query int false "Only template versions pinning this library version"
// @Success 200 {array} formulatemplateports.LibraryUsage
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/usages [get]
func (h *Handler) usages(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, ok := h.libraryID(c)
	if !ok {
		return
	}

	var versionNumber *int64
	if raw := strings.TrimSpace(c.Query("versionNumber")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 1 {
			h.eh.HandleError(c, errortypes.NewValidationError(
				"versionNumber",
				errortypes.ErrInvalid,
				"versionNumber must be a version number",
			))
			return
		}
		versionNumber = &parsed
	}

	resp, err := h.service.ListUsages(c.Request.Context(), &formulalibraryservice.ListUsagesRequest{
		TenantInfo:    pagination.FromAuthAsUser(authCtx),
		LibraryID:     id,
		VersionNumber: versionNumber,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary List who may review a formula library
// @ID listFormulaLibraryReviewers
// @Tags Formula Libraries
// @Produce json
// @Param libraryID path string true "Formula library ID"
// @Success 200 {array} formulalibraryservice.Reviewer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/{libraryID}/reviewers [get]
func (h *Handler) reviewers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if _, ok := h.libraryID(c); !ok {
		return
	}

	resp, err := h.service.ListReviewers(c.Request.Context(), pagination.FromAuthAsUser(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, resp)
}
