package accountingsyncservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	summaryDocPrefix = "JE "
	openingDocPrefix = "OB "
)

type ledgerLine struct {
	accountID    pulid.ID
	accountLabel string
	accountName  string
	description  string
	netMinor     int64
	party        repositories.LedgerParty
}

type ledgerEntry struct {
	externalAccount string
	accountLabel    string
	description     string
	netMinor        int64
	need            accountingsync.LedgerPartyNeed
	party           repositories.LedgerParty
}

type ledgerEntryKey struct {
	externalAccount string
	kind            repositories.LedgerPartyKind
	partyID         pulid.ID
}

type ledgerLinesRequest struct {
	label string
	lines []ledgerLine
	sum   bool
}

func (s *Service) pushLedger(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	if sess.journals == nil {
		return nil, blocked(
			accountingsync.SyncErrorConfiguration,
			sess.providerName+" cannot receive journal entries from Trenova",
			"Skip this record",
		)
	}
	switch {
	case record.ObjectType == accountingsync.SyncObjectJournalEntry:
		return s.pushJournalEntry(ctx, sess, record)
	case record.IsOpeningBalances():
		return s.pushOpeningBalances(ctx, sess, record)
	default:
		return s.pushJournalDay(ctx, sess, record)
	}
}

func functionalCurrency(sess *pushSession) string {
	if sess.control != nil {
		if code := strings.ToUpper(
			strings.TrimSpace(sess.control.FunctionalCurrencyCode),
		); code != "" {
			return code
		}
	}
	return strings.ToUpper(strings.TrimSpace(sess.conn.ExternalHomeCurrency))
}

func ledgerRoleFor(accountID pulid.ID, control *tenant.AccountingControl) string {
	if control == nil || accountID.IsNil() {
		return ""
	}
	switch accountID {
	case control.DefaultARAccountID:
		return accountingsync.AccountRoleAR
	case control.DefaultRevenueAccountID:
		return accountingsync.AccountRoleRevenue
	case control.DefaultCashAccountID:
		return accountingsync.AccountRoleDeposit
	case control.DefaultWriteOffAccountID:
		return accountingsync.AccountRoleWriteOff
	case control.DefaultAPAccountID, control.DefaultSettlementsPayableAccountID:
		return accountingsync.AccountRoleAP
	case control.DefaultPurchasedTransportationAccountID:
		return accountingsync.AccountRolePurchasedTransportation
	default:
		return ""
	}
}

func journalLines(journal *repositories.LedgerJournal) []ledgerLine {
	lines := make([]ledgerLine, 0, len(journal.Lines))
	for idx := range journal.Lines {
		line := &journal.Lines[idx]
		party := line.Party
		if party.IsZero() {
			party = journal.Party
		}
		description := strings.TrimSpace(line.Description)
		if description == "" {
			description = strings.TrimSpace(journal.Description)
		}
		lines = append(lines, ledgerLine{
			accountID:    line.AccountID,
			accountLabel: glAccountLabel(line.AccountCode, line.AccountName),
			accountName:  glAccountName(line.AccountCode, line.AccountName),
			description:  description,
			netMinor:     line.NetMinor(),
			party:        party,
		})
	}
	return lines
}

func journalNote(journal *repositories.LedgerJournal) string {
	note := trenovaNotePrefix + "journal entry " + journal.EntryNumber
	if journal.IsReversal && journal.ReversalOfNumber != "" {
		note += ", reversing " + journal.ReversalOfNumber
	}
	if source := strings.TrimSpace(journal.SourceDocumentNumber); source != "" {
		note += " (" + source + ")"
	}
	if description := strings.TrimSpace(journal.Description); description != "" {
		note += ": " + description
	}
	return note
}

