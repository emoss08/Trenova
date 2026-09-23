---
path: /shipment-management/orders
aliases: [customer orders, multi-leg orders, parent order, order number, PO, purchase order, consolidated billing]
related:
  - /shipment-management/shipments
  - /billing/queue
  - /billing/invoices
---

## What it's for
An order is the customer's commercial request that one or more shipments carry out. Each shipment attached to an order is a leg. The order holds the customer, owner, PO number and BOL, the quoted price and currency, order-level charges (such as customs brokerage or order-wide fuel), and an accounts receivable rollup across every leg. Customer service and billing staff use it to track multi-leg work and invoice the legs together.

## Tasks

### Create an order
Keywords: new order, add order, open an order
1. Open [Orders](/shipment-management/orders).
2. Select **New order**.
3. Under **General information**, choose the **Status**, **Customer** and **Owner**, and enter the **PO number** and **BOL**. The **Order number** is generated for you.
4. Under **Commercial**, choose the **Currency** and enter the **Quoted amount** and **Base amount**.
5. Select **Save**.

### Attach shipments to an order
Keywords: add legs, link shipments to order, group loads
1. Open [Orders](/shipment-management/orders) and select the order.
2. Under **Legs**, select **Add legs** (or **Add first leg**).
3. Pick the shipments to attach and select **Add leg** (the button counts the legs when you pick several).
4. To take a shipment off the order, select **Detach leg** on its row and confirm. The shipment moves onto its own order.

### Invoice an order's legs
Keywords: bill order, one invoice for several loads, consolidated invoice
1. Open the order from [Orders](/shipment-management/orders).
2. Under **Legs**, tick the legs to bill (legs in a status that cannot be invoiced cannot be ticked).
3. Select the create-invoice button. Billing a single leg on its own asks you to confirm with **Create invoice**.

### Add an order-level charge
Keywords: order fee, customs brokerage charge, order-wide fuel
1. Open the order from [Orders](/shipment-management/orders).
2. Under **Order charges**, select **Add charge** (or **Add first charge**).
3. Enter the **Description** and **Amount**, then select **Add charge**. Use **Edit charge** or **Remove charge** on a row to change it later; invoiced charges are marked **Invoiced**.

### Close or cancel an order
Keywords: settle order, finish order, cancel all legs
1. Open the order from [Orders](/shipment-management/orders).
2. Under **Accounts receivable**, select **Close order** once the order is billed, or **Cancel order**, enter the reason for cancellation, and confirm with **Cancel order**.

## Notes
Needs read access to orders; creating and editing need create and update permission. The **Accounts receivable**, **Legs** and **Order charges** sections appear only on an existing order. Canceling an order cancels every remaining leg. **Close order** is offered only when the order's status is Billed.
