package apikeyservice

import "github.com/emoss08/trenova/internal/core/domain/permission"

var runtimePolicy = map[string][]permission.Operation{
	permission.ResourceCustomer.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceLocation.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceLocationCategory.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceWorker.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceTrailer.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceTractor.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceEquipmentType.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceEquipmentManufacturer.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceCommodity.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceHazardousMaterial.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceShipmentType.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceServiceType.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
	permission.ResourceShipment.String(): {
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
	},
}
