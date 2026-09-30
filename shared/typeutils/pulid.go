package typeutils

import "github.com/emoss08/trenova/shared/pulid"

// DerefID returns the id a pointer refers to, or pulid.Nil when the pointer is
// absent. A pointer to a nil id is treated the same as an absent pointer,
// because both mean "no id was supplied".
func DerefID(id *pulid.ID) pulid.ID {
	if id == nil || id.IsNil() {
		return pulid.Nil
	}

	return *id
}

// IDPtr returns a pointer to a copy of id, or nil when id is nil, so an
// optional id can be carried through a struct without aliasing the caller's
// value.
func IDPtr(id pulid.ID) *pulid.ID {
	if id.IsNil() {
		return nil
	}

	value := id

	return &value
}

// IDString renders an id for a string field, reporting nil when there is no
// id, so an unset reference is stored as NULL rather than an empty string.
func IDString(id pulid.ID) *string {
	if id.IsNil() {
		return nil
	}

	value := id.String()

	return &value
}

// IDStringOrEmpty renders an optional id for a field that spells "absent" as
// the empty string.
func IDStringOrEmpty(id *pulid.ID) string {
	if id == nil || id.IsNil() {
		return ""
	}

	return id.String()
}
