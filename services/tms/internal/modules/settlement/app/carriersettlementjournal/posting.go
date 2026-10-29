package carriersettlementjournal

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalposting"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/carriersettlementcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func (s *Writer) resolvePostingAccounts(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	control *accountingcontrol.AccountingControl,
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
	carrierControl *carriersettlementcontrol.CarrierSettlementControl,
	control *accountingcontrol.AccountingControl,
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

type Draft struct {
	request     *journalposting.WriteRequest
	posting     *journalposting.Plan
	JournalPlan *settlementshared.JournalPlan
}

func (s *Writer) PostSettlement(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	actor *permissionports.RequestActor,
	reversal bool,
) (*pulid.ID, error) {
	draft, err := s.PlanSettlement(ctx, entity, actor.UserID, reversal)
	if err != nil || draft == nil {
		return nil, err
	}
	return s.writeJournal(ctx, entity, actor, draft)
}

func (s *Writer) PlanSettlement(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	userID pulid.ID,
	reversal bool,
) (*Draft, error) {
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
	sourceEvent := accountingcontrol.JournalSourceEventCarrierSettlementPosted
	idempotencyPrefix := "carrier-settlement-posted:"
	if reversal {
		description = "Void of carrier settlement " + entity.SettlementNumber
		sourceEvent = accountingcontrol.JournalSourceEventCarrierSettlementVoided
		idempotencyPrefix = "carrier-settlement-voided:"
	}

	legs := BuildCarrierSettlementPostingLegs(entity, postingAccounts)
	if len(legs) == 0 {
		return nil, nil //nolint:nilnil // a zero-amount settlement posts no journal
	}
	if reversal {
		legs = ReverseLegs(legs)
	}

	return s.draftJournal(ctx, entity, userID, &createJournalPostingParams{
		Control:        control,
		Legs:           legs,
		Description:    description,
		SourceEvent:    sourceEvent,
		IdempotencyKey: idempotencyPrefix + entity.ID.String(),
	})
}

func (s *Writer) PostPayment(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	actor *permissionports.RequestActor,
	paidAt int64,
) (*pulid.ID, error) {
	draft, err := s.PlanPayment(ctx, entity, actor.UserID, paidAt)
	if err != nil || draft == nil {
		return nil, err
	}
	return s.writeJournal(ctx, entity, actor, draft)
}

func (s *Writer) PlanPayment(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	userID pulid.ID,
	paidAt int64,
) (*Draft, error) {
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

	return s.draftJournal(ctx, entity, userID, &createJournalPostingParams{
		Control:        control,
		Legs:           legs,
		Description:    "Payment of carrier settlement " + entity.SettlementNumber,
		SourceEvent:    accountingcontrol.JournalSourceEventCarrierSettlementPaid,
		IdempotencyKey: "carrier-settlement-paid:" + entity.ID.String(),
		AccountingDate: paidAt,
	})
}

type createJournalPostingParams struct {
	Control        *accountingcontrol.AccountingControl
	Legs           []PostingLeg
	Description    string
	SourceEvent    accountingcontrol.JournalSourceEventType
	IdempotencyKey string
	AccountingDate int64
}

func (s *Writer) draftJournal(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	userID pulid.ID,
	params *createJournalPostingParams) (*Draft, error) {
	now := timeutils.NowUnix()
	accountingDate := params.AccountingDate
	if accountingDate == 0 {
		accountingDate = now
	}

	lines := make([]journalposting.Line, 0, len(params.Legs))
	for _, leg := range params.Legs {
		lines = append(lines, journalposting.Line{
			AccountID: leg.AccountID,
			Debit:     leg.Debit,
			Credit:    leg.Credit,
		})
	}

	request := &journalposting.WriteRequest{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		Control:        params.Control,
		ActorID:        userID,
		AccountingDate: accountingDate,
		Now:            now,
		Subject:        "carrier settlement",
		DateField:      "payDate",
		Description:    params.Description,
		Source: journalposting.Source{
			ObjectType:     "CarrierSettlement",
			ObjectID:       entity.ID,
			DocumentNumber: entity.SettlementNumber,
			Event:          params.SourceEvent,
			IdempotencyKey: params.IdempotencyKey,
		},
		Lines: lines,
	}
	posting, err := s.journalWriter().Plan(ctx, request)
	if err != nil {
		return nil, err
	}

	return &Draft{
		request:     request,
		posting:     posting,
		JournalPlan: settlementshared.JournalPlanFrom(posting, params.Description),
	}, nil
}

func (s *Writer) writeJournal(
	ctx context.Context,
	_ *carriersettlement.CarrierSettlement,
	_ *permissionports.RequestActor,
	draft *Draft,
) (*pulid.ID, error) {
	result, err := s.journalWriter().WritePlan(ctx, draft.request, draft.posting)
	if err != nil {
		return nil, err
	}
	return &result.BatchID, nil
}

func (s *Writer) journalWriter() journalposting.Writer {
	return journalposting.Writer{
		Periods:  s.fiscalPeriodRepo,
		Numbers:  s.generator,
		Journals: s.journalRepo,
	}
}
