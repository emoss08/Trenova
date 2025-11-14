package reportjobs

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/reportuils"
	"github.com/minio/minio-go/v7"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Logger              *zap.Logger
	DB                  *postgres.Connection
	MinIOClient         *minio.Client
	TemporalClient      client.Client
	NotificationService services.NotificationService
	ReportRepository    repositories.ReportRepository
}

type Activities struct {
	logger              *zap.Logger
	db                  *postgres.Connection
	minioClient         *minio.Client
	temporalClient      client.Client
	notificationService services.NotificationService
	reportRepo          repositories.ReportRepository
	fileGenerator       *FileGenerator
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		logger:              p.Logger.With(zap.String("worker", "report")),
		db:                  p.DB,
		minioClient:         p.MinIOClient,
		temporalClient:      p.TemporalClient,
		notificationService: p.NotificationService,
		reportRepo:          p.ReportRepository,
		fileGenerator:       NewFileGenerator(),
	}
}

func (a *Activities) UpdateReportStatusActivity(
	ctx context.Context,
	reportID pulid.ID,
	status string,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating report status", "reportID", reportID, "status", status)

	reportStatus, err := report.StatusFromString(status)
	if err != nil {
		return fmt.Errorf("invalid status: %w", err)
	}

	return a.reportRepo.UpdateStatus(ctx, repositories.UpdateStatusRequest{
		ReportID: reportID,
		Status:   reportStatus,
	})
}

func (a *Activities) ExecuteQueryActivity(
	ctx context.Context,
	payload *temporaltype.GenerateReportPayload,
) (*temporaltype.QueryExecutionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(
		"Executing query for report",
		"reportID",
		payload.ReportID,
		"resourceType",
		payload.ResourceType,
	)

	activity.RecordHeartbeat(ctx, "building query")

	qb, err := NewQueryBuilder(payload.ResourceType)
	if err != nil {
		return nil, err
	}

	db, err := a.db.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	baseQuery := qb.BuildBaseQuery(db, payload.OrganizationID, payload.BusinessUnitID)
	query := qb.ApplyFilters(baseQuery, payload.FilterState)

	activity.RecordHeartbeat(ctx, "executing query")

	result, err := qb.ExecuteAndFilter(ctx, query)
	if err != nil {
		return nil, err
	}

	activity.RecordHeartbeat(ctx, fmt.Sprintf("fetched %d rows", result.Total))

	return result, nil
}

