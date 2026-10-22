# Reference views: showing another record on a screen

Most screens show more than one kind of record. A shipment shows its customer, its stops'
locations, its drivers and its carrier; an invoice shows its customer; a tractor shows its
drivers. Whoever may read the shipment may see enough of each record it links to to identify and
use it, and no more. The rest of a linked record belongs to that record's own kind and needs that
kind's read.

What a linking screen may always show is the kind's **reference view**. A reader who may read the
shipment sees its customer's name, code and billing address, and its driver's name; the customer's
credit limit and the driver's phone number need customer read and worker read.

## The rule

A record is reached in one of four ways, and the way decides what is checked:

| Reached as | Example | Checked |
|---|---|---|
| Root | what `shipment(id:)` returns, through a connection's edges or a mutation's payload | the root resolver's own permission check; the fields are the record's own. A shipment, order, invoice or worker a payload carries is linked for a caller who cannot read its kind, unless the mutation declares `@authorizes` |
| Self-scoped | anything under `mySettlements` | nothing more: every record there is the caller's own (`selfscoped.Resolvers`) |
| Part of another | a shipment's moves and stops | the record it is part of: a kind whose `ParentResource` family is the carrier's |
| Linked | a shipment's customer, a tractor's driver | the reference view always; every other field needs the kind's read |

A kind is part of another only through its `ParentResource` chain. A kind granted on its own
has no parent, so wherever it is carried it is a linked record: a worker's leave case
(`worker_leave`) or drug test (`worker_dot_test`) under its worker needs that kind's read, and
since nothing links to either it has no view, so without that read it shows its id alone. An
injury case, a disciplinary action, a performance review or a benefit plan carried by a worker
shows its reference view and nothing more. The reverse holds too: a worker's safety event
carried by a disciplinary action is a linked record of the worker's family, so the discipline
operations select `...WorkerSafetyEventReference`. See "Parent resources" in
[permissions.md](permissions.md#parent-resources) for when a kind should have a parent.

A linked record's fields outside its view, read without the kind's read:

- a nullable field is answered `null`, and the response names its path, without list indices, in
  `extensions.withheldByAccess`;
- a non-null field is refused with an authorization error. An operation that selects one must
  declare `@requiresRead`, and CI fails until the field is nullable (below).

Sensitivity still applies on top. A person's role ceiling withholds a field they see only through a
view if it ranks above that ceiling; an agent's data access ceiling applies to every field, view or
not. Every view field ranks Internal unless its resource ranks it otherwise, and none may rank above
Internal.

## Where it is decided

- **Declaration.** Each kind's `ReferenceView` sits on its definition in
  `internal/modules/identity/domain/permission/resources_*.go`, next to its field sensitivities.
  Every view also includes `id`, `organizationId`, `businessUnitId`, `version`, `createdAt` and
  `updatedAt`, which carry nothing about a record's content. A kind that only REST responses carry
  still has a view: an invoice's PDF and attachments show a document's `originalName`, `fileType`
  and `fileSize` (never its stored file name, storage path or checksum), its billing queue item
  its `number` and `status`, an email attempt's email log its `status`, `sentAt`, `failedAt` and
  `providerMessageId` (never its recipients, subject, body or error), and an email profile shows
  its `name` (never its sender address or provider). Views are fixed in the backend: there
  is no table for them and no screen to edit them. An organization decides who sees whole linked
  records through bundles and job templates, and how sensitive a field may be through the role's
  ceiling.
- **Enforcement.** One package, `internal/modules/identity/app/refview`, answers
  `Decide(ctx, access, kind, field, reach)`, and `Filter` for a list of fields. `PersonAccess` is a
  person's grants for one request, `AgentAccess` an agent read's. Every surface asks it; none
  reimplements the rule.
- **GraphQL.** `FieldGuardExtension` (`internal/api/graphql/gqlext/fieldguard.go`) maps every
  object type to its kind (a type named for a resource, or one that declares
  `@kind(resource:)`), works out how the record holding each field was reached, and asks
  `refview`. A new type, field or resolver is covered with no permission code. People and agents
  go through the same guard.
- **REST.** Every handler writes its response with `helpers.JSON`, which hands the value to the
  request's guard (`internal/api/responseguard`, installed on every signed-in route group). The
  guard walks the value by reflection with a plan cached per Go type: a domain struct named for a
  resource is that kind (`customer.Customer`, `worker.Worker`), `namedKinds` holds the few whose
  name does not spell theirs (`shipment.Stop`, `billingqueue.BillingQueueItem`,
  `worker.WorkerLeaveCase`, `worker.TimeClockEntry`, `worker.WorkerDOTViolation`), and any other
  struct is part of the record that carries it. The value a route returns (each item of a list or
  page) is the root record; a record of a kind reached through it is linked, unless it is part of
  the root's family (a shipment's moves and stops), and so on down. A linked record of a kind the
  caller cannot read is written as an object with all its keys and `null` for each field
  `refview` withholds, and the response names it, without list indices, in the
  `Withheld-By-Access` header (`moves.assignment.primaryWorker`). The handler's own value is never
  changed. Protected field groups are withheld from every record in the response through the
  entity's `WithholdFields`. Select-option routes mark their records `responseguard.References()`
  (they authorize no kind, so each is shown as a linked record), and routes that return only the
  caller's own records (`/users/me`, the driver portal) mark them `responseguard.SelfScoped()`.
  On the client, a REST schema carries a linked record as
  `linkedRecordSchema(kind, schema)` (`@trenova/shared/lib/linked-record`), which makes every field
  outside the view nullable, and a screen draws `WithheldValue` where one it shows is null.

