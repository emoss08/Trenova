package listquery

import "github.com/emoss08/keel"

func OptionalInt64(col keel.Int64Col, value *int64) keel.Assignment {
	if value == nil {
		return col.ToNull()
	}

	return col.To(*value)
}
