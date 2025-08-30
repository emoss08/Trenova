/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories/resourceeditorrepo"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories/shipment"
	"go.uber.org/fx"
)

var Module = fx.Module("postgres-repositories", fx.Provide(
	NewPermissionRepository,
	NewAuditRepository,
	NewUserRepository,
	NewOrganizationRepository,
	NewUsStateRepository,
	NewDocumentRepository,
	NewWorkerRepository,
	NewHazmatExpirationRepository,
	NewTableConfigurationRepository,
	NewFleetCodeRepository,
	NewDocumentQualityConfigRepository,
	NewEquipmentTypeRepository,
	NewEquipmentManufacturerRepository,
	NewShipmentTypeRepository,
	NewServiceTypeRepository,
	NewHazardousMaterialRepository,
	NewCommodityRepository,
	NewLocationCategoryRepository,
	NewLocationRepository,
	NewTractorRepository,
	NewTrailerRepository,
	NewCustomerRepository,
	NewProNumberRepository,
	NewStopRepository,
	NewAdditionalChargeRepository,
	NewShipmentCommodityRepository,
	NewShipmentMoveRepository,
	NewShipmentCommentRepository,
	shipment.NewShipmentHoldRepository,
	shipment.NewShipmentRepository,
	NewPCMilerConfigurationRepository,
	NewAssignmentRepository,
	NewShipmentControlRepository,
	NewBillingControlRepository,
	NewHazmatSegregationRuleRepository,
	NewAccessorialChargeRepository,
	NewDocumentTypeRepository,
	NewIntegrationRepository,
	NewBillingQueueRepository,
	resourceeditorrepo.NewRepository,
	NewPatternConfigRepository,
	NewFavoriteRepository,
	NewDedicatedLaneRepository,
	NewDedicatedLaneSuggestionRepository,
	NewNotificationRepository,
	NewNotificationPreferenceRepository,
	NewConsolidationRepository,
	NewConsolidationSettingRepository,
	NewFormulaTemplateRepository,
	NewEmailProfileRepository,
	NewEmailTemplateRepository,
	NewEmailQueueRepository,
	NewEmailLogRepository,
	NewHoldReasonRepository,
	NewDataRetentionRepository,
))
