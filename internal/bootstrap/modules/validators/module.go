package validators

import (
	"github.com/trenova-app/transport/internal/pkg/validator/commodityvalidator"
	"github.com/trenova-app/transport/internal/pkg/validator/compliancevalidator"
	"github.com/trenova-app/transport/internal/pkg/validator/equipmentmanufacturervalidator"
	"github.com/trenova-app/transport/internal/pkg/validator/equipmenttypevalidator"
	"github.com/trenova-app/transport/internal/pkg/validator/fleetcodevalidator"
	"github.com/trenova-app/transport/internal/pkg/validator/hazardousmaterialvalidator"
	"github.com/trenova-app/transport/internal/pkg/validator/servicetypevalidator"
	"github.com/trenova-app/transport/internal/pkg/validator/shipmenttypevalidator"
	"github.com/trenova-app/transport/internal/pkg/validator/workervalidator"
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
))
