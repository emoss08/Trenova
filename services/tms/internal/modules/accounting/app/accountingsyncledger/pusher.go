package accountingsyncledger

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	summaryDocPrefix = "JE "
)

func (p *Pusher) Push(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	if sess.Journals == nil {
		return nil, accountingsyncpush.Blocked(
			accountingsync.SyncErrorConfiguration,
			sess.ProviderName+" cannot receive journal entries from Trenova",
			"Skip this record",
		)
	}
	switch {
	case record.ObjectType == accountingsync.SyncObjectJournalEntry:
		return p.pushJournalEntry(ctx, sess, record)
	case record.IsOpeningBalances():
		return p.pushOpeningBalances(ctx, sess, record)
	default:
		return p.pushJournalDay(ctx, sess, record)
	}
}

func functionalCurrency(sess *accountingsyncpush.Session) string {
	if sess.Control != nil {
		if code := strings.ToUpper(
			strings.TrimSpace(sess.Control.FunctionalCurrencyCode),
		); code != "" {
			return code
		}
	}
	return strings.ToUpper(strings.TrimSpace(sess.Conn.ExternalHomeCurrency))
}

func journalNote(journal *journalentryports.LedgerJournal) string {
	note := accountingsyncpush.NotePrefix + "journal entry " + journal.EntryNumber
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

func (p *Pusher) pushJournalEntry(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	journal, err := p.ledger.GetJournal(ctx, &journalentryports.GetLedgerJournalRequest{
		TenantInfo: sess.Tenant,
		ID:         record.ObjectID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, accountingsyncpush.Blocked(
				accountingsync.SyncErrorValidation,
				accountingsynceffects.DocumentLabel(
					record.ObjectType,
					record.ObjectNumber,
				)+" is not posted in Trenova",
				"Skip this record",
			)
		}
		return nil, err
	}
	label := accountingsynceffects.DocumentLabel(record.ObjectType, journal.EntryNumber)
	if !accountingsync.JournalSendable(
		journalentry.EntryType(journal.EntryType),
		journalentry.EntryType(journal.ReversesEntryType),
	) {
		return nil, &accountingsyncpush.NoopError{
			Reason: label + " closes, opens or reopens a fiscal year, which " + sess.ProviderName +
				" does on its own",
		}
	}

	currency := functionalCurrency(sess)
	sentDate := record.SentDate(journal.AccountingDate)
	if err = sess.CheckBooks(currency, sentDate, label); err != nil {
		return nil, err
	}

	res := p.kit.NewResolver(sess)
	lines, err := p.ledgerDocLines(ctx, sess, res, &ledgerLinesRequest{
		label: label,
		lines: journalLines(journal),
	})
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, &accountingsyncpush.NoopError{
			Reason: label + " has no amounts, so there is nothing to send",
		}
	}

	rate, err := p.kit.ExchangeRate(ctx, sess, &accountingsyncpush.DocumentRate{
		Label:          label,
		Currency:       currency,
		DocumentDate:   journal.AccountingDate,
		AccountingDate: journal.AccountingDate,
	})
	if err != nil {
		return nil, err
	}

	doc := &accountingconnectorports.AccountingJournalDocument{
		Auth:         sess.Auth,
		RequestID:    record.RequestID,
		Kind:         record.ObjectType,
		DocNumber:    ledgerDocNumber(sess, journal.EntryNumber),
		TxnDate:      timeutils.FormatCalendarDate(sentDate, sess.Loc),
		CurrencyCode: currency,
		ExchangeRate: rate,
		PrivateNote: journalNote(
			journal,
		) + accountingsyncpush.SentDateNote(
			record,
			journal.AccountingDate,
			sess.Loc,
		),
		Lines: lines,
	}
	written, err := sess.Journals.CreateJournalEntry(ctx, doc)
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.WithProviderURL(
		accountingsyncpush.Finished(sess, record, written, doc, res.MappingIDs()),
	)
}

func ledgerDocNumber(sess *accountingsyncpush.Session, number string) string {
	number = strings.TrimSpace(number)
	if limit := sess.Limits.MaxDocNumberLength; limit > 0 && len([]rune(number)) > limit {
		return ""
	}
	return number
}

