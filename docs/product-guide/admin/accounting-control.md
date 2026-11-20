---
path: /admin/accounting-control
aliases: [accounting settings, GL defaults, default accounts, journal posting settings, period close settings, multi-currency, escrow adjustment account]
related:
  - /accounting/manual-journals
  - /accounting/journal-reversals
  - /accounting/configuration-files/fiscal-years
  - /admin/billing-controls
  - /admin/settlement-control
---

## What it's for
Accounting control holds the organization's accounting rules and default general ledger accounts.
It is one settings form, saved once, in tabs: **Posting** (accounting basis, when revenue and
expenses are recognized, automatic and manual journals, reversals), **Default accounts** (the
accounts journals post to), **Driver pay** (accounts driver settlements post to; only for an
organization that runs a fleet and has driver pay), **Close & reconciliation** (period close,
posting into locked or closed periods, reconciliation) and **Currency**.

Controllers and accounting administrators use it when setting up the ledger and when changing how
postings flow. When a save is refused, the tab holding the setting to fix opens and is marked.

## Tasks

### Set the accounting basis
Keywords: accrual, cash basis, revenue recognition
1. Open [Accounting controls](/admin/accounting-control).
2. On **Posting**, under **Recognition**, choose the **Accounting basis** (**Accrual** or **Cash**).
   Each basis allows one **Revenue recognition** and one **Expense recognition** policy, and
   choosing the basis sets both.
3. Select **Save changes**.

### Post journals automatically from business events
Keywords: auto post, automatic journal entries, invoice posted journal
1. Open [Accounting controls](/admin/accounting-control).
2. On **Posting**, under **Journals**, set **Journal posting** to **Automatic**. The events your
   recognition policies need are chosen for you.
3. Under **Events that post journals**, tick any other events that may create journal entries,
   grouped by **Customers**, **Vendors**, **Driver settlements**, **Carrier settlements** and
   **Factoring**.
4. On **Default accounts**, pick the accounts marked required.
5. Select **Save changes**.

### Set the default GL accounts
Keywords: default revenue account, AR account, AP account, chart of accounts defaults
1. Open [Accounting controls](/admin/accounting-control).
2. On **Default accounts**, pick accounts such as **Revenue account**, **Accounts receivable
   account**, **Cash account**, **Accounts payable account** and **Write-off account**. With
   automatic posting, revenue, expense, receivable and payable accounts are required, and cash and
   unapplied cash too once customer payments post.
3. On **Driver pay**, pick the **Driver pay expense account**, **Settlements payable account**
   and the other settlement and escrow accounts. Every escrow movement posts a journal: a
   manual escrow adjustment books between the **Escrow liability account** and the **Escrow
   adjustment account**, kept apart from purchased transportation so corrections stay visible
   and reconcilable and never change owner-operator cost per mile, and the refund when an escrow
   account closes pays out of the **Cash account**, so an adjustment or a closure is refused
   until those are set.
4. Select **Save changes**.

### Control manual journals and period close
Keywords: manual JE approval, closed period posting, reconciliation tolerance
1. Open [Accounting controls](/admin/accounting-control).
2. On **Posting**, set **Manual journal entries** (**Allow all**, **Adjustment only** or
   **Disallow**), **Require approval for manual journals** and **Journal reversals**.
3. On **Close & reconciliation**, set **Period close**, **Require approval to close a period**,
   **Posting into a locked period** and **Posting into a closed period**.
4. Set **Reconciliation**; when it is not **Disabled**, also set the **Tolerance**, **Require
   reconciliation to close** and **Notify on discrepancies**.
5. Select **Save changes**.

### Turn on multi-currency
Keywords: foreign currency, exchange rates, FX gain loss, OANDA
1. Open [Accounting controls](/admin/accounting-control).
2. On **Currency**, set **Currency mode** to **Multi currency** and choose the **Functional
   currency**.
3. If the row says **OANDA required**, select **Configure** to connect it.
4. Once OANDA is connected, set the **Exchange rate date**, **Exchange rate overrides**,
   **Realized FX gain account** and **Realized FX loss account**.
5. Select **Save changes**.

## Notes
Opening the page needs read access to accounting control; saving needs update access. The currency
policy settings only appear once OANDA is connected, and **Configure** needs read access to
integrations. A setting hidden because another one turns it off is saved the way that setting
requires: switching back to **Single currency** clears the FX accounts, manual posting clears the
auto-post events, and turning reconciliation off zeroes the tolerance.
