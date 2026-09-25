# AI Audit Trail

The AI audit trail is an append-only record of what every agent did, decided, or had decided
for it. It covers each run or turn starting and ending, each model call, each tool call and
refusal, each proposal being filed, decided, executed or expiring, and each task handed to
another agent. Each tenant's rows form a hash chain, signed with a key held outside the
database. A row that is changed, removed or reordered is found the next time the chain is
verified. Compliance reads it in AI Control and exports it as CSV or JSON.

The trail adds no writes to the agent runtime. A projector reads what the runtime already
records and derives the trail from it: `agent_runs`, `assistant_turns`, `ai_usage_records`,
`agent_run_steps`, `agent_run_events`, `agent_proposals` and `agent_decisions`. It is the
trail's only writer. How those source rows get their trace and owner links is described in
[ai-tracing.md](ai-tracing.md).

## Tables

Migration `20261231006750_ai_audit_ledger` (Postgres and the SQLite mirror) creates:

| Table | Holds |
|-------|-------|
| `ai_audit_events` | The trail. One row per event, with `seq`, `prev_hash`, `hash`, `hash_key_id` and `hash_version`. `source_key` is unique per tenant, and so is `seq`. |
| `ai_audit_chain_heads` | One row per tenant: the last `seq` and hash, and what the last verification found. The projector locks it to assign the next `seq`. |
| `ai_audit_seals` | One row per projector batch: `from_seq`..`to_seq` and the hash of the row the batch ends on. Retention only cuts at a seal. |
| `ai_audit_exports` | Export requests and their files. |
| `ai_audit_projector_state` | One watermark per source table. |

On Postgres, a trigger (`ai_audit_events_append_only`) refuses every `UPDATE` and `TRUNCATE`
on `ai_audit_events`. It refuses `DELETE` unless the session has set
`trenova.ai_audit_prune = on`, which only the retention sweep does, or the delete cascades
from a deleted organization. The trigger stops accidents and ordinary application code. It
does not stop a database superuser, and neither does anything else inside the database. The
chain is what shows that such a change happened. SQLite has no trigger.

## Events

Each source row stands for one or more events. Each event has a `source_key` that names its
source row and moment, so projecting the same row twice writes nothing the second time.

| Source | Events | Source key |
|--------|--------|------------|
| `agent_runs` | `RunStarted` when started, `RunEnded` (`Completed` / `Failed`) when finished, with the masked error and summary and the taint | `run:{id}:started`, `run:{id}:ended` |
| `assistant_turns` | `RunStarted`, `RunEnded` (`Completed` / `Refused` / `Stopped` / `Failed`). The principal is the turn's user and the agent comes from the thread | `turn:{id}:started`, `turn:{id}:ended` |
| `ai_usage_records` | `ModelCall`: provider, model, tokens, cost, latency, attempt, failover and the masked error. The owner comes from the row's owner columns. Older rows fall back to `run_id`, then to the turn that was running in the thread at that moment, and are marked `reconstructed`. Evaluation calls get the purpose `Evaluation` | `usage:{id}` |
| `agent_run_steps` (tool steps) | Settled: `ToolCall` with outcome `Ran` / `Proposed` / `Simulated` / `Failed` / `Denied`, plus tier, what held it, egress, tier source, proposal, record versions and redacted arguments. Still `Started` when its run or turn ended: `ToolCall` with outcome `Unknown` | `step:{owner}:{step_key}` |
| `agent_run_events` | `tool_finished` failed with no step for that call: `ToolRefused`. `delegate_started` / `delegate_finished`: `DelegationStarted` / `DelegationEnded` (`Completed` / `Exhausted` / `Refused` / `Declined` / `Stopped` / `Failed`) | `event:{id}` |
| `agent_proposals` | `ProposalFiled`. `ProposalExecuted` / `ProposalExecutionFailed` / `ProposalSimulated`, with the principal set to the executing user, or the agent for auto-execution. `ProposalExpired`, with the principal set to the system | `proposal:{id}:filed`, `…:executed`, `…:expired` |
| `agent_decisions` (proposal decisions) | `ProposalDecided` (`Accepted` / `Modified` / `Rejected`): who decided, and the redacted modifications | `decision:{id}` |

