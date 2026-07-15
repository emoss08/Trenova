# The Order Aggregate — A Domain Design Primer

> Status: design exploration. Purpose: explain what an "order aggregate" is (in DDD terms), why Trenova needs one, and what it looks like as a concrete domain entity that fits our existing conventions.

---

## 1. The one-sentence definition

An **order aggregate** is a single commercial entity that sits **above** the shipment and owns the customer relationship, the quoted price, and the accounts-receivable rollup — while one or more shipments underneath it carry out the physical work.

Put differently: today the **shipment** is doing two jobs at once — it is both *the thing you sell to the customer* and *the thing a truck executes*. An order aggregate splits those two jobs apart.

```
                     ┌─────────────────────────────┐
                     │           ORDER             │   ← what you SELL
                     │  customer · quote · AR       │     (commercial)
                     └──────────────┬──────────────┘
                                    │ 1..many
              ┌─────────────────────┼─────────────────────┐
              ▼                     ▼                     ▼
        ┌───────────┐        ┌───────────┐        ┌───────────┐
        │ SHIPMENT  │        │ SHIPMENT  │        │ SHIPMENT  │   ← what you EXECUTE
        │  CA leg   │        │  US leg   │        │  MX leg   │     (operational)
        └───────────┘        └───────────┘        └───────────┘
```

Every gap the GitHub reporter listed — order-level AR, one invoice over many legs, per-leg executing entity under one commercial order — is a direct consequence of *not* having this split.

---

## 2. What "aggregate" actually means (DDD)

The word comes from Domain-Driven Design. An **aggregate** is a cluster of domain objects that you treat as a **single unit** for the purposes of data changes and consistency. It has three defining properties:

| Property | Meaning | In our case |
|---|---|---|
| **Aggregate root** | The one entity that is the entry point. Outside code only ever holds a reference to the root, never to the internals. | `Order` |
| **Consistency boundary** | Business invariants that must *always* be true are enforced inside the boundary, in one transaction. | "Order total = sum of leg charges + order-level charges", "an order cannot be marked Billed while a leg is still In Transit" |
| **Identity & lifecycle** | The root has its own ID and its own status lifecycle, independent of its children. | `Order.ID`, `Order.Status` (Draft → Confirmed → InProgress → Completed → Billed → Closed) |