func (s *Service) pushJournalEntry(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	journal, err := s.ledger.GetJournal(ctx, &repositories.GetLedgerJournalRequest{
		TenantInfo: sess.tenant,
		ID:         record.ObjectID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, blocked(
				accountingsync.SyncErrorValidation,
				documentLabel(record.ObjectType, record.ObjectNumber)+" is not posted in Trenova",
				"Skip this record",
			)
		}
		return nil, err
	}
	label := documentLabel(record.ObjectType, journal.EntryNumber)
	if !accountingsync.JournalSendable(journalentry.EntryType(journal.EntryType)) {
		return nil, &noopError{
			reason: label + " closes or opens a fiscal year, which " + sess.providerName +
				" does on its own",
		}
	}

	currency := functionalCurrency(sess)
	sentDate := record.SentDate(journal.AccountingDate)
	if err = s.checkBooks(sess, currency, sentDate, label); err != nil {
		return nil, err
	}

	res := newResolver(s, sess)
	lines, err := s.ledgerDocLines(ctx, sess, res, &ledgerLinesRequest{
		label: label,
		lines: journalLines(journal),
	})
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, &noopError{reason: label + " has no amounts, so there is nothing to send"}
	}

	rate, err := s.exchangeRate(ctx, sess, &documentRate{
		label:          label,
		currency:       currency,
		documentDate:   journal.AccountingDate,
		accountingDate: journal.AccountingDate,
	})
	if err != nil {
		return nil, err
	}

	doc := &services.AccountingJournalDocument{
		Auth:         sess.auth,
		RequestID:    record.RequestID,
		Kind:         record.ObjectType,
		DocNumber:    ledgerDocNumber(sess, journal.EntryNumber),
		TxnDate:      timeutils.FormatCalendarDate(sentDate, sess.loc),
		CurrencyCode: currency,
		ExchangeRate: rate,
		PrivateNote:  journalNote(journal) + sentDateNote(record, journal.AccountingDate, sess.loc),
		Lines:        lines,
	}
	written, err := sess.journals.CreateJournalEntry(ctx, doc)
	if err != nil {
		return partial(written), err
	}
	return withProviderURL(finishedResult(sess, record, written, doc, res.mappingIDs()))
}

func ledgerDocNumber(sess *pushSession, number string) string {
	number = strings.TrimSpace(number)
	if limit := sess.limits.MaxDocNumberLength; limit > 0 && len([]rune(number)) > limit {
		return ""
	}
	return number
}

func (s *Service) pushJournalDay(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	label := documentLabel(record.ObjectType, record.ObjectNumber)
	if record.DocumentDate == nil {
		return nil, blocked(
			accountingsync.SyncErrorValidation,
			label+" has no date",
			"Skip this record",
		)
	}
	from := timeutils.DayStart(*record.DocumentDate, sess.loc)
	before := timeutils.NextDayStart(from, sess.loc)

	externalID, err := s.dayTarget(ctx, sess, record, label)
	if err != nil {
		return nil, err
	}

	journals, err := s.ledger.ListJournals(ctx, &repositories.ListLedgerJournalsRequest{
		TenantInfo: sess.tenant,
		From:       from,
		Before:     before,
	})
	if err != nil {
		return nil, err
	}
	lines := make([]ledgerLine, 0, len(journals)*2)
	for _, journal := range journals {
		lines = append(lines, journalLines(journal)...)
	}

	currency := functionalCurrency(sess)
	sentDate := record.SentDate(from)
	if err = s.checkBooks(sess, currency, sentDate, label); err != nil {
		return nil, err
	}

	res := newResolver(s, sess)
	docLines, err := s.ledgerDocLines(ctx, sess, res, &ledgerLinesRequest{
		label: label,
		lines: lines,
		sum:   true,
	})
	if err != nil {
		return nil, err
	}
	if len(docLines) == 0 {
		return nil, s.retireDay(ctx, sess, record, externalID, label)
	}

	rate, err := s.exchangeRate(ctx, sess, &documentRate{
		label:          label,
		currency:       currency,
		documentDate:   from,
		accountingDate: from,
	})
	if err != nil {
		return nil, err
	}

	day := timeutils.FormatCalendarDate(from, sess.loc)
	doc := &services.AccountingJournalDocument{
		Auth:         sess.auth,
		RequestID:    record.RequestID,
		ExternalID:   externalID,
		Kind:         record.ObjectType,
		DocNumber:    ledgerDocNumber(sess, summaryDocPrefix+day),
		TxnDate:      timeutils.FormatCalendarDate(sentDate, sess.loc),
		CurrencyCode: currency,
		ExchangeRate: rate,
		PrivateNote: trenovaNotePrefix + "journal entries posted for " + day +
			sentDateNote(record, from, sess.loc),
		Lines: docLines,
	}
	var written *services.AccountingDocumentResult
	if externalID != "" {
		written, err = sess.journals.UpdateJournalEntry(ctx, doc)
	} else {
		written, err = sess.journals.CreateJournalEntry(ctx, doc)
	}
	if err != nil {
		return partial(written), err
	}
	return withProviderURL(finishedResult(sess, record, written, doc, res.mappingIDs()))
}

