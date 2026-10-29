package reportscheduleservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Reporting    *reporting.Service
	ScheduleRepo reportports.ReportScheduleRepository
	UserRepo     tenantports.UserRepository
}

type Service struct {
	l            *zap.Logger
	reporting    *reporting.Service
	scheduleRepo reportports.ReportScheduleRepository
	userRepo     tenantports.UserRepository
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.reportschedule"),
		reporting:    p.Reporting,
		scheduleRepo: p.ScheduleRepo,
		userRepo:     p.UserRepo,
	}
}
