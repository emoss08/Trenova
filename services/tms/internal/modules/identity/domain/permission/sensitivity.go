package permission

type FieldSensitivity string

const (
	SensitivityPublic       FieldSensitivity = "public"
	SensitivityInternal     FieldSensitivity = "internal"
	SensitivityRestricted   FieldSensitivity = "restricted"
	SensitivityConfidential FieldSensitivity = "confidential"
)

func (s FieldSensitivity) Level() int {
	switch s {
	case SensitivityPublic:
		return 0
	case SensitivityInternal:
		return 1
	case SensitivityRestricted:
		return 2
	case SensitivityConfidential:
		return 3
	default:
		return 0
	}
}

func (s FieldSensitivity) CanAccess(target FieldSensitivity) bool {
	return s.Level() >= target.Level()
}

func (s FieldSensitivity) String() string {
	return string(s)
}

func (s FieldSensitivity) IsValid() bool {
	switch s {
	case SensitivityPublic, SensitivityInternal, SensitivityRestricted, SensitivityConfidential:
		return true
	default:
		return false
	}
}

type DataScope string

const (
	DataScopeOwn DataScope = "own"
	// DataScopeTeam narrows a grant to the people the holder manages — their
	// own reports, the terminals they run, and anybody delegated to them. It
	// sits above "own" because a manager can act on more than themselves, and
	// below "organization" because they cannot act on everybody.
	DataScopeTeam         DataScope = "team"
	DataScopeOrganization DataScope = "organization"
	DataScopeBusinessUnit DataScope = "business_unit"
	DataScopeAll          DataScope = "all"
)

func (s DataScope) Level() int {
	switch s {
	case DataScopeOwn:
		return 0
	case DataScopeTeam:
		return 1
	case DataScopeOrganization:
		return 2
	case DataScopeBusinessUnit:
		return 3
	case DataScopeAll:
		return 4
	default:
		return 0
	}
}

func (s DataScope) IsMorePermissive(other DataScope) bool {
	return s.Level() > other.Level()
}

func (s DataScope) String() string {
	return string(s)
}

func (s DataScope) IsValid() bool {
	switch s {
	case DataScopeOwn, DataScopeTeam, DataScopeOrganization, DataScopeBusinessUnit, DataScopeAll:
		return true
	default:
		return false
	}
}
