---
path: /billing/configuration-files/factoring-companies
aliases: [factor, factoring, factored invoices, notice of assignment, remit to factor, invoice factoring, recourse, non-recourse, without recourse, factor page, factor reserve, funding pace]
related:
  - /billing/configuration-files/customers
  - /billing/invoices
  - /billing/factoring
---

## What it's for
Factoring companies lists the factors you sell receivables to. Selecting a factor opens its own
page, and **New factoring company** opens a page to add one.

A factor's page has these tabs:

- **Overview**: Nova's summary of the factor, then where its invoices are on their way to cash:
  **Ready to submit**, **With the factor**, **Funded, reserve held** and **Released**, with the
  recourse invoices nearing their chargeback window branching off. Each stage opens the
  [Factoring](/billing/factoring) workspace narrowed to this factor. Below it, **Needs attention**
  lists what needs someone, worst first, each with the step that moves it; **Terms in force** says
  the agreement in plain words; and **Funding pace** plots how many business days the factor took to
  fund its last schedules against the days agreed. A factor that has funded nothing yet shows the
  steps it needs instead: an agreement, schedule delivery, customers and their notices.
- **Details**: the factor's **Identity**, **Remit-to address**, **Payment instructions and notice**,
  **Contact**, **Recourse** and **Notes**, each saying where it prints, beside a sample invoice laid
  out as your invoices print. Pointing at or editing a section marks its part of the invoice. The save
  bar names what changed.
- **Agreement**: every version of the agreement. **What an invoice pays** works out the advance, fee
  and payment on release for an amount and the days left on its terms, the way funding computes
  them, and what the fee costs a year. Drafts show what they would change from the agreement in force.
  **Under this agreement** totals the schedules sent on the version (how many were funded, their
  invoices, face value, the advance and fees expected); **Booked to** names the ledger account each
  part of the factoring posts to, or your organization's default; and **History** shows when the
  version was drafted, activated, took effect and ends, and what it changed from the one before.
- **Customers**: the customers assigned to the factor in their billing profiles, each with where its
  notice of assignment stands and the action that moves it on.
- **Delivery**: how schedules reach the factor, its last test, and the last schedules sent.
- **Ledger**: the reserve held over the last 90 days, every reserve movement, and the factor's
  remittances.
- **Activity**: the factor's history.

Assign a customer to a factor on the customer's billing profile. From then on, each invoice for that
customer records the factor when it posts. The invoice PDF and email name the factor as the place
to pay and print its notice of assignment, and the customer's statements do the same. Payment
reminders leave factored invoices out, because the factor collects them.

A factor is either recourse or non-recourse. With recourse, the usual arrangement, an invoice the
customer never pays comes back to you, so factored invoices stay on the
[Collections](/accounting/ar/collections) worklist and still count toward the customer's credit limit.
A non-recourse factor buys the invoices outright and carries the risk, so its invoices leave the
worklist and stop counting toward the credit limit. Once a factor has an agreement, the agreement
sets recourse.

A customer's invoices are only ready to factor on the [Factoring](/billing/factoring) page once the
factor has an active agreement and the customer has acknowledged a notice. Emails the inbox
classifies as a factor remittance are matched to the factor by its **Email** address, or by its
domain when no factor uses that exact address, so keep the factor's email current.

## Tasks

### Add a factoring company
Keywords: new factor, add factor, factoring agreement
1. Open [Factoring companies](/billing/configuration-files/factoring-companies).
2. Select **New factoring company**.
3. Fill in **Code**, **Name** and the remit-to address (**Address line 1**, **City**, **State**,
   **Postal code**), and optionally **Email** and **Phone**.
4. Fill in **Remittance instructions** with the factor's bank, lockbox or ACH details.
5. Turn on **Non-recourse** if the factor buys invoices without recourse.
6. Optionally enter the **Notice of assignment** your agreement requires, then select **Create
   factoring company**. The factor's page opens on the steps it still needs.

### Change a factor's remit-to or invoice wording
Keywords: factor address, lockbox, remittance instructions, notice wording
1. Open [Factoring companies](/billing/configuration-files/factoring-companies) and select the
   factor.
2. Open the **Details** tab and change the fields. The sample invoice shows how invoices will print
   them. If customers already have a notice naming the current remit-to, the page says how many;
   send them an updated notice from **Customers** after you save.
3. Select **Save changes**.

### See what needs attention on a factor
Keywords: chargebacks due, schedule failed, funding late, remittance to apply, factor problems
1. Open [Factoring companies](/billing/configuration-files/factoring-companies) and select the
   factor.
2. On the **Overview** tab, read **Needs attention** and select the button on the item to act on.

### Factor a customer's invoices
Keywords: assign customer to factor, factored customer, notice of assignment
1. Open [Customers](/billing/configuration-files/customers) and select the customer's row.
2. Open the **Billing profile** tab.
3. Under **Factoring**, choose the **Factoring company**, then select **Save**.

### Stop factoring a customer
Keywords: unassign factor, end factoring
1. Open [Customers](/billing/configuration-files/customers) and select the customer's row.
2. Open the **Billing profile** tab.
3. Clear the **Factoring company**, then select **Save**.

