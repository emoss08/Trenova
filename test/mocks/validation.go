/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package mocks

import (
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
)

type MockValidationEngineFactory struct{}

func (f *MockValidationEngineFactory) CreateEngine() *framework.ValidationEngine {
	return framework.NewValidationEngine()
}
