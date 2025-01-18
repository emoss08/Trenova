package api

import (
	"github.com/trenova-app/transport/internal/api/handlers/auth"
	"github.com/trenova-app/transport/internal/api/handlers/commodity"
	"github.com/trenova-app/transport/internal/api/handlers/documentqualityconfig"
	"github.com/trenova-app/transport/internal/api/handlers/equipmentmanufacturer"
	"github.com/trenova-app/transport/internal/api/handlers/equipmenttype"
	"github.com/trenova-app/transport/internal/api/handlers/fleetcode"
	"github.com/trenova-app/transport/internal/api/handlers/hazardousmaterial"
	"github.com/trenova-app/transport/internal/api/handlers/organization"
	"github.com/trenova-app/transport/internal/api/handlers/search"
	"github.com/trenova-app/transport/internal/api/handlers/servicetype"
	"github.com/trenova-app/transport/internal/api/handlers/session"
	"github.com/trenova-app/transport/internal/api/handlers/shipmenttype"
	"github.com/trenova-app/transport/internal/api/handlers/tableconfiguration"
	"github.com/trenova-app/transport/internal/api/handlers/user"
	"github.com/trenova-app/transport/internal/api/handlers/usstate"
	"github.com/trenova-app/transport/internal/api/handlers/worker"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"go.uber.org/fx"
)

var HandlersModule = fx.Module("api.Handlers", fx.Provide(
	validator.NewErrorHandler,
	auth.NewHandler,
	organization.NewHandler,
	usstate.NewHandler,
	user.NewHandler,
	session.NewHandler,
	search.NewHandler,
	worker.NewHandler,
	tableconfiguration.NewHandler,
	fleetcode.NewHandler,
	documentqualityconfig.NewHandler,
	equipmenttype.NewHandler,
	equipmentmanufacturer.NewHandler,
	shipmenttype.NewHandler,
	servicetype.NewHandler,
	hazardousmaterial.NewHandler,
	commodity.NewHandler,
))
