package agentdefinition

import "github.com/emoss08/trenova/internal/core/domain/permission"

type DataAccessCeiling string

const (
	DataAccessInternal   = DataAccessCeiling("Internal")
	DataAccessRestricted = DataAccessCeiling("Restricted")
)

func (c DataAccessCeiling) IsValid() bool {
	switch c {
	case DataAccessInternal, DataAccessRestricted:
		return true
	default:
		return false
	}
}

func (c DataAccessCeiling) String() string { return string(c) }

func (c DataAccessCeiling) Sensitivity() permission.FieldSensitivity {
	if c == DataAccessRestricted {
		return permission.SensitivityRestricted
	}

	return permission.SensitivityInternal
}

func AllDataAccessCeilings() []DataAccessCeiling {
	return []DataAccessCeiling{DataAccessInternal, DataAccessRestricted}
}

func (d *Definition) DataAccessSensitivity() permission.FieldSensitivity {
	if d == nil {
		return permission.SensitivityInternal
	}

	return d.DataAccessCeiling.Sensitivity()
}

func (t Template) StarterDataAccess() DataAccessCeiling {
	if t == TemplateCashApplication || t.StarterTrigger() == TriggerChat {
		return DataAccessRestricted
	}

	return DataAccessInternal
}
