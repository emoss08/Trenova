package settlementreleaseports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultBlocked   = 25
	MaxBlocked       = 100
	MaxSettlements   = 120
	MaxWeeks         = 12
	MaxBatches       = 20
	BatchesPerPeriod = 4
)

type Tally struct {
	Count    int64
	NetMinor int64
}

type Batch struct {
	Name                string
	Batches             int
	PeriodStart         int64
	PeriodEnd           int64
	PayDate             int64
	Currency            string
	Currencies          int64
	Settlements         int64
	GrossMinor          int64
	ReimbursementsMinor int64
	NetMinor            int64
	Released            Tally
	Ready               Tally
	Blocked             map[driversettlement.ReleaseBlock]Tally
}

func (b *Batch) EarnedMinor() int64 {
	return b.GrossMinor + b.ReimbursementsMinor
}

func (b *Batch) DeductedMinor() int64 {
	return max(b.EarnedMinor()-b.NetMinor, 0)
}

func (b *Batch) BlockedTotal() Tally {
	var total Tally
	for _, tally := range b.Blocked {
		total.Count += tally.Count
		total.NetMinor += tally.NetMinor
	}

	return total
}

type BatchRequest struct {
	BatchID pulid.ID
}

type WeeksRequest struct {
	Weeks int
}

type Week struct {
	PeriodStart         int64
	PeriodEnd           int64
	PayDate             int64
	Batches             int64
	Currency            string
	Currencies          int64
	GrossMinor          int64
	ReimbursementsMinor int64
	NetMinor            int64
}

func (w *Week) EarnedMinor() int64 {
	return w.GrossMinor + w.ReimbursementsMinor
}

func (w *Week) DeductedMinor() int64 {
	return max(w.EarnedMinor()-w.NetMinor, 0)
}

type Settlement struct {
	ID                   pulid.ID
	WorkerID             pulid.ID
	BatchID              *pulid.ID
	BatchName            string
	SettlementNumber     string
	Driver               string
	Status               driversettlement.Status
	PayDate              int64
	NetPayMinor          int64
	GrossEarningsMinor   int64
	DeductionsMinor      int64
	CarryForwardOutMinor int64
	Currency             string
	Exceptions           []driversettlement.Exception
	HoldReason           driversettlement.HoldReason
	HoldNote             string
	DisputeCategory      driversettlement.DisputeCategory
	Block                driversettlement.ReleaseBlock
}

func (s *Settlement) Has(code driversettlement.ExceptionCode) bool {
	for _, exception := range s.Exceptions {
		if exception.Code == code {
			return true
		}
	}

	return false
}

func (s *Settlement) CriticalException() (driversettlement.Exception, bool) {
	for _, exception := range s.Exceptions {
		if exception.Severity == driversettlement.ExceptionSeverityCritical {
			return exception, true
		}
	}

	return driversettlement.Exception{}, false
}

type SettlementsRequest struct {
	IDs      []pulid.ID
	Disputes bool
}

type BlockedRequest struct {
	BatchID pulid.ID
	Limit   int
}

type Reader interface {
	Batch(ctx context.Context, req *BatchRequest) (*Batch, error)
	Weeks(ctx context.Context, req *WeeksRequest) ([]*Week, error)
	Settlements(ctx context.Context, req *SettlementsRequest) ([]*Settlement, error)
	Blocked(ctx context.Context, req *BlockedRequest) ([]*Settlement, error)
}

func Day(at int64, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	moment := time.Unix(at, 0).Add(12 * time.Hour).In(loc)

	return time.Date(moment.Year(), moment.Month(), moment.Day(), 12, 0, 0, 0, time.UTC)
}
