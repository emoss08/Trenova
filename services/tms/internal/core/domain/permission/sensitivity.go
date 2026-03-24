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
	DataScopeOwn          DataScope = "own"
	DataScopeOrganization DataScope = "organization"
	DataScopeAll          DataScope = "all"
)

func (s DataScope) Level() int {
	switch s {
	case DataScopeOwn:
		return 0
	case DataScopeOrganization:
		return 1
	case DataScopeAll:
		return 2
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
	case DataScopeOwn, DataScopeOrganization, DataScopeAll:
		return true
	default:
		return false
	}
}
