package accountinginboundservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type creditPlan struct {
	memoID       pulid.ID
	number       string
	applications []*services.CreditMemoApplicationInput
}

type settlementPlan struct {
	kind       repositories.PayableKind
	id         pulid.ID
	partyID    pulid.ID
	number     string
	objectType accountingsync.SyncObjectType
	netMinor   int64
}

type postingPlan struct {
	change      *accountingsync.AccountingInboundChange
	provider    string
	reason      accountingsync.InboundChangeReason
	resolution  string
	paidAt      int64
	customerID  pulid.ID
	method      string
	cash        []*services.CustomerPaymentApplicationInput
	cashMinor   int64
	credits     []*creditPlan
	settlements []*settlementPlan
	lines       []services.AccountingInboundPostingLine
}

func (p *postingPlan) block(
	reason accountingsync.InboundChangeReason,
	resolution string,
) *postingPlan {
	if p.reason == "" {
		p.reason = reason
		p.resolution = resolution
	}
	return p
}

func (p *postingPlan) blocked() bool {
	return p.reason != ""
}

func (p *postingPlan) ignorable() bool {
	return p.reason == accountingsync.InboundReasonNotTrenovaDocument ||
		p.reason == accountingsync.InboundReasonSentFromTrenova
}

func (p *postingPlan) objectCount() int {
	count := len(p.credits) + len(p.settlements)
	if p.change.Kind == accountingsync.InboundCustomerPayment && p.change.AmountMinor > 0 {
		count++
	}
	return count
}

func (s *Service) plan(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
	change *accountingsync.AccountingInboundChange,
) (*postingPlan, error) {
	p := &postingPlan{
		change:   change,
		provider: accountingsync.ProviderName(conn.IntegrationType),
		paidAt:   change.TxnDate,
	}

	echo, err := s.echoes(ctx, change.TenantInfo(), change.ConnectionID, change.Kind,
		[]string{change.ExternalID})
	if err != nil {
		return nil, err
	}
	if echo[change.ExternalID] {
		return p.block(accountingsync.InboundReasonSentFromTrenova, sentFromTrenovaText(p)), nil
	}

	switch change.Kind {
	case accountingsync.InboundCustomerPayment:
		err = s.planCustomerPayment(ctx, conn, p)
	case accountingsync.InboundBillPayment:
		err = s.planBillPayment(ctx, p)
	}
	if err != nil {
		return nil, err
	}
	if p.blocked() {
		return p, nil
	}
	return p, s.checkPeriod(ctx, p)
}

func (s *Service) matchLines(
	ctx context.Context,
	change *accountingsync.AccountingInboundChange,
	objectTypes []accountingsync.SyncObjectType,
	kinds []accountingsync.InboundDocumentKind,
) (matched []*accountingsync.InboundLine, unmatched int, err error) {
	ids := make([]string, 0, len(change.Document.Lines))
	for _, line := range change.Document.Lines {
		if slices.Contains(kinds, line.DocumentKind) && line.DocumentExternalID != "" {
			ids = append(ids, line.DocumentExternalID)
		}
	}
	records, err := s.records.ListByExternalIDs(
		ctx,
		&repositories.ListAccountingSyncRecordsByExternalIDsRequest{
			TenantInfo:   change.TenantInfo(),
			ConnectionID: change.ConnectionID,
			ObjectTypes:  objectTypes,
			ExternalIDs:  ids,
		},
	)
	if err != nil {
		return nil, 0, err
	}
	byExternal := make(map[string]*accountingsync.AccountingSyncRecord, len(records))
	for _, record := range records {
		if record.Operation == accountingsync.SyncOperationCreate {
			byExternal[record.ExternalID] = record
		}
	}

	matched = make([]*accountingsync.InboundLine, 0, len(change.Document.Lines))
	for _, line := range change.Document.Lines {
		line.ObjectType = ""
		line.ObjectID = pulid.Nil
		line.ObjectNumber = ""
		line.OpenMinor = 0
		record, ok := byExternal[line.DocumentExternalID]
		if !ok || !slices.Contains(kinds, line.DocumentKind) ||
			!slices.Contains(line.DocumentKind.SyncObjectTypes(), record.ObjectType) {
			if line.AmountMinor != 0 {
				unmatched++
			}
			continue
		}
		line.ObjectType = record.ObjectType
		line.ObjectID = record.ObjectID
		line.ObjectNumber = record.ObjectNumber
		matched = append(matched, line)
	}
	return matched, unmatched, nil
}

