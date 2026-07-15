# Order Aggregate — Engineering Handoff

> Audience: a Claude Code agent (or engineer) continuing the Order aggregate work.
> Companion to `docs/design/order-aggregate.md` (the design primer). This document is
> the *implementation* map: what exists, how it's wired, the conventions to follow, and
> the remaining features needed to fully close GitHub issue #507.

Origin: issue **#507** ("Never Min") asked whether Trenova could model multi-country /
multi-leg cross-border movements as one commercial order over many shipments, with
order-level AR and one invoice covering all legs. The **Order aggregate** was built as
the answer. It is on `master` (commits `601b92042`, `c769a8444`, `42f081bcc`).

---

## 1. What an Order is (one paragraph)

An **Order** is the commercial entity above the shipment: it owns the customer, the
quote (quoted/base), currency, order-level charges, and the AR total. Shipments stay the
execution unit ("legs") and gain a nullable `order_id`. The order's **status is derived**
from its legs (`Draft → Confirmed → InProgress → Completed → Billed → Closed`, plus
`Canceled`). One customer per order. A **grouped invoice** covers all billable legs plus
order-level charges. Creating a plain shipment auto-creates a one-leg order; existing
shipments were backfilled 1:1.

---

## 2. Architecture map (backend)

Hexagonal (domain → ports → services → infrastructure), Bun ORM, Uber FX DI, gqlgen
GraphQL + Gin REST. Reads go through GraphQL connections; **writes go through GraphQL
mutations** (see conventions).

| Layer | Path | Notes |
|---|---|---|
| Domain | `internal/core/domain/order/` | `order.go` (aggregate root), `status.go` (`Derive(legs)` pure fn + `OrderStatus`), `charge.go` (`OrderCharge`), `fieldmap_gen.go` (colgen) |
| Shipment link | `internal/core/domain/shipment/shipment.go` | `OrderID pulid.ID` (nullzero) scalar only — **no `Order` relation** (avoids an import cycle; Order owns the `Shipments` has-many) |
| Ports | `internal/core/ports/repositories/order.go` | `OrderRepository` (List/Get/Create/Update/UpdateStatus/CreateInTx/GetShipmentStatuses/Attach/Detach/AddCharge/RemoveCharge/ListCharges/RecalculateTotal/CountShipmentsWithDifferentCustomer/SelectOptions) |
| Repo | `internal/infrastructure/postgres/repositories/orderrepository/order.go` | Bun impl. `RecalculateTotal` = one UPDATE summing leg `total_charge_amount` + `order_charges.amount` |
| Service | `internal/core/services/orderservice/service.go` + `validator.go` | CRUD, `AttachShipments`/`DetachShipment` (+ customer invariant), `AddCharge`/`RemoveCharge`, `finishMembershipChange` (re-derive status + `RecalculateTotal` + audit) |
| Derivation | `internal/core/services/orderderivation/service.go` | `ShipmentEventObserver` in the `shipment_event_observers` FX group; recomputes order status on leg status events (optimistic version + bounded retry) |
| Auto-order | `internal/infrastructure/postgres/repositories/shipmentrepository/shipment.go` `Create` | Mints a 1-leg order in the same tx when `entity.OrderID` is nil |
| Grouped invoice | `internal/core/services/invoiceservice/delivery.go` (`CreateFromOrder`, `collectBillableLegs`, `groupedInvoiceFromShipments`) + `service.go` (`buildInvoiceEntityForOrder`, `buildInvoiceLinesForShipment`, `markInvoicedLegs`, `invoiceLegShipmentIDs`) | `CreateFromShipments` `len>1` now delegates instead of erroring |
| Sequence | `tenant/enums.go`, `tenant/sequencecatalog.go`, `pkg/seqgen/{types,generator}.go` | `SequenceTypeOrder` (prefix `O`) + `GenerateOrderNumber` |
| Permissions | `internal/core/domain/permission/{resource_gen.go, registry.go}` | `ResourceOrder` (Operations category) |
| GraphQL | `internal/api/graphql/schema/order.graphqls`, `resolver/order.resolvers.go`, `ordermapping.go` | Queries `order`/`orders`; mutations `createOrder`/`updateOrder`/`attachOrderShipments`/`detachOrderShipment`/`addOrderCharge`/`removeOrderCharge`/`createInvoiceFromOrder`/`createInvoiceFromShipments`; `order(id)` returns lean `legs` + `charges` |
| REST | `internal/api/handlers/orderhandler/handler.go` | Still present (list/get/select-options + CRUD); client no longer uses REST for order writes |
| Migrations | `migrations/20260719000000_orders_and_grouped_invoicing.*`, `20260720000000_order_charges.*` | orders + `shipments.order_id` + backfill + invoice/BQI/line columns; `order_charges` table |
| FX wiring | `bootstrap/modules/{repositories,validators}.go`, `bootstrap/modules/api/{services,handlers}.go`, `internal/api/router.go`, `graphql/resolver/resolver.go` | |
| Integration test | `internal/core/services/invoiceservice/grouped_invoice_integration_test.go` (build tag `integration`) | Full flow: order + 2 legs + 1 order charge → grouped invoice ($350 legs + $75 charge = $425) → Post sweeps all BQIs + marks all legs Invoiced |

