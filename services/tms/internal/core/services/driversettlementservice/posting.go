package driversettlementservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/journalposting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

func (s *Service) Post(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Settlement posting"); err != nil {
		return nil, err
	}

	var updated *driversettlement.Settlement
	var previous driversettlement.Settlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if entity.Status != driversettlement.StatusApproved {
			return transitionError(entity.Status, driversettlement.StatusPosted)
		}
		previous = *entity

		batchID, txErr := s.postSettlementJournal(txCtx, entity, actor, false)
		if txErr != nil {
			return txErr
		}

		now := timeutils.NowUnix()
		entity.Status = driversettlement.StatusPosted
		entity.PostedByID = actor.UserID
		entity.PostedAt = &now
		entity.PostedJournalBatchID = batchID
		if updated, txErr = s.settlementRepo.Update(txCtx, entity); txErr != nil {
			return txErr
		}
		return s.queueSync(
			txCtx,
			updated,
			accountingsync.SyncObjectDriverBill,
			accountingsync.SyncOperationCreate,
			accountingsync.SyncSourceDriverSettlementPosted,
		)
	})
	if err != nil {
		return nil, err
	}
	s.logSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpApprove,
		"Settlement posted to general ledger")
	if s.driverNotify != nil {
		s.driverNotify.Notify(ctx, &drivernotificationservice.DriverNotification{
			TenantInfo: tenantInfo,
			WorkerID:   updated.WorkerID,
			EventType:  "dash.settlement_posted",
			Priority:   notification.PriorityHigh,
			Context: documenttemplate.SettlementNotificationContext{
				SettlementNumber: updated.SettlementNumber,
				Amount: money.FormatMinor(
					updated.NetPayMinor, updated.CurrencyCode,
				),
				Currency: updated.CurrencyCode,
			},
			Link: "/dash/pay/" + updated.ID.String(),
			RelatedEntities: map[string]any{
				"settlementId": updated.ID.String(),
			},
		})
	}
	return updated, nil
}

func (s *Service) postVoidReversal(
	ctx context.Context,
	entity *driversettlement.Settlement,
	actor *serviceports.RequestActor,
) error {
	batchID, err := s.postSettlementJournal(ctx, entity, actor, true)
	if err != nil {
		return err
	}
	entity.VoidJournalBatchID = batchID
	return nil
}

type settlementPostingTotals struct {
	Expense         int64
	Reimbursements  int64
	NetPayable      int64
	AdvanceRecovery int64
	EscrowWithheld  int64
	OtherDeductions int64
	CarryForwardIn  int64
	CarryForwardOut int64
}

type PostingAccounts struct {
	Expense       pulid.ID
	Reimbursement pulid.ID
	Payable       pulid.ID
	Advance       pulid.ID
	Escrow        pulid.ID
}

type PostingLeg struct {
	AccountID pulid.ID
	Debit     int64
	Credit    int64
}

func BuildSettlementPostingLegs(
	entity *driversettlement.Settlement,
	accounts *PostingAccounts,
	codeAccounts map[pulid.ID]pulid.ID,
) []PostingLeg {
	totals, codeDebits, codeCredits := settlementTotals(entity, codeAccounts)
	reimbursementAccountID := accounts.Reimbursement
	if reimbursementAccountID.IsNil() {
		reimbursementAccountID = accounts.Expense
	}

	legs := make([]PostingLeg, 0, 6+len(codeDebits)+len(codeCredits))
	legs = append(legs, codeLegs(codeDebits, codeCredits)...)
	if totals.Expense > 0 {
		legs = append(legs, PostingLeg{AccountID: accounts.Expense, Debit: totals.Expense})
	}
	if totals.Reimbursements > 0 {
		legs = append(legs, PostingLeg{
			AccountID: reimbursementAccountID,
			Debit:     totals.Reimbursements,
		})
	}
	if totals.CarryForwardOut > 0 {
		legs = append(legs, PostingLeg{
			AccountID: accounts.Advance,
			Debit:     totals.CarryForwardOut,
		})
	}
	if totals.NetPayable > 0 {
		legs = append(legs, PostingLeg{AccountID: accounts.Payable, Credit: totals.NetPayable})
	}
	if totals.AdvanceRecovery > 0 {
		legs = append(legs, PostingLeg{
			AccountID: accounts.Advance,
			Credit:    totals.AdvanceRecovery,
		})
	}
	if totals.CarryForwardIn > 0 {
		legs = append(legs, PostingLeg{
			AccountID: accounts.Advance,
			Credit:    totals.CarryForwardIn,
		})
	}
	if totals.EscrowWithheld > 0 {
		legs = append(legs, PostingLeg{
			AccountID: accounts.Escrow,
			Credit:    totals.EscrowWithheld,
		})
	}
	if totals.OtherDeductions > 0 {
		legs = append(legs, PostingLeg{
			AccountID: accounts.Expense,
			Credit:    totals.OtherDeductions,
		})
	}
	return legs
}

