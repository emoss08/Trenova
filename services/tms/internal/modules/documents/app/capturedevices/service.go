package capturedevices

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureprofiles"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	AppConfig     *appconfig.Config
	DB            kernelports.Transactor
	DeviceRepo    captureports.CaptureDeviceRepository
	Logger        *zap.Logger
	Organizations tenantports.OrganizationRepository
	Pairings      captureports.CapturePairingRepository
	Profiles      *captureprofiles.Service
	Support       *capturesupport.Service
	Users         tenantports.UserRepository
}

type Service struct {
	cfg           *appconfig.Config
	db            kernelports.Transactor
	deviceRepo    captureports.CaptureDeviceRepository
	l             *zap.Logger
	organizations organizationReader
	pairings      captureports.CapturePairingRepository
	users         userReader
	profiles      *captureprofiles.Service
	support       *capturesupport.Service
}

func New(p Params) *Service {
	return &Service{
		cfg:           p.AppConfig,
		db:            p.DB,
		deviceRepo:    p.DeviceRepo,
		l:             p.Logger.Named("service.capture"),
		organizations: p.Organizations,
		pairings:      p.Pairings,
		users:         p.Users,
		profiles:      p.Profiles,
		support:       p.Support,
	}
}

var ErrNoWebAddress = errors.New(
	"this Trenova server has no web address configured (app.webBaseUrl), so a computer cannot be approved",
)

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
