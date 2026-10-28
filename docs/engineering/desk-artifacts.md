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
| Where a link may stand | Inside a sentence (`agentruntime/placement.go`); see "Reply passes" in [agent-runtime.md](agent-runtime.md). A link standing apart from any sentence (on its own line, after a list, after a sentence that has ended) is cut and joined, in parentheses, to the reply's own prose line nearest above where it stood, without asking the model again; only a link in a table, a heading or a fenced block, an id nobody holds, or a reply with no prose to join is asked for again. |
| Which artifacts a reply must name | `ShownArtifact.MustBeLinked`: a published document, an EDI connection draft, a written record, and a table that is long, ranked, worked from, pages (`Paged`: its list has more rows than it holds, or the agent read more than one page into it) or opens the live view. Only the newest version is held to it: a call that shows the next version of an artifact the turn already showed (a second draft of the same guide, whichever partner the agent named, a table read again, a revised document; `ShownArtifact.Lineage`, from `assistantartifact.LineageKeyFor`) takes the earlier one's place in `Turn.keepShown`, so the reply is never asked for the version it replaced. The note the model reads with the result (`shownNote`) says so for each of them: one that must be named is never told that pointing to it is optional. A reply that still leaves one out has it linked from its own first sentence, in parentheses, with no words of the server's own (the server does not know the reply's language); it is not asked again for that alone. |

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
                          analysis, portal, record, rate, diff, plan, email,
                          view, decision, each with its __tests__/
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
`skeletons/`; a kind or a card never imports another's folder. Shared card styles are the `dk-rec-*` classes in
`routes/desk/_styles/desk-v2.css`; a card's own styles take its own prefix
(`dk-pln-*` for the plan, `dk-bi-*` for the billing item, `dk-ft-*` for the formula template, `dk-sh-*` for the
shipment, `dk-cu-*` for the customer, `dk-ra-*` for the rate agreement, `dk-tr-*` for the
tractor, `dk-tl-*` for the trailer, `dk-wk-*` for the worker, `dk-lo-*` for the location, `dk-ca-*` for the carrier, `dk-or-*` for the order, `dk-svf-*` for the service failure, `dk-edi-*` for the EDI message, `dk-eif-*` for the EDI inbound file, `dk-tdr-*` for the EDI transfer, `dk-inv-*` for the invoice, `dk-pay-*` for the worker settlement, `dk-cset-*` for the carrier settlement, `dk-dtn-*` for the detention charge, `dk-edp-*` for the EDI partner, `dk-jen-*` for the journal entry, `dk-mj-*` for the manual journal, `dk-fpu-*` for the fuel purchase, `dk-fcd-*` for the fuel card, `dk-tv-*` for a table that pages, `dk-rr-*` for a report run or preview; the parts the two settlement cards share are `dk-rec-*` in `record-card/`, and so are the parts the two journal cards share: `dk-rec-jrnl`, `dk-rec-eqn`, `dk-rec-jref`, `dk-rec-seal` and `dk-rec-period`; the two fuel cards share `dk-fu-*`). An artifact kind drawn on the card frame takes one too:
`dk-ecd-*` for the EDI connection draft.
Before taking a prefix or a modifier class, search every stylesheet the Desk loads, the
Desk chat's `components/desk-chat/desk-chat.css` included: `dk-sf` is the chat's tool
failures, and the status pill's warn tone once shared `dk-hold` with the hold-to-confirm
button, which stretched every warn pill to the button's width.

## The analysis

`analysis` is what an `analysis_run` showed (see "Kept analyses and the Desk card" in
[agent-code-mode.md](agent-code-mode.md)): its title, then the headline figures, each chart and
table in the order the script showed them, the error it raised, and what it printed and the
code, folded. `kinds/analysis/` only reads the payload (`analysisFrom`) and reads the run for a
table longer than the agent was shown; the drawing is `AnalysisResult` in
`components/assistant/`, which AI control's analysis history draws too. A chart is a spec drawn
with Recharts (`AnalysisChartView`, loaded with the chart). An analysis is made on purpose, so it
is not a lookup: it is kept whether or not the reply names it, and the reply must name it.

## The portal task

`portal_task` is a portal task an agent ran in a cloud browser (see
[agent-portal-automation.md](agent-portal-automation.md)): a `web_read_portal` call,
titled by its question, or an approved `web_use_portal`, titled "Portal action on"
its site (`assistantservice/portal_task_artifact.go`). The payload is only what the
agent was told; `kinds/portal/` reads the run itself (`agentExtension.portalRun`)
and reads it again every few seconds while its browser is open, so the card is
the task as it stands: running, with a way to watch the live browser; waiting on
a sign-in code, with the code box and the way to sign in by hand; or ended, with
what the portal said marked as read from outside the organization, why it did not
finish, the pages it reached and its recording. The drawing is `PortalRunResult`
in `components/assistant/`, which AI control's portal history draws too. A run
the viewer cannot read (another person's, in a shared thread) shows the payload's
answer and pages with nothing to act on. Each run is its own artifact, with no
versions: two reads of one site are two tasks. It is made on purpose, so it is
not a lookup, and the reply must name it, which is how the person waiting on a
code finds where to enter it.

## The EDI connection draft

`edi_connection_draft` is what `edi_draft_connection` drafted from a trading partner's
implementation guide (see [edi-spec-onboarding.md](edi-spec-onboarding.md)), drawn by
`kinds/edi-connection-draft/` on the record card's frame with its own skeleton
(`ConnectionDraftSkeleton`, which `ArtifactBodySkeleton` draws while its contents are read).
Every value carries a page tab, `SourceTab`, that opens the guide's quoted words in a hover
card; a held value lists each reading with its page, and picking one has the agent draft
again with it (`resolutionQuestion`), never a value chosen in the browser. The credentials
are listed for the person to enter and never shown with a value. Saving is the
`edi_apply_connection_draft` proposal, matched to the card by the document digest the person
reviewed; the card asks for it (`proposeQuestion`) only once the draft is ready. The recorder
drops the note meant for the model, cuts each sample's parse to its version and counts, and
trims mappings, samples and gaps to the payload limit with the full mapping count kept
(`assistantservice/edi_connection_draft.go`). A redraft of the same guide is a new version,
and the reply must always link it (`ShownArtifact.MustBeLinked`).

## The plan

`plan` is several of a turn's writes decided as one (`agentplanservice`): the
`assistantservice` recorder files the artifact with the plan's id, title, summary and its steps'
proposal ids, and `kinds/plan/` reads everything else live, so the card is the plan as it
stands. It reads the thread's plans (`GET /threads/:id/plans/`, `AssistantPlan`), its
proposals (one per step) and the plan's preview (`myPlanPreview`: every step's records, each
field from → to, the money it moves, the message it sends, with a decided step's preview as
recorded at its decision). `plan-card-model.ts` holds every reading as pure functions:

- **Phase** (`planPhase`): awaiting, held (a shadow switch), undoable (approved and inside its
  undo window, `commitsAt` still ahead), running, failed, done, declined or expired.
- **The track**: a segment a step under the plan's name (`CardHead`'s `under` slot), and the
  steps down a line, each node coloured by its own proposal's status (`stepTone`): waiting, up
  next, running, ran, did not go through, not run.
