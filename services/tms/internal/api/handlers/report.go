package handlers

import (
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/reportservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ReportHandlerParams struct {
	fx.In

	Logger       *zap.Logger
	Service      *reportservice.Service
	MinIOClient  *minio.Client
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type ReportHandler struct {
	logger       *zap.Logger
	service      *reportservice.Service
	minioClient  *minio.Client
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewReportHandler(p ReportHandlerParams) *ReportHandler {
	return &ReportHandler{
		logger:       p.Logger.Named("handler.report"),
		service:      p.Service,
		minioClient:  p.MinIOClient,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *ReportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/reports/")
	api.GET("", h.list)
	api.GET(":id/", h.get)
	api.POST("generate/", h.generate)
	api.GET(":id/download/", h.download)
	api.DELETE(":id/", h.delete)
}

func (h *ReportHandler) list(c *gin.Context) {
	pagination.Handle[*report.Report](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*report.Report], error) {
			return h.service.List(c.Request.Context(), &repositories.ListReportRequest{
				Filter: opts,
			})
		})
}

func (h *ReportHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetReportByIDRequest{
			ReportID: id,
			OrgID:    authCtx.OrganizationID,
			BuID:     authCtx.BusinessUnitID,
			UserID:   authCtx.UserID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

type GenerateReportRequest struct {
	ResourceType   string                  `json:"resourceType"             binding:"required"`
	Name           string                  `json:"name"                     binding:"required"`
	Format         string                  `json:"format"                   binding:"required,oneof=CSV EXCEL"`
	DeliveryMethod string                  `json:"deliveryMethod"           binding:"required,oneof=DOWNLOAD EMAIL"`
	FilterState    pagination.QueryOptions `json:"filterState"`
	EmailProfileID *pulid.ID               `json:"emailProfileId,omitempty"`
}

func (h *ReportHandler) generate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	format, err := report.FormatFromString(req.Format)
	if err != nil {
		h.errorHandler.HandleError(c, fmt.Errorf("invalid format: %w", err))
		return
	}

	deliveryMethod, err := report.DeliveryMethodFromString(req.DeliveryMethod)
	if err != nil {
		h.errorHandler.HandleError(c, fmt.Errorf("invalid delivery method: %w", err))
		return
	}

	userEmail := c.GetString("userEmail")
	if userEmail == "" {
		h.logger.Warn("user email not found in context, email delivery may fail")
	}

	rpt, err := h.service.GenerateReport(c.Request.Context(), &reportservice.GenerateReportRequest{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		UserEmail:      userEmail,
		ResourceType:   req.ResourceType,
		Name:           req.Name,
		Format:         format,
		DeliveryMethod: deliveryMethod,
		FilterState:    req.FilterState,
		EmailProfileID: req.EmailProfileID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Report generation started",
		"reportId": rpt.ID,
		"status":   rpt.Status,
	})
}

func (h *ReportHandler) download(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	rpt, err := h.service.Get(
		c.Request.Context(),
		repositories.GetReportByIDRequest{
			ReportID: id,
			OrgID:    authCtx.OrganizationID,
			BuID:     authCtx.BusinessUnitID,
			UserID:   authCtx.UserID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	if rpt.Status != report.StatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Report is not ready for download",
			"status": rpt.Status,
		})
		return
	}

	if rpt.FilePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Report file not found",
		})
		return
	}

	bucketName := "trenova-reports"
	object, err := h.minioClient.GetObject(
		c.Request.Context(),
		bucketName,
		rpt.FilePath,
		minio.GetObjectOptions{},
	)
	if err != nil {
		h.logger.Error("failed to get object from MinIO",
			zap.Error(err),
			zap.String("reportID", id.String()),
			zap.String("filePath", rpt.FilePath),
		)
		h.errorHandler.HandleError(c, fmt.Errorf("failed to retrieve report file: %w", err))
		return
	}
	defer object.Close()

	stat, err := object.Stat()
	if err != nil {
		h.logger.Error("failed to stat object",
			zap.Error(err),
			zap.String("reportID", id.String()),
		)
		h.errorHandler.HandleError(c, fmt.Errorf("failed to retrieve report file info: %w", err))
		return
	}

	contentType := "text/csv"
	if rpt.Format == report.FormatExcel {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", rpt.Name))
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))

	c.DataFromReader(http.StatusOK, stat.Size, contentType, object, nil)
}

func (h *ReportHandler) delete(c *gin.Context) {
	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