//nolint:cyclop // enumerates every settlement line category with code-mapped branches
func settlementTotals(
	entity *driversettlement.Settlement,
	codeAccounts map[pulid.ID]pulid.ID,
) (totals settlementPostingTotals, codeDebits, codeCredits map[pulid.ID]int64) {
	totals = settlementPostingTotals{
		NetPayable:      entity.NetPayMinor,
		CarryForwardIn:  -entity.CarryForwardInMinor,
		CarryForwardOut: -entity.CarryForwardOutMinor,
	}
	codeDebits = make(map[pulid.ID]int64)
	codeCredits = make(map[pulid.ID]int64)

	codeAccount := func(line *driversettlement.SettlementLine) (pulid.ID, bool) {
		if line.PayCodeID == nil || line.PayCodeID.IsNil() {
			return pulid.Nil, false
		}
		accountID, ok := codeAccounts[*line.PayCodeID]
		return accountID, ok && !accountID.IsNil()
	}

	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		switch line.Category {
		case driversettlement.LineCategoryEarning,
			driversettlement.LineCategoryGuaranteeTopUp:
			if accountID, ok := codeAccount(line); ok {
				codeDebits[accountID] += line.AmountMinor
			} else {
				totals.Expense += line.AmountMinor
			}
		case driversettlement.LineCategoryReimbursement:
			if accountID, ok := codeAccount(line); ok {
				codeDebits[accountID] += line.AmountMinor
			} else {
				totals.Reimbursements += line.AmountMinor
			}
		case driversettlement.LineCategoryAdvanceRecovery:
			totals.AdvanceRecovery += -line.AmountMinor
		case driversettlement.LineCategoryEscrowContribution:
			totals.EscrowWithheld += -line.AmountMinor
		case driversettlement.LineCategoryDeduction:
			if accountID, ok := codeAccount(line); ok {
				codeCredits[accountID] += -line.AmountMinor
			} else {
				totals.OtherDeductions += -line.AmountMinor
			}
		case driversettlement.LineCategoryAdjustment:
			accountID, mapped := codeAccount(line)
			switch {
			case line.AmountMinor >= 0 && mapped:
				codeDebits[accountID] += line.AmountMinor
			case line.AmountMinor >= 0:
				totals.Expense += line.AmountMinor
			case mapped:
				codeCredits[accountID] += -line.AmountMinor
			default:
				totals.OtherDeductions += -line.AmountMinor
			}
		case driversettlement.LineCategoryCarryForward:
		}
	}
	return totals, codeDebits, codeCredits
}

func codeLegs(codeDebits, codeCredits map[pulid.ID]int64) []PostingLeg {
	accountIDs := make([]pulid.ID, 0, len(codeDebits)+len(codeCredits))
	for accountID := range codeDebits {
		accountIDs = append(accountIDs, accountID)
	}
	for accountID := range codeCredits {
		if _, seen := codeDebits[accountID]; !seen {
			accountIDs = append(accountIDs, accountID)
		}
	}
	slices.SortFunc(accountIDs, func(a, b pulid.ID) int {
		return strings.Compare(a.String(), b.String())
	})

	legs := make([]PostingLeg, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		if debit := codeDebits[accountID]; debit > 0 {
			legs = append(legs, PostingLeg{AccountID: accountID, Debit: debit})
		}
		if credit := codeCredits[accountID]; credit > 0 {
			legs = append(legs, PostingLeg{AccountID: accountID, Credit: credit})
		}
	}
	return legs
}

