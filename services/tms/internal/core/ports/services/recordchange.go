package services

type RecordChange[T any] struct {
	Before *T
	After  *T
}

func Befores[T any](changes []RecordChange[T]) []*T {
	befores := make([]*T, 0, len(changes))
	for idx := range changes {
		befores = append(befores, changes[idx].Before)
	}

	return befores
}
