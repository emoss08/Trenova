package capturesupport

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Audit           auditports.AuditService `optional:"true"`
	DocumentControl tenantports.DocumentControlRepository
	Encryption      *encryptionservice.Service
	Logger          *zap.Logger
	Permissions     permissionports.PermissionEngine
	// Realtime keeps the intake queue and the device's stream current. Without
	// it both are right on their next load, and the device polls.
	Realtime realtimeports.RealtimeService `optional:"true"`
	Storage  kernelports.Client
}

type Service struct {
	audit       auditports.AuditService
	cipher      *encryptionservice.Service
	controls    tenantports.DocumentControlRepository
	l           *zap.Logger
	permissions permissionports.PermissionEngine
	realtime    realtimeports.RealtimeService
	storage     kernelports.Client
}

func New(p Params) *Service {
	return &Service{
		audit:       p.Audit,
		cipher:      p.Encryption,
		controls:    p.DocumentControl,
		l:           p.Logger.Named("service.capture"),
		permissions: p.Permissions,
		realtime:    p.Realtime,
		storage:     p.Storage,
	}
}
