package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type AssignmentStatus string

const (
	AssignmentStatusActive   = AssignmentStatus("Active")
	AssignmentStatusInactive = AssignmentStatus("Inactive")
)

func (s AssignmentStatus) String() string {
	return string(s)
}

func (AssignmentStatus) Values() []string {
	return []string{"Active", "Inactive"}
}

func (s AssignmentStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *AssignmentStatus) Scan(value any) error {
	if value == nil {
		return errors.New("AssignmentStatus: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*s = AssignmentStatus(v)
	case []byte:
		*s = AssignmentStatus(string(v))
	default:
		return fmt.Errorf("AssignmentStatus: cannot scan type %T into AssignmentStatus", value)
	}
	return nil
}
