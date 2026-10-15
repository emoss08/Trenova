package factoring

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

type AccountRole string

const (
	AccountRoleAR            = AccountRole("AccountsReceivable")
	AccountRoleCash          = AccountRole("Cash")
	AccountRoleDueFromFactor = AccountRole("DueFromFactor")
	AccountRoleFeeExpense    = AccountRole("FeeExpense")
	AccountRoleLiability     = AccountRole("Liability")
)

type InvoiceEffect string

const (
	InvoiceEffectNone   = InvoiceEffect("")
	InvoiceEffectSettle = InvoiceEffect("Settle")
	InvoiceEffectReopen = InvoiceEffect("Reopen")
)

type MovementLine struct {
	Role   AccountRole `json:"role"`
	Debit  int64       `json:"debit"`
	Credit int64       `json:"credit"`
}

type Movement struct {
	Kind           ReserveEntryKind    `json:"kind"`
	Treatment      AccountingTreatment `json:"treatment"`
	Lines          []MovementLine      `json:"lines"`
	CashMinor      int64               `json:"cashMinor"`
	FeeMinor       int64               `json:"feeMinor"`
	ReserveMinor   int64               `json:"reserveMinor"`
	LiabilityMinor int64               `json:"liabilityMinor"`
	ARMinor        int64               `json:"arMinor"`
	InvoiceEffect  InvoiceEffect       `json:"invoiceEffect"`
	InvoiceMinor   int64               `json:"invoiceMinor"`
}

type deltas struct {
	cash      int64
	fee       int64
	reserve   int64
	liability int64
	ar        int64
}

func (d *deltas) balanceWithFee() {
	d.fee = -(d.cash + d.reserve + d.ar - d.liability)
}

func PlanFundingMovement(item *SubmissionItem, treatment AccountingTreatment) (Movement, error) {
	if err := requirePlannable(item, treatment); err != nil {
		return Movement{}, err
	}
	advance, fee, reserve := item.Actuals()
	d := deltas{cash: advance, fee: fee}
	movement := Movement{Kind: ReserveEntryFunding, Treatment: treatment}
	switch treatment {
	case AccountingTreatmentSecuredBorrowing:
		d.liability = advance + fee
	case AccountingTreatmentSale:
		d.reserve = reserve
		d.ar = -item.FaceMinor
		movement.InvoiceEffect = InvoiceEffectSettle
		movement.InvoiceMinor = item.FaceMinor
	}
	return finish(&movement, d)
}

func PlanReleaseMovement(
	item *SubmissionItem,
	treatment AccountingTreatment,
	cashMinor int64,
) (Movement, error) {
	if err := requirePlannable(item, treatment); err != nil {
		return Movement{}, err
	}
	if cashMinor < 0 {
		return Movement{}, errortypes.NewValidationError(
			"cashMinor",
			errortypes.ErrInvalid,
			"The amount the factor paid cannot be negative",
		)
	}
	advance, fee, reserve := item.Actuals()
	d := deltas{cash: cashMinor}
	movement := Movement{Kind: ReserveEntryRelease, Treatment: treatment}
	switch treatment {
	case AccountingTreatmentSecuredBorrowing:
		d.liability = -(advance + fee)
		d.ar = -item.FaceMinor
		movement.InvoiceEffect = InvoiceEffectSettle
		movement.InvoiceMinor = item.FaceMinor
	case AccountingTreatmentSale:
		d.reserve = -reserve
	}
	d.balanceWithFee()
	return finish(&movement, d)
}

