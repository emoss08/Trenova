# Desk artifacts and resource cards

An artifact is what a Desk turn puts beside the conversation: a table, a report,
a document, a record. Most are drawn by kind (`table_view`, `entity_card`, …) in
`client/apps/web/src/routes/desk/_components/artifacts/`. A **resource card** is
drawn for one kind of record and what a person reads it for: the billing queue
item a biller reviews, the formula template a biller reads and prices, the
shipment a dispatcher follows from pickup to delivery, the customer an account
manager or a biller keeps, the tractor and trailer a fleet manager runs, the driver a
dispatcher puts on loads, the dock a planner sends trucks to, the carrier a carrier manager
qualifies and pays, the order a customer service rep follows to billing, the service failure
a customer service rep explains and settles, the rate agreement a pricing analyst holds
freight to, the invoice a biller sends and a collector follows to paid, the worker
settlement a payroll clerk approves and pays, the carrier settlement accounts payable
reconciles against the carrier's invoices and pays, and the detention charge a biller
approves for a stop that ran past its free time.

## Naming and linking artifacts

A reply names an artifact as a markdown link, `[Title](artifact:<id>)`, written as a noun inside
the sentence that says what it shows. The Desk draws it there as a link in the brand's blue, lit while its artifact is open. Who owns each part:

| Part | Owner |
|---|---|
| What an artifact is called | `assistantartifact/naming.go`. Every title passes `NormalizeTitle` when the recorder saves it: sentence case, no markdown, quotes or trailing punctuation, at most 120 characters. A record card is `RecordTitle` (the record kind's noun from the permission registry, then the record's label: "Formula template Per CWT (Hundredweight)"); a table is `RecordsTitle` ("Shipments", "DOT random pools"); a name built from an identifier is `PhraseTitle`, which keeps initialisms ("Send EDI status"). |
| The words of a link | The artifact's title, always. `FinishTurn` rewrites every link in the turn's replies with `agentruntime.NameArtifactLinks` before saving, so the transcript, search, exports and the next turn's context read the same name. A link to an id the thread does not hold is saved as its words. |
| The link | `DeskInlineArtifact` draws `artifact.title`, never the reply's words, so a reply still streaming already shows the right name. |
| Where a link may stand | Inside a sentence (`agentruntime/placement.go`); see "Reply passes" in [agent-runtime.md](agent-runtime.md). |
| Which artifacts a reply must name | `ShownArtifact.MustBeLinked`: a published document, and a table that is long, ranked, worked from or opens the live view. |

The client never moves, adds or renames a link: what the server saved is what is drawn.
Approval drafts and plans are made after the reply and open from their approval cards, not
from the reply's text.

Adding an artifact kind: build its title with the functions above, never with
`HumanizeSnakeCase` or `CapitalizeFirst` by hand, and decide whether `MustBeLinked` covers it.
The Desk Bench checks every turn that shows an artifact for stray links, links whose words are
not the title, and replies that needed a correction (`deskbench/checks.go`).

## Layout

```
artifacts/
  desk-workspace.tsx      the pane: which artifact is open, its versions, and the
                          switch that draws each kind
  artifact-payloads.ts    reading every kind's stored payload (shared with export,
                          previews and the server-contract tests)
  kinds/
    artifact-body-parts.tsx  what several kinds' bodies share
    <kind>/               one kind's body and everything only it uses: table,
                          report, document, extraction, edi-connection-draft,
                          record, rate, diff, plan, email, view, decision, each
                          with its __tests__/
  record-card/            the parts every card is built from: head, stepper, status,
                          sections, checks, timeline, a list's next page, the
                          "could not be read" head, bottom bar (record-card.tsx),
                          money, moments and table stepping (record-card-values.ts),
                          a record's shipments, paged (card-shipments.tsx, read by
                          useRecordShipments in record-shipments.ts), where a
                          record is on a small still OpenStreetMap map
                          (card-where.tsx, the map lazily in still-map.tsx from
                          map-tiles.ts, with a location's fence drawn round
                          the dot), what falls due (card-due.tsx: a check is
                          bad, warn, info, which is worth knowing and asks
                          for no look, or ok), a share
                          drawn as a meter (CardMeter: credit used, a rate
                          structure's groups), figures in a ruled strip
                          (CardStats: payment pace), a person's initials
                          (CardAvatar), what a
                          tractor's and a trailer's cards read alike
                          (equipment.ts), and a record's documents
                          (card-documents.tsx, a rail drawn by document-rail.tsx
                          from record-documents.ts)
  skeletons/              the parts every card's skeleton is built from
  cards/
    registry.tsx          the resource cards, in place and as artifacts
    <resource>/           everything that belongs to one card: the card, its
                          skeleton, its helpers, its store, its __tests__/
```

