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

## Who holds them

| Template | Runs | Holds |
| --- | --- | --- |
| Billing assistant | in chat, as the person | every tool above |
| Billing exception agent | unattended, on queue events | the reads, review, hold, exception, send back |

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

- A preview shows twenty records; a transfer of more lists its refusals first, and a
  background transfer previews its first hundred shipments and says it is partial.
- `send_invoice` cannot send an invoice whose attachments exceed the email provider's
  limit; a person sends it from the invoice page.
- An item an agent moves to exception or onto hold raises the billing exception event; the
  desk does not start a second run on an item it already has open, but a person's later
  approval of such a proposal wakes it on the item again.
