package identitytest

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"go.uber.org/zap"
)

type UserServiceDeps struct {
	Users   tenantports.UserRepository
	Audit   auditports.AuditService
	Roles   userservice.RoleAssigner
	Invites userservice.AccountInviter
}

func NewUserServiceForTest(deps UserServiceDeps) *userservice.Service {
	return userservice.New(userservice.Params{
		Logger:        zap.NewNop(),
		Repo:          deps.Users,
		AuditService:  deps.Audit,
		Auditor:       &securityaudittest.Recorder{},
		Realtime:      &mocks.NoopRealtimeService{},
		StorageConfig: &storageconfig.Config{},
		Validator:     userservice.NewTestValidatorWithRepository(deps.Users),
		DB:            dbtest.NopConnection{},
		Roles:         deps.Roles,
		Invites:       deps.Invites,
	})
}