func (s *Service) postSettlementJournal(
	ctx context.Context,
	entity *driversettlement.Settlement,
	actor *serviceports.RequestActor,
	reversal bool,
) (*pulid.ID, error) {
	control, err := s.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		return nil, err
	}

	expenseAccountID := control.DefaultDriverPayExpenseAccountID
	if entity.Classification == driverpay.PayeeClassificationOwnerOperator {
		expenseAccountID = control.DefaultPurchasedTransportationAccountID
	}
	multiErr := errortypes.NewMultiError()
	if expenseAccountID.IsNil() {
		multiErr.Add(
			"accountingControl",
			errortypes.ErrRequired,
			"A driver pay expense account (company) or purchased transportation account (owner-operator) must be configured before posting settlements",
		)
	}
	if control.DefaultSettlementsPayableAccountID.IsNil() {
		multiErr.Add(
			"accountingControl",
			errortypes.ErrRequired,
			"A settlements payable account must be configured before posting settlements",
		)
	}

	codeAccounts, err := s.settlementCodeAccounts(ctx, entity)
	if err != nil {
		return nil, err
	}

	totals, _, _ := settlementTotals(entity, codeAccounts)
	if (totals.AdvanceRecovery > 0 || totals.CarryForwardIn > 0 ||
		totals.CarryForwardOut > 0) && control.DefaultDriverAdvanceAccountID.IsNil() {
		multiErr.Add(
			"accountingControl",
			errortypes.ErrRequired,
			"A driver advance receivable account must be configured to post advance recoveries and carry-forwards",
		)
	}
	if totals.EscrowWithheld > 0 && control.DefaultEscrowLiabilityAccountID.IsNil() {
		multiErr.Add(
			"accountingControl",
			errortypes.ErrRequired,
			"An escrow liability account must be configured to post escrow contributions",
		)
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	description := "Driver settlement " + entity.SettlementNumber
	sourceEvent := tenant.JournalSourceEventDriverSettlementPosted
	idempotencyPrefix := "driver-settlement-posted:"
	if reversal {
		description = "Void of driver settlement " + entity.SettlementNumber
		sourceEvent = tenant.JournalSourceEventDriverSettlementVoided
		idempotencyPrefix = "driver-settlement-voided:"
	}

	if !reversal {
		payable := control.DefaultSettlementsPayableAccountID
		entity.PostedPayableAccountID = &payable
	}

	legs := BuildSettlementPostingLegs(entity, &PostingAccounts{
		Expense:       expenseAccountID,
		Reimbursement: control.DefaultDriverReimbursementAccountID,
		Payable:       control.DefaultSettlementsPayableAccountID,
		Advance:       control.DefaultDriverAdvanceAccountID,
		Escrow:        control.DefaultEscrowLiabilityAccountID,
	}, codeAccounts)
	if len(legs) == 0 {
		return nil, nil //nolint:nilnil // a zero-amount settlement posts no journal
	}
	if reversal {
		legs = reverseLegs(legs)
	}

	now := timeutils.NowUnix()
	return s.writeJournal(ctx, &SettlementJournal{
		Entity:         entity,
		Control:        control,
		ActorID:        actor.UserID,
		AccountingDate: now,
		Now:            now,
		Description:    description,
		Event:          sourceEvent,
		IdempotencyKey: idempotencyPrefix + entity.ID.String(),
		Legs:           legs,
	})
}

type SettlementJournal struct {
	Entity         *driversettlement.Settlement
	Control        *tenant.AccountingControl
	ActorID        pulid.ID
	AccountingDate int64
	Now            int64
	Description    string
	Event          tenant.JournalSourceEventType
	IdempotencyKey string
	Legs           []PostingLeg
}

func (j *SettlementJournal) WriteRequest() *journalposting.WriteRequest {
	lines := make([]journalposting.Line, 0, len(j.Legs))
	for _, leg := range j.Legs {
		lines = append(lines, journalposting.Line{
			AccountID: leg.AccountID,
			Debit:     leg.Debit,
			Credit:    leg.Credit,
		})
	}
	return &journalposting.WriteRequest{
		OrganizationID: j.Entity.OrganizationID,
		BusinessUnitID: j.Entity.BusinessUnitID,
		Control:        j.Control,
		ActorID:        j.ActorID,
		AccountingDate: j.AccountingDate,
		Now:            j.Now,
		Subject:        "driver settlement",
		DateField:      "payDate",
		Description:    j.Description,
		Source: journalposting.Source{
			ObjectType:     "DriverSettlement",
			ObjectID:       j.Entity.ID,
			DocumentNumber: j.Entity.SettlementNumber,
			Event:          j.Event,
			IdempotencyKey: j.IdempotencyKey,
		},
		Lines: lines,
	}
}

func (s *Service) writeJournal(ctx context.Context, journal *SettlementJournal) (*pulid.ID, error) {
	result, err := s.journalWriter().Write(ctx, journal.WriteRequest())
	if err != nil {
		return nil, err
	}
	return &result.BatchID, nil
}

