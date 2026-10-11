// Package captureservice is the server side of Trenova Capture: pairing a
// Windows companion to a person, receiving the pages it scans or prints,
// dividing them into documents, and filing those documents onto records.
//
// The companion never decides anything. It captures pages and uploads them;
// where a stack divides, where each part goes, and whether it may be filed
// without a person are all decided here, so the rules are the same for every
// device and can change without shipping a new install.
package captureservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ErrCaptureDisabled is what every capture call gets in an organization that
// turned capture off. It is a refusal, not a failure: the device keeps its
// pages and tells the person why.
var ErrCaptureDisabled = errors.New("document capture is turned off for this organization")

var ErrNoWebAddress = errors.New(
	"this Trenova server has no web address configured (app.webBaseUrl), so a computer cannot be approved",
)

// DisabledReasonParam and DisabledReason tag ErrCaptureDisabled in the problem
// body, so the companion can tell "hold the pages until capture is turned
// back on" from a refusal of the upload itself without reading its text.
const (
	DisabledReasonParam = "reason"
	DisabledReason      = "capture_disabled"
)

// envelopeCipher is the slice of the encryption service capture uses: sealing
// pages and thumbnails at rest and opening them again.
type envelopeCipher interface {
	EncryptBytesWithAAD(plaintext []byte, aad encryptionservice.AAD) (string, error)
	DecryptBytesWithAAD(value string, aad encryptionservice.AAD) ([]byte, error)
}

type Params struct {
	fx.In

	Logger          *zap.Logger
	Config          *config.Config
	DB              kernelports.DBConnection
	Devices         captureports.CaptureDeviceRepository
	Pairings        captureports.CapturePairingRepository
	Profiles        captureports.CaptureProfileRepository
	Requests        captureports.CaptureRequestRepository
	Batches         captureports.CaptureBatchRepository
	Pages           captureports.CapturePageRepository
	Items           captureports.CaptureItemRepository
	CoverSheets     captureports.CaptureCoverSheetRepository
	Records         captureports.CaptureRecordFinder
	DocumentControl tenantports.DocumentControlRepository
	DocumentTypes   documenttypeports.DocumentTypeRepository
	Users           tenantports.UserRepository
	Organizations   tenantports.OrganizationRepository
	Permissions     permissionports.PermissionEngine
	Storage         kernelports.Client
	Encryption      *encryptionservice.Service
	Uploads         documentuploadports.DocumentUploadService
	Assembler       captureports.CapturePDFAssembler
	Inspector       captureports.CapturePageInspector
	QRCodes         captureports.CaptureQREncoder
	// Workflows is optional so an installation without a worker still
	// receives pages. A sealed batch then waits, and the reconcile sweep
	// starts its processing once a worker appears.
	Workflows kernelports.WorkflowStarter `optional:"true"`
	// Realtime keeps the intake queue and the device's stream current. Without
	// it both are right on their next load, and the device polls.
	Realtime agentports.RealtimeService `optional:"true"`
	Audit    auditports.AuditService    `optional:"true"`
	// Analyzer suggests what a document is and which shipment it belongs to.
	// Without it every item waits for a person to say.
	Analyzer captureports.CaptureAnalyzer `optional:"true"`
	// Shipments resolves a reference number the analyzer found to a shipment.
	Shipments inboundmessageports.InboundShipmentFinder `optional:"true"`
	// Notifications tells an owner their unfiled pages are about to go.
	// Without it the pages are still kept and deleted on time; nobody is told.
	Notifications *notificationservice.Service `optional:"true"`
}

// userReader and organizationReader name the person and organization a device
// acts for, so the companion can say who it is signed in as.
type userReader interface {
	GetByID(ctx context.Context, req tenantports.GetUserByIDRequest) (*tenant.User, error)
}

type organizationReader interface {
	GetByID(
		ctx context.Context,
		req tenantports.GetOrganizationByIDRequest,
	) (*tenant.Organization, error)
}

// notifier is the one thing capture asks of notifications.
type notifier interface {
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
}

type Service struct {
	l             *zap.Logger
	cfg           *config.Config
	db            kernelports.DBConnection
	devices       captureports.CaptureDeviceRepository
	pairings      captureports.CapturePairingRepository
	profiles      captureports.CaptureProfileRepository
	requests      captureports.CaptureRequestRepository
	batches       captureports.CaptureBatchRepository
	pages         captureports.CapturePageRepository
	items         captureports.CaptureItemRepository
	coverSheets   captureports.CaptureCoverSheetRepository
	records       captureports.CaptureRecordFinder
	controls      tenantports.DocumentControlRepository
	documentTypes documenttypeports.DocumentTypeRepository
	users         userReader
	organizations organizationReader
	permissions   permissionports.PermissionEngine
	storage       kernelports.Client
	cipher        envelopeCipher
	uploads       documentuploadports.DocumentUploadService
	assembler     captureports.CapturePDFAssembler
	inspector     captureports.CapturePageInspector
	qrCodes       captureports.CaptureQREncoder
	notifications notifier
	workflows     kernelports.WorkflowStarter
	realtime      agentports.RealtimeService
	audit         auditports.AuditService
	analyzer      captureports.CaptureAnalyzer
	shipments     inboundmessageports.InboundShipmentFinder
	received      *receivedAnnouncer
	arrivals      *arrivalReaders
}

