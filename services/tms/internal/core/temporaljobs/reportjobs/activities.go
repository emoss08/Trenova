package reportjobs

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/minio/minio-go/v7"
	"github.com/uptrace/bun"
	"github.com/xuri/excelize/v2"
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
}

type Activities struct {
	logger              *zap.Logger
	db                  *postgres.Connection
	minioClient         *minio.Client
	temporalClient      client.Client
	notificationService services.NotificationService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		logger:              p.Logger.With(zap.String("worker", "report")),
		db:                  p.DB,
		minioClient:         p.MinIOClient,
		temporalClient:      p.TemporalClient,
		notificationService: p.NotificationService,
	}
}

func (a *Activities) UpdateReportStatusActivity(
	ctx context.Context,
	reportID pulid.ID,
	status string,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating report status", "reportID", reportID, "status", status)

	db, err := a.db.DB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	reportStatus, err := report.StatusFromString(status)
	if err != nil {
		return fmt.Errorf("invalid status: %w", err)
	}

	_, err = db.NewUpdate().
		Model((*report.Report)(nil)).
		Set("status = ?", reportStatus).
		Set("updated_at = ?", utils.NowUnix()).
		Where("id = ?", reportID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update report status: %w", err)
	}

	return nil
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

	resourceInfo, err := GetResourceInfo(payload.ResourceType)
	if err != nil {
		return nil, fmt.Errorf("unsupported resource type: %w", err)
	}

	db, err := a.db.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	query := db.NewSelect().
		Table(resourceInfo.TableName).
		TableExpr(fmt.Sprintf("%s AS %s", resourceInfo.TableName, resourceInfo.Alias)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(fmt.Sprintf("%s.organization_id = ?", resourceInfo.Alias), payload.OrganizationID).
				Where(fmt.Sprintf("%s.business_unit_id = ?", resourceInfo.Alias), payload.BusinessUnitID)
		})

	fieldConfig := querybuilder.NewFieldConfigBuilder(resourceInfo.Entity).
		WithAutoMapping().
		WithAllFieldsFilterable().
		WithAllFieldsSortable().
		WithAutoEnumDetection().
		WithRelationshipFields().
		Build()

	qb := querybuilder.NewWithPostgresSearch(query, resourceInfo.Alias, fieldConfig, resourceInfo.Entity)
	qb.WithTraversalSupport(true)

	if len(payload.FilterState.FieldFilters) > 0 {
		qb.ApplyFilters(payload.FilterState.FieldFilters)
	}

	if len(payload.FilterState.Sort) > 0 {
		qb.ApplySort(payload.FilterState.Sort)
	}

	if payload.FilterState.Query != "" {
		searchConfig := resourceInfo.Entity.GetPostgresSearchConfig()
		if len(searchConfig.SearchableFields) > 0 {
			searchFields := make([]string, len(searchConfig.SearchableFields))
			for i, field := range searchConfig.SearchableFields {
				searchFields[i] = field.Name
			}
			qb.ApplyTextSearch(payload.FilterState.Query, searchFields)
		}
	}

	query = qb.GetQuery()

	activity.RecordHeartbeat(ctx, "executing query")

	var rows []map[string]any
	err = query.Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	activity.RecordHeartbeat(ctx, fmt.Sprintf("fetched %d rows", len(rows)))

	if len(rows) == 0 {
		return &temporaltype.QueryExecutionResult{
			Columns: []string{},
			Rows:    []map[string]any{},
			Total:   0,
		}, nil
	}

	columns := make([]string, 0, len(rows[0]))
	for col := range rows[0] {
		columns = append(columns, col)
	}

	return &temporaltype.QueryExecutionResult{
		Columns: columns,
		Rows:    rows,
		Total:   len(rows),
	}, nil
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
		fileData, err = a.generateCSV(queryResult)
	case report.FormatExcel:
		fileName = baseName + ".xlsx"
		fileData, err = a.generateExcel(queryResult, payload.ResourceType)
	default:
		return nil, fmt.Errorf("unsupported format: %s", payload.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate file: %w", err)
	}

	activity.RecordHeartbeat(ctx, "file generated")

	filePath := fmt.Sprintf(
		"reports/%s/%s/%s",
		payload.OrganizationID,
		payload.BusinessUnitID,
		fileName,
	)

	return &temporaltype.ReportResult{
		ReportID:  payload.ReportID,
		FilePath:  filePath,
		FileSize:  int64(len(fileData)),
		RowCount:  queryResult.Total,
		Status:    "COMPLETED",
		Timestamp: utils.NowUnix(),
	}, nil
}

