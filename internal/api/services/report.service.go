// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package services

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/imroc/req/v3"
	"github.com/uptrace/bun"
)

type ReportService struct {
	db              *bun.DB
	logger          *config.ServerLogger
	websocketServer *WebsocketService
}

func NewReportService(s *server.Server) *ReportService {
	return &ReportService{
		db:              s.DB,
		logger:          s.Logger,
		websocketServer: NewWebsocketService(s),
	}
}

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

type Relationship struct {
	ForeignKey      string   `json:"foreignKey"`
	ReferencedTable string   `json:"referencedTable"`
	Columns         []string `json:"columns"`
}

// GenerateReportRequest represents the payload for generating a report.
type GenerateReportRequest struct {
	TableName      string         `json:"tableName"`
	Columns        []string       `json:"columns"`
	Relationships  []Relationship `json:"relationships"`
	FileFormat     FileFormat     `json:"fileFormat"`
	DeliveryMethod DeliveryMethod `json:"deliveryMethod"`
	OrganizationID uuid.UUID      `json:"organizationId"`
	BusinessUnitID uuid.UUID      `json:"businessUnitId"`
	UserID         uuid.UUID      `json:"userId"`
}

func (gr GenerateReportRequest) Validate() error {
	return validation.ValidateStruct(
		&gr,
		validation.Field(&gr.BusinessUnitID, validation.Required),
		validation.Field(&gr.OrganizationID, validation.Required),
		validation.Field(&gr.UserID, validation.Required),
		validation.Field(&gr.FileFormat, validation.In("csv", "xls", "xlsx", "pdf")),
		validation.Field(&gr.DeliveryMethod, validation.In("email", "local")),
	)
}

// GenerateReportResponse represents the response for generating a report.
type GenerateReportResponse struct {
	TaskID string `json:"task_id"`
}

type ColumnValue struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// TableRelationship represents a relationship between tables, including column information
// Update the TableRelationship struct to include the new fields
type TableRelationship struct {
	TableName         string        `json:"tableName"`
	ForeignKey        string        `json:"foreignKey"`
	ReferencedTable   string        `json:"referencedTable"`
	ReferencedColumn  string        `json:"referencedColumn"`
	RelationshipType  string        `json:"relationshipType"`
	TableColumns      []ColumnValue `json:"tableColumns"`
	ReferencedColumns []ColumnValue `json:"referencedColumns"`
	Columns           []ColumnValue `json:"columns"`
}

// GetTableRelationships returns the relationships and columns for the specified table and related tables
func (s ReportService) GetTableRelationships(ctx context.Context, tableName string) ([]TableRelationship, error) {
	query := `
		WITH table_relationships AS (
			SELECT
				tc.table_name AS table_name,
				kcu.column_name AS foreign_key,
				ccu.table_name AS referenced_table,
				ccu.column_name AS referenced_column,
				'references' AS relationship_type
			FROM
				information_schema.table_constraints AS tc
			JOIN
				information_schema.key_column_usage AS kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN
				information_schema.constraint_column_usage AS ccu
				ON ccu.constraint_name = tc.constraint_name
				AND ccu.table_schema = tc.table_schema
			WHERE
				tc.constraint_type = 'FOREIGN KEY'
				AND tc.table_name = ?
			UNION ALL
			SELECT
				ccu.table_name AS table_name,
				kcu.column_name AS foreign_key,
				tc.table_name AS referenced_table,
				ccu.column_name AS referenced_column,
				'referenced by' AS relationship_type
			FROM
				information_schema.table_constraints AS tc
			JOIN
				information_schema.key_column_usage AS kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN
				information_schema.constraint_column_usage AS ccu
				ON ccu.constraint_name = tc.constraint_name
				AND ccu.table_schema = tc.table_schema
			WHERE
				tc.constraint_type = 'FOREIGN KEY'
				AND ccu.table_name = ?
		),
		columns_info AS (
			SELECT
				c.table_name,
				c.column_name,
				COALESCE(pgd.description, 'No description available') AS description
			FROM
				information_schema.columns AS c
			LEFT JOIN pg_catalog.pg_statio_all_tables AS st
				ON c.table_schema = st.schemaname AND c.table_name = st.relname
			LEFT JOIN pg_catalog.pg_description AS pgd
				ON pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
			WHERE
				c.table_schema = 'public'
				AND (c.table_name = ? OR c.table_name IN (SELECT DISTINCT referenced_table FROM table_relationships))
		)
		SELECT
			r.table_name,
			r.foreign_key,
			r.referenced_table,
			r.referenced_column,
			r.relationship_type,
			json_agg(DISTINCT jsonb_build_object(
				'label', INITCAP(REPLACE(c1.column_name, '_', ' ')),
				'value', c1.column_name,
				'description', c1.description
			)) FILTER (WHERE c1.table_name = r.table_name) AS table_columns,
			json_agg(DISTINCT jsonb_build_object(
				'label', INITCAP(REPLACE(c2.column_name, '_', ' ')),
				'value', c2.column_name,
				'description', c2.description
			)) FILTER (WHERE c2.table_name = r.referenced_table) AS referenced_columns
		FROM
			table_relationships r
		LEFT JOIN
			columns_info c1 ON r.table_name = c1.table_name
		LEFT JOIN
			columns_info c2 ON r.referenced_table = c2.table_name
		GROUP BY
			r.table_name, r.foreign_key, r.referenced_table, r.referenced_column, r.relationship_type
		ORDER BY
			r.relationship_type, r.referenced_table, r.foreign_key;
    `

	var relationships []TableRelationship
	if err := s.db.NewRaw(query, tableName, tableName, tableName).Scan(ctx, &relationships); err != nil {
		s.logger.Err(err).Msg("Failed to query table relationships")
		return nil, err
	}

	// Process the results to combine table_columns and referenced_columns
	for i, rel := range relationships {
		if rel.TableName == tableName {
			relationships[i].Columns = rel.TableColumns
		} else {
			relationships[i].Columns = rel.ReferencedColumns
		}
	}

	return relationships, nil
}

