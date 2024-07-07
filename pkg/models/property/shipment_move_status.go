package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type ShipmentMoveStatus string

const (
	ShipmentMoveStatusNew        = ShipmentMoveStatus("New")
	ShipmentMoveStatusInProgress = ShipmentMoveStatus("InProgress")
	ShipmentMoveStatusCompleted  = ShipmentMoveStatus("Completed")
	ShipmentMoveStatusVoided     = ShipmentMoveStatus("Voided")
)

func (o ShipmentMoveStatus) String() string {
	return string(o)
}

func (ShipmentMoveStatus) Values() []string {
	return []string{"New", "InProgress", "Completed", "Voided"}
}

func (o ShipmentMoveStatus) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *ShipmentMoveStatus) Scan(value any) error {
	if value == nil {
		return errors.New("ShipmentMoveStatus: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = ShipmentMoveStatus(v)
	case []byte:
		*o = ShipmentMoveStatus(string(v))
	default:
		return fmt.Errorf("ShipmentMoveStatusType: cannot can type %T into ShipmentMoveStatusType", value)
	}
	return nil
}
