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
freight to.

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
                          report, document, extraction, record, rate, diff, plan,
                          email, view, decision, each with its __tests__/
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
tractor, `dk-tl-*` for the trailer, `dk-wk-*` for the worker, `dk-lo-*` for the location, `dk-ca-*` for the carrier, `dk-or-*` for the order, `dk-svf-*` for the service failure, `dk-edi-*` for the EDI message, `dk-eif-*` for the EDI inbound file).
Before taking a prefix or a modifier class, search every stylesheet the Desk loads, the
Desk chat's `components/desk-chat/desk-chat.css` included: `dk-sf` is the chat's tool
failures, and the status pill's warn tone once shared `dk-hold` with the hold-to-confirm
button, which stretched every warn pill to the button's width.

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
| EDI inbound file | the file as a sheet with its corner folded and the way it came in tabbed to its foot, its intake as chevrons (received, envelope read, sets handled) with each processing note the service wrote kept as written, a routing label from the ISA (sender and receiver with their qualifiers, the interchange number, the channel, the AS2 message or mailbox path, the checksum drawn as bars), what it carried as a folder per functional group with each transaction set opening in place as the message it was kept as and its own notes, SE01 miscount and findings pinned under it, a ruler of the whole interchange shaded by group, and the envelope's ISA/IEA and group counts checked; a file that is not X12 shows how it begins |

Tone is carried by the part's own fill, icon or text, never by an accent bar down or across
its edge. A part only one card draws stays in that card's folder; it moves to `record-card/` when a
second card needs it. The shared parts take a shape where cards differ:
`CardShipments` takes a `layout` (`rows`, `table`, `board` with the pickup time first, or
`route` numbered down a line) and `CardShipmentsSkeleton` the same one; `CardHead` takes a
`lead` (a `CardAvatar` of a person's initials before their name, the EDI message's
stamp, the EDI inbound file's folded sheet; `CardHeadSkeleton`'s `lead` is `true` for the avatar's outline or the node to draw); `CardActivity` takes `showChanges` to read the first changed fields
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
filters on the customer; `{ kind: "tractor", id }` asks `shipments(input: { tractorIds })`
for loads with a live move assigned to it). One operation, `DeskRecordShipments`, reads
every scope a page at a time with the count on the first page only, and the next page is
read on request. A new scope is a case in `scopeInput` (`lib/graphql/record-shipments.ts`)
and, where the server cannot yet filter by it, a `ShipmentsInput` field
(`tractorIds`, `trailerIds` and `workerIds` match loads with a live move assigned to the
unit or the driver; `stopLocationIds` matches loads with a live stop at the location;
`carrierIds` matches loads with a move tendered to the carrier and not canceled; an order's
legs filter on `orderId`).

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