func (a *Activities) generateCSV(result *temporaltype.QueryExecutionResult) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if err := writer.Write(result.Columns); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, row := range result.Rows {
		record := make([]string, len(result.Columns))
		for i, col := range result.Columns {
			if val, ok := row[col]; ok && val != nil {
				record[i] = fmt.Sprintf("%v", val)
			}
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

func (a *Activities) generateExcel(
	result *temporaltype.QueryExecutionResult,
	resourceType string,
) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := resourceType
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet: %w", err)
	}

	f.SetActiveSheet(index)

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create header style: %w", err)
	}

	for i, col := range result.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err = f.SetCellValue(sheetName, cell, col); err != nil {
			return nil, fmt.Errorf("failed to set header cell: %w", err)
		}
		if err = f.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return nil, fmt.Errorf("failed to set header style: %w", err)
		}
	}

	for rowIdx, row := range result.Rows {
		for colIdx, col := range result.Columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if val, ok := row[col]; ok && val != nil {
				if err = f.SetCellValue(sheetName, cell, val); err != nil {
					return nil, fmt.Errorf("failed to set cell value: %w", err)
				}
			}
		}
	}

	for i := range result.Columns {
		col, _ := excelize.ColumnNumberToName(i + 1)
		if err = f.SetColWidth(sheetName, col, col, 15); err != nil {
			return nil, fmt.Errorf("failed to set column width: %w", err)
		}
	}

	if err = f.DeleteSheet("Sheet1"); err != nil {
		return nil, fmt.Errorf("failed to delete default sheet: %w", err)
	}

	var buf bytes.Buffer
	if err = f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write Excel file: %w", err)
	}

	return buf.Bytes(), nil
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

	db, err := a.db.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	activity.RecordHeartbeat(ctx, "uploading to storage")

	var rpt report.Report
	err = db.NewSelect().
		Model(&rpt).
		Where("id = ?", result.ReportID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	resourceInfo, err := GetResourceInfo(rpt.ResourceType)
	if err != nil {
		return nil, fmt.Errorf("unsupported resource type: %w", err)
	}

	query := db.NewSelect().
		Table(resourceInfo.TableName).
		TableExpr(fmt.Sprintf("%s AS %s", resourceInfo.TableName, resourceInfo.Alias)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(
					fmt.Sprintf("%s.organization_id = ?", resourceInfo.Alias),
					rpt.OrganizationID,
				).
				Where(
					fmt.Sprintf("%s.business_unit_id = ?", resourceInfo.Alias),
					rpt.BusinessUnitID,
				)
		})

	fieldConfig := querybuilder.NewFieldConfigBuilder(resourceInfo.Entity).
		WithAutoMapping().
		WithAllFieldsFilterable().
		WithAllFieldsSortable().
		WithAutoEnumDetection().
		WithRelationshipFields().
		Build()

	qb := querybuilder.NewWithPostgresSearch(query, resourceInfo.Alias, fieldConfig, resourceInfo.Entity)
	qb.WithTraversalSupport(true)

	if len(rpt.FilterState.FieldFilters) > 0 {
		qb.ApplyFilters(rpt.FilterState.FieldFilters)
	}
	if len(rpt.FilterState.Sort) > 0 {
		qb.ApplySort(rpt.FilterState.Sort)
	}
	if rpt.FilterState.Query != "" {
		searchConfig := resourceInfo.Entity.GetPostgresSearchConfig()
		if len(searchConfig.SearchableFields) > 0 {
			searchFields := make([]string, len(searchConfig.SearchableFields))
			for i, field := range searchConfig.SearchableFields {
				searchFields[i] = field.Name
			}
			qb.ApplyTextSearch(rpt.FilterState.Query, searchFields)
		}
	}

	var rows []map[string]any
	err = qb.GetQuery().Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to re-execute query: %w", err)
	}

	queryResult := &temporaltype.QueryExecutionResult{
		Rows:  rows,
		Total: len(rows),
	}
	if len(rows) > 0 {
		columns := make([]string, 0, len(rows[0]))
		for col := range rows[0] {
			columns = append(columns, col)
		}
		queryResult.Columns = columns
	}

	var fileData []byte
	if rpt.Format == report.FormatCSV {
		fileData, err = a.generateCSV(queryResult)
	} else {
		fileData, err = a.generateExcel(queryResult, rpt.ResourceType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to generate file for upload: %w", err)
	}

	// Update file size and row count to match the actual generated file
	result.FileSize = int64(len(fileData))
	result.RowCount = queryResult.Total

	reader := bytes.NewReader(fileData)
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

	db, err := a.db.DB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	now := utils.NowUnix()
	expiresAt := now + (7 * 24 * 60 * 60)

	_, err = db.NewUpdate().
		Model((*report.Report)(nil)).
		Set("status = ?", report.StatusCompleted).
		Set("file_path = ?", result.FilePath).
		Set("file_size = ?", result.FileSize).
		Set("row_count = ?", result.RowCount).
		Set("completed_at = ?", now).
		Set("expires_at = ?", expiresAt).
		Set("updated_at = ?", now).
		Where("id = ?", result.ReportID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update report: %w", err)
	}

	return nil
}

