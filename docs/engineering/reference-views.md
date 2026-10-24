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

- **Reports and exports.** The report catalog is keel's, built from the models' `keel` tags;
  `reporting/reportaccess` is its `analytics.Policy` and maps each catalog entity to its kind
  (`kinds.go`). A field or edge is in the reference view when its resource's view names its API
  key (`APIKey` of the column, or the edge name); entities that are part of another record are
  marked `withinRecord` and never use a view. One package decides what a caller may use of the
  catalog: `reporting/reportaccess`. `Access.Field` and `Access.Edge` are the rule: on a kind the caller
  reads, a field is held to their ceiling; on a kind they cannot read, only a `reference` field
  is usable, decided by `refview` through `permissionports.ReferenceViews` (ceiling included);
  an edge is a field of its source, and a pick edge needs read on each kind it picks rows from;
  a protected field (`protected: Personal|Margins`) needs its group either way. Every report
  surface asks it, from one `GetResourcePermissionsBatch` read of the caller's permissions per
  request:
  - the compiler (`compiler/authorize.go`), which also requires read, and export for a run, on
    the root and on any entity whose own rows the query reads or whose `kind_read` fields it
    uses; its refusals (`Resolver.Reason`) name the missing kind read, tier or group;
  - the `reportCatalog` query (`reporting.Service.Catalog`), which returns every kind the caller
    reads plus every kind they can reach only through a join, with `readable`, `exportable`
    and, per field and edge, `access` (`Available`, `NeedsKindRead`, `WithheldByCeiling`,
    `NeedsFieldGroup`), `inReferenceView`, `protection { group held }` and `needsReadOf`;
  - drill-through, and the agent's `report_list_datasets` and `report_get_dataset`, which
    describe a reference-only dataset, mark edges `referenceOnly` and say what a closed field
    or edge `needs`.

  The builder starts a report only from a readable kind, lists only the reference view of a kind
  reached through a join ("Reference only — the rest needs {Kind} read"), shows protected fields
  with a lock and their group, disables what is unavailable with its reason, and gives a saved
  report that names unavailable fields an alert that lists them and removes them
  (`lib/report-catalog-access.ts`, `unavailable-fields-alert.tsx`). Table exports and dashboard
  exports compile through the same check.
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
   view's own fields; the bookkeeping fields are allowed but not selected. A type of a kind that
   carries neither an `id` nor a field of the view (a detail or a summary of the kind, such as
   `EdiMessageDetail`, `EdiAckHealth`, `FuelIndexLatestPrice` or `FactoringSummary`) is never
   linked to, so it gets no fragment.
2. **When the screen needs more, say so.** Selecting a linked record's field outside its view
   needs `@requiresRead(kind: "customer")` on the operation (repeat it for each kind). The field
   must be nullable, and the screen draws the field's withheld state where it is null for a reader
   without that kind's read. Ask first whether the job needs the full record: if it does, the
   bundle for that job should grant the kind's read.
3. **A card or overview that is about one record.** A root field that returns a card wrapping a
   record its resolver checks (`journalEntryCard` checks `journal_entry` read) declares
   `@authorizes(resource: "journal_entry")`, so that record is the root record and not a linked
   one. A test holds the declaration to the resolver's check. A card whose own parts carry the
   record's data (a journal's lines and their amounts) also declares `@kind` of that record
   (`JournalEntryCard @kind(resource: "journal_entry")`), so those parts are gated by the
   record's sensitivities; the records it carries of a kind the root declares with `@authorizes`
   are still the root's own, and the card's own fields are declared in that kind's
   sensitivities. A card of a kind has no reference fragment.
4. **A projection of a kind.** A type that carries another kind's data under its own name
   (`ShipmentCustomer`, `InboundShipmentRef`) declares `@kind(resource: "customer")`, so the guard
   gates it as that kind. A field named for a kind that returns some other type fails CI until the
   type declares its kind or the field is listed, with its reason, in `notAKind`.

## The page access manifest

`services/tms/pkg/pageaccess/manifest_gen.json` records, for every screen (route) of the web app:

- `requires`: the permissions its route's loaders demand;
- `operations`: the GraphQL operations its code runs, and `endpoints`: the REST endpoints it calls,
  each with the permissions its route checks before the handler (empty when the handler checks
  for itself);
- `reads`: the kinds its queries return from their root resolvers, which need the kind's read;
- `references`: the kinds it shows only through their reference views;
- `declared`: the kinds whose further fields it asks for with `@requiresRead`;

and, per operation, every field it selects on a linked record of each kind (a field written
`Move.status` is of a record that is part of the kind, which has no view of its own). The server
embeds it, so a build always describes the app it was built with.

It is derived, never written by hand, in two steps:

1. `pnpm --filter @trenova/web access:generate` (`client/apps/web/scripts/page-access/`) reads the
   route tree the product guide reads (`scripts/product-guide/sources.mjs`) and, for each route,
   every module reachable from its page module through static and dynamic imports, short of the
   app shell (what `main.tsx` loads statically). A module that imports documents is held to the
   documents of the exports its importers name, so a page that imports one wrapper of
   `lib/graphql/shipment.ts` runs that wrapper's operation, not all of them; a query key factory is
   one module, so a page that imports it runs every operation the factory can. REST calls are
   `api.<method>(path)` literals, and an `apiService.<service>.<method>` is resolved to that one
   service method. It writes `pkg/pageaccess/screens_gen.json`.
