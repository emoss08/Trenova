package modules

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/accessorialchargeservice"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolservice"
	"github.com/emoss08/trenova/internal/core/services/accounttypeservice"
	"github.com/emoss08/trenova/internal/core/services/billingcontrolservice"
	"github.com/emoss08/trenova/internal/core/services/commodityservice"
	"github.com/emoss08/trenova/internal/core/services/customerservice"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/internal/core/services/dataentrycontrolservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchcontrolservice"
	"github.com/emoss08/trenova/internal/core/services/distanceoverrideservice"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/documenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/core/services/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/core/services/fiscalyearservice"
	"github.com/emoss08/trenova/internal/core/services/fleetcodeservice"
	"github.com/emoss08/trenova/internal/core/services/glaccountservice"
	"github.com/emoss08/trenova/internal/core/services/hazardousmaterialservice"
	"github.com/emoss08/trenova/internal/core/services/hazmatsegregationruleservice"
	"github.com/emoss08/trenova/internal/core/services/holdreasonservice"
	"github.com/emoss08/trenova/internal/core/services/locationcategoryservice"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/core/services/organizationservice"
	"github.com/emoss08/trenova/internal/core/services/roleservice"
	"github.com/emoss08/trenova/internal/core/services/sequenceconfigservice"
	"github.com/emoss08/trenova/internal/core/services/servicetypeservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentcontrolservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice"
	"github.com/emoss08/trenova/internal/core/services/shipmenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"github.com/emoss08/trenova/internal/core/services/userservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
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

var ValidationFrameworkModule = fx.Module("validation-framework",
	fx.Provide(
		ProvideValidationEngineFactory,
		ProvideValidationContextFactory,
	),
)

var ValidatorModule = fx.Module("validators",
	fx.Provide(
		organizationservice.NewValidator,
		equipmentmanufacturerservice.NewValidator,
		equipmenttypeservice.NewValidator,
		fleetcodeservice.NewValidator,
		tractorservice.NewValidator,
		trailerservice.NewValidator,
		userservice.NewValidator,
		workerservice.NewValidator,
		roleservice.NewValidator,
		customfieldservice.NewValidator,
		customfieldservice.NewValuesValidator,
		documentservice.NewValidator,
		accessorialchargeservice.NewValidator,
		servicetypeservice.NewValidator,
		sequenceconfigservice.NewValidator,
		shipmentcontrolservice.NewValidator,
		shipmentservice.NewValidator,
		shipmenttypeservice.NewValidator,
		hazardousmaterialservice.NewValidator,
		hazmatsegregationruleservice.NewValidator,
		commodityservice.NewValidator,
		customerservice.NewValidator,
		accountingcontrolservice.NewValidator,
		accounttypeservice.NewValidator,
		glaccountservice.NewValidator,
		fiscalyearservice.NewValidator,
		fiscalperiodservice.NewValidator,
		locationcategoryservice.NewValidator,
		locationservice.NewValidator,
		documenttypeservice.NewValidator,
		holdreasonservice.NewValidator,
		billingcontrolservice.NewValidator,
		dataentrycontrolservice.NewValidator,
		dispatchcontrolservice.NewValidator,
		distanceoverrideservice.NewValidator,
	),
	fx.Options(
		ValidationFrameworkModule,
	),
)
