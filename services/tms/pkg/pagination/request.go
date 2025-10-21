package pagination

// FilteredRequest is a generic request wrapper for services/repositories
// that combines pagination options with domain-specific filter options
type FilteredRequest[T any] struct {
	Filter  *QueryOptions `json:"filter"`
	Options T             `json:"options"`
}

func BuildRequest[T any](filter *QueryOptions, options T) *FilteredRequest[T] {
	return &FilteredRequest[T]{
		Filter:  filter,
		Options: options,
	}
}

func (r *FilteredRequest[T]) WithFilter(filter *QueryOptions) *FilteredRequest[T] {
	r.Filter = filter
	return r
}

func (r *FilteredRequest[T]) WithOptions(options T) *FilteredRequest[T] {
	r.Options = options
	return r
}
