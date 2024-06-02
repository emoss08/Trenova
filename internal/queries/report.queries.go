package queries

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"
)

type ReportQueryService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

func NewReportQueryService(c *ent.Client, l *zerolog.Logger) *ReportQueryService {
	return &ReportQueryService{
		Client: c,
		Logger: l,
	}
}

func (r *ReportQueryService) GetTableColumnsNames(ctx context.Context, tableName string) ([]string, error) {
	query := "SELECT column_name FROM information_schema.columns WHERE table_name = $1"

	row, err := r.Client.QueryContext(ctx, query, tableName)
	if err != nil {
		r.Logger.Err(err).Msg("Error getting table columns names")
		return nil, err
	}
	defer row.Close()

	var columns []string
	for row.Next() {
		var column string
		if err = row.Scan(&column); err != nil {
			r.Logger.Err(err).Msg("Error scanning row")
			return nil, err
		}
		columns = append(columns, column)
	}

	return columns, nil
}

// getColumnsAndRelationships returns the column names for a given table name and the relationships between the tables.
//
// This function is used to retrieve the column names for a given table name and identify relationships (foreign keys) between the tables.
// It will exclude any columns that are in the excludedColumns map and any tables that are in the excludedTableNames map.
func (r *ReportQueryService) GetColumnsAndRelationships(
	ctx context.Context, tableName string, excludedTableNames map[string]bool, excludedColumns map[string]bool,
) ([]types.ColumnValue, []types.Relationship, int, error) {
	if excludedTableNames[tableName] {
		r.Logger.Warn().Msgf("Table %s is excluded", tableName)
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
		r.Logger.Err(err).Msg("Failed to query columns")
		return nil, nil, 0, err
	}
	defer rows.Close()

	var columns []types.ColumnValue
	for rows.Next() {
		var columnName, description string
		if err = rows.Scan(&columnName, &description); err != nil {
			r.Logger.Err(err).Msg("Failed to scan columns")
			return nil, nil, 0, err
		}

		if excludedColumns[columnName] {
			continue // Skip excluded columns
		}

		formattedLabel := strings.ReplaceAll(util.ToTitleFormat(strings.ReplaceAll(columnName, "_", " ")), "_", " ")

		columns = append(columns, types.ColumnValue{
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
		r.Logger.Err(err).Msg("Failed to query relationships")
		return nil, nil, 0, err
	}
	defer relRows.Close()

	var relationships []types.Relationship
	for relRows.Next() {
		var foreignKey, referencedTable, referencedColumn string
		if err = relRows.Scan(&foreignKey, &referencedTable, &referencedColumn); err != nil {
			r.Logger.Err(err).Msg("Failed to scan relationships")
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

		relationships = append(relationships, types.Relationship{
			ForeignKey:       foreignKey,
			ReferencedTable:  referencedTable,
			ReferencedColumn: referencedColumn,
			Columns:          refColumns,
		})
	}

	r.Logger.Info().Msgf("Found %d columns and %d relationships for table %s", len(columns), len(relationships), tableName)

	return columns, relationships, len(columns), nil
}

// getColumnsNames returns the column names for a given table name.
//
// This function is used to retrieve the column names for a given table name. It will exclude any columns
// that are in the excludedColumns map and any tables that are in the excludedTableNames map.
func (r *ReportQueryService) getColumnsNames(
	ctx context.Context, tableName string, excludedTableNames map[string]bool, excludedColumns map[string]bool,
) ([]types.ColumnValue, int, error) {
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

	var columns []types.ColumnValue
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

		columns = append(columns, types.ColumnValue{
			Label:       formattedLabel,
			Value:       columnName,
			Description: description,
		})
	}

	return columns, len(columns), nil
}
