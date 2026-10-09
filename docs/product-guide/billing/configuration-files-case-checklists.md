---
path: /billing/configuration-files/case-checklists
aliases: [ready to bill checklist, ready to close checklist, desk case checklist, case steps, billing checklist, customer checklist]
related:
  - /admin/billing-controls
  - /billing/configuration-files/customers
  - /billing/configuration-files/document-types
  - /desk
---

## What it's for
Case checklists decide what a Desk case shows between a record and what comes next. **Ready to bill** is the checklist for a case about a shipment; **Ready to close** is the one for a case about an invoice. Each step is **Required** (it keeps the record from being ready and can be the next step), **Optional** (shown and ticked, but never blocks) or **Off** (not shown).

Your organization's checklist applies to every case unless the customer billed for the record has its own, which replaces it for that customer. Billing leads use it to drop steps a customer does not need (such as notifying a customer that does not want delivery notices), put steps in the order their team works them, and add steps of their own.

## Tasks

### Change your organization's checklist
Keywords: required steps, optional step, turn off step, reorder steps
1. Open [Case checklists](/billing/configuration-files/case-checklists) and choose **Ready to bill** or **Ready to close**.
2. With **Your organization** selected, drag a step by its handle to move it, and set each step to **Required**, **Optional** or **Off**.
3. Select **Save changes**.

### Give a customer its own checklist
Keywords: customer checklist, customer does not want notification, per customer steps
1. Open [Case checklists](/billing/configuration-files/case-checklists) and choose the checklist.
2. Select **Add a customer** and pick the customer. Its checklist starts as a copy of your organization's.
3. Change the steps, then select **Save checklist**.

### Add a step of your own
Keywords: custom step, add step, manual tick, document step
1. On the checklist you are changing, select **Add a step**.
2. Give it a **Name** and choose what it is **Ticked by**: **A person on the case**, who ticks it on the Desk, or **A document on file**, which ticks it once an accepted copy of the chosen **Document type** is attached to the shipment.
3. Optionally set the **Button** text and **What the button asks the agent**, which is sent to the case's agent when the step is the next one.
4. Select **Save changes**.

### Remove a customer's checklist or go back to the default
Keywords: reset checklist, delete customer checklist, default checklist
1. Open [Case checklists](/billing/configuration-files/case-checklists) and select the customer or **Your organization**.
2. Select **Remove this customer's checklist** (the customer goes back to your organization's) or **Go back to the default**, and select it again to confirm.

## Notes
Viewing the page needs read access to billing control, and changing it needs update access.

Some steps follow rules kept elsewhere and always stay required, though they can be moved: **Proof of delivery received** and **Paperwork in** follow the customer's **Required document types** on [Customers](/billing/configuration-files/customers), **Rate confirmation matched** follows rate validation on [Billing control](/admin/billing-controls), and **Delivered**, **Nothing holding the bill**, **Posted**, **Clear of dispute** and **Paid** are steps nothing can be billed or closed without.

A step a person ticks belongs to the record, so everyone working a case about the same shipment or invoice sees the same tick and who made it.