func (a *Activities) GenerateFileActivity(
	ctx context.Context,
	payload *temporaltype.GenerateReportPayload,
	queryResult *temporaltype.QueryExecutionResult,
) (*temporaltype.ReportResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating file", "reportID", payload.ReportID, "format", payload.Format)

	activity.RecordHeartbeat(ctx, "generating file")

	var fileData []byte
	var fileName string
	var err error

	timestamp := time.Now().Format("20060102_150405")
	baseName := fmt.Sprintf("%s_export_%s", payload.ResourceType, timestamp)

	switch payload.Format {
	case report.FormatCSV:
		fileName = baseName + ".csv"
		fileData, err = a.fileGenerator.GenerateCSV(queryResult)
	case report.FormatExcel:
		fileName = baseName + ".xlsx"
		fileData, err = a.fileGenerator.GenerateExcel(queryResult, payload.ResourceType)
	default:
		return nil, fmt.Errorf("unsupported format: %s", payload.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate file: %w", err)
	}

	activity.RecordHeartbeat(ctx, "file generated")

	filePath := fmt.Sprintf(
		"reports/%s/%s/%s",
		payload.BusinessUnitID.String(),
		payload.OrganizationID.String(),
		fileName,
	)

	return &temporaltype.ReportResult{
		ReportID:       payload.ReportID,
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		UserID:         payload.UserID,
		FilePath:       filePath,
		FileSize:       int64(len(fileData)),
		RowCount:       queryResult.Total,
		Status:         report.StatusCompleted,
		Timestamp:      utils.NowUnix(),
	}, nil
}

func (a *Activities) UploadToStorageActivity(
	ctx context.Context,
	result *temporaltype.ReportResult,
) (*temporaltype.ReportResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(
		"Uploading file to storage",
		"reportID",
		result.ReportID,
		"filePath",
		result.FilePath,
	)

	activity.RecordHeartbeat(ctx, "uploading to storage")

	rpt, err := a.reportRepo.Get(ctx, repositories.GetReportByIDRequest{
		ReportID: result.ReportID,
		OrgID:    result.OrganizationID,
		BuID:     result.BusinessUnitID,
		UserID:   result.UserID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	qb, err := NewQueryBuilder(rpt.ResourceType)
	if err != nil {
		return nil, err
	}

	db, err := a.db.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	baseQuery := qb.BuildBaseQuery(db, rpt.OrganizationID, rpt.BusinessUnitID)
	query := qb.ApplyFilters(baseQuery, rpt.FilterState)

	queryResult, err := qb.ExecuteAndFilter(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to re-execute query: %w", err)
	}

	var fileData []byte
	if rpt.Format == report.FormatCSV {
		fileData, err = a.fileGenerator.GenerateCSV(queryResult)
	} else {
		fileData, err = a.fileGenerator.GenerateExcel(queryResult, rpt.ResourceType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to generate file for upload: %w", err)
	}

	// Update file size and row count to match the actual generated file
	result.FileSize = int64(len(fileData))
	result.RowCount = queryResult.Total

	contentType := "text/csv"
	if rpt.Format == report.FormatExcel {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}

	bucketName := "trenova-reports"
	exists, err := a.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		err = a.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	reader := bytes.NewReader(fileData)
	_, err = a.minioClient.PutObject(
		ctx,
		bucketName,
		result.FilePath,
		reader,
		result.FileSize,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to MinIO: %w", err)
	}

	activity.RecordHeartbeat(ctx, "upload complete")

	return result, nil
}

func (a *Activities) UpdateReportCompletedActivity(
	ctx context.Context,
	result *temporaltype.ReportResult,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating report as completed", "reportID", result.ReportID)

	now := utils.NowUnix()
	expiresAt := now + (7 * 24 * 60 * 60)

	return a.reportRepo.UpdateCompleted(ctx, repositories.UpdateCompletedRequest{
		ReportID:  result.ReportID,
		FilePath:  result.FilePath,
		FileSize:  result.FileSize,
		RowCount:  result.RowCount,
		ExpiresAt: expiresAt,
	})
}

func (a *Activities) MarkReportFailedActivity(
	ctx context.Context,
	reportID pulid.ID,
	errorMsg string,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Marking report as failed", "reportID", reportID, "error", errorMsg)

	return a.reportRepo.MarkFailed(ctx, repositories.MarkFailedRequest{
		ReportID:     reportID,
		ErrorMessage: errorMsg,
	})
}

func (a *Activities) SendReportEmailActivity(
	ctx context.Context,
	payload *temporaltype.GenerateReportPayload,
	result *temporaltype.ReportResult,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending report via email", "reportID", result.ReportID)

	activity.RecordHeartbeat(ctx, "preparing email")

	downloadURL := fmt.Sprintf("/api/reports/%s/download", result.ReportID)

	emailPayload := &temporaltype.SendEmailPayload{
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		UserID:         payload.UserID,
		ProfileID:      payload.EmailProfileID,
		To:             []string{payload.UserEmail},
		Subject:        fmt.Sprintf("Your %s Export is Ready", payload.ResourceType),
		HTMLBody: fmt.Sprintf(`
			<h2>Your Export is Ready</h2>
			<p>Your %s export has been generated successfully.</p>
			<p><strong>Details:</strong></p>
			<ul>
				<li>Resource: %s</li>
				<li>Format: %s</li>
				<li>Rows: %d</li>
				<li>File Size: %.2f MB</li>
			</ul>
			<p><a href="%s">Download Report</a></p>
			<p>This report will be available for 7 days.</p>
		`,
			payload.ResourceType,
			payload.ResourceType,
			payload.Format,
			result.RowCount,
			reportuils.FileSizeMB(result.FileSize),
			downloadURL,
		),
		TextBody: fmt.Sprintf(
			"Your %s export is ready. Rows: %d, Size: %.2f MB. Download: %s",
			payload.ResourceType,
			result.RowCount,
			reportuils.FileSizeMB(result.FileSize),
			downloadURL,
		),
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: temporaltype.EmailTaskQueue,
	}

	_, err := a.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"SendEmailWorkflow",
		emailPayload,
	)
	if err != nil {
		return fmt.Errorf("failed to start email workflow: %w", err)
	}

	activity.RecordHeartbeat(ctx, "email sent")

	return nil
}

func (a *Activities) SendReportReadyNotificationActivity(
	ctx context.Context,
	payload *temporaltype.GenerateReportPayload,
	result *temporaltype.ReportResult,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info(
		"Sending report ready notification",
		"reportID",
		result.ReportID,
		"userID",
		payload.UserID,
	)

	activity.RecordHeartbeat(ctx, "preparing notification")

	downloadURL := fmt.Sprintf("/api/v1/reports/%s/download/", result.ReportID)

	formatDisplay := "CSV"
	if payload.Format == report.FormatExcel {
		formatDisplay = "Excel"
	}

	activity.RecordHeartbeat(ctx, "sending notification")

	err := a.notificationService.SendReportExportNotification(
		ctx,
		&services.ReportExportNotificationRequest{
			UserID:         payload.UserID,
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
			ReportID:       result.ReportID,
			ReportName:     fmt.Sprintf("Report %s", result.ReportID.String()),
			ReportType:     payload.ResourceType,
			ReportFormat:   formatDisplay,
			ReportSize:     result.FileSize,
			ReportRowCount: result.RowCount,
			ReportURL:      downloadURL,
		},
	)
	if err != nil {
		logger.Error(
			"Failed to send report ready notification",
			"error",
			err,
			"reportID",
			result.ReportID,
			"userID",
			payload.UserID,
		)
		return fmt.Errorf("failed to send notification: %w", err)
	}

	activity.RecordHeartbeat(ctx, "notification sent")
	logger.Info("Report ready notification sent successfully", "reportID", result.ReportID)

	return nil
}
