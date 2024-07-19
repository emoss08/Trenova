// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
	registerFlexibleHandler(r, NewShipmentHandler(s))

	// Test routes for development.
	registerFlexibleHandler(r, NewTestHandler(s))
}
