package rateimporthandler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateimportservice"
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
	api.GET("/template/", h.pm.RequirePermission(resource, permission.OpRead), h.template)
	api.POST("/", h.pm.RequirePermission(resource, permission.OpUpdate), h.upload)
}

// TemplateResponse is the starter sheet, ready to be saved as a file.
type TemplateResponse struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

// @Summary Download the rate sheet template
// @Description Returns a starter CSV whose columns the importer recognises, with two example rows showing how lanes are described.
// @ID rateSheetTemplate
// @Tags Rate Imports
// @Produce json
// @Success 200 {object} rateimporthandler.TemplateResponse
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-imports/template/ [get]
func (h *Handler) template(c *gin.Context) {
	fileName, content := h.service.Template()

	helpers.JSON(c, http.StatusOK, TemplateResponse{FileName: fileName, Content: content})
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

	helpers.JSON(c, http.StatusCreated, created)
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

func tenantOf(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
