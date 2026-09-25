---
path: /billing/queue
aliases: [billing review, ready to bill, ready to invoice, statements, statement billing, billing transfer]
related:
  - /billing/invoices
  - /billing/configuration-files/customers
  - /billing/configuration-files/formula-templates
---

## What it's for
The billing queue is where billers review shipments that are ready to bill before they become invoices. Each queue item shows the shipment's payer, BOL, route and assigned biller, with **Charges**, **Documents**, **Comments** and **Activity** tabs, so a biller can check the rating and paperwork and then approve, hold, send back or flag it.

The page has two views, switched in the header. **Shipments** lists queue items one at a time for review. **Statements** shows each customer billed on a statement cycle, what has accrued to their current period, and the invoices it will become when billed.

## Tasks

### Review and approve a shipment
Keywords: approve billing, start review, billing check, ready to invoice
1. Open [Billing queue](/billing/queue) and choose the **Shipments** view.
2. Select an item in the list (the j and k keys move down and up the list).
3. Select **Start review** to take the item yourself, or **Assign biller** to give it to someone else.
4. Check the **Charges** and **Documents** tabs.
5. Select **Approve**. If required documents are missing, or a detention charge on the shipment still needs approval, **Approve** stays disabled and its tooltip says why. After approval the next open item is selected automatically.

### Clear a detention charge that is holding an item
Keywords: held detention, detention needs approval, detention blocking approval, approve detention charge
1. Open [Billing queue](/billing/queue) and select the item. A notice above the tabs lists each detention charge on the shipment that still needs approval, with its amount and why it is held: over the policy's approval threshold, a required notice not sent in time, or escalated for review.
2. Select a charge in the notice to open it on the [Detention desk](/detention/desk).
3. Check the evidence, then select **Approve charge** if it supports the charge, or **Waive** with a coded **Reason** and a **Note** if it does not.
4. Go back to the item. Once no charge is held, the notice goes away and **Approve** is available again.

### Put an item on hold, send it back or flag an exception
Keywords: hold billing, send back to ops, billing exception, dispute charges
1. Open [Billing queue](/billing/queue) and select the item.
2. Select **Hold** to park it, or **Send back** to return it to operations, or **Exception** to flag a problem.
3. For **Send back** and **Exception**, pick a **Reason**, add **Notes** if needed, and select **Submit**.
4. When the problem is fixed, select **Resume** (on a held item) or **Resolve** (on an exception or sent-back item) to return it to review.

### Correct charges before approval
Keywords: add accessorial, edit charge, re-rate, change formula template, base rate
1. Open [Billing queue](/billing/queue), select the item and open the **Charges** tab.
2. Select **Add charge** to add an accessorial: choose the **Accessorial charge**, **Method**, **Unit** and **Amount**, then **Save**.
3. To re-rate the freight, select **Change template**, choose a **Formula template** and select **Re-rate**.
4. To change the rate the formula starts from, use **Adjust base rate**.

### Find items with search and filters
Keywords: search PRO, BOL lookup, filter by biller, filter preset
1. Open [Billing queue](/billing/queue).
2. Type a PRO or BOL number in the search box above the list.
3. Select **Filters** to narrow by **Status**, **Bill type**, **Payer** or **Assigned billers**, or turn on **Include posted items** to see records that already produced a posted invoice.
4. To reuse a combination, save it as a preset with the save icon in the filters panel; pick a saved preset from the preset picker there. **Clear all** removes every filter.

### Transfer completed shipments into the queue
Keywords: bulk transfer, move shipments to billing, mark ready to invoice
1. Open [Billing queue](/billing/queue) and select **Transfer to billing** in the header.
2. Search or filter the shipment list, then tick the shipments to move, or use the transfer-all button to take everything that matches.
3. Select **Start transfer**. Each shipment runs the billing readiness check; completed shipments that pass are marked Ready to Invoice and moved into the queue.
4. Close with **Run in background** to keep working while it runs, or wait for the results and use **Download report** to get a CSV of what moved and what failed.

### Bill a customer's statement
Keywords: consolidated invoice, statement cycle, bill period, bill early
1. Open [Billing queue](/billing/queue) and choose the **Statements** view.
2. Select the customer in the list to see the shipments accrued to the current period, grouped into the invoices they will become.
3. Untick any shipment you want to hold back to the next period (**Put them back** restores them).
4. Select **Bill now** once the period has closed, or **Bill early** to bill before it closes, then confirm in the dialog. Billing early asks for a reason and does not move the customer's cycle.

## Notes
Opening the page needs read access to the billing queue. **Transfer to billing** only appears for people who can update shipments.

A detention charge that went over its policy's approval threshold, missed a notice the policy requires, or was escalated keeps its shipment off every invoice until someone approves or waives it. While one is held the item cannot be approved, a transfer that would approve it on its own leaves it for review instead, and a statement run skips the customer's invoice until the charge is decided. A pending charge that simply was not approved automatically holds nothing. The detention desk agent can propose approving a held charge the evidence supports, but a person always decides.

Approving an item creates its invoice, which then appears on [Invoices](/billing/invoices). Every detention charge on that invoice is marked billed; voiding or crediting the invoice so that nothing bills the charge any more returns it to approved. For a customer billed on a statement, the approved shipment waits on the customer's statement until the period is billed from the **Statements** view. Invoices under the customer's minimum are skipped and their shipments roll into the next period.
