package keelassign

import "github.com/emoss08/keel"

func StringOrNull(col keel.StringCol, value string) keel.Assignment {
	if value == "" {
		return col.ToNull()
	}
	return col.To(value)
}
