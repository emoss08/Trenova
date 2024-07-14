package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type ShipmentStatus string

const (
	ShipmentStatusNew        = ShipmentStatus("New")
	ShipmentStatusInProgress = ShipmentStatus("InProgress")
	ShipmentStatusCompleted  = ShipmentStatus("Completed")
	ShipmentStatusBilled     = ShipmentStatus("Billed")
	ShipmentStatusVoided     = ShipmentStatus("Voided")
	ShipmentStatusHold       = ShipmentStatus("Hold")
)

func (o ShipmentStatus) String() string {
	return string(o)
}

func (ShipmentStatus) Values() []string {
	return []string{"New", "InProgress", "Completed", "Hold", "Billed", "Voided"}
}

func (o ShipmentStatus) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *ShipmentStatus) Scan(value any) error {
	if value == nil {
		return errors.New("ShipmentStatus: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = ShipmentStatus(v)
	case []byte:
		*o = ShipmentStatus(string(v))
	default:
		return fmt.Errorf("ShipmentStatusType: cannot can type %T into ShipmentStatusType", value)
	}
	return nil
}
