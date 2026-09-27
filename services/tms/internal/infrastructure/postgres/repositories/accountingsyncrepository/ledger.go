package accountingsyncrepository

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ledgerAccountLabel     = "ledger_account_id"
	ledgerCustomerLabel    = "ledger_customer_id"
	ledgerSourceEntryLabel = "ledger_source_entry_id"
	ledgerDebitLabel       = "ledger_debit_minor"
	ledgerCreditLabel      = "ledger_credit_minor"
	ledgerObjectLabel      = "ledger_object_id"
	ledgerPartyLabel       = "ledger_party_id"
	ledgerNameLabel        = "ledger_party_name"
	journalEntryEntity     = "JournalEntry"
)

type LedgerSourceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type ledgerSource struct {
	db *postgres.Connection
	l  *zap.Logger
}

type ledgerOrigin struct {
	objectType     string
	objectID       string
	documentNumber string
	party          repositories.LedgerParty
}

type ledgerSumRow struct {
	AccountID     pulid.ID `bun:"ledger_account_id"`
	CustomerID    pulid.ID `bun:"ledger_customer_id"`
	SourceEntryID pulid.ID `bun:"ledger_source_entry_id"`
	DebitMinor    int64    `bun:"ledger_debit_minor"`
	CreditMinor   int64    `bun:"ledger_credit_minor"`
}

type ledgerPartyRow struct {
	ObjectID string   `bun:"ledger_object_id"`
	PartyID  pulid.ID `bun:"ledger_party_id"`
}

type ledgerNameRow struct {
	PartyID pulid.ID `bun:"ledger_party_id"`
	Name    string   `bun:"ledger_party_name"`
}

type ledgerBalanceKey struct {
	accountID pulid.ID
	kind      repositories.LedgerPartyKind
	partyID   pulid.ID
}

type partyLookup struct {
	model   any
	id      buncolgen.Column
	party   buncolgen.Column
	tenant  func(*bun.SelectQuery) *bun.SelectQuery
	objects []string
}

func NewLedgerSource(p LedgerSourceParams) repositories.AccountingLedgerSource {
	return &ledgerSource{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-ledger-source"),
	}
}

func unsentEntryTypes() []journalentry.EntryType {
	return []journalentry.EntryType{journalentry.EntryTypeClosing, journalentry.EntryTypeOpening}
}

func (s *ledgerSource) GetJournal(
	ctx context.Context,
	req *repositories.GetLedgerJournalRequest,
) (*repositories.LedgerJournal, error) {
	cols := buncolgen.JournalEntryColumns
	entry := new(journalentry.JournalEntry)
	if err := s.db.DBForContext(ctx).
		NewSelect().
		Model(entry).
		Apply(buncolgen.JournalEntryApplyTenant(req.TenantInfo)).
		Where(cols.ID.Eq(), req.ID).
		Where(cols.IsPosted.IsTrue()).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, journalEntryEntity)
	}

	journals, err := s.assemble(ctx, req.TenantInfo, []*journalentry.JournalEntry{entry})
	if err != nil {
		return nil, err
	}
	return journals[0], nil
}

func (s *ledgerSource) ListJournals(
	ctx context.Context,
	req *repositories.ListLedgerJournalsRequest,
) ([]*repositories.LedgerJournal, error) {
	cols := buncolgen.JournalEntryColumns
	entries := make([]*journalentry.JournalEntry, 0, 16)
	if err := s.db.DBForContext(ctx).
		NewSelect().
		Model(&entries).
		Apply(buncolgen.JournalEntryApplyTenant(req.TenantInfo)).
		Where(cols.IsPosted.IsTrue()).
		Where(cols.EntryType.NotIn(), bun.List(unsentEntryTypes())).
		Where(cols.AccountingDate.Gte(), req.From).
		Where(cols.AccountingDate.Lt(), req.Before).
		Order(cols.AccountingDate.OrderAsc(), cols.EntryNumber.OrderAsc(), cols.ID.OrderAsc()).
		Scan(ctx); err != nil {
		s.l.Error("failed to list posted journals", zap.Error(err))
		return nil, fmt.Errorf("list posted journals: %w", err)
	}
	if len(entries) == 0 {
		return []*repositories.LedgerJournal{}, nil
	}
	return s.assemble(ctx, req.TenantInfo, entries)
}