func (s *Service) matchedDocuments(
	ctx context.Context,
	p *postingPlan,
	objectTypes []accountingsync.SyncObjectType,
	kinds []accountingsync.InboundDocumentKind,
) ([]*accountingsync.InboundLine, error) {
	matched, unmatched, err := s.matchLines(ctx, p.change, objectTypes, kinds)
	if err != nil {
		return nil, err
	}
	switch {
	case len(matched) == 0:
		p.block(accountingsync.InboundReasonNotTrenovaDocument, notTrenovaText(p))
		return nil, nil
	case unmatched > 0:
		p.block(accountingsync.InboundReasonUnknownDocument, unknownDocumentText(p, unmatched))
		return nil, nil
	default:
		return matched, nil
	}
}

func orderedObjects(matched []*accountingsync.InboundLine) []pulid.ID {
	ids := make([]pulid.ID, 0, len(matched))
	for _, line := range matched {
		if !slices.Contains(ids, line.ObjectID) {
			ids = append(ids, line.ObjectID)
		}
	}
	return ids
}

func (s *Service) planCustomerPayment(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
	p *postingPlan,
) error {
	change := p.change
	matched, err := s.matchedDocuments(
		ctx,
		p,
		[]accountingsync.SyncObjectType{
			accountingsync.SyncObjectInvoice,
			accountingsync.SyncObjectDebitMemo,
			accountingsync.SyncObjectCreditMemo,
		},
		[]accountingsync.InboundDocumentKind{
			accountingsync.InboundDocInvoice,
			accountingsync.InboundDocDebitMemo,
			accountingsync.InboundDocCreditMemo,
		},
	)
	if err != nil || p.blocked() {
		return err
	}

	found, err := s.invoices.GetByIDs(ctx, repositories.GetInvoicesByIDsRequest{
		TenantInfo: change.TenantInfo(),
		InvoiceIDs: orderedObjects(matched),
	})
	if err != nil {
		return err
	}
	invoices := make(map[pulid.ID]*invoice.Invoice, len(found))
	for _, inv := range found {
		invoices[inv.ID] = inv
	}

	credit, owed := checkInvoices(p, matched, invoices)
	if p.blocked() || !checkBalances(p, orderedObjects(matched), invoices, credit, owed) {
		return nil
	}
	if leftover := allocate(p, matched, invoices, credit); leftover > 0 {
		p.block(accountingsync.InboundReasonUnknownDocument, unknownDocumentText(p, 1))
		return nil
	}
	if p.cashMinor > change.AmountMinor {
		p.block(accountingsync.InboundReasonOverpayment,
			totalOverText(p.cashMinor, change.AmountMinor, change.CurrencyCode))
		return nil
	}
	p.method = string(s.paymentMethod(ctx, conn, change))
	return nil
}

func checkInvoices(
	p *postingPlan,
	matched []*accountingsync.InboundLine,
	invoices map[pulid.ID]*invoice.Invoice,
) (credit, owed map[pulid.ID]int64) {
	credit = make(map[pulid.ID]int64, len(matched))
	owed = make(map[pulid.ID]int64, len(matched))
	var customerID pulid.ID
	for _, line := range matched {
		inv, ok := invoices[line.ObjectID]
		switch {
		case !ok:
			p.block(accountingsync.InboundReasonUnknownDocument, unknownDocumentText(p, 1))
		case !customerID.IsNil() && customerID != inv.CustomerID:
			p.block(accountingsync.InboundReasonPartyMismatch, partyMismatchText(p))
		case !strings.EqualFold(inv.CurrencyCode, p.change.CurrencyCode):
			p.block(accountingsync.InboundReasonCurrencyMismatch,
				currencyMismatchText(p, inv.Number, inv.CurrencyCode))
		case inv.Status != invoice.StatusPosted:
			p.block(accountingsync.InboundReasonAlreadyPaid, notOpenText(inv.Number, inv.Status))
		}
		if p.blocked() {
			return credit, owed
		}
		customerID = inv.CustomerID
		if inv.IsCreditMemo() {
			credit[inv.ID] += line.AmountMinor
			line.OpenMinor = inv.CreditRemainingMinor()
			continue
		}
		owed[inv.ID] += line.AmountMinor
		line.OpenMinor = inv.OpenBalanceMinor()
	}
	p.change.PartyObjectID = customerID
	p.customerID = customerID
	return credit, owed
}

