package rateimporthandler

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateimport"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/rateimportservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *rateimportservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *rateimportservice.Service
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
	resource := permission.ResourceRateAgreement.String()

	api := rg.Group("/rate-imports")
	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/:rateImportID/", h.pm.RequirePermission(resource, permission.OpRead), h.get)
	api.GET("/:rateImportID/rows/", h.pm.RequirePermission(resource, permission.OpRead), h.rows)

	// Uploading stages a dry run and changes nothing, so it needs no more than
	// permission to edit the agreement it will eventually amend.
	api.POST("/", h.pm.RequirePermission(resource, permission.OpUpdate), h.upload)
	api.POST(
		"/:rateImportID/commit/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.commit,
	)
	api.POST(
		"/:rateImportID/discard/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.discard,
	)
}

// @Summary List rate imports
// @ID listRateImports
// @Tags Rate Imports
// @Produce json
// @Param rateAgreementId query string false "Narrow to one agreement's imports"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]rateimport.RateImportBatch]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-imports/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	var agreementID *pulid.ID
	if raw := helpers.QueryString(c, "rateAgreementId"); raw != "" {
		parsed, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		agreementID = &parsed
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*rateimport.RateImportBatch], error) {
			return h.service.List(c.Request.Context(), &repositories.ListRateImportBatchesRequest{
				Filter:          req,
				RateAgreementID: agreementID,
			})
		},
	)
}

// @Summary Get a rate import
// @Description Returns the staged import and its dry run: what committing this sheet would do to each lane.
// @ID getRateImport
// @Tags Rate Imports
// @Produce json
// @Param rateImportID path string true "Rate import ID"
// @Success 200 {object} rateimport.RateImportBatch
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-imports/{rateImportID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	importID, err := h.importID(c)
	if err != nil {
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		&repositories.GetRateImportBatchByIDRequest{
			RateImportBatchID: importID,
			TenantInfo:        tenantOf(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary List a rate import's rows
// @ID listRateImportRows
// @Tags Rate Imports
// @Produce json
// @Param rateImportID path string true "Rate import ID"
// @Param failedOnly query bool false "Only the rows that could not be read"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]rateimport.RateImportRow]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-imports/{rateImportID}/rows [get]
func (h *Handler) rows(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	importID, err := h.importID(c)
	if err != nil {
		return
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*rateimport.RateImportRow], error) {
			return h.service.ListRows(
				c.Request.Context(),
				&repositories.ListRateImportRowsRequest{
					RateImportBatchID: importID,
					TenantInfo:        tenantOf(authCtx),
					Filter:            req,
					FailedOnly:        helpers.QueryBool(c, "failedOnly", false),
				},
			)
		},
	)
}

// @Summary Upload a rate sheet
// @Description Reads a CSV or XLSX rate sheet and stages what committing it would do. Nothing about the agreement changes: applying it takes a second, deliberate call.
// @ID uploadRateSheet
// @Tags Rate Imports
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "The rate sheet"
// @Param rateAgreementId formData string true "The agreement to import into"
// @Param effectiveFrom formData int true "The day the imported rules start pricing"
// @Success 201 {object} rateimport.RateImportBatch
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-imports/ [post]
func (h *Handler) upload(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	header, err := c.FormFile("file")
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	agreementID, err := pulid.MustParse(c.PostForm("rateAgreementId"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	content, err := readUpload(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.Upload(c.Request.Context(), &rateimportservice.UploadRequest{
		TenantInfo:      tenantOf(authCtx),
		RateAgreementID: agreementID,
		FileName:        header.Filename,
		Content:         content,
		EffectiveFrom:   effectiveFrom(c),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// effectiveFrom is the day the imported rules start pricing.
//
// It arrives as a form field rather than a query parameter because the upload
// is multipart, and parsing it here keeps the handler's failure — a date that
// is not a number — in the same place as the file's.
func effectiveFrom(c *gin.Context) int64 {
	parsed, err := strconv.ParseInt(c.PostForm("effectiveFrom"), 10, 64)
	if err != nil {
		return 0
	}

	return parsed
}

func readUpload(c *gin.Context) ([]byte, error) {
	header, err := c.FormFile("file")
	if err != nil {
		return nil, err
	}

	opened, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = opened.Close() }()

	return io.ReadAll(opened)
}

// @Summary Apply a reviewed rate import
// @Description Amends the agreement with the sheet's lanes, closing out the ones it replaces. History is never mutated.
// @ID commitRateImport
// @Tags Rate Imports
// @Produce json
// @Param rateImportID path string true "Rate import ID"
// @Success 200 {object} rateimport.RateImportBatch
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-imports/{rateImportID}/commit [post]
func (h *Handler) commit(c *gin.Context) {
	h.transition(c, h.service.Commit)
}

// @Summary Discard a rate import
// @Description Closes an import somebody read and said no to.
// @ID discardRateImport
// @Tags Rate Imports
// @Produce json
// @Param rateImportID path string true "Rate import ID"
// @Success 200 {object} rateimport.RateImportBatch
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-imports/{rateImportID}/discard [post]
func (h *Handler) discard(c *gin.Context) {
	h.transition(c, h.service.Discard)
}

// transition runs the two steps that differ only in which service call they
// make, so they share one path rather than two that could drift.
func (h *Handler) transition(
	c *gin.Context,
	fn func(
		ctx context.Context,
		req *rateimportservice.CommitRequest,
	) (*rateimport.RateImportBatch, error),
) {
	authCtx := authctx.GetAuthContext(c)

	importID, err := h.importID(c)
	if err != nil {
		return
	}

	entity, err := fn(c.Request.Context(), &rateimportservice.CommitRequest{
		TenantInfo:        tenantOf(authCtx),
		RateImportBatchID: importID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) importID(c *gin.Context) (pulid.ID, error) {
	importID, err := pulid.MustParse(c.Param("rateImportID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pulid.Nil, err
	}

	return importID, nil
}

func tenantOf(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
