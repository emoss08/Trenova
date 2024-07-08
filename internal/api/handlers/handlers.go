package handlers

import (
	"github.com/emoss08/trenova/internal/server"
	"github.com/gofiber/fiber/v2"
)

func registerHandler(r fiber.Router, h StandardHandler) {
	h.RegisterRoutes(r)
}

func registerFlexibleHandler(r fiber.Router, h FlexibleHandler) {
	h.RegisterRoutes(r)
}

func AttachAllRoutes(s *server.Server, r fiber.Router) {
	// Routes that follow the standard pattern.
	registerHandler(r, NewLocationCategoryHandler(s))
	registerHandler(r, NewFleetCodeHandler(s))
	registerHandler(r, NewDelayCodeHandler(s))
	registerHandler(r, NewChargeTypeHandler(s))
	registerHandler(r, NewCommentTypeHandler(s))
	registerHandler(r, NewTableChangeAlertHandler(s))
	registerHandler(r, NewGeneralLedgerAccountHandler(s))
	registerHandler(r, NewTagHandler(s))
	registerHandler(r, NewDivisionCodeHandler(s))
	registerHandler(r, NewDocumentClassificationHandler(s))
	registerHandler(r, NewEquipmentTypeHandler(s))
	registerHandler(r, NewRevenueCodeHandler(s))
	registerHandler(r, NewAccessorialChargeHandler(s))
	registerHandler(r, NewEquipmentManufacturerHandler(s))
	registerHandler(r, NewTrailerHandler(s))
	registerHandler(r, NewTractorHandler(s))
	registerHandler(r, NewHazardousMaterialHandler(s))
	registerHandler(r, NewCommodityHandler(s))
	registerHandler(r, NewReasonCodeHandler(s))
	registerHandler(r, NewShipmentTypeHandler(s))
	registerHandler(r, NewServiceTypeHandler(s))
	registerHandler(r, NewQualifierCodeHandler(s))
	registerHandler(r, NewWorkerHandler(s))
	registerHandler(r, NewLocationHandler(s))
	registerHandler(r, NewCustomerHandler(s))

	// Routes that don't follow the standard pattern.
	registerFlexibleHandler(r, NewUserHandler(s))
	registerFlexibleHandler(r, NewUserFavoriteHandler(s))
	registerFlexibleHandler(r, NewUserNotificationHandler(s))
	registerFlexibleHandler(r, NewReportHandler(s))
	registerFlexibleHandler(r, NewOrganizationHandler(s))
	registerFlexibleHandler(r, NewUserTaskHandler(s))
	registerFlexibleHandler(r, NewUSStateHandler(s))

	// Test routes for development.
	registerFlexibleHandler(r, NewTestHandler(s))
}
