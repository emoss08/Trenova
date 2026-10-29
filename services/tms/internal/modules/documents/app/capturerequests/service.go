package capturerequests

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	CoverSheets   captureports.CaptureCoverSheetRepository
	DeviceRepo    captureports.CaptureDeviceRepository
	DocumentTypes documenttypeports.DocumentTypeRepository
	Logger        *zap.Logger
	ProfileRepo   captureports.CaptureProfileRepository
	QRCodes       captureports.CaptureQREncoder
	Records       captureports.CaptureRecordFinder
	RequestRepo   captureports.CaptureRequestRepository
	Support       *capturesupport.Service
}

type Service struct {
	coverSheets   captureports.CaptureCoverSheetRepository
	deviceRepo    captureports.CaptureDeviceRepository
	documentTypes documenttypeports.DocumentTypeRepository
	l             *zap.Logger
	profileRepo   captureports.CaptureProfileRepository
	qrCodes       captureports.CaptureQREncoder
	records       captureports.CaptureRecordFinder
	requestRepo   captureports.CaptureRequestRepository
	support       *capturesupport.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		coverSheets:   p.CoverSheets,
		deviceRepo:    p.DeviceRepo,
		documentTypes: p.DocumentTypes,
		l:             p.Logger.Named("service.capture"),
		profileRepo:   p.ProfileRepo,
		qrCodes:       p.QRCodes,
		records:       p.Records,
		requestRepo:   p.RequestRepo,
		support:       p.Support,
	}
}