func PlanChargebackMovement(
	item *SubmissionItem,
	treatment AccountingTreatment,
	cashMinor int64,
) (Movement, error) {
	if err := requirePlannable(item, treatment); err != nil {
		return Movement{}, err
	}
	if cashMinor < 0 {
		return Movement{}, errortypes.NewValidationError(
			"cashMinor",
			errortypes.ErrInvalid,
			"The amount paid back to the factor cannot be negative",
		)
	}
	advance, fee, reserve := item.Actuals()
	d := deltas{cash: -cashMinor}
	movement := Movement{Kind: ReserveEntryChargeback, Treatment: treatment}
	switch treatment {
	case AccountingTreatmentSecuredBorrowing:
		d.liability = -(advance + fee)
	case AccountingTreatmentSale:
		d.reserve = -reserve
		d.ar = item.FaceMinor
		movement.InvoiceEffect = InvoiceEffectReopen
		movement.InvoiceMinor = item.FaceMinor
	}
	d.balanceWithFee()
	return finish(&movement, d)
}

func ChargebackRepurchaseMinor(item *SubmissionItem) int64 {
	advance, fee, _ := item.Actuals()
	return advance + fee
}

func ReverseMovement(original *Movement) Movement {
	reversed := Movement{
		Kind:           ReserveEntryChargebackReversal,
		Treatment:      original.Treatment,
		Lines:          make([]MovementLine, 0, len(original.Lines)),
		CashMinor:      -original.CashMinor,
		FeeMinor:       -original.FeeMinor,
		ReserveMinor:   -original.ReserveMinor,
		LiabilityMinor: -original.LiabilityMinor,
		ARMinor:        -original.ARMinor,
		InvoiceMinor:   original.InvoiceMinor,
	}
	for _, line := range original.Lines {
		reversed.Lines = append(reversed.Lines, MovementLine{
			Role:   line.Role,
			Debit:  line.Credit,
			Credit: line.Debit,
		})
	}
	switch original.InvoiceEffect {
	case InvoiceEffectSettle:
		reversed.InvoiceEffect = InvoiceEffectReopen
	case InvoiceEffectReopen:
		reversed.InvoiceEffect = InvoiceEffectSettle
	case InvoiceEffectNone:
	}
	return reversed
}

func (m *Movement) Balanced() bool {
	var debit, credit int64
	for _, line := range m.Lines {
		debit += line.Debit
		credit += line.Credit
	}
	return debit == credit && debit > 0
}

func (m *Movement) IsEmpty() bool {
	return len(m.Lines) == 0
}

func (m *Movement) Total() int64 {
	var debit int64
	for _, line := range m.Lines {
		debit += line.Debit
	}
	return debit
}

func requirePlannable(item *SubmissionItem, treatment AccountingTreatment) error {
	if !treatment.IsValid() {
		return errortypes.NewBusinessError(
			"The agreement's accounting treatment is not set; choose sale or secured borrowing",
		)
	}
	if item == nil || item.AdvanceMinor == nil || item.FeeMinor == nil ||
		item.ReserveMinor == nil {
		return errortypes.NewBusinessError("The invoice has not been funded")
	}
	return nil
}

func finish(movement *Movement, d deltas) (Movement, error) {
	movement.CashMinor = d.cash
	movement.FeeMinor = d.fee
	movement.ReserveMinor = d.reserve
	movement.LiabilityMinor = d.liability
	movement.ARMinor = d.ar
	movement.Lines = make([]MovementLine, 0, 5)
	movement.Lines = appendSigned(movement.Lines, AccountRoleCash, d.cash)
	movement.Lines = appendSigned(movement.Lines, AccountRoleFeeExpense, d.fee)
	movement.Lines = appendSigned(movement.Lines, AccountRoleDueFromFactor, d.reserve)
	movement.Lines = appendSigned(movement.Lines, AccountRoleLiability, -d.liability)
	movement.Lines = appendSigned(movement.Lines, AccountRoleAR, d.ar)
	if movement.IsEmpty() {
		return *movement, nil
	}
	if !movement.Balanced() {
		return Movement{}, errortypes.NewBusinessError(
			"The factoring entry does not balance; check the amounts",
		)
	}
	return *movement, nil
}

func appendSigned(lines []MovementLine, role AccountRole, amount int64) []MovementLine {
	switch {
	case amount > 0:
		return append(lines, MovementLine{Role: role, Debit: amount})
	case amount < 0:
		return append(lines, MovementLine{Role: role, Credit: -amount})
	default:
		return lines
	}
}
