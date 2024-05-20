package services

import (
	"context"
	"fmt"
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

// FileFormat represents the format of the report file.
type FileFormat string

// DeliveryMethod represents the method of delivery for the report.
type DeliveryMethod string

// Constants for the file format of the report.
const (
	CSV  = FileFormat("csv")
	XLS  = FileFormat("xls")
	XLSX = FileFormat("xlsx")
	PDF  = FileFormat("pdf")
)

// Constants for the delivery method of the report.
const (
	Email = DeliveryMethod("email")
	Local = DeliveryMethod("local")
)

type RelationshipRequest struct {
	ForeignKey      string   `json:"foreignKey" validate:"omitempty"`
	ReferencedTable string   `json:"referencedTable" validate:"omitempty"`
	Columns         []string `json:"columns" validate:"omitempty"`
}

// GenerateReportRequest represents the payload for generating a report.
type GenerateReportRequest struct {
	TableName      string                `json:"tableName" validate:"required"`
	Columns        []string              `json:"columns" validate:"required"`
	Relationships  []RelationshipRequest `json:"relationships" validate:"omitempty"`
	FileFormat     FileFormat            `json:"fileFormat" validate:"required"`
	DeliveryMethod DeliveryMethod        `json:"deliveryMethod" validate:"required"`
	OrganizationID uuid.UUID             `json:"organizationId"`
	BusinessUnitID uuid.UUID             `json:"businessUnitId"`
}

// GenerateReportResponse represents the response for generating a report.
type GenerateReportResponse struct {
	ReportURL string `json:"report_url"`
}

// ReportService represents the report service.
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

type ColumnValue struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// Relationship represents a foreign key relationship between tables.
type Relationship struct {
	ForeignKey       string        `json:"foreignKey"`
	ReferencedTable  string        `json:"referencedTable"`
	ReferencedColumn string        `json:"referencedColumn"`
	Columns          []ColumnValue `json:"columns"`
}

// GetColumnsByTableName returns the column names and relationships for a given table name.
//
// This function is used to retrieve the column names and relationships for a given table name. It will exclude any columns
func (r *ReportService) GetColumnsByTableName(ctx context.Context, tableName string) ([]ColumnValue, []Relationship, int, error) {
	excludedTableNames := map[string]bool{
		"table_change_alerts":       true,
		"shipment_controls":         true,
		"billing_controls":          true,
		"sessions":                  true,
		"organizations":             true,
		"business_units":            true,
		"feasibility_tool_controls": true,
		"users":                     true,
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

	columns, relationships, _, err := r.getColumnsAndRelationships(ctx, tableName, excludedTableNames, excludedColumns)
	if err != nil {
		r.Logger.Err(err).Msg("Failed to get columns and relationships")
		return nil, nil, 0, err
	}

	return columns, relationships, len(columns), nil
}

// getColumnsAndRelationships returns the column names for a given table name and the relationships between the tables.
//
// This function is used to retrieve the column names for a given table name and identify relationships (foreign keys) between the tables.
// It will exclude any columns that are in the excludedColumns map and any tables that are in the excludedTableNames map.
func (r *ReportService) getColumnsAndRelationships(
	ctx context.Context, tableName string,
	excludedTableNames map[string]bool, excludedColumns map[string]bool,
) ([]ColumnValue, []Relationship, int, error) {
	if excludedTableNames[tableName] {
		return nil, nil, 0, fmt.Errorf("table %s is excluded", tableName)
	}

	columnsQuery := `SELECT
						c.column_name,
						COALESCE(pgd.description, 'No description available') AS description
					FROM
						information_schema.columns AS c
					LEFT JOIN pg_catalog.pg_statio_all_tables AS st
						ON c.table_schema = st.schemaname AND c.table_name = st.relname
					LEFT JOIN pg_catalog.pg_description AS pgd
						ON pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
					WHERE
						c.table_schema = 'public' AND c.table_name = $1
					ORDER BY
						c.ordinal_position ASC;`

	rows, err := r.Client.QueryContext(ctx, columnsQuery, tableName)
	if err != nil {
		return nil, nil, 0, err
	}
	defer rows.Close()

	var columns []ColumnValue
	for rows.Next() {
		var columnName, description string
		if err = rows.Scan(&columnName, &description); err != nil {
			return nil, nil, 0, err
		}

		if excludedColumns[columnName] {
			continue // Skip excluded columns
		}

		formattedLabel := strings.ReplaceAll(util.ToTitleFormat(strings.ReplaceAll(columnName, "_", " ")), "_", " ")

		columns = append(columns, ColumnValue{
			Label:       formattedLabel,
			Value:       columnName,
			Description: description,
		})
	}

	relationshipsQuery := `SELECT
								kcu.column_name AS foreign_key,
								ccu.table_name AS referenced_table,
								ccu.column_name AS referenced_column
							FROM
								information_schema.table_constraints AS tc
							JOIN
								information_schema.key_column_usage AS kcu
							ON
								tc.constraint_name = kcu.constraint_name
								AND tc.table_schema = kcu.table_schema
							JOIN
								information_schema.constraint_column_usage AS ccu
							ON
								ccu.constraint_name = tc.constraint_name
								AND ccu.table_schema = tc.table_schema
							WHERE
								tc.constraint_type = 'FOREIGN KEY' AND tc.table_name = $1`

	relRows, err := r.Client.QueryContext(ctx, relationshipsQuery, tableName)
	if err != nil {
		return nil, nil, 0, err
	}
	defer relRows.Close()

	var relationships []Relationship
	for relRows.Next() {
		var foreignKey, referencedTable, referencedColumn string
		if err = relRows.Scan(&foreignKey, &referencedTable, &referencedColumn); err != nil {
			return nil, nil, 0, err
		}

		// Exclude relationships to certain tables
		if excludedTableNames[referencedTable] {
			continue
		}

		// Get columns and descriptions for the referenced table
		refColumns, _, cErr := r.getColumnsNames(ctx, referencedTable, excludedTableNames, excludedColumns)
		if cErr != nil {
			r.Logger.Err(cErr).Msg("Failed to get columns for referenced table")
			return nil, nil, 0, cErr
		}

		relationships = append(relationships, Relationship{
			ForeignKey:       foreignKey,
			ReferencedTable:  referencedTable,
			ReferencedColumn: referencedColumn,
			Columns:          refColumns,
		})
	}

	return columns, relationships, len(columns), nil
}

// getColumnsNames returns the column names for a given table name.
//
// This function is used to retrieve the column names for a given table name. It will exclude any columns
// that are in the excludedColumns map and any tables that are in the excludedTableNames map.
func (r *ReportService) getColumnsNames(
	ctx context.Context, tableName string,
	excludedTableNames map[string]bool, excludedColumns map[string]bool,
) ([]ColumnValue, int, error) {
	if excludedTableNames[tableName] {
		return nil, 0, fmt.Errorf("table %s is excluded", tableName)
	}

	query := `SELECT
                c.column_name,
                COALESCE(pgd.description, 'No description available') AS description
            FROM
                information_schema.columns AS c
            LEFT JOIN pg_catalog.pg_statio_all_tables AS st
                ON c.table_schema = st.schemaname AND c.table_name = st.relname
            LEFT JOIN pg_catalog.pg_description AS pgd
                ON pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
            WHERE
                c.table_schema = 'public' AND c.table_name = $1
            ORDER BY
                c.ordinal_position ASC;`

	rows, err := r.Client.QueryContext(ctx, query, tableName)
	if err != nil {
		r.Logger.Err(err).Msg("Failed to query columns")
		return nil, 0, err
	}
	defer rows.Close()

	var columns []ColumnValue
	for rows.Next() {
		var columnName, description string
		if err = rows.Scan(&columnName, &description); err != nil {
			r.Logger.Err(err).Msg("Failed to scan columns")
			return nil, 0, err
		}

		if excludedColumns[columnName] {
			continue // Skip excluded columns
		}

		formattedLabel := strings.ReplaceAll(util.ToTitleFormat(strings.ReplaceAll(columnName, "_", " ")), "_", " ")

		columns = append(columns, ColumnValue{
			Label:       formattedLabel,
			Value:       columnName,
			Description: description,
		})
	}

	return columns, len(columns), nil
}

// GenerateReport creates a report based on the provided payload.
//
// This function is used to generate a report based on the provided payload. The report will be generated
// based on the table name, columns, file format, and delivery method provided in the payload. The report
// will be sent to the user via the delivery method specified in the payload.
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
			if err != nil {
				r.Logger.Err(err).Msg("Failed to add report to user")
				return err
			}

			// Send the Notify client message in goroutine
			go func() {
				// Notify the client that the report has been generated
				NewWebsocketService(r.Client, r.Logger).NotifyClient(userID.String(), message)
			}()

			return nil
		})
		if err != nil {
			return GenerateReportResponse{}, err
		}

		return result, nil
	}

	return GenerateReportResponse{}, nil
}

// addReportToUser adds the report URL to the user's report list.
//
// This function is used to add the report URL to the user's report list. The report URL is the URL where the
// report can be downloaded from. The report URL is stored in the user_report table in the database.
func (r *ReportService) addReportToUser(
	ctx context.Context, tx *ent.Tx, userID, orgID, buID uuid.UUID, reportURL string,
) error {
	return tx.UserReport.Create().
		SetOrganizationID(orgID).
		SetBusinessUnitID(buID).
		SetUserID(userID).
		SetReportURL(reportURL).
		Exec(ctx)
}
