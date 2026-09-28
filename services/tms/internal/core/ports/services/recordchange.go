package services

type RecordChange[T any] struct {
	Before *T
	After  *T
}
