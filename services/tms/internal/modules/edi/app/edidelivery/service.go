package edidelivery

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/documentmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edialert"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Alert           *edialert.Service
	InvoiceRepo     invoiceports.InvoiceRepository
	Logger          *zap.Logger
	MessageRepo     edimessageports.EDIMessageRepository
	Metrics         *metrics.Registry `optional:"true"`
	PartnerRepo     edipartnerports.EDIPartnerRepository
	ProfileRepo     edipartnerports.EDICommunicationProfileRepository
	Realtime        realtimeports.RealtimeService `optional:"true"`
	Recipient       *editenderrecipient.Service
	Secret          *ediprofilesecret.Service
	Transport       editransportports.EDITransportDispatcher
	WorkflowStarter kernelports.WorkflowStarter
}

type Service struct {
	invoiceRepo     invoiceports.InvoiceRepository
	l               *zap.Logger
	messageRepo     edimessageports.EDIMessageRepository
	metrics         *documentmetrics.EDI
	partnerRepo     edipartnerports.EDIPartnerRepository
	profileRepo     edipartnerports.EDICommunicationProfileRepository
	realtime        realtimeports.RealtimeService
	transport       editransportports.EDITransportDispatcher
	workflowStarter kernelports.WorkflowStarter
	alert           *edialert.Service
	recipient       *editenderrecipient.Service
	secret          *ediprofilesecret.Service
}

func New(p Params) *Service {
	ediMetrics := documentmetrics.NewEDI(nil, p.Logger, false)
	if p.Metrics != nil {
		ediMetrics = p.Metrics.EDI
	}
	return &Service{
		invoiceRepo:     p.InvoiceRepo,
		l:               p.Logger.Named("service.edi"),
		messageRepo:     p.MessageRepo,
		metrics:         ediMetrics,
		partnerRepo:     p.PartnerRepo,
		profileRepo:     p.ProfileRepo,
		realtime:        p.Realtime,
		transport:       p.Transport,
		workflowStarter: p.WorkflowStarter,
		alert:           p.Alert,
		recipient:       p.Recipient,
		secret:          p.Secret,
	}
}
