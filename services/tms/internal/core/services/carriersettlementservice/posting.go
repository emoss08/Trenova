package carriersettlementservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type PostingAccounts struct {
	Expense pulid.ID
	Payable pulid.ID
}

type PostingLeg struct {
	AccountID pulid.ID
	Debit     int64
	Credit    int64
}

// BuildCarrierSettlementPostingLegs builds the balanced journal legs for a
// carrier settlement: DR purchased-transportation expense per resolved account
// (line-level GL override wins over the default), CR accounts payable for the
// net. Negative adjustment lines credit their expense account; a negative net
// debits AP.
func BuildCarrierSettlementPostingLegs(
	entity *carriersettlement.CarrierSettlement,
	accounts *PostingAccounts,
) []PostingLeg {
	amounts := make(map[pulid.ID]int64)
	var net int64
	for _, line := range entity.Lines {
		if line == nil || line.AmountMinor == 0 {
			continue
		}
		accountID := accounts.Expense
		if line.GLAccountID != nil && !line.GLAccountID.IsNil() {
			accountID = *line.GLAccountID
		}
		amounts[accountID] += line.AmountMinor
		net += line.AmountMinor
	}

	accountIDs := make([]pulid.ID, 0, len(amounts))
	for accountID := range amounts {
		accountIDs = append(accountIDs, accountID)
	}
	slices.SortFunc(accountIDs, func(a, b pulid.ID) int {
		return strings.Compare(a.String(), b.String())
	})

	legs := make([]PostingLeg, 0, len(accountIDs)+1)
	for _, accountID := range accountIDs {
		amount := amounts[accountID]
		switch {
		case amount > 0:
			legs = append(legs, PostingLeg{AccountID: accountID, Debit: amount})
		case amount < 0:
			legs = append(legs, PostingLeg{AccountID: accountID, Credit: -amount})
		}
	}
	switch {
	case net > 0:
		legs = append(legs, PostingLeg{AccountID: accounts.Payable, Credit: net})
	case net < 0:
		legs = append(legs, PostingLeg{AccountID: accounts.Payable, Debit: -net})
	}
	return legs
}

// BuildCarrierSettlementPaymentLegs builds the payment journal for MarkPaid:
// DR accounts payable, CR cash for the settlement's net payable.
func BuildCarrierSettlementPaymentLegs(
	entity *carriersettlement.CarrierSettlement,
	payableAccountID, cashAccountID pulid.ID,
) []PostingLeg {
	if entity.NetPayableMinor == 0 {
		return nil
	}
	amount := entity.NetPayableMinor
	if amount > 0 {
		return []PostingLeg{
			{AccountID: payableAccountID, Debit: amount},
			{AccountID: cashAccountID, Credit: amount},
		}
	}
	return []PostingLeg{
		{AccountID: cashAccountID, Debit: -amount},
		{AccountID: payableAccountID, Credit: -amount},
	}
}

func (s *Service) Post(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
	actor *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement posting"); err != nil {
		return nil, err
	}

	var updated *carriersettlement.CarrierSettlement
	var previous carriersettlement.CarrierSettlement
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.getForUpdate(txCtx, tenantInfo, settlementID)
		if txErr != nil {
			return txErr
		}
		if entity.Status != carriersettlement.StatusApproved {
			return transitionError(entity.Status, carriersettlement.StatusPosted)
		}
		previous = *entity

		batchID, txErr := s.postSettlementJournal(txCtx, entity, actor, false)
		if txErr != nil {
			return txErr
		}

		now := timeutils.NowUnix()
		if txErr = s.appendLedgerEntry(txCtx, entity, &ledgerEntryParams{
			EntryType:       carriersettlement.LedgerEntryTypeBill,
			SourceEvent:     tenant.JournalSourceEventCarrierSettlementPosted,
			JournalBatchID:  batchID,
			AmountMinor:     entity.NetPayableMinor,
			TransactionDate: now,
			Actor:           actor,
		}); txErr != nil {
			return txErr
		}
		if txErr = s.costEventRepo.MarkSettled(txCtx, tenantInfo, entity.ID); txErr != nil {
			return txErr
		}

		entity.Status = carriersettlement.StatusPosted
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
		"Carrier settlement posted to general ledger")
	return updated, nil
}

