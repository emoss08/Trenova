package api

import (
	"github.com/emoss08/trenova/internal/api/handlers/auth"
	"github.com/emoss08/trenova/internal/api/handlers/commodity"
	"github.com/emoss08/trenova/internal/api/handlers/customer"
	"github.com/emoss08/trenova/internal/api/handlers/documentqualityconfig"
	"github.com/emoss08/trenova/internal/api/handlers/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/api/handlers/equipmenttype"
	"github.com/emoss08/trenova/internal/api/handlers/fleetcode"
	"github.com/emoss08/trenova/internal/api/handlers/hazardousmaterial"
	"github.com/emoss08/trenova/internal/api/handlers/location"
	"github.com/emoss08/trenova/internal/api/handlers/locationcategory"
	"github.com/emoss08/trenova/internal/api/handlers/organization"
	"github.com/emoss08/trenova/internal/api/handlers/reporting"
	"github.com/emoss08/trenova/internal/api/handlers/routing"
	"github.com/emoss08/trenova/internal/api/handlers/search"
	"github.com/emoss08/trenova/internal/api/handlers/servicetype"
	"github.com/emoss08/trenova/internal/api/handlers/session"
	"github.com/emoss08/trenova/internal/api/handlers/shipment"
	"github.com/emoss08/trenova/internal/api/handlers/shipmenttype"
	"github.com/emoss08/trenova/internal/api/handlers/tableconfiguration"
	"github.com/emoss08/trenova/internal/api/handlers/tractor"
	"github.com/emoss08/trenova/internal/api/handlers/trailer"
	"github.com/emoss08/trenova/internal/api/handlers/user"
	"github.com/emoss08/trenova/internal/api/handlers/usstate"
	"github.com/emoss08/trenova/internal/api/handlers/worker"
	"github.com/emoss08/trenova/internal/pkg/validator"
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
	locationcategory.NewHandler,
	reporting.NewHandler,
	location.NewHandler,
	tractor.NewHandler,
	trailer.NewHandler,
	customer.NewHandler,
	shipment.NewHandler,
	routing.NewHandler,
))
