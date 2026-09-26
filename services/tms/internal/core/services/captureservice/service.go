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

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
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
	DB              ports.DBConnection
	Devices         repositories.CaptureDeviceRepository
	Pairings        repositories.CapturePairingRepository
	Profiles        repositories.CaptureProfileRepository
	Requests        repositories.CaptureRequestRepository
	Batches         repositories.CaptureBatchRepository
	Pages           repositories.CapturePageRepository
	Items           repositories.CaptureItemRepository
	CoverSheets     repositories.CaptureCoverSheetRepository
	Records         repositories.CaptureRecordFinder
	DocumentControl repositories.DocumentControlRepository
	DocumentTypes   repositories.DocumentTypeRepository
	Permissions     services.PermissionEngine
	Storage         storage.Client
	Encryption      *encryptionservice.Service
	Uploads         services.DocumentUploadService
	Assembler       services.CapturePDFAssembler
	Inspector       services.CapturePageInspector
	QRCodes         services.CaptureQREncoder
	// Workflows is optional so an installation without a worker still
	// receives pages. A sealed batch then waits, and the reconcile sweep
	// starts its processing once a worker appears.
	Workflows services.WorkflowStarter `optional:"true"`
	// Realtime keeps the intake queue and the device's stream current. Without
	// it both are right on their next load, and the device polls.
	Realtime services.RealtimeService `optional:"true"`
	Audit    services.AuditService    `optional:"true"`
	// Analyzer suggests what a document is and which shipment it belongs to.
	// Without it every item waits for a person to say.
	Analyzer services.CaptureAnalyzer `optional:"true"`
	// Shipments resolves a reference number the analyzer found to a shipment.
	Shipments repositories.InboundShipmentFinder `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	cfg           *config.Config
	db            ports.DBConnection
	devices       repositories.CaptureDeviceRepository
	pairings      repositories.CapturePairingRepository
	profiles      repositories.CaptureProfileRepository
	requests      repositories.CaptureRequestRepository
	batches       repositories.CaptureBatchRepository
	pages         repositories.CapturePageRepository
	items         repositories.CaptureItemRepository
	coverSheets   repositories.CaptureCoverSheetRepository
	records       repositories.CaptureRecordFinder
	controls      repositories.DocumentControlRepository
	documentTypes repositories.DocumentTypeRepository
	permissions   services.PermissionEngine
	storage       storage.Client
	cipher        envelopeCipher
	uploads       services.DocumentUploadService
	assembler     services.CapturePDFAssembler
	inspector     services.CapturePageInspector
	qrCodes       services.CaptureQREncoder
	workflows     services.WorkflowStarter
	realtime      services.RealtimeService
	audit         services.AuditService
	analyzer      services.CaptureAnalyzer
	shipments     repositories.InboundShipmentFinder
}

//nolint:gocritic // dependency injection param
func New(p Params) *Service {
	var cipher envelopeCipher
	if p.Encryption != nil {
		cipher = p.Encryption
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
	}
}

// control is the tenant's document settings. A tenant that never saved them
// gets the defaults, which is what it would have been given on creation.
func (s *Service) control(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.DocumentControl, error) {
	control, err := s.controls.Get(ctx, repositories.GetDocumentControlRequest{
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
) (*services.PermissionCheckResult, error) {
	if tenantInfo.UserID.IsNil() {
		return &services.PermissionCheckResult{Allowed: false, Reason: "no_user"}, nil
	}

	return s.permissions.Check(ctx, services.UserActor(tenantInfo).PermissionCheck(resource, op))
}

// require is allowed, turned into the error a handler maps to 403.
func (s *Service) require(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource permission.Resource,
	op permission.Operation,
) (*services.PermissionCheckResult, error) {
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
func seesEveryone(result *services.PermissionCheckResult) bool {
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
		&services.PublishResourceInvalidationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			AudienceUserID: audience,
			Resource:       resource.String(),
			Action:         action,
			RecordID:       recordID,
			Entity:         entity,
			ActorUserID:    tenantInfo.UserID,
			ActorType:      services.PrincipalTypeUser,
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
func (s *Service) logAudit(params *services.LogActionParams, comment string) {
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
