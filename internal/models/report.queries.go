package models

import (
	"context"
)

func (r *QueryService) GetTableColumnsNames(ctx context.Context, tableName string) ([]string, error) {
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