func (p *Pusher) pushJournalDay(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	label := accountingsynceffects.DocumentLabel(record.ObjectType, record.ObjectNumber)
	if record.DocumentDate == nil {
		return nil, accountingsyncpush.Blocked(
			accountingsync.SyncErrorValidation,
			label+" has no date",
			"Skip this record",
		)
	}
	from := timeutils.DayStart(*record.DocumentDate, sess.Loc)
	before := timeutils.NextDayStart(from, sess.Loc)

	var (
		target *accountingsync.AccountingSyncRecord
		err    error
	)
	if record.Operation == accountingsync.SyncOperationUpdate {
		if target, err = p.dayTarget(ctx, sess, record, label); err != nil {
			return nil, err
		}
	}
	externalID := ""
	dated := record
	if target != nil {
		externalID = target.ExternalID
		if record.RedatedTo == nil && target.RedatedTo != nil {
			dated = target
		}
	}

	journals, err := p.ledger.ListJournals(ctx, &journalentryports.ListLedgerJournalsRequest{
		TenantInfo: sess.Tenant,
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
	sentDate := dated.SentDate(from)
	if err = sess.CheckBooks(currency, sentDate, label); err != nil {
		return nil, err
	}

	res := p.kit.NewResolver(sess)
	docLines, err := p.ledgerDocLines(ctx, sess, res, &ledgerLinesRequest{
		label: label,
		lines: lines,
		sum:   true,
	})
	if err != nil {
		return nil, err
	}
	if len(docLines) == 0 {
		return nil, p.retireDay(ctx, sess, record, externalID, label)
	}

	rate, err := p.kit.ExchangeRate(ctx, sess, &accountingsyncpush.DocumentRate{
		Label:          label,
		Currency:       currency,
		DocumentDate:   from,
		AccountingDate: from,
	})
	if err != nil {
		return nil, err
	}

	day := timeutils.FormatCalendarDate(from, sess.Loc)
	doc := &accountingconnectorports.AccountingJournalDocument{
		Auth:         sess.Auth,
		RequestID:    record.RequestID,
		ExternalID:   externalID,
		Kind:         record.ObjectType,
		DocNumber:    ledgerDocNumber(sess, summaryDocPrefix+day),
		TxnDate:      timeutils.FormatCalendarDate(sentDate, sess.Loc),
		CurrencyCode: currency,
		ExchangeRate: rate,
		PrivateNote: accountingsyncpush.NotePrefix + "journal entries posted for " + day +
			accountingsyncpush.SentDateNote(dated, from, sess.Loc),
		Lines: docLines,
	}
	var written *accountingconnectorports.AccountingDocumentResult
	if externalID != "" {
		written, err = sess.Journals.UpdateJournalEntry(ctx, doc)
	} else {
		written, err = sess.Journals.CreateJournalEntry(ctx, doc)
	}
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.WithProviderURL(
		accountingsyncpush.Finished(sess, record, written, doc, res.MappingIDs()),
	)
}

func (p *Pusher) retireDay(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
	externalID, label string,
) error {
	if externalID == "" {
		return &accountingsyncpush.NoopError{
			Reason: label + " nets to zero, so there is nothing to send",
		}
	}
	if _, err := sess.Journals.DeleteJournalEntry(ctx, &accountingconnectorports.AccountingDocumentRef{
		Auth:       sess.Auth,
		RequestID:  record.RequestID,
		Kind:       record.ObjectType,
		ExternalID: externalID,
	}); err != nil {
		return err
	}
	return &accountingsyncpush.NoopError{
		Reason: label + " now nets to zero, so its journal entry was deleted from " +
			sess.ProviderName,
	}
}

func (p *Pusher) dayTarget(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
	label string,
) (*accountingsync.AccountingSyncRecord, error) {
	existing, err := p.kit.ObjectRecords(ctx, sess, record.ObjectType, record.ObjectID)
	if err != nil {
		return nil, err
	}
	var sent, inFlight *accountingsync.AccountingSyncRecord
	for _, other := range existing {
		if other.ID == record.ID {
			continue
		}
		switch {
		case other.Status == accountingsync.SyncStatusSynced:
			if sent == nil || sentLater(other, sent) {
				sent = other
			}
		case other.Revision < record.Revision && !other.Status.IsFinal(),
			other.Status == accountingsync.SyncStatusInFlight:
			if inFlight == nil || other.Revision < inFlight.Revision {
				inFlight = other
			}
		}
	}
	if inFlight != nil {
		p.kit.Kick(ctx, sess)
		return nil, accountingsyncpush.WaitingOn(inFlight, label)
	}
	return sent, nil
}

func sentLater(a, b *accountingsync.AccountingSyncRecord) bool {
	at, bt := int64(0), int64(0)
	if a.SyncedAt != nil {
		at = *a.SyncedAt
	}
	if b.SyncedAt != nil {
		bt = *b.SyncedAt
	}
	if at != bt {
		return at > bt
	}
	return a.Revision > b.Revision
}

type Deps struct {
	Kit        *accountingsyncpush.Kit
	Ledger     journalentryports.AccountingLedgerSource
	References accountingmappingports.AccountingReferenceObjectRepository
}

type Pusher struct {
	kit        *accountingsyncpush.Kit
	ledger     journalentryports.AccountingLedgerSource
	references accountingmappingports.AccountingReferenceObjectRepository
}

func New(d Deps) *Pusher {
	return &Pusher{kit: d.Kit, ledger: d.Ledger, references: d.References}
}