func checkBalances(
	p *postingPlan,
	ids []pulid.ID,
	invoices map[pulid.ID]*invoice.Invoice,
	credit, owed map[pulid.ID]int64,
) bool {
	for _, id := range ids {
		inv := invoices[id]
		if amount, ok := credit[id]; ok && amount > inv.CreditRemainingMinor() {
			p.block(accountingsync.InboundReasonOverpayment,
				creditOverText(inv.Number, amount, inv.CreditRemainingMinor(), inv.CurrencyCode))
			return false
		}
		amount, ok := owed[id]
		if !ok {
			continue
		}
		open := inv.OpenBalanceMinor()
		if open == 0 {
			p.block(accountingsync.InboundReasonAlreadyPaid, alreadyPaidText(inv.Number))
			return false
		}
		if amount > open {
			p.block(accountingsync.InboundReasonOverpayment,
				overpaymentText(inv.Number, amount, open, inv.CurrencyCode))
			return false
		}
	}
	return true
}

func allocate(
	p *postingPlan,
	matched []*accountingsync.InboundLine,
	invoices map[pulid.ID]*invoice.Invoice,
	credit map[pulid.ID]int64,
) int64 {
	memos := make([]pulid.ID, 0, len(credit))
	for _, line := range matched {
		if invoices[line.ObjectID].IsCreditMemo() && !slices.Contains(memos, line.ObjectID) {
			memos = append(memos, line.ObjectID)
		}
	}
	remaining := make(map[pulid.ID]int64, len(credit))
	for id, amount := range credit {
		remaining[id] = amount
	}
	byMemo := make(map[pulid.ID]*creditPlan, len(memos))
	cash := make(map[pulid.ID]int64, len(matched))
	order := make([]pulid.ID, 0, len(matched))

	for _, line := range matched {
		inv := invoices[line.ObjectID]
		if inv.IsCreditMemo() {
			continue
		}
		due := line.AmountMinor
		for _, memoID := range memos {
			if due == 0 {
				break
			}
			take := min(due, remaining[memoID])
			if take == 0 {
				continue
			}
			remaining[memoID] -= take
			due -= take
			plan := byMemo[memoID]
			if plan == nil {
				plan = &creditPlan{memoID: memoID, number: invoices[memoID].Number}
				byMemo[memoID] = plan
			}
			plan.applications = append(plan.applications, &services.CreditMemoApplicationInput{
				InvoiceID:          inv.ID,
				AppliedAmountMinor: take,
			})
		}
		if due > 0 {
			if _, seen := cash[inv.ID]; !seen {
				order = append(order, inv.ID)
			}
			cash[inv.ID] += due
		}
		p.lines = append(p.lines, services.AccountingInboundPostingLine{
			ObjectType:   line.ObjectType,
			ObjectID:     inv.ID,
			ObjectNumber: inv.Number,
			AmountMinor:  line.AmountMinor,
			OpenMinor:    inv.OpenBalanceMinor(),
		})
	}

	for _, id := range order {
		p.cash = append(p.cash, &services.CustomerPaymentApplicationInput{
			InvoiceID:          id,
			AppliedAmountMinor: cash[id],
		})
		p.cashMinor += cash[id]
	}
	for _, memoID := range memos {
		if plan := byMemo[memoID]; plan != nil {
			p.credits = append(p.credits, plan)
		}
		p.lines = append(p.lines, services.AccountingInboundPostingLine{
			ObjectType:   accountingsync.SyncObjectCreditMemo,
			ObjectID:     memoID,
			ObjectNumber: invoices[memoID].Number,
			AmountMinor:  credit[memoID],
			OpenMinor:    invoices[memoID].CreditRemainingMinor(),
		})
	}

	var leftover int64
	for _, amount := range remaining {
		leftover += amount
	}
	return leftover
}

func (s *Service) paymentMethod(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
	change *accountingsync.AccountingInboundChange,
) customerpayment.Method {
	if change.Document.MethodExternalID != "" {
		mappings, err := s.mappings.ListByConnection(
			ctx,
			&repositories.ListAccountingMappingsRequest{
				TenantInfo:   change.TenantInfo(),
				ConnectionID: conn.ID,
				TargetTypes: []accountingsync.MappingTargetType{
					accountingsync.TargetPaymentMethod,
				},
			},
		)
		if err == nil {
			for _, mapping := range mappings {
				method := customerpayment.Method(mapping.TrenovaKey)
				if mapping.State == accountingsync.MappingStateConfirmed &&
					mapping.ExternalID == change.Document.MethodExternalID && method.IsValid() {
					return method
				}
			}
		}
	}
	name := strings.ReplaceAll(strings.TrimSpace(change.Document.MethodName), " ", "")
	for _, method := range customerpayment.AllMethods() {
		if strings.EqualFold(name, string(method)) {
			return method
		}
	}
	return customerpayment.MethodOther
}