func (s *Service) retireDay(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
	externalID, label string,
) error {
	if externalID == "" {
		return &noopError{reason: label + " nets to zero, so there is nothing to send"}
	}
	if _, err := sess.journals.DeleteJournalEntry(ctx, &services.AccountingDocumentRef{
		Auth:       sess.auth,
		RequestID:  record.RequestID,
		Kind:       record.ObjectType,
		ExternalID: externalID,
	}); err != nil {
		return err
	}
	return &noopError{
		reason: label + " now nets to zero, so its journal entry was deleted from " +
			sess.providerName,
	}
}

func (s *Service) dayTarget(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
	label string,
) (string, error) {
	if record.Operation != accountingsync.SyncOperationUpdate {
		return "", nil
	}
	existing, err := s.objectRecords(ctx, sess, record.ObjectType, record.ObjectID)
	if err != nil {
		return "", err
	}
	var sent, inFlight *accountingsync.AccountingSyncRecord
	for _, other := range existing {
		if other.ID == record.ID || other.Revision >= record.Revision {
			continue
		}
		switch {
		case other.Status == accountingsync.SyncStatusSynced:
			if sent == nil || other.Revision > sent.Revision {
				sent = other
			}
		case !other.Status.IsFinal():
			if inFlight == nil || other.Revision < inFlight.Revision {
				inFlight = other
			}
		}
	}
	if inFlight != nil {
		s.kick(ctx, sess.tenant, sess.conn.ID)
		return "", waitingOn(inFlight, label)
	}
	if sent == nil {
		return "", nil
	}
	return sent.ExternalID, nil
}

func (s *Service) pushOpeningBalances(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	label := "Opening balances"
	if record.ObjectNumber != "" {
		label += " as of " + record.ObjectNumber
	}
	if sess.conn.SyncStartDate == nil {
		return nil, blocked(
			accountingsync.SyncErrorConfiguration,
			label+" needs the sync start date",
			"Set the start date, then retry",
		)
	}
	start := *sess.conn.SyncStartDate
	dated := timeutils.PreviousDayStart(start, sess.loc)

	res := newResolver(s, sess)
	partyAccounts, err := s.openingPartyAccounts(ctx, sess, res, start)
	if err != nil {
		return nil, err
	}
	balances, err := s.ledger.SumLines(ctx, &repositories.SumLedgerRequest{
		TenantInfo:      sess.tenant,
		Before:          start,
		PartyAccountIDs: partyAccounts,
		IncludeClosing:  true,
	})
	if err != nil {
		return nil, err
	}
	lines := make([]ledgerLine, 0, len(balances))
	for idx := range balances {
		balance := &balances[idx]
		lines = append(lines, ledgerLine{
			accountID:    balance.AccountID,
			accountLabel: glAccountLabel(balance.AccountCode, balance.AccountName),
			accountName:  glAccountName(balance.AccountCode, balance.AccountName),
			netMinor:     balance.NetMinor(),
			party:        balance.Party,
		})
	}

	currency := functionalCurrency(sess)
	sentDate := record.SentDate(dated)
	if err = s.checkBooks(sess, currency, sentDate, label); err != nil {
		return nil, err
	}
	docLines, err := s.ledgerDocLines(ctx, sess, res, &ledgerLinesRequest{
		label: label,
		lines: lines,
		sum:   true,
	})
	if err != nil {
		return nil, err
	}
	day := timeutils.FormatCalendarDate(dated, sess.loc)
	if len(docLines) == 0 {
		return nil, &noopError{
			reason: "Trenova has no balances before " + timeutils.FormatCalendarDate(
				start,
				sess.loc,
			) +
				", so there are no opening balances to send",
		}
	}

	rate, err := s.exchangeRate(ctx, sess, &documentRate{
		label:          label,
		currency:       currency,
		documentDate:   dated,
		accountingDate: dated,
	})
	if err != nil {
		return nil, err
	}
	doc := &services.AccountingJournalDocument{
		Auth:         sess.auth,
		RequestID:    record.RequestID,
		Kind:         record.ObjectType,
		DocNumber:    ledgerDocNumber(sess, openingDocPrefix+day),
		TxnDate:      timeutils.FormatCalendarDate(sentDate, sess.loc),
		CurrencyCode: currency,
		ExchangeRate: rate,
		PrivateNote: trenovaNotePrefix + "opening balances as of " + day +
			sentDateNote(record, dated, sess.loc),
		Lines: docLines,
	}
	written, err := sess.journals.CreateJournalEntry(ctx, doc)
	if err != nil {
		return partial(written), err
	}
	return withProviderURL(finishedResult(sess, record, written, doc, res.mappingIDs()))
}