func (s *ledgerSource) SumLines(
	ctx context.Context,
	req *repositories.SumLedgerRequest,
) ([]repositories.LedgerAccountBalance, error) {
	lines := buncolgen.JournalEntryLineColumns
	entries := buncolgen.JournalEntryColumns
	query := s.postedLines(ctx, req.TenantInfo, req.From, &req.Before).
		ColumnExpr(lines.GLAccountID.As(ledgerAccountLabel)).
		ColumnExpr(buncolgen.Sum(lines.DebitAmount, ledgerDebitLabel)).
		ColumnExpr(buncolgen.Sum(lines.CreditAmount, ledgerCreditLabel))
	if len(req.PartyAccountIDs) > 0 {
		query = query.
			ColumnExpr(
				buncolgen.Expr("CASE WHEN {0} IN (?) THEN {1} END AS "+ledgerCustomerLabel,
					lines.GLAccountID, lines.CustomerID),
				bun.List(req.PartyAccountIDs),
			).
			ColumnExpr(
				buncolgen.Expr("CASE WHEN {0} IN (?) THEN COALESCE({1}, {2}) END AS "+
					ledgerSourceEntryLabel, lines.GLAccountID, entries.ReversalOfID, entries.ID),
				bun.List(req.PartyAccountIDs),
			).
			GroupExpr(ledgerAccountLabel + ", " + ledgerCustomerLabel + ", " + ledgerSourceEntryLabel)
	} else {
		query = query.GroupExpr(ledgerAccountLabel)
	}

	rows := make([]ledgerSumRow, 0, 64)
	if err := query.Scan(ctx, &rows); err != nil {
		s.l.Error("failed to sum posted journal lines", zap.Error(err))
		return nil, fmt.Errorf("sum posted journal lines: %w", err)
	}
	if len(rows) == 0 {
		return []repositories.LedgerAccountBalance{}, nil
	}
	return s.balancesOf(ctx, req.TenantInfo, rows)
}

func (s *ledgerSource) balancesOf(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rows []ledgerSumRow,
) ([]repositories.LedgerAccountBalance, error) {
	sourceEntries := make([]pulid.ID, 0, len(rows))
	accountIDs := make([]pulid.ID, 0, len(rows))
	for idx := range rows {
		accountIDs = append(accountIDs, rows[idx].AccountID)
		if rows[idx].CustomerID.IsNil() && !rows[idx].SourceEntryID.IsNil() {
			sourceEntries = append(sourceEntries, rows[idx].SourceEntryID)
		}
	}
	accounts, err := s.loadAccounts(ctx, tenantInfo, accountIDs)
	if err != nil {
		return nil, err
	}
	origins, err := s.resolveOrigins(ctx, tenantInfo, sourceEntries)
	if err != nil {
		return nil, err
	}

	names := make(partyNames, 3)
	byKey := make(map[ledgerBalanceKey]int, len(rows))
	out := make([]repositories.LedgerAccountBalance, 0, len(rows))
	for idx := range rows {
		row := &rows[idx]
		party := repositories.LedgerParty{}
		switch {
		case !row.CustomerID.IsNil():
			party = repositories.LedgerParty{
				Kind: repositories.LedgerPartyCustomer,
				ID:   row.CustomerID,
			}
		case !row.SourceEntryID.IsNil():
			party = origins[row.SourceEntryID].party
		}
		key := ledgerBalanceKey{accountID: row.AccountID, kind: party.Kind, partyID: party.ID}
		pos, seen := byKey[key]
		if !seen {
			account := accounts[row.AccountID]
			out = append(out, repositories.LedgerAccountBalance{
				AccountID:   row.AccountID,
				AccountCode: account.Code,
				AccountName: account.Name,
				Category:    account.Category,
				Party:       party,
			})
			pos = len(out) - 1
			byKey[key] = pos
			names.want(party)
		}
		out[pos].DebitMinor += row.DebitMinor
		out[pos].CreditMinor += row.CreditMinor
	}

	if err = s.nameParties(ctx, tenantInfo, names); err != nil {
		return nil, err
	}
	for idx := range out {
		out[idx].Party.Name = names.of(out[idx].Party)
	}
	slices.SortFunc(out, func(a, b repositories.LedgerAccountBalance) int {
		return cmp.Or(
			cmp.Compare(a.AccountCode, b.AccountCode),
			cmp.Compare(a.AccountID, b.AccountID),
			cmp.Compare(a.Party.Kind, b.Party.Kind),
			cmp.Compare(a.Party.ID, b.Party.ID),
		)
	})
	return out, nil
}

