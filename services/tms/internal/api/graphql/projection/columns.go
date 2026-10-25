package projection

import "github.com/emoss08/keel"

func ColumnNames(cols ...keel.Col) []string {
	names := make([]string, len(cols))
	for i, col := range cols {
		names[i] = col.Ref.Table.Columns[col.Index].Name
	}

	return names
}