### Set up a factoring agreement
Keywords: factoring terms, advance rate, reserve, fee schedule, chargeback window, factoring contract
1. Open the factor from [Factoring companies](/billing/configuration-files/factoring-companies).
2. On the **Agreement** tab, select **New agreement**, or **New version from these terms** to start
   from the selected one.
3. Fill in **Starts**, **Advance rate**, **Reserve rate**, **Chargeback window**, **Recourse** and
   **Accounting treatment**.
4. Under **Fees**, enter each band's **Until day**, **Fee rate** and **Flat fee**. Select **Add tier**
   for another band and leave the last one open-ended. The **Advance rate** plus any band's
   **Fee rate** cannot be more than 100% of the invoice.
5. Fill in **Funding days** and **Daily schedule time**, and pick the **Documents the factor needs**.
6. Select **Save draft**, then **Activate** once the contract is signed.

### Work out what an invoice will pay
Keywords: factoring calculator, advance, fee, reserve release, cost of factoring
1. Open the factor from [Factoring companies](/billing/configuration-files/factoring-companies).
2. On the **Agreement** tab, pick the version, then enter the **Invoice amount** and drag **Days left
   on its terms when submitted**.

### End a factoring agreement
Keywords: stop factoring with a factor, terminate agreement
1. Open the factor from [Factoring companies](/billing/configuration-files/factoring-companies).
2. On the **Agreement** tab, pick the active version and select **End**.
3. Optionally choose when it **Ends**, then select **End agreement**.

### Send a notice of assignment
Keywords: NOA, send NOA, notice of assignment letter, tell the customer to pay the factor
1. Open the factor from [Factoring companies](/billing/configuration-files/factoring-companies).
2. On the **Customers** tab, select **Draft notice** on the customer, or **New notice** to choose
   one, then select **Save draft**.
3. In the dialog that opens, optionally enter a **Recipient**, then select **Send notice**.

### Record a signed notice of assignment
Keywords: NOA acknowledged, signed NOA, upload signed notice, acknowledgement
1. Open the factor from [Factoring companies](/billing/configuration-files/factoring-companies).
2. On the **Customers** tab, select **Record acknowledgement** on the customer.
3. Choose **Acknowledged on**, upload the signed copy or pick it under **Signed copy**, and optionally
   set when it **Expires**.
4. Select **Record acknowledgement**.

### Release a notice of assignment
Keywords: release NOA, stop the factor collecting, customer pays us again
1. Open the factor from [Factoring companies](/billing/configuration-files/factoring-companies).
2. On the **Customers** tab, select **Release** on the customer.
3. Enter the **Reason**, then select **Release notice**.

### Set up where schedules are sent
Keywords: factor email, factor SFTP, schedule delivery, factoring upload, factor inbox, host key
1. Open the factor from [Factoring companies](/billing/configuration-files/factoring-companies).
2. On the **Delivery** tab, select **Set up** (or **Edit**).
3. Choose **Send by**:
   - **Email**: add the **Recipients** and choose whether to **Attach invoice packets**.
   - **SFTP**: fill in **Host**, **Port**, **User name** and **Folder**, choose **Sign in with**,
     enter the **Password** or **Private key**, and paste the server's **Host key** from the factor.
4. Select **Save**, then select **Test** to connect with the saved settings without sending anything.

### Import a factor's remittance
Keywords: funding report, reserve report, remittance file
1. Open the factor from [Factoring companies](/billing/configuration-files/factoring-companies).
2. Select **Import remittance** at the top of the page.

## Notes
Opening the page needs read access to factoring companies; adding and changing them needs create and
update access. Only an active factor can be assigned to a customer, and a factor cannot be made
inactive while customers are still assigned to it.

Amounts on the factor's page need read access to factoring reserves. Without it the money path shows
invoice counts, the summary leaves money out and the **Ledger** tab shows only remittances when you
can read those. The **Agreement** tab needs factoring agreement access, the **Customers** tab
customer access, and the **Activity** tab audit log access.

An invoice records its factor when it posts, so changing a customer's factor affects only invoices
posted afterwards. A draft shows the customer's current factor until it posts. A customer assigned to
a factor that has since been made inactive cannot post invoices until the assignment is changed.

Recourse is a property of the agreement, so it applies to every invoice assigned to the factor,
including ones already posted. Credit standings follow at the next credit review, which runs each
night and whenever the customer's receivables change.

The factor takes its fee out of the invoice, so the advance and the highest fee rate together must
fit within it. A **Flat fee** or **Minimum fee** is an amount rather than a share of the invoice, so
on a small invoice where it leaves less than the advance rate, the expected advance is lowered to
what remains after the fee. **What an invoice pays** follows the same rule.

Funding pace counts business days, Monday to Friday, in your organization's time zone. A schedule is
late when it took more than the agreement's funding days.

When **Notice of assignment** is blank, invoices print a standard notice saying the invoice is
payable only to the factor; the preview shows it word for word. Your factoring agreement may require
exact wording; use it if so.

Schedule delivery needs update access to factoring companies. A stored password or key is never shown
again; leave it blank to keep it. An SFTP server is checked against the pasted host key and refused
if it presents another, and only public addresses are reached.
