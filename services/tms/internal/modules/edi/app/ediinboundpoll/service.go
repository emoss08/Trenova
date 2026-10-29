package ediinboundpoll

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/documentmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	ProfileRepo edipartnerports.EDICommunicationProfileRepository
	Secrets     *ediprofilesecret.Service
	Inbound     *ediinboundservice.Service
	Transport   editransportports.EDITransportDispatcher
	Metrics     *metrics.Registry `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	profileRepo edipartnerports.EDICommunicationProfileRepository
	secrets     *ediprofilesecret.Service
	inbound     *ediinboundservice.Service
	transport   editransportports.EDITransportDispatcher
	metrics     *documentmetrics.EDI
}

func New(p Params) *Service {
	ediMetrics := documentmetrics.NewEDI(nil, p.Logger, false)
	if p.Metrics != nil {
		ediMetrics = p.Metrics.EDI
	}
	return &Service{
		l:           p.Logger.Named("service.edi-inbound"),
		profileRepo: p.ProfileRepo,
		secrets:     p.Secrets,
		inbound:     p.Inbound,
		transport:   p.Transport,
		metrics:     ediMetrics,
	}
}
