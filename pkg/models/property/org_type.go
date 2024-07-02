package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type OrganizationType string

const (
	OrganizationTypeAsset     = OrganizationType("Asset")
	OrganizationTypeBrokerage = OrganizationType("Brokerage")
	OrganizationTypeBoth      = OrganizationType("Both")
)

func (o OrganizationType) String() string {
	return string(o)
}

func (OrganizationType) Values() []string {
	return []string{"Asset", "Brokerage", "Both"}
}

func (o OrganizationType) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *OrganizationType) Scan(value any) error {
	if value == nil {
		return errors.New("organizationtype: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = OrganizationType(v)
	case []byte:
		*o = OrganizationType(string(v))
	default:
		return fmt.Errorf("organizationtype: cannot can type %T into OrganizationType", value)
	}
	return nil
}