- **Reports and exports.** The report catalog generator classifies every field and edge
  `reference` or `kind_read` from its resource's reference view (`reportcatalog.Classify`);
  entities that are part of another record are marked `withinRecord` and are always `kind_read`.
  The compiler (`reporting/compiler/authorize.go`) requires read, and export for a run, on the
  root entity and on any joined entity a `kind_read` field or edge is used from; a joined entity
  used only through reference fields needs no read of its own, and each of those fields is decided
  by `refview` through `permissionports.ReferenceViews`, so the role's ceiling applies inside the
  view. Table exports and dashboard exports compile through the same check. Protected fields are
  declared in the catalog (`protected: Personal|Margins`) and refused without the group.
- **Row-level security.** The `read_scope` policies on shipments, orders, invoices and workers
  narrow only by an `own` or `team` grant. A kind the reader cannot read at all is not narrowed in
  the database, so a linked record of those four kinds is returned and shows its reference view,
  the same with RLS on or off. A root read of one of them is refused by its resolver, route, tool
  or report before it queries, and a record of one that a mutation's payload carries (a training
  record's worker) is linked for a caller who cannot read its kind unless the mutation declares
  `@authorizes` for it. See "Data scopes" in [permissions.md](permissions.md#data-scopes).

## Search, analytics and insights

Three surfaces put another kind's data into a payload of their own rather than a record, and each
asks the same port, `permissionports.ReferenceViews`, which refview implements:

- **Global search** (`globalsearchservice`). `hitMetadata` in `enrich.go` classifies every metadata
  key a hit can carry: the hit's own record, a linked record's fields (a shipment hit's
  `customerName` is the customer's `name`, `origin` a location's `city` and `state`, `driverName` a
  worker's `firstName` and `lastName`), or a figure about a kind no view carries (`openBalance` needs
  accounts receivable read). Each linked value is decided field by field for the person, ceiling
  included; a service type has no view, so `serviceTypeCode` needs service type read. A key with no
  classification is dropped, and `TestEveryHitMetadataKeyIsClassified` fails until it has one.
- **Shipment analytics** (`/analytics?page=shipment`, `shipmentAnalytics`). The page needs shipment
  read; `shipmentprovider/access.go` then gates each card. Names of customers, locations and drivers
  are view fields, so they stay, held to the ceiling (a withheld one is `null`; the lane heatmap,
  whose regions come from location states, is `null` as a whole). A customer's revenue, share,
  loads and trend are figures about the customer and need customer read, and the customer mix
  lists names in revenue order, so the order alone would rank them: without customer read the
  whole customer mix is `null`, and the card says the reader needs access to customers
  (`WithheldPanel`). The average margin, and how many loads lost money, need the margins field
  group (see "Margins" in [permissions.md](permissions.md)). The client draws every other
  `null` with `WithheldValue`, and a protected field with `WithheldValue group="Margins"`.
- **Insights** (`insightservice`). A finding names a customer, a location or a driver beside figures
  about it, which no view carries, so the reader needs read on every kind it names, not a view.
  Each detector declares those kinds in `Resources()`; each finding lists the kinds it names in
  `Names`, and a finding naming a kind its detector does not declare is dropped at refresh.
  `TestEveryDetectorDeclaresEveryKindItsFindingsName` runs every detector and checks both.

## Building a screen

1. **Select the reference fragment.** `go run ./internal/api/graphql/gqlext/refgen` (from
   `services/tms`) writes one fragment per kind with a view to
   `client/packages/graphql/src/operations/reference/reference.graphql` (`CustomerReference`,
   `WorkerReference`, …) and the client's copy of the views to
   `@trenova/shared/types/generated/reference-views`. A linking screen selects
   `customer { ...CustomerReference }` and needs nothing else. The fragment selects `id` and the
   view's own fields; the bookkeeping fields are allowed but not selected.
2. **When the screen needs more, say so.** Selecting a linked record's field outside its view
   needs `@requiresRead(kind: "customer")` on the operation (repeat it for each kind). The field
   must be nullable, and the screen draws the field's withheld state where it is null for a reader
   without that kind's read. Ask first whether the job needs the full record: if it does, the
   bundle for that job should grant the kind's read.
3. **A card or overview that is about one record.** A root field that returns a card wrapping a
   record its resolver checks (`journalEntryCard` checks `journal_entry` read) declares
   `@authorizes(resource: "journal_entry")`, so that record is the root record and not a linked
   one. A test holds the declaration to the resolver's check.
4. **A projection of a kind.** A type that carries another kind's data under its own name
   (`ShipmentCustomer`, `InboundShipmentRef`) declares `@kind(resource: "customer")`, so the guard
   gates it as that kind. A field named for a kind that returns some other type fails CI until the
   type declares its kind or the field is listed, with its reason, in `notAKind`.

## Adding a field to a reference view

A view field is shown to everyone who can read any record that links to the kind, so adding one
widens access across the product. Add it only when linking screens need it to identify or use the
record, never money, credit, contact details, personal or HR data, notes or configuration secrets.
The change needs a security review on the pull request, which names every screen it is for. Then:

1. add the field to the kind's `ReferenceView`;
2. run `go run ./internal/api/graphql/gqlext/refgen` and `pnpm --filter @trenova/graphql codegen`;
3. drop any `@requiresRead` the operation no longer needs.

## What CI checks

Each failure says what to add.

| Test | Fails when |
|---|---|
| `TestReferenceViewsAreAtMostInternal` (permission) | a view field ranks above Internal |
| `TestEveryReferenceFieldExistsOnItsKind` (gqlext) | a view names a field no GraphQL type of its kind has |
| `TestEveryReferenceFieldExistsOnItsRESTKind` (responseguard) | a kind no GraphQL type shows declares a view field that no REST type carrying it has, or no response carries it at all |
| `TestEveryKindToKindFieldIsCovered` (gqlext) | a field named for a kind returns a type that declares no kind |
| `TestEveryLinkedFieldIsDeclared` (gqlext) | a client operation selects a field outside a linked record's view without `@requiresRead` for its kind |
| `TestEveryDeclaredLinkedFieldIsNullable` (gqlext) | a field selected under `@requiresRead` is non-null |
| `TestEveryAuthorizesDeclarationIsCheckedByItsResolver` (gqlext) | a root field declares `@authorizes` for a kind its resolver does not check |
| `TestEveryRootOfADatabaseScopedKindChecksItsRead` (gqlext) | a root field returns shipments, orders, invoices or workers, directly or through a connection, without its resolver checking the kind's read |
| `TestEveryRESTRootOfADatabaseScopedKindIsWrittenByItsOwnHandler` (responseguard) | a REST response returns a shipment, order, invoice or worker as its own record outside that kind's handler |
| `TestReferenceFragmentsAreGenerated` (gqlext) | the generated fragments or views are out of date |
| `TestEveryFieldAndEdgeIsClassifiedByItsReferenceView` (reportcatalog) | a report catalog field or edge is unclassified, or its class disagrees with the registry's view |
| `TestEveryProtectedFieldIsGatedInTheReportCatalog` (gqlext) | a schema `@protected` field is not marked `protected` in the report catalog, or the reverse |
| `TestEveryMarginFieldIsProtectedByTheMarginsGroup` (gqlext) | a GraphQL field named for a margin or profit is not `@protected(group: Margins)`, is not on a type reached only through such fields, and is not listed in `unprotectedMarginNames` with why |
| `TestEveryMarginEnumValueSaysWhereItIsRefused` (gqlext) | an enum value named for a margin is not listed in `marginEnumValues` with where it is refused |
| `TestEveryProtectedTypeSaysWhatStandsBehindIt` (gqlext) | a type with protected fields names neither the entity REST withholds it through nor why only GraphQL serves it |
| `TestEveryMarginFieldARESTResponseCarriesIsWithheld` (responseguard) | a JSON field a REST response carries, named for a margin or profit, has no `WithholdFields` in `restMargins` that clears it, and is not listed in `unprotectedRESTMarginNames` with why |
| `TestEveryRESTMarginIsClearedWithoutTheMarginsGroupAndKeptWithIt` (responseguard) | a type in `restMargins` leaves a margin to a caller without the group, or clears one for a caller with it |
| `TestEveryProtectedFieldIsWithheldFromTheAuditTrail` (responseguard) | a field a stored record's `WithholdFields` clears (worker, credential and every domain type in `restMargins`) stays in an audit entry of that record for a reader without the group, because no `permission.ProtectedPaths` entry covers it, or is withheld from a reader with every group |
| `TestEveryRelationAResponseReachesIsClassified` (responseguard) | a REST response reaches a struct through a bun relation that is neither a kind nor listed in `carried` |
| `TestEveryKindAgreesWithTheGraphQLGuard` (responseguard) | a Go type is gated as a different kind on REST than the GraphQL type bound to it |
| `TestHandlersWriteEveryResponseThroughTheGuard` (responseguard) | a handler writes a response with gin directly instead of `helpers.JSON` |
| `TestEveryHitMetadataKeyIsClassified` (globalsearchservice) | a search hit carries a metadata key `hitMetadata` does not classify |
| `TestEveryDetectorDeclaresEveryKindItsFindingsName` (detectors) | an insight detector declares no kind, an unregistered kind, or a finding names a kind it does not declare |