func (s *ledgerSource) ListActiveAccounts(
	ctx context.Context,
	req *repositories.ListLedgerAccountsRequest,
) ([]repositories.LedgerAccount, error) {
	lines := buncolgen.JournalEntryLineColumns
	ids := make([]pulid.ID, 0, 64)
	if err := s.postedLines(ctx, req.TenantInfo, req.Since, nil).
		Distinct().
		ColumnExpr(lines.GLAccountID.As(ledgerAccountLabel)).
		Scan(ctx, &ids); err != nil {
		s.l.Error("failed to list accounts with posted lines", zap.Error(err))
		return nil, fmt.Errorf("list accounts with posted lines: %w", err)
	}
	if len(ids) == 0 {
		return []repositories.LedgerAccount{}, nil
	}

	accounts, err := s.loadAccounts(ctx, req.TenantInfo, ids)
	if err != nil {
		return nil, err
	}
	out := make([]repositories.LedgerAccount, 0, len(accounts))
	for _, account := range accounts {
		out = append(out, account)
	}
	slices.SortFunc(out, func(a, b repositories.LedgerAccount) int {
		return cmp.Or(cmp.Compare(a.Code, b.Code), cmp.Compare(a.ID, b.ID))
	})
	return out, nil
}

func (s *ledgerSource) postedLines(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	from, before *int64,
) *bun.SelectQuery {
	lines := buncolgen.JournalEntryLineColumns
	entries := buncolgen.JournalEntryColumns
	query := s.db.DBForContext(ctx).
		NewSelect().
		Model((*journalentry.JournalEntryLine)(nil)).
		Join(joinOn(
			buncolgen.JournalEntryTable,
			buncolgen.JournalEntryTable.Alias,
			entries.ID.EqColumn(lines.JournalEntryID),
			entries.OrganizationID.EqColumn(lines.OrganizationID),
			entries.BusinessUnitID.EqColumn(lines.BusinessUnitID),
		)).
		Apply(buncolgen.JournalEntryLineApplyTenant(tenantInfo)).
		Where(entries.IsPosted.IsTrue()).
		Where(entries.EntryType.NotIn(), bun.List(unsentEntryTypes()))
	if from != nil {
		query = query.Where(entries.AccountingDate.Gte(), *from)
	}
	if before != nil {
		query = query.Where(entries.AccountingDate.Lt(), *before)
	}
	return query
}

