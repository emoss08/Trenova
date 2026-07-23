package driversettlementservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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
		updated, txErr = s.settlementRepo.Update(txCtx, entity)
		return txErr
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
			Title:      "Your settlement statement is ready",
			Message:    "Settlement " + updated.SettlementNumber + " has been issued. Open Dash to review it.",
			Link:       "/dash/pay/" + updated.ID.String(),
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

//nolint:funlen,cyclop // journal assembly enumerates every posting leg explicitly
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

	period, err := s.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
		Date:  timeutils.NowUnix(),
	})
	if err != nil {
		return nil, errortypes.NewValidationError(
			"payDate",
			errortypes.ErrInvalid,
			"Settlement posting date must fall within a fiscal period",
		)
	}

	batchNumber, err := s.generator.GenerateJournalBatchNumber(
		ctx, entity.OrganizationID, entity.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}
	entryNumber, err := s.generator.GenerateJournalEntryNumber(
		ctx, entity.OrganizationID, entity.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved,
		approvedByID, approvedAt := settlementPostingWorkflow(control, actor.UserID, now)

	description := "Driver settlement " + entity.SettlementNumber
	sourceEvent := tenant.JournalSourceEventDriverSettlementPosted
	idempotencyPrefix := "driver-settlement-posted:"
	if reversal {
		description = "Void of driver settlement " + entity.SettlementNumber
		sourceEvent = tenant.JournalSourceEventDriverSettlementVoided
		idempotencyPrefix = "driver-settlement-voided:"
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

	lines := make([]repositories.JournalPostingLine, 0, len(legs))
	var totalDebit, totalCredit int64
	for idx, l := range legs {
		debit, credit := l.Debit, l.Credit
		if reversal {
			debit, credit = credit, debit
		}
		totalDebit += debit
		totalCredit += credit
		lines = append(lines, repositories.JournalPostingLine{
			ID:           pulid.MustNew("jel_"),
			GLAccountID:  l.AccountID,
			LineNumber:   int16(idx + 1),
			Description:  description,
			DebitAmount:  debit,
			CreditAmount: credit,
			NetAmount:    debit - credit,
		})
	}

	batchID := pulid.MustNew("jb_")
	if err = s.journalRepo.CreatePosting(ctx, repositories.CreateJournalPostingParams{
		BatchID:              batchID,
		OrganizationID:       entity.OrganizationID,
		BusinessUnitID:       entity.BusinessUnitID,
		BatchNumber:          batchNumber,
		BatchType:            "System",
		BatchStatus:          batchStatus,
		BatchDescription:     description,
		FiscalYearID:         period.FiscalYearID,
		FiscalPeriodID:       period.ID,
		AccountingDate:       now,
		PostedAt:             postedAt,
		PostedByID:           postedByID,
		CreatedByID:          actor.UserID,
		UpdatedByID:          actor.UserID,
		EntryID:              pulid.MustNew("je_"),
		EntryNumber:          entryNumber,
		EntryType:            "Standard",
		EntryStatus:          entryStatus,
		ReferenceNumber:      entity.SettlementNumber,
		ReferenceType:        sourceEvent.String(),
		ReferenceID:          entity.ID.String(),
		EntryDescription:     description,
		TotalDebit:           totalDebit,
		TotalCredit:          totalCredit,
		IsPosted:             postedAt != nil,
		IsAutoGenerated:      false,
		RequiresApproval:     requiresApproval,
		IsApproved:           isApproved,
		ApprovedByID:         approvedByID,
		ApprovedAt:           approvedAt,
		SourceID:             pulid.MustNew("jsrc_"),
		SourceObjectType:     "DriverSettlement",
		SourceObjectID:       entity.ID.String(),
		SourceEventType:      sourceEvent.String(),
		SourceStatus:         entryStatus,
		SourceDocumentNumber: entity.SettlementNumber,
		SourceIdempotencyKey: idempotencyPrefix + entity.ID.String(),
		Lines:                lines,
	}); err != nil {
		return nil, err
	}

	return &batchID, nil
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

func settlementPostingWorkflow( //nolint:gocritic // mirrors customer payment workflow shape
	control *tenant.AccountingControl,
	userID pulid.ID,
	now int64,
) (string, string, *int64, pulid.ID, bool, bool, pulid.ID, *int64) {
	entryStatus := "Posted"
	batchStatus := "Posted"
	postedAt := &now
	postedByID := userID
	requiresApproval := false
	isApproved := true
	approvedByID := userID
	approvedAt := &now
	if control != nil && control.JournalPostingMode == tenant.JournalPostingModeManual {
		entryStatus = "Pending"
		batchStatus = "Pending"
		postedAt = nil
		postedByID = pulid.Nil
		requiresApproval = control.RequireManualJEApproval
		isApproved = !control.RequireManualJEApproval
		if !requiresApproval {
			entryStatus = "Approved"
			batchStatus = "Approved"
		} else {
			approvedByID = pulid.Nil
			approvedAt = nil
		}
	}
	return entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved,
		approvedByID, approvedAt
}
