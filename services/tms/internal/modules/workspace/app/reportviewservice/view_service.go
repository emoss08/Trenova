package reportviewservice

import (
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger    *zap.Logger
	Reporting *reporting.Service
	ViewRepo  reportports.ReportViewRepository
}

type Service struct {
	l         *zap.Logger
	reporting *reporting.Service
	viewRepo  reportports.ReportViewRepository
}

//nolint:gocritic // fx.In parameter structs must be passed by value
func New(p Params) *Service {
	return &Service{
		l:         p.Logger.Named("service.reportview"),
		reporting: p.Reporting,
		viewRepo:  p.ViewRepo,
	}
}