func (s *ledgerSource) assemble(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entries []*journalentry.JournalEntry,
) ([]*repositories.LedgerJournal, error) {
	entryIDs := make([]pulid.ID, 0, len(entries))
	originIDs := make([]pulid.ID, 0, len(entries))
	reversalOf := make([]pulid.ID, 0, 4)
	for _, entry := range entries {
		entryIDs = append(entryIDs, entry.ID)
		originIDs = append(originIDs, originEntryID(entry))
		if entry.IsReversal && !entry.ReversalOfID.IsNil() {
			reversalOf = append(reversalOf, entry.ReversalOfID)
		}
	}

	lines, err := s.loadLines(ctx, tenantInfo, entryIDs)
	if err != nil {
		return nil, err
	}
	numbers, err := s.entryNumbers(ctx, tenantInfo, reversalOf)
	if err != nil {
		return nil, err
	}
	origins, err := s.resolveOrigins(ctx, tenantInfo, originIDs)
	if err != nil {
		return nil, err
	}

	names := make(partyNames, 3)
	out := make([]*repositories.LedgerJournal, 0, len(entries))
	for _, entry := range entries {
		origin := origins[originEntryID(entry)]
		journal := &repositories.LedgerJournal{
			ID:                   entry.ID,
			EntryNumber:          entry.EntryNumber,
			EntryType:            entry.EntryType.String(),
			Description:          entry.Description,
			AccountingDate:       entry.AccountingDate,
			IsReversal:           entry.IsReversal,
			SourceObjectType:     origin.objectType,
			SourceObjectID:       origin.objectID,
			SourceDocumentNumber: origin.documentNumber,
			Party:                origin.party,
		}
		if entry.PostedAt != nil {
			journal.PostedAt = *entry.PostedAt
		}
		if entry.IsReversal {
			journal.ReversalOfNumber = numbers[entry.ReversalOfID]
		}
		names.want(origin.party)

		entryLines := lines[entry.ID]
		journal.Lines = make([]repositories.LedgerJournalLine, 0, len(entryLines))
		for _, line := range entryLines {
			entryLine := repositories.LedgerJournalLine{
				AccountID:   line.GLAccountID,
				DebitMinor:  line.DebitAmount,
				CreditMinor: line.CreditAmount,
				Description: line.Description,
			}
			if line.GLAccount != nil {
				entryLine.AccountCode = line.GLAccount.AccountCode
				entryLine.AccountName = line.GLAccount.Name
			}
			if !line.CustomerID.IsNil() {
				entryLine.Party = repositories.LedgerParty{
					Kind: repositories.LedgerPartyCustomer,
					ID:   line.CustomerID,
				}
				names.want(entryLine.Party)
			}
			journal.Lines = append(journal.Lines, entryLine)
		}
		out = append(out, journal)
	}

	if err = s.nameParties(ctx, tenantInfo, names); err != nil {
		return nil, err
	}
	for _, journal := range out {
		journal.Party.Name = names.of(journal.Party)
		for idx := range journal.Lines {
			journal.Lines[idx].Party.Name = names.of(journal.Lines[idx].Party)
		}
	}
	return out, nil
}

func originEntryID(entry *journalentry.JournalEntry) pulid.ID {
	if entry.IsReversal && !entry.ReversalOfID.IsNil() {
		return entry.ReversalOfID
	}
	return entry.ID
}

func (s *ledgerSource) loadLines(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entryIDs []pulid.ID,
) (map[pulid.ID][]*journalentry.JournalEntryLine, error) {
	cols := buncolgen.JournalEntryLineColumns
	lines := make([]*journalentry.JournalEntryLine, 0, len(entryIDs)*4)
	if err := s.db.DBForContext(ctx).
		NewSelect().
		Model(&lines).
		Relation(buncolgen.JournalEntryLineRelations.GLAccount).
		Apply(buncolgen.JournalEntryLineApplyTenant(tenantInfo)).
		Where(cols.JournalEntryID.In(), bun.List(entryIDs)).
		Order(cols.JournalEntryID.OrderAsc(), cols.LineNumber.OrderAsc()).
		Scan(ctx); err != nil {
		s.l.Error("failed to load journal lines", zap.Error(err))
		return nil, fmt.Errorf("load journal lines: %w", err)
	}

	out := make(map[pulid.ID][]*journalentry.JournalEntryLine, len(entryIDs))
	for _, line := range lines {
		out[line.JournalEntryID] = append(out[line.JournalEntryID], line)
	}
	return out, nil
}