- **How it runs**: `AssistantPlan.runMode` is `agent.PlanRunModeOf(steps)`, the same function
  the executor calls, so the card's "stops at the first failure" or "every step runs on its
  own" can never disagree with what runs.
- **Folding**: before a decision, and while an approval can still be undone, every change is
  open. Once the plan has run, each step folds to one line ("Changed 3 fields on S-240201 ·
  +$120.00", `foldedLine`) behind a disclosure, except a step that failed or carries a warning.
- **Records it touches** (`recordsTouched`): each record once, with the steps that change it;
  one the reader may not see is counted, never named.
- **Approval** (`planLog`): who proposed it, who decided it (`decidedByName`, resolved on the
  server, never a single-user read per row), the note they wrote (`decisionNote`, plain text),
  the digest it was approved against (`previewDigest`), the commit after the undo window, and
  each step.

The person who owns the conversation, while its agent can still be asked
(`thread.canContinue`), decides the plan on the card through the approval box's own
mutation (`decideMyPlan`), approving with the digest of the preview the card shows. A digest
that moved is refused as a conflict: the card reads the preview again and says so
(`ChangedNotice`), with no toast. Decline asks for an optional note (`ReasonDialog` with
`optional`, at most 2000 characters) that goes to the agent with the decision. Inside the undo
window the decider sees "Can be undone for 0:08" counting on `useNowSeconds` and Undo
(`undoMyDecision`). A plan that failed, or expired with steps undone, offers "Ask the agent
to finish it", which posts the person's own message into the thread (`askAgent`) naming the
plan, the step it stopped at and the server's failure text; anything the agent does in reply
is a new proposal decided as usual, and the button is gone once a later plan in the
conversation takes over (`supersededBy`). No success toast follows any of these: the card
redraws with the decision.

The mock API serves the plan as `athr_desk_plan` with the `deskPlan` scenario field (awaiting,
held, approving, running, failed, done, declined, expired, superseded), and
`e2e/mock-api/desk-plan-screenshots.mjs` shoots every stage. While the plans, steps or preview
are read the card draws `PlanSkeleton`, which `ArtifactBodySkeleton` draws too.

## The extraction card

`extraction` is what document intelligence read off a file, drawn as the record it is about to
become (`kinds/extraction/`, on the shared `record-card/` frame). The head names the load by its
lane and leads with the rate as read; the readings a person must settle come first, each beside
the bit of page it came from; then the load as the reading has it, route first; then the
document itself, its pages with a quiet box over every value found; and the bar says where the
shipment stands.

**The head.** `extraction-summary.ts` works it out from the payload. The stub is the paper's
abbreviation (`documentMark`: RC, BOL, POD, INV, LMP, SCL, DR, else DOC). The title is the lane,
from the first stop's `place` to the last's (`laneOf`), and falls back to the kind of document
when fewer than two stops name a place, as on a card saved before stops carried one. The figure
is the rate exactly as the document gave it (`headFigure`): the fuel surcharge is named beside it
and never added, because a document's rate line is as often its total as its linehaul; a rate the
reading looked for and did not find says "Rate not read", and a document with no rate (a bill of
lading) has no figure. The pill says how many readings need a person until there is a shipment
to speak of, then where it stands (`headStanding`); beside it are what was not read and what was
read clearly, and the classifier's confidence in the kind only when it is under 90%.

**The page.** Each page is the server's drawing of it, `GET /documents/:id/pages/:page/image/`
(a PDF page drawn by the imaging worker to fit 1400 × 1800, or the image itself as page 1, as
WebP; the first `imaging.prerenderPages` pages are rendered and stored after the upload, and any
other page is rendered once on its first view and stored, so a read serves a stored render; see
[imaging-worker.md](imaging-worker.md#page-renders-in-the-tms). It is read on
`document:read`, recorded as a view on the `page_image` channel and cached against an ETag;
every API response says `Vary: Origin` (`middleware.NewVaryOriginMiddleware`), so a copy a
browser cached from a request without an origin is never handed to a cross-origin fetch that
needs the CORS headers). The card fetches it with `api.download` (`lib/queries/document-page.ts`), so it carries the session's
credentials and the app's error handling, and keeps it as a blob in the query cache; the next
page and the first page with something to check are read ahead. No PDF library renders pages in
the browser: `PdfFrame` still shows every PDF, and the card's "Open file" opens the whole file in
it (or the image itself) for search and print. While page 1 is drawn the upload's stored preview
stands in for it, blurred, with the boxes already on it; any other page stands as its skeleton in
the page's own shape, from the payload's `pages` sizes.

**The boxes.** A box is a fraction of the page. `assistantservice/extraction.go` finds each value
in the page's layout lines (`documentcontent.LayoutLine`): first on the line the reading quoted
as its evidence (read from the stored draft, not from the tool result the model sees), then on a
line that carries the field's label before the value (so "Bill to" is not boxed on a letterhead
that happens to hold the same name), then on any line; the page the reading names is searched
first, and a value split by other words on the page ("10/22/2026 Appt 09:00") is boxed on its
leading words. The value's span is measured on the line's own characters, and where a line records where
each word sits (`LayoutLine.Words`) the box runs from the first word's edge to the last; Docling's
lines carry no word positions, so the box takes each character's share of the line. On the
client the sheet is the image at its own size and the boxes are percentages of a layer exactly
over it (`boxStyle`), so they stay on their text at any width, zoom step or pixel ratio and after
a resize. Fit draws the page at the viewer's width but never under 600px (`sheetWidth`), so on a
phone it scrolls sideways rather than shrinking past reading.

**What was read.** `extraction-payload.ts` reads the payload, old and new. Each field carries
`reason` (`conflict`, `review` or `low`) and, for a conflict, every `readings`; each stop is its
`name`, `address` and `window` parts (`stop`, `role`, `part`, `appointment`, and `place`, the
stop's "City, ST" from `stopPlace` in `extraction.go`); `missingFields` names what the document
lacked with its key. A card saved before these were kept still reads: no sizes means a letter
frame, a stop without parts is one row, a stop without a place leaves the lane out, a reason is
inferred from the confidence, and a missing field is placed by its label.
`extraction-groups.ts` owns the order (parties, the stops in sequence, references, equipment and
freight, charges, then Other, which takes any key it does not know) and the labels; a pickup or
delivery window read apart from its stop joins it, and is left out when it is settled and only
repeats the stop's own window. A value read at 90% or more is settled and faintly tinted, one
between 80% and 90% is underlined dotted, and one that needs a look is underlined amber. The
tally counts a stop under review once.

**Check, then the load.** `ExtractionCheck` lists what a person settles before the reading
becomes a shipment: each doubtful field (a stop once, with its doubtful parts) with why, a
conflict's readings side by side with the one used marked, and a snippet of the page beside it
(`ExtractionSnippet`): the same cached page image, placed so only the region around the value's
boxes shows through (`snippetRegion`, `regionImageStyle`), with the value outlined. Pressing the
snippet chooses the value and scrolls the document into view. Then each field that was not found.
Nothing is red: neither is a mistake yet. `ExtractionLoad` lays the load out as the shipment would
be: the route as stops with their role, appointment and parts, then a panel per group. Every value
is a toggle button whose name says its value, its state, its confidence and its page
(`rowAnnouncement`); choosing one lights its box and turns to its page, pointing at one lights its
box, and choosing a box chooses its value. The document's thumbnails (up to eight pages, else a
pager) count the fields to check on each page.

**The layout.** The card is a container (`dkex`): at 900px or wider the document stands on the
left and stays in view (`position: sticky`) while the reading scrolls beside it; narrower, the
reading comes first and the document follows. The artifact's own page widens to 1280px for this
card, and its scrolling frames give way (`.dk-apage` visible, `.dk-apage-body` clip) so the page
scrolls with the window and sticky has a scroller to stick in.

**The shipment.** `shipment-stage.ts` works out where it stands from the server and the
conversation, never from the component: the newest `shipment_create` proposal whose
`sourceDocumentId` is the file (waiting, creating, created and linked with `recordPath`, failed
with the server's reason, declined, expired, simulated), `attachedShipmentId`, the conversation's
running turn (`assistant.activeTurns`), and what the card asked (`documentAsks` in the Desk
store, kept for the session only). A create asked after the newest proposal is drafting while the
conversation is on it and "not drafted" once the reply ends without one. The bar's check button
names every field it will ask about, with why, in its title, and Create asks the agent to draft
the shipment, which waits for approval like every other write; Review the draft opens that
approval's `decision_request` artifact, and a created shipment is linked from the head and the
bar. While
the conversation replies to something else, the card asks nothing.

The body is its own chunk (`lazyComponent` in `desk-workspace.tsx`), and `ExtractionSkeleton`,
drawn on the card's own classes, stands in for both the code and the payload.

The agent
opens it by reading a draft with `intake_get_draft`, and a file the person attaches opens it
without anyone asking (`assistantservice/attachment_cards.go`; see "Files a person attaches" in
[agent-runtime.md](agent-runtime.md)). Both are one lineage per document, so a later reading
(the model's after the rule-based one) is a new version of the same card. The summary keeps
the payload's `documentId` (`SummaryPayloadKeys`), and the file's chip under the message that
carried it opens the newest card read off it: the thread hands `extractionCardsByDocument`'s
map to `ArtifactOpenerProvider` as `attachmentCards`, and `DeskMessageAttachments` reads it
with `useAttachmentCard`. A chip with no card yet, or on a surface with no pane, is the file's
name and nothing to press. Never publish a document to show what was read off a file.

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
| EDI partner | a port of its ways in (AS2, SFTP, VAN, or Trenova to Trenova) before its code, kind and standing; the figure is the share of what we sent in the last seven days that was delivered; what the two sides trade as two lanes between our end and theirs, each transaction set a stamp with what it waits on and the week's count, its failures and dead letters marked (a Trenova-to-Trenova partner's lanes count nothing, since no envelope goes over a line); what needs a look first, worst first (a pickup failing, a message dead-lettered, acknowledgments overdue past their window, files set aside), each opening its message or file in place; the fortnight's traffic as a pair of columns a day in the reader's own days, with the acknowledgments' ruler and how long they take against what was promised; each line we reach them over as a patch between our end and theirs, its credentials named as on file or missing and never shown, its certificates as tiles read for expiry, its last pickup and why the last one failed with secrets redacted (`EDIPartnerTransport`); a Trenova-to-Trenova partner shows its connection instead; how the envelope is addressed both ways with the sets addressed differently counted; what it still lacks before it can trade, first while it lacks something; their codes we have matched by kind of record; the latest messages a page at a time; the customer and the carriers that send through it (read only under permission to read carriers); and who to call, with their local time. It is read in one query, `ediPartnerCard`, so the card never reads a partner's credentials |
| Detention charge | a DET stub beside the facility, the customer and the appointment; the figure is the charge to bill with its rounded time and rate, a running clock priced as of now (`DetentionOccurrenceDetail.currentlyBillable`, the engine's price for an open clock, since the stored amount is zero until the truck leaves), struck through once waived; the charge's way from the dock to the invoice, struck off when waived, disputed or not billable, with who approved it (`approvedBy`); the stop's clock as one ruler from the appointment or the arrival to the departure or now (before the clock hatched, free time, billable time, the notice flagged above it, every other crowded moment a row lower); the calculation trace as a receipt in the engine's own words, the worker's detention pay left to the split beside it; what the customer is billed against the worker's pay and what is kept; the notice the policy asks for, warned when overdue with when it must go; the collectability score with what each short factor needs, and the evidence chain; the load (`{ kind: "shipment", id }`) and the facility (`location`, read only under permission to read locations) and customer, each opened in place; and why a waived or disputed charge stands as it does (`waivedBy`) |
| Journal entry | a JE stub beside what it booked, its type and whether Trenova or a person booked it; the figure is what each side comes to, struck through once reversed or void; when it is reversed, the entry and its reversal bracketed together to what they leave in the books (nothing), with the request's reason code and words, and a reversal asked for and not yet posted, or refused, as a slip naming who asked and what became of it; its way to the ledger (recorded, approved where the organization reviews entries, posted) with its batch; the lines written into the book with debits first and credits indented under them; what it does to each term of the accounting equation; and the record that made it (an invoice, a settlement, a manual journal, the entry it reverses), opened in place where it has a card |
| Manual journal | an MJ stub beside what it books and who asked for it; the figure is what each side comes to, or how far it is off while the lines do not balance, struck through once rejected or cancelled; the requester's reason in their own words; why it was rejected or cancelled; its way to the ledger (draft, submitted, approved, posted) with who signed each step; the lines and the equation as the entry draws them; where it will post as a calendar leaf, or the shut period it asked for beside the first open day it moves to under the organization's policy, or why it cannot post at all (`posting.refusal`, the server's own words); and the journal entries it became, each opened in place |
| Fuel purchase | a stub with the fuel as a pump abbreviates it (DSL, DEF, RFR) beside the vendor, the place and when; the figure is the total with the quantity and price under it, amber when it differs from quantity × price; the receipt as it was rung up (each line the purchase records, a litre purchase's gallons under its quantity, quantity × price underlined where it disagrees with the total, a missing odometer marked) beside what it does to the quarter's IFTA return in its jurisdiction: the gallons, how many were taxed at the pump, the credit those earn at the quarter's rate (`ifta.taxPaidCredit`, tax-paid gallons × the jurisdiction's `RatePerGallon`, left out when no rate is on file) and what the tractor has bought there this quarter; checks on the total, the odometer against the tractor's previous fill with an odometer (with miles per gallon) and the card against the unit; how it was filed (the feed's or the statement's row and each match the import made, read only under fuel purchase import read, or who entered it by hand); and the card it was bought on, opened in place |
| Fuel card | the network's card in small beside its label; the figure is this month's spend and gallons, or what is waiting on a card nobody holds yet; the card as it is printed (the network's colour and mark from `FuelCardMark`'s faces, the number hidden but for the last four, its label, its expiry, the driver it is issued to), dashed round while nobody holds it and greyed and stamped once cancelled; who holds it, the driver and the tractor each opened in place; the month's figures; the statement rows waiting on the card, dashed because they are not purchases yet, with why each waits; its latest purchases, each opened in place, with what each is to IFTA; the quarter's IFTA fuel by jurisdiction as bars against the largest; whether the provider's feed is reading; why a cancelled card was cancelled and its replacements, opened in place. Purchases and imports are left out for someone who may not read them |

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
and what comes next. The order, both settlement cards, the detention card and both journal
cards draw their lifecycles with it.

The two settlement cards share their frame in `record-card/card-settlement.tsx` and
`record-card/settlement-values.ts`: the stub, the track's steps and note (`settlementTrack`,
`trackNote`), who signed each step (`SettlementSignoffs`), why a voided one was voided and how
far it had got (`SettlementVoidSlip`), the summed statement (`SettlementStatement`), and the
bar of the settlement's own steps (`SettlementBar`). Each kind draws what only it holds, the
worker's waterfall, flags and disputes, the carrier's invoices and remittance, in its own
folder.

The journal entry and manual journal cards share theirs in `record-card/card-journal.tsx` and
`record-card/journal-values.ts`: the lines written into the book (`CardJournal`), whether they
balance (`JournalSeal`), what they do to `Assets = Liabilities + Equity + Revenue − Expenses`
(`CardEquation`, each line moving its account's balance by the side its account type sits on,
cost of revenue counted as an expense, and a line whose account type could not be read left out
and counted), a row for a related record (`JournalRefRow`) and a fiscal period with its state
(`PeriodChip`). Both are read by one query each (`journalEntryCard`, `manualJournalCard`),
assembled on the server by `journalcardservice`, which also works out where an unposted journal
would post today under the organization's closed-period policy. A journal's accounting date is the
first second of its day in UTC, so the cards write it with `accountingDay` and `accountingLeaf`,
which read it in UTC; reading it in the reader's own timezone would give the day before west of
UTC.

The fuel purchase and fuel card cards share the network's card in small (`FuelCardLead`) and a
row naming a card (`FuelCardRow`) in `record-card/card-fuel.tsx`. Each is read by one query
(`fuelPurchaseDeskCard`, `fuelCardDeskCard`), assembled on the server by `fueldeskcardservice`,
which works out the purchase's IFTA quarter in the organization's time zone, its credit, and the
tractor's previous fill, and the card's month, quarter and held rows: the statement rows whose
card is this one that have not posted, in a feed import that is not discarded or an upload not yet
committed. Whether a fuel counts for IFTA is `countsForIfta` in `@trenova/shared/types/fuel-ifta-enums`,
the client's copy of `IFTAFuelType.CountsForIFTA`.

## Actions on a card

A card's buttons are the record's own steps, taken by the person and never by the agent,
each shown only to someone holding the permission the server checks for it
(`useSettlementSteps` reads them once for a settlement card; the invoice card reads its own;
the detention card reads detention policy update once, the permission the server checks for
approving, sending the notice, recording a dispute and waiving; the EDI partner card reads EDI
update once, for testing a line and picking up its files now, and offers neither for a
Trenova-to-Trenova partner, which has no line; the journal entry card reads journal entry approve
for approving an entry waiting on approval and posting an approved one, and journal reversal
create for asking for a posted entry to be reversed; the manual journal card reads manual journal
submit, approve, reject and cancel, and approves and posts a submitted journal in one step,
reading the journal again when the approval holds but the post does not; the fuel card card reads
fuel card update for assigning a card nobody holds, with the fuel cards page's
`AssignFuelCardDialog`, and fuel purchase import import for then working each import holding its
rows again (`resolveFuelPurchaseImportRows`, one import after another at the version the card read,
so a refusal is said in the card and the rest are still worked); assigning a suspended card
activates it on the server).
`CardActionButton` (`record-card/card-action.tsx`) draws each: when something stands in the
way (an invoice that has not been posted cannot be sent by EDI) it stays where it is,
disabled, and says why on hover and to a screen reader, rather than vanishing. A step that
needs more from the person (a reason to send back or void, how a settlement was paid) opens
the dialog its page already uses, imported from the page, so the card and the page cannot ask
for different things; when that dialog lives in a larger component, it moves to a module of its
own first (the detention desk's waive and dispute dialogs are `occurrence-action-dialogs.tsx`), so
the card does not bring the page's sheet with it A dialog more than one route uses lives in
`src/components/` (`reason-dialog.tsx` asks why a manual journal is rejected or cancelled and why
a remittance line is ignored), and a dialog that asks for what a page's form asks for draws that
form (the journal entry card's `request-reversal-dialog.tsx` draws the reversals page's
`JournalReversalForm` with the entry fixed, and both send `reversalRequestBody`). After a step the card reads its record again through the module's own
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
what they may see. A record's page opens the record named by its link (`panelEntityId`) even
when it is not on the table's first page: `DataTable` reads that one row by its id, under the
table's own scope, and opens its panel once it arrives. Every card's in-place surface takes `backLabel` (`InPlaceRecordProps`) and
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

## A table that pages

A list is read a page at a time, and the table it makes is the whole list paged, not the page
the agent read. `tableArtifact` keeps the agent's rows with where the list goes on from them
(`hasMore`, `nextCursor` or `nextOffset`, `pages`, and `total` when the read counted), in
`assistantservice/artifact_paging.go`.

- **The agent's later pages join the table.** A list call that starts after a cursor or at an
  offset the conversation's newest table of that lineage handed out (`continueTable`) adds its
  rows to that table, in the table's own columns, and moves its position on; it is never a new
  version (the "v2" that held only page 2) nor, when the first page had been dropped, a new
  artifact. A page that would push the payload past `MaxPayloadBytes` leaves the table as it is,
  and the person reads the rest from where it ends. A cursor the table never handed out is kept as
  before, as the lineage's next version.
- **A paged table is always kept.** `ShownArtifact.Paged` makes it `MustBeLinked`, and `shownNote`
  tells the model the person pages through every row beside the conversation, so it reads another
  page only when its answer needs one.
- **The person's pages are read again with their own access.** `GET
  /assistant/threads/:id/artifacts/:artifactId/rows/` (`ArtifactRows`) runs the tool the table came
  from with the arguments it ran with, as `UserActor` under the conversation agent's data
  access ceiling, exactly as the CSV export does (`listOf` serves both): `after` or `offset`,
  `limit` (at most 100), `q` when the list takes a free-text search the agent's call did not
  already use, `sortBy` (a column key, mapped to the list's sort field) and `sortDirection`, and
  `total=true` to count. It answers the rows projected into the table's columns, where the list
  goes on, the total when asked, the columns the person may not see, and whether the list
  searches and which columns it sorts by. A person who may no longer read the list gets a 403.
- **A table keeps the arguments its list ran with** (`readWith` on the payload, from
  `ToolObservation.RanWith`), not the model's call as the thread records it. The runtime repairs a
  call before the tool runs (aliases, coercion, relative dates resolved against the day), and a
  list's cursor is bound to the arguments it ran with, so replaying the model's raw call is refused
  as a stale cursor, and replaying it a day later reads another range. A step replayed from the
  ledger keeps them too (`RunStepOutcome.RanWith`). A table or report preview without `readWith`
  is not read again (`errNotReread`); a cursor the list no longer takes is a 400 on `after`
  (`errStalePage`), and the client then reads the list from its start.
- **The client reads a buffer, not pages.** `useTablePages` (`kinds/table/use-table-pages.ts`)
  starts with the agent's rows and reads on from their position with `useInfiniteQuery`, a page
  as soon as the table opens (so Next is instant, and the total arrives) and the next whenever the
  page shown needs rows not yet read; a page is a slice of the buffer at the person's size (25 or
  50, kept per browser). Searching or sorting reads the list from its start; a list the server
  does not search is filtered over the rows read, and a column it does not sort by stays fixed. A
  table cut to fit has no position, so paging past its rows reads the list from its start. A
  table that holds its whole list pages where it is and reads nothing.
- **Whose read a page is** is shown: rows the conversation's agent read wear its mark and name
  ("Dispatch Desk read this page", a part page says so), "Read just now, with your access" on
  the rest, and the strip of pages under the table marks the agent's pages in its accent, the
  ones read and the one open. A failed page keeps the pages read and offers Try again; a 403
  stops the table at the agent's rows with a lock note; withheld columns are named.
- **The person can read the list again now.** A table that kept `readWith` offers "Read again
  now": `useTablePages` reads the list from its start, live, with the person's access, in place
  of the agent's rows (also for a table that holds its whole list, which otherwise reads nothing),
  and a note under the toolbar says what the agent read is kept, with "Show it" to return to it. The stored table is never changed, since it is
  what the agent's answer read. Each ask is a read of its own (`readAt` in the query key), never
  one kept from before; a local sort the server may not take is dropped. A 403 on that read goes
  back to the agent's rows with the lock note.
- **The agent is the one the turn ran as, never Nova.** Nova is the product's own AI (Triage,
  briefs, Ask Nova) and wears the Nova badge; a Desk conversation is with an agent the person
  chose and can hand over. The recorder stamps a table with that agent's name and mark when it
  keeps it (`readBy`: id, name, icon, accent, read once per turn by `agentIdentities`), as a
  document keeps its author, so a hand-off later leaves the table credited to the agent that
  read it. A table without `readBy` (the agent could not be read) says it was read "in this
  conversation".

The mock API's `athr_desk_table` conversation serves it (`handlers/desktable.mjs`, the
`deskTable` scenario: paged, slow, failing, locked, withheld, cut, whole, short), and
`e2e/mock-api/desk-table-screenshots.mjs` shoots each state.

## A report run

A `report_run` artifact is the report the agent (or the person) ran, followed to its result;
a `report_preview` is the same sheet for the rows a preview read. Both draw in
`kinds/report/` from one sheet model (`report-sheet.ts`), fed by the run's stored sample or the
preview's payload.

- **A finished run keeps a sample.** Every run captures the digest scheduled email digests
  use (`digestCapture`: the first 20 detail rows, the grand total and the columns with their
  resolved display, written as a keel export JSON envelope); `FinalizeRunActivity` stores the
  envelope on `report_runs.sample` when it is at most 48 KB (`MaxRunSampleBytes`), and skips
  it otherwise. The column is `BYTEA`, not `JSONB`: `export.ReadJSON` needs the envelope's
  keys in the order they were written, and JSONB reorders them. Lists never select it
  (`Omit(Sample)`), and expiry clears it with the file. A run with no sample shows the run and
  the download.
- **One read draws the card.** `reportRunDeskCard(id)` (`reporting.GetRunCard`) returns the
  run, the report's name, description and category, its parameters with the window a relative
  parameter resolved to when the run started (`analytics.RelativeRange` in the organization's
  time zone), the sample read back as a `ReportPreview` (the type the Reports grid formats), the
  person's previous finished run of the same report and parameters
  (`ReportRunRepository.LatestComparable`, behind `idx_report_runs_comparable`), the schedule it
  came from and the report's first chart and sort. A finished run's sample is withheld
  (`withheld`: `access` or `changed`) exactly when its download would be refused.
- **The picture follows the report.** The report's own chart decides: bar, pie and donut rank
  (a bar in each row, with each row's share of the grand total), line and area trend (columns
  and a line by period), map, scatter and KPI leave the table. Without a chart a result led by
  dates trends and one led by a label ranks by its sorted measure, else its first money measure.
  A trend is drawn only when the sample holds every period.
- **Figures are the grand totals**, against the previous run's ("vs Oct 20"), never summed on
  the client. A capped run says its figures are of the kept rows and compares nothing.
- **Run again** (`POST …/artifacts/:id/run/`, `report:export`) starts the same report with the
  same parameters and format with the person's access and keeps it as the artifact's next
  version, `ranByPerson` set; on a preview it is "Run the full report", refused for a definition
  that was never saved. The card names the agent that ran it from `readBy`, like a table.

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
it with `formula_list_templates`, and the person sees the template rather than a row
to open; a search that found one shipment is the shipment's card the same way. A page of one with more behind it (`hasMore`, or a larger `count`) stays a
table.

The billing queue's table is the richer case: its rows are also selected and
approved in bulk, so it keeps its own table body (`BillingTableBody`), and a row
opens in the conversation's billing item artifact when there is one. Its card is a
registry entry like any other.

## Adding a card

The `desk-artifact` skill (`.claude/skills/desk-artifact/`) walks through the whole
change in order, from the mockup to a green PR, with copyable shapes in its `recipes.md`.
The steps below are the rules it follows.

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
