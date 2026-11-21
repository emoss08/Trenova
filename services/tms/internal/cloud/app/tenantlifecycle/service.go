package tenantlifecycle

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	applyLockTimeout = 10 * time.Second
	controlPlane     = "trenova_control_plane"
	auditSource      = "shared_tenant_lifecycle"
)

var ErrNotShared = errors.New(
	"tenants are managed only on a shared server (managed.tenancy: shared)",
)

type Params struct {
	fx.In

	Config         *managedconfig.Config
	DB             kernelports.Transactor
	Tenants        cloudtenantports.Repository
	Administrators cloudtenantports.Administrators
	Members        cloudtenantports.Members
	Bootstrap      tenantports.TenantBootstrapRepository
	Features       managedfeaturesports.Writer
	Usage          managedusageports.Writer
	Readings       managedusageports.OrganizationReadings
	Invites        instancebootstrapports.FirstAdministratorInviteService
	Sessions       sessionports.SessionRepository
	Auditor        permissionports.SecurityAuditor
	Workflows      kernelports.WorkflowStarter
	Commands       cloudtenantports.Commands
	Secrets        kernelports.SecretCipher
	Scopes         *managedscope.Resolver
	Clock          timeutils.Clock
	Logger         *zap.Logger
}

type Service struct {
	shared         bool
	db             kernelports.Transactor
	tenants        cloudtenantports.Repository
	administrators cloudtenantports.Administrators
	members        cloudtenantports.Members
	bootstrap      tenantports.TenantBootstrapRepository
	features       managedfeaturesports.Writer
	usage          managedusageports.Writer
	readings       managedusageports.OrganizationReadings
	invites        instancebootstrapports.FirstAdministratorInviteService
	sessions       sessionports.SessionRepository
	auditor        permissionports.SecurityAuditor
	workflows      kernelports.WorkflowStarter
	commands       cloudtenantports.Commands
	secrets        kernelports.SecretCipher
	scopes         *managedscope.Resolver
	clock          timeutils.Clock
	l              *zap.Logger
}

//nolint:gocritic // this is dependency injection
func New(p Params) *Service {
	return &Service{
		shared:         p.Config.SharedTenancy(),
		db:             p.DB,
		tenants:        p.Tenants,
		administrators: p.Administrators,
		members:        p.Members,
		bootstrap:      p.Bootstrap,
		features:       p.Features,
		usage:          p.Usage,
		readings:       p.Readings,
		invites:        p.Invites,
		sessions:       p.Sessions,
		auditor:        p.Auditor,
		workflows:      p.Workflows,
		commands:       p.Commands,
		secrets:        p.Secrets,
		scopes:         p.Scopes,
		clock:          p.Clock,
		l:              p.Logger.Named("cloud.tenant-lifecycle"),
	}
}

func (s *Service) Shared() bool {
	return s.shared
}
