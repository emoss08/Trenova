package worker_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// A position created with the driving flag off must be inserted as FALSE. A
// zero-valued bool on a column with a database default is exactly the case
// Bun swaps for DEFAULT, which is how an unticked box came back ticked.
func TestJobPosition_InsertWritesDrivingFlagOff(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())
	entity := &worker.JobPosition{
		Code:              "DISPATCH",
		Title:             "Dispatcher",
		Department:        worker.JobDepartment("Operations"),
		IsDrivingPosition: false,
	}

	sql := db.NewInsert().Model(entity).String()
	columns := sql[strings.Index(sql, "(")+1 : strings.Index(sql, ")")]
	values := sql[strings.LastIndex(sql, "(")+1 : strings.LastIndex(sql, ")")]

	names := strings.Split(columns, ", ")
	vals := strings.Split(values, ", ")
	index := -1
	for i, name := range names {
		if name == `"is_driving_position"` {
			index = i
		}
	}
	if assert.NotEqual(t, -1, index, "column missing from insert: %s", sql) &&
		assert.Len(t, vals, len(names), "value list does not line up: %s", sql) {
		assert.Equal(t, "FALSE", strings.ToUpper(vals[index]), "insert: %s", sql)
	}
}
