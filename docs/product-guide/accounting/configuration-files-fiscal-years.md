---
path: /accounting/configuration-files/fiscal-years
aliases: [financial year, accounting year, accounting periods, fiscal periods, period close, year-end close, month-end close]
related:
  - /accounting/configuration-files/account-types
  - /accounting/manual-journals
  - /accounting/reports/trial-balance
  - /accounting/reports/income-statement
---

## What it's for
Fiscal years set the accounting calendar that every ledger posting is dated into. Each year has a
**Year**, a **Name**, a **Status** (**Draft**, **Open**, **Closed** or **Permanently closed**),
its start and end dates, and twelve monthly fiscal periods that are created with it. One year is
the current fiscal year, the one open for posting. The table shows **Status**, **Year**,
**Name**, **Date range**, **Description** and **Created At**.

Controllers and accounting administrators use the page to set up new years, open and lock
periods through month-end, and close the year.

## Tasks

### Add a fiscal year
Keywords: new fiscal year, set up next year
1. Open [Fiscal years](/accounting/configuration-files/fiscal-years).
2. Select **New fiscal year**.
3. Fill in **Year** and **Name**, and optionally a **Description**.
4. Leave **Calendar year** on for January 1 to December 31, or turn it off and set the
   **Start date** and **End date**. The calendar cannot be changed after the year is created.
5. Turn on **Allow adjusting entries** if adjustments may be posted after year-end close.
6. Select **Save**. The year is created as a draft with its monthly periods.

### Make a fiscal year current
Keywords: activate fiscal year, open year for posting
1. Open [Fiscal years](/accounting/configuration-files/fiscal-years).
2. Right-click the year and choose **Set as current**, or open the year and use **Set as current**
   at the top of its panel.
3. Confirm with **Set as current**. Any other current year stops being current.

### Open, lock or close a fiscal period
Keywords: month-end close, lock month, close period, reopen period, unlock period
1. Open [Fiscal years](/accounting/configuration-files/fiscal-years) and select the year.
2. Under **Fiscal periods**, open the actions menu on the period's row.
3. Choose **Open period**, **Lock period**, **Unlock period**, **Close period** or
   **Reopen period**, and confirm. Reopening asks for a **Reason**, which is recorded on the
   period.

### Close or reopen a fiscal year
Keywords: year-end close, close the books, reverse year close
1. Open [Fiscal years](/accounting/configuration-files/fiscal-years).
2. Right-click an open year and choose **Close year**. The dialog previews the closing entry
   (revenue, cost of revenue, operating expense and net income) and the opening balances carried
   into the next year, and lists anything blocking the close.
3. Select **Post and close year**.
4. To undo a close, right-click a closed year, choose **Reopen year**, enter the **Reason**, and
   select **Reverse and reopen**.

## Notes
- Opening the page needs read access to fiscal years. Setting a year current, closing and
  reopening each need their own fiscal year permission, and period actions need fiscal period
  permissions; actions you are not allowed to take do not appear.
- An open period accepts all postings. A locked period blocks invoices, receivables, payables and
  other subledger postings but still accepts manual journal entries. A closed period blocks every
  posting until it is reopened. Periods open and close in order, so an earlier period may need
  attention first.
- Closing a year before its end date shows **Early close**: nothing more can be posted for the
  rest of that year.
- Reopening a year posts reversing entries against its closing and opening entries; the year must
  be closed again afterwards. A permanently closed year or period cannot change.
