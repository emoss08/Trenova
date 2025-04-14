package mocks

import (
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
)

type MockValidationEngineFactory struct{}

func (f *MockValidationEngineFactory) CreateEngine() *framework.ValidationEngine {
	return framework.NewValidationEngine()
}
