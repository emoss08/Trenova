package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type DeliveryMethod string

const (
	DeliveryMethodInsert = DeliveryMethod("Email")
	DeliveryMethodUpdate = DeliveryMethod("Local")
	DeliveryMethodDelete = DeliveryMethod("Api")
	DeliveryMethodAll    = DeliveryMethod("Sms")
)

func (o DeliveryMethod) String() string {
	return string(o)
}

func (DeliveryMethod) Values() []string {
	return []string{"Email", "Local", "Api", "Sms"}
}

func (o DeliveryMethod) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *DeliveryMethod) Scan(value any) error {
	if value == nil {
		return errors.New("deliverymethod: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = DeliveryMethod(v)
	case []byte:
		*o = DeliveryMethod(string(v))
	default:
		return fmt.Errorf("deliverymethod: cannot can type %T into DeliveryMethod", value)
	}
	return nil
}
