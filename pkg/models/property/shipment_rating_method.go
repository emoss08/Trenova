package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type ShipmentRatingMethod string

const (
	ShipmentRatingMethodFlatRate         = ShipmentRatingMethod("FlatRate")
	ShipmentRatingMethodPerMile          = ShipmentRatingMethod("PerMile")
	ShipmentRatingMethodPerHundredWeight = ShipmentRatingMethod("PerHundredWeight")
	ShipmentRatingMethodPerStop          = ShipmentRatingMethod("PerStop")
	ShipmentRatingMethodPerPound         = ShipmentRatingMethod("PerPound")
	ShipmentRatingMethodOther            = ShipmentRatingMethod("Other")
)

func (o ShipmentRatingMethod) String() string {
	return string(o)
}

func (ShipmentRatingMethod) Values() []string {
	return []string{"New", "InProgress", "Completed", "Hold", "Billed", "Voided"}
}

func (o ShipmentRatingMethod) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *ShipmentRatingMethod) Scan(value any) error {
	if value == nil {
		return errors.New("ShipmentRatingMethod: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = ShipmentRatingMethod(v)
	case []byte:
		*o = ShipmentRatingMethod(string(v))
	default:
		return fmt.Errorf("ShipmentRatingMethodType: cannot can type %T into ShipmentRatingMethodType", value)
	}
	return nil
}