The key mental shift: **a shipment stops being a top-level thing you create directly.** You create an *order*; shipments come into existence as the execution plan for that order. (We keep the ability to create a single-leg order in one step so day-to-day truckload entry doesn't get slower — see §7.)

### Aggregate root vs. entity vs. value object

- **Order** — aggregate root. Has identity, has a lifecycle, is persisted on its own.
- **Shipment** — an entity *inside* the boundary that also happens to be an aggregate root of its *own* smaller aggregate (moves/stops/assignments). It has a foreign key up to the order. This is the one nuance: a shipment is big and independently useful, so it stays its own root but gains a nullable parent pointer. This is the standard "aggregate referencing another aggregate by ID" pattern — not nesting shipments physically inside the order.
- **OrderCharge** — a value-ish entity (order-level accessorials/quoted base that don't belong to any single leg): fuel surcharge quoted at the order level, a customs brokerage fee, etc.

Rule of thumb we'll follow: **reference other aggregates by ID, own only what must be transactionally consistent.** The order owns its charges and its status; it *references* shipments and the customer by ID.

---

## 3. Why the shipment can't just keep doing this

Concretely, here's what breaks without the split, mapped to the code:

- **AR lives on the leg, not the sale.** `Invoice.ShipmentID` and `BillingQueueItem.ShipmentID` are single, non-null FKs. There is nowhere to attach "the amount the customer owes for the whole journey." (`domain/invoice/invoice.go`, `domain/billingqueue/billingqueue.go`)
- **One invoice can't span legs.** `CreateFromShipments` rejects `len(ShipmentIDs) > 1` ("Grouped invoices are not supported yet"). Even the plural signature is anticipating this exact aggregate. (`services/invoiceservice/delivery.go`)
- **No commercial continuity across executing entities.** Inter-org EDI transfers link two shipments for *status sync* (`ShipmentLink`, field ownership), but there's no shared parent that says "these two shipments are the same *sale* to the same customer." (`domain/edi/sync.go`)
- **"Consolidation" is the wrong tool.** `Shipment.ConsolidationGroupID` groups loads for *dispatch efficiency* (put nearby loads on one truck). That's operational, and it's the inverse relationship: one truck / many customers' loads. An order is the opposite: one customer's sale / many trucks' legs. Don't overload it.

---

## 4. The domain model, concretely

Here's what the aggregate root looks like as a Trenova entity, following the exact conventions already used by `Shipment` and `Customer` (composite tenant PK, PULID IDs, epoch-millis `int64` timestamps, `decimal.NullDecimal` money, optimistic `Version`, ozzo `Validate`). This is illustrative, not final.

```go
package order

type Order struct {
	bun.BaseModel             `json:"-" bun:"table:orders,alias:ord"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	// Identity + tenancy (mirrors Shipment/Customer composite PK)
	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	// Commercial ownership — the whole point of the aggregate
	CustomerID  pulid.ID    `json:"customerId"  bun:"customer_id,type:VARCHAR(100),notnull"`
	OrderNumber string      `json:"orderNumber" bun:"order_number,type:VARCHAR(100),notnull"` // seqgen, like ProNumber
	Status      OrderStatus `json:"status"      bun:"status,type:order_status_enum,notnull,default:'Draft'"`

	// The customer-facing quote: base + extras (extras live in OrderCharges)
	QuotedAmount   decimal.NullDecimal `json:"quotedAmount"   bun:"quoted_amount,type:NUMERIC(19,4),notnull,default:0"`
	BaseAmount     decimal.NullDecimal `json:"baseAmount"     bun:"base_amount,type:NUMERIC(19,4),notnull,default:0"`
	TotalAmount    decimal.NullDecimal `json:"totalAmount"    bun:"total_amount,type:NUMERIC(19,4),notnull,default:0"` // derived: base + charges + legs
	CurrencyCode   string              `json:"currencyCode"   bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`

	// Optional commercial references the reporter mentioned
	PONumber string `json:"poNumber" bun:"po_number,type:VARCHAR(100),nullzero"`
	BOL      string `json:"bol"      bun:"bol,type:VARCHAR(100),nullzero"`

	// Lifecycle audit
	OwnerID     pulid.ID `json:"ownerId"     bun:"owner_id,type:VARCHAR(100),nullzero"`
	EnteredByID pulid.ID `json:"enteredById" bun:"entered_by_id,type:VARCHAR(100),nullzero"`
	Version     int64    `json:"version"     bun:"version,type:BIGINT"`
	CreatedAt   int64    `json:"createdAt"   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt   int64    `json:"updatedAt"   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Customer     *customer.Customer   `json:"customer,omitempty"     bun:"rel:belongs-to,join:customer_id=id"`

	// The children — legs of the journey. Owned by reference (FK on the shipment side).
	Shipments []*shipment.Shipment `json:"shipments,omitempty" bun:"rel:has-many,join:id=order_id"`
	// Order-level charges (customs brokerage, quoted fuel, etc.) that aren't tied to one leg.
	Charges []*OrderCharge `json:"charges,omitempty" bun:"rel:has-many,join:id=order_id"`
}
```

And the single change on the child side:

```go
// domain/shipment/shipment.go — add one nullable FK. Nullable = backward compatible.
OrderID pulid.ID `json:"orderId" bun:"order_id,type:VARCHAR(100),nullzero"`
Order   *order.Order `json:"order,omitempty" bun:"rel:belongs-to,join:order_id=id"`
```

That `nullzero` matters: **every existing shipment keeps working with `OrderID` empty.** Nothing is forced to have an order on day one; the aggregate is adopted incrementally.

### The status lifecycle

The order has its *own* lifecycle, distinct from any shipment's status. A first cut:

```
Draft ──► Confirmed ──► InProgress ──► Completed ──► Billed ──► Closed
  │            │                                        
  └──► Canceled ◄───────────────────────────────────────
```

- `Draft` — quote being built, legs not yet dispatched.
- `Confirmed` — customer accepted the quote; legs (shipments) get created/tendered.
- `InProgress` — at least one leg is executing.
- `Completed` — all legs delivered.
- `Billed` — a single (grouped) invoice has been issued for the order.
- `Closed` — paid/settled.

The order status is **derived from, but not equal to,** its legs' statuses. That derivation is exactly the kind of invariant the aggregate root exists to own.

---

## 5. The invariants the root enforces

This is what makes it an aggregate and not just a table with a foreign key. All of these are enforced *inside* the order's transaction:

1. **Money rolls up.** `TotalAmount == BaseAmount + Σ(OrderCharges) + Σ(leg charges attributed to the order)`. Recomputed whenever a leg or charge changes.
2. **Status is monotonic and consistent with legs.** You cannot move an order to `Completed` while any leg is still `InTransit`. You cannot move to `Billed` before `Completed`.
3. **Tenancy is uniform.** Every child shipment shares the order's `BusinessUnitID`/`OrganizationID` (or, in the cross-entity case, is an *internal EDI-linked* shipment in a sibling org — see §6).
4. **One customer per order.** Legs may be executed by different operating companies, but the *commercial* customer is singular and lives on the order.
5. **Billing gate.** An order becomes billable only when all legs reach `ReadyToInvoice`. This replaces the current per-shipment billing-readiness gate as the *commercial* trigger (the per-shipment gate still governs each leg operationally).

If you remember one thing: **the invariants are the aggregate.** The struct is just where they live.

---

## 6. How it composes with what already exists

The order aggregate doesn't replace any current machinery — it sits above it and gives the existing pieces a shared parent.

| Existing concept | Relationship to the order |
|---|---|
| **Shipment → Move → Stop → Assignment** | Unchanged. Each shipment is still executed exactly as today. A shipment just optionally points up to an order. |
| **Split pickup/delivery (`StopType`)** | Unchanged — still a within-shipment concern. |
| **Inter-org EDI transfer + `ShipmentLink`** | This is how a *leg executed by a different legal entity* gets created. The order references the source shipment; the linked target-org shipment is the same leg executed by the sibling org. The order gives those linked shipments the commercial parent they currently lack. |
| **`ConsolidationGroupID`** | Orthogonal. Consolidation groups loads across customers for dispatch; the order groups legs within one customer's sale. A shipment can have both. |
| **Invoice / BillingQueue** | Gains an `OrderID` path. This is the grouped-invoice work (Layer 2 in the extension plan) — move shipment attribution onto invoice *lines*, add `Invoice.OrderID`, drop the `len > 1` guard. |
| **`AdditionalCharge` (per-move)** | Still exists per leg. `OrderCharge` is the new sibling for charges that belong to the *sale*, not a leg. |

Cross-border example, end to end:
1. Sales creates an **Order** for the customer, quoted amount $X (`Draft`).
2. Customer accepts → `Confirmed`. Three **Shipments** (CA, US, MX legs) are created under `OrderID`.
3. The US and MX legs are executed by sibling operating companies → created via **inter-org EDI transfer**, `ShipmentLink` keeps status in sync, each still points up to the same `OrderID`.
4. All three legs deliver → order derives to `Completed`.
5. **One grouped invoice** is issued against the `OrderID`, itemizing all three legs + order-level customs charge → `Billed`.
6. Payment settles → `Closed`.

---

## 7. Deliberate non-goals (so scope stays honest)

- **Don't force an order for single-truckload entry.** 90% of freight is one leg. The create-shipment path should transparently create a one-leg order behind the scenes (or allow `OrderID` to stay null and treat a bare shipment as its own commercial unit). The aggregate must not slow down the common case.
- **The order does not own execution details.** Drivers, equipment, stop actuals, and appointment windows stay on the shipment/move/stop. The order never reaches down into a stop.
- **Buy-side cost and a `Route`/customs-checkpoint aggregate are separate future domains.** The order gives you order-level *revenue/AR*. Per-leg carrier cost and first-class border checkpoints are their own additions and are explicitly out of scope for the first cut — they bolt onto the order later.

---

## 8. The mental model to walk away with

- **Shipment = a truck's job.** Order = a customer's purchase.
- The order is an **aggregate root**: it has its own identity, its own status lifecycle, and it *owns the business rules* that must stay true across all its legs (money rollup, billing gate, single customer).
- Shipments stay independent aggregates and simply gain a **nullable pointer up** to their order — so adoption is incremental and nothing existing breaks.
- Everything the reporter asked for (order-level AR, one invoice over many legs, multi-entity execution under one sale) falls out naturally once the commercial layer and the operational layer are separate objects instead of one overloaded shipment.
