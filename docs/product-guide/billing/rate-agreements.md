---
path: /billing/rate-agreements
aliases: [contracts, customer contracts, carrier contracts, tariffs, lane rates, pricing agreements, GRI, general rate increase, rate sheets]
related:
  - /billing/configuration-files/rate-zones
  - /billing/configuration-files/rate-matrices
  - /billing/configuration-files/formula-templates
  - /billing/configuration-files/accessorial-charges
  - /fuel/configuration-files/surcharge
  - /billing/configuration-files/customers
---

## What it's for
Rate agreements holds the contracts that decide what a shipment costs: what a customer is billed, or what a carrier is paid. Each agreement has a header (who it is with, its term and pricing defaults) and the lanes it prices, plus its own accessorial prices and fuel terms. Pricing and billing staff write agreements here, send them through review, and apply rate increases or rate sheets across many lanes at once.

The table shows each agreement's **Status**, **Code**, **Name**, **Side** (customer or carrier), **Type**, **In force from**, **Until**, **Priority** and **Currency**. Opening an agreement shows the tabs **Overview**, **Lanes**, **Accessorials**, **Fuel**, **Simulation** and **Versions**.

## Tasks

### Write a new rate agreement
Keywords: new contract, create agreement, customer pricing, carrier pay agreement
1. Open [Rate agreements](/billing/rate-agreements).
2. Select **New rate agreement**, then **New rate agreement** again in the menu.
3. On **Overview**, choose the **Party type** (**Customer** or **Carrier**) and pick the **Customer** or **Carrier**, then fill in **Code**, **Name**, **Agreement type** and **Effective From**. Optionally set **Effective To**, **Auto renew**, the pricing defaults and, for a customer, **Bill To**; for a carrier, the **Margin floor** and **Maximum pay** guardrails.
4. On **Lanes**, select **Add lane** for each lane. Give it a **Label** and **Direction**, set the origin and destination scope (anywhere, country, state, zone, radius, city, postal prefix, postal code or a single location), then either a **Rating method** with a **Rate**, or a **Rate matrix**. Optionally set **Minimum charge**, **Maximum charge**, **Minimum billable miles** and **Priority**.
5. On **Accessorials**, select **Add accessorial** for any contract-specific accessorial prices, and on **Fuel**, select **Add fuel terms** to tie the agreement to a **Fuel program**.
6. Select **Save**. A new agreement always starts as a draft.

### Send an agreement through review
Keywords: approve contract, activate agreement, submit agreement, reject agreement
1. Open [Rate agreements](/billing/rate-agreements) and select the agreement's row.
2. On a draft, select **Submit for review** in the panel header, add an optional comment and confirm.
3. A reviewer opens the agreement and selects **Approve** to activate it, or **Reject** (a comment is required) to send it back to draft.
4. An active agreement can be paused with **Suspend** and brought back with **Resume**; a suspended or expired one can be retired with **Archive**.

### Test an agreement against past shipments
Keywords: simulate, what if, replay, backtest agreement
1. Open [Rate agreements](/billing/rate-agreements) and select a saved agreement.
2. Open the **Simulation** tab, set **Window (days)** and select **Run simulation**.
3. Read the results: shipments whose charge would change, what they were **Billed** and what the agreement **Would charge**, and **Lanes that did nothing**. A simulation never changes a shipment.

### Apply a rate increase
Keywords: GRI, general rate increase, raise rates, decrease rates, across the board
1. Open [Rate agreements](/billing/rate-agreements).
2. Select **New rate agreement** and choose **Apply Rate Increase**, or tick agreements in the table and select **Rate Increase** in the bar that appears.
3. Choose the scope (**One customer**, **One carrier**, **Across the board**, or the agreements you picked), set **Takes effect** and the change as a percent or a flat amount. A negative change is a decrease.
4. Select **Preview changes**, check each lane's **Before** and **After**, then select **Apply Increase**.

### Import a rate sheet
Keywords: upload rates, CSV rates, XLSX, bulk lanes
1. Open [Rate agreements](/billing/rate-agreements), select **New rate agreement** and choose **Import rate sheet**.
2. Pick the **Agreement** and when **Rates take effect**, then drop a CSV or XLSX file (use **Download template** for the expected layout).
3. Review what happens to each lane, then select **Apply these rates**, or **Discard**.

### Copy an agreement
Keywords: duplicate contract, clone agreement
1. Open [Rate agreements](/billing/rate-agreements).
2. Right-click the agreement's row and choose **Duplicate agreement**. The copy is a new draft ready to edit.

## Notes
Viewing the page needs read access to rate agreements. Submitting, approving, rejecting, archiving and duplicating each need their own rate agreement permission, and the matching buttons only appear for people who have it.

The **Status** field cannot be edited; an agreement changes status only through the review actions. Approving activates the agreement and its lanes start pricing shipments; suspending stops pricing immediately; archiving is permanent.

A rate increase or rate sheet does not edit lanes in place: each affected lane is closed out and succeeded at the new rate from the effective date, and the old rates stay in history (see a lane's **History**). Lanes priced from a rate matrix are not moved by a rate increase; change the matrix on [Rate matrices](/billing/configuration-files/rate-matrices) instead.
