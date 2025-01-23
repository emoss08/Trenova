package validators

import (
	"github.com/emoss08/trenova/internal/pkg/validator/commodityvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/compliancevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/equipmentmanufacturervalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/equipmenttypevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/fleetcodevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/hazardousmaterialvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/locationvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/servicetypevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmenttypevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/workervalidator"
	"go.uber.org/fx"
)

var Module = fx.Module("validators", fx.Provide(
	compliancevalidator.NewValidator,
	workervalidator.NewWorkerProfileValidator,
	workervalidator.NewWorkerPTOValidator,
	workervalidator.NewValidator,
	fleetcodevalidator.NewValidator,
	equipmenttypevalidator.NewValidator,
	equipmentmanufacturervalidator.NewValidator,
	shipmenttypevalidator.NewValidator,
	servicetypevalidator.NewValidator,
	hazardousmaterialvalidator.NewValidator,
	commodityvalidator.NewValidator,
	locationvalidator.NewLocationCategoryValidator,
	locationvalidator.NewValidator,
))
