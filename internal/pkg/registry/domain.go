package registry

import (
	"github.com/trenova-app/transport/internal/core/domain/businessunit"
	"github.com/trenova-app/transport/internal/core/domain/commodity"
	"github.com/trenova-app/transport/internal/core/domain/compliance"
	"github.com/trenova-app/transport/internal/core/domain/documentqualityconfig"
	"github.com/trenova-app/transport/internal/core/domain/documentqualityfeedback"
	"github.com/trenova-app/transport/internal/core/domain/equipmentmanufacturer"
	"github.com/trenova-app/transport/internal/core/domain/equipmenttype"
	"github.com/trenova-app/transport/internal/core/domain/fleetcode"
	"github.com/trenova-app/transport/internal/core/domain/hazardousmaterial"
	"github.com/trenova-app/transport/internal/core/domain/location"
	"github.com/trenova-app/transport/internal/core/domain/organization"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/domain/pretrainedmodels"
	"github.com/trenova-app/transport/internal/core/domain/servicetype"
	"github.com/trenova-app/transport/internal/core/domain/session"
	"github.com/trenova-app/transport/internal/core/domain/shipment"
	"github.com/trenova-app/transport/internal/core/domain/shipmenttype"
	"github.com/trenova-app/transport/internal/core/domain/tableconfiguration"
	"github.com/trenova-app/transport/internal/core/domain/user"
	"github.com/trenova-app/transport/internal/core/domain/usstate"
	"github.com/trenova-app/transport/internal/core/domain/worker"
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