func (s *ledgerSource) entryNumbers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	if len(ids) == 0 {
		return map[pulid.ID]string{}, nil
	}
	cols := buncolgen.JournalEntryColumns
	entries := make([]*journalentry.JournalEntry, 0, len(ids))
	if err := s.db.DBForContext(ctx).
		NewSelect().
		Model(&entries).
		Column(cols.ID.Bare(), cols.EntryNumber.Bare()).
		Apply(buncolgen.JournalEntryApplyTenant(tenantInfo)).
		Where(cols.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		s.l.Error("failed to load reversed journal numbers", zap.Error(err))
		return nil, fmt.Errorf("load reversed journal numbers: %w", err)
	}

	out := make(map[pulid.ID]string, len(entries))
	for _, entry := range entries {
		out[entry.ID] = entry.EntryNumber
	}
	return out, nil
}

func (s *ledgerSource) resolveOrigins(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entryIDs []pulid.ID,
) (map[pulid.ID]ledgerOrigin, error) {
	if len(entryIDs) == 0 {
		return map[pulid.ID]ledgerOrigin{}, nil
	}
	cols := buncolgen.SourceColumns
	sources := make([]*journalsource.Source, 0, len(entryIDs))
	if err := s.db.DBForContext(ctx).
		NewSelect().
		Model(&sources).
		Apply(buncolgen.SourceApplyTenant(tenantInfo)).
		Where(cols.JournalEntryID.In(), bun.List(slices.Compact(slices.Sorted(slices.Values(entryIDs))))).
		Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
		Scan(ctx); err != nil {
		s.l.Error("failed to load journal sources", zap.Error(err))
		return nil, fmt.Errorf("load journal sources: %w", err)
	}

	origins := make(map[pulid.ID]ledgerOrigin, len(sources))
	objects := make(map[string][]string, 5)
	for _, source := range sources {
		if _, seen := origins[source.JournalEntryID]; seen {
			continue
		}
		origins[source.JournalEntryID] = ledgerOrigin{
			objectType:     source.SourceObjectType,
			objectID:       source.SourceObjectID,
			documentNumber: source.SourceDocumentNumber,
		}
		objects[source.SourceObjectType] = append(
			objects[source.SourceObjectType],
			source.SourceObjectID,
		)
	}

	parties, err := s.originParties(ctx, tenantInfo, objects)
	if err != nil {
		return nil, err
	}
	for entryID, origin := range origins {
		origin.party = parties[origin.objectType][origin.objectID]
		origins[entryID] = origin
	}
	return origins, nil
}

