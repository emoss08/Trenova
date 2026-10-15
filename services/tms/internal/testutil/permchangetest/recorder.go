package permchangetest

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ permissionports.PermissionChangeNotifier = (*Recorder)(nil)

type UserChange struct {
	Tenant pagination.TenantInfo
	UserID pulid.ID
}

type Recorder struct {
	mu            sync.Mutex
	organizations []pagination.TenantInfo
	users         []UserChange
}

func (r *Recorder) OrganizationChanged(_ context.Context, tenant pagination.TenantInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.organizations = append(r.organizations, tenant)
}

func (r *Recorder) UserChanged(_ context.Context, tenant pagination.TenantInfo, userID pulid.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users = append(r.users, UserChange{Tenant: tenant, UserID: userID})
}

func (r *Recorder) Organizations() []pagination.TenantInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pagination.TenantInfo(nil), r.organizations...)
}

func (r *Recorder) Users() []UserChange {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]UserChange(nil), r.users...)
}
