package auditports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type OverviewWindow struct {
	From int64
	To   int64
}

type OverviewRequest struct {
	TenantInfo     pagination.TenantInfo
	Access         EntryAccess
	Current        OverviewWindow
	Previous       OverviewWindow
	Hourly         bool
	ActorLimit     int
	IncludeReviews bool
	ReviewLimit    int
}

type OverviewFigures struct {
	Entries        int64
	People         int64
	AgentsAsked    int64
	AgentsOwn      int64
	Integrations   int64
	Jobs           int64
	Sensitive      int64
	Security       int64
	DeletesExports int64
	PeopleOffHours int64
}

func (f OverviewFigures) Agents() int64 { return f.AgentsAsked + f.AgentsOwn }

type ActivityBucket struct {
	At           time.Time
	People       int64
	Agents       int64
	Integrations int64
	Jobs         int64
}

type ActorStat struct {
	PrincipalID   pulid.ID
	PrincipalType string
	Name          string
	Entries       int64
	Sensitive     int64
}

type ActorCounts struct {
	People       int
	Agents       int
	Integrations int
	Jobs         int
}

func (c ActorCounts) Total() int { return c.People + c.Agents + c.Integrations + c.Jobs }

type OverviewRollup struct {
	Totals          OverviewFigures
	PreviousEntries int64
	Activity        []ActivityBucket
	Actors          ActorCounts
	TopActors       []ActorStat
	OpenReviews     []*audit.Review
	OpenReviewCount int
}

type OverviewRepository interface {
	OrganizationTimezone(ctx context.Context, tenant pagination.TenantInfo) (string, error)
	Overview(ctx context.Context, req *OverviewRequest) (*OverviewRollup, error)
}
