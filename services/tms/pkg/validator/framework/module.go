package framework

import (
	"context"

	"github.com/emoss08/trenova/pkg/validator"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidationEngineFactory interface {
	CreateEngine() *ValidationEngine
	CreateEngineWithConfig(config *EngineConfig) *ValidationEngine
}

type defaultValidationEngineFactory struct{}

func (f *defaultValidationEngineFactory) CreateEngine() *ValidationEngine {
	return NewValidationEngine(DefaultEngineConfig())
}

func (f *defaultValidationEngineFactory) CreateEngineWithConfig(
	config *EngineConfig,
) *ValidationEngine {
	return NewValidationEngine(config)
}

func ProvideValidationEngineFactory() ValidationEngineFactory {
	return &defaultValidationEngineFactory{}
}

type ValidationBuilderFactory interface {
	CreateBuilder() *ValidationBuilder
	CreateBuilderWithConfig(config *EngineConfig) *ValidationBuilder
}

type defaultValidationBuilderFactory struct{}

func (f *defaultValidationBuilderFactory) CreateBuilder() *ValidationBuilder {
	return NewValidationBuilder()
}

func (f *defaultValidationBuilderFactory) CreateBuilderWithConfig(
	config *EngineConfig,
) *ValidationBuilder {
	return NewValidationBuilder().WithConfig(config)
}

func ProvideValidationBuilderFactory() ValidationBuilderFactory {
	return &defaultValidationBuilderFactory{}
}

type TenantedValidatorProvider interface {
	CreateValidatorFactory(getDB func(context.Context) (*bun.DB, error)) interface{}
}

type ValidationContextFactory interface {
	CreateContext(isCreate bool) *validator.ValidationContext
}

type defaultValidationContextFactory struct{}

func (f *defaultValidationContextFactory) CreateContext(
	isCreate bool,
) *validator.ValidationContext {
	return &validator.ValidationContext{
		IsCreate: isCreate,
	}
}

func ProvideValidationContextFactory() ValidationContextFactory {
	return &defaultValidationContextFactory{}
}

var Module = fx.Module("validation-framework",
	fx.Provide(
		ProvideValidationEngineFactory,
		ProvideValidationBuilderFactory,
		ProvideValidationContextFactory,
	),
)
