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
| `FromSimulation(rec, simulation)` | A tool that simulates rather than previews, read as a `Partial` preview. |
| `Chain(steps)` | A plan's projection (below). |

Options: `WithRefs(path → resource)` (ids shown by the record's label, resolved later),
`Ignore`, `Volatile` (shown, left out of the digest), `Only` (for a create), `Labels`,
`Types`, `SensitiveAs`, `WithRegistry`.

The engine drops bookkeeping (`createdAt`, `updatedAt`, `version`, tenant ids), relation
objects and bare ids, types each value with the artifact display classifier
(`assistantartifact/display_classify.go`, shared with artifacts), labels it, stamps its
sensitivity from the permission registry and **drops a Confidential value where it is
built**, so none reaches a baseline or a recorded preview.

Until a tool previews itself, `toolsimulation.Simulate` and the preview service read its
`ToolSimulator` as a `Partial` preview, and any other tool as `Unavailable`.

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
moved since. Nothing about it can fail the proposal.

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
only the digest it was sent, when it is one. A digest mismatch on an approval (of a
proposal or a plan) is the only `ConflictError` the decide path returns; a preview whose
changes fail validation is a validation error. A batch (`decideAgentProposals`) takes one
digest per proposal: a mismatch fails that proposal alone, and a proposal without one is
approved unreviewed. Withheld parts do not block an approval; the count is recorded in the
decision's preview.

A plan (`agentplanservice.Decide`) previews its steps once, refuses a stale plan and a plan
digest that no longer matches, and hands each step the preview it was approved on
(`StepPreview`, `StepPreviewReviewed`, server-only fields). A later step on a record an
earlier step changed runs against the version that step left
(`ExpectedTargetVersion`, from `DecisionOutcome.ExecutedTargetVersion`); a record moved by
anything else still stops the plan.

An approver's changes can never point the write at another record: `refuseRetarget`
compares `Target(proposed)` with `Target(merged)` in `CheckModifications` and again where it
runs. The parameter holding the target's id is `readOnly` in `parameterFields` and in the
chat's editable fields (`services.ProposalFields`).

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

- **Coverage follows the tools.** A tool shows as much as its `Preview` says; one without
  shows its parameters. The contract test that fails for an action tool with no previewer
  lands with the tool previewers.
- **Labels cover the records previews name most.** A record of a resource the labeler has
  no table for is shown without a name; people (`user`) are never labelled, since a user
  is not scoped to one tenant.
- **A digest is per reader.** Two people with different access see different digests for
  the same proposal; each approves what they saw.
- **Calls outside the database are guarded by tests only.** The read-only transaction stops
  a preview writing to the database, not sending mail or starting a workflow.
- **Field labels are English**, humanized from the record's keys, as artifacts' are.
