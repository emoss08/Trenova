package permission

type FieldGroup string

const (
	FieldGroupPersonal FieldGroup = "Personal"
	FieldGroupMargins  FieldGroup = "Margins"
)

var fieldGroups = [...]FieldGroup{FieldGroupPersonal, FieldGroupMargins}

func FieldGroups() []FieldGroup {
	return fieldGroups[:]
}

func (g FieldGroup) String() string {
	return string(g)
}

func (g FieldGroup) IsValid() bool {
	return g.bit() != 0
}

func (g FieldGroup) Resource() Resource {
	switch g {
	case FieldGroupPersonal:
		return ResourceProtectedPersonal
	case FieldGroupMargins:
		return ResourceProtectedMargins
	default:
		return ""
	}
}

func (g FieldGroup) bit() FieldGroupSet {
	switch g {
	case FieldGroupPersonal:
		return 1
	case FieldGroupMargins:
		return 1 << 1
	default:
		return 0
	}
}

type FieldGroupSet uint8

func AllFieldGroups() FieldGroupSet {
	var set FieldGroupSet
	for _, group := range fieldGroups {
		set = set.With(group)
	}
	return set
}

func (s FieldGroupSet) With(group FieldGroup) FieldGroupSet {
	return s | group.bit()
}

func (s FieldGroupSet) Has(group FieldGroup) bool {
	bit := group.bit()
	return bit != 0 && s&bit == bit
}

func (s FieldGroupSet) HasAll() bool {
	return s == AllFieldGroups()
}

type FieldWithholder interface {
	WithholdFields(visible FieldGroupSet)
}