An owner whose id carries the evaluation prefix is an evaluation replay. Its events get the
purpose `Evaluation`, so a reader can keep them apart from live work.

### What is kept of arguments

Tool arguments go through `aiauditservice.Redactor` before they are hashed:

- A field the tool's policy marks confidential is replaced with `[confidential]` and is never
  recorded.
- Values that `auditservice` treats as sensitive are masked, by field name and by pattern
  inside free text.
- Parameters the runtime writes itself are dropped, such as the owner a self-scoped call is
  about.
- The map is capped at 16 KiB. Values that do not fit become `[truncated]`, and the row sets
  `argumentsTruncated`.
- `argument_sensitivity` records the resource the tool acts on and the sensitivity of each
  path.

When a row is read, `ReaderArguments` replaces with `[withheld]` every path above the
reader's field-sensitivity ceiling on that resource. The ceiling comes from
`fieldsensitivity.PersonCeiling`, which the agent query tools also use. The recorded value is
never changed, so the hash still verifies.

## The projector

`ProjectAIAuditWorkflow` runs on the audit queue on the `ai-audit-projector` schedule. The
schedule fires every `aiAudit.projector.interval` (default one minute, minimum ten seconds)
and skips a run while the previous one is still going. Each pass:

1. Reads every source forward from its watermark in `(timestamp, id)` order. It reads up to
   `aiAudit.projector.batchSize` rows per page (default 1000) and at most ten pages per
   source, so a backfill moves forward a step each pass. It stops ten seconds behind now, so
   rows still committing are left for the next pass.
2. Reads the last 15 minutes behind the watermark again. This catches rows whose timestamp
   was taken long before their transaction committed. Rows already on the trail are skipped
   by their source key before anything else is loaded.
3. Derives each tenant's rows. Then, under the chain head's row lock, it assigns `seq`,
   links and signs each row, inserts with `ON CONFLICT DO NOTHING`, checks that every row
   landed, advances the head (optimistically on `last_seq`), and writes one seal for the
   batch.
4. Saves the watermarks, but only if every tenant succeeded. When one tenant fails, the pass
   fails, and the next pass reads the same rows again. The other tenants' rows are skipped by
   their source keys.

`trenova_ai_audit_projector_lag_seconds{source}` is how far each watermark is behind. Other
metrics: `trenova_ai_audit_events_projected_total`, `…_projector_passes_total`,
`…_verifications_total`, `…_exports_total` and `…_events_pruned_total`.

A source table's own retention sweep must not delete rows the trail has not read yet.
`AIAuditService.SourcePruneHorizon(source)` is the watermark minus the 15-minute overlap, and
it is the latest point a source sweep may delete before. No source sweep calls it yet. The
first one to delete agent rows must.

## The chain

Each row's hash covers its content and its link to the row before it:

```
message = "trenova.ai-audit/v1\n" + hash_version + "\n" + hash_key_id + "\n" + prev_hash + "\n"
          + canonical JSON of the row
hash    = hex(HMAC-SHA256(key[hash_key_id], message))   hash_version 1, signed
        = hex(SHA-256(message))                         hash_version 2, unsigned
```