func (s ReportService) GetColumnsByTableName(ctx context.Context, tableName string) ([]ColumnValue, []TableRelationship, int, error) {
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

	if excludedTableNames[tableName] {
		return nil, nil, 0, fmt.Errorf("table %s is excluded", tableName)
	}

	relationships, err := s.GetTableRelationships(ctx, tableName)
	if err != nil {
		return nil, nil, 0, err
	}

	var tableColumns []ColumnValue
	var tableRelationships []TableRelationship

	for _, rel := range relationships {
		if rel.TableName == tableName {
			// Filter out excluded columns
			filteredColumns := make([]ColumnValue, 0, len(rel.Columns))
			for _, col := range rel.Columns {
				if !excludedColumns[col.Value] {
					filteredColumns = append(filteredColumns, col)
				}
			}
			tableColumns = filteredColumns

			// Only add relationships for non-excluded tables
			if !excludedTableNames[rel.ReferencedTable] {
				tableRelationships = append(tableRelationships, rel)
			}
		}
	}

	return tableColumns, tableRelationships, len(tableColumns), nil
}

// GenerateReport generates a report based on the given payload.
//
// This function is used to generate a report based on the given payload. It will call the integration service to generate the report
// and then add the report to the user's account.
func (s ReportService) GenerateReport(ctx context.Context, payload GenerateReportRequest, userID, orgID, buID uuid.UUID) (GenerateReportResponse, error) {
	cfg, err := config.DefaultServiceConfigFromEnv(false)
	if err != nil {
		s.logger.Err(err).Msg("Failed to load server configuration")
		return GenerateReportResponse{}, err
	}

	client := req.C().SetTimeout(10 * time.Second)

	var result GenerateReportResponse

	// Convert uuid.UUID to string for JSON serialization
	payload.OrganizationID = orgID
	payload.BusinessUnitID = buID
	payload.UserID = userID

	resp, err := client.R().
		SetBody(payload).
		SetSuccessResult(&result).
		Post(cfg.Integration.GenerateReportEndpoint)
	if err != nil {
		s.logger.Err(err).Msg("Failed to generate report")
		return GenerateReportResponse{}, err
	}

	if resp.IsSuccessState() {
		if err = s.addReportToUser(ctx, userID, orgID, buID, result.TaskID); err != nil {
			s.logger.Err(err).Msg("Failed to add report to user")
			return GenerateReportResponse{}, err
		}

		return result, nil
	}

	return GenerateReportResponse{}, fmt.Errorf("failed to generate report: %s", resp.String())
}

// addReportToUser adds the report to the user's account.
//
// This function is used to add the report to the user's account.
func (s ReportService) addReportToUser(ctx context.Context, userID, orgID, buID uuid.UUID, reportURL string) error {
	report := &models.UserReport{
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ReportURL:      reportURL,
	}

	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().
			Model(report).
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
}
