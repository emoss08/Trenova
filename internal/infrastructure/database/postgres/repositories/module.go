package repositories

import (
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories/resourceeditorrepo"
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
	NewShipmentRepository,
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
	NewFavoriteRepository,
))
