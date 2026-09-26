# Agent tools for the billing lifecycle

What an agent can do between a delivered shipment and an invoice in the customer's hands,
how far each step may run without a person, and why four of them never do.

Read [agent-runtime.md](agent-runtime.md) for tiers, egress classes and taint, and
[proposal-previews.md](proposal-previews.md) for how a proposal is previewed and approved.
The generated [ai-tool-safety.md](ai-tool-safety.md) is the authority on each policy; this
page explains them.

## The lifecycle

```
list_billing_transfer_candidates ──► transfer_to_billing
                                          │ (one queue item per payer)
list_billing_queue_items / get_billing_queue_item
        │
        ├─► hold_billing_queue_item / move_billing_item_to_exception /
        │   send_billing_item_back_to_ops / transition_item_to_in_review /
        │   assign_billing_queue_biller
        │
        ├─► approve_billing_queue_item ──► (draft invoice) ──► post_invoice ──► send_invoice
        └─► cancel_billing_queue_item
```

Every write goes through the service the page uses, so an agent's write and a person's
write cannot differ: `ShipmentService.BulkTransferToBilling` and `billingtransferservice`,
`BillingQueueService.UpdateStatus` and `AssignBiller`, `invoiceservice.Post` and `Send`.

## The tools

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_billing_transfer_candidates` | shipment / read | reads only | automatic | — |
| `list_billing_queue_items` | billing queue / read | reads only | automatic | — |
| `get_billing_queue_item` | billing queue / read | reads only | automatic | — |
| `transfer_to_billing` | shipment / update | inside | Propose → Automatic | no |
| `transition_item_to_in_review` | billing queue / update | inside | Propose → Automatic (a held item: Propose) | no |
| `hold_billing_queue_item` | billing queue / update | inside | Propose → Automatic | no |
| `move_billing_item_to_exception` | billing queue / update | inside | Propose → Automatic | no |
| `send_billing_item_back_to_ops` | billing queue / update | inside | Propose → Automatic | no |
| `assign_billing_queue_biller` | billing queue / assign | inside | Propose → Automatic | no |
| `approve_billing_queue_item` | billing queue / update | money | Propose → Propose | yes |
| `cancel_billing_queue_item` | billing queue / update | money | Propose → Propose | yes |
| `post_invoice` | invoice / update | money | Propose → Propose | yes |
| `send_invoice` | invoice / submit | sent outside | Propose → Propose | yes |

**Reads.** `list_billing_transfer_candidates` is the transfer dialog's list (the same
candidate predicate through `ListBillingTransferCandidateIDs`, oldest first, with its
search, status, customer and delivery-date filters) and each row says what a transfer would
do: `Transfer`, `MarkReadyAndTransfer`, `Refused` with the failure code, missing documents
and rate or validation issues, or `ReturnToOperations`. It is decided by
`ShipmentService.PlanBillingTransfers`, which runs the transfer's own checks
(`transferGate`, the readiness evaluator, `billingTransferPolicyViolation`) and reads the
shipments, payers, policy, documents and service failures once for a page rather than once
per shipment. `list_billing_queue_items` lists through the queue repository the billing
page uses and, like the page, includes posted items only when a status filter names
Posted. `get_billing_queue_item` adds the payer's charges, detention holds, notes, what the
shipment still lacks, whether it can be approved and the invoice approval made.

**Transfer.** `transfer_to_billing` takes the shipments as a record subset (below), so the
person approving can untick some. Up to 100 go through the synchronous bulk transfer; more
start a background transfer run (scope Selected) as the person who asked, which allows one
active run per person. Its preview plans the same shipments with `PlanBillingTransfers` and
shows one record per shipment, refusals first.

**Decisions that make no money.** Hold, exception, send back to operations, review and
assignment change who works an item next and are undone by moving it again. The billing
queue lets an agent principal make exactly these moves (`billingqueueservice.AgentMayMoveTo`:
InReview, OnHold, Exception, SentBackToOps) and refuses it Approved, Canceled, Posted and a
return to ReadyForReview. Hold, exception and send-back write notes a biller or operations
reads next, so each carries a taint hold: a run that has read outside text proposes them.

**Decisions a person makes.** Approving creates the invoice a customer is billed on;
canceling drops a charge for good; posting books the receivable and hands it to the
accounting system and the customer's EDI; sending puts the invoice in the customer's inbox.
Each stops at Propose, runs only when `ToolExecuteParams.ApprovedFromProposal()` (the
executor runs it as the approver), and is refused otherwise. There is no model approval:
unattended approval comes only from the organization's deterministic auto-approve rule,
applied when a shipment is transferred, and hands-off posting only from the billing-control
auto-post setting. Approval's preview shows the draft invoice
(`invoiceservice.PreviewApprovalInvoice`, the plan `CreateFromApprovedBillingQueueItem`
saves), whether auto-post follows, and refuses an item a detention charge still holds.
`post_invoice` previews `invoiceservice.PreviewPost`, which is Post's own plan (`planPost`,
`planInvoicedLegs`, `planInvoiceJournal`) plus the accounting connections the sync is queued
for (`AccountingSyncPlanner`) and the EDI 210 plan. `send_invoice` takes only the invoice;
the recipients, wording and attachments are `PlanSend`'s, and a send whose attachments would
go as download links is refused, since those links are built against the browser's address.

## After the invoice: receivables

The same rules carry past posting. Every write calls the service its page calls
(`invoiceservice`, `invoiceadjustmentservice`, `customerpaymentservice`,
`invoicedisputeservice`, `invoicerunservice`, `latechargeservice`, `invoiceshareservice`),
and each preview is that service's own plan, read-only, so it refuses what the write would.

| Tool | Class | Most it may do | Runs only from a person's approval |
| --- | --- | --- | --- |
| `update_invoice_draft` | inside | Ask first (a change of recipients: Propose) | no |
| `generate_invoice_pdf` | inside | Automatic | no |
| `create_invoice`, `create_invoice_memo`, `void_invoice` | money | Propose | yes |
| `send_invoice_edi` | sent outside | Propose | yes |
| `save_invoice_adjustment_draft` | inside | Automatic (from outside content: Propose) | no |
| `submit_invoice_adjustment`, `approve_invoice_adjustment` | money | Propose | yes |
| `reject_invoice_adjustment` | inside | Propose | yes |
| `apply_customer_payment`, `reverse_customer_payment` | money | Propose | yes |
| `apply_credit_memo`, `unapply_credit_memo` | money | Propose | yes |
| `open_invoice_dispute`, `withdraw_invoice_dispute` | inside | Ask first (from outside content: Propose) | no |
| `resolve_invoice_dispute` | inside | Propose | yes |
| `build_invoice_run` | inside | Automatic | no |
| `adjust_invoice_run_membership` | inside | Automatic (from outside content: Propose) | no |
| `cancel_invoice_run` | inside | Ask first | no |
| `commit_invoice_run`, `bill_statement_now`, `assess_late_charges` | money | Propose | yes |
| `share_invoice` | inside | Propose | yes |

`create_invoice` covers the four create mutations (one invoice or one per shipment, from
shipments or an order) through `invoiceservice.PlanCreateInvoices`. A memo is always manual
and never auto-posted. `generate_invoice_pdf` refuses whenever the customer's billing profile
would email the invoice the moment its PDF exists, so it cannot send by the back door.
`submit_invoice_adjustment` covers the single, draft and bulk submit routes. A preview of
a planned draft or memo shows no number, since numbers are issued when a record is made.

The reads that pick targets: `get_invoice` (now with each line's id), `list_invoices` (a
bill type filter), `list_invoice_adjustments`, `get_invoice_adjustment`,
`list_invoice_disputes`, `list_credit_memo_applications`, `list_invoice_runs`,
`get_invoice_run`, `list_open_statements` and `list_invoice_share_candidates`. Amounts are
left out when the reader's data access does not reach invoice totals.

Pinning: disputes, runs and payments pin to their version; adjustments to theirs (record
kind `invoice_adjustment`); a credit memo application to its `updated_at` (record kind
`credit_memo_application`), which moves only when it is unapplied. `bill_statement_now` is
unpinned: an open statement is computed when asked and has no row.

## Who holds them

| Template | Runs | Holds |
| --- | --- | --- |
| Billing assistant | in chat, as the person | the lifecycle up to an invoice in the customer's hands and its corrections: transfer, the queue decisions, `create_invoice`, `update_invoice_draft`, `generate_invoice_pdf`, `post_invoice`, `send_invoice`, `send_invoice_edi`, `void_invoice`, `create_invoice_memo`, the adjustment tools, invoice runs and statements, and `share_invoice` |
| Receivables assistant | in chat, as the person | what happens after: `apply_customer_payment`, `reverse_customer_payment`, `apply_credit_memo`, `unapply_credit_memo`, the dispute tools, `assess_late_charges`, `send_invoice` to send a copy again, `share_invoice`, and the reads `get_ar_aging`, `list_ar_open_items`, `list_collections_worklist`, `get_customer_statement`, `list_customer_payments`, `list_credit_memo_applications`, `list_invoice_adjustments` |
| Cash application agent | unattended, on bank receipt exceptions | `match_bank_receipt` and `post_customer_payment` for money that arrived at the bank |
| Billing exception agent | unattended, on queue events | the reads, review, hold, exception, send back |

The receivables assistant works cash already recorded and never posts a payment or matches a
bank receipt, so it does not do the cash application agent's job twice. A dispute settled by a
credit or a write-off is closed by receivables naming the adjustment, which the billing
assistant makes. Each prompt tells its agent to hand the other's work over when the other is on
its delegation allowlist ([agent-delegation.md](agent-delegation.md)); a template cannot carry
the allowlist itself, since it names agents by their id in one organization.

The agent permission ceiling (`permission/agent.go`) is unchanged but for being claimed: the
exception desk needs billing queue read and update and shipment read, which it already had.
Transfer (shipment update), assignment (billing queue assign) and posting and sending
(invoice update and submit) are offered only to the chat assistant, which acts as the
person, so no agent principal holds them.

The billing queue's permission resource now marks its workflow fields (status, reason,
notes, biller, dates, ids) Internal and leaves its amounts Restricted, as the invoice
resource does, so an unattended desk at Internal data access sees what an item is waiting
on without seeing what it bills.

## Known limits

- An agent definition holds at most 64 tools. Every template leaves room for at least eight of
  an organization's own (`TestTemplates_LeaveRoomForAnOrganizationsOwnTools`): the billing
  assistant holds 53 and the receivables assistant 27.
- A template is copied into an agent when the agent is made. An agent made from the billing
  assistant before collections moved to receivables keeps the tools it was saved with, and no
  existing agent gains `share_invoice`; an administrator changes either in AI control.

- A preview shows twenty records; a transfer of more lists its refusals first, and a
  background transfer previews its first hundred shipments and says it is partial.
- `send_invoice` cannot send an invoice whose attachments exceed the email provider's
  limit; a person sends it from the invoice page.
- An item an agent moves to exception or onto hold raises the billing exception event; the
  desk does not start a second run on an item it already has open, but a person's later
  approval of such a proposal wakes it on the item again.
