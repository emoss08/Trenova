package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/reportservice"
	"github.com/emoss08/trenova/internal/core/services/storageservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ReportHandlerParams struct {
	fx.In

	Logger         *zap.Logger
	Service        *reportservice.Service
	StorageService *storageservice.Service
	PM             *middleware.PermissionMiddleware
	ErrorHandler   *helpers.ErrorHandler
}

type ReportHandler struct {
	logger         *zap.Logger
	service        *reportservice.Service
	storageService *storageservice.Service
	pm             *middleware.PermissionMiddleware
	errorHandler   *helpers.ErrorHandler
}

func NewReportHandler(p ReportHandlerParams) *ReportHandler {
	return &ReportHandler{
		logger:         p.Logger.Named("handler.report"),
		service:        p.Service,
		storageService: p.StorageService,
		pm:             p.PM,
		errorHandler:   p.ErrorHandler,
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

func (h *ReportHandler) generate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req reportservice.GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	format, err := report.FormatFromString(req.Format.String())
	if err != nil {
		h.errorHandler.HandleError(c, fmt.Errorf("invalid format: %w", err))
		return
	}

	deliveryMethod, err := report.DeliveryMethodFromString(req.DeliveryMethod.String())
	if err != nil {
		h.errorHandler.HandleError(c, fmt.Errorf("invalid delivery method: %w", err))
		return
	}

	rpt, err := h.service.GenerateReport(c.Request.Context(), &reportservice.GenerateReportRequest{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		ResourceType:   req.ResourceType,
		Name:           req.Name,
		Format:         format,
		DeliveryMethod: deliveryMethod,
		FilterState:    req.FilterState,
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

	// Stream file from storage
	object, err := h.storageService.StreamFile(c.Request.Context(), bucketName, rpt.FilePath)
	if err != nil {
		h.logger.Error("failed to get object from storage",
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

	// Determine content type and file extension
	contentType := "text/csv"
	extension := ".csv"
	if rpt.Format == report.FormatExcel {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		extension = ".xlsx"
	}

	// Get base filename from the stored file path or use report name
	baseFilename := rpt.Name
	if baseFilename == "" {
		baseFilename = filepath.Base(rpt.FilePath)
		// Remove extension if present
		baseFilename = baseFilename[:len(baseFilename)-len(filepath.Ext(baseFilename))]
	}

	// Ensure filename has the correct extension
	filename := baseFilename
	if filepath.Ext(filename) != extension {
		filename = baseFilename + extension
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
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