func (s *ledgerSource) originParties(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	objects map[string][]string,
) (map[string]map[string]repositories.LedgerParty, error) {
	out := make(map[string]map[string]repositories.LedgerParty, len(objects))

	adjustments, err := s.lookupParties(ctx, &partyLookup{
		model:   (*invoiceadjustment.InvoiceAdjustment)(nil),
		id:      buncolgen.InvoiceAdjustmentColumns.ID,
		party:   buncolgen.InvoiceAdjustmentColumns.OriginalInvoiceID,
		tenant:  buncolgen.InvoiceAdjustmentApplyTenant(tenantInfo),
		objects: objects[journalsource.ObjectInvoiceAdjustment],
	})
	if err != nil {
		return nil, err
	}
	invoiceIDs := slices.Clone(objects[journalsource.ObjectInvoice])
	for _, invoiceID := range adjustments {
		invoiceIDs = append(invoiceIDs, invoiceID.String())
	}
	invoices, err := s.lookupParties(ctx, &partyLookup{
		model:   (*invoice.Invoice)(nil),
		id:      buncolgen.InvoiceColumns.ID,
		party:   buncolgen.InvoiceColumns.CustomerID,
		tenant:  buncolgen.InvoiceApplyTenant(tenantInfo),
		objects: invoiceIDs,
	})
	if err != nil {
		return nil, err
	}
	out[journalsource.ObjectInvoice] = partiesOf(repositories.LedgerPartyCustomer, invoices)
	adjusted := make(map[string]pulid.ID, len(adjustments))
	for adjustmentID, invoiceID := range adjustments {
		if customerID, ok := invoices[invoiceID.String()]; ok {
			adjusted[adjustmentID] = customerID
		}
	}
	out[journalsource.ObjectInvoiceAdjustment] = partiesOf(
		repositories.LedgerPartyCustomer,
		adjusted,
	)

	lookups := []struct {
		objectType string
		kind       repositories.LedgerPartyKind
		lookup     partyLookup
	}{
		{
			objectType: journalsource.ObjectCustomerPayment,
			kind:       repositories.LedgerPartyCustomer,
			lookup: partyLookup{
				model:  (*customerpayment.Payment)(nil),
				id:     buncolgen.PaymentColumns.ID,
				party:  buncolgen.PaymentColumns.CustomerID,
				tenant: buncolgen.PaymentApplyTenant(tenantInfo),
			},
		},
		{
			objectType: journalsource.ObjectCarrierSettlement,
			kind:       repositories.LedgerPartyCarrier,
			lookup: partyLookup{
				model:  (*carriersettlement.CarrierSettlement)(nil),
				id:     buncolgen.CarrierSettlementColumns.ID,
				party:  buncolgen.CarrierSettlementColumns.CarrierID,
				tenant: buncolgen.CarrierSettlementApplyTenant(tenantInfo),
			},
		},
		{
			objectType: journalsource.ObjectDriverSettlement,
			kind:       repositories.LedgerPartyDriver,
			lookup: partyLookup{
				model:  (*driversettlement.Settlement)(nil),
				id:     buncolgen.SettlementColumns.ID,
				party:  buncolgen.SettlementColumns.WorkerID,
				tenant: buncolgen.SettlementApplyTenant(tenantInfo),
			},
		},
	}
	for idx := range lookups {
		entry := &lookups[idx]
		entry.lookup.objects = objects[entry.objectType]
		found, lookupErr := s.lookupParties(ctx, &entry.lookup)
		if lookupErr != nil {
			return nil, lookupErr
		}
		out[entry.objectType] = partiesOf(entry.kind, found)
	}
	return out, nil
}

func partiesOf(
	kind repositories.LedgerPartyKind,
	found map[string]pulid.ID,
) map[string]repositories.LedgerParty {
	out := make(map[string]repositories.LedgerParty, len(found))
	for objectID, partyID := range found {
		out[objectID] = repositories.LedgerParty{Kind: kind, ID: partyID}
	}
	return out
}

func (s *ledgerSource) lookupParties(
	ctx context.Context,
	lookup *partyLookup,
) (map[string]pulid.ID, error) {
	if len(lookup.objects) == 0 {
		return map[string]pulid.ID{}, nil
	}
	rows := make([]ledgerPartyRow, 0, len(lookup.objects))
	if err := s.db.DBForContext(ctx).
		NewSelect().
		Model(lookup.model).
		ColumnExpr(lookup.id.As(ledgerObjectLabel)).
		ColumnExpr(lookup.party.As(ledgerPartyLabel)).
		Apply(lookup.tenant).
		Where(lookup.id.In(), bun.List(lookup.objects)).
		Scan(ctx, &rows); err != nil {
		s.l.Error("failed to resolve journal parties", zap.Error(err))
		return nil, fmt.Errorf("resolve journal parties: %w", err)
	}

	out := make(map[string]pulid.ID, len(rows))
	for idx := range rows {
		if !rows[idx].PartyID.IsNil() {
			out[rows[idx].ObjectID] = rows[idx].PartyID
		}
	}
	return out, nil
}

type partyNames map[repositories.LedgerPartyKind]map[pulid.ID]string