func (s *Service) postVoidReversal(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	actor *serviceports.RequestActor,
) error {
	batchID, err := s.postSettlementJournal(ctx, entity, actor, true)
	if err != nil {
		return err
	}
	entity.VoidJournalBatchID = batchID
	if entity.NetPayableMinor == 0 {
		return nil
	}
	return s.appendLedgerEntry(ctx, entity, &ledgerEntryParams{
		EntryType:       carriersettlement.LedgerEntryTypeAdjustment,
		SourceEvent:     tenant.JournalSourceEventCarrierSettlementVoided,
		JournalBatchID:  batchID,
		AmountMinor:     -entity.NetPayableMinor,
		TransactionDate: timeutils.NowUnix(),
		Actor:           actor,
	})
}

type ResolvedPostingAccounts struct {
	Expense pulid.ID
	Payable pulid.ID
	Cash    pulid.ID
}

// ReverseLegs swaps debit and credit on every leg, producing the reversal
// journal for a void.
func ReverseLegs(legs []PostingLeg) []PostingLeg {
	swapped := make([]PostingLeg, 0, len(legs))
	for _, leg := range legs {
		swapped = append(swapped, PostingLeg{
			AccountID: leg.AccountID,
			Debit:     leg.Credit,
			Credit:    leg.Debit,
		})
	}
	return swapped
}

func (s *Service) resolvePostingAccounts(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	control *tenant.AccountingControl,
	requireCash bool,
) (*ResolvedPostingAccounts, error) {
	carrierControl, err := s.settlementControl.GetOrCreate(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}
	return ResolvePostingAccounts(entity, carrierControl, control, requireCash)
}

// ResolvePostingAccounts resolves the GL accounts a carrier settlement posts
// to: line-level overrides win, then the carrier settlement control defaults,
// then the accounting control fallbacks. Missing required accounts return a
// field-level validation error.
func ResolvePostingAccounts(
	entity *carriersettlement.CarrierSettlement,
	carrierControl *tenant.CarrierSettlementControl,
	control *tenant.AccountingControl,
	requireCash bool,
) (*ResolvedPostingAccounts, error) {
	resolved := &ResolvedPostingAccounts{}
	if carrierControl.DefaultPurchasedTransportationAccountID != nil &&
		!carrierControl.DefaultPurchasedTransportationAccountID.IsNil() {
		resolved.Expense = *carrierControl.DefaultPurchasedTransportationAccountID
	} else {
		resolved.Expense = control.DefaultPurchasedTransportationAccountID
	}
	if carrierControl.DefaultAPAccountID != nil && !carrierControl.DefaultAPAccountID.IsNil() {
		resolved.Payable = *carrierControl.DefaultAPAccountID
	} else {
		resolved.Payable = control.DefaultAPAccountID
	}
	resolved.Cash = control.DefaultCashAccountID

	multiErr := errortypes.NewMultiError()
	if resolved.Expense.IsNil() && settlementHasUnmappedLines(entity) {
		multiErr.Add(
			"carrierSettlementControl",
			errortypes.ErrRequired,
			"A purchased transportation expense account must be configured before posting carrier settlements",
		)
	}
	if resolved.Payable.IsNil() {
		multiErr.Add(
			"carrierSettlementControl",
			errortypes.ErrRequired,
			"An accounts payable account must be configured before posting carrier settlements",
		)
	}
	if requireCash && resolved.Cash.IsNil() {
		multiErr.Add(
			"accountingControl",
			errortypes.ErrRequired,
			"A default cash account must be configured before recording carrier settlement payments",
		)
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return resolved, nil
}

// PostingAccountsFromSnapshot returns the accounts stamped on the settlement
// when it was posted. Paid and void journals must build against these, never
// against a re-resolved control row.
func PostingAccountsFromSnapshot(
	entity *carriersettlement.CarrierSettlement,
) (*PostingAccounts, error) {
	if entity.PostedAPAccountID == nil || entity.PostedAPAccountID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"Carrier settlement has no posted account snapshot; it cannot be paid or voided",
		).WithParam("settlementId", entity.ID.String())
	}

	accounts := &PostingAccounts{Payable: *entity.PostedAPAccountID}
	if entity.PostedExpenseAccountID != nil {
		accounts.Expense = *entity.PostedExpenseAccountID
	}
	return accounts, nil
}

