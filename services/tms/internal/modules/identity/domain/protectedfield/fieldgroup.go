package protectedfield

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type Group string

const (
	Personal Group = "Personal"
	Margins  Group = "Margins"
)

var fieldGroups = [...]Group{Personal, Margins}

func Groups() []Group {
	return fieldGroups[:]
}

func (g Group) String() string {
	return string(g)
}

func (g Group) IsValid() bool {
	return g.bit() != 0
}

func (g Group) Resource() permission.Resource {
	switch g {
	case Personal:
		return permission.ResourceProtectedPersonal
	case Margins:
		return permission.ResourceProtectedMargins
	default:
		return ""
	}
}

func (g Group) bit() GroupSet {
	switch g {
	case Personal:
		return 1
	case Margins:
		return 1 << 1
	default:
		return 0
	}
}

type GroupSet uint8

func AllGroups() GroupSet {
	var set GroupSet
	for _, group := range fieldGroups {
		set = set.With(group)
	}
	return set
}

func (s GroupSet) With(group Group) GroupSet {
	return s | group.bit()
}

func (s GroupSet) Has(group Group) bool {
	bit := group.bit()
	return bit != 0 && s&bit == bit
}

func (s GroupSet) HasAll() bool {
	return s == AllGroups()
}

type Withholder interface {
	WithholdFields(visible GroupSet)
}
