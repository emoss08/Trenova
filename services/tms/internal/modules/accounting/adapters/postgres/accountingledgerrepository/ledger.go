package accountingledgerrepository

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const journalEntryEntity = "JournalEntry"

var ledgerJournalErrors = dberror.Mapping{Entity: journalEntryEntity}

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
	party          journalentryports.LedgerParty
}

type ledgerSumRow struct {
	AccountID     pulid.ID
	CustomerID    pulid.ID
	SourceEntryID pulid.ID
	DebitMinor    int64
	CreditMinor   int64
}

type ledgerPartyRow struct {
	ObjectID string
	PartyID  pulid.ID
}

type ledgerNameRow struct {
	PartyID pulid.ID
	Name    string
}

type ledgerBalanceKey struct {
	accountID pulid.ID
	kind      journalentryports.LedgerPartyKind
	partyID   pulid.ID
}

type partyLookup struct {
	query   func(objects []string) keel.Projection
	objects []string
}

func NewLedgerSource(p LedgerSourceParams) journalentryports.AccountingLedgerSource {
	return &ledgerSource{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-ledger-source"),
	}
}

func unsentEntryTypes() []journalentry.EntryType {
	return []journalentry.EntryType{journalentry.EntryTypeClosing, journalentry.EntryTypeOpening}
}

func carriedEntryTypes() []journalentry.EntryType {
	return []journalentry.EntryType{journalentry.EntryTypeOpening}
}

func SendableJournals(je *journalentrydb.JournalEntriesTable) keel.Pred {
	return withoutEntryTypes(je, unsentEntryTypes())
}

func withoutEntryTypes(
	je *journalentrydb.JournalEntriesTable,
	excluded []journalentry.EntryType,
) keel.Pred {
	reversed := je.ReversalOf()
	return keel.And(
		je.EntryType.NotIn(excluded...),
		keel.Or(reversed.EntryType.IsNull(), reversed.EntryType.NotIn(excluded...)),
	)
}

func (s *ledgerSource) GetEntryType(
	ctx context.Context,
	req *journalentryports.GetLedgerJournalRequest,
) (journalentry.EntryType, error) {
	return dbtx.Read(ctx, s.db, func(ctx context.Context) (journalentry.EntryType, error) {
		je := journalentrydb.JournalEntries
		entry, err := je.Pick(je.EntryType).Where(je.ID.Eq(req.ID)).One(ctx, s.db.Keel(ctx))
		if err != nil {
			return "", ledgerJournalErrors.Map(err, req.ID.String())
		}
		return entry.EntryType, nil
	})
}

func (s *ledgerSource) GetJournal(
	ctx context.Context,
	req *journalentryports.GetLedgerJournalRequest,
) (*journalentryports.LedgerJournal, error) {
	return dbtx.Write(
		ctx,
		s.db,
		func(ctx context.Context) (*journalentryports.LedgerJournal, error) {
			je := journalentrydb.JournalEntries
			entry, err := je.Where(je.ID.Eq(req.ID), je.IsPosted.IsTrue()).One(ctx, s.db.Keel(ctx))
			if err != nil {
				return nil, ledgerJournalErrors.Map(err, req.ID.String())
			}

			journals, err := s.assemble(ctx, req.TenantInfo, []*journalentry.JournalEntry{entry})
			if err != nil {
				return nil, err
			}
			return journals[0], nil
		},
	)
}

func (s *ledgerSource) ListJournals(
	ctx context.Context,
	req *journalentryports.ListLedgerJournalsRequest,
) ([]*journalentryports.LedgerJournal, error) {
	return dbtx.Write(
		ctx,
		s.db,
		func(ctx context.Context) ([]*journalentryports.LedgerJournal, error) {
			je := journalentrydb.JournalEntries
			entries, err := je.Where(
				je.IsPosted.IsTrue(),
				SendableJournals(je),
				je.AccountingDate.Gte(req.From),
				je.AccountingDate.Lt(req.Before),
			).
				Order(je.AccountingDate.Asc(), je.EntryNumber.Asc(), je.ID.Asc()).
				All(ctx, s.db.Keel(ctx))
			if err != nil {
				s.l.Error("failed to list posted journals", zap.Error(err))
				return nil, fmt.Errorf("list posted journals: %w", err)
			}
			if len(entries) == 0 {
				return []*journalentryports.LedgerJournal{}, nil
			}
			return s.assemble(ctx, req.TenantInfo, entries)
		},
	)
}

