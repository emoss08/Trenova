package artifactpayload

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
)

const (
	PayloadTool        = "tool"
	PayloadEntity      = "entity"
	PayloadColumns     = "columns"
	PayloadRows        = "rows"
	PayloadRowCount    = "rowCount"
	PayloadReadWith    = "readWith"
	PayloadSearchedFor = "searchedFor"
	PayloadTotals      = "totals"
	PayloadName        = "name"
)

func ColumnsFrom(value any) []assistantartifact.DisplayColumn {
	if columns, ok := value.([]assistantartifact.DisplayColumn); ok {
		return columns
	}

	var columns []assistantartifact.DisplayColumn
	if !DecodeDisplay(value, &columns) {
		return nil
	}

	return columns
}

func RowsFrom(value any) []any {
	rows, _ := value.([]any)

	return rows
}

func DecodeDisplay(value, into any) bool {
	if value == nil {
		return false
	}
	encoded, err := sonic.Marshal(value)
	if err != nil {
		return false
	}

	return sonic.Unmarshal(encoded, into) == nil
}
