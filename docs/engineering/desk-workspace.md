# The Desk as the operation's room

The Desk is one room: the rail, the canvas (Today, Needs you, a record's case, a line of Nova
handled, Triage) and Nova's margin, under one top bar. This page is how the parts that join them
work. The approved design and its prototype live in the git-ignored `design-plans/desk-workspace/`;
the surfaces copy its anatomy. Shared pieces live in `routes/desk/_styles/desk-room.css` under the
`dk-w-` prefix; a rebuilt surface has its own stylesheet and prefix beside its components.

| Part | Where |
|---|---|
| The top bar's situation line, Search or do, Ask | `desk-topbar.tsx`, `desk-situation.tsx` |
| Nova's margin | `margin/desk-margin.tsx`, `stores/desk-margin-store.ts` |
| Needs you, a row, an item's detail, deciding in place | `deskattentionservice`, `deskAttention`, `attention/` |
| Agents handled and one line in full | `deskactivityservice`, `deskActivity`, `deskActivityEntry`, `today/`, `handled/` |
| A record's case | `recordcaseservice`, `deskCase`, `record-case/`, `/desk/r/:kind/:id` |
| A composed view on the canvas | `view/` (`dk-vw-`), `/desk/v/:viewThreadId/:slug` |
| Labels under the reader's access | `deskrecords.Labeler` over `recordsummaryports.RecordSummarizer` |
| Triage, the one place mail is worked | `/desk/triage` renders `TriageRoom` (`routes/inbox/page.tsx`); its folders are drawn in the Desk's rail (`TriageFolders`, `desk-rail-slot-store`) |

## The room and Nova's margin

`.dsk` is a grid of rail, canvas and, while it is open, a 380px margin (`dk-w-mg-open`). The margin
holds one conversation (`DeskThread`, compact) or, before one exists, the Desk's box. Its state is
`useDeskMarginStore`, in memory only: which thread, the record it is about, the subject the open
page publishes (`setPageSubject`) and the cases opened this session (Open work in the rail). A
record's name is never written to browser storage, and the store is cleared with the session.

- **Asking opens the margin, never a page.** Today's box, the case's box, ⌘J and the top bar's
  **Ask** start a conversation in the margin (`DeskContext.startInMargin`). A question about a
  record carries it as a mention and binds the thread to the record's case
  (`useStartConversation`'s `StartBinding`: title, case subject, page context), so the margin
  reopens the same case conversation next time.
- **The rail opens conversations beside the work.** A plain click on a conversation row shows it in
  the margin; a modified click, or a click while a conversation fills the room, follows the link.
  **Open full width** goes to `/desk/t/:id`, which closes the margin on that thread.
- **Beside a case sits the case's conversation and no other.** While a case is in front
  (`casePage` in the margin store, set by the case page), `showThread` refuses a conversation about
  anything else and `openMargin` opens on the case. A rail conversation clicked there follows its
  link, and New conversation goes to `/desk`. A margin opened on something else gives way when a
  case opens.
- ⌘J toggles the margin on what the page is about; ⌘\ and Esc fold it.
- **What an agent makes opens on the canvas.** An artifact opened from the margin's conversation,
  or with **Open on the canvas** in a conversation's workspace, goes to
  `/desk/v/:viewThreadId/:slug` (`view/desk-view.tsx`, `viewPath`): the question it answers, its
  title, who composed it and when it was read, then `ArtifactBody` at full width, with the
  conversation that made it in the margin. The route's parameter is not named `threadId` on
  purpose: `DeskPage` reads `threadId` as the conversation filling the room. The view opens the
  margin itself only from 901px, where the margin sits beside the canvas rather than over it, and
  joins Open work (`DeskOpenWork.path`). The name in the link is the lineage's slug, or an
  artifact's id when it has none (`ArtifactBySlug` looks the id up as written, not lowercased).
  There is no bare artifact page any more.
- A page that knows its own name publishes it with `setPageTitle` for the top bar's crumb (a
  ledger line's title, a view's title).

## The situation line

Up to three of the morning briefing's first figures (section title, first item's value and path),
then **Needs you** (open items across lenses) and **Agents working**: `WatchtowerCounts.withAgents`, open
items whose agent run is running right now, counted in the same pass as the queue's other totals.

**Nova is not the agents.** Nova is Trenova's own assistant: the margin, Triage's reading and
drafting, the briefing's voice. Agents are the organization's, defined in AI control. Work an
agent did or is doing is labelled as the agents' ("Agents handled", "Agents are working on",
"Agents working") and wears the agent's own tile, never Nova's mark.

