package ediinboundas2

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger          *zap.Logger
	ProfileRepo     edipartnerports.EDICommunicationProfileRepository
	EDIService      *ediservice.Service
	Inbound         *ediinboundservice.Service
	WorkflowStarter kernelports.WorkflowStarter
	Metrics         *metrics.Registry `optional:"true"`
}

type Service struct {
	l               *zap.Logger
	profileRepo     edipartnerports.EDICommunicationProfileRepository
	ediService      *ediservice.Service
	inbound         *ediinboundservice.Service
	workflowStarter kernelports.WorkflowStarter
	metrics         *metrics.EDI
}

func New(p Params) *Service {
	ediMetrics := metrics.NewEDI(nil, p.Logger, false)
	if p.Metrics != nil {
		ediMetrics = p.Metrics.EDI
	}
	return &Service{
		l:               p.Logger.Named("service.edi-inbound"),
		profileRepo:     p.ProfileRepo,
		ediService:      p.EDIService,
		inbound:         p.Inbound,
		workflowStarter: p.WorkflowStarter,
		metrics:         ediMetrics,
	}
}
