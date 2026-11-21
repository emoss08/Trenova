package reportdashboardservice

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	Reporting      *reporting.Service
	DashboardRepo  reportports.ReportDashboardRepository
	DefinitionRepo reportports.ReportDefinitionRepository
	Engine         reportports.ReportEngine
	Slots          kernelports.HeavyJobSlots
}

type Service struct {
	l             *zap.Logger
	reporting     *reporting.Service
	dashboardRepo reportports.ReportDashboardRepository
	defRepo       reportports.ReportDefinitionRepository
	engine        reportports.ReportEngine
	slots         kernelports.HeavyJobSlots
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.reportdashboard"),
		reporting:     p.Reporting,
		dashboardRepo: p.DashboardRepo,
		defRepo:       p.DefinitionRepo,
		engine:        p.Engine,
		slots:         p.Slots,
	}
}
