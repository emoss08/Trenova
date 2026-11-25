# Agent capabilities: registry, search, scopes and consolidated reads

One agent can do anything the calling person may do; a dedicated agent does the same within a
narrower scope. Permission, plan, extension, scope and taint are decided on the server for every
call; what a turn is shown is a convenience, never the boundary.

## The pieces

| Piece | Where | What it does |
|---|---|---|
| Capability registry | `domain/agentcapability` | Entities (name, area, synonyms, the reader behind `list`/`get`, includable sections, field aliases), per-capability guidance (summary, use this when, use X instead when, synonyms, examples, chains) and `Classify`, which files every registered tool under area, entity, verb, read/write, approval and permission. `agenttoolports.ModelDescription` composes every model-facing description from it. |
| `search_tools` | `agentruntime/toolaccess/search.go`, `app/capabilitysearch` | Ranks the registry (IDF-weighted name, phrase and body terms, synonym expansion, entity and verb intent, the keyword catalog as a second opinion, embeddings when configured, chain expansion) and answers up to 8 candidates with a status: `loaded`, `available`, `not_permitted` (the permission or add-on), `not_in_this_agent` (and who has it in scope). Low confidence comes with rephrasings. The JSON is capped at 1,500 tokens. |
| Agent policy | `agentdefinition.Scope`, `PinnedTools`, `ToolTiers` | Areas and capabilities allowed or denied (denials win; empty is everything), at most 12 pinned capabilities, and approval overrides. A person may only set an override stricter than the tool's default; earned trust (`agent_tool_trust`) still promotes a tool a person keeps approving. |
| Consolidated reads | `querytools/recordquerytools` | `records_query`, `record_get` and `describe_entity` read every entity through its own reader. Dispatch resolves the call to the reader (`recordquerytools.Backing`) and runs scope, extension, plan, permission and the reader's taint policy against it. |
| Read envelope | `recordquerytools.Envelope` | `items`, `total` (with `totalEstimated`), `hasMore`, `nextCursor`, `appliedFilters`, `ignoredFilters`, `truncated` with a reason, `omittedForAccess`, `groups`, `includes`. It forwards taint marks and page cutting from the reader. |
| Eval harness | `agentevalgate/evals/capabilities.yaml`, `capability_current_test.go` | 167 requests (vague, specific, writes, chains, desks, hand-offs, denied, paraphrases) run through the real runtime with a scripted model. |

## The grammar

`records_query(entity, where, or, query, sort, fields, include, countOnly, groupBy, limit, after)`.
`where` conditions all hold; each `or` group holds when any of its conditions does. A condition is
`{field, op, value | values | days}` with `eq ne in notin contains startswith gt gte lt lte between
isnull isnotnull lastndays nextndays today`; a `YYYY-MM-DD` compares as the start (or, for an upper
bound, the end) of that day in the organization's zone. A field is one the entity's list
configuration marks filterable, including paths through its relations (`customer.name`,
`pickupAppointment.scheduledWindowStart`), an alias the registry declares (`pickup.scheduled`,
`equipment.type`), or an entity filter the reader declares (`querytool.WhereExtender`). An unknown
field or operator is refused naming the valid ones. Up to 50 conditions and 10 groups.

Shipments add `stops.city`, `stops.state`, `stops.type`, `stops.locationId`, `driverId`,
`tractorId`, `trailerId`, `carrierId`, `commodityId`, `unassigned`, `late`, `onHold`,
`hold.reasonCode`, `hold.ageDays` and `billingReady` (`shipmentquerytools/shipment_where.go`,
`shipmentpredicate`), and include `stops, moves, assignments, commodities, charges, rating, holds,
comments` from the shipment's own read and `tracking, documents, permits` from theirs.

`countOnly` asks the reader for its total; `groupBy` pages at most 500 rows and says so in
`truncated` when there are more. `include` on a list reads related sections for at most 25 rows,
one query per section for the whole set.

## Adding to it

- A new entity: add it to `entities.go` with its reader, area and synonyms; the reader is then
  hidden from the model and reached through the records tools. A list or get that is a
  projection rather than an entity's records is declared in `projections` with its reason.
  `TestEveryListOrGetReadAnswersInTheEnvelope` holds the line.
