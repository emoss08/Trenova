package fieldgate

type List[T any] struct {
	Items    []T      `json:"items"`
	Count    int      `json:"count"`
	Note     string   `json:"note,omitempty"`
	Withheld []string `json:"withheldByAccess,omitempty"`
}

func NewList[T any](items []T, gate *Gate) *List[T] {
	list := &List[T]{Items: items, Count: len(items)}
	if gate != nil {
		list.Withheld = gate.Withheld()
	}

	return list
}
