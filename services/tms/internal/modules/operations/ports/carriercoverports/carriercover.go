package carriercoverports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxCarriers           = 100
	DefaultCarriers       = 25
	MaxIDs                = 120
	MaxWithinDays         = 90
	LapsedLookbackSeconds = int64(90 * 24 * 60 * 60)
	CandidateCarriers     = 4 * MaxCarriers
)

type Cover struct {
	CarrierID        pulid.ID
	Name             string
	Code             string
	Status           carrier.Status
	ComplianceStatus carrier.ComplianceStatus
	Policy           *carrier.CarrierInsurancePolicy
	Missing          []carrier.InsurancePolicyType
	LoadsRead        bool
	OpenLoads        int64
	ReadableLoads    int64
}

func (c *Cover) LoadsReadable() bool {
	return c.LoadsRead && c.OpenLoads == c.ReadableLoads
}

func (c *Cover) EndsAt() int64 {
	if c.Policy == nil {
		return 0
	}

	return c.Policy.ExpirationDate
}

type CoversRequest struct {
	IDs   []pulid.ID
	Loads bool
}

type EndingRequest struct {
	Now   int64
	Until int64
	Limit int
	Loads bool
}

type Load struct {
	CarrierID  pulid.ID
	ShipmentID pulid.ID
	ProNumber  string
	DeliversAt int64
}

type LoadsRequest struct {
	CarrierIDs []pulid.ID
	PerCarrier int
}

type Reader interface {
	Covers(ctx context.Context, req *CoversRequest) (map[pulid.ID]*Cover, error)
	Ending(ctx context.Context, req *EndingRequest) ([]*Cover, error)
	Loads(ctx context.Context, req *LoadsRequest) (map[pulid.ID][]Load, error)
}
