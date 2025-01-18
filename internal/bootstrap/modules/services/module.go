package services

import (
	"github.com/trenova-app/transport/internal/core/services/audit"
	"github.com/trenova-app/transport/internal/core/services/auth"
	"github.com/trenova-app/transport/internal/core/services/commodity"
	"github.com/trenova-app/transport/internal/core/services/documentqualityconfig"
	"github.com/trenova-app/transport/internal/core/services/equipmentmanufacturer"
	"github.com/trenova-app/transport/internal/core/services/equipmenttype"
	"github.com/trenova-app/transport/internal/core/services/file"
	"github.com/trenova-app/transport/internal/core/services/fleetcode"
	"github.com/trenova-app/transport/internal/core/services/hazardousmaterial"
	"github.com/trenova-app/transport/internal/core/services/organization"
	"github.com/trenova-app/transport/internal/core/services/permission"
	"github.com/trenova-app/transport/internal/core/services/search"
	"github.com/trenova-app/transport/internal/core/services/servicetype"
	"github.com/trenova-app/transport/internal/core/services/session"
	"github.com/trenova-app/transport/internal/core/services/shipmenttype"
	"github.com/trenova-app/transport/internal/core/services/tableconfiguration"
	"github.com/trenova-app/transport/internal/core/services/user"
	"github.com/trenova-app/transport/internal/core/services/usstate"
	"github.com/trenova-app/transport/internal/core/services/worker"
	"go.uber.org/fx"
)

var Module = fx.Module("services", fx.Provide(
	permission.NewService,
	search.NewService,
	file.NewService,
	audit.NewService,
	auth.NewService,
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
))
