package keelexpr

import "github.com/emoss08/keel"

func OptionalInt64(col keel.Int64Col, value *int64) keel.Assignment {
	if value == nil {
		return col.ToNull()
	}

	return col.To(*value)
}

func TextContainsFold(e keel.Expr, s string) keel.Pred {
	return keel.CastText(e).ContainsFold(s)
}
