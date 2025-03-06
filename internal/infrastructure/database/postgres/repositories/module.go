package repositories

import (
	"go.uber.org/fx"
)

var Module = fx.Module("postgres-repositories", fx.Provide(
	NewPermissionRepository,
	NewAuditRepository,
	NewUserRepository,
	NewOrganizationRepository,
	NewUsStateRepository,
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
	NewShipmentCommodityRepository,
	NewShipmentMoveRepository,
	NewShipmentRepository,
	NewPCMilerConfigurationRepository,
	NewAssignmentRepository,
	NewShipmentControlRepository,
))
