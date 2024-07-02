package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
)

func (s Status) String() string {
	return string(s)
}

func (Status) Values() []string {
	return []string{"Active", "Inactive"}
}

func (s Status) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *Status) Scan(value any) error {
	if value == nil {
		return errors.New("status: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*s = Status(v)
	case []byte:
		*s = Status(string(v))
	default:
		return fmt.Errorf("status: cannot scan type %T into Status", value)
	}
	return nil
}