//nolint:gocritic // dependency injection param
func New(p Params) *Service {
	var cipher envelopeCipher
	if p.Encryption != nil {
		cipher = p.Encryption
	}
	var notifications notifier
	if p.Notifications != nil {
		notifications = p.Notifications
	}

	return &Service{
		l:             p.Logger.Named("service.capture"),
		cfg:           p.Config,
		db:            p.DB,
		devices:       p.Devices,
		pairings:      p.Pairings,
		profiles:      p.Profiles,
		requests:      p.Requests,
		batches:       p.Batches,
		pages:         p.Pages,
		items:         p.Items,
		coverSheets:   p.CoverSheets,
		records:       p.Records,
		controls:      p.DocumentControl,
		documentTypes: p.DocumentTypes,
		users:         p.Users,
		organizations: p.Organizations,
		permissions:   p.Permissions,
		storage:       p.Storage,
		cipher:        cipher,
		uploads:       p.Uploads,
		assembler:     p.Assembler,
		inspector:     p.Inspector,
		qrCodes:       p.QRCodes,
		workflows:     p.Workflows,
		realtime:      p.Realtime,
		audit:         p.Audit,
		analyzer:      p.Analyzer,
		shipments:     p.Shipments,
		notifications: notifications,
		received:      newReceivedAnnouncer(receivedAnnounceEvery),
		arrivals:      newArrivalReaders(arrivalReaderCount, arrivalQueueLength),
	}
}

// control is the tenant's document settings. A tenant that never saved them
// gets the defaults, which is what it would have been given on creation.
func (s *Service) control(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.DocumentControl, error) {
	control, err := s.controls.Get(ctx, tenantports.GetDocumentControlRequest{
		TenantInfo: tenantInfo,
	})
	if err == nil {
		return control, nil
	}
	if errortypes.IsNotFoundError(err) {
		return tenant.NewDefaultDocumentControl(tenantInfo.OrgID, tenantInfo.BuID), nil
	}

	return nil, err
}

// requireEnabled refuses when the tenant turned capture off.
func (s *Service) requireEnabled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.DocumentControl, error) {
	control, err := s.control(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !control.EnableCapture {
		return nil, errortypes.NewBusinessError(ErrCaptureDisabled.Error()).
			WithParam(DisabledReasonParam, DisabledReason).
			WithInternal(ErrCaptureDisabled)
	}

	return control, nil
}

// allowed reports whether the person may do op on resource, and at what data
// scope. It is the one place capture asks, so a device and a browser session
// acting for the same person always get the same answer.
func (s *Service) allowed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource permission.Resource,
	op permission.Operation,
) (*permissionports.PermissionCheckResult, error) {
	if tenantInfo.UserID.IsNil() {
		return &permissionports.PermissionCheckResult{Allowed: false, Reason: "no_user"}, nil
	}

	return s.permissions.Check(
		ctx,
		permissionports.UserActor(tenantInfo).PermissionCheck(resource, op),
	)
}

// require is allowed, turned into the error a handler maps to 403.
func (s *Service) require(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource permission.Resource,
	op permission.Operation,
) (*permissionports.PermissionCheckResult, error) {
	result, err := s.allowed(ctx, tenantInfo, resource, op)
	if err != nil {
		return nil, err
	}
	if !result.Allowed {
		return nil, errortypes.NewAuthorizationError(
			"You do not have permission to " + string(op) + " " + resource.String(),
		)
	}

	return result, nil
}

// seesEveryone reports whether a permission result reaches past the person's
// own records.
func seesEveryone(result *permissionports.PermissionCheckResult) bool {
	return result != nil && result.DataScope != permission.DataScopeOwn
}

// publish tells open screens that a capture record changed. It is addressed
// to one person when audience is set, which is how a device's stream learns
// of a request without the rest of the tenant hearing about it.
func (s *Service) publish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource permission.Resource,
	recordID pulid.ID,
	action string,
	audience pulid.ID,
	entity any,
) {
	if s.realtime == nil {
		return
	}

	if err := s.realtime.PublishResourceInvalidation(
		ctx,
		&agentports.PublishResourceInvalidationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			AudienceUserID: audience,
			Resource:       resource.String(),
			Action:         action,
			RecordID:       recordID,
			Entity:         entity,
			ActorUserID:    tenantInfo.UserID,
			ActorType:      permissionports.PrincipalTypeUser,
			ActorID:        tenantInfo.UserID,
		},
	); err != nil {
		s.l.Warn("capture invalidation lost",
			zap.String("resource", resource.String()),
			zap.String("recordId", recordID.String()),
			zap.Error(err))
	}
}

// logAudit records an action in the audit trail. A failure to record is
// logged, never returned: the action already happened.
func (s *Service) logAudit(params *auditports.LogActionParams, comment string) {
	if s.audit == nil {
		return
	}

	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to record capture audit entry",
			zap.String("resource", params.Resource.String()),
			zap.String("resourceId", params.ResourceID),
			zap.Error(err))
	}
}

// Access is what the web app needs to decide whether to offer capture at
// all: whether the organization turned it on, and whether this person may
// use it. Neither is a secret, and asking needs no permission beyond being
// signed in, so a record's page can hide the scan button rather than show
// one that fails.
type Access struct {
	Enabled    bool `json:"enabled"`
	CanCapture bool `json:"canCapture"`
}

func (s *Service) Access(ctx context.Context, tenantInfo pagination.TenantInfo) (*Access, error) {
	control, err := s.control(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	result, err := s.allowed(ctx, tenantInfo, permission.ResourceCaptureBatch, permission.OpCreate)
	if err != nil {
		return nil, err
	}

	return &Access{Enabled: control.EnableCapture, CanCapture: result.Allowed}, nil
}
