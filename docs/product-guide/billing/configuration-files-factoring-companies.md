---
path: /billing/configuration-files/factoring-companies
aliases: [factor, factoring, factored invoices, notice of assignment, remit to factor, invoice factoring, recourse, non-recourse, without recourse]
related:
  - /billing/configuration-files/customers
  - /billing/invoices
  - /billing/factoring
---

## What it's for
Factoring companies lists the factors you sell receivables to. Each one holds the factor's
**Code**, **Name** and remit-to address, how customers pay it (**Remittance instructions**) and the
**Notice of assignment** printed on every invoice assigned to it.

Assign a customer to a factor on the customer's billing profile. From then on, each invoice for that
customer records the factor when it posts. The invoice PDF and email name the factor as the place
to pay and print its notice of assignment, and the customer's statements do the same. Payment
reminders leave factored invoices out, because the factor collects them.

A factor is either recourse or **Non-recourse**. With recourse, the usual arrangement, an invoice the
customer never pays comes back to you, so factored invoices stay on the
[Collections](/accounting/ar/collections) worklist and still count toward the customer's credit limit.
A non-recourse factor buys the invoices outright and carries the risk, so its invoices leave the
worklist and stop counting toward the credit limit.

Each factor's panel also holds its **Agreements** and its **Notices of assignment**. The agreement
records the terms: advance and reserve rates, the fee by days outstanding, the chargeback window,
funding days and the documents the factor needs with every invoice. A notice of assignment tells one
customer to pay the factor. A customer's invoices are only ready to factor on the
[Factoring](/billing/factoring) page once the factor has an active agreement and the customer has
acknowledged a notice.

The panel's **Remittances** section lists the factor's newest funding and reserve reports, each
opening its review on its own page, and **Import remittance** reads the next one. Emails the inbox
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
6. Optionally enter the **Notice of assignment** your agreement requires, then select **Save**.

### Mark a factor as non-recourse
Keywords: without recourse, factor carries the risk, remove factored invoices from collections
1. Open [Factoring companies](/billing/configuration-files/factoring-companies) and select the
   factor's row.
2. Under **Agreement**, turn on **Non-recourse**, then select **Save**.

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
1. Open [Factoring companies](/billing/configuration-files/factoring-companies) and select the
   factor's row.
2. Under **Agreements**, select **New agreement**.
3. Fill in **Starts**, **Advance rate**, **Reserve rate**, **Chargeback window**, **Recourse** and
   **Accounting treatment**.
4. Under **Fees**, enter each band's **Until day**, **Fee rate** and **Flat fee**. Select **Add tier**
   for another band and leave the last one open-ended.
5. Fill in **Funding days** and **Daily schedule time**, and pick the **Documents the factor needs**.
6. Select **Save draft**, then **Activate** once the contract is signed.

### End a factoring agreement
Keywords: stop factoring with a factor, terminate agreement
1. Open the factor's row in [Factoring companies](/billing/configuration-files/factoring-companies).
2. Under **Agreements**, select **End** on the active agreement.
3. Optionally choose when it **Ends**, then select **End agreement**.

### Send a notice of assignment
Keywords: NOA, send NOA, notice of assignment letter, tell the customer to pay the factor
1. Open the factor's row in [Factoring companies](/billing/configuration-files/factoring-companies).
2. Under **Notices of assignment**, select **New notice**, choose the **Customer**, then select
   **Save draft**.
3. In the dialog that opens, optionally enter a **Recipient**, then select **Send notice**.

### Record a signed notice of assignment
Keywords: NOA acknowledged, signed NOA, upload signed notice, acknowledgement
1. Open the factor's row in [Factoring companies](/billing/configuration-files/factoring-companies).
2. Under **Notices of assignment**, select **Record acknowledgement** on the customer's notice.
3. Choose **Acknowledged on**, upload the signed copy or pick it under **Signed copy**, and optionally
   set when it **Expires**.
4. Select **Record acknowledgement**.

### Release a notice of assignment
Keywords: release NOA, stop the factor collecting, customer pays us again
1. Open the factor's row in [Factoring companies](/billing/configuration-files/factoring-companies).
2. Under **Notices of assignment**, select **Release** on the notice.
3. Enter the **Reason**, then select **Release notice**.

### Set up where schedules are sent
Keywords: factor email, factor SFTP, schedule delivery, factoring upload, factor inbox, host key
1. Open the factor's row in [Factoring companies](/billing/configuration-files/factoring-companies).
2. Under **Schedule delivery**, select **Set up** (or **Edit**).
3. Choose **Send by**:
   - **Email**: add the **Recipients** and choose whether to **Attach invoice packets**.
   - **SFTP**: fill in **Host**, **Port**, **User name** and **Folder**, choose **Sign in with**,
     enter the **Password** or **Private key**, and paste the server's **Host key** from the factor.
4. Select **Save**, then select **Test** to connect with the saved settings without sending anything.

## Notes
Opening the page needs read access to factoring companies; adding and changing them needs create and
update access. Only an active factor can be assigned to a customer, and a factor cannot be made
inactive while customers are still assigned to it.

An invoice records its factor when it posts, so changing a customer's factor affects only invoices
posted afterwards. A draft shows the customer's current factor until it posts. A customer assigned to
a factor that has since been made inactive cannot post invoices until the assignment is changed.

Recourse is a property of the agreement, so it applies to every invoice assigned to the factor,
including ones already posted. Credit standings follow at the next credit review, which runs each
night and whenever the customer's receivables change.

Agreements need factoring agreement access and notices need notice of assignment access. An ended
agreement cannot be changed, and a factor has one active agreement at a time.

When **Notice of assignment** is blank, invoices print a standard notice saying the invoice is
payable only to the factor. Your factoring agreement may require exact wording; use it if so.

Schedule delivery needs update access to factoring companies. A stored password or key is never shown
again; leave it blank to keep it. An SFTP server is checked against the pasted host key and refused
if it presents another, and only public addresses are reached.
