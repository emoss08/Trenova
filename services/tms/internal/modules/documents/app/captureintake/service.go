package captureintake

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/captureprocessing"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturerequests"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Arrivals    *ArrivalReaders
	Batches     captureports.CaptureBatchRepository
	ContentGate contentscanports.Gate
	Imaging     kernelports.Imaging
	Logger      *zap.Logger
	Pages       captureports.CapturePageRepository
	Processing  *captureprocessing.Service
	Received    *ReceivedAnnouncer
	RequestRepo captureports.CaptureRequestRepository
	Requests    *capturerequests.Service
	Support     *capturesupport.Service
	// Workflows is optional so an installation without a worker still
	// receives pages. A sealed batch then waits, and the reconcile sweep
	// starts its processing once a worker appears.
	Workflows kernelports.WorkflowStarter `optional:"true"`
}

type Service struct {
	arrivals    *ArrivalReaders
	batches     captureports.CaptureBatchRepository
	contentGate contentscanports.Gate
	imaging     kernelports.Imaging
	l           *zap.Logger
	pages       captureports.CapturePageRepository
	received    *ReceivedAnnouncer
	requestRepo captureports.CaptureRequestRepository
	workflows   kernelports.WorkflowStarter
	processing  *captureprocessing.Service
	requests    *capturerequests.Service
	support     *capturesupport.Service
}

func New(p Params) *Service {
	return &Service{
		arrivals:    p.Arrivals,
		batches:     p.Batches,
		contentGate: p.ContentGate,
		imaging:     p.Imaging,
		l:           p.Logger.Named("service.capture"),
		pages:       p.Pages,
		received:    p.Received,
		requestRepo: p.RequestRepo,
		workflows:   p.Workflows,
		processing:  p.Processing,
		requests:    p.Requests,
		support:     p.Support,
	}
}