func (s *Service) openingPartyAccounts(
	ctx context.Context,
	sess *pushSession,
	res *resolver,
	start int64,
) ([]pulid.ID, error) {
	totals, err := s.ledger.SumLines(ctx, &repositories.SumLedgerRequest{
		TenantInfo:     sess.tenant,
		Before:         start,
		IncludeClosing: true,
	})
	if err != nil {
		return nil, err
	}
	lines := make([]ledgerLine, 0, len(totals))
	for idx := range totals {
		lines = append(lines, ledgerLine{
			accountID:    totals[idx].AccountID,
			accountLabel: glAccountLabel(totals[idx].AccountCode, totals[idx].AccountName),
		})
	}
	external, err := s.ledgerAccounts(ctx, res, sess, lines)
	if err != nil {
		return nil, err
	}
	needs, err := s.ledgerPartyNeeds(ctx, sess, external)
	if err != nil {
		return nil, err
	}
	accounts := make([]pulid.ID, 0, 2)
	for accountID, externalID := range external {
		if needs[externalID] != accountingsync.LedgerPartyNone {
			accounts = append(accounts, accountID)
		}
	}
	return accounts, nil
}

func (s *Service) ledgerAccounts(
	ctx context.Context,
	res *resolver,
	sess *pushSession,
	lines []ledgerLine,
) (map[pulid.ID]string, error) {
	external := make(map[pulid.ID]string, len(lines))
	for idx := range lines {
		line := &lines[idx]
		if _, seen := external[line.accountID]; seen {
			continue
		}
		externalID, err := s.accountRef(ctx, res, &accountRefRequest{
			accountID: line.accountID,
			label:     line.accountLabel,
			role:      ledgerRoleFor(line.accountID, sess.control),
			missing:   "A journal line has no GL account",
		})
		if err != nil {
			return nil, err
		}
		external[line.accountID] = externalID
	}
	return external, nil
}

