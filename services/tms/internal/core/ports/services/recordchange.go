package services

// RecordChange is what a planned write would do to one record: the record as
// it stands and as the write would leave it. A service's Plan method returns
// it without writing, so a tool previews and validates with the code that
// decides the write.
type RecordChange[T any] struct {
	Before *T
	After  *T
}