2. `go run ./internal/api/graphql/gqlext/refgen` reads that file, walks each operation with
   `gqlext.Operations` (the same walk the field guard's checks are built on, so a root, a linked
   record and a part of one are reached exactly as at run time) and joins each REST call to its
   route through `routelint` and `permissionlint`. It writes `manifest_gen.json`.

A mutation's payload is a root record its resolver checked with its own operation, so a mutation
adds no kind to `reads`; its linked fields are listed like any other.

## Page access panel and viewing a screen as a role

The keybinds config binds `Ctrl+Alt+P` to a **Page access** panel for administrators, who hold role
read, and on every screen of a development build. It has no menu entry: it is a tool for
administrators and developers, reached only by its keybind. It opens a lazy
sheet (`components/page-access/`) that is not in the first visit's download. It shows what the screen
needs, which kinds the reader holds read on, and which linked fields come back as reference view
only, from:

- `pageAccess(location:)` for the reader's own access. It checks only authentication (listed in
  `authOnlyAllowlist`): it describes the caller's own grants against the static manifest. Every
  permission the screen needs is asked of the engine in one `CheckBatch`.
- `pageAccessPreview(input:)` for a role, saved (`roleId`) or as the role editor holds it
  (`draft`), with the roles it inherits. It needs role read, like the role editor. The grants are
  united by `permission.GrantsOf` with their implied operations and the widest ceiling, wrapped in
  `refview.RoleAccess`, and every field goes through `refview` (`Decide`, `DecidePart`) with the
  role's ceiling (`fieldsensitivity.RoleCeiling`), so the preview cannot disagree with the guard.
  It is worked out before organization access policies (ABAC), and an inherited role from another
  organization is refused.

Neither reads a record; both are `internal/modules/identity/app/pageaccessservice`. The role editor
draws the same report in **Preview a screen**, with the screens from `pageAccessScreens`.

## Role editor sentences

Each resource row of the role editor's Advanced tab says what granting and withholding its read
means, from `roleResourceGuides(input:)` (role read; a category's resources at a time, at most 64).
The sentences are built by `pageaccessservice.Guide` from the registry, not written per kind:

- with read, every field of the kind's records, on their own screens and wherever another record
  shows one;
- without it, the view's fields as words wherever a record the role can read links to one, naming
  the linking kinds the most screens read first, and that the rest needs the kind's read;
- the view's fields that rank above the role's ceiling and stay withheld;
- **Screens this affects**: every screen of the manifest that requires, reads, declares or
  references the kind.

The words for each view and the kinds that link to each are generated by refgen into
`pageaccessservice/viewwords_gen.go` (`pagemanifest.RenderWords`): bookkeeping fields left out, an
address's parts as one address, a link's id beside the link as the link, and a carrier being the
family of the type holding the field. The Go extractor reads that file's `Label:` values, so each
word is a catalog entry, and lists are joined in the request's language by `i18n.List`.

## Adding a field to a reference view

A view field is shown to everyone who can read any record that links to the kind, so adding one
widens access across the product. Add it only when linking screens need it to identify or use the
record, never money, credit, contact details, personal or HR data, notes or configuration secrets.
The change needs a security review on the pull request, which names every screen it is for. Then:

1. add the field to the kind's `ReferenceView`;
2. run `go run ./internal/api/graphql/gqlext/refgen` and `pnpm --filter @trenova/graphql codegen`;
3. drop any `@requiresRead` the operation no longer needs;
4. run `task i18n` and translate the field's new words, if the role editor names it in a way no
   view did before.

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
| `access:check` (`test-client.yml`) | a route, an import or a REST call moved what a screen runs and `screens_gen.json` was not regenerated |
| `TestPageAccessManifestIsGenerated` (pagemanifest) | `manifest_gen.json` differs from what refgen writes from the screens, the operations, the schema, the registry and the routes |
| `TestViewWordsAreGenerated` (pagemanifest) | a view or a link between kinds changed and `viewwords_gen.go` was not regenerated |
| `TestEveryReportableEntityHasAKind` (reportengine) | a reportable model has no row in `reportaccess/kinds.go`, or names an unregistered resource or a missing ownership column |
| `TestReportSurfacesDecideAccessOnlyThroughReportaccess` (reportaccess) | the reporting service, the report jobs, the engine, the report resolvers or the agent's report tools call an access primitive (`CanAccess`, `Decide`, `Person`, `InReferenceView`, `ProtectedGroupAt`, `Visible`) instead of asking reportaccess |
| `TestTheEngineAndTheCatalogShareTheDecision` (reportaccess) | the engine's policy or the builder's catalog stops deciding through `FieldUse` and `EdgeUse` |
| `TestReportsProtectWhatTheSchemaProtects` (gqlext) | a schema `@protected` field of a reportable kind is not gated in reports in its group, or reports gate a field the schema does not protect |
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
