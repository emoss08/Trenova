package formulatemplatereviewservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	DB            kernelports.Transactor
	Templates     *formulatemplateservice.Service
	Repo          formulatemplateports.FormulaTemplateRepository
	VersionRepo   formulatemplateports.FormulaTemplateVersionRepository
	ReviewRepo    formulatemplateports.FormulaTemplateReviewRepository
	Checker       *tariff.Checker
	Resolver      *tariff.TemplateResolver
	Dependents    rateagreementports.TemplateInputsLister
	Notifications *notificationservice.Service   `optional:"true"`
	Roles         permissionports.RoleRepository `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	db            kernelports.Transactor
	templates     *formulatemplateservice.Service
	repo          formulatemplateports.FormulaTemplateRepository
	versionRepo   formulatemplateports.FormulaTemplateVersionRepository
	reviewRepo    formulatemplateports.FormulaTemplateReviewRepository
	checker       formulatemplateservice.SourceChecker
	resolver      formulatemplateservice.BillingResolver
	dependents    rateagreementports.TemplateInputsLister
	notifications *notificationservice.Service
	roles         permissionports.RoleRepository
}

//nolint:gocritic // fx param structs are passed by value
func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.formulatemplate"),
		db:            p.DB,
		templates:     p.Templates,
		repo:          p.Repo,
		versionRepo:   p.VersionRepo,
		reviewRepo:    p.ReviewRepo,
		checker:       p.Checker,
		resolver:      p.Resolver,
		dependents:    p.Dependents,
		notifications: p.Notifications,
		roles:         p.Roles,
	}
}
