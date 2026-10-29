package captureflow

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturedevices"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturefiling"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureintake"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturemaintenance"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureprocessing"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureprofiles"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturerequests"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/zap"
)

type testService struct {
	l             *zap.Logger
	cfg           *appconfig.Config
	db            kernelports.Transactor
	deviceRepo    captureports.CaptureDeviceRepository
	pairings      captureports.CapturePairingRepository
	profileRepo   captureports.CaptureProfileRepository
	requestRepo   captureports.CaptureRequestRepository
	batches       captureports.CaptureBatchRepository
	pages         captureports.CapturePageRepository
	items         captureports.CaptureItemRepository
	coverSheets   captureports.CaptureCoverSheetRepository
	records       captureports.CaptureRecordFinder
	controls      tenantports.DocumentControlRepository
	documentTypes documenttypeports.DocumentTypeRepository
	users         tenantports.UserRepository
	organizations tenantports.OrganizationRepository
	permissions   permissionports.PermissionEngine
	storage       kernelports.Client
	cipher        *encryptionservice.Service
	uploads       documentuploadports.DocumentUploadService
	imaging       kernelports.Imaging
	qrCodes       captureports.CaptureQREncoder
	notifications capturemaintenance.Notifier
	workflows     kernelports.WorkflowStarter
	realtime      realtimeports.RealtimeService
	audit         auditports.AuditService
	analyzer      captureports.CaptureAnalyzer
	shipments     inboundmessageports.InboundShipmentFinder
	received      *captureintake.ReceivedAnnouncer
	arrivals      *captureintake.ArrivalReaders
	contentGate   contentscanports.Gate
}

func (t *testService) support() *capturesupport.Service {
	return capturesupport.New(capturesupport.Params{
		Audit:           t.audit,
		DocumentControl: t.controls,
		Encryption:      t.cipher,
		Logger:          t.l,
		Permissions:     t.permissions,
		Realtime:        t.realtime,
		Storage:         t.storage,
	})
}

func (t *testService) profiles() *captureprofiles.Service {
	return captureprofiles.New(captureprofiles.Params{
		ProfileRepo: t.profileRepo,
		Support:     t.support(),
	})
}

func (t *testService) devices() *capturedevices.Service {
	return capturedevices.New(capturedevices.Params{
		AppConfig:     t.cfg,
		DB:            t.db,
		DeviceRepo:    t.deviceRepo,
		Logger:        t.l,
		Organizations: t.organizations,
		Pairings:      t.pairings,
		Profiles:      t.profiles(),
		Support:       t.support(),
		Users:         t.users,
	})
}

func (t *testService) requests() *capturerequests.Service {
	return capturerequests.New(capturerequests.Params{
		CoverSheets:   t.coverSheets,
		DeviceRepo:    t.deviceRepo,
		DocumentTypes: t.documentTypes,
		Logger:        t.l,
		ProfileRepo:   t.profileRepo,
		QRCodes:       t.qrCodes,
		Records:       t.records,
		RequestRepo:   t.requestRepo,
		Support:       t.support(),
	})
}

func (t *testService) processing() *captureprocessing.Service {
	return captureprocessing.New(captureprocessing.Params{
		Analyzer:      t.analyzer,
		Batches:       t.batches,
		CoverSheets:   t.coverSheets,
		DocumentTypes: t.documentTypes,
		Imaging:       t.imaging,
		Items:         t.items,
		Logger:        t.l,
		Pages:         t.pages,
		ProfileRepo:   t.profileRepo,
		Shipments:     t.shipments,
		Support:       t.support(),
	})
}

func (t *testService) intake() *captureintake.Service {
	return captureintake.New(captureintake.Params{
		Arrivals:    t.arrivals,
		Batches:     t.batches,
		ContentGate: t.contentGate,
		Imaging:     t.imaging,
		Logger:      t.l,
		Pages:       t.pages,
		Processing:  t.processing(),
		Received:    t.received,
		RequestRepo: t.requestRepo,
		Requests:    t.requests(),
		Support:     t.support(),
		Workflows:   t.workflows,
	})
}

func (t *testService) filing() *capturefiling.Service {
	return capturefiling.New(capturefiling.Params{
		Batches:     t.batches,
		CoverSheets: t.coverSheets,
		DB:          t.db,
		Items:       t.items,
		Logger:      t.l,
		Pages:       t.pages,
		Processing:  t.processing(),
		Requests:    t.requests(),
		Support:     t.support(),
		Uploads:     t.uploads,
		Workflows:   t.workflows,
	})
}

func (t *testService) maintenance() *capturemaintenance.Service {
	return capturemaintenance.New(capturemaintenance.Params{
		Batches:       t.batches,
		Devices:       t.devices(),
		Filing:        t.filing(),
		Intake:        t.intake(),
		Logger:        t.l,
		Notifications: t.notifications,
		Pages:         t.pages,
		Requests:      t.requests(),
		Support:       t.support(),
		Workflows:     t.workflows,
	})
}