func (s *Service) journalWriter() journalposting.Writer {
	return journalposting.Writer{
		Periods:  s.fiscalPeriodRepo,
		Numbers:  s.generator,
		Journals: s.journalRepo,
	}
}

func reverseLegs(legs []PostingLeg) []PostingLeg {
	reversed := make([]PostingLeg, len(legs))
	for idx, leg := range legs {
		reversed[idx] = PostingLeg{AccountID: leg.AccountID, Debit: leg.Credit, Credit: leg.Debit}
	}
	return reversed
}

func BuildSettlementPaymentLegs(
	entity *driversettlement.Settlement,
	payableAccountID, cashAccountID pulid.ID,
) []PostingLeg {
	amount := entity.NetPayMinor
	switch {
	case amount > 0:
		return []PostingLeg{
			{AccountID: payableAccountID, Debit: amount},
			{AccountID: cashAccountID, Credit: amount},
		}
	case amount < 0:
		return []PostingLeg{
			{AccountID: cashAccountID, Debit: -amount},
			{AccountID: payableAccountID, Credit: -amount},
		}
	default:
		return nil
	}
}

func PaymentJournal(
	entity *driversettlement.Settlement,
	control *tenant.AccountingControl,
	actorID pulid.ID,
	paidAt, now int64,
) (*SettlementJournal, error) {
	if payable, err := paymentNeeded(entity); err != nil || !payable {
		return nil, err
	}
	if control.DefaultCashAccountID.IsNil() {
		return nil, errortypes.NewValidationError(
			"accountingControl",
			errortypes.ErrRequired,
			"A default cash account must be configured before recording driver settlement payments",
		)
	}

	return &SettlementJournal{
		Entity:         entity,
		Control:        control,
		ActorID:        actorID,
		AccountingDate: paidAt,
		Now:            now,
		Description:    "Payment of driver settlement " + entity.SettlementNumber,
		Event:          tenant.JournalSourceEventDriverSettlementPaid,
		IdempotencyKey: PaymentIdempotencyKey(entity.ID),
		Legs: BuildSettlementPaymentLegs(
			entity,
			*entity.PostedPayableAccountID,
			control.DefaultCashAccountID,
		),
	}, nil
}

func paymentNeeded(entity *driversettlement.Settlement) (bool, error) {
	if entity.NetPayMinor == 0 {
		return false, nil
	}
	if entity.PostedPayableAccountID == nil || entity.PostedPayableAccountID.IsNil() {
		return false, errortypes.NewBusinessError(
			"Driver settlement has no posted payable account; it cannot be paid",
		).WithParam("settlementId", entity.ID.String())
	}
	return true, nil
}

func PaymentIdempotencyKey(settlementID pulid.ID) string {
	return "driver-settlement-paid:" + settlementID.String()
}

func (s *Service) postPaymentJournal(
	ctx context.Context,
	entity *driversettlement.Settlement,
	actorID pulid.ID,
	paidAt int64,
) (*pulid.ID, error) {
	if payable, err := paymentNeeded(entity); err != nil || !payable {
		return nil, err
	}

	control, err := s.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		return nil, err
	}
	journal, err := PaymentJournal(entity, control, actorID, paidAt, timeutils.NowUnix())
	if err != nil || journal == nil {
		return nil, err
	}
	return s.writeJournal(ctx, journal)
}

func (s *Service) settlementCodeAccounts(
	ctx context.Context,
	entity *driversettlement.Settlement,
) (map[pulid.ID]pulid.ID, error) {
	seen := make(map[pulid.ID]struct{}, len(entity.Lines))
	ids := make([]pulid.ID, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil || line.PayCodeID == nil || line.PayCodeID.IsNil() {
			continue
		}
		if _, ok := seen[*line.PayCodeID]; ok {
			continue
		}
		seen[*line.PayCodeID] = struct{}{}
		ids = append(ids, *line.PayCodeID)
	}
	if len(ids) == 0 {
		return map[pulid.ID]pulid.ID{}, nil
	}

	codes, err := s.payCodeRepo.GetByIDs(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}, ids)
	if err != nil {
		return nil, err
	}

	accounts := make(map[pulid.ID]pulid.ID, len(codes))
	for _, code := range codes {
		if code.GLAccountID != nil && !code.GLAccountID.IsNil() {
			accounts[code.ID] = *code.GLAccountID
		}
	}
	return accounts, nil
}
