---
path: /admin/billing-controls
aliases: [billing settings, invoice settings, auto invoicing, billing automation, billing rules, late charge settings]
related:
  - /billing/queue
  - /billing/invoices
  - /accounting/ar/late-charges
  - /billing/configuration-files/formula-templates
  - /admin/invoice-adjustment-controls
---

## What it's for
Billing control holds the organization-wide rules for billing. The page is one settings form,
saved once, in five tabs: **Invoices** (the fallback payment term, terms and footer, and what
invoices show), **Automation** (how shipments move into billing, whether invoices are drafted and
posted for you, and whether the billing queue approves clean items on its own), **Exceptions**
(what a failed billing requirement or rate check does, and where reviews go), **Rating** (what
happens to a shipment no rate agreement covers) and **Collections** (late charges, payment
reminders and monthly statements). A save that is refused opens the tab holding the setting to fix
and marks it.

Unless **When no rate applies** is set to zero the rate and flag it for review, a shipment that
nothing can price is refused wherever it is entered: the shipment form, the API, an agent, or a
copy of another shipment. Choose a rating method on the shipment, or set a fallback formula
template here. Minimums, maximums and rounding are not set here or on an agreement: each tariff
writes them in its own source.

The fallback formula template and margin floors are offered on plans that include rating.

Raised late charges post the way invoices do: while **Post invoices** is **Manual review
required**, the memos the nightly run raises wait for someone to post them, and the tab says so
with a link to **Automation**.

Billing managers and finance leads use it. These settings affect revenue processing and
invoicing, so change them only after review.

## Tasks

### Set invoice defaults
Keywords: payment terms, invoice footer, due date on invoice, net 30
1. Open [Billing controls](/admin/billing-controls) on the **Invoices** tab.
2. Under **Terms**, choose the **Payment term** and fill in **Invoice terms**. These apply when a
   customer's billing profile does not set its own.
3. Under **On the invoice**, turn **Show the due date** and **Show the balance due** on or off, and
   fill in the **Footer**.
4. Select **Save changes**.

### Automate the billing queue and invoice drafts
Keywords: auto transfer, auto invoice, ready to bill, auto post invoices, auto approve
1. Open [Billing controls](/admin/billing-controls) and select the **Automation** tab.
2. Under **Ready to bill**, set **Mark ready to bill** and **Move to the billing queue**. With
   **Automatic when ready**, also set **How often** and **Shipments per run**.
3. Under **Drafts and posting**, set **Create invoice drafts**. With **Automatic when
   transferred**, also set **Drafts per run** and whether to **Notify when drafts are created**.
4. Set **Post invoices** to **Manual review required** or **Automatic when no blocking
   exceptions**. Invoices can post on their own only while drafts are created on their own, so
   making drafts by hand again puts posting back to review.
5. Under **Queue approval**, choose whether the queue may **Approve on its own**, and optionally
   cap it with **Only up to this amount**. On a plan with Nova's review, **Nova reviews each item**
   turns that review on and **When the checks clear and Nova agrees** can be chosen. If the plan
   later loses Nova's review, that choice reads as **Off** and every item waits for a biller.
6. Select **Save changes**.

### Decide how billing exceptions are handled
Keywords: rate variance, billing requirements, block billing, review routing
1. Open [Billing controls](/admin/billing-controls) and select the **Exceptions** tab.
2. Under **Checks**, set **Billing requirements** and **Rate validation** to **Ignore**, **Warn**,
   **Require review** or **Block**.
3. With rate validation at **Require review**, choose whether to **Skip review for small
   differences**. With **Bypass review within tolerance**, set the **Tolerance** as a percentage of
   the rate.
4. Under **Review**, choose where a shipment waiting on review goes with **Send for review to**:
   **Route to billing review** or **Return to operations**. It is offered unless both checks block,
   since a customer that validates its rates raises rate validation to review. Turn on **Notify
   on exceptions** if wanted.
5. Select **Save changes**.

### Handle shipments with no rate agreement
Keywords: unrated shipments, fallback formula, margin floor, rate override reason
1. Open [Billing controls](/admin/billing-controls) and select the **Rating** tab.
2. Under **Unrated shipments**, choose **When no rate applies**: **Fall back to formula
   template**, **Zero the rate and flag for review** or **Block the save**.
3. For the fallback, optionally pick a **Fallback formula template**. It runs with the defaults its tariff declares, since no contract supplies its inputs; a tariff with an input that has no default refuses to rate such a shipment and says which input is missing.
4. Under **Rate changes**, turn on **Require a reason for rate overrides** and **Enforce margin
   floors** as needed.
5. Select **Save changes**.

### Turn on automatic late charges
Keywords: late fees, overdue invoices, debit memo, finance charges
1. Open [Billing controls](/admin/billing-controls) and select the **Collections** tab.
2. Under **Late charges**, set **Assess late charges** to **Disabled**, **Preview only** or
   **Automatic**.
3. Set the **Minimum late charge**; customers whose charges for a run add up to less are skipped.
4. Select **Save changes**.

### Email customers payment reminders
Keywords: dunning, overdue reminder, past due email, collection letter
1. Open [Billing controls](/admin/billing-controls) and select the **Collections** tab.
2. Under **Payment reminders**, turn on **Send payment reminders**.
3. In **Remind on days past due**, list up to six days from 1 to 365, such as 1, 15, 30. The last
   one is sent as the final reminder.
4. Select **Save changes**.

### Email customers a monthly statement
Keywords: monthly statement, statement of account, account summary email, AR statement
1. Open [Billing controls](/admin/billing-controls) and select the **Collections** tab.
2. Under **Monthly statements**, turn on **Send monthly statements**.
3. Set the **Day of the month to send**, from 1 to 28.
4. Select **Save changes**.

## Notes
Opening the page needs read access to billing control; saving needs update access. Late charge
rates and grace periods are set on each customer's billing profile, not here. In **Preview only**
the nightly run writes nothing, and memos are raised by hand on
[Late charges](/accounting/ar/late-charges).

Payment reminders go out each morning: a customer with an invoice newly reaching one of the
**Remind on days past due** days is emailed one reminder listing every overdue invoice it has.
Disputed invoices are left off, and a customer waiting on an open promise to pay recorded in
[Collections](/accounting/ar/collections) is not reminded until the promised date passes. Turn off
**Send payment reminders** on a customer's billing profile to leave that customer out.

Monthly statements cover the calendar month before the **Day of the month to send** and are sent
once per customer and month. Customers with a balance, or with any invoice or payment in the month,
receive one; a customer whose account was settled and quiet all month does not. Turn off **Send
monthly statements** on a customer's billing profile to leave that customer out. Customers with no
invoice email recipient are reported to the people who record customer payments.
