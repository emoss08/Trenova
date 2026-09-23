---
path: /billing/configuration-files/customers
aliases: [customer list, clients, shippers, bill-to, billing profile, customer master, accounts]
related:
  - /billing/queue
  - /billing/invoices
  - /billing/rate-agreements
  - /billing/configuration-files/document-types
---

## What it's for
Customers is the master list of the companies you bill. Each customer record holds its code, name and address, and three settings tabs that decide how it is billed: **Billing profile** (payment terms, credit, invoice format, billing schedule, automation and requirements), **Email profile** (who receives invoices by email and what the email says) and, on an existing customer, **Broker vetting**.

Billing and customer-service staff use it to add new customers and to change how an existing customer is invoiced, for example switching them to a monthly statement or requiring a PO number before billing.

## Tasks

### Add a customer
Keywords: new customer, create customer, add client, add shipper
1. Open [Customers](/billing/configuration-files/customers).
2. Select **New customer**.
3. On the **General** tab, fill in **Status**, **Code** and **Name**, then the address (**Address Line 1**, **City**, **State**, **Postal code**).
4. Set up the **Billing profile** and **Email profile** tabs as needed, then select **Save** (or **Save & close**).

### Change how a customer is invoiced
Keywords: billing schedule, statement customer, consolidated billing, payment terms, billing cycle
1. Open [Customers](/billing/configuration-files/customers) and select the customer's row.
2. Open the **Billing profile** tab.
3. Under billing schedule, set **Invoice delivery** to **Per shipment**, **Per order** or **Statement (consolidated)**. For a statement, choose **Bill every** (the cycle) and how the period is split with **Separate invoice for each**.
4. Adjust **Payment term**, **Credit limit**, **Required document types** or the billing automation switches (such as **Auto-approve clean shipments** and **Auto-generate invoices**) as needed, then select **Save**.

### Set who receives the customer's invoices
Keywords: invoice email, bill-to email, invoice recipients, cc
1. Open [Customers](/billing/configuration-files/customers) and select the customer's row.
2. Open the **Email profile** tab.
3. Fill in **To recipients**, and optionally the **Subject line**, CC and BCC recipients and the attachment filename, then select **Save**.

### Vet a customer as a broker
Keywords: broker check, carrier intelligence, DOT lookup
1. Open [Customers](/billing/configuration-files/customers) and select the customer's row.
2. On the **General** tab, enter the **DOT number** and turn on **Vet as a broker**, then save.
3. Open the **Broker vetting** tab and select **Vet broker** to pull a vetting snapshot.

### Activate or deactivate several customers at once
Keywords: bulk status, deactivate customers
1. Open [Customers](/billing/configuration-files/customers).
2. Tick the rows you want to change.
3. Use **Update status** in the action bar that appears and pick **Active** or **Inactive**.

## Notes
Viewing the page needs read access to customers; adding one needs create access and changing one needs update access. The **Broker vetting** tab needs a carrier intelligence provider connected under integrations.

A statement customer's shipments collect in the **Statements** view of the [Billing queue](/billing/queue) until their period is billed.