Anything that belongs to one artifact lives in that artifact's folder. A part two
kinds or two cards use moves to `kinds/artifact-body-parts.tsx`, `record-card/` or
`skeletons/`; a kind or a card never imports another's folder, except that a report
is drawn as a table and reads `kinds/table/`. Shared card styles are the `dk-rec-*` classes in
`routes/desk/_styles/desk-v2.css`; a card's own styles take its own prefix
(`dk-bi-*` for the billing item, `dk-ft-*` for the formula template, `dk-sh-*` for the
shipment, `dk-cu-*` for the customer, `dk-ra-*` for the rate agreement, `dk-tr-*` for the
tractor, `dk-tl-*` for the trailer, `dk-wk-*` for the worker, `dk-lo-*` for the location, `dk-ca-*` for the carrier, `dk-or-*` for the order, `dk-svf-*` for the service failure, `dk-edi-*` for the EDI message, `dk-eif-*` for the EDI inbound file, `dk-tdr-*` for the EDI transfer, `dk-inv-*` for the invoice, `dk-pay-*` for the worker settlement, `dk-cset-*` for the carrier settlement, `dk-dtn-*` for the detention charge; the parts the two settlement cards share are `dk-rec-*` in `record-card/`). An artifact kind drawn on the card frame takes one too:
`dk-ecd-*` for the EDI connection draft.
Before taking a prefix or a modifier class, search every stylesheet the Desk loads, the
Desk chat's `components/desk-chat/desk-chat.css` included: `dk-sf` is the chat's tool
failures, and the status pill's warn tone once shared `dk-hold` with the hold-to-confirm
button, which stretched every warn pill to the button's width.

## The EDI connection draft

`edi_connection_draft` is what `draft_edi_connection` drafted from a trading partner's
implementation guide (see [edi-spec-onboarding.md](edi-spec-onboarding.md)), drawn by
`kinds/edi-connection-draft/` on the record card's frame with its own skeleton
(`ConnectionDraftSkeleton`, which `ArtifactBodySkeleton` draws while its contents are read).
Every value carries a page tab, `SourceTab`, that opens the guide's quoted words in a hover
card; a held value lists each reading with its page, and picking one has the agent draft
again with it (`resolutionQuestion`), never a value chosen in the browser. The credentials
are listed for the person to enter and never shown with a value. Saving is the
`apply_edi_connection_draft` proposal, matched to the card by the document digest the person
reviewed; the card asks for it (`proposeQuestion`) only once the draft is ready. The recorder
drops the note meant for the model, cuts each sample's parse to its version and counts, and
trims mappings, samples and gaps to the payload limit with the full mapping count kept
(`assistantservice/edi_connection_draft.go`). A redraft of the same guide is a new version,
and the reply must always link it (`ShownArtifact.MustBeLinked`).

## Giving a card its own shape

`record-card/` is the frame every card shares: the head, the stepper, sections, the
bottom bar, the way back. What a card is *for* is drawn by parts of its own, in
`cards/<resource>/<resource>-parts.tsx`, so two cards that hold similar facts still read
as different records. Before reaching for `CardChecks`, `CardStats` or `CardFactGrid`,
ask what the record looks like to the person who reads it, and draw that:

