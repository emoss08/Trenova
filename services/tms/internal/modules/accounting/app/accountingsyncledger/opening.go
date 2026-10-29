package accountingsyncledger

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	openingDocPrefix = "OB "
)

func (p *Pusher) pushOpeningBalances(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	label := "Opening balances"
	if record.ObjectNumber != "" {
		label += " as of " + record.ObjectNumber
	}
	if sess.Conn.SyncStartDate == nil {
		return nil, accountingsyncpush.Blocked(
			accountingsync.SyncErrorConfiguration,
			label+" needs the sync start date",
			"Set the start date, then retry",
		)
	}
	start := *sess.Conn.SyncStartDate
	dated := timeutils.PreviousDayStart(start, sess.Loc)

	res := p.kit.NewResolver(sess)
	partyAccounts, err := p.openingPartyAccounts(ctx, sess, res, start)
	if err != nil {
		return nil, err
	}
	balances, err := p.ledger.SumLines(ctx, &journalentryports.SumLedgerRequest{
		TenantInfo:      sess.Tenant,
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
			accountID: balance.AccountID,
			accountLabel: accountingsyncpush.GLAccountLabel(
				balance.AccountCode,
				balance.AccountName,
			),
			accountName: accountingsyncpush.GLAccountName(
				balance.AccountCode,
				balance.AccountName,
			),
			netMinor: balance.NetMinor(),
			party:    balance.Party,
		})
	}

	currency := functionalCurrency(sess)
	sentDate := record.SentDate(dated)
	if err = sess.CheckBooks(currency, sentDate, label); err != nil {
		return nil, err
	}
	docLines, err := p.ledgerDocLines(ctx, sess, res, &ledgerLinesRequest{
		label: label,
		lines: lines,
		sum:   true,
	})
	if err != nil {
		return nil, err
	}
	day := timeutils.FormatCalendarDate(dated, sess.Loc)
	if len(docLines) == 0 {
		return nil, &accountingsyncpush.NoopError{
			Reason: "Trenova has no balances before " + timeutils.FormatCalendarDate(
				start,
				sess.Loc,
			) +
				", so there are no opening balances to send",
		}
	}

	rate, err := p.kit.ExchangeRate(ctx, sess, &accountingsyncpush.DocumentRate{
		Label:          label,
		Currency:       currency,
		DocumentDate:   dated,
		AccountingDate: dated,
	})
	if err != nil {
		return nil, err
	}
	doc := &accountingconnectorports.AccountingJournalDocument{
		Auth:         sess.Auth,
		RequestID:    record.RequestID,
		Kind:         record.ObjectType,
		DocNumber:    ledgerDocNumber(sess, openingDocPrefix+day),
		TxnDate:      timeutils.FormatCalendarDate(sentDate, sess.Loc),
		CurrencyCode: currency,
		ExchangeRate: rate,
		PrivateNote: accountingsyncpush.NotePrefix + "opening balances as of " + day +
			accountingsyncpush.SentDateNote(record, dated, sess.Loc),
		Lines: docLines,
	}
	written, err := sess.Journals.CreateJournalEntry(ctx, doc)
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.WithProviderURL(
		accountingsyncpush.Finished(sess, record, written, doc, res.MappingIDs()),
	)
}

func (p *Pusher) openingPartyAccounts(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	res *accountingsyncpush.Resolver,
	start int64,
) ([]pulid.ID, error) {
	totals, err := p.ledger.SumLines(ctx, &journalentryports.SumLedgerRequest{
		TenantInfo:     sess.Tenant,
		Before:         start,
		IncludeClosing: true,
	})
	if err != nil {
		return nil, err
	}
	lines := make([]ledgerLine, 0, len(totals))
	for idx := range totals {
		lines = append(lines, ledgerLine{
			accountID: totals[idx].AccountID,
			accountLabel: accountingsyncpush.GLAccountLabel(
				totals[idx].AccountCode,
				totals[idx].AccountName,
			),
		})
	}
	external, err := p.ledgerAccounts(ctx, res, sess, lines)
	if err != nil {
		return nil, err
	}
	needs, err := p.ledgerPartyNeeds(ctx, sess, external)
	if err != nil {
		return nil, err
	}
	accounts := make([]pulid.ID, 0, 2)
	for accountID, externalID := range external {
		if needs[externalID] != accountingmapping.LedgerPartyNone {
			accounts = append(accounts, accountID)
		}
	}
	return accounts, nil
}