func (s *ledgerSource) SumLines(
	ctx context.Context,
	req *journalentryports.SumLedgerRequest,
) ([]journalentryports.LedgerAccountBalance, error) {
	return dbtx.Read(
		ctx,
		s.db,
		func(ctx context.Context) ([]journalentryports.LedgerAccountBalance, error) {
			jel := journalentrydb.JournalEntryLines
			je := jel.JournalEntry()
			query := postedLines(&postedLinesFilter{
				from:           req.From,
				before:         &req.Before,
				includeClosing: req.IncludeClosing,
			})

			customer := keel.Text("")
			sourceEntry := customer
			if len(req.PartyAccountIDs) > 0 {
				party := jel.GLAccountID.In(req.PartyAccountIDs...)
				customer = keel.CaseString().When(party, jel.CustomerID).End().Coalesce("")
				sourceEntry = keel.CaseString().
					When(party, je.ReversalOfID.Coalesce(je.ID)).
					End().
					Coalesce("")
			}

			rows, err := keel.Into[ledgerSumRow](ctx, s.db.Keel(ctx), query.
				Pick(
					jel.GLAccountID,
					customer,
					sourceEntry,
					jel.DebitAmount.Sum(),
					jel.CreditAmount.Sum(),
				).
				Group(jel.GLAccountID, customer, sourceEntry))
			if err != nil {
				s.l.Error("failed to sum posted journal lines", zap.Error(err))
				return nil, fmt.Errorf("sum posted journal lines: %w", err)
			}
			if len(rows) == 0 {
				return []journalentryports.LedgerAccountBalance{}, nil
			}
			return s.balancesOf(ctx, req.TenantInfo, rows)
		},
	)
}