func (s *Service) ledgerPartyNeeds(
	ctx context.Context,
	sess *pushSession,
	external map[pulid.ID]string,
) (map[string]accountingsync.LedgerPartyNeed, error) {
	ids := make([]string, 0, len(external))
	seen := make(map[string]struct{}, len(external))
	for _, externalID := range external {
		if _, ok := seen[externalID]; ok || externalID == "" {
			continue
		}
		seen[externalID] = struct{}{}
		ids = append(ids, externalID)
	}
	needs := make(map[string]accountingsync.LedgerPartyNeed, len(ids))
	if len(ids) == 0 || s.references == nil {
		return needs, nil
	}
	refs, err := s.references.GetByExternalIDs(
		ctx,
		&repositories.GetAccountingReferenceObjectsRequest{
			TenantInfo:   sess.tenant,
			ConnectionID: sess.conn.ID,
			Kind:         accountingsync.ReferenceKindAccount,
			ExternalIDs:  ids,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		needs[ref.ExternalID] = ref.LedgerParty()
	}
	return needs, nil
}

func (s *Service) ledgerDocLines(
	ctx context.Context,
	sess *pushSession,
	res *resolver,
	req *ledgerLinesRequest,
) ([]services.AccountingJournalLine, error) {
	external, err := s.ledgerAccounts(ctx, res, sess, req.lines)
	if err != nil {
		return nil, err
	}
	needs, err := s.ledgerPartyNeeds(ctx, sess, external)
	if err != nil {
		return nil, err
	}

	entries := make([]ledgerEntry, 0, len(req.lines))
	positions := make(map[ledgerEntryKey]int, len(req.lines))
	var total int64
	for idx := range req.lines {
		line := &req.lines[idx]
		total += line.netMinor
		externalAccount := external[line.accountID]
		need := needs[externalAccount]
		party := repositories.LedgerParty{}
		if need != accountingsync.LedgerPartyNone {
			party = line.party
		}
		if !req.sum {
			entries = append(entries, ledgerEntry{
				externalAccount: externalAccount,
				accountLabel:    line.accountLabel,
				description:     line.description,
				netMinor:        line.netMinor,
				need:            need,
				party:           party,
			})
			continue
		}
		key := ledgerEntryKey{externalAccount: externalAccount, kind: party.Kind, partyID: party.ID}
		if pos, ok := positions[key]; ok {
			entries[pos].netMinor += line.netMinor
			continue
		}
		description := line.accountName
		if party.Name != "" {
			description += ", " + party.Name
		}
		positions[key] = len(entries)
		entries = append(entries, ledgerEntry{
			externalAccount: externalAccount,
			accountLabel:    line.accountLabel,
			description:     description,
			netMinor:        line.netMinor,
			need:            need,
			party:           party,
		})
	}
	if total != 0 {
		return nil, blocked(
			accountingsync.SyncErrorValidation,
			req.label+" does not balance: its debits and credits differ by "+
				money.DecimalFromMinor(intutils.AbsDiff(total, 0)).StringFixed(2),
			"Correct the journal entries in Trenova, then retry; or skip this record",
		)
	}

	lines := make([]services.AccountingJournalLine, 0, len(entries))
	for idx := range entries {
		entry := &entries[idx]
		if entry.netMinor == 0 {
			continue
		}
		line := services.AccountingJournalLine{
			Posting:           services.AccountingJournalDebit,
			AccountExternalID: entry.externalAccount,
			Amount:            money.DecimalFromMinor(intutils.AbsDiff(entry.netMinor, 0)),
			Description:       entry.description,
		}
		if entry.netMinor < 0 {
			line.Posting = services.AccountingJournalCredit
		}
		if entry.need != accountingsync.LedgerPartyNone {
			if line.PartyKind, line.PartyExternalID, err = s.ledgerPartyRef(
				ctx,
				sess,
				res,
				req.label,
				entry,
			); err != nil {
				return nil, err
			}
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func (s *Service) ledgerPartyRef(
	ctx context.Context,
	sess *pushSession,
	res *resolver,
	label string,
	entry *ledgerEntry,
) (services.AccountingJournalPartyKind, string, error) {
	wanted := providerKindLabel(accountingsync.ReferenceKindCustomer)
	kind := services.AccountingJournalCustomer
	if entry.need == accountingsync.LedgerPartyVendor {
		wanted = providerKindLabel(accountingsync.ReferenceKindVendor)
		kind = services.AccountingJournalVendor
	}
	account := entry.accountLabel
	if account == "" {
		account = "an account"
	}
	if entry.party.IsZero() {
		return "", "", blocked(
			accountingsync.SyncErrorMapping,
			label+" has a line on "+account+" that needs a "+wanted+
				", and Trenova does not know whose it is",
			"Name the "+wanted+" on the journal line in Trenova, or skip this record",
		)
	}

	var objectType accountingsync.SyncObjectType
	switch {
	case entry.need == accountingsync.LedgerPartyCustomer &&
		entry.party.Kind == repositories.LedgerPartyCustomer:
		objectType = accountingsync.SyncObjectCustomer
	case entry.need == accountingsync.LedgerPartyVendor &&
		entry.party.Kind == repositories.LedgerPartyCarrier:
		objectType = accountingsync.SyncObjectCarrierVendor
	case entry.need == accountingsync.LedgerPartyVendor &&
		entry.party.Kind == repositories.LedgerPartyDriver:
		objectType = accountingsync.SyncObjectDriverVendor
	default:
		return "", "", blocked(
			accountingsync.SyncErrorMapping,
			label+" has a line on "+account+" that needs a "+wanted+", but it belongs to "+
				strings.ToLower(string(entry.party.Kind))+" "+entry.party.Name,
			"Map "+account+" to a "+sess.providerName+" account of the right type, or skip this record",
		)
	}
	externalID, err := s.partyRef(ctx, sess, res, objectType, entry.party.ID)
	if err != nil {
		return "", "", err
	}
	return kind, externalID, nil
}
