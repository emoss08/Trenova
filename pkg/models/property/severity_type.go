package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type Severity string

const (
	SeverityHigh   = Severity("High")
	SeverityMedium = Severity("Medium")
	SeverityLow    = Severity("Low")
)

func (o Severity) String() string {
	return string(o)
}

func (Severity) Values() []string {
	return []string{"High", "Medium", "Low"}
}

func (o Severity) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *Severity) Scan(value any) error {
	if value == nil {
		return errors.New("Severity: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = Severity(v)
	case []byte:
		*o = Severity(string(v))
	default:
		return fmt.Errorf("SeverityType: cannot can type %T into SeverityType", value)
	}
	return nil
}
