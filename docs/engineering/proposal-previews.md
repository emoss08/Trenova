# Previewing a proposal before it is approved

A person approving an agent's write sees what the write would do to each record, computed
from the world as it is now and filtered for what they may see, and approves exactly that:
the approval carries the digest of the preview they were shown, and a preview that has
changed since is refused. This page is how that works and what it promises.

Read [agent-runtime.md](agent-runtime.md) first; the filing baseline rides its dispatch
activity and must keep to its determinism rules.

## The model

`internal/core/domain/agent/preview.go`.

| Type | Is |
|---|---|
| `ToolPreview` | What a tool says its write would do: a summary, one `RecordChange` per record, warnings, `Partial`. No reader, digest or staleness. |
| `ProposalPreview` | What the service serves and records: the tool's preview for one reader, with `Coverage`, `Staleness`, `TargetVersion`, `WithheldCount`, `Recorded`, `ComputedAt` and `Digest`. |
| `RecordChange` | One record: resource, id, label, record link, `Operation` (`Create`, `Update`, `Delete`, `Archive`, `Send`, `Run`), version, `Withheld`, `DependsOnStep`, fields, and an optional `MessagePreview` and `MoneyPreview`. |
| `PreviewFieldChange` | One value: path, label, display type, `Before`/`After` (and refs to the records they name), sensitivity, `Withheld`, `Volatile`, `Truncated`, `ChangedSinceProposed` with `ProposedBefore`, `ProjectedFromStep`. |
| `MessagePreview` | What would be sent, rendered by the code that sends it, with the recipients it would reach: channel (`Email`, `SMS`, `Dash`, `EDI`, `Comment`), from, to, cc, bcc, attachments, subject, body, visibility, cadence, template version. |
| `MoneyPreview` | Amounts before and after, line by line, with totals and delta, and the sensitivity of the least visible figure. |
| `PlanPreview` | A plan's steps in order, each a `ProposalPreview`, with one digest over the step digests. |

`Coverage` is `Full` (the tool previewed its whole write), `Partial` (part of it, a
simulation read as a preview, or cut to bounds) or `Unavailable` (the tool cannot say; the
parameters it would run with are shown as one `Run` change).

**Bounds.** At most 20 records, 60 fields a record, 2 KiB a value, 16 KiB a message body
(`Bounded()`); a cut value or body is flagged, and cut records and fields are counted. A
baseline over 32 KiB is not kept. `ToolPreview.Simulation()` is the preview as the runtime's
simulation reads it, capped at 8 KiB.

**Warning codes** the client translates (`Message` is the English fallback): `would_fail`,
`already_told_customer`, `driver_unreachable`, `depends_on_step`, `target_changed`,
`record_missing`, `tool_removed`, `preview_failed`, `withheld`, `sensitive_content`,
`retarget_refused`, `unpinned`.

