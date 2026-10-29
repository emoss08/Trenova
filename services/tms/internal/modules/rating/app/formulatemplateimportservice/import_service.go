package formulatemplateimportservice

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	DB          kernelports.Transactor
	Templates   *formulatemplateservice.Service
	Repo        formulatemplateports.FormulaTemplateRepository
	VersionRepo formulatemplateports.FormulaTemplateVersionRepository
}

type Service struct {
	l           *zap.Logger
	db          kernelports.Transactor
	templates   *formulatemplateservice.Service
	repo        formulatemplateports.FormulaTemplateRepository
	versionRepo formulatemplateports.FormulaTemplateVersionRepository
}

//nolint:gocritic // fx param structs are passed by value
func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.formulatemplate"),
		db:          p.DB,
		templates:   p.Templates,
		repo:        p.Repo,
		versionRepo: p.VersionRepo,
	}
}
