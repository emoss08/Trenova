package property

import "database/sql/driver"

type Status string

const (
	StatusActive   = Status("A")
	StatusInactive = Status("I")
)

func (s Status) String() string {
	return string(s)
}

func (Status) Values() []string {
	return []string{"A", "I"}
}

func (s Status) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *Status) Scan(value any) error {
	*s = Status(value.(string))
	return nil
}
