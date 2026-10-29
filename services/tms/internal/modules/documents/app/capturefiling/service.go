package capturefiling

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/captureprocessing"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturerequests"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Batches     captureports.CaptureBatchRepository
	CoverSheets captureports.CaptureCoverSheetRepository
	DB          kernelports.Transactor
	Items       captureports.CaptureItemRepository
	Logger      *zap.Logger
	Pages       captureports.CapturePageRepository
	Processing  *captureprocessing.Service
	Requests    *capturerequests.Service
	Support     *capturesupport.Service
	Uploads     documentuploadports.DocumentUploadService
	// Workflows is optional so an installation without a worker still
	// receives pages. A sealed batch then waits, and the reconcile sweep
	// starts its processing once a worker appears.
	Workflows kernelports.WorkflowStarter `optional:"true"`
}

type Service struct {
	batches     captureports.CaptureBatchRepository
	coverSheets captureports.CaptureCoverSheetRepository
	db          kernelports.Transactor
	items       captureports.CaptureItemRepository
	l           *zap.Logger
	pages       captureports.CapturePageRepository
	uploads     documentuploadports.DocumentUploadService
	workflows   kernelports.WorkflowStarter
	processing  *captureprocessing.Service
	requests    *capturerequests.Service
	support     *capturesupport.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		batches:     p.Batches,
		coverSheets: p.CoverSheets,
		db:          p.DB,
		items:       p.Items,
		l:           p.Logger.Named("service.capture"),
		pages:       p.Pages,
		uploads:     p.Uploads,
		workflows:   p.Workflows,
		processing:  p.Processing,
		requests:    p.Requests,
		support:     p.Support,
	}
}