Grouped-invoicing schema touch points: `Invoice.OrderID`/`OrderNumber` + nullable
`ShipmentID`; `InoviceLine.ShipmentID`/`ShipmentProNumber`/`ShipmentBOL`;
`BillingQueueItem.OrderID`. BQI `number` is globally unique → only the anchor leg's BQI
carries the invoice number, siblings NULL; posting sweeps siblings via
`MarkPostedByOrderID`.

## 3. Architecture map (client, `client/`)

| Piece | Path |
|---|---|
| Type + schemas | `src/types/order.ts` (`orderSchema`, `OrderFormValues` = `z.input`, `orderChargeFormSchema`) |
| GraphQL ops | `src/graphql/operations/order/{table,detail,mutations}.graphql` → `pnpm graphql:codegen` |
| GraphQL calls | `src/lib/graphql/order.ts` (`fetchOrderDetail`, `createOrder`/`updateOrder` via `toOrderInput`, attach/detach, add/remove charge, `createInvoiceFromOrder`), `src/lib/graphql/order-table.ts` |
| Table | `src/routes/order/_components/order-table.tsx` (row-click opens the panel) |
| Edit sheet | `order-panel.tsx` — `TabbedFormEditPanel` (create uses `FormCreatePanel`), `mutationFn` = GraphQL; History tab; owner in header |
| Form | `order-form.tsx` — `FormSection`s (General / Commercial) + inline `OrderLegsSection` + `OrderChargesSection` (edit mode) |
| Legs | `order-legs-section.tsx` + `add-leg-dialog.tsx` (shipment picker scoped to the order's `customerId`) |
| Charges | `order-charges-section.tsx` (local `useForm` + `zodResolver(orderChargeFormSchema)`, `NumberField` amount) |
| Autocomplete | `OrderAutocompleteField` in `src/components/autocomplete-fields.tsx`; on the shipment billing form for `orderId` |
| Registries | `Resource.Order` (`types/permission.ts`), `/orders/` (`types/server.ts`), nav (`config/navigation.config.ts`) |

---

## 4. Conventions & gotchas (read before editing)

- **GraphQL-only writes.** New features use GraphQL for everything incl. mutations, never
  REST. Pass a `mutationFn` to the shared form panels instead of the REST `url`. Map form
  values → the generated `*Input`; form values are `z.input` (decimals may be strings),
  entities are `z.output` — add an `<Entity>FormValues = z.input<typeof schema>` type for
  the mutation signatures.
- **Table fragment must select every editable field** (e.g. `ownerId`) — the edit panel
  seeds defaults from the table row; a missing field shows blank though it saved.
- **Do NOT run `mockery`** (it pins the user's CPU). Hand-patch `internal/testutil/mocks/*`
  when an interface changes (compact `_mock.Called(...)` methods are fine).
- **Codegen chain after backend edits:** `go generate ./internal/infrastructure/database/colgen/...`
  (buncolgen + domain `fieldmap_gen.go`), `go run github.com/99designs/gqlgen generate`,
  `go generate ./internal/api/graphql/projection/...`. gqlgen relocates helper funcs out
  of `*.resolvers.go` — keep helpers in `*mapping.go`.
- **Sandbox can't run a live server** (a booted API gets SIGTERM → exit 144; foreground
  `sleep` is blocked too). Verify via `go build ./...`, targeted `go test`, a
  `timeout 45 ./build/trenova-cli api run` FX-boot check, `docker exec db psql -U postgres -d trenova_go_db`,
  and `task db-migrate`. Integration tests use testcontainers and also get killed here —
  they still run under `task test-integration` locally.
- **Lint:** exhaustive-switch requires `//nolint:exhaustive // reason` even with a
  `default:`. Prefer `bun.List(...)` over deprecated `bun.In(...)`. Run `golines -w` +
  `golangci-lint run ./path/...` per package.
- **Client typecheck:** `npx tsc -b --force` (project note: `pnpm tsc --noEmit` is false-green).
  A React Doctor pre-commit hook warns but does not block; `react-doctor --staged --fail-on warning`
  to inspect.
- **Import cycle:** `order` imports `shipment` (owns `Shipments`); shipment must NOT import
  `order` (scalar `OrderID` only).
- **Migrations** are `.tx.up.sql`/`.tx.down.sql`; composite tenant FKs are
  `(x_id, organization_id, business_unit_id) REFERENCES t(id, organization_id, business_unit_id)`.
  Enum values: `ALTER TYPE "..._enum" ADD VALUE IF NOT EXISTS '...'`. PULIDs in SQL
  backfills: `CONCAT('prefix_', replace(gen_random_uuid()::text,'-',''))`.

## 5. Deliberate deviation from the design primer

`Order.TotalAmount = Σ(leg total_charge_amount) + Σ(order_charges.amount)` — i.e. what the
grouped invoice actually sums to. The primer's literal formula also added `BaseAmount`,
which would double-count against the invoice. `QuotedAmount`/`BaseAmount` stay as
quote-side reference fields. If product wants the primer's formula, change
`orderRepository.RecalculateTotal`.

---

## 6. #507 coverage — what's DONE vs REMAINING

**Done** (issue asks satisfied): top-level order; order → many shipments; one customer
invoice over many legs (grouped invoicing); order-level AR/total rollup; quoted/base +
**order-level extra charges**; per-leg execution by different legal entities (existing
inter-org EDI transfers now share the `order_id`); derived order status; auto 1-leg order.

**Remaining to *completely* support Never Min's request** — these were explicitly
scoped out of the first cut (design primer §7). Listed in recommended build order:

1. **Order origin & final destination.** The reporter asked for "an origin and final
   destination for the order." Today origin/dest live on each leg's stops. Add
   order-level origin/destination — simplest as *derived* GraphQL fields on `Order`
   (first pickup of the first leg → last delivery of the last leg, mirroring
   `Shipment.ShipperStop()`/destination logic in `domain/shipment/shipment.go`), or as
   stored `OriginLocationID`/`DestinationLocationID` if they must be set independently of
   legs. Smallest/highest-value item; no new aggregate.

2. **Route aggregate + route-specific pricing.** The reporter asked for "a route concept
   representing the full journey" and "route-specific pricing." A `Route` groups the
   order's legs into an ordered journey with its own rating. Decide: is `Route` a separate
   aggregate referenced by the order, or is the ordered set of legs on the order already
   "the route" + a route-level rate? Recommendation: start with an ordered `sequence` on
   the order↔leg relationship + an order/route-level pricing model (reuse the formula
   template / rating engine, e.g. `formulatemplate` + `RatingDetail` on shipment) before
   introducing a standalone `Route` entity. This is the largest lift.

3. **Customs / border checkpoints as first-class objects.** "one or more intermediate
   border/customs handoff points." Model an `OrderCheckpoint` (or `RouteCheckpoint`)
   between legs: a border-crossing/customs-handoff entity with a location, the two legs it
   joins, and handoff status. Mirror the `OrderCharge` vertical slice
   (`domain/order/charge.go` → migration → repo add/list/remove → service → GraphQL type +
   mutations → inline client section). Depends on the route ordering from (2).

4. **Per-leg supplier/carrier + buy-side cost.** "potentially different suppliers/carriers
   for each leg" and (reporter's Q4) "supplier/vendor cost per leg." Trenova is asset-based
   today (worker/tractor/trailer; revenue-only fields). This needs a new
   carrier/supplier/vendor domain and a per-leg **buy-side cost** on the shipment/leg
   (distinct from the revenue-side `FreightChargeAmount`/`TotalChargeAmount`). Largest new
   domain; enables order-level margin (revenue − buy-side cost). The only existing buy-side
   artifact is inbound EDI 210 parsing (`domain/edi/carrierinvoice.go`).

5. **Order-level margin / P&L rollup** (follow-on to 4). Once buy-side cost exists, roll
   revenue vs. cost to the order for a margin view.

### Suggested sequencing
(1) order origin/destination → (2) route ordering + route pricing → (3) customs
checkpoints → (4) supplier/buy-side cost → (5) margin. Each is an independent, shippable
increment; (3) and (4) can proceed in parallel once (2) lands.

---

## 7. How to add a field/entity (the vertical slice)

Backend: domain struct (composite tenant PK, PULID `BeforeAppendModel`, ozzo `Validate`,
getters, `GetPostgresSearchConfig`) → migration → colgen → port interface → Bun repo →
service (+validator) → FX wiring (`bootstrap/modules/*`) → GraphQL (schema + gqlgen
regen + fill resolver stub + mapping + `resolver.go` Params) → REST handler + `router.go`
(only if a REST surface is needed) → hand-patch mocks. Model on the `servicetype` slice
for simple entities, or `order`/`OrderCharge` for aggregate-scoped ones.

Client: `types/<entity>.ts` (zod) → `graphql/operations/<entity>/*.graphql` +
`pnpm graphql:codegen` → `lib/graphql/<entity>.ts` (mutations via `requestGraphQL`) →
`routes/<entity>/` (table + panel with `mutationFn`) → registries (`Resource`,
`server.ts`, router, nav) → `npx tsc -b --force`.

## 8. Verification checklist

```
cd services/tms
go build ./...
go test ./internal/core/domain/order/... ./internal/core/services/invoiceservice/...
golangci-lint run ./internal/core/services/orderservice/... ./internal/infrastructure/postgres/repositories/orderrepository/...
task db-migrate                                   # apply migrations to local DB
timeout 45 ./build/trenova-cli api run            # FX boot / route registration check
# integration (local, not in the sandbox):
go test -tags integration -run TestGroupedInvoiceFromOrderEndToEnd ./internal/core/services/invoiceservice/

cd client
pnpm graphql:codegen
npx tsc -b --force
```
