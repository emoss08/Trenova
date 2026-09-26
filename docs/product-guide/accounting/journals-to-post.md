---
path: /accounting/journals-to-post
aliases: [journals awaiting approval, approve journal entries, post journal entries, pending journals, unposted journals, manual posting, journal not in trial balance, GL not updated]
related:
  - /accounting/manual-journals
  - /accounting/journal-reversals
  - /accounting/reports/trial-balance
---

## What it's for
When the organization's accounting control sets journal posting to **Manual**, the journals that
invoices, credit and debit memos, invoice adjustments, customer payments and driver and carrier
settlements write do not reach the general ledger on their own. They wait on **Journals to post**
until someone posts them, and until someone approves them first when manual journal approval is
required. Until an entry is posted it is not in the trial balance or any account balance.

The strip at the top counts the entries **Awaiting approval** and those **Ready to post**, and
shows the accounting date of the oldest entry waiting; selecting a count filters the table to it.
Selecting a row opens the journal entry with its lines.

## Tasks

### Approve journal entries
Keywords: approve journals, sign off journals, review journals
1. Open [Journals to post](/accounting/journals-to-post).
2. Select the entries marked **Awaiting approval**.
3. Select **Approve**. Each entry is decided on its own: the message says how many were approved
   and why any were not.

### Post journal entries to the general ledger
Keywords: post journals, post to GL, update trial balance, release journals
1. Open [Journals to post](/accounting/journals-to-post).
2. Select the entries marked **Ready to post**.
3. Select **Post to ledger**. The entries update the account balances and the trial balance
   straight away.

## Notes
Viewing the page needs read access to journal entries; approving and posting need the journal
entry approve permission, which agents never hold. An entry is posted on its own accounting date.
If that date's fiscal period has been closed since the entry was written, the closed-period
policy decides: the entry moves to the first day of the next open period, or it is refused until
the period is reopened. An entry that does not balance is refused. When posting is switched to
**Automatic**, new journals post on their own; entries written while posting was manual stay here
until someone posts them.