| Card | Its own parts |
| --- | --- |
| Worker | hours of service as `RingGauge`s (`HosRings`, the tones from `hosClockTone`, as the worker panel draws them), credentials as a wallet of tiles, `tel:`/`mailto:` ways to reach them beside their record |
| Order | its lifecycle as one segmented track, its legs down a numbered route, its charges as a ledger with what is invoiced against what is still to invoice, its references as chips |
| Tractor | an ink dashboard of its last report (speed, heading, fuel ring, engine, odometer), its drivers as seats, a specification sheet |
| Trailer | a service calendar a season back and a year ahead with each date its papers hang on, a data plate |
| Location | amenity chips under the map, a dwell ruler with median, average and p90 pins and the share that ran past free time, an address card, terms as tickets |
| Carrier | a verdict banner for whether it can be tendered, insurance as coverage tiles, authority as pills over the CSA BASICs as columns against their thresholds, contacts with initials |
| Service failure | its clock along one track, each stretch as wide as it ran (before the cutoff, in grace, late) with its moments beneath, the 214's codes as tiles |
| EDI message | its transaction set as a stamp, its way as a relay edge to edge (generated, delivered, acknowledged; or received, read, tendered) with the envelope resting at the furthest stop and, where it stopped, a slip pointing at that stop with why, the attempt and the words sent back, what it says read in plain words over a stub of its fields, the ISA/GS/ST envelope nested with each trailer's check, the X12 as a tape with every finding pinned to its segment and element |
| EDI inbound file | the file as a sheet with its corner folded and the way it came in tabbed to its foot, its intake as chevrons (received, envelope read, sets handled) with each processing note the service wrote kept as written, a routing label from the ISA (sender and receiver with their qualifiers, the interchange number, the channel, the AS2 message or mailbox path, the checksum drawn as bars), what it carried as a folder per functional group with each transaction set opening in place as the message it was kept as and its own notes, SE01 miscount and findings pinned under it, a ruler of the whole interchange shaded by group, and the envelope's ISA/IEA and group counts checked; a file that is not X12 shows how it begins; each 204 names the tender it opened, which opens in place |
| EDI transfer | the tender as a ticket torn from a book (its 204 over the tear, the way it travels under it), titled by its bill of lading and named for the partner as the reading organization keeps them (the other side's partner row is hidden by row-level security), its decision as four gates in one meter (received or sent, mapped, decided, shipment) with a mapping hold in amber, a failure in red and a close struck through, why it closed kept as written, its stops down one rail with each window, load and whether the partner's location is mapped yet, its freight beside the rate run down to the total, and the partner's codes against our records with those still unmapped first; mappings are read live while an inbound tender waits on a decision, from the snapshot once it is decided |
| Invoice | a receipt slip of its bill type (INV, CR, DR) beside who it bills; the figure is what is still owed (amber once late), a credit memo's credit left, a draft's total or a voided total struck through; what is owed as one strip (paid, credited hatched, in dispute amber, still open) keyed underneath, and the terms as a ruler from the invoice date to the due date with today on it, past it in amber once late; a draft instead shows how many of the customer's days have already run while it waits to be posted, since its terms count from the invoice date; the charges as the bill reads them, one row per way a charge was billed (`Invoice.chargeSummary`, summed on the server so a consolidated invoice of hundreds of lines is a handful of rows), the freight rated by the formula template named on the line and opening that template's card, its name cut to fit with the whole of it on hover; the foot from the total down to the balance due over a double rule; payments and credits as remittance stubs (a reversed payment or a credit taken back struck through, a short pay tagged); the open dispute as an amber slip over the closed ones; and where it went as one slip each for email, the EDI 210 (opening its message) and the accounting system it is in, each line cut to one row and told in full on hover only when cut. A voided invoice says why and what became of its freight; a credit memo, what it corrects and the invoices it settled |
| Worker settlement | a pay stub marked PAY beside the worker; the figure is the net pay, struck through once voided and quiet at zero, with what carries forward under it; the flags the run raised to look at before approving; how earnings, reimbursements and deductions add up to the net pay as a waterfall, a shortfall carried to the next settlement hanging below zero; the statement summed by how each line was paid (`DriverSettlement.lineSummary`, one row per pay component, method, recurring item, advance or escrow account, so a week of loads is a handful of rows), the advance still owed and the escrow held beside their rows, pay added by hand marked; the worker's disputes, an open one in their words; and each load with the share the settlement pays for it (`loadShares`) in place of its charge |
| Carrier settlement | an AP stub beside the carrier and its MC number; the figure is the net payable with its terms; the carrier's invoices reconciled move by move (`CarrierSettlement.moveInvoices`, each move's standing invoice read through its carrier assignment, since a match names a settlement only once its variance is accepted) as a strip of cells (matched, off the buy rate, waiting to be accepted, not invoiced yet) keyed underneath, with only the moves that need someone listed, each invoice against the buy rate; the costs summed by kind over the adjustments (`costSummary`) down to the net payable; where the money goes (the remit-to address, the method, and the terms); and each load with its share |
| Detention charge | a DET stub beside the facility, the customer and the appointment; the figure is the charge to bill with its rounded time and rate, a running clock priced as of now (`DetentionOccurrenceDetail.currentlyBillable`, the engine's price for an open clock, since the stored amount is zero until the truck leaves), struck through once waived; the charge's way from the dock to the invoice, struck off when waived, disputed or not billable, with who approved it (`approvedBy`); the stop's clock as one ruler from the appointment or the arrival to the departure or now (before the clock hatched, free time, billable time, the notice flagged above it, every other crowded moment a row lower); the calculation trace as a receipt in the engine's own words, the worker's detention pay left to the split beside it; what the customer is billed against the worker's pay and what is kept; the notice the policy asks for, warned when overdue with when it must go; the collectability score with what each short factor needs, and the evidence chain; the load (`{ kind: "shipment", id }`) and the facility (`location`, read only under permission to read locations) and customer, each opened in place; and why a waived or disputed charge stands as it does (`waivedBy`) |

Tone is carried by the part's own fill, icon or text, never by an accent bar down or across
its edge. `CardStatus` takes `danger` with an `x` mark only for a state that is final and failed
(a tender whose shipment could not be built); a state someone can still act on is `warn`. A part only one card draws stays in that card's folder; it moves to `record-card/` when a
second card needs it. The shared parts take a shape where cards differ:
`CardShipments` takes a `layout` (`rows`, `table`, `board` with the pickup time first, or
`route` numbered down a line) and `CardShipmentsSkeleton` the same one; `CardHead` takes a
`lead` (a `CardAvatar` of a person's initials before their name, the EDI message's
stamp, the EDI inbound file's folded sheet, the EDI transfer's ticket; `CardHeadSkeleton`'s `lead` is `true` for the avatar's outline or the node to draw); `CardActivity` takes `showChanges` to read the first changed fields
of an update from what to what (`activityChanges` in `record-activity.ts`, which reads an
enum value as words and never shows a redacted one). Every new part has its outline in the
card's skeleton, drawn on the part's own classes.

## Lazy loading and skeletons

A card is drawn in two places, so each entry names two surfaces: `inPlace`, the
record opened from a table's row (`InPlaceRecord`), and `artifact`, the record as
an artifact of its own when the agent looked it up with a `get_` tool
(`RecordArtifact`, drawn by `DeskRecordBody` for an `entity_card` whose entity has
a card). Both come from the card's one module.

Each card in `cards/registry.tsx` is its own chunk (`lazyComponent` from
`src/lib/lazy-component.ts`), read the first time one is opened, or while the
browser is idle once a table of its kind is on screen (`usePreloadRecordCard`).
Every entry names a `Skeleton`: the card's outline, drawn on the card's own
layout classes with the Desk shimmer and each bar sized to the text it stands
for, so nothing moves when the card lands. The skeleton lives in the card's
folder but is imported by the registry, so it is in the workspace's chunk and
never waits on the code it stands in for. The card draws the same skeleton
while its record is read, so the wait for the code and the wait for the data
look the same.

## What a person may see on a card

A card opens for anyone who may read its own record. A section built from another
resource — a customer's receivables, shipments or rate agreements, any record's
documents — is left out for someone who may not read that resource, and its query is
never sent; the card is never refused for it. Declare a card's sections as a
module-level map from section to the resource it reads, in a module of its own so the
skeleton can read it too (`cards/customer/customer-sections.ts`), and ask
`useCardAccess` (`record-card/card-access.ts`) which of them to draw. Gate each
section's `useQuery` with `enabled` and its markup with the same flag, and leave the
section's outline out of the skeleton the same way, so nothing moves when the card
lands. `CardDocuments` and `CardDocumentsSkeleton` already do this for documents.

A section the person may read but that fails to load says so in its own place
(`describeApiError`), and the rest of the card stands. A card whose own record cannot be
read draws `CardUnreadable` with the way back.

## A record's shipments

A card that lists the shipments of its record (a customer's freight, the loads a tractor
pulled) reads them with `useRecordShipments({ scope, pageSize, enabled })` and draws them
with `CardShipments`. The scope says how the record owns them (`{ kind: "customer", id }`
filters on the customer; `{ kind: "shipment", id }` filters on the shipment's own id, for a
card about one stop of a load; `{ kind: "tractor", id }` asks `shipments(input: { tractorIds })`
for loads with a live move assigned to it). One operation, `DeskRecordShipments`, reads
every scope a page at a time with the count on the first page only, and the next page is
read on request. A new scope is a case in `scopeInput` (`lib/graphql/record-shipments.ts`)
and, where the server cannot yet filter by it, a `ShipmentsInput` field
(`tractorIds`, `trailerIds` and `workerIds` match loads with a live move assigned to the
unit or the driver; `stopLocationIds` matches loads with a live stop at the location;
`carrierIds` matches loads with a move tendered to the carrier and not canceled; an order's
legs filter on `orderId`; `invoiceIds` matches the shipments an invoice bills, as its shipment or
on a line, and needs permission to read invoices: the resolver reads the invoices through the
billing module and hands the shipment list their ids, so the operations module never reads a
billing table; `driverSettlementIds` and `carrierSettlementIds` match the shipments a
settlement's lines pay, read through the settlement module the same way, under permission to
read that kind of settlement). `CardShipments` takes `figureOf(shipmentId)` to show, in place of
a load's charge, what the listing record makes of it (the share a settlement pays for it).

A record's service failures are `CardFailures` over `useRecordFailures({ scope, pageSize,
enabled })` (`record-card/record-failures.ts`), one operation, `DeskRecordFailures`, for every
scope: `{ kind: "location", id }` reads the failures at stops made there and names each row's
load, `{ kind: "shipment", id }` reads a load's failures and names each row's stop.
`withoutFailure` leaves out the failure a card is about when it lists the rest of its load.

A record's way through review or approval, step by step down a rail, is `CardPath`
(`record-card/card-path.tsx`, with `CardPathSkeleton`): each step is done, where the record
stands, or still to come; a closing step is held or closed; and a note (a reviewer's
comment, why a failure was voided) sits on the step it was left at. The rate agreement's
approval and the service failure's review are both drawn with it.

A record's lifecycle in a fixed order of steps, drawn across the card as segments, is
`CardTrack` (`record-card/card-track.tsx`, with `CardTrackSkeleton`): the steps behind it
ticked, the one it stands at in ink, those ahead open, and the whole track struck off
(`off`) once the record is voided or canceled, with a note under it saying who moved it last
and what comes next. The order, both settlement cards and the detention card draw their
lifecycles with it.

The two settlement cards share their frame in `record-card/card-settlement.tsx` and
`record-card/settlement-values.ts`: the stub, the track's steps and note (`settlementTrack`,
`trackNote`), who signed each step (`SettlementSignoffs`), why a voided one was voided and how
far it had got (`SettlementVoidSlip`), the summed statement (`SettlementStatement`), and the
bar of the settlement's own steps (`SettlementBar`). Each kind draws what only it holds, the
worker's waterfall, flags and disputes, the carrier's invoices and remittance, in its own
folder.

## Actions on a card

A card's buttons are the record's own steps, taken by the person and never by the agent,
each shown only to someone holding the permission the server checks for it
(`useSettlementSteps` reads them once for a settlement card; the invoice card reads its own;
the detention card reads detention policy update once, the permission the server checks for
approving, sending the notice, recording a dispute and waiving).
`CardActionButton` (`record-card/card-action.tsx`) draws each: when something stands in the
way (an invoice that has not been posted cannot be sent by EDI) it stays where it is,
disabled, and says why on hover and to a screen reader, rather than vanishing. A step that
needs more from the person (a reason to send back or void, how a settlement was paid) opens
the dialog its page already uses, imported from the page, so the card and the page cannot ask
for different things; when that dialog lives in a larger component, it moves to a module of its
own first (the detention desk's waive and dispute dialogs are `occurrence-action-dialogs.tsx`), so
the card does not bring the page's sheet with it. After a step the card reads its record again through the module's own
invalidation (`invalidateDriverWorkspace`, `invalidateCarrierWorkspace`), whose prefixes
include the card's query keys (`useInvalidateDetention` invalidates every detention query,
the card's among them). What takes a form of its own (adding pay by hand, accepting an
invoice's variance) stays on the record's page, and the bar's hint says so.

Any other list a card reads a page at a time (a carrier's
settlements, a record's activity) goes through `usePagedRows` (`record-card/paged-rows.ts`),
which turns an infinite query into rows, a count and the next page, and is drawn by
`CardPagedList` (`record-card/card-paged-list.tsx`), which holds the rows while the first page
is read, says when there is nothing or it could not be read, and reads the next page on
request. A card's activity is `CardActivity` over `useRecordActivity`: what was done to the
record from its audit trail, newest first and without reads, each entry a sentence with who
did it and when, shown only to someone who may read the audit log.

`CardHead` sets the title level with the foot of the figure beside it. A title with no
subtitle is marked `dk-alone` and holds the top of the row instead, so a card whose record
has no description or trade name does not leave a gap under its kicker; there is no need
to invent a subtitle for it.

## Where a record is

`CardWhere` draws a place as a 112px still map of OpenStreetMap's tiles with a dot on it,
beside the place's name and how the record stands there. The map is a picture: nothing on it
moves or takes a click but the credit OpenStreetMap asks for. Its code is its own chunk
(`still-map.tsx`), mounted only once the square comes within reach of the screen, and it reads
at most four tiles, each lazily; a record with no coordinates shows a bare pin in the same
square. Tiles come from `tile.openstreetmap.org` (`OSM_TILE_URL` in `map-tiles.ts`), which the
Caddy CSP allows.

A place with a fence passes it as `outline` (a circle by its radius, or the corners drawn for
it) with a `zoom`; the map draws the fence in ink over the tiles, projected the same way the
tiles are laid (`projectOnMap`), and `zoomToFit` picks the closest zoom that keeps the whole
fence in view. A fence too small to see beside the dot is left off.

## Records a card lists

A card that lists records of another kind (a customer's shipments and rate agreements)
opens them the way a table opens its rows. `useListedRecords` (`cards/registry.tsx`) takes
the kind, the listed ids in the order shown and the way back's label ("Back to Acme
Foods"). For a kind with a card that the person may read (each registry entry names the
`resource` its card reads) it returns `open`, and `opened` is that record's card
drawn in the listing card's place, stepping through the list, with its way back returning
to the listing card; the card returns `opened` instead of itself while one is open. Draw
each listed record with `CardRecordRow` (`record-card/record-card.tsx`), which is a button
that calls `open` when the kind has a card, and a link to the record's page when it does
not. A kind that gains a card later starts opening in place with no change to the cards
that list it, and one the person may not read links to its page, which says for itself
what they may see. Every card's in-place surface takes `backLabel` (`InPlaceRecordProps`) and
falls back to its own ("Back to the table", "Back to the queue").

## A record's documents

A card whose record carries documents shows them with one line,
`<CardDocuments resourceType="shipment" resourceId={id} manageTo={recordPath(...)} />`,
at the foot of the card. It reads nothing until the section comes within reach
of the screen (`useInView`, `hooks/use-in-view.ts`): not the documents
(`resourceDocumentsQuery`, `lib/resource-documents.ts`, the key the documents tab
shares) and not the rail's code, which is its own chunk. The rail scrolls
sideways and draws twelve tiles at a time, the next batch as its end marker
comes into view, and each preview loads only as its tile comes near. Every tile
is the same size: three one-line rows and the review on the preview. It is for
looking; managing documents stays on the record's page. The shipment card shows the
shipment's documents, and so does the billing queue item's card: a billing item
has none of its own, and bills from the ones filed on its shipment.

## How a row opens in place

1. The record kind has an entry in `client/apps/web/src/config/record-links.ts`
   (regenerate the product guide). The server then keeps each row's id in a list
   tool's table (`recordEntity`, `assistantservice/artifact_display.go`), and the
   ↗ link on a row opens the record's page.
2. The kind has an entry in `cards/registry.tsx`. `DeskTableBody` then opens a
   clicked row as the card in place of the table, steps through the rows in the
   order the table shows them, and goes back to the table as it was left.
3. A table whose rows open in place has to be kept beside the conversation even
   when it is short, or the reply answers it inline and there is nothing to open.
   `opensRecords` in `assistantservice/artifacts.go` marks it (`ShownArtifact.OpensRecords`)
   for every kind with a card: the product guide generator reads the registry's keys
   into the catalog (`deskCard`), so the server needs no change. `shownNote` tells the
   model to point to it however few rows it has.

A list or search that found exactly one record of a kind with a card is drawn as
that card, not as a table of one row (`soleRecordCard` in
`assistantservice/artifacts.go`). Asked to show a template by name, the agent finds
it with `list_formula_templates`, and the person sees the template rather than a row
to open; a search that found one shipment is the shipment's card the same way. A page of one with more behind it (`hasMore`, or a larger `count`) stays a
table.

The billing queue's table is the richer case: its rows are also selected and
approved in bulk, so it keeps its own table body (`BillingTableBody`), and a row
opens in the conversation's billing item artifact when there is one. Its card is a
registry entry like any other.

## Adding a card

1. Add the record kind to `record-links.ts` if it has none. It must be a kind the
   server's permission registry knows; see [record-kinds.md](record-kinds.md).
2. Create `cards/<resource>/` with the card's two surfaces (taking
   `InPlaceRecordProps` and `RecordArtifactProps`), its skeleton, its pure helpers
   and their tests. Read the record live through `queries`; the table's row and the
   card's payload are only what the agent saw. Gate every section that reads another
   resource as "What a person may see on a card" describes, and test the card with
   only its own record readable (`seedPermissions` from `src/test/permissions.ts`).
   Never read a list that can grow without bound along with the record: read its
   count and its first page, and the next page on request (the rate agreement card
   reads its lanes from `GET /rate-agreements/:id/lanes/` with `useInfiniteQuery`,
   and the agreement itself with `includeRules=false`). When the card needs the shape of
   the whole list rather than its rows, ask the server for the aggregate: the rate
   agreement's rate structure reads `GET /rate-agreements/:id/lanes/summary/`, its live
   lanes grouped by formula template or rate table with each group's count and rate
   range, and draws each group as a `CardMeter` of its share, folding the groups past
   the first few into one.
3. Add the entry to `cards/registry.tsx`, and the card's mark to `desk-art-kinds.tsx`
   when it should not read as a plain record. Run `pnpm --filter @trenova/web
   guide:generate`, which marks the kind as having a card for the server.
4. Name the permission resource the card reads in its registry entry (`resource`), so
   other cards open it in place only for someone who may read it.
5. If the agent should be able to show one record, give it a `get_<entity>` tool.
