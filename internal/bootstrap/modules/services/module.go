package services

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/services/ai"
	"github.com/emoss08/trenova/internal/core/services/assignment"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/services/auth"
	"github.com/emoss08/trenova/internal/core/services/billingcontrol"
	"github.com/emoss08/trenova/internal/core/services/billingqueue"
	"github.com/emoss08/trenova/internal/core/services/calculator"
	"github.com/emoss08/trenova/internal/core/services/commodity"
	"github.com/emoss08/trenova/internal/core/services/consolidation"
	"github.com/emoss08/trenova/internal/core/services/consolidationsetting"
	"github.com/emoss08/trenova/internal/core/services/customer"
	"github.com/emoss08/trenova/internal/core/services/dbbackup"
	"github.com/emoss08/trenova/internal/core/services/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/services/docpreview"
	"github.com/emoss08/trenova/internal/core/services/document"
	"github.com/emoss08/trenova/internal/core/services/documentqualityconfig"
	"github.com/emoss08/trenova/internal/core/services/documenttype"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/services/equipmenttype"
	"github.com/emoss08/trenova/internal/core/services/favorite"
	"github.com/emoss08/trenova/internal/core/services/file"
	"github.com/emoss08/trenova/internal/core/services/fleetcode"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/services/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/services/imagegen"
	"github.com/emoss08/trenova/internal/core/services/integration"
	"github.com/emoss08/trenova/internal/core/services/location"
	"github.com/emoss08/trenova/internal/core/services/locationcategory"
	"github.com/emoss08/trenova/internal/core/services/notification"
	"github.com/emoss08/trenova/internal/core/services/organization"
	"github.com/emoss08/trenova/internal/core/services/permission"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/role"
	"github.com/emoss08/trenova/internal/core/services/routing"
	"github.com/emoss08/trenova/internal/core/services/servicetype"
	"github.com/emoss08/trenova/internal/core/services/session"
	"github.com/emoss08/trenova/internal/core/services/shipment"
	"github.com/emoss08/trenova/internal/core/services/shipmentcontrol"
	"github.com/emoss08/trenova/internal/core/services/shipmentmove"
	"github.com/emoss08/trenova/internal/core/services/shipmenttype"
	"github.com/emoss08/trenova/internal/core/services/stop"
	"github.com/emoss08/trenova/internal/core/services/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/services/tractor"
	"github.com/emoss08/trenova/internal/core/services/trailer"
	"github.com/emoss08/trenova/internal/core/services/user"
	"github.com/emoss08/trenova/internal/core/services/usstate"
	"github.com/emoss08/trenova/internal/core/services/websocket"
	"github.com/emoss08/trenova/internal/core/services/worker"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

var Module = fx.Module("services", fx.Provide(
	permission.NewService,
	role.NewService,
	file.NewService,
	audit.NewService,
	auth.NewService,
	ai.NewClassificationService,
	organization.NewService,
	session.NewService,
	usstate.NewService,
	user.NewService,
	worker.NewService,
	tableconfiguration.NewService,
	fleetcode.NewService,
	documentqualityconfig.NewService,
	equipmenttype.NewService,
	equipmentmanufacturer.NewService,
	shipmenttype.NewService,
	servicetype.NewService,
	hazardousmaterial.NewService,
	commodity.NewService,
	locationcategory.NewService,
	reporting.NewService,
	location.NewService,
	tractor.NewService,
	trailer.NewService,
	customer.NewService,
	consolidation.NewService,
	shipment.NewService,
	routing.NewService,
	assignment.NewService,
	shipmentmove.NewService,
	stop.NewService,
	shipmentcontrol.NewService,
	billingcontrol.NewService,
	dbbackup.NewService,
	hazmatsegregationrule.NewService,
	accessorialcharge.NewService,
	imagegen.NewService,
	docpreview.NewService,
	document.NewService,
	documenttype.NewService,
	integration.NewService,
	billingqueue.NewService,
	favorite.NewService,
	dedicatedlane.NewService,
	dedicatedlane.NewAssignmentService,
	dedicatedlane.NewPatternService,
	dedicatedlane.NewSuggestionService,
	websocket.NewService,
	notification.NewService,
	notification.NewPreferenceService,
	notification.NewBatchProcessor,
	notification.NewAuditListenerService,
	consolidationsetting.NewService,
	formula.NewService,
),
	fx.Invoke(func(s services.WebSocketService) { //nolint:revive // required for fx
		log.Info().Msg("websocket service initialized")
	}),
)

var CalculatorModule = fx.Module("calculator", fx.Provide(
	calculator.NewShipmentCalculator,
))
