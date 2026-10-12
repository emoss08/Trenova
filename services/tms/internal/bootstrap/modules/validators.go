package modules

import (
	"context"

	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidationEngineFactory interface {
	CreateEngine() *validationframework.Engine
	CreateEngineWithConfig(config *validationframework.EngineConfig) *validationframework.Engine
}

type defaultValidationEngineFactory struct{}

func (f *defaultValidationEngineFactory) CreateEngine() *validationframework.Engine {
	return validationframework.NewEngine(validationframework.DefaultEngineConfig())
}

func (f *defaultValidationEngineFactory) CreateEngineWithConfig(
	config *validationframework.EngineConfig,
) *validationframework.Engine {
	return validationframework.NewEngine(config)
}

func ProvideValidationEngineFactory() ValidationEngineFactory {
	return &defaultValidationEngineFactory{}
}

type TenantedValidatorProvider interface {
	CreateValidatorFactory(getDB func(context.Context) (*bun.DB, error)) any
}

type ValidationContextFactory interface {
	CreateContext(isCreate bool) *validationframework.ValidationContext
}

type defaultValidationContextFactory struct{}

func (f *defaultValidationContextFactory) CreateContext(
	isCreate bool,
) *validationframework.ValidationContext {
	return &validationframework.ValidationContext{
		IsCreate: isCreate,
	}
}

func ProvideValidationContextFactory() ValidationContextFactory {
	return &defaultValidationContextFactory{}
}

var ValidationFrameworkModule = fx.Module(
	"validation-framework",
	fx.Provide(
		ProvideValidationEngineFactory,
		ProvideValidationContextFactory,
	),
)

var ValidatorModule = fx.Module(
	"validators",
	fx.Options(
		ValidationFrameworkModule,
	),
)
