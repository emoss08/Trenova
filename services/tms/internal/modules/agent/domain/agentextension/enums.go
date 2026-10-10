package agentextension

type Type string

const (
	TypeExa = Type("Exa")
)

func AllTypes() []Type {
	return []Type{TypeExa}
}

func (t Type) String() string { return string(t) }

func (t Type) IsValid() bool {
	switch t {
	case TypeExa:
		return true
	default:
		return false
	}
}

type Category string

const (
	CategoryWebResearch = Category("WebResearch")
)

func AllCategories() []Category {
	return []Category{CategoryWebResearch}
}

func (c Category) String() string { return string(c) }

func (c Category) IsValid() bool {
	switch c {
	case CategoryWebResearch:
		return true
	default:
		return false
	}
}

type Availability string

const (
	AvailabilityAllAgents      = Availability("AllAgents")
	AvailabilitySelectedAgents = Availability("SelectedAgents")
)

func AllAvailabilities() []Availability {
	return []Availability{AvailabilityAllAgents, AvailabilitySelectedAgents}
}

func (a Availability) String() string { return string(a) }

func (a Availability) IsValid() bool {
	switch a {
	case AvailabilityAllAgents, AvailabilitySelectedAgents:
		return true
	default:
		return false
	}
}
