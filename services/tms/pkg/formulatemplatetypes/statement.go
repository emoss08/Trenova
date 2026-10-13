package formulatemplatetypes

type StatementKind string

const (
	StatementLet    = StatementKind("let")
	StatementResult = StatementKind("result")
)

type Statement struct {
	Index int           `json:"index"`
	Kind  StatementKind `json:"kind"`
	Name  string        `json:"name,omitempty"`
	Code  string        `json:"code"`
	Start int           `json:"start"`
	End   int           `json:"end"`
	Value any           `json:"value,omitempty"`
	Error string        `json:"error,omitempty"`
}
