---
path: /admin/accounting-control
aliases: [accounting settings, GL defaults, default accounts, journal posting settings, period close settings, multi-currency]
related:
  - /accounting/manual-journals
  - /accounting/journal-reversals
  - /accounting/configuration-files/fiscal-years
  - /admin/billing-controls
  - /admin/settlement-control
---

## What it's for
Accounting control holds the organization's accounting rules and default general ledger accounts.
The page is one settings form in five cards: **Recognition policy** (accounting basis and when
revenue and expenses are recognized), **Journal policy** (automatic journal posting, manual
journals, reversals and default accounts), **Driver settlement posting** (accounts used when driver
settlements post), **Period and reconciliation** (period close and posting into locked or closed
periods) and **Currency settings**.

Controllers and accounting administrators use it when setting up the ledger and when changing how
postings flow.

## Tasks

### Set the accounting basis and recognition policies
Keywords: accrual, cash basis, revenue recognition
1. Open [Accounting controls](/admin/accounting-control).
2. In **Recognition policy**, choose the **Accounting basis** (**Accrual** or **Cash**).
3. Set the **Revenue recognition policy** and **Expense recognition policy**; they must be
   compatible with the basis.
4. Select **Save changes**.

### Post journals automatically from business events
Keywords: auto post, automatic journal entries, invoice posted journal
1. Open [Accounting controls](/admin/accounting-control).
2. In **Journal policy**, set **Journal posting mode** to **Automatic**.
3. Under **Auto-post source events**, tick the events that may create journal entries, such as
   **Invoice posted** or **Customer payment posted**.
4. Select **Save changes**.

### Set the default GL accounts
Keywords: default revenue account, AR account, AP account, chart of accounts defaults
1. Open [Accounting controls](/admin/accounting-control).
2. In **Journal policy**, pick accounts such as **Default Revenue account**, **Default cash
   account**, **Default AR account**, **Default AP account** and **Default write-off account**.
3. In **Driver settlement posting**, pick the **Driver pay expense account**, **Settlements payable
   account** and the other settlement accounts.
4. Select **Save changes**.

### Control manual journals and period close
Keywords: manual JE approval, closed period posting, reconciliation tolerance
1. Open [Accounting controls](/admin/accounting-control).
2. In **Journal policy**, set the **Manual journal entry policy** (**Allow all**, **Adjustment
   only** or **Disallow**), **Require manual JE approval** and the **Journal reversal policy**.
3. In **Period and reconciliation**, set the **Period close mode**, **Require period close
   approval**, **Locked period posting policy** and **Closed period posting policy**.
4. Set the **Reconciliation mode**; when it is not **Disabled**, also set the **Reconciliation
   tolerance amount**.
5. Select **Save changes**.

### Turn on multi-currency
Keywords: foreign currency, exchange rates, FX gain loss, OANDA
1. Open [Accounting controls](/admin/accounting-control).
2. In **Currency settings**, set **Currency mode** to **Multi currency** and choose the
   **Functional currency**.
3. If the OANDA panel says OANDA is required, select **Configure** to connect it.
4. Once OANDA is connected, set the **Exchange rate date policy**, **Exchange rate override
   policy**, **Realized FX gain account** and **Realized FX loss account**.
5. Select **Save changes**.

## Notes
Opening the page needs read access to accounting control; saving needs update access. The currency
policy fields only appear once OANDA is connected, and **Configure** needs read access to
integrations. Switching back to **Single currency** clears the FX accounts.
