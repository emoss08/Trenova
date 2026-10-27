package listquery

import "github.com/emoss08/keel"

func MatchVector(col keel.TSVectorCol, query string) keel.Pred {
	return col.Match(query, searchOptions(query)...)
}
