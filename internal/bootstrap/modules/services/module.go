package services

import (
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/services/auth"
	"github.com/emoss08/trenova/internal/core/services/commodity"
	"github.com/emoss08/trenova/internal/core/services/documentqualityconfig"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/services/equipmenttype"
	"github.com/emoss08/trenova/internal/core/services/file"
	"github.com/emoss08/trenova/internal/core/services/fleetcode"
	"github.com/emoss08/trenova/internal/core/services/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/services/location"
	"github.com/emoss08/trenova/internal/core/services/locationcategory"
	"github.com/emoss08/trenova/internal/core/services/organization"
	"github.com/emoss08/trenova/internal/core/services/permission"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/search"
	"github.com/emoss08/trenova/internal/core/services/servicetype"
	"github.com/emoss08/trenova/internal/core/services/session"
	"github.com/emoss08/trenova/internal/core/services/shipmenttype"
	"github.com/emoss08/trenova/internal/core/services/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/services/user"
	"github.com/emoss08/trenova/internal/core/services/usstate"
	"github.com/emoss08/trenova/internal/core/services/worker"
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
	locationcategory.NewService,
	reporting.NewService,
	location.NewService,
))
