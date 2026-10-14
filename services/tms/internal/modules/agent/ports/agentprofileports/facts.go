package agentprofileports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// OrganizationWindowDays is how far back an organization's work is read:
	// long enough to see its regular customers and lanes through a slow week,
	// short enough that a customer it lost last year is gone.
	OrganizationWindowDays = 90
	// UserWindowDays is how far back a person's work is read.
	UserWindowDays = 30
	// TopRows is how many customers, lanes, locations or kinds of work a fact
	// names; the rest are counted, not named.
	TopRows = 5
	// MaxActiveUsers bounds how many people's profiles one organization's
	// nightly pass works out.
	MaxActiveUsers = 500
)

type FactsWindow struct {
	TenantInfo pagination.TenantInfo
	Start      int64
	End        int64
}

type NamedCount struct {
	Name  string `bun:"name"`
	Count int64  `bun:"count"`
}

type LaneCount struct {
	Origin      string `bun:"origin_state"`
	Destination string `bun:"destination_state"`
	Count       int64  `bun:"count"`
}

type NamedStatus struct {
	Name   string `bun:"name"`
	Status string `bun:"status"`
}

type ShipmentVolume struct {
	Total     int64 `bun:"total"`
	Canceled  int64 `bun:"canceled"`
	Moves     int64 `bun:"moves"`
	Brokered  int64 `bun:"brokered"`
	Customers int64 `bun:"customers"`
}

type FleetCounts struct {
	Active       int64        `bun:"active"`
	OutOfService int64        `bun:"out_of_service"`
	ByType       []NamedCount `bun:"-"`
}

type WorkforceCounts struct {
	Active       int64        `bun:"active"`
	Contractors  int64        `bun:"contractors"`
	ByDriverType []NamedCount `bun:"-"`
}

// OrganizationActivity is what an organization's records say about the work
// it does over a window, every figure a count or a share of the records.
type OrganizationActivity struct {
	Volume                ShipmentVolume
	TopCustomers          []NamedCount
	TopLanes              []LaneCount
	TopPickupLocations    []NamedCount
	Tractors              FleetCounts
	Trailers              FleetCounts
	Workers               WorkforceCounts
	CarriersUsed          int64
	Integrations          []string
	AccountingConnections []NamedStatus
	EDIPartners           int64
}

type UserFactsRequest struct {
	Window FactsWindow
	UserID pulid.ID
}

// UserActivity is what the records say a person works on over a window.
type UserActivity struct {
	JobTitle          string
	Driver            bool
	OwnedShipments    []NamedCount
	EnteredShipments  int64
	Customers         []NamedCount
	ChangedRecords    []NamedCount
	ConversationCount int64
	ConversationAbout []NamedCount
	ToolsUsed         []NamedCount
	FavoritePages     []string
}

type ActiveUsersRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	Limit      int
}

// FactsReader works out profile facts from the records. Every read is
// tenant-scoped on its driving table and on each join.
type FactsReader interface {
	OrganizationActivity(ctx context.Context, window FactsWindow) (*OrganizationActivity, error)
	UserActivity(ctx context.Context, req *UserFactsRequest) (*UserActivity, error)
	// ActiveUsers are the people who signed in or talked to an agent since the
	// given time, most recent first.
	ActiveUsers(ctx context.Context, req ActiveUsersRequest) ([]pulid.ID, error)
}
