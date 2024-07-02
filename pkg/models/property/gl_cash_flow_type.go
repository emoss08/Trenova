package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type GLCashFlowType string

const (
	GLCashFlowTypeOperating = GLCashFlowType("Operating")
	GLCashFlowTypeInvesting = GLCashFlowType("Investing")
	GLCashFlowTypeFinancing = GLCashFlowType("Financing")
)

func (s GLCashFlowType) String() string {
	return string(s)
}

func (GLCashFlowType) Values() []string {
	return []string{"Operating", "Investing", "Financing"}
}

func (s GLCashFlowType) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *GLCashFlowType) Scan(value any) error {
	if value == nil {
		return errors.New("GLCashFlowType: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*s = GLCashFlowType(v)
	case []byte:
		*s = GLCashFlowType(string(v))
	default:
		return fmt.Errorf("GLCashFlowType: cannot scan type %T into GLCashFlowType", value)
	}
	return nil
}
