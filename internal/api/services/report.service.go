package services

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/config"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/imroc/req/v3"
	"github.com/rs/zerolog"
)

type ColumnValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type FileFormat string

const (
	CSV  FileFormat = "csv"
	XLS  FileFormat = "xls"
	XLSX FileFormat = "xlsx"
	PDF  FileFormat = "pdf"
)

type DeliveryMethod string

const (
	Email DeliveryMethod = "email"
	S3    DeliveryMethod = "s3"
	Minio DeliveryMethod = "minio"
	Kafka DeliveryMethod = "kafka"
	Redis DeliveryMethod = "redis"
)

type GenerateReportRequest struct {
	TableName      string         `json:"tableName"`
	Columns        []string       `json:"columns"`
	FileFormat     FileFormat     `json:"fileFormat"`
	DeliveryMethod DeliveryMethod `json:"deliveryMethod"`
}

type GenerateReportResponse struct {
	ReportURL string `json:"report_url"`
}

type ReportService struct {
	Client *ent.Client
	Logger *zerolog.Logger
	Config *config.Server
	Server *api.Server
}

// NewReportService creates a new report service.
func NewReportService(s *api.Server) *ReportService {
	return &ReportService{
		Client: s.Client,
		Logger: s.Logger,
		Config: &s.Config,
		Server: s,
	}
}

// GetColumnsByTableName returns the column names for a given table name.
//
// This function is used to retrieve the column names for a given table name. It will exclude any columns
func (r *ReportService) GetColumnsByTableName(ctx context.Context, tableName string) ([]ColumnValue, int, error) {
	columns := make([]ColumnValue, 0)
	excludedTableNames := map[string]bool{
		"table_change_alerts":       true,
		"shipment_controls":         true,
		"billing_controls":          true,
		"sessions":                  true,
		"organizations":             true,
		"business_units":            true,
		"feasibility_tool_controls": true,
		"users":                     true,
		"general_ledger_accounts":   true,
		"user_favorites":            true,
		"us_states":                 true,
		"invoice_controls":          true,
		"email_controls":            true,
		"route_controls":            true,
		"accounting_controls":       true,
		"email_profiles":            true,
	}

	excludedColumns := map[string]bool{
		"id":               true,
		"business_unit_id": true,
		"organization_id":  true,
	}

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		columns, _, err = r.getColumnsNames(ctx, tx, tableName, excludedTableNames, excludedColumns)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	return columns, len(columns), nil
}

// getColumnsNames returns the column names for a given table name.
//
// This function is used to retrieve the column names for a given table name. It will exclude any columns
// that are in the excludedColumns map and any tables that are in the excludedTableNames map.
func (r *ReportService) getColumnsNames(
	ctx context.Context, tx *ent.Tx, tableName string,
	excludedTableNames map[string]bool, excludedColumns map[string]bool,
) ([]ColumnValue, int, error) {
	if excludedTableNames[tableName] {
		return nil, 0, nil // Return empty if the table is in the excluded list
	}

	query := `SELECT 
		column_name 
	FROM 
		information_schema.columns 
	WHERE 
		table_schema = 'public' 
	AND
		table_name = $1
  	ORDER BY 
		ordinal_position DESC;
	`

	rows, err := tx.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var columns []ColumnValue
	for rows.Next() {
		var columnName string
		if err = rows.Scan(&columnName); err != nil {
			return nil, 0, err
		}

		if excludedColumns[columnName] {
			continue // Skip the loop iteration if column is to be excluded
		}

		formattedLabel := strings.ReplaceAll(util.ToTitleFormat(columnName), "_", " ")

		columns = append(columns, ColumnValue{
			Label: formattedLabel,
			Value: columnName,
		})
	}

	return columns, len(columns), nil
}

func (r *ReportService) GenerateReport(
	ctx context.Context, payload GenerateReportRequest, userID, orgID, buID uuid.UUID,
) (GenerateReportResponse, error) {
	client := req.C().
		SetTimeout(10 * time.Second)

	var result GenerateReportResponse

	resp, err := client.R().
		SetBody(&payload).
		SetSuccessResult(&result).
		Post(r.Config.Integration.GenerateReportEndpoint)
	if err != nil {
		return GenerateReportResponse{}, err
	}

	if resp.IsErrorState() {
		return GenerateReportResponse{}, nil
	}

	if resp.IsSuccessState() {
		err = util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
			message := Message{
				Type:     "success",
				Content:  "Report job completed successfully. Check your inbox for the requested report.",
				ClientID: userID.String(),
			}

			err = r.addReportToUser(ctx, tx, userID, orgID, buID, result.ReportURL)
			NewWebsocketService(r.Server).NotifyClient(userID.String(), message)
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return GenerateReportResponse{}, err
		}

		return result, nil
	}

	return GenerateReportResponse{}, nil
}

func (r *ReportService) addReportToUser(
	ctx context.Context, tx *ent.Tx, userID, orgID, buID uuid.UUID, reportURL string,
) error {
	_, err := tx.UserReport.Create().
		SetOrganizationID(orgID).
		SetBusinessUnitID(buID).
		SetUserID(userID).
		SetReportURL(reportURL).
		Save(ctx)
	if err != nil {
		return err
	}

	return nil
}
