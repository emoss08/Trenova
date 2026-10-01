package resolvertest

import (
	"fmt"

	"github.com/emoss08/trenova/shared/pulid"
)

func BenchmarkConnectionCursorItems(count int) ([]TestCursorEntity, [][]any) {
	items := make([]TestCursorEntity, 0, count)
	values := make([][]any, 0, count)
	for i := range count {
		id := pulid.MustNew("trac_")
		name := fmt.Sprintf("TRC-%05d", i)
		createdAt := int64(1710000000000 + i)
		items = append(items, TestCursorEntity{
			Node:      name,
			ID:        id,
			CreatedAt: createdAt,
		})
		values = append(values, []any{name, createdAt, id.String()})
	}

	return items, values
}
