package weekhoursports

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrNoTeam = errors.New(
	"the person's access reaches only their own team, and they answer for no one",
)

const (
	MaxWorkers = 120
	MaxLeading = 25
)

type WorkerWeek struct {
	WorkerID pulid.ID
	Name     string
	Status   string
	Week     weekhours.Week
}

type WorkersRequest struct {
	Tenant     pagination.TenantInfo
	WorkerIDs  []pulid.ID
	ManagerIDs []pulid.ID
	Now        int64
	Location   *time.Location
}

type LeadingRequest struct {
	Tenant     pagination.TenantInfo
	ManagerIDs []pulid.ID
	Now        int64
	Location   *time.Location
	Limit      int
	Room       int
}

type Leading struct {
	Start  int64
	End    int64
	Rows   []*WorkerWeek
	Room   []*WorkerWeek
	OnRota int64
	Over   int64
	OnLine int64
	Exact  bool
}

type Reader interface {
	Workers(ctx context.Context, req *WorkersRequest) (map[pulid.ID]*WorkerWeek, error)
	Leading(ctx context.Context, req *LeadingRequest) (*Leading, error)
}

type TeamNarrower interface {
	Narrow(
		ctx context.Context,
		actor *permissionports.RequestActor,
		tenant pagination.TenantInfo,
	) ([]pulid.ID, error)
}
