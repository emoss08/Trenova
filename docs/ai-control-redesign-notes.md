# AI control redesign: notes

Where the handoff in `design_handoff_ai_control/` and the code disagree, and what was
done about each. The spec is the README's §8 and PROMPT.md.

## Routes and transports

- **Page path.** The spec calls the page `/agent-control`. It is `/admin/agent-control`,
  with the tab held in the URL; the existing path is kept so links and the product guide
  stay valid.
- **Agent saves are REST.** The spec writes `dryRunAgent(draft: SaveAgentDefinitionRequest, …)`
  as if agents were saved through GraphQL. Agents, providers and extensions are saved
  over REST (`/agent-definitions`, `/ai-providers`); reads are GraphQL. The dry run follows
  the saves: `POST /agent-definitions/dry-run/` takes the same body as a save and answers
  with server-sent events, like the assistant's turn stream.
- **Dry run on one request.** The run is started and read on the same request, so only the
  person who asked can read it, and a reader who leaves cancels it. It runs as
  `AgentDryRunWorkflow` on the chat queue (see "Dry runs" in
  `docs/engineering/agent-runtime.md`). Besides the six outcomes in the spec, a step can be
  `Failed`: bad arguments, a permission the person lacks or a spent budget are not
  "not held", and saying so would send the person to the wrong fix.
- **Versions query names.** `agentVersions(id)` / `restoreAgentVersion` are
  `agentDefinitionVersions(agentId)` and `agentDefinitionVersionDraft(agentId, version)`,
  following the `agentDefinition*` naming of the schema file they live in. The draft query
  never saves, as the spec asks.

## Concurrency (§8.1)

- `ai_providers.version` and the agent control settings' `version` already existed and the
  provider repository already refused a stale save. What was missing was the 409's
  content: who saved in between, when, and what they changed. Audit entries are written
  asynchronously, so they cannot explain a conflict that happened a moment ago. Each save
  now writes the version it became in the same transaction: `agent_definition_versions`
  for agents and `ai_setting_versions` for the agent controls and providers. The 409
  carries `conflict: {version, updatedById, updatedByName, updatedAt, changes[]}` on REST
  and in `extensions.conflict` on GraphQL.
- The agent controls mutation ignored the version the client sent. `AgentControlInput`
  gains `version`; a stale one is refused like any other.

## Shadow report (§8.4)

The spec does not say what "match" means. A write an agent recorded in shadow is set
beside what a person did to the same record in the three days after, from the audit trail:

- **matched**: a person changed the record, and the fields the write named.
- **would reject**: a person changed the record some other way.
- **would fail**: the write failed when it was simulated.
- **unanswered**: nobody touched the record.

The match rate is matched over matched plus would-reject; it stays empty until people have
answered one. `audit_entries` gains an index on `(organization_id, business_unit_id,
resource_id, timestamp)` so the report reads only the records it needs.

## Instruction lint (§8.5)

The capabilities come from the tool registry: the record each tool reads or changes, named
as the permission registry names it, and the operation it performs. A sentence that names a
record and an operation that no held tool performs is a finding, with the tools that would
do it. The only fixed vocabulary is the words a person uses for each operation (update,
change, set…). Sentences that say what not to do ("Never…", "Do not…") are guardrails and
are skipped.

## Tool registry

There are no tenant tool-rule overrides in the code today; the registry's tiers are
per tool, and each agent sets its own. The spec's tool rule editor (§8.8) needs them; this
note is updated when that lands.

## Context providers

The spec's list of context providers is replaced by the real enum (Organization, Clock,
User, Page, Tools, Memory).

## SQLite migrations

The spec asks for SQLite migrations alongside the PostgreSQL ones. Trenova is not using
SQLite going forward, so this work adds none, and the ones added earlier in it were
removed.