func (s *ledgerSource) balancesOf(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rows []ledgerSumRow,
) ([]journalentryports.LedgerAccountBalance, error) {
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
	out := make([]journalentryports.LedgerAccountBalance, 0, len(rows))
	for idx := range rows {
		row := &rows[idx]
		party := journalentryports.LedgerParty{}
		switch {
		case !row.CustomerID.IsNil():
			party = journalentryports.LedgerParty{
				Kind: journalentryports.LedgerPartyCustomer,
				ID:   row.CustomerID,
			}
		case !row.SourceEntryID.IsNil():
			party = origins[row.SourceEntryID].party
		}
		key := ledgerBalanceKey{accountID: row.AccountID, kind: party.Kind, partyID: party.ID}
		pos, seen := byKey[key]
		if !seen {
			account := accounts[row.AccountID]
			out = append(out, journalentryports.LedgerAccountBalance{
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
	slices.SortFunc(out, func(a, b journalentryports.LedgerAccountBalance) int {
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
	req *journalentryports.ListLedgerAccountsRequest,
) ([]journalentryports.LedgerAccount, error) {
	return dbtx.Read(
		ctx,
		s.db,
		func(ctx context.Context) ([]journalentryports.LedgerAccount, error) {
			jel := journalentrydb.JournalEntryLines
			gla := glaccountdb.GLAccounts
			posted := postedLines(&postedLinesFilter{from: req.Since}).
				Pick(jel.ID).
				Where(jel.GLAccountID.EqCol(gla.ID))
			ids, err := keel.Into[pulid.ID](
				ctx,
				s.db.Keel(ctx),
				gla.Pick(gla.ID).Where(keel.Exists(posted)),
			)
			if err != nil {
				s.l.Error("failed to list accounts with posted lines", zap.Error(err))
				return nil, fmt.Errorf("list accounts with posted lines: %w", err)
			}
			if len(ids) == 0 {
				return []journalentryports.LedgerAccount{}, nil
			}

			accounts, err := s.loadAccounts(ctx, req.TenantInfo, ids)
			if err != nil {
				return nil, err
			}
			out := make([]journalentryports.LedgerAccount, 0, len(accounts))
			for _, account := range accounts {
				out = append(out, account)
			}
			slices.SortFunc(out, func(a, b journalentryports.LedgerAccount) int {
				return cmp.Or(cmp.Compare(a.Code, b.Code), cmp.Compare(a.ID, b.ID))
			})
			return out, nil
		},
	)
}

type postedLinesFilter struct {
	from           *int64
	before         *int64
	includeClosing bool
}

func postedLines(filter *postedLinesFilter) *keel.Query[*journalentry.JournalEntryLine] {
	jel := journalentrydb.JournalEntryLines
	je := jel.JournalEntry()
	excluded := unsentEntryTypes()
	if filter.includeClosing {
		excluded = carriedEntryTypes()
	}

	preds := []keel.Pred{je.IsPosted.IsTrue(), withoutEntryTypes(je, excluded)}
	if filter.from != nil {
		preds = append(preds, je.AccountingDate.Gte(*filter.from))
	}
	if filter.before != nil {
		preds = append(preds, je.AccountingDate.Lt(*filter.before))
	}
	return jel.Where(preds...)
}

func (s *ledgerSource) assemble(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entries []*journalentry.JournalEntry,
) ([]*journalentryports.LedgerJournal, error) {
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
	reversed, err := s.reversedEntries(ctx, tenantInfo, reversalOf)
	if err != nil {
		return nil, err
	}
	origins, err := s.resolveOrigins(ctx, tenantInfo, originIDs)
	if err != nil {
		return nil, err
	}

	names := make(partyNames, 3)
	out := make([]*journalentryports.LedgerJournal, 0, len(entries))
	for _, entry := range entries {
		origin := origins[originEntryID(entry)]
		journal := &journalentryports.LedgerJournal{
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
		if original, ok := reversed[entry.ReversalOfID]; ok && entry.IsReversal {
			journal.ReversalOfNumber = original.EntryNumber
			journal.ReversesEntryType = original.EntryType.String()
		}
		names.want(origin.party)

		entryLines := lines[entry.ID]
		journal.Lines = make([]journalentryports.LedgerJournalLine, 0, len(entryLines))
		for _, line := range entryLines {
			entryLine := journalentryports.LedgerJournalLine{
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
				entryLine.Party = journalentryports.LedgerParty{
					Kind: journalentryports.LedgerPartyCustomer,
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
	_ pagination.TenantInfo,
	entryIDs []pulid.ID,
) (map[pulid.ID][]*journalentry.JournalEntryLine, error) {
	return dbtx.Read(
		ctx,
		s.db,
		func(ctx context.Context) (map[pulid.ID][]*journalentry.JournalEntryLine, error) {
			jel := journalentrydb.JournalEntryLines
			lines, err := jel.Where(jel.JournalEntryID.In(entryIDs...)).
				Load(jel.GLAccount()).
				Order(jel.JournalEntryID.Asc(), jel.LineNumber.Asc()).
				All(ctx, s.db.Keel(ctx))
			if err != nil {
				s.l.Error("failed to load journal lines", zap.Error(err))
				return nil, fmt.Errorf("load journal lines: %w", err)
			}

			out := make(map[pulid.ID][]*journalentry.JournalEntryLine, len(entryIDs))
			for _, line := range lines {
				out[line.JournalEntryID] = append(out[line.JournalEntryID], line)
			}
			return out, nil
		},
	)
}

func (s *ledgerSource) reversedEntries(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]*journalentry.JournalEntry, error) {
	return dbtx.Read(
		ctx,
		s.db,
		func(ctx context.Context) (map[pulid.ID]*journalentry.JournalEntry, error) {
			if len(ids) == 0 {
				return map[pulid.ID]*journalentry.JournalEntry{}, nil
			}
			je := journalentrydb.JournalEntries
			entries, err := je.Pick(je.ID, je.EntryNumber, je.EntryType).
				Where(je.ID.In(ids...)).
				All(ctx, s.db.Keel(ctx))
			if err != nil {
				s.l.Error("failed to load reversed journal numbers", zap.Error(err))
				return nil, fmt.Errorf("load reversed journal numbers: %w", err)
			}

			out := make(map[pulid.ID]*journalentry.JournalEntry, len(entries))
			for _, entry := range entries {
				out[entry.ID] = entry
			}
			return out, nil
		},
	)
}

func (s *ledgerSource) resolveOrigins(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entryIDs []pulid.ID,
) (map[pulid.ID]ledgerOrigin, error) {
	return dbtx.Write(ctx, s.db, func(ctx context.Context) (map[pulid.ID]ledgerOrigin, error) {
		if len(entryIDs) == 0 {
			return map[pulid.ID]ledgerOrigin{}, nil
		}
		js := journalsourcedb.Sources
		sources, err := js.Where(
			js.JournalEntryID.In(slices.Compact(slices.Sorted(slices.Values(entryIDs)))...),
		).
			Order(js.CreatedAt.Asc(), js.ID.Asc()).
			All(ctx, s.db.Keel(ctx))
		if err != nil {
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
	})
}

func (s *ledgerSource) originParties(
	ctx context.Context,
	_ pagination.TenantInfo,
	objects map[string][]string,
) (map[string]map[string]journalentryports.LedgerParty, error) {
	out := make(map[string]map[string]journalentryports.LedgerParty, len(objects))

	adjustments, err := s.lookupParties(ctx, &partyLookup{
		query: func(ids []string) keel.Projection {
			ia := invoiceadjustmentdb.InvoiceAdjustments
			return ia.Pick(ia.ID, ia.OriginalInvoiceID).Where(ia.ID.StringCol.In(ids...))
		},
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
		query: func(ids []string) keel.Projection {
			inv := invoicedb.Invoices
			return inv.Pick(inv.ID, inv.CustomerID).Where(inv.ID.StringCol.In(ids...))
		},
		objects: invoiceIDs,
	})
	if err != nil {
		return nil, err
	}
	out[journalsource.ObjectInvoice] = partiesOf(journalentryports.LedgerPartyCustomer, invoices)
	adjusted := make(map[string]pulid.ID, len(adjustments))
	for adjustmentID, invoiceID := range adjustments {
		if customerID, ok := invoices[invoiceID.String()]; ok {
			adjusted[adjustmentID] = customerID
		}
	}
	out[journalsource.ObjectInvoiceAdjustment] = partiesOf(
		journalentryports.LedgerPartyCustomer,
		adjusted,
	)

	lookups := []struct {
		objectType string
		kind       journalentryports.LedgerPartyKind
		lookup     partyLookup
	}{
		{
			objectType: journalsource.ObjectCustomerPayment,
			kind:       journalentryports.LedgerPartyCustomer,
			lookup: partyLookup{
				query: func(ids []string) keel.Projection {
					cp := customerpaymentdb.Payments
					return cp.Pick(cp.ID, cp.CustomerID).Where(cp.ID.StringCol.In(ids...))
				},
			},
		},
		{
			objectType: journalsource.ObjectCarrierSettlement,
			kind:       journalentryports.LedgerPartyCarrier,
			lookup: partyLookup{
				query: func(ids []string) keel.Projection {
					cs := carriersettlementdb.CarrierSettlements
					return cs.Pick(cs.ID, cs.CarrierID).Where(cs.ID.StringCol.In(ids...))
				},
			},
		},
		{
			objectType: journalsource.ObjectDriverSettlement,
			kind:       journalentryports.LedgerPartyDriver,
			lookup: partyLookup{
				query: func(ids []string) keel.Projection {
					ds := driversettlementdb.Settlements
					return ds.Pick(ds.ID, ds.WorkerID).Where(ds.ID.StringCol.In(ids...))
				},
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
	kind journalentryports.LedgerPartyKind,
	found map[string]pulid.ID,
) map[string]journalentryports.LedgerParty {
	out := make(map[string]journalentryports.LedgerParty, len(found))
	for objectID, partyID := range found {
		out[objectID] = journalentryports.LedgerParty{Kind: kind, ID: partyID}
	}
	return out
}

func (s *ledgerSource) lookupParties(
	ctx context.Context,
	lookup *partyLookup,
) (map[string]pulid.ID, error) {
	return dbtx.Read(ctx, s.db, func(ctx context.Context) (map[string]pulid.ID, error) {
		if len(lookup.objects) == 0 {
			return map[string]pulid.ID{}, nil
		}
		rows, err := keel.Into[ledgerPartyRow](ctx, s.db.Keel(ctx), lookup.query(lookup.objects))
		if err != nil {
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
	})
}

type partyNames map[journalentryports.LedgerPartyKind]map[pulid.ID]string

func (n partyNames) want(party journalentryports.LedgerParty) {
	if party.IsZero() {
		return
	}
	if n[party.Kind] == nil {
		n[party.Kind] = make(map[pulid.ID]string, 8)
	}
	n[party.Kind][party.ID] = ""
}

func (n partyNames) of(party journalentryports.LedgerParty) string {
	if party.IsZero() {
		return ""
	}
	return n[party.Kind][party.ID]
}

func (n partyNames) ids(kind journalentryports.LedgerPartyKind) []pulid.ID {
	out := make([]pulid.ID, 0, len(n[kind]))
	for id := range n[kind] {
		out = append(out, id)
	}
	return out
}

func (s *ledgerSource) nameParties(
	ctx context.Context,
	_ pagination.TenantInfo,
	names partyNames,
) error {
	return dbtx.ReadErr(ctx, s.db, func(ctx context.Context) error {
		lookups := []struct {
			kind  journalentryports.LedgerPartyKind
			query func(ids []pulid.ID) keel.Projection
		}{
			{
				kind: journalentryports.LedgerPartyCustomer,
				query: func(ids []pulid.ID) keel.Projection {
					cus := customerdb.Customers
					return cus.Pick(cus.ID, cus.Name).Where(cus.ID.In(ids...))
				},
			},
			{
				kind: journalentryports.LedgerPartyCarrier,
				query: func(ids []pulid.ID) keel.Projection {
					carr := carrierdb.Carriers
					return carr.Pick(carr.ID, carr.Name).Where(carr.ID.In(ids...))
				},
			},
			{
				kind: journalentryports.LedgerPartyDriver,
				query: func(ids []pulid.ID) keel.Projection {
					wrk := workerdb.Workers
					return wrk.Pick(
						wrk.ID,
						keel.ConcatWS(" ", wrk.FirstName, wrk.LastName),
					).Where(wrk.ID.In(ids...))
				},
			},
		}

		for idx := range lookups {
			lookup := &lookups[idx]
			ids := names.ids(lookup.kind)
			if len(ids) == 0 {
				continue
			}
			rows, err := keel.Into[ledgerNameRow](ctx, s.db.Keel(ctx), lookup.query(ids))
			if err != nil {
				s.l.Error("failed to name journal parties", zap.Error(err))
				return fmt.Errorf("name journal parties: %w", err)
			}
			for row := range slices.Values(rows) {
				names[lookup.kind][row.PartyID] = row.Name
			}
		}
		return nil
	})
}

func (s *ledgerSource) loadAccounts(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]journalentryports.LedgerAccount, error) {
	return dbtx.Read(
		ctx,
		s.db,
		func(ctx context.Context) (map[pulid.ID]journalentryports.LedgerAccount, error) {
			gla := glaccountdb.GLAccounts
			accounts, err := gla.Where(gla.ID.In(slices.Compact(slices.Sorted(slices.Values(ids)))...)).
				Load(gla.AccountType()).
				All(ctx, s.db.Keel(ctx))
			if err != nil {
				s.l.Error("failed to load ledger accounts", zap.Error(err))
				return nil, fmt.Errorf("load ledger accounts: %w", err)
			}

			out := make(map[pulid.ID]journalentryports.LedgerAccount, len(accounts))
			for _, account := range accounts {
				entry := journalentryports.LedgerAccount{
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
		},
	)
}
