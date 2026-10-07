package setutils

// SameMembers reports whether two slices hold the same members, whatever
// their order or repeats: a list of grants is a set, so reordering it is no
// change.
func SameMembers[T comparable](a, b []T) bool {
	members := make(map[T]struct{}, len(a))
	for _, item := range a {
		members[item] = struct{}{}
	}

	seen := make(map[T]struct{}, len(b))
	for _, item := range b {
		if _, ok := members[item]; !ok {
			return false
		}
		seen[item] = struct{}{}
	}

	return len(seen) == len(members)
}