- A new capability: register the tool as before; `TestEveryRegisteredToolIsInTheCapabilityRegistry`
  checks it is classified. Add guidance only where the description leaves a near neighbour
  ambiguous, and never against a held-out eval case.
- A new entity filter: implement it on the reader (`WhereExtras`, applied in its `Query`).

## Results

`TRENOVA_CAPABILITY_HARNESS=1 go test -run TestCapabilityHarness ./internal/modules/agent/app/agentevalgate/ -update`
writes `evals/capabilities.results.json`. The baseline was measured on the commit before this
change with the same suite.

| | Top 1 | Top 3 | Vague top 1 | Status right | Denied/hand-off right | Calls per request | Schema tokens per turn | Capability schema tokens per turn | False "not available" | Proposals where a read was expected |
|---|---|---|---|---|---|---|---|---|---|---|
| Before | 41.9% | 64.1% | 27.1% | 72.5% | 0% | 2.49 | 5,157 | 2,937 | 65 | 58 |
| After | 88.6% | 99.4% | 89.8% | 100% | 100% | 1.73 | 3,539 | 1,266 | 1 | 6 |

Schema tokens count every schema sent in full on a turn; capability schema tokens leave out the
conversation tools (`ask_user`, `search_tools`, `withdraw_proposal`, `name_plan` and the like),
which no consolidation touches. `ask_user` alone is about 1,580 tokens.

## Decisions that differ from the brief

- **Autonomy stays earned.** Writes still run at their policy's tier; a tool a person keeps
  approving is promoted by the trust ledger (the owner asked for this). Only a person-set override
  is held to "stricter than the default".
- **Readers stay, unlisted.** The per-entity list and get tools were not deleted: they are the
  readers `records_query` and `record_get` run through, with their tested permission, field
  gating, paging and totals. They are not offered on every turn and not in prompts, templates
  or starter lists, but a model or a script that names one (`shipment_list`,
  `trenova.shipment.list`) reads through it when the agent's scope covers the entity, past the
  same plan and permission gates. Refusing them by name refused work the agent was allowed to do.
- **Artifacts follow the reader.** The Desk draws a records_query result as the entity's table and
  a single record_get result as its card (`agentcapability.ReaderOf`); the table pages and sorts
  by re-running records_query, its sort passed in records_query's own form.
- **Write capabilities keep their names.** They are registry entries now and their names already
  follow the conventions; renaming 425 writes would change nothing a model or a person sees.
- **No `legacy/` directory.** The owner asked to delete rather than move; Git history is the
  reference. Migrations and `deskbench/findings.yaml` keep their historical names.
- **Specialized reads keep their shapes.** Projections (the dispatch board, aging, briefings) are
  declared in the registry and return what their cards draw; the envelope covers every record
  list and get.
- **Schema budget.** The capability schemas fell 57%, not 70%; total per-turn schemas fell 31%
  because `ask_user` and the other conversation tools were left as they are, and each desk's
  pinned capabilities ride on every turn by design. Removing the entity
  list from `records_query` would reach 70% at the cost of a search before most reads; the
  harness shows the trade (`calls per request`) and it was not taken.
- **Names, not ids, for shipment parties.** `customer.name` filters by name; drivers, tractors,
  trailers and carriers filter by id (from `records_query` on worker, tractor, trailer, carrier).
  `status age` and `tender status` beyond the stored `tenderStatus` field are not filters yet.
- **The legacy selection suite.** On the 568 older selection cases, written and tuned against the
  previous keyword ranker, top 3 is 90.8% (the old ranker's top 5 was 100%); on its 80 paraphrases, written to share no word with their tool and served by embeddings in production, keyword-only top 3 is 36.2%. The new suite was
  added because the old one held no short or vague request; both run.

## Known gaps

- The red-team cases (`agentredteam/testdata/cases`) and their cassettes still have the model
  call `inbox_get_inbound_message` directly, which dispatch now refuses; they need rewriting to
  `record_get` (entity `inbound_message`) and re-recording against a live model.
- `analysis_run` is held by every agent its scope does not deny, and loads when called rather
  than riding on every turn.

## Release note

Agents are now policies. Every agent built before this release is switched off and kept as an
archive; set it up again by choosing its scope. Every organization gets a universal agent and
dispatch, billing and safety desks. Run `trenova db reset-agents` once after migrating: it
restores the system agents' scopes and deletes memories, earned trust and rule overrides that
name tools that no longer exist.
