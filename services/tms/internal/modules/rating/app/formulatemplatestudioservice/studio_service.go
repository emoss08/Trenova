package formulatemplatestudioservice

import (
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger          *zap.Logger
	Templates       *formulatemplateservice.Service
	VersionRepo     formulatemplateports.FormulaTemplateVersionRepository
	LibraryVersions formulalibraryports.FormulaLibraryVersionRepository
	Checker         *tariff.Checker
	Resolver        *tariff.TemplateResolver
	Rater           tariffports.Rater
	Records         tariffports.RecordBuilder  `optional:"true"`
	RatedShipments  tariffports.RatedShipments `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	templates   *formulatemplateservice.Service
	versionRepo formulatemplateports.FormulaTemplateVersionRepository
	libraries   formulatemplateservice.ApprovedLibraries
	checker     formulatemplateservice.SourceChecker
	resolver    formulatemplateservice.BillingResolver
	rater       tariffports.Rater
	records     tariffports.RecordBuilder
	rated       tariffports.RatedShipments
}

//nolint:gocritic // fx param structs are passed by value
func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.formulatemplate"),
		templates:   p.Templates,
		versionRepo: p.VersionRepo,
		libraries:   p.LibraryVersions,
		checker:     p.Checker,
		resolver:    p.Resolver,
		rater:       p.Rater,
		records:     p.Records,
		rated:       p.RatedShipments,
	}
}
