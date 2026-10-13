---
path: /billing/configuration-files/factoring-companies
aliases: [factor, factoring, factored invoices, notice of assignment, remit to factor, invoice factoring]
related:
  - /billing/configuration-files/customers
  - /billing/invoices
---

## What it's for
Factoring companies lists the factors you sell receivables to. Each one holds the factor's
**Code**, **Name** and remit-to address, how customers pay it (**Remittance instructions**) and the
**Notice of assignment** printed on every invoice assigned to it.

Assign a customer to a factor on the customer's billing profile. From then on, each invoice for that
customer records the factor when it posts. The invoice PDF and email name the factor as the place
to pay and print its notice of assignment, and the customer's statements do the same. Payment
reminders leave factored invoices out, because the factor collects them.

## Tasks

### Add a factoring company
Keywords: new factor, add factor, factoring agreement
1. Open [Factoring companies](/billing/configuration-files/factoring-companies).
2. Select **New factoring company**.
3. Fill in **Code**, **Name** and the remit-to address (**Address line 1**, **City**, **State**,
   **Postal code**), and optionally **Email** and **Phone**.
4. Fill in **Remittance instructions** with the factor's bank, lockbox or ACH details.
5. Optionally enter the **Notice of assignment** your agreement requires, then select **Save**.

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

## Notes
Opening the page needs read access to factoring companies; adding and changing them needs create and
update access. Only an active factor can be assigned to a customer, and a factor cannot be made
inactive while customers are still assigned to it.

An invoice records its factor when it posts, so changing a customer's factor affects only invoices
posted afterwards. A draft shows the customer's current factor until it posts. A customer assigned to
a factor that has since been made inactive cannot post invoices until the assignment is changed.

When **Notice of assignment** is blank, invoices print a standard notice saying the invoice is
payable only to the factor. Your factoring agreement may require exact wording; use it if so.
