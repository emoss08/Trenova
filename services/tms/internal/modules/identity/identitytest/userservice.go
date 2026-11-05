package identitytest

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountguard"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/internal/testutil/sessiontest"
	"go.uber.org/zap"
)

type UserServiceDeps struct {
	Users   tenantports.UserRepository
	Audit   auditports.AuditService
	Roles   userservice.RoleAssigner
	Invites userservice.AccountInviter
	Held    permissionports.PermissionEngine
}

func NewUserServiceForTest(deps UserServiceDeps) *userservice.Service {
	held := deps.Held
	if held == nil {
		held = &permtest.Engine{}
	}

	return userservice.New(userservice.Params{
		Logger:         zap.NewNop(),
		Repo:           deps.Users,
		AuditService:   deps.Audit,
		Auditor:        &securityaudittest.Recorder{},
		Realtime:       &mocks.NoopRealtimeService{},
		StorageConfig:  &storageconfig.Config{},
		Validator:      userservice.NewValidator(userservice.ValidatorParams{Repo: deps.Users}),
		DB:             dbtest.NopConnection{},
		Roles:          deps.Roles,
		Invites:        deps.Invites,
		PasswordStepUp: sessiontest.OpenStepUp{},
		Accounts: accountguard.New(accountguard.Params{
			Users:       deps.Users,
			Permissions: held,
			Auditor:     &securityaudittest.Recorder{},
		}),
	})
}
