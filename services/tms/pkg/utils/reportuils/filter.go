package reportuils

// ExcludedColumns are the columns that are excluded from the report global filter.
var ExcludedColumns = map[string]bool{
	"search_vector":    true,
	"business_unit_id": true,
	"organization_id":  true,
	"version":          true,
}

// FilterColumns filters the columns to exclude the excluded columns.
func FilterColumns(columns []string) []string {
	filtered := make([]string, 0, len(columns))
	for _, col := range columns {
		if !ExcludedColumns[col] {
			filtered = append(filtered, col)
		}
	}
	return filtered
}

// FilterRowData filters the rows to exclude the excluded columns.
func FilterRowData(rows []map[string]any, allowedColumns []string) []map[string]any {
	allowedSet := make(map[string]bool)
	for _, col := range allowedColumns {
		allowedSet[col] = true
	}

	filtered := make([]map[string]any, len(rows))
	for i, row := range rows {
		filteredRow := make(map[string]any)
		for col, val := range row {
			if allowedSet[col] {
				filteredRow[col] = val
			}
		}
		filtered[i] = filteredRow
	}
	return filtered
}

func FileSizeMB(fileSize int64) float64 {
	return float64(fileSize) / (1024 * 1024)
}
