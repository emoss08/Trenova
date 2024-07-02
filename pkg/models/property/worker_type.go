package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type WorkerType string

const (
	WorkerTypeEmployee   = WorkerType("Employee")
	WorkerTypeContractor = WorkerType("Contractor")
)

func (o WorkerType) String() string {
	return string(o)
}

func (WorkerType) Values() []string {
	return []string{"Employee", "Contractor"}
}

func (o WorkerType) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *WorkerType) Scan(value any) error {
	if value == nil {
		return errors.New("WorkerType: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = WorkerType(v)
	case []byte:
		*o = WorkerType(string(v))
	default:
		return fmt.Errorf("WorkerType: cannot can type %T into WorkerType", value)
	}
	return nil
}
