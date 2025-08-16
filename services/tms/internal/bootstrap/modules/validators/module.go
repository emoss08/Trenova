/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package validators

import (
	"github.com/emoss08/trenova/internal/pkg/validator/accessorialchargevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/assignmentvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/billingcontrolvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/commodityvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/compliancevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/consolidationsettingvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/consolidationvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/customervalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/documenttypevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/equipmentmanufacturervalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/equipmenttypevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/fleetcodevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/emoss08/trenova/internal/pkg/validator/hazardousmaterialvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/hazmatsegreationrulevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/holdreasonvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/locationvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/notificationpreferencevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/organizationvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/servicetypevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentcontrolvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmenttypevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/tractorvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/trailervalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/workervalidator"
	"go.uber.org/fx"
)

var Module = fx.Module("validators",
	fx.Provide(
		compliancevalidator.NewValidator,
		fleetcodevalidator.NewValidator,
		locationvalidator.NewLocationCategoryValidator,
		equipmenttypevalidator.NewValidator,
		equipmentmanufacturervalidator.NewValidator,
		shipmenttypevalidator.NewValidator,
		servicetypevalidator.NewValidator,
		hazardousmaterialvalidator.NewValidator,
		commodityvalidator.NewValidator,
		locationvalidator.NewValidator,
		tractorvalidator.NewValidator,
		trailervalidator.NewValidator,
		customervalidator.NewValidator,
		hazmatsegreationrulevalidator.NewValidator,
		shipmentvalidator.NewStopValidator,
		shipmentvalidator.NewMoveValidator,
		shipmentvalidator.NewShipmentHoldValidator,
		shipmentvalidator.NewValidator,
		assignmentvalidator.NewValidator,
		shipmentcontrolvalidator.NewValidator,
		billingcontrolvalidator.NewValidator,
		accessorialchargevalidator.NewValidator,
		documenttypevalidator.NewValidator,
		organizationvalidator.NewValidator,
		workervalidator.NewWorkerPTOValidator,
		workervalidator.NewWorkerProfileValidator,
		workervalidator.NewValidator,
		notificationpreferencevalidator.NewValidator,
		consolidationsettingvalidator.NewValidator,
		consolidationvalidator.NewValidator,
		holdreasonvalidator.NewValidator,
	),
	fx.Options(
		framework.Module,
	),
)