func (s *Service) planBillPayment(ctx context.Context, p *postingPlan) error {
	change := p.change
	matched, err := s.matchedDocuments(
		ctx,
		p,
		[]accountingsync.SyncObjectType{
			accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncObjectDriverBill,
		},
		[]accountingsync.InboundDocumentKind{accountingsync.InboundDocBill},
	)
	if err != nil || p.blocked() {
		return err
	}

	paid := make(map[pulid.ID]int64, len(matched))
	kinds := make(map[pulid.ID]accountingsync.SyncObjectType, len(matched))
	for _, line := range matched {
		paid[line.ObjectID] += line.AmountMinor
		kinds[line.ObjectID] = line.ObjectType
	}

	var partyID pulid.ID
	for _, id := range orderedObjects(matched) {
		settlement, ok, planErr := s.planSettlement(ctx, p, id, kinds[id], paid[id])
		if planErr != nil || !ok {
			return planErr
		}
		if !partyID.IsNil() && partyID != settlement.partyID {
			p.block(accountingsync.InboundReasonPartyMismatch, partyMismatchText(p))
			return nil
		}
		partyID = settlement.partyID
		p.settlements = append(p.settlements, settlement)
		p.lines = append(p.lines, services.AccountingInboundPostingLine{
			ObjectType:   kinds[id],
			ObjectID:     id,
			ObjectNumber: settlement.number,
			AmountMinor:  paid[id],
			OpenMinor:    settlement.netMinor,
		})
	}
	change.PartyObjectID = partyID
	p.method = strings.TrimSpace(change.Document.MethodName)
	if p.method == "" {
		p.method = string(customerpayment.MethodOther)
	}
	return nil
}

func (s *Service) planSettlement(
	ctx context.Context,
	p *postingPlan,
	id pulid.ID,
	billType accountingsync.SyncObjectType,
	paidMinor int64,
) (*settlementPlan, bool, error) {
	kind := repositories.PayableCarrier
	payment := accountingsync.SyncObjectCarrierBillPay
	if billType == accountingsync.SyncObjectDriverBill {
		kind = repositories.PayableDriver
		payment = accountingsync.SyncObjectDriverBillPay
	}
	settlement, err := s.payables.GetSettlement(ctx, &repositories.GetPayableSettlementRequest{
		TenantInfo: p.change.TenantInfo(),
		Kind:       kind,
		ID:         id,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			p.block(accountingsync.InboundReasonUnknownDocument, unknownDocumentText(p, 1))
			return nil, false, nil
		}
		return nil, false, err
	}
	switch {
	case settlement.CurrencyCode != "" &&
		!strings.EqualFold(settlement.CurrencyCode, p.change.CurrencyCode):
		p.block(accountingsync.InboundReasonCurrencyMismatch,
			currencyMismatchText(p, settlement.Number, settlement.CurrencyCode))
	case settlement.Voided:
		p.block(accountingsync.InboundReasonAlreadyPaid, settlementVoidedText(settlement.Number))
	case settlement.PaidAt != nil:
		p.block(accountingsync.InboundReasonAlreadyPaid, alreadyPaidText(settlement.Number))
	case paidMinor != settlement.NetMinor:
		p.block(
			accountingsync.InboundReasonPartialBillPayment,
			partialBillText(
				settlement.Number,
				paidMinor,
				settlement.NetMinor,
				p.change.CurrencyCode,
			),
		)
	}
	if p.blocked() {
		return nil, false, nil
	}
	return &settlementPlan{
		kind:       kind,
		id:         id,
		partyID:    settlement.PartyID,
		number:     settlement.Number,
		objectType: payment,
		netMinor:   settlement.NetMinor,
	}, true, nil
}

func (s *Service) checkPeriod(ctx context.Context, p *postingPlan) error {
	tenant := p.change.TenantInfo()
	period, err := s.fiscalPeriods.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: tenant.OrgID,
		BuID:  tenant.BuID,
		Date:  p.paidAt,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			p.block(accountingsync.InboundReasonPeriodNotOpen, noPeriodText(p))
			return nil
		}
		return err
	}
	if period.Status != fiscalperiod.StatusOpen {
		p.block(accountingsync.InboundReasonPeriodNotOpen, periodNotOpenText(p, period.Status))
	}
	return nil
}