**Reasons.** A `would_fail` warning also carries `Reasons []PreviewReason{Field, Label,
Message, Param}`: `toolpreview.WouldFail(err)` splits a validation `MultiError` into one reason
a field (a refusal of the whole write is a reason with no field), and
`toolpreview.LocateReasons` walks the tool's parameter schema to name the parameter each field
rides in (`bol` is `shipment.bol` for `create_shipment`). The preview service locates reasons
for every tool, including a failing `ToolValidator`. GraphQL serves them as
`AgentPreviewWarning.reasons`. The client lists each reason ("BOL: already in use by shipment
SEED-DET-009"), falling back to `Message` without its leading sentence, never the bare "would
not go through" when the server said why. It offers "Change {label}", which opens the editor
on that parameter (a nested one such as `shipment.bol` gets an input of its own bound to the
value inside the JSON field), and "Ask the agent to fix it", which in a conversation's
approval box opens "Tell the agent instead" with the reasons written (the rejection carries
them as its note, and the follow-up asks the agent for a corrected proposal), and on the
Desk and in AI Control opens the rejection with the reasons written. Approve stays offered
and warned.

## How a tool previews

A write tool implements `ToolPreviewer` (`ports/services/agenttool.go`):

```go
Preview(ctx context.Context, params ToolExecuteParams) (*agent.ToolPreview, error)
```

The contract: read-only, deterministic given the database and the parameters, and it decides
what changes with **the same code `Execute` does**. A diff built by laying parameters over a
record would be plausible and wrong — parameter names are not fields, transitions set derived
fields, services compute totals — so each tool shares one plan function between `Execute` and
`Preview`.

`internal/core/services/toolpreview` builds the preview from that plan function:

| Function | Builds |
|---|---|
| `Update[T](rec, before, mutate, opts...)` | A change to an existing record: `before` is copied by way of JSON, `mutate` (the tool's plan function) applied to the copy, and the two diffed. What `mutate` reads must be carried by the record's JSON tags. |
| `Archive[T]` | `Update` for a write that retires the record. |
| `Changed[T](rec, before, after, opts...)` | The diff of two states the tool computed itself. |
| `Create[T](rec, created, opts...)` / `Delete[T](rec, before, opts...)` | A record made or removed. |
| `Send(rec, *MessagePreview)` | A message the write would send. |
| `MoneyBlock(currency, lines...)`, `Money(rec, block, opts...)`, `AttachMoney(change, block, opts...)` | Amounts with totals and delta; the block takes the highest sensitivity of the fields named by `SensitiveAs`, and a Confidential one is dropped. |
| `Build(summary, changes...)`, `Warn(preview, code, message, args...)` | The preview, bounded. |
| `Describe(tool, params)`, `Parameters(tool, resource, params)` | The `Unavailable` fallback: the parameters, never the owner a self-scoped call records, stamped with their sensitivity on the tool's resource. |
| `Chain(steps)` | A plan's projection (below). |

Options: `WithRefs(path → resource)` (ids shown by the record's label, resolved later),
`Ignore`, `Volatile` (shown, left out of the digest), `Only` (for a create), `Labels`,
`Types`, `SensitiveAs`, `WithRegistry`.

The engine drops bookkeeping (`createdAt`, `updatedAt`, `version`, tenant ids), relation
objects and bare ids, types each value with the artifact display classifier
(`assistantartifact/display_classify.go`, shared with artifacts), labels it, stamps its
sensitivity from the permission registry and **drops a Confidential value where it is
built**, so none reaches a baseline or a recorded preview.

Every write tool previews itself; `TestEveryActionToolPreviewsWhatItWouldDo` fails for one that does
not. A tool that does not preview is shown as `Unavailable`: its parameters.

An agent in simulation records what its write would have done from the same snapshot.
The runtime reads the baseline it has just taken (`toolsimulation.FromBaseline`). A
preview that failed there is reported as failed and **never run again outside the
snapshot**, because the failure may be the snapshot refusing a write. The executor, on an
approved proposal for an agent in simulation, runs the preview in a read-only,
repeatable-read transaction of its own (`toolsimulation.InSnapshot`).

## The preview service

`internal/core/services/proposalpreviewservice`, behind `services.ProposalPreviewService`.

`ForProposal(ctx, {Proposal, Modifications, Viewer})`, in order:

1. Refuses a reader outside the proposal's tenant (`ErrTenantMismatch`).
2. A decided proposal returns the preview recorded when it was decided, filtered again for
   this reader (`Recorded: true`); without one, its parameters.
3. A self-scoped tool's proposal is previewed only for its owner.
4. Settles the parameters: changes are checked the way a decision checks them
   (`CheckModifications`, which refuses a change pointing the write at another record);
   without changes, a failing `ToolValidator` becomes a `would_fail` warning.
5. Reads, in one **read-only, repeatable-read** transaction: the pinned target's version,
   then the tool's preview (limited to 3 s; a timeout, error or panic falls back to
   `Unavailable` with `preview_failed`), then the labels of every record the preview names
   (`RecordLabeler`, `recordlabelrepository`, one query per resource). A write the preview
   attempted fails in that transaction.
6. Staleness: the pinned version against the one read (`target_changed`, or
   `record_missing` when the record is gone). A targeted tool whose proposal was never
   pinned gets `unpinned`. Each value is compared with the filing baseline and marked
   `ChangedSinceProposed` with its `ProposedBefore` ("was 40,000 when proposed").
7. The reader filter — tools read without regard to who is asking, so this is the only
   guard: a record of a resource the reader may not read is withheld whole with its label; a
   value above their ceiling is withheld; a referenced record's label is withheld when they
   may not read its resource; an amount above their ceiling is withheld. Everything withheld
   is counted (`WithheldCount`, `withheld` warning). The ceiling and read checks are
   per-request caches (`fieldsensitivity.Ceilings`, `fieldsensitivity.ReadAccess`), made
   for the reader as an actor in the GraphQL layer and in a decision alike, so the digest a
   person is shown is the one their approval is checked against.
8. The digest: SHA-256 of the canonical JSON (`shared/jsonutils/canonical.go`: sorted keys,
   numbers as exact decimals) of the proposal id, tool, parameters as they would run,
   coverage, withheld count, target version and the filtered changes without volatile
   values. `ComputedAt`, warnings and staleness are left out. It names what **this** reader
   was shown.

`ForPlan` previews each pending step the same way, compares each with its baseline, then
**chains** them (`toolpreview.Chain`): a step on a record an earlier step also changes
`DependsOnStep` it, starts each shared value from what that step leaves
(`ProjectedFromStep`) and carries `depends_on_step`. Then each is filtered and digested; the
plan's digest covers the step digests in order and the plan is stale when any step is.

`ForDecision` is a decision's recorded preview filtered for a later reader.

## The filing baseline

When the runtime holds a write for a person, the dispatch activity pins its target and
previews it in one snapshot (`Baseline`, limited to 5 s) and keeps the preview in
`agent_proposal_baselines`, keyed by the proposal id the activity already mints. The
baseline is unfiltered but for Confidential values and is read only to tell which values
moved since. A baseline that cannot be taken never fails the proposal.

**A write that would be refused is not filed.** When the baseline preview carries a
`would_fail` warning (`ToolPreview.Refusal()`), the activity files nothing and tells the model
the write was not proposed, each reason with its parameter (`PreviewWarning.ReasonLines()`),
and to ask the person for the value or look one up and propose again, with the same `invalid`
trace outcome a `ToolValidator` refusal has. A person could only ever reject that card. The
baseline service keeps no row for a refused preview unless `ProposalBaselineRequest.FileRefused`
says the write is filed anyway, which the runtime sets for a write on a record an earlier
unexecuted write of the same turn changes: the two become steps of one plan, the earlier step
may be what makes the later one valid, and the plan's preview says it depends on that step.

**The model is told what the card holds.** The result of a filing carries the baseline
preview's summary and each warning's reason lines, and says that is all the model may say the
proposal holds. A tool with no preview (or a baseline that could not be taken) echoes the filed
arguments instead, as canonical JSON (`jsonutils.CanonicalMarshal`) up to 2 KiB; past that,
one line per argument, lists and long text counted rather than repeated. An argument the
runtime renamed to the tool's parameter (`aliasArguments`) is named, so the model uses the
right name next time. The echo is text in the tool activity's result, so it took no gate.

**It never rides `PendingAction`.** The action is an activity result that
`DispatchCall.ProposedSoFar` hands to every later tool activity of the turn, so a preview
there would grow the history by the square of the proposals. `PendingAction` is unchanged:
no workflow command, no payload change, no `GetVersion` gate, and the recorded histories
under `agentjobs/testdata/replay*` and `assistantjobs/testdata/replay` replay unchanged.
Evaluations keep no baseline; a write in simulation reads its simulation from the same
preview. A retried activity mints a new id and leaves an orphan, which
`ExpireStaleProposalsActivity` purges once it is 48 hours old.

## Decisions

`agentdecisionservice.DecideWithOutcome` settles the preview after the modifications and
before anything is written, for `Accepted` and `Modified`:

- the write is previewed as the decider sees it, with their changes;
- **stale → refused**, nothing recorded ("reject it, or ask the agent again");
- a `previewDigest` that no longer matches → **`ConflictError`**, nothing recorded; the
  client refetches the preview and says it changed;
- otherwise the decision records `preview`, `preview_digest`, `preview_target_version` and
  `preview_reviewed` (the decision named the digest).

A rejection is never refused over its preview — a person can always say no — and records
only the digest it was sent, when it is one.

Every decide input takes an optional `note` (at most 2000 characters): what the decider tells
the agent, such as why they turned the change down. It is stored on the decision
(`agent_decisions.note`; a plan's note on each step's decision) and served on
`AgentDecision.note` and on the thread's proposals as `decisionNote`. The conversation's
follow-up turn reads it fenced in `<untrusted_data>` as "The person declined:" (or "The
person's note with the decision:" on an approval), capped, and is told it is the person's
words about the decision, never instructions; after a declined note the agent may propose a
different change, never the same one again. It is not part of the decision's hashed audit
form. A digest mismatch on an approval (of a
proposal or a plan) is the only `ConflictError` the decide path returns; a preview whose
changes fail validation is a validation error. A batch (`decideAgentProposals`) takes one
digest per proposal: a mismatch fails that proposal alone, and a proposal without one is
approved unreviewed. Withheld parts do not block an approval; the count is recorded in the
decision's preview.

A plan's steps may be filed by several agents of one turn (the conversation's agent and
an agent it handed a task to); each step keeps the run of the agent that filed it, and
the plan is refused while any of those agents is behind a shadow switch. A plan
(`agentplanservice.Decide`) previews its steps once, refuses a stale plan and a plan
digest that no longer matches, and hands each step the preview it was approved on
(`StepPreview`, `StepPreviewReviewed`, server-only fields). A later step on a record an
earlier step changed runs against the version that step left
(`ExpectedTargetVersion`, from `DecisionOutcome.ExecutedTargetVersion`); a record moved by
anything else still stops the plan. A plan whose steps each target a different record of one
resource (five `post_invoice` steps) has no step resting on another, so every step runs,
each settles on its own, and the plan records the steps that completed and, as its failure,
each step that did not with its reason; a plan with a shared record, several resources or
an untargeted step keeps the stop at the first failure.

`decideMyProposals` is `decideAgentProposals` for proposals raised in the caller's own
conversations (`AgentDecisionQueueService.DecideManyOwn`): every id passes
`AssertOwnProposal` or nothing is decided, and each carries the digest of the preview the
person had read. The approval box keeps a batch's rows mounted behind its folded **Details**
(`keepDetailsMounted`), so the first five (`BATCH_OPEN_LIMIT`) read their previews as the box
appears and an approval made without opening **Details** sends their digests: a batch is
reviewed by default. A row past the first five reads its preview only once it is opened, and
the box says how many were not previewed before such an approval records them unreviewed.

An approver's changes can never point the write at another record: `refuseRetarget`
compares `Target(proposed)` with `Target(merged)` in `CheckModifications` and again where it
runs. The parameter holding the target's id is `readOnly` in `parameterFields` and in the
chat's editable fields (`services.ProposalFields`).

## Record subsets

A write over a set of records can let the person approving it drop some. A tool marks an
array-of-ids parameter as a subset of one permission resource's records with
`toolschema.RecordSubset(resource, property)`, which sets the `x-subsetOf` keyword;
`transfer_to_billing.shipmentIds` is one, over `shipment`.

- **The form.** `toolschema.Fields` reads the parameter as kind `RecordSubset` with its
  `Resource`, and GraphQL serves both on `AgentProposal.parameterFields`:
  `AgentProposalField.kind` is `RecordSubset` and `AgentProposalField.resource` names the
  resource (`null` for every other kind). `AgentProposalField.choices` lists every record
  the parameter proposed, from the proposal's own parameters (never a modification), in the
  order proposed and each once, bounded by the parameter's `maxItems` and never past
  `toolschema.MaxSubsetChoices` (5000): `{ id, label }`, the label read through
  `RecordLabeler` and the id when the record is gone or the reader may not read the
  resource. A client lists them as rows a person can untick and sends the ids it kept, as
  a list of strings, back as the parameter's modification.
- **Labels.** GraphQL reads them through the `SubsetLabels` loader: each subset field asks
  once for all of its ids, and a request's fields are labelled together, one query per
  resource for each `services.MaxRecordLabelsPerResource` (5000) records. The chat's
  `/assistant/threads/{id}/proposals/` labels every pending proposal in a thread in one
  `Labels` read. Both check the reader's read permission on the resource first and read
  nothing without it.
- **The rule.** `CheckModifications` (and `admit`, where the write runs) refuses a
  modification to a subset parameter that is not a narrowing of what was proposed
  (`toolschema.CheckSubsets`): an id the agent did not propose is refused as `forbidden`, and
  an empty list, or one that is not a list, is refused. Removing ids is allowed, and the
  preview, the digest and the write all follow the narrowed list.
- **The model never sees the keyword.** Every model adapter sends tools through
  `toolschema.ForModel`, which strips `x-` keywords at any depth (a strict endpoint refuses
  a keyword it does not know) and keeps parameter names that merely look like one. The
  same goes for `x-enumOf`, the source an enum's values were taken from
  (`agenttoolschema.Enum`).

- **Selecting by criteria.** A subset tool that also takes criteria (`transfer_to_billing`'s
  `allTransferable` with the candidates filters) implements `ToolSelectionResolver`. The
  dispatch activity resolves the call to concrete ids after the argument contract and
  before the tier, validation, baseline and filing, so `PendingAction.Arguments` holds the
  ids, never the criteria: the card, the preview digest and the choices a person may untick
  pin exactly those records, and execution never evaluates the criteria again. The model's
  own call stays in the thread as it sent it, and the ledger key is taken from it. A
  selection that cannot be resolved is refused to the model with the reason.

  `retry_accounting_sync` takes `errorCategories` as criteria beside its `syncRecordIds`
  subset, and pins them only when the call is filed for a person: it implements
  `ToolProposalSelectionResolver`, which the dispatch activity asks once the tier is decided
  and only when that tier is not Automatic. A proposed call naming categories alone becomes
  the Blocked, DeadLettered and Retrying records that last failed that way then, at most 50,
  with the categories kept beside them to say why they were chosen; the categories only
  narrow those ids when the approval runs, never add to them, so a record that fails after
  the proposal is left for another retry. The preview says how many other records that
  failed that way it leaves and is then `Partial`. An approval that names no records (one
  filed before pinning) is refused and asked to be proposed again. An automatic retry has no
  approval to keep faith with, so it still reads its categories when it runs, with no cap.

  The other record-subset tools keep ids only: `attach_order_shipments` (its records come
  from `search_shipments`, a general search, and the order's own rules decide membership),
  `release_accounting_sync` (a person-only send to the books from a general list; releasing
  "everything awaiting approval" unseen is what the tool exists to prevent),
  `set_carrier_monitoring` (billed per carrier from `list_carriers`, a general list), and
  `retry_edi_message_delivery` and `reprocess_edi_inbound_files` (general lists of messages
  and files, capped at `ediservice.MaxBulkEDIActionItems`), and `post_invoices`,
  `send_invoices`, `approve_billing_queue_items` and the four settlement approve and post
  twins (up to 50, each record previewed and run through its single-record tool; see
  [Bulk twins](#bulk-twins-of-single-record-tools)).
  None of them has a candidates read that decides each record the way the write would.

A preview still shows at most 20 records, so a subset of more is shown in part; the
parameter's value is the whole list, and the field's choices list it all. The approval form
shows each preview record's outcome on its row and the rest by label.

### Bulk twins of single-record tools

Posting five drafts used to be five `get_invoice` calls and five `post_invoice` proposals the
person approved one by one, and a pay period's settlements one approval and one posting
proposal each. Seven person-only billing and settlement steps now each have a bulk twin that
takes up to 50 records as a record subset, so the person approving sees one card, may untick
records, and approves exactly the set that remains:

| Bulk tool | Parameter | Runs each record as |
| --- | --- | --- |
| `approve_billing_queue_items` | `billingQueueItemIds` (billing queue), `reviewNotes` | `approve_billing_queue_item` |
| `assign_billing_queue_billers` | `billingQueueItemIds` (billing queue), `billerId` (optional: the person asking) | `assign_billing_queue_biller` |
| `transition_items_to_in_review` | `billingQueueItemIds` (billing queue), `billerId` (optional: the person asking) | `transition_item_to_in_review` |
| `post_invoices` | `invoiceIds` (invoice) | `post_invoice` |
| `send_invoices` | `invoiceIds` (invoice) | `send_invoice` |
| `approve_driver_settlements` | `settlementIds` (driver settlement) | `approve_driver_settlement` |
| `post_driver_settlements` | `settlementIds` (driver settlement) | `post_driver_settlement` |
| `approve_carrier_settlements` | `settlementIds` (carrier settlement) | `approve_carrier_settlement` |
| `post_carrier_settlements` | `settlementIds` (carrier settlement) | `post_carrier_settlement` |

Each bulk tool wraps its single-record tool (`agenttoolservice/bulk_records.go`) and copies
its policy: the same class, permission and floor. A twin of a person-only tool runs only from a
person's approval; a twin of a tool that may run on its own (`assign_billing_queue_billers`,
`transition_items_to_in_review`) runs wherever its single would, and the review twin is held
to what its most guarded item allows, so one held item among ten waiting ones still puts the
call before a person. The preview runs the single tool's
own preview for each of the first 20 records and keeps that record's change, with a `What
happens` field carrying the single preview's sentence, refusals first; a larger set is
`Partial` and says the rest are checked when it runs. A record the single tool would refuse,
one already posted, or one not found in the caller's tenant is shown as refused and the rest
still go; only a set of which nothing would go carries `would_fail`, which is also what
`Validate` refuses. Execution runs only from a person's approval, as the approver, record by
record through the single tool (so every single-record guard, including
`ApprovedFromProposal`, still applies), on the approved ids only, and reports "N of M posted;
refused: …". A run in which every record was refused fails. The single tools stay, pinned to
their record, and their descriptions point to the bulk tool when there is more than one
record.

The settlement twins (`agenttoolservice/settlement_bulk_tools.go`) build their single tool
from the same lifecycle text and ledger, so each settlement is planned and performed by
`PlanAction` and `Perform` exactly as the single approve or post would, including the
journal entry a posting books and, for a driver, the driver-visible egress that classifies
each call as money. The record-subset choices and a refused settlement in the outcome are
named by settlement number. The settlements clerk holds the four twins in place of the four
singles (still 56 tools): a twin takes one settlement as well as fifty, so the clerk loses
nothing, and the singles stay registered for agents an organization builds.

`get_invoices` reads up to 50 invoices by id in one call (`InvoiceRepository.GetByIDs`,
tenant-scoped, amounts gated as `get_invoice` gates them) and names ids that are not an
invoice of the organization. `list_invoices` and `list_billing_queue_items` also filter on
`id` with `in`.

Two billing-queue steps that are not money also have twins, because eleven items waiting on a
biller were eleven proposals: `assign_billing_queue_billers` and `transition_items_to_in_review`.
Both take `billerId` as optional and, when it is absent, assign **the person asking**: "assign
me", "start reviewing these" and "get these ready to post" name nobody else, and an item in
review must carry a biller (`CheckStatusFields`), which is what used to fail a plan at step 1
after the person had approved it. `transition_item_to_in_review` now assigns that biller as it
moves an item with nobody on it, validates the move it previews, and pins its target so a plan
of such steps runs each on its own record. An unattended agent is nobody's biller, so a call
from one that names none is refused with the parameter it needs. `get_billing_queue_items`
reads up to 50 items in one call with what blocks each (approvable, missing documents,
validation failures, detention holds), as `get_invoices` does for drafts, so checking a queue
is one read and one table rather than a card per item.

The billing assistant template holds the twins in place of `assign_billing_queue_biller` and
`transition_item_to_in_review` (a twin takes one item as well as fifty), plus `get_billing_queue_items`,
and gave up `preview_report` (the report analyst's) to stay under the 64-tool cap with room for
an organization's own. To make room it gave up `list_insights`,
`get_insight` (the insight analyst's) and `get_report_run` (a run's progress is already on
screen). A template is copied when an agent is made, so an existing billing agent gains the
bulk tools only when an administrator adds them in AI control.

## API

`internal/api/graphql/schema/agentpreview.graphqls`:

| Operation | Needs |
|---|---|
| `agentProposalPreview(id, modifications)`, `agentPlanPreview(id)` | `agent_proposal:read` |
| `myProposalPreview(id, modifications)`, `myPlanPreview(id)` | `assistant:read` and the proposal or plan raised in one of the caller's conversations (`AssertOwnProposal`, `AssertOwnPlan`, the latter with `MayUseAgent`) |
| `AgentDecision.preview` | the decision's reader; filtered again for them |
| `AgentDecision.previewDigest`, `previewReviewed` | columns |

Previews are root queries rather than a field of `AgentProposal`: a per-row computation on a
list would break the dataloader rule, and the arguments differ per call. The inputs
`AgentProposalDecisionInput.previewDigest`, `AgentPlanDecisionInput.previewDigest` and
`DecideAgentProposalsInput.previewDigests` carry the digests back. All of it is additive.

## Audit, tracing and metrics

The AI audit trail's `ProposalDecided` row carries `result_summary` "Reviewed preview
sha256:…" or "Preview not reviewed; sha256:…" and `version_before` from
`preview_target_version`: fields the hashed canonical form already has, so no `hash_version`
change ([ai-audit-trail.md](ai-audit-trail.md)).

Each preview is a `trenova.ai.preview {tool}` span with its purpose (`read`, `decide`,
`baseline`), coverage, staleness, record and withheld counts and `error.type`, and no value.
The decide span carries `trenova.ai.preview.digest` and `.reviewed`
([ai-tracing.md](ai-tracing.md)). `trenova_ai_proposal_preview_total{tool,coverage}`,
`trenova_ai_proposal_preview_duration_seconds{tool}` and
`trenova_ai_proposal_preview_conflicts_total{tool}`; a rising conflict rate is how an
unstable digest shows itself.

## Where the code is

| Piece | Path (under `services/tms`) |
|---|---|
| Model, bounds, digest | `internal/core/domain/agent/preview.go`, `proposalbaseline.go` |
| Ports | `internal/core/ports/services/{agenttool.go,proposalpreview.go}`, `ports/repositories/agentproposalbaseline.go` |
| Builder | `internal/core/services/toolpreview/` |
| Service | `internal/core/services/proposalpreviewservice/` |
| Labels, baselines | `internal/infrastructure/postgres/repositories/{recordlabelrepository,agentproposalbaselinerepository}/` |
| Filing | `internal/core/services/agentruntime/baseline.go` |
| Decisions | `agentdecisionservice/preview.go`, `agentplanservice/preview.go`, `proposalexecutor` |
| Migration | `20261231006790_agent_proposal_previews` (Postgres and the SQLite mirror) |

## Known limits

- **Coverage follows the tools.** A tool shows as much as its `Preview` says.
  `TestEveryActionToolPreviewsWhatItWouldDo` (`agenttoolpolicy/preview_contract_test.go`)
  holds every registered action tool to a previewer, with no exemptions, so a new action
  tool without one fails CI.
- **Labels cover the records previews name most.** A record of a resource the labeler has
  no table for is shown without a name; people (`user`) are never labelled, since a user
  is not scoped to one tenant.
- **A digest is per reader.** Two people with different access see different digests for
  the same proposal; each approves what they saw.
- **Calls outside the database are guarded by the read-only mark and by tests.** The
  read-only transaction stops a preview writing to the database. `WithTx` also marks a
  read-only transaction's context (`ports.IsReadOnly`), and after-commit callbacks queued
  in it are dropped, since nothing was committed. Work a service hands off outside its
  transaction must check the mark. Distance resolution does: it counts no stored-mileage
  hit and buffers no stored-mileage candidate, though it still asks PC*Miler, as the
  billing panel's rate preview does. The mark does not stop mail or a workflow start a
  preview calls directly; the previewers' tests do.
- **Field labels are English**, humanized from the record's keys, as artifacts' are.