**"Working" means a run that is running.** One definition, `agent.WorkingRunStatuses` (Pending,
Gathering context, Diagnosing), decides it everywhere: the count (`agentWorking` in the Watchtower
repository joins the item's run), an item's `agentWork.status` (`agentwork.From`), and Today's
**Agents are working on**, which reads `deskAttention(working: true)` so a run far down the queue is
still listed. A run that finished, failed or waits on a person is not working, whatever status its
item still has. A run that ends having proposed nothing closes nothing: the condition the item
reports is still true, so the item stays open and the case says the agent looked and proposed
nothing. (It once resolved the item as No action needed, which closed a missing proof of
delivery that was still missing.) The run's own realtime notice (`agent_run`) refreshes the count and the Desk's rows
(`handleRun` in `watchtower-realtime.ts`), so a row appears when a run starts and leaves when it
stops, with no queue on screen. The Desk reads the working runs themselves
(`aiControl.workingRuns`, the query AI control uses) for whoever may read runs, so the two screens
show the same number; a run on an item of Needs you is drawn as that item (`workingRows`). A run
still in a working status six hours after it started has lost its workflow and is ended as failed
by the proposal-expiry sweep (`AgentRunRepository.FailAbandoned`), so nothing says an agent is
working that is not.

## The list pane

A Desk page that lists things on the left and reads the chosen one beside it uses the one list
pane, `routes/desk/_components/list/desk-list.tsx`. Needs you (and so its Watchtower and Decisions
lenses) and Triage's queue are both it; a new page with a list in that place starts there rather
than with a scroll container of its own.

- `DeskListPane` is the frame (the Desk's canvas, the hairline on the right); the page stacks its
  head in it from the parts in `list/desk-list-head.tsx`, one row each: `DeskListTitle` (the name,
  with a scope beside it and progress at its end), `DeskListProgress`, `DeskListTabs` (the views on
  the list, each with what it holds, and the filter at the row's end), `DeskListSearch`
  (`list/desk-list-search.tsx`) and `DeskListTokens`.
- **A view and a filter never look alike.** A view (a lens, an order, whose cases) is a quiet word
  that holds the active surface while chosen (`DeskListTabs`, `DeskListToggle`). A filter is a
  bordered token with an x, drawn only while it is on (`DeskListTokens`), and every list's filters
  sit behind one **Filter** button (`deskListFilterClass`, `DeskListFilterLabel`) whose menu the
  page brings, since what can be filtered is the page's to say. The menu narrows as a choice is
  made and stays open; nothing is applied with a button.
- **Search runs on the server.** A list shows a first page, so a search over the rows in the
  browser would miss most of the list. `DeskListSearch` hands the page a settled value a quarter
  second after typing stops; Escape empties it and Enter keeps it, and both give the keys back to
  the list. The page keeps the words and the filters in the address.
- **The page's keys stand down while the filter is open.** The menu takes typing, and E, S or A
  pressed in it must not clear or approve the row behind it: the page holds whether the menu is
  open and turns its keys off (`keysEnabled`, `keysPaused`).
- `DeskList` is the rows: `VirtualRows` in the app's `ScrollArea` with the Desk's fade, the list's
  gutter, and one hover that glides from row to row (`HoverGlideGroup`).
- `DeskListHeading` is a group's heading (a lane, "Over target"); `DeskListStill` holds what stands
  in the list's place (empty, error, skeleton).
- A row spreads `deskListRow(chosen)`: it marks the row as a glide target and gives the chosen row
  the Desk's active surface. A row never sets a hover background of its own.

What a row says, what it is filtered and searched by, and its keys stay the page's own. A long
page that is not a list (Triage's Briefing) scrolls in `ScrollArea` with the same fade.

### A pane whose list has views

When the views on a list hold different things (Needs you's lenses), the page is still one
component, and a view is a variant of it, never a second page drawn in the same place. Two
pages that each draw the pane take the whole pane down and build it again on a change of view:
the head blinks, the tab that was clicked is gone with its focus, and nothing can move from
one view to the next because nothing survives the change. `list/desk-views.tsx` is the frame
that prevents it.

- **The frame is drawn once and a view fills two cells of it.** `DeskViews` is a grid: the head
  (`DeskListHead` with `DESK_VIEWS_HEAD_CLASS`) at the top of the list's column, and under it and
  beside it the cells a view brings, `DeskViewList` and `DeskViewDetail`. The page owns the head
  and everything it shows (the view in front, the search, the filters, how far along the work
  is); a view owns its rows, its selection, its keys and what the chosen row reads. It is a grid
  and not a pane beside a page because a view's list and detail are one component with one
  selection, and one component cannot draw into two parents.
- **A view left is kept, not unmounted.** Each view sits in `Kept` (`components/kept.tsx`), so
  its scroll, its selection and what it has read are there when the person comes back, its
  effects (keys, address writes) rest while it is hidden, and a view never opened is never
  mounted or read. The view takes `shown`, and marks its detail `reading` only while it is in
  front and a row is open, which is what gives a narrow window to the detail.
- **A view is read when it is opened, or a moment before.** `DeskListTabs` tells the page when
  the pointer or the focus reaches a tab (`onIntent`), and the page reads that view ahead:
  its code (`lazyComponent(...).preload`) and its first page (`prefetchQuery` with the same
  options the view reads with, such as `pendingDecisionsOptions`). So the view is there when it
  is picked, and one nobody heads for costs nothing.
- **What the head says is read where the head is.** The page reads what a view's head needs
  (counts, progress, what the filter offers) with the same query options the view uses, so the
  two share one request, and keeps the last answer while another view is being read. Tab
  counts therefore hold still from view to view. A view never hands its head up through an
  effect, and never draws a head of its own.
- **While a view is read its rows are a skeleton, and the head stays.** `DeskListSkeleton`
  stands in the list cell only; the frame is never replaced by a skeleton of the whole page.

### Moving between views

- **One marker under the tabs.** `DeskListTabs` draws the active surface once and glides it to
  the tab that is picked (`translate` and `width`, 200 ms, `ease-settle`), measured from the
  chosen tab and again when a tab's width changes. A tab never lights itself.
- **The two cells cross-fade; the rest of the page does not move.** A change of view is a
  navigation that asks for a view transition (`navigate(..., { viewTransition: true })`, as the
  rail's links do). Each view names its list and its detail after itself
  (`NEEDS_LIST_TRANSITION_CLASS`, `NEEDS_DETAIL_TRANSITION_CLASS` in `desk-motion.ts`), so a
  change of view has one pair leaving and another arriving (`:only-child` in `desk-motion.css`),
  and anything that keeps the view finds the same names on both sides and changes at once. The
  head is not named: it is the same element before and after.
- **Rows rise on every arrival.** `useListEntrance(view, hasRows, shown)` says when a list's rows
  (`dk-w-ar`) have settled; a view shown again starts over, so arriving at a lens reads the same
  whichever lens it is and whether it was open before. A row on its way out gives the entrance
  up, so its leaving is not held in place.
- **Nothing moves under reduced motion** (the system's, or the Desk's `dk-calm`): the regions
  are left unnamed and the marker does not travel.
- **The address is read from the router for the view and written through it.** The search and
  the filters are written to the address without a navigation (nuqs), so the router's own copy
  of the query goes stale; a change of view builds the next address from `window.location` and
  not from `useSearchParams`, or it would drop them.

## An item's panel

A Needs you item is not read from one generic table. Its source says what is happening in a typed
shape, and the item is drawn by the view made for that shape, so a missed stop, a detention clock
and missing paperwork do not look alike.

- **The shape is the server's.** `watchtowerpanel.Panel` (`domain/watchtowerpanel`) is a `Kind` and
  the one struct that kind names: `Stop` (missed, projected late, at risk), `Clock` (detention),
  `Paper` (documents a load waits on), `Message`, `Expiry` (credentials), `Coverage` (uncovered
  move, exhausted tender), `Contact` (no check call), `Run` (failed agent run), `Books`
  (accounting), `Money` (factoring, a billing exception), `Change` (a carrier's record, an agent's
  score), `Ask` (an agent that stopped), `Finding` (an insight), `File` (a quarantined EDI file)
  and `Sighting` (a stop visit the tracker saw and nobody recorded). A late arrival is a `Stop`
  with `ArrivedLate`. A source fills it in
  its `Describe…` function beside the facts; it is stored as `watchtower_items.panel` (jsonb),
  normalised like every other text on the feed (no record ids, bounded), and is part of what makes
  an upsert a change. GraphQL: `WatchtowerItem.panel`.
- **The view is the client's.** `attention/panels/item-panel.tsx` picks the component by which
  field is set; each panel is its own file in the shared frame (`PanelCard`, `Figures`). An item
  with no panel is read from its facts by `FactsPanel`, in the same frame.
- **Opening a case to work it takes it.** **Open the case** from Needs you carries `START_CASE`;
  the case page then claims an unowned case for the opener (`useClaimOnOpen`, `assignDeskCase`
  with no owner). A case that is someone else's stays theirs, even for a lead.
- **A failed run is news until the agent runs again.** `FailedRunSource` reports an agent's latest
  failure only while no later run of that agent has ended well (`stillFailing`), so a timeout the
  agent got over is cleared at the next reconcile.
- **Not every item needs a case.** A panel works the thing where it can: `PaperPanel` reads the
  shipment's own document checklist (`billingReadiness`) and files a missing paper from its row
  (`documentService.upload` with that row's document type), then closes the item as Document
  received once nothing is missing. Every panel also ends in a link to where its thing is fixed
  (`PanelActionContext`, worded per kind in `item-panel.tsx`), from the item's `path`.
- **Adding one** is a struct and a kind in `panel.go`, a type in `watchtower.graphqls` and
  `gqlgen.yml`, the field in the `WatchtowerItemFields` fragment, and one component. Existing
  items take their panel at the next reconcile.
- **Two clocks, two sizes.** A deadline the record sets (`dueSource: Source`) is the large clock. A
  policy target is one quiet line, since the panel already says how late the thing itself is.
- **Triage leads with what its kind is read for.** `leadFacts` (`reader/reading-lead.ts`) puts a
  tender's lane and respond-by, an invoice's amount, a dispute's money first and larger among the
  facts Nova read.

## Showing less

The Desk is read by dispatchers, billers and collectors between other things. A page shows what
the task in front of the person needs and keeps the rest one press away.

- **An agent's note is a report; the queue shows its gist.** `AgentWorkBlock`
  (`attention/agent-work.tsx`) says where the agent stands ("is on it", "needs your decision",
  "looked at this"), its steps while it runs, and one plain sentence (`workNoteGist`). The whole
  note is rendered as markdown behind **What it found**, pre-rendered in a hidden `<Activity>`.
- **A list of groups opens one.** Triage's Briefing lists every group of what is left, with only the
  first open; the others name what they are, how long they take and their one button.
- **A view someone returns to keeps its state** through `Kept` (`components/kept.tsx`): Triage's
  Briefing and Rules stay as they were while the queue is in front.

## Needs you

One row per record, soonest deadline first, made of Watchtower items: the Watchtower already
projects proposals, plans and inbound messages as items, so grouping is a regrouping of one queue,
not a join of three. Items group on the record they are about ([exception-queue.md](exception-queue.md),
"What an item is about"); an item about nothing beyond itself, or about a kind with no permission
resource, is its own row with no record.

- **Two bounded reads, both through `WatchtowerService.List`**, so kind visibility, seen marks and
  cursors stay in one place. The first reads up to 200 open items (Open, Acknowledged, In progress)
  in the lens's kinds, narrowed by `kinds` and `severities` when the address names them (the
  briefing's links do), soonest first, and keeps the first `first` rows (at most 100). The second
  reads every open item about those rows' records in any lens (`AboutIDs`), so a row in the
  Decisions lens still shows its Watchtower condition and its email.
- **Lens counts** sum `WatchtowerCounts.ByKind` by lens.
- **Search and filters narrow the first read, never the second.** `deskAttention` takes `search`
  (an item whose title, summary or record label holds the words, any case, at most
  `watchtowerports.MaxSearchLength` characters; `%` and `_` match themselves), `severities`,
  `kinds`, `unseen` (occurred after the reader's cursor and never opened, the predicate the counts
  use: `unseenBy`) and `working`. They pick which records make the page; the second read still
  brings every open item about those records, so a row found by its late warning shows its email
  and its plan too. The read walks the tenant's open queue in due order
  (`idx_watchtower_items_queue`) and filters as it goes, so its cost is bounded by what is open,
  as the counts' is. There is no trigram index: the queue is the open items of one tenant, and a
  GIN index on a table every reconcile rewrites would cost more than the scan it saves.
- **The filter offers what is there.** `DeskAttention.kinds` counts the open items of each kind in
  the list's scope (`mine` included), most first, from the counts the read already makes; a kind
  the address names is listed with nothing open, so its token keeps its name. The menu offers the
  kinds of the lens in front (`offeredKinds`). The filters live in the address (`needs-filters.ts`:
  `q`, `severities`, `kinds`, `unseen`, `working`), so the briefing's links and a link sent to
  someone open the same list. The Decisions lens is narrowed to one agent and one kind of change
  (`decision-filters.ts`: `agent`, `change`), which `pendingDecisions` already takes, and offers
  them from `pendingDecisionSummary`.
- **The search is the page's, and every lens answers it.** The words are one address key (`q`)
  that stays as the person goes from lens to lens. Under Decisions, `pendingDecisions(search:)`
  keeps the proposals and plans whose Watchtower item holds the words in its title, its summary
  or its record's name (`said` in `agentdecisionqueuerepository`, an `EXISTS` on
  `uq_watchtower_items_source`). It reads the item rather than the proposal on purpose: those
  are the words Needs you searches under Everything, so a search finds the same decisions under
  every lens, and a record's name is on the item and not on the proposal. What was decided
  today is not narrowed by the search or the filters, so the queue lists it only while it is
  whole.
- **Labels** come from the record summarizer, one batch per kind, and only for kinds the reader may
  read (`deskrecords.Labeler`); a record they may not read is counted, never named. A readable
  record with no label is treated as no record: it names nothing and opens no case.
- **The page** is two panes: lanes of rows on the left, the item on the right with its condition,
  the agent on it or the one that could take it, the message and the decision, and a sticky bar.
  `useWatchtowerActions` owns J/K/E/S/H, done, snooze and hand-off; the page adds Enter for the
  case and A/D for a decision, and refuses E and S on a decision.
- **What a row reads is centred in the detail, under every lens.** The item (`ItemDetail`) keeps
  to one measure in the middle of the detail cell, as a decision's card does, and the bar under
  it holds its note and buttons to the same measure while its rule and blur run the cell's
  width (`ITEM_MEASURE_CLASS`). A detail is never left hugging the list.
- **One page, one head, and a lens is a variant of it.** `DeskNeeds`
  (`attention/desk-needs.tsx`) is the frame ("A pane whose list has views", above): it reads
  the lens from the address and draws `NeedsHead` once, which owns the name, the progress bar,
  the lens tabs and the keys 1–3, / for the search and F for the filter. Under it,
  `NeedsAttention` (the rows by record) fills the cells for Everything and Watchtower and
  `DecisionFlow` (`decisions/decision-flow.tsx`) fills them for Decisions; each is kept while the
  other is in front, and `DecisionFlow`'s code is read when the lens is first opened or headed
  for. What each lens says in the head comes from `useAttentionHead` and `useDecisionsHead`,
  which also hold what the lens is narrowed to and hand it down. A person sees the lenses they
  may read (Watchtower for the first two, agent proposals for Decisions); the route lets in
  whoever may read either (`anyOfLoaders`, and the `or` requirement in `routeaccess`), and does
  not run its guards again when only the lens changes.
- **The Decisions lens is the only place changes are decided together.** It lists the pending
  queue (`pendingDecisions`) by the kind of write; changes are ticked in the list and decided
  from the dock at its foot, and the page beside it reads one change
  ([proposal-previews.md](proposal-previews.md)). There is no `/desk/decisions`: every link to
  the queue is `/desk/needs?lens=decisions`, with `&proposal=<id>` or `&run=<id>` to open on one
  change (`lib/desk-paths.ts`). A decision on a record's row under Everything still has its card
  there, and links to its sheet.
- **A message is a signal here, never a lens.** `desklens.Triage` says where an item came from, and
  a row draws it as mail beside the record it is about, but Needs you has no Triage lens
  (`NEEDS_LENSES`): mail is read, answered, filed and set aside in Triage and nowhere else. A
  message's item offers **Open in Triage** (Enter when the row opens no case) and takes no Done or
  Snooze (`itemsWorkedElsewhere`), because the Watchtower clears it when Triage settles the message.
- **Deciding in place** (`useItemDecision`) reads the item's proposal, or every step of its plan,
  and approves with `decideAgentProposal` / `decideAgentPlan` as the Decisions lens does, carrying
  the digest of every preview the server asks to be shown (`ReviewedChange`/`ReviewedPlan` with
  the approver scope). Nothing reads as done until the server answers; a refusal is shown in the
  server's words. Decline asks why and sends it as the decision's note.
- `/desk/watchtower` is gone: the Watchtower lens is it, and the old address redirects there.
  Links to one item are `/desk/needs?item=<id>`.
- **Mine and Everyone.** A person who may see everyone's cases (`caseowner.Sight.All`, the
  Watchtower's Manage) reads their own and the unowned ones by default and everyone's when
  they ask. `deskAttention(mine: true)` narrows the list and its lens counts
  (`ListWatchtowerItemsRequest.Mine`, `WatchtowerService.CountsFor`), and the answer's
  `seesEveryone` says whether to draw the switch at all, beside the page's name. The choice is
  the `cases` Desk setting, kept in the person's browser. The rail's badge still counts everything they may
  see.


## Agents handled

What agents changed, read from `agent_proposals` itself rather than the AI audit trail: every write
an agent makes is a proposal, and one that ran on its own carries `AutonomyTier = AutoExecute`. The
read orders by `COALESCE(executed_at, simulated_at)` over `idx_agent_proposals_settled` and folds
consecutive writes of one tool in one run, or one plan's steps, into one line. The outcome is never
more than the record says: previewed (`simulated_at`), failed (`execution_error` or
`ExecutionFailed`), ran on its own (`AutoExecute`), else approved, with the decider read in one batch
from `agent_decisions` or the plan. Titles use the write's own `ExecutionResult` ("Filed 11 rate
confirmations") or the tool in words, never the tool's name. It needs read access to agent
proposals and reaches back at most 31 days.

`deskActivityEntry(id)` reads one line by the id of any write it folds: the run's settled proposals
(`ListByRun`, bounded by the run), folded the same way, the line that holds the id, every write it
folds with its record, and how often this agent made this write in the last 30 days
(`CountExecutedTool`). Each write carries what it made in words (`whatOf`: the execution result, or
the tool in words), the agent's reason and when it was proposed. `/desk/handled/:id` draws it: the
rule it ran under or who approved it, its record, what each write made and to which record, and
the run as it happened (`runTrail`: proposed, what let it run, what the server confirmed). When the
writes of a line each have their own reason the reasons sit in their rows, not over the page.
There is no undo for a write that has run: only an approval inside its undo window can be taken
back, so the page offers none.

## A record's case

`/desk/r/:kind/:id` reads `deskCase(kind, id, timezone)` in one round trip and draws the record as
one decision. The anatomy is the same for every kind of record; what a case of a kind is made of is
decided on the server, so a late shipment, an invoice past due and a disputed invoice are one page.

| Part | Where |
|---|---|
| The read model | `agent/domain/recordcase` (`Case`, `Head`, `Finding`, `Exhibit`, `Decision`) |
| The service | `agent/app/recordcaseservice` (`service.go` composes, `assemble.go` words it) |
| What a kind's case is made of | a `subjectReader` per kind: `shipment.go`, `invoice.go` (invoice and dispute) |
| Options and Nova's reading | `record_case_options`, `record_case_readings` (`recordcaserepository`) |
| Nova's reading, written | `recordcaseservice/casereading`, `WriteCaseReadingWorkflow` (`completionjobs`) |
| The page | `client/.../desk/_components/record-case/` (`dk-cz-` in `case.css`; `dk-cs-` belongs to the conversation's case card) |

- **Signals.** Every open Watchtower item about the record (`AboutIDs`, the reader's kind
  visibility applied by `WatchtowerService.List`). A record the reader may not read is refused; one
  with no label has no case (`recordsummaryrepository` names it, invoice disputes included).
- **Exhibits are evidence, not items.** A kind's reader returns typed exhibits (`Route`, `Meters`,
  `Timeline`, `Facts`) from the record's own sources: the shipment's from one
  `ShipmentTrackingReader.TrackingSnapshots` read (position, hours, stops with their projected
  arrivals), the invoice's from the invoice and its open dispute. Each inbound message about the
  record is a `Message` exhibit (two at most), and each distinct condition the reader's exhibits do
  not already show is a `Facts` exhibit (three at most); identical conditions are one. Exhibits are
  numbered in the order drawn, and each carries the `Claims` the reading may cite.
- **The decision is found by its record, not by "the first item".** `Decisions` reads the plans and
  single proposals whose target is the record (`idx_agent_plans_target`,
  `idx_agent_proposals_solo_target`), live or decided in the last day, so a plan stays on the case
  after its Needs you item clears. What is in motion leads, then what waits, then what settled;
  other waiting decisions are counted once per title (`Decision.Others`). `DeskCaseDecision.phase`
  is the only source of where a plan stands: the client holds no phase of its own.
- **Approving from the case waits out the undo window.** `decideAgentPlan` and
  `decideAgentProposal` take `undoWindow`; the approval is recorded as `Approving` and commits when
  the window closes, exactly as an approval from the person's own conversation does.
  `undoAgentDecision` and `commitAgentDecisionNow` settle it for the person who approved it
  (`SettleApprovalRequest.AsDecider`), behind the same right as the approval.
- **What could happen.** An agent offers two or three options with `agent_offer_options` (a core
  tool: it writes the choices and changes no record), one recommended and drafted in the same run.
  `chooseDeskCaseOption` marks another chosen and starts a run of the agent that offered it with
  the option on its payload (`AgentRunPayload.CaseOptionID`, `Option.DraftNote`); the plan that run
  drafts is the case's plan (`Option.DraftRunID`). An option's `Projection` (a step it adds, a
  window it moves) is drawn on the exhibits as proposed. Choosing to leave the record as it is
  drafts nothing: `acceptDeskCaseRisk` resolves the record's conditions as needing no action, with
  the reason.
- **One case per piece of work.** A record that is worked as another one answers with that one's
  case: an invoice with an open dispute returns its dispute's case (`Service.stands`,
  `invoiceStand`), where the plan and the open items are, and the page follows `record` to that
  address. A reader who may not open the dispute gets the invoice's own case.
- **Nova's reading** is one short Nova task (`aitask.TaskCaseReading`, billed under
  `aiusage.FeatureCaseReading`): the verdict with `[n]` citations, the deciding fact as its own
  field (`decidingFact`, words copied from the verdict, which `casereading.marked` wraps where the
  verdict already says them, so the model never writes a sentence to carry a mark), why it
  matters and what happens if nothing changes, written only from the exhibits' claims and refused
  whole when it cites a figure they do not hold (`numberguard`) or trails off in an ellipsis
  (`trailsOff`); either is asked for once more with the reason. A verdict that cites no
  message exhibit while one is there is asked for once more too (`unreadMessages`): what a
  customer or a carrier wrote is the one kind of evidence a reading may not pass over, and a
  second answer that still leaves it out is kept rather than the reading lost. The page says how
  much of the evidence the reading cites and marks the exhibits it did not draw on
  (`citedExhibits`). The prompt asks for each fact once
  and one citation per run of claims on the same exhibit, and `recordcase.ParseVerdict` holds the
  second half whatever the model writes: a citation followed by the same exhibit's is dropped, so
  a reading drawn from one exhibit carries one number. It is kept per record and set of signals (`uq_record_case_readings_record_fingerprint`:
  two readers with different sight of a record each keep their own, so neither rewrites the
  other's and nobody reads one drawn from signals they may not see; readings from other signals
  not asked for in a week are deleted with the next claim) and keyed by
  `recordcase.ReadingFingerprint` (the open signals and where the record stands, a deadline read to
  the hour). Opening a case whose fingerprint has no reading claims the row (`ClaimReading`, one
  writer per record) and starts the workflow, whose id is the record and fingerprint; the page
  shows what was computed until it lands, then hears `desk-cases` and reads again. With no
  provider the case says what the Watchtower raised and the plan can still be decided.
- **The page** polls only while something is in motion (a reading being written, a plan in its
  window or running, an option being drafted) and otherwise follows realtime (`deskCase` is under
  the proposal, plan and Watchtower roots). The plan card draws each step from the tool's own
  preview (`agentPlanPreview`, was and new per value) and approves with that preview's digest; a
  tool that cannot preview is drawn from its presenter. ⌘Enter approves, D declines, 1–3 choose.

### The case page's layout

The reading and the evidence scroll in their own columns (`ScrollArea`, `dk-cz-col`), so a claim's
citation brings its exhibit into view beside it (`showExhibit`) without the reading moving. The
box to ask in sits under both, not over them, and its suggested questions keep to one line. Under
1080px of container the page is one column and scrolls as a whole.

The route exhibit falls back to the sketch whenever Google will not draw the map: no key, a key it
refuses or that is out of quota (`gm_authFailure`), a script that does not load (`APIProvider`
`onError`), or a throw while drawing (the exhibit's own error boundary). A map that will not draw
is never shown as an error on a case.

### Asking about a case

- **The agent working the case is the only one asked about it.** The box under a case, and Nova's
  margin beside it, name that agent (`DeskSubject.agentId`, `useDeskHomeAsk`'s `subjectAgentId`)
  and offer no other: no picker, no "another agent fits", no page chip and no model picker, since
  the answer comes on the model the agent runs on. A record with no agent on it keeps the Desk's
  ordinary box.
- **The case continues its own agent's conversation.** `caseThreadOf` (`case-subject.ts`) finds the
  thread about the record with the case's agent; a conversation another agent had about the same
  record is never continued from the case.
- **An agent that runs on its own can be talked to.** `agentaccess` no longer refuses a
  conversation with a scheduled, continuous or event agent; anyone who may use the agent may ask
  it. What it writes there always waits for the person: `agenttoolpolicy.Decide` lowers a
  background agent's call to Ask first when `DecideInput.Conversation` is set
  (`HeldByConversation`), so a person never borrows the autonomy the agent was given, or earned,
  for its own runs. `myAgents` lists such an agent only when it is named by id; the pickers still
  list chat agents.
- **The scope guard knows what the question is asked beside.** `agentguard.EvaluateRequest.About`
  carries the conversation's subject, the page and the records named (`aboutOf`), so "what are my
  options here?" asked on a case is judged as a question about that record. It is part of the
  verdict cache key.

- **A step never shows a record id.** `argumentRows` leaves a lone id out and says a list of them
  as how many records (`isRecordId`); a tool's own presenter words the rest
  (`worker_request_credential_renewal`). The record a step is about comes from the step's preview.
- **A dry run proves the write.** A tool's `Preview` runs the same read and validation its
  `Execute` does (`workerservice.CheckDispatchHold`), so "would go through" is never said of a
  step the save then refuses.
- **A plan names its failed step by number.** `DeskCaseDecision.failedStep` counts from one, as
  `agent_plans.failed_step` does; the page turns it into a place in the list (`failedIndex`).

- **A run nobody asked for does not repeat what its agent has waiting.** A scheduled agent reaches
  the same write again every run until a person decides the first. The recorder reads what the
  agent has pending from the tools it just called (`proposalrecorder.standAside`,
  `ListStandingByDefinition`, `idx_agent_proposals_pending_tool`) and records a write with the same
  tool, target and arguments as superseded: nothing new is planned, projected or notified. A
  conversation's writes are always filed.
- **One plan per record for a run nobody asked for.** A run an event, a schedule or a person in
  AI Control started, whose writes are all about records of one kind and reach two or more of
  them, files one plan per record (`proposalrecorder.stepsByRecord`, `RecordResult.Records`); a
  record with a single write gets a lone proposal. A desk renewing three drivers' papers is three
  pieces of work, so no case shows another record's steps and one failed step stops only its own
  record's. A conversation's writes, and a run across kinds of record (a shipment and its
  invoice), stay one plan. The split plans take the agent's default title, since the name the
  agent gave was for the whole run.

- **A step is checked before it is approved, and the check is a warning.** The plan's line reads
  "Checked: …"; a step the server says would be refused carries the reason under it
  (`previewWarningText`) and never disables Approve (`CaseDecision.refusedStep`). A tool's
  `Preview` that comes back with one of the write's own refusals is a `would_fail`, not a preview
  that could not be made (`proposalpreviewservice.assemble`, `errortypes.IsRefusal`).
- **A warning can carry its remedy.** `pkg/remedy` names what would put a refusal right
  (`InviteDriverToPortal`, `SetUpEmailProfile`); the module that refuses marks its error
  (`remedy.Mark`) or attaches the remedy to its warning, and it reaches the client as
  `AgentPreviewWarning.remedies` with the record named, never its id. The client's one registry
  (`proposal-preview/preview-remedies.tsx`, `REMEDIES`) says who may take each, the action when
  Trenova has an API for it and the place to do it by hand; `PreviewRemedies` draws them under the
  warning wherever a plan's warnings are drawn. A new remedy is one kind, one mark and one entry.
- **Only a live agent's work makes a case.** A plan or change from an agent in shadow, or while
  agents are paused, is left out of a record's case (`Service.live`, `agentshadow.Resolver`), and
  a shadow or simulation agent's plan is projected about no record (`proposalrecorder.worksLive`),
  so it is never in Open work.
- **A plan that runs to its end closes what it answered.** `agentplanservice.settle` resolves the
  conditions open about the plan's record as Fixed, through `watchtowerports.RecordSettler`
  (the same audited path a person's Resolve takes, as the approver); decisions waiting on the
  record are left alone, and a source that still finds the problem raises it again. A plan that
  stops part-way or is declined closes nothing.
- **The case shows the plan that just ended** for fifteen minutes before another decision waiting
  on the record takes its place (`rank`), so approving shows how it came out.
- **A stop's times are on the stop's clock.** The shipment case words a window and an arrival in
  the stop location's zone, falling back to the organization's (`subjectRead.at`,
  `StopSnapshot.Timezone`), which is the clock the tools that move a window use; the zone is named
  when it differs from the reader's. Lateness is measured against the window as the record holds
  it now (`againstTheWindowAsItIs`).

- **A remedy remembers that it was taken.** `PreviewRemedy.TakenAt` and `TakenNote` come from the
  record, not the browser: a driver's open portal invitation is read with the preview
  (`drivernotificationservice.openInvitation`), so after a refresh the warning says the invitation
  was sent, when and to whom, and offers "Send it again" (`inviteWorkerToPortal` with
  `replacePending`, which voids the open invitation and issues a new link).
- **Where it is, on a map.** The route exhibit carries each stop's and the truck's coordinates;
  `CaseRoute` draws them on Google Maps (`route-map.tsx`, read on demand) when the organization
  has a key and the server sent the places, and the sketch otherwise, which is also the map's
  loading state.
- **An edit tells the shipment's followers.** `shipmenteffects.EmitUpdatedEvent` records
  `ShipmentUpdated` for a save that left the status alone. Four listeners waited for that event
  and nothing sent it, so moving a stop's window never re-projected its arrival.

### Who owns a case

A case has one owner (`record_case_owners`, one row per record; no row is unowned). Ownership is
the Desk's answer to two people working one thing: the owner decides the case, and everyone else
either does not see it or reads it without deciding.

- **The rule is one function.** `caseownerservice.Sight` is whose cases a person works: their own,
  and those of everyone who has handed them their cases for now (an approval delegation of scope
  `Cases` or `All`, active at this instant). `Sight.All` (the `watchtower:manage` permission, "See
  every case") reads other people's cases without working them. Anything that is not a person
  acting for themselves (an agent run, a job) sees and works every case.
- **Lists hide it in SQL.** `caseownerquery.InSight` is the one predicate: a row about a record
  whose case is someone else's is left out, by one probe of `uq_record_case_owners_record` per row.
  The Watchtower's list and counts (so Needs you, the rail and the situation line), and the
  Decisions queue's proposals and plans, all take it. A new list of work about records takes it
  too, rather than filtering in Go.
- **Single reads and writes go through the gate.** `kindAccess.assertMaySee` refuses a Watchtower
  item about a hidden case; `assertMayWork` refuses dismissing, resolving, assigning or handing
  one off. `recordcaseservice.compose` refuses a hidden case and carries `Ownership` on the case.
  `agentdecisionservice.record` and `agentplanservice.prepare` refuse a decision on a change to a
  record whose case is someone else's (`AssertMayWork` on the proposal's or plan's target), which
  covers the case page, the queue, a conversation's own card and an undo window's commit.
- **Routing.** When a record first gets something open about it (`Projector.project`), and again
  when its case is read while still unowned, `Route` gives the case to the first person the
  record names who may work cases: the shipment's owner, then the manager of its driver's fleet;
  a worker's manager, then their fleet's; an invoice's assigned biller, the customer's collector,
  the customer's default biller; an inbound message's owner (`caseownerrepository.Candidates`, one
  batch). Nobody named means unowned, and the first person to claim it owns it. A new kind of
  record adds its people to `sources` there.
- **Changing hands.** `assignDeskCase` with no owner claims (anyone who works cases, when
  unowned); naming someone, or taking over an owned case, needs `watchtower:assign`.
  `releaseDeskCase` leaves it unowned. Each is audited through `SecurityAuditor`, announced on
  `desk-cases` and `watchtower`, and tells a new owner who is not the actor. The Desk bundle
  **Case lead** grants assign and see-all.
- **Away.** Cover moves no rows. A person works the cases of anyone who has handed them over: by
  an approval delegation of scope `Cases` or `All`, or by marking themselves away on their profile
  with that person named to cover (`users.away_delegate_user_id`, read by
  `caseownerrepository.AwayFor`). The cover sees "Covering for …" on the case, and when the
  delegation ends or the owner is back the cases are the owner's alone again. Routing never gives
  a case to someone away with nobody covering (`Uncovered`): it goes to the next person the
  record names, or stays unowned.
- **The page.** `CaseOwner` in the case's head names the owner and holds Take, Assign and Let go;
  a case the reader does not work turns every decision control off (`ownership.canWork`) and says
  who decides it. Who else is on the case is `CaseViewers` in the top bar (presence scope
  `record:watchtower:<record id>`), a small avatar group that takes the same room for forty
  people as for four.

### A case nobody has worked

A condition the Watchtower raised with no agent on it used to open as a reading and nothing to
do. `DeskCase.handOff` is what the page offers instead (`recordcaseservice.handOffOf`,
`CaseUnworked`): the first open condition that names a subject an agent run can start on, its
kind, whether an agent has it now, and what the last agent on it said if it finished and proposed
nothing.

- **Nothing starts by opening a case.** An agent run costs the organization, and most of what
  the Watchtower raises a person settles faster than a model reads the record, so an agent is
  something the person asks for. **Open the case** in the Watchtower (the button and ↵) carries
  `START_CASE` router state, which only makes an unowned case the reader's (`useClaimOnOpen`).
- **A kind's first move is the person's own.** What an item needs first follows from its kind
  (`watchtowerplay.Of`), and who to ask and how follows from what is on the record, so
  nothing asks a model what to do and nothing depends on an agent being set up, enabled or out
  of shadow. The case offers the move as its first button (`DeskCaseHandOff.play`,
  `casePlayLabel`); it opens `CasePlayDialog`, which reads `watchtowerPlay(itemId)`: who it
  goes to and how, and the words, which the person may change. **Send** calls
  `sendWatchtowerPlay`, and the move goes out under the person's name with their permissions.

  | Kind | Play | Playbook | Who and how, in order |
  |---|---|---|---|
  | Missing proof of delivery | `AskForPod` | `podchase.Playbook` | The driver in Dash; the driver by email; the carrier by email; the driver by text |
  | Missed pickup, missed delivery, ETA at risk, no check call | `AskForStatus` | `statusask.Playbook` | The driver in Dash; the driver by text; the driver by email; the carrier by email |
  | Tender exhausted | `TenderAgain` | `retender.Playbook` | The move is offered down its routing guide again, to the carriers the guide names, in its order |

  The rules that keep it simple and safe:
  - **The server plans, every time.** `watchtowerservice.Plays` plans the play from the record
    on preview and again on send (`Playbook.Plan`). The client sends back only the words; who it
    goes to is closed over by the plan's `Send`, so an address or a phone number never reaches
    the browser, a log or a model.
  - **The person's permissions decide.** A play needs what the page for the same act asks:
    `driver_message` create, `customer_communication` create or `tender` create
    (`PlannedPlay.Needs`, `assertMaySend`), on top of the Watchtower's update and the case
    being theirs to work. A refusal says which access is missing.
  - **One sender per channel.** A driver is reached through `drivermessage.Sender` (Dash with
    a login, else a text to the number on file when the plan includes text messages) and an
    outside party through `requestemail.Sender` (the organization's template for the kind, from
    its billing email profile or else its general one). The agent tools `worker_send_message`
    and `shipment_request_documents` send through the same two, so a person's message and an
    agent's go out identically.
  - **When it cannot be sent, the dialog says why** in words the person can act on: who has no
    Dash login, email or textable number, that no email profile exists, that the recipient is
    on the do-not-email list, that the paperwork is already on file.
  - **The item remembers.** A sent play is noted on the item (`Item.RecordRequest`,
    `watchtower_items.last_request`: who asked, when, and what in a sentence), in its history
    (`EventRequested`) and in the audit log, and the item goes in progress. The case then
    leads with "Asked …" and offers **Ask again**, so the next person does not ask twice
    without knowing.
  - **Rewording is the only model call, and only when asked.** `rewordWatchtowerPlay` has Nova
    rewrite the words in the box to follow the person's guidance (`requestwording.Service`,
    `aitask.TaskRequestWording`, the `RequestWording` usage feature). Nova is given the
    message and the guidance, never the recipient, and nothing is sent: the person reads the
    result and sends it themselves. The dialog offers it only where Nova can run
    (`WatchtowerPlay.canReword`).
- **Hand it to an agent** calls `handOffWatchtowerItem`, and an agent only ever runs because a
  person asked. For a kind with no play the first live agent that subscribes to the item's
  event is started on it directly (`handOffToAgent`), and when none does, the person picks one.
  A kind with a play is never given to an agent unnamed, even when one subscribes: the answer
  is who could take it, and the person picks. An agent switched to shadow is neither picked nor
  offered (`withoutShadow`), and while the organization's agents are paused no agent is put on
  anything (`refusePaused`); a play is not an agent's work and is sent all the same. The event
  is never published again for a hand-off: an event that already happened is not delivered
  twice, which once left the page saying an agent was working when no run had begun. The run is
  recorded on the item, so the case says for itself that an agent has it (`handOff.working`), a
  reload shows the same, and a second hand-off is refused while it runs.
- **An agent that already answered is not run again on the same reading.** An item records
  when what it says last changed (`watchtower_items.described_at`, set by the upsert when the
  description differs). An agent whose run on the item finished, with nothing new said since
  it was handed the item (`agentwork.AnsweredUnchanged`), is refused when named and passed over
  when one is picked; the person may still name a different agent. A failed run answered
  nothing and may be tried again.
- **An item an agent raised to ask for a person is never handed to an agent**
  (`SourceAgentException`): the server refuses it and the page offers no agent (`agentRaised`).
  A second agent reading the same record can only ask the same question.
- **Put away.** A case whose plan ran or was declined, or that an agent looked at and proposed
  nothing on, can be put away while something about its record is still open
  (`putAwayDeskCase(kind, id, until)`, `recordcaseservice.PutAway`). It snoozes every open
  condition about the record until the time chosen, so the case leaves Needs you for everyone;
  nothing is closed, and it is refused while a change waits to be decided. `DeskCase.away` is
  set while every item about the record is snoozed (`awayOf`): the case page reads snoozed
  items too, and shows "Put away until…" with **Bring it back** (`bringBackDeskCase`). It comes
  back at its time (the Watchtower's own wake), or at once when anything new is raised, since a
  new item is not snoozed. The person's conversation on the case is snoozed with it
  (`snoozeCase`), and the rail leaves a snoozed case out unless it waits on them again
  (`railCaseRows`). Accepting the risk is still how a condition is closed for good.
- **The address says what is in front.** Needs you keeps its lens, its filters and the item
  open beside the list in the address with `nuqs` (`NEEDS_ADDRESS` in `desk-needs.tsx`), written
  with `history: "replace"` as the person moves through the list, so a link opens on the same
  item. New Desk state that a link should carry goes in the address the same way.
- **Share** in the top bar (`top/desk-share.tsx`) opens the app's one share dialog on the
  address in front. A case, or the record of the item open in Needs you, can be shared with
  teammates and remembers who it was shared with; a conversation or a list offers its link
  only. See [record-kinds.md](record-kinds.md#sharing-a-record).
- **An item worked in a case stays in the queue and is closed in the case.** The item is what
  the case is made of and what clears it when the problem goes away, so it is never removed.
  `DeskAttentionGroup.ownership` carries the record's case owner (one batched read,
  `caseownerports.Service.Standings`), and `caseWorkOf` (`attention/case-work.ts`) says a row
  is being worked once a person took or was given the case, while an agent an item was handed
  to is still on it, or while a change about the record waits to be approved. An agent that
  finished and proposed nothing, or whose run failed, has handed the item back: the row is the
  queue's again, with what the agent said shown on the item. An owner the
  record supplied on its own (`auto`) has not picked anything up. A worked row says whose case
  it is, and its items lose Snooze, Hand off and Done, by button and by key
  (`useWatchtowerActions({ held })`, `heldBack`), leaving Open the case.
- **The page is told, it does not guess.** `agent_run` and `watchtower` realtime events read the
  case again, so it moves when the run starts, proposes or ends. It polls only while the server
  says something is in motion.
- **What the handed agent proposes is the case's plan** whatever record the change is made on
  (`DecisionsRequest.HandedRuns`): a detention desk put on a load proposes on the detention
  occurrence, and the load's case shows it.
- **What a person can do themselves** is declared per kind of condition on the page. A missing
  proof of delivery offers **Upload the POD** (`CasePodUpload`: the shipment's own upload, filed as
  the organization's POD document type) and **Request the driver's paperwork**, which asks the
  case's agent to draft the message for approval.
- **A case shows what its own conversation proposes.** `recordcaserepository.askedOnTheCase` adds
  the proposals and plans of the reader's conversations about the record to the record's
  decisions, whatever record the change is made on: the message to the driver is the load's plan.
- **A sweep's finding can wake an agent.** A source that finds a condition, rather than causing
  it, has no moment of its own to publish an agent event from. It sets
  `WatchtowerItemInput.EventKind` and `RaisesEvent`, and `Projector.project` publishes the event
  when the item is first raised or reopens, never on a later sweep that finds it still there.
  `PodMissing` raises `shipment.pod_missing` this way; its task frame is
  `chase-a-proof-of-delivery`, and the billing exception template carries the tools for it
  (`worker_send_message`, `shipment_request_documents`). No template listens for it by default:
  an organization adds "Proof of delivery missing" to an agent's events in AI Control, because
  every delivered load without paper would otherwise cost a run.
- Not built: `MissedDelivery` items still carry no agent event. A person hands them off from
  the case.
- An agent woken by a sweep's event, with nobody handing it off, leaves no mark on the item
  (`agent_run_id` is set by a hand-off); what it proposes still reaches the case when the change
  is made on the case's record.

### Conversation titles

A conversation is named from its first question as it opens (`conversation.TitleFrom`), and the
model's name for its first chapter replaces that (`assistantchapterservice.nameThread`,
`ConversationRepository.NameThread`): one short call per conversation, on the Desk chapters
setting and its usage feature. `assistant_threads.title_source` keeps a name a person gave
(`Person`) from being replaced. A case's conversation is no longer titled with the record's label.

### The rail's cases and conversations

The rail lists cases and conversations apart, and nothing moves between the two.

- **One rail, in three views chosen by the place in front** (`railViewOf`). Case work (Needs you,
  a record's case, what Agents handled, Decisions) draws the cases' rail: **Back to chats**, Needs you
  with its lenses, Triage, and the cases by day. Triage draws its own: **Back to chats**, Needs you,
  then Triage's folders (the Briefing, the lanes, kinds waiting, views, sources and rules).
  Everywhere else draws the conversations' rail: Today, **Cases** (the way in, with what waits),
  Triage, Scheduled, Memory, and the chats by day. There is no
  switch to keep in step with the page: the rail follows where the person is. Either way the list
  is shelved by day by the same queries, counts and paging.
- **Every view scrolls in the one scroll area.** The rail's body is a single `ScrollArea`, fading
  into the rail at an edge with more beyond it, and it stays while the view changes: the chats
  and the cases (`RailList`, windowed against the area's viewport) and Triage's places are what
  it holds, not scroll containers of their own. A new view of the rail goes inside it; another
  view is read from its top.
- **A room fills the rail, it does not add a second one.** The rail offers an element
  (`useDeskRailSlotStore`) while Triage is in front and `TriageRoom` portals `TriageFolders` into
  it, so the folders keep the room's state and actions and sit in the one rail a person reads.
  While the rail is out of sight (folded, floating, a sheet on a phone) the room shows its lanes
  as a strip (`LaneStrip`). A new room with places of its own does the same; it never draws a side
  panel beside the rail. Triage's count on the rail is Triage's own Waiting on you
  (`inbox.counts`), the number the room shows.
- **Changing view moves the rail and the page; staying in one does not.** The rail's body and the
  page are each named for the view (`desk-motion.ts`: `dk-rail-<view>`, `dk-page-<view>`), so a
  view transition (`viewTransition` on the rail's links, G chords) finds the same name on both
  sides inside a view and a leaving and an arriving region across views, which `desk-motion.css`
  animates: the chats leave and return on the left, the cases and Triage arrive from and leave to
  the right, and the page fades and rises. `<main>` is a one-cell grid so it can carry the name.
  Reduced motion (the system's or the Desk's) leaves the regions unnamed.
- **Triage's way back is where the person was.** The layout keeps the last page outside Triage
  (`TriageBack`, in memory only), so the rail says **Back to cases** and returns to that case or
  lens when Triage was opened from the cases, and **Back to chats** otherwise.
- **Triage has one address and one door.** `/desk/triage`, opened from the Desk's rail (G then T in
  the Desk). There is no Triage module on the app's rail and no `/inbox` or `/triage` address;
  the Watchtower's message items link straight to it.
- **A click on a conversation opens its page.** A row is a link and nothing takes its click over,
  so it does the same thing from Today as from a conversation. Beside the page is a choice made
  from the row's menu (**Open beside the page**, `showThread`), offered wherever the margin is free.
- **The split is the server's.** `ListThreadsRequest.AboutRecord` and
  `CountThreadShelvesRequest.AboutRecord` (`?aboutRecord=` on the thread list and its shelf counts)
  filter on the thread's subject, so each half's shelves are true without the client taking rows
  out. `deskShelfFilter(key, edges, cases)` asks for one half.
- **Cases** (`railCaseRows` in `rail/open-work.ts`) is one row per record: a record the person
  has asked an agent about, on the shelf its conversation was last touched; and, with no
  conversation yet, a record a plan or a single change waits on (`deskAttention`, the Decisions
  lens), on the shelf it was raised, and what was opened this session, under Today. A row opens
  the case, never a conversation alone, and is named by the record when Needs you, the page or
  the conversation's case summary knows it. What waits says so on its row.
- **A decision is filed under its case's record.** An agent proposes on the thing it works: a
  detention desk on the detention occurrence, not the load. `Projector.caseRecord` files a plan
  or a change about such a record under the record that record's own Watchtower item is about,
  so Needs you groups it with the load and the rail marks the load's case as waiting.
- Every row has a right-click menu (`rail-menus.ts`, the shared context-menu host).

A record has one conversation per person and agent. `StartThread` on the Desk with a subject hands
back the person's live conversation with that agent about the record
(`assistantthreads.caseConversation`, `LatestSubjectThread`, `idx_assistant_threads_user_subject`)
rather than opening another; a hand-off always opens its own.

### Width

The Desk's width setting carries to a case (`--dk-case-w`: 980, 1180 or 1380px) and to the box
under it (`--dk-chat-w`); the case stays centred on the canvas at each.

## The composer

`DeskComposer` (`components/desk-chat/composer/`) is the one box every Desk surface asks from:
a conversation, Today, a record's case and Nova's margin.

- **The box is Tiptap over plain lines** (`composer/editor/desk-editor.tsx`): one paragraph a
  line and nothing else in the document. It holds its own words, so a keystroke is on screen
  before React hears of it; `value` is followed only when it changes from outside. Everything
  drawn over the words (a named record's chip, a slash command's name and its empty slots, the
  placeholder) is a ProseMirror decoration, never a node, so the draft stays the text the
  person typed and `editor-text.ts` maps a caret to an offset into it. Menus read "the draft
  and the caret" through `DeskEditorHandle`, never a document position.
- **Drafts** live in `stores/composer-draft-store.ts`, one a conversation (`draftKey.thread`)
  and one a space (`draftKey.space`: `desk-today`, `assistant-home`, `desk-margin`, and
  `recordSpace(kind, id)`, which a record's case and the margin beside its page share). A
  draft is its words and the records it names. Typing changes memory at once; the disk is
  written 250 ms after typing pauses and when the page is hidden, other tabs are followed
  through the `storage` event, and `clearSessionCache()` forgets them all. A new surface
  passes `space` to `useDeskHomeAsk`; it never keeps a draft in component state.
- **Paste** is cleaned (`cleanPastedText`): one line ending, no invisible characters, a
  spreadsheet's cells two spaces apart, at most one blank line in a row. A clipboard with
  words and a picture of them (a spreadsheet's cells) pastes the words; files alone are
  attached; text past `PASTE_AS_FILE_CHARS` or `PASTE_AS_FILE_LINES` is attached as a text
  file where the message has room for one.
- **Prefetch.** The first time the box takes the focus it reads what a bare @ lists and the
  agents the picker offers. The @ search is a query (`assistant.mentionSearch`), so words
  already answered show at once and only new words wait for typing to settle.
- **Keys.** Enter sends and Shift+Enter breaks a line (or the reverse of the first, with "send
  with ⌘Enter"); ↑ in the empty box brings back what was last said; one Backspace takes a named
  record whole; in a menu the arrows wrap, Home, End, Page Up and Page Down jump, and with
  nothing typed after an @ the left and right arrows walk the kinds. A menu opens above the
  box, or below it where there is no room above (`useMenuSide`).
- **Tests** write into the box through `composer/editor/editor-test-kit.ts` (`writeInBox`,
  `typeInBox`, `textInBox`, `placeholderOf`); `fireEvent.change` does nothing to an editor.
  Keys are still fired at the element.

## Keys

One map in `keybinds.config.ts`. ⌘K is the command palette everywhere, mounted in the Desk too; its
**Conversations** scope is Desk search (`assistant.deskSearch`). ⌘J toggles Nova's margin (in a
conversation it focuses its box). ⌘\ folds the margin, or toggles a conversation's workspace. G then
D, N, W or T goes to Today, Needs you, Needs you on the Watchtower lens, or Triage
(`useGoToChords`). A page's single keys fire only when no earlier handler took the key
(`defaultPrevented`); Needs you listens in the capture phase so a decision is never cleared by E or S.

## Verified outcomes

After an approved change runs, the follow-up turn tells the agent to set a wait for the moment a
change is meant to produce something (a delivery inside a new window, a document arriving), say
when it will look, and speak only if it did not happen (`decisionnotes.verifyOutcome`). The Desk
Bench reschedule scenario holds it (`writes.yaml`).
