# Detention charges

How a detention occurrence becomes money on a shipment, and how a decision on it reaches the
invoice. Read this before changing `detentionsync`, `detentionreview`, `detentioncharge`,
`detentionbillingservice`, the shipment's detention line, or anything that approves, waives
or disputes detention.

## One line per accessorial

Each billed stop is a `detention_occurrences` row. The shipment bills them as **one**
system additional charge per accessorial (`IsSystemGenerated`, `IsDetention`), whose amount
is the sum of its chargeable occurrences (`Chargeable()`: Pending, Approved or Billed, with a
positive amount) and whose `DetentionOccurrenceIDs` link them; saving the shipment points each
occurrence's `additional_charge_id` at it. Two functions in `domain/detentioncharge` build it:

- `Reconcile` is the rating pass's fold (`shipmentcommercial.syncDetentionFromPolicyEngine`):
  it recomputes every stop, then rewrites the lines, adding and removing as needed.
- `Settle` is a decision's fold: it rewrites only the lines the shipment already carries for
  the accessorials the occurrences name, sets each to what its occurrences still bill and
  removes one left with nothing. It never rates and never adds a line, so it runs on a
  shipment that is locked in billing.

## The rule that bites

**A decision on an occurrence settles its shipment's line in the same transaction.** Waive,
dispute (`detentionreview`) and anything new that changes whether an occurrence is chargeable
calls `settleShipmentCharges`, which saves the shipment through `UpdateDerivedState`,
recomputes the order and writes what moved on the shipment thread. Leaving it to the next
rating pass is how a waived $1,650 was invoiced: once a shipment is in billing nothing rates
it again, and the queue item bills the line as it stands.

Approving a queue item settles once more before the draft is built
(`invoicecreationservice.settleDetentionCharges`, through
`DetentionBillingService.SettleCharges`), in memory for the approval preview and saved for
the approval itself, so a line that drifted from its occurrences never bills.

After an invoice carries the line, occurrences are Billed and a waiver goes through the credit
path (`detentioncreditservice`), never `Settle`.

## Where people decide

The detention desk, the Desk's detention card, the billing queue's holds notice and the
detention line on the billing queue's charges tab all approve and waive through the same
mutations (`approveDetentionOccurrence`, `waiveDetentionOccurrence`, detention `update`).
A billing transfer lists the held detention each shipment carries
(`BillingTransferDecision.HeldDetention`) so nobody sends one into billing unwarned.
