package registry

import (
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/compliance"
	"github.com/emoss08/trenova/internal/core/domain/documentqualityconfig"
	"github.com/emoss08/trenova/internal/core/domain/documentqualityfeedback"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/pretrainedmodels"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
)

func RegisterEntities() []any {
	return []any{
		&usstate.UsState{},
		&businessunit.BusinessUnit{},
		&organization.Organization{},
		&session.Event{},
		&session.Session{},
		&permission.RolePermission{},
		&permission.Permission{},
		&permission.Role{},
		&user.UserRole{},
		&user.UserOrganization{},
		&user.User{},
		&worker.WorkerDocument{},
		&worker.WorkerPTO{},
		&worker.Worker{},
		&worker.WorkerProfile{},
		&compliance.HazmatExpiration{},
		&tableconfiguration.ConfigurationShare{},
		&tableconfiguration.Configuration{},
		&fleetcode.FleetCode{},
		&documentqualityconfig.DocumentQualityConfig{},
		&documentqualityfeedback.DocumentQualityFeedback{},
		&pretrainedmodels.PretrainedModel{},
		&equipmenttype.EquipmentType{},
		&equipmentmanufacturer.EquipmentManufacturer{},
		&shipmenttype.ShipmentType{},
		&servicetype.ServiceType{},
		&hazardousmaterial.HazardousMaterial{},
		&commodity.Commodity{},
		&shipment.ShipmentCommodity{},
		&shipment.Shipment{},
		&location.LocationCategory{},
		&location.Location{},
	}
}
