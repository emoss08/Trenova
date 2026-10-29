package accountingsyncledger

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

type ledgerLine struct {
	accountID    pulid.ID
	accountLabel string
	accountName  string
	description  string
	netMinor     int64
	party        journalentryports.LedgerParty
}

type ledgerEntry struct {
	externalAccount string
	accountLabel    string
	description     string
	netMinor        int64
	need            accountingmapping.LedgerPartyNeed
	party           journalentryports.LedgerParty
}

type ledgerEntryKey struct {
	externalAccount string
	kind            journalentryports.LedgerPartyKind
	partyID         pulid.ID
}

type ledgerLinesRequest struct {
	label string
	lines []ledgerLine
	sum   bool
}

func journalLines(journal *journalentryports.LedgerJournal) []ledgerLine {
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
			accountLabel: accountingsyncpush.GLAccountLabel(line.AccountCode, line.AccountName),
			accountName:  accountingsyncpush.GLAccountName(line.AccountCode, line.AccountName),
			description:  description,
			netMinor:     line.NetMinor(),
			party:        party,
		})
	}
	return lines
}

func (p *Pusher) ledgerAccounts(
	ctx context.Context,
	res *accountingsyncpush.Resolver,
	sess *accountingsyncpush.Session,
	lines []ledgerLine,
) (map[pulid.ID]string, error) {
	external := make(map[pulid.ID]string, len(lines))
	for idx := range lines {
		line := &lines[idx]
		if _, seen := external[line.accountID]; seen {
			continue
		}
		externalID, err := res.Account(ctx, &accountingsyncpush.AccountRefRequest{
			AccountID: line.accountID,
			Label:     line.accountLabel,
			Role:      accountingsync.LedgerAccountRole(line.accountID, sess.Control),
			Missing:   "A journal line has no GL account",
		})
		if err != nil {
			return nil, err
		}
		external[line.accountID] = externalID
	}
	return external, nil
}

func (p *Pusher) ledgerPartyNeeds(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	external map[pulid.ID]string,
) (map[string]accountingmapping.LedgerPartyNeed, error) {
	ids := make([]string, 0, len(external))
	seen := make(map[string]struct{}, len(external))
	for _, externalID := range external {
		if _, ok := seen[externalID]; ok || externalID == "" {
			continue
		}
		seen[externalID] = struct{}{}
		ids = append(ids, externalID)
	}
	needs := make(map[string]accountingmapping.LedgerPartyNeed, len(ids))
	if len(ids) == 0 || p.references == nil {
		return needs, nil
	}
	refs, err := p.references.GetByExternalIDs(
		ctx,
		&accountingmappingports.GetAccountingReferenceObjectsRequest{
			TenantInfo:   sess.Tenant,
			ConnectionID: sess.Conn.ID,
			Kind:         accountingprovider.ReferenceKindAccount,
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

func (p *Pusher) ledgerDocLines(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	res *accountingsyncpush.Resolver,
	req *ledgerLinesRequest,
) ([]accountingconnectorports.AccountingJournalLine, error) {
	external, err := p.ledgerAccounts(ctx, res, sess, req.lines)
	if err != nil {
		return nil, err
	}
	needs, err := p.ledgerPartyNeeds(ctx, sess, external)
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
		party := journalentryports.LedgerParty{}
		if need != accountingmapping.LedgerPartyNone {
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
		return nil, accountingsyncpush.Blocked(
			accountingsync.SyncErrorValidation,
			req.label+" does not balance: its debits and credits differ by "+
				money.DecimalFromMinor(intutils.AbsDiff(total, 0)).StringFixed(2),
			"Correct the journal entries in Trenova, then retry; or skip this record",
		)
	}

	lines := make([]accountingconnectorports.AccountingJournalLine, 0, len(entries))
	for idx := range entries {
		entry := &entries[idx]
		if entry.netMinor == 0 {
			continue
		}
		line := accountingconnectorports.AccountingJournalLine{
			Posting:           accountingconnectorports.AccountingJournalDebit,
			AccountExternalID: entry.externalAccount,
			Amount:            money.DecimalFromMinor(intutils.AbsDiff(entry.netMinor, 0)),
			Description:       entry.description,
		}
		if entry.netMinor < 0 {
			line.Posting = accountingconnectorports.AccountingJournalCredit
		}
		if entry.need != accountingmapping.LedgerPartyNone {
			if line.PartyKind, line.PartyExternalID, err = p.ledgerPartyRef(
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

func (p *Pusher) ledgerPartyRef(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	res *accountingsyncpush.Resolver,
	label string,
	entry *ledgerEntry,
) (accountingconnectorports.AccountingJournalPartyKind, string, error) {
	wanted := accountingsyncpush.ProviderKindLabel(accountingprovider.ReferenceKindCustomer)
	kind := accountingconnectorports.AccountingJournalCustomer
	if entry.need == accountingmapping.LedgerPartyVendor {
		wanted = accountingsyncpush.ProviderKindLabel(accountingprovider.ReferenceKindVendor)
		kind = accountingconnectorports.AccountingJournalVendor
	}
	account := entry.accountLabel
	if account == "" {
		account = "an account"
	}
	if entry.party.IsZero() {
		return "", "", accountingsyncpush.Blocked(
			accountingsync.SyncErrorMapping,
			label+" has a line on "+account+" that needs a "+wanted+
				", and Trenova does not know whose it is",
			"Name the "+wanted+" on the journal line in Trenova, or skip this record",
		)
	}

	var objectType accountingsync.SyncObjectType
	switch {
	case entry.need == accountingmapping.LedgerPartyCustomer &&
		entry.party.Kind == journalentryports.LedgerPartyCustomer:
		objectType = accountingsync.SyncObjectCustomer
	case entry.need == accountingmapping.LedgerPartyVendor &&
		entry.party.Kind == journalentryports.LedgerPartyCarrier:
		objectType = accountingsync.SyncObjectCarrierVendor
	case entry.need == accountingmapping.LedgerPartyVendor &&
		entry.party.Kind == journalentryports.LedgerPartyDriver:
		objectType = accountingsync.SyncObjectDriverVendor
	default:
		return "", "", accountingsyncpush.Blocked(
			accountingsync.SyncErrorMapping,
			label+" has a line on "+account+" that needs a "+wanted+", but it belongs to "+
				strings.ToLower(string(entry.party.Kind))+" "+entry.party.Name,
			"Map "+account+" to a "+sess.ProviderName+" account of the right type, or skip this record",
		)
	}
	externalID, err := p.kit.PartyRef(ctx, sess, res, objectType, entry.party.ID)
	if err != nil {
		return "", "", err
	}
	return kind, externalID, nil
}
