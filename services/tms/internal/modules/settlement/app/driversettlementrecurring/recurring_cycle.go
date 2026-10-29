package driversettlementrecurring

import (
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CycleRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	PeriodEnd  int64
}

type Params struct {
	SettlementRepo driversettlementports.DriverSettlementRepository
	DeductionRepo  driverpayports.RecurringDeductionRepository
	EarningRepo    driverpayports.RecurringEarningRepository
	AdvanceRepo    driverpayports.PayAdvanceRepository
	EscrowRepo     driverpayports.EscrowAccountRepository
}

type Cycle struct {
	settlementRepo driversettlementports.DriverSettlementRepository
	deductionRepo  driverpayports.RecurringDeductionRepository
	earningRepo    driverpayports.RecurringEarningRepository
	advanceRepo    driverpayports.PayAdvanceRepository
	escrowRepo     driverpayports.EscrowAccountRepository
}

func New(p *Params) *Cycle {
	return &Cycle{
		settlementRepo: p.SettlementRepo,
		deductionRepo:  p.DeductionRepo,
		earningRepo:    p.EarningRepo,
		advanceRepo:    p.AdvanceRepo,
		escrowRepo:     p.EscrowRepo,
	}
}
