package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type StopType string

const (
	StopTypePickup      = StopType("Pickup")
	StopTypeSplitPickup = StopType("SplitPickup")
	StopTypeSplitDrop   = StopType("SplitDrop")
	StopTypeDelivery    = StopType("Delivery")
	StopTypeDropOff     = StopType("DropOff")
)

func (s StopType) String() string {
	return string(s)
}

func (StopType) Values() []string {
	return []string{"Pickup", "SplitPickup", "SplitDrop", "Delivery", "DropOff"}
}

func (s StopType) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *StopType) Scan(value any) error {
	if value == nil {
		return errors.New("StopType: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*s = StopType(v)
	case []byte:
		*s = StopType(string(v))
	default:
		return fmt.Errorf("StopType: cannot scan type %T into StopType", value)
	}
	return nil
}
