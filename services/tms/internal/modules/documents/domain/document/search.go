package document

import "slices"

func UnsearchableStatuses() []Status {
	return []Status{StatusRejected}
}

func (d *Document) Searchable() bool {
	return d.IsCurrentVersion && !slices.Contains(UnsearchableStatuses(), d.Status)
}