The canonical JSON always uses the same field order (every content column except `hash`,
`hash_key_id`, `hash_version` and the database's `created_at`), sorts map keys, and writes
every number as its exact decimal. Cost is written with six decimal places. A value read
back from `jsonb` therefore hashes exactly as it did before it was stored. A tenant's first
row links to `GenesisHash`, which is 64 zeros.

Keys come from `aiAudit.chain.keys` and `aiAudit.chain.activeKeyId`. In production they come
from `TRENOVA_AI_AUDIT_CHAIN_KEYS="id:secret,id:secret"` and
`TRENOVA_AI_AUDIT_CHAIN_ACTIVE_KEY_ID`, never from the database. Each secret is at least 32
characters. Production and staging refuse placeholder secrets. New rows are signed with the
active key. Every row names its key, so rows signed before a rotation still verify as long
as their key stays configured. **Rotating means adding a key and making it active. Never
remove a key while rows signed with it are retained.**

### Unsigned mode

When no key is configured, the trail is still written, but as a plain SHA-256 chain:
`hash_key_id` is empty and `hash_version` is 2. The process logs a warning once, and the
chain status reports `signed: false`. An unsigned chain still detects a changed or removed
row. It does not detect someone who can write the database and recomputes every hash after
the change. When a key is configured later, new rows are signed and the older unsigned rows
still verify.

### Verification

`VerifyAIAuditChainWorkflow` (the `ai-audit-verify` schedule, daily at 03:37 UTC, or
`verifyAIAuditChain` on demand for one tenant) walks each tenant's chain from its oldest
retained row. It checks that:

- no `seq` is missing;
- each row names the hash of the row before it;
- each hash recomputes under the key it names;
- each seal matches the row it ends on;
- the head names the last row.

The oldest retained row must link to the genesis hash, or, after a prune, to the seal the
prune cut at. The result is stored on the chain head, and `aiAuditChainStatus` returns it:

| Status | Meaning |
|--------|---------|
| `Verified` | Every link holds, up to `lastVerifiedSeq`. |
| `Mismatch` | A link fails at `failedSeq`, and `detail` says which check. It is written to the audit log as critical, and everyone who holds `ai_audit_trail:read` is notified (`ai_audit_chain_mismatch`), at most once a week for the same failure. |
| `KeyMissing` | A row names a key that is no longer configured. Put the key back. |

The all-tenant run (`VerifyAIAuditTenantsWorkflow`) continues as new every 200 tenants.

## Retention

`PruneAIAuditWorkflow` (the `ai-audit-retention-purge` schedule, daily at 04:13 UTC) deletes
each tenant's rows older than the organization's AI audit retention. The retention is set in
data retention settings. The default is 2555 days (seven years) and the minimum is 365. The
sweep cuts only at the end of a seal, so the oldest row left still links to a seal. Seals are
never deleted.

## Exports

`requestAIAuditExport` takes a format (`CSV` or `JSON`), a time range of at most seven years,
and the same query, field filters, filter groups and sort as the table, with at most 20
filters. When the export is requested, it fixes `snapshot_seq` to the head's current
`last_seq`. Rows projected afterwards are not in the file, even when the file is written
later.

- If the range holds more than `aiAudit.export.maxRows` rows (default 1,000,000), the request
  is refused with a validation error.
- If it holds at most `aiAudit.export.syncMaxRows` rows (default 5000), the file is written
  during the request.
- Otherwise `AIAuditExportWorkflow` writes it on the report queue, using the report upload
  pipe (`storage/uploadpipe`). When the file is ready, the requester is notified
  (`ai_audit_export_ready` / `ai_audit_export_failed`).

The file is stored at `ai-audit-exports/{org}/{id}.{csv|json}`, and the export record keeps
its SHA-256. Only the person who requested an export can download it
(`aiAuditExportDownload`). The download link is presigned for 60 seconds. The request and
every download are written to the audit log as critical export operations. Files are deleted
after `aiAudit.export.ttl` (default seven days) by the `ai-audit-export-cleanup` schedule.

The CSV has a fixed column contract. Every column is always present, in this order:

```
seq, id, occurred_at, occurred_at_utc, recorded_at, kind, outcome, purpose, principal_type,
principal_id, on_behalf_of_user_id, on_behalf_of_user_name, decided_by_user_id,
decided_by_user_name, agent_definition_id, agent_definition_version, agent_name, owner_kind,
owner_id, run_id, turn_id, thread_id, proposal_id, plan_id, decision_id, step_key, call_id,
delegate_call_id, parent_owner_id, trace_id, span_id, provider_id, provider_kind, model,
attempt, failover, input_tokens, output_tokens, reasoning_tokens, cache_read_tokens,
cache_write_tokens, cost_usd, latency_ms, tool_name, tool_effect, egress_class, tier,
tier_source, held_by, reason, arguments, arguments_withheld, redacted_paths,
arguments_truncated, result_summary, entity_type, entity_id, version_before, version_after,
window_start, window_end, tainted, taint, external_content, simulated, reconstructed,
source_key, audit_entry_ids, prev_hash, hash, hash_key_id, hash_version
```

Every cell goes through `csvutils.SafeCell`. A value that starts with `=`, `+`, `-`, `@`, a
tab or a carriage return gets a leading `'`, so a spreadsheet does not run it as a formula.

The JSON file is one object:

```json
{
  "export": {
    "format": "trenova.ai-audit/v1",
    "id": "…", "tenant": { … }, "generatedAt": 0, "generatedBy": { … },
    "range": { "from": 0, "to": 0 }, "filters": { … },
    "chain": { "keyId": "…", "signed": true, "hashAlgorithm": "HMAC-SHA256",
               "firstSeq": 0, "lastSeq": 0, "snapshotSeq": 0, "complete": true },
    "columns": [ … ]
  },
  "events": [ … ]
}
```

`chain.complete` (also `chainComplete` on the export) is true only when nothing was filtered
out and every `seq` from the first row to the last is in the file. Only then can an auditor
holding the key check the file's chain on their own.

Arguments in an export are cut to the requester's own sensitivity ceiling, the same way as
when they are read in the app. `audit_entry_ids` is filled only when the requester can also
read the audit log.

## Linking to the audit log

The audit log has no column that names an AI event, and it does not need one.
`AIAuditEvent.auditEntries` finds the audit log rows written:

- for the same record (a decision is matched against its proposal);
- by the same principal (the acting user, or the agent principal);
- inside the event's time window, with one second of slack at each end.

Only tool calls that ran or failed, executions and decisions have a match. The lookup is
batched per page through a dataloader, and it returns nothing to a reader without
`audit_log:read`.

## GraphQL and permissions

The resource is `ai_audit_trail`, with the operations `read` and `export`. Reading an
agent's runs does not grant it, and agents can never hold it.

| Operation | Needs |
|-----------|-------|
| `aiAuditEvents`, `aiAuditEvent`, `aiAuditChainStatus`, `aiAuditExports`, `aiAuditExport`, `verifyAIAuditChain` | `ai_audit_trail:read` |
| `requestAIAuditExport`, `aiAuditExportDownload` | `ai_audit_trail:export` |
| `AIAuditEvent.auditEntries` | additionally `audit_log:read` (empty otherwise) |

`aiAuditEvents` and `aiAuditExports` are cursor connections that take the data table's
query, filters and sort. They run their `COUNT` only when `totalCount` is selected. The client
operations are in `client/packages/graphql/src/operations/agent/audit.graphql`:
`AIAuditEventTable`, `AIAuditEventDetail`, `AIAuditChainStatus`, `AIAuditExportTable`,
`AIAuditExportDetail`, `RequestAIAuditExport`, `AIAuditExportDownload` and
`VerifyAIAuditChain`.

## Where the code is

| Piece | Path (under `services/tms`) |
|-------|------------------------------|
| Domain, canonical form, hashing | `internal/core/domain/aiaudit/` |
| Ports | `internal/core/ports/repositories/aiaudit.go`, `internal/core/ports/services/aiaudit.go` |
| Repositories | `internal/infrastructure/postgres/repositories/aiauditrepository/` |
| Projector, derivation, redaction, verification, retention, exports | `internal/core/services/aiauditservice/` |
| Workflows and schedules | `internal/core/temporaljobs/aiauditjobs/` |
| GraphQL | `internal/api/graphql/schema/aiaudit.graphqls`, `resolver/aiaudit.resolvers.go`, `loaders/aiauditentriesloader.go` |
| Field sensitivity | `internal/core/services/fieldsensitivity/` |