func (n partyNames) want(party repositories.LedgerParty) {
	if party.IsZero() {
		return
	}
	if n[party.Kind] == nil {
		n[party.Kind] = make(map[pulid.ID]string, 8)
	}
	n[party.Kind][party.ID] = ""
}

func (n partyNames) of(party repositories.LedgerParty) string {
	if party.IsZero() {
		return ""
	}
	return n[party.Kind][party.ID]
}

func (n partyNames) ids(kind repositories.LedgerPartyKind) []pulid.ID {
	out := make([]pulid.ID, 0, len(n[kind]))
	for id := range n[kind] {
		out = append(out, id)
	}
	return out
}

func (s *ledgerSource) nameParties(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	names partyNames,
) error {
	workers := buncolgen.WorkerColumns
	lookups := []struct {
		kind   repositories.LedgerPartyKind
		model  any
		id     buncolgen.Column
		name   string
		tenant func(*bun.SelectQuery) *bun.SelectQuery
	}{
		{
			kind:   repositories.LedgerPartyCustomer,
			model:  (*customer.Customer)(nil),
			id:     buncolgen.CustomerColumns.ID,
			name:   buncolgen.CustomerColumns.Name.As(ledgerNameLabel),
			tenant: buncolgen.CustomerApplyTenant(tenantInfo),
		},
		{
			kind:   repositories.LedgerPartyCarrier,
			model:  (*carrier.Carrier)(nil),
			id:     buncolgen.CarrierColumns.ID,
			name:   buncolgen.CarrierColumns.Name.As(ledgerNameLabel),
			tenant: buncolgen.CarrierApplyTenant(tenantInfo),
		},
		{
			kind:  repositories.LedgerPartyDriver,
			model: (*worker.Worker)(nil),
			id:    workers.ID,
			name: buncolgen.Expr("CONCAT_WS(' ', {0}, {1}) AS "+ledgerNameLabel,
				workers.FirstName, workers.LastName),
			tenant: buncolgen.WorkerApplyTenant(tenantInfo),
		},
	}

	for idx := range lookups {
		lookup := &lookups[idx]
		ids := names.ids(lookup.kind)
		if len(ids) == 0 {
			continue
		}
		rows := make([]ledgerNameRow, 0, len(ids))
		if err := s.db.DBForContext(ctx).
			NewSelect().
			Model(lookup.model).
			ColumnExpr(lookup.id.As(ledgerPartyLabel)).
			ColumnExpr(lookup.name).
			Apply(lookup.tenant).
			Where(lookup.id.In(), bun.List(ids)).
			Scan(ctx, &rows); err != nil {
			s.l.Error("failed to name journal parties", zap.Error(err))
			return fmt.Errorf("name journal parties: %w", err)
		}
		for row := range slices.Values(rows) {
			names[lookup.kind][row.PartyID] = row.Name
		}
	}
	return nil
}

func (s *ledgerSource) loadAccounts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]repositories.LedgerAccount, error) {
	cols := buncolgen.GLAccountColumns
	accounts := make([]*glaccount.GLAccount, 0, len(ids))
	if err := s.db.DBForContext(ctx).
		NewSelect().
		Model(&accounts).
		Relation(buncolgen.GLAccountRelations.AccountType).
		Apply(buncolgen.GLAccountApplyTenant(tenantInfo)).
		Where(cols.ID.In(), bun.List(slices.Compact(slices.Sorted(slices.Values(ids))))).
		Scan(ctx); err != nil {
		s.l.Error("failed to load ledger accounts", zap.Error(err))
		return nil, fmt.Errorf("load ledger accounts: %w", err)
	}

	out := make(map[pulid.ID]repositories.LedgerAccount, len(accounts))
	for _, account := range accounts {
		entry := repositories.LedgerAccount{
			ID:   account.ID,
			Code: account.AccountCode,
			Name: account.Name,
		}
		if account.AccountType != nil {
			entry.Category = string(account.AccountType.Category)
		}
		out[account.ID] = entry
	}
	return out, nil
}