func stampPostedAccounts(
	entity *carriersettlement.CarrierSettlement,
	accounts *PostingAccounts,
) {
	if !accounts.Expense.IsNil() {
		expense := accounts.Expense
		entity.PostedExpenseAccountID = &expense
	}
	payable := accounts.Payable
	entity.PostedAPAccountID = &payable
}

func settlementHasUnmappedLines(entity *carriersettlement.CarrierSettlement) bool {
	for _, line := range entity.Lines {
		if line == nil || line.AmountMinor == 0 {
			continue
		}
		if line.GLAccountID == nil || line.GLAccountID.IsNil() {
			return true
		}
	}
	return false
}

func (s *Service) postSettlementJournal(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	actor *serviceports.RequestActor,
	reversal bool,
) (*pulid.ID, error) {
	control, err := s.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		return nil, err
	}

	// A void must reverse exactly the legs that were posted. The accounts are
	// therefore resolved from the control tables only at post time and stamped
	// onto the settlement; a control edit between post and void can never split
	// the reversal across accounts.
	var postingAccounts *PostingAccounts
	if reversal {
		postingAccounts, err = PostingAccountsFromSnapshot(entity)
		if err != nil {
			return nil, err
		}
	} else {
		accounts, resolveErr := s.resolvePostingAccounts(ctx, entity, control, false)
		if resolveErr != nil {
			return nil, resolveErr
		}
		postingAccounts = &PostingAccounts{
			Expense: accounts.Expense,
			Payable: accounts.Payable,
		}
		stampPostedAccounts(entity, postingAccounts)
	}

	description := "Carrier settlement " + entity.SettlementNumber
	sourceEvent := tenant.JournalSourceEventCarrierSettlementPosted
	idempotencyPrefix := "carrier-settlement-posted:"
	if reversal {
		description = "Void of carrier settlement " + entity.SettlementNumber
		sourceEvent = tenant.JournalSourceEventCarrierSettlementVoided
		idempotencyPrefix = "carrier-settlement-voided:"
	}

	legs := BuildCarrierSettlementPostingLegs(entity, postingAccounts)
	if len(legs) == 0 {
		return nil, nil //nolint:nilnil // a zero-amount settlement posts no journal
	}
	if reversal {
		legs = ReverseLegs(legs)
	}

	return s.createJournalPosting(ctx, &createJournalPostingParams{
		Entity:         entity,
		Actor:          actor,
		Control:        control,
		Legs:           legs,
		Description:    description,
		SourceEvent:    sourceEvent,
		IdempotencyKey: idempotencyPrefix + entity.ID.String(),
	})
}

func (s *Service) postPaymentJournal(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	actor *serviceports.RequestActor,
) (*pulid.ID, error) {
	control, err := s.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		return nil, err
	}

	// The payment must relieve the same AP account the posting credited, so the
	// payable side comes from the post-time snapshot; only the cash account is
	// resolved live.
	accounts, err := PostingAccountsFromSnapshot(entity)
	if err != nil {
		return nil, err
	}
	if control.DefaultCashAccountID.IsNil() {
		return nil, errortypes.NewValidationError(
			"accountingControl",
			errortypes.ErrRequired,
			"A default cash account must be configured before recording carrier settlement payments",
		)
	}

	legs := BuildCarrierSettlementPaymentLegs(
		entity,
		accounts.Payable,
		control.DefaultCashAccountID,
	)
	if len(legs) == 0 {
		return nil, nil //nolint:nilnil // a zero-amount settlement records no payment journal
	}

	return s.createJournalPosting(ctx, &createJournalPostingParams{
		Entity:         entity,
		Actor:          actor,
		Control:        control,
		Legs:           legs,
		Description:    "Payment of carrier settlement " + entity.SettlementNumber,
		SourceEvent:    tenant.JournalSourceEventCarrierSettlementPaid,
		IdempotencyKey: "carrier-settlement-paid:" + entity.ID.String(),
	})
}

type createJournalPostingParams struct {
	Entity         *carriersettlement.CarrierSettlement
	Actor          *serviceports.RequestActor
	Control        *tenant.AccountingControl
	Legs           []PostingLeg
	Description    string
	SourceEvent    tenant.JournalSourceEventType
	IdempotencyKey string
}