func (a *Activities) MarkReportFailedActivity(
	ctx context.Context,
	reportID pulid.ID,
	errorMsg string,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Marking report as failed", "reportID", reportID, "error", errorMsg)

	db, err := a.db.DB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	_, err = db.NewUpdate().
		Model((*report.Report)(nil)).
		Set("status = ?", report.StatusFailed).
		Set("error_message = ?", errorMsg).
		Set("completed_at = ?", utils.NowUnix()).
		Set("updated_at = ?", utils.NowUnix()).
		Where("id = ?", reportID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to mark report as failed: %w", err)
	}

	return nil
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
			float64(result.FileSize)/(1024*1024),
			downloadURL,
		),
		TextBody: fmt.Sprintf(
			"Your %s export is ready. Rows: %d, Size: %.2f MB. Download: %s",
			payload.ResourceType,
			result.RowCount,
			float64(result.FileSize)/(1024*1024),
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

	notificationReq := &services.JobCompletionNotificationRequest{
		JobID:          result.ReportID.String(),
		JobType:        "report_export",
		UserID:         payload.UserID,
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		Success:        true,
		Result: fmt.Sprintf(
			"Your %s export (%s) is ready for download with %d rows.",
			payload.ResourceType,
			formatDisplay,
			result.RowCount,
		),
		Data: map[string]any{
			"reportId":     result.ReportID.String(),
			"resourceType": payload.ResourceType,
			"format":       formatDisplay,
			"rowCount":     result.RowCount,
			"fileSize":     result.FileSize,
		},
		RelatedEntities: []notification.RelatedEntity{
			{
				EntityType: "report",
				EntityID:   result.ReportID.String(),
			},
		},
		Actions: []notification.Action{
			{
				Label: "Download Report",
				Type:  "link",
				URL:   downloadURL,
			},
		},
	}

	activity.RecordHeartbeat(ctx, "sending notification")

	err := a.notificationService.SendJobCompletionNotification(ctx, notificationReq)
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