func (s *Service) createJournalPosting(
	ctx context.Context,
	params *createJournalPostingParams,
) (*pulid.ID, error) {
	entity := params.Entity
	period, err := s.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
		Date:  timeutils.NowUnix(),
	})
	if err != nil {
		return nil, errortypes.NewValidationError(
			"payDate",
			errortypes.ErrInvalid,
			"Carrier settlement posting date must fall within a fiscal period",
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
	workflow := settlementshared.ResolvePostingWorkflow(params.Control, params.Actor.UserID, now)

	lines := make([]repositories.JournalPostingLine, 0, len(params.Legs))
	var totalDebit, totalCredit int64
	for idx, leg := range params.Legs {
		totalDebit += leg.Debit
		totalCredit += leg.Credit
		lines = append(lines, repositories.JournalPostingLine{
			ID:           pulid.MustNew("jel_"),
			GLAccountID:  leg.AccountID,
			LineNumber:   int16(idx + 1),
			Description:  params.Description,
			DebitAmount:  leg.Debit,
			CreditAmount: leg.Credit,
			NetAmount:    leg.Debit - leg.Credit,
		})
	}

	batchID := pulid.MustNew("jb_")
	if err = s.journalRepo.CreatePosting(ctx, repositories.CreateJournalPostingParams{
		BatchID:              batchID,
		OrganizationID:       entity.OrganizationID,
		BusinessUnitID:       entity.BusinessUnitID,
		BatchNumber:          batchNumber,
		BatchType:            "System",
		BatchStatus:          workflow.BatchStatus,
		BatchDescription:     params.Description,
		FiscalYearID:         period.FiscalYearID,
		FiscalPeriodID:       period.ID,
		AccountingDate:       now,
		PostedAt:             workflow.PostedAt,
		PostedByID:           workflow.PostedByID,
		CreatedByID:          params.Actor.UserID,
		UpdatedByID:          params.Actor.UserID,
		EntryID:              pulid.MustNew("je_"),
		EntryNumber:          entryNumber,
		EntryType:            "Standard",
		EntryStatus:          workflow.EntryStatus,
		ReferenceNumber:      entity.SettlementNumber,
		ReferenceType:        params.SourceEvent.String(),
		ReferenceID:          entity.ID.String(),
		EntryDescription:     params.Description,
		TotalDebit:           totalDebit,
		TotalCredit:          totalCredit,
		IsPosted:             workflow.PostedAt != nil,
		IsAutoGenerated:      false,
		RequiresApproval:     workflow.RequiresApproval,
		IsApproved:           workflow.IsApproved,
		ApprovedByID:         workflow.ApprovedByID,
		ApprovedAt:           workflow.ApprovedAt,
		SourceID:             pulid.MustNew("jsrc_"),
		SourceObjectType:     "CarrierSettlement",
		SourceObjectID:       entity.ID.String(),
		SourceEventType:      params.SourceEvent.String(),
		SourceStatus:         workflow.EntryStatus,
		SourceDocumentNumber: entity.SettlementNumber,
		SourceIdempotencyKey: params.IdempotencyKey,
		Lines:                lines,
	}); err != nil {
		return nil, err
	}

	return &batchID, nil
}

type ledgerEntryParams struct {
	EntryType       carriersettlement.LedgerEntryType
	SourceEvent     tenant.JournalSourceEventType
	JournalBatchID  *pulid.ID
	AmountMinor     int64
	TransactionDate int64
	Actor           *serviceports.RequestActor
}

func (s *Service) appendLedgerEntry(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	params *ledgerEntryParams,
) error {
	if params.AmountMinor == 0 {
		return nil
	}
	settlementID := entity.ID
	entry := &carriersettlement.LedgerEntry{
		OrganizationID:      entity.OrganizationID,
		BusinessUnitID:      entity.BusinessUnitID,
		CarrierID:           entity.CarrierID,
		EntryType:           params.EntryType,
		SourceObjectType:    "CarrierSettlement",
		SourceObjectID:      entity.ID.String(),
		SourceEventType:     params.SourceEvent.String(),
		RelatedSettlementID: &settlementID,
		JournalBatchID:      params.JournalBatchID,
		DocumentNumber:      entity.SettlementNumber,
		TransactionDate:     params.TransactionDate,
		LineNumber:          1,
		AmountMinor:         params.AmountMinor,
		CreatedByID:         params.Actor.UserID,
	}
	multiErr := errortypes.NewMultiError()
	entry.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return s.ledgerRepo.AppendEntries(ctx, []*carriersettlement.LedgerEntry{entry})
}
