# Agent tool conventions

Every agent tool is named and shaped the same way, so a model that has learned one area of
Trenova reads the next one without relearning it, and the runtime can build its namespace map
from the tools themselves. Every tool moved into these conventions in Phase 3 of
[desk-harness-review.md](../desk-harness-review.md); `TestToolsMoveIntoNamespacesAndNeverOut`
in `agenttoolcatalog` now holds the count of tools outside a namespace at zero, so a new tool
without one fails it, and `TestNamespacedToolsFollowTheConventions` holds every tool to the
rules below.

Trenova is not in production, so a rename carries no alias. A renamed or merged tool is renamed
everywhere at once (see "Renaming a tool" below).

## Names

A tool is `<namespace>_<verb>[_<object>]`, all lower case: `shipment_get`, `shipment_list`,
`shipment_place_hold`, `order_attach_shipments`.

- **Namespace.** One word from `agentports.ToolNamespaces()`, the area of the product the tool
  works in (`shipment`, `order`, `invoice`, `worker`, …). Each namespace has a title and a
  one-line summary; the runtime's namespace map is built from that table and from
  `AgentToolDescriptor.Namespace`, never from a list kept anywhere else. A new namespace is
  added to the table with its summary.
- **Verb.** What the tool does, from the words already in use: `get` reads one record, `list`
  reads many (by words, status or filters), `create`, `update`, `delete`, `cancel`, `record`
  (something that happened), `add` and `remove` (a line on a record), `place` and `release`,
  `assign`, `send`, `share` and `revoke`, `set` (one value), `match`, `rerate`, and so on. A
  tool that files a proposal is named for what it does once approved, not "propose".
- **Object.** What the verb acts on when it is not the namespace's own record:
  `shipment_add_comment`, `shipment_list_permits`, `billing_get_queue_item`. A read of the
  namespace's own records has no object: `shipment_list`, not `shipment_list_shipments`. The
  keyword ranker reads a bare `<namespace>_list` as a list of that record and indexes the
  plural too (`agentsearch.NameTokenSet`). The object is singular even when the tool takes
  several records (`invoice_post` takes `invoiceIds`, `billing_approve_queue_item` takes
  `billingQueueItemIds`): a `get` tool's name in reading order, less its verb, is the kind of
  record its card draws (`get_billing_queue_item` draws a `billing_queue_item`).
- **Qualifiers.** A namespace whose records come in kinds names the words that tell them apart
  (`settlement`: `driver`, `carrier`). A qualifier opening the object reads before the
  namespace, so `settlement_get_driver` reads `get_driver_settlement`, the kind of record it
  reads, `settlement_list_carrier` reads `list_carrier_settlements` and
  `settlement_add_driver_adjustment` reads `add_driver_settlement_adjustment`.
  `agentports.ToolNameFor` reads a name in reading order back to its tool, which is how a
  record link finds the tool that reads it (`ReadRecord`).
- **Records.** A kind of record that keeps its own name in the registry (`bank_receipt`,
  `customer_payment`) is listed in its namespace's `Records`. An object that opens with one
  reads without the namespace: `receivable_get_bank_receipt` reads `get_bank_receipt`,
  `receivable_list_customer_payments` reads `list_customer_payments`, and
  `agentports.ToolNameFor("get_bank_receipt")` finds the tool. Without it a record link, a
  run's subject read (`ReadRecord`) and a card's kind would all name a `receivable_bank_receipt`
  that does not exist. List a record only when the registry names it so; anything else reads
  with the namespace (`receivable_apply_credit_memo` reads `apply_receivable_credit_memo`).
- **No other area's noun.** A name weighs four times its description in keyword ranking, so a
  noun in a name pulls the tool into every question that says it. `billing_transfer` moves
  shipments into the billing queue; named `billing_transfer_shipments`, it came first for
  "which shipments are picking up today". Say the object's area in the description instead.

One tool does one thing to one kind of record. Two tools that differ only by a trailing "s"
(one record and several) become one tool taking a list. Two reads of the same records become
one read with the union of their parameters (`shipment_list` was `list_shipments` and
`search_shipments`). Tools are not merged across permission operations: a tool's policy names
one resource and one operation, so pin and unpin stay two tools.

A write over one record or many is a `common.RecordBatch` around the single-record tool, which
is no longer registered. A call naming one record is the single tool's call, with its preview,
its validation, its target (so the proposal is pinned to that record's version) and its
result; a call naming several previews each through the single tool and runs each through it
on the ids the person kept ("Tools that take one record or many" in
[proposal-previews.md](proposal-previews.md)). A read over one record or many returns the
record in full for one id and a table for several (`invoice_get`, `billing_get_queue_item`).

## Parameters

| Kind | Convention |
| --- | --- |
| Record ids | `<record>Id` or `<record>Ids`, marked with the resource they belong to (`agenttoolschema.RecordIDText`, `OfResource`), with a description that names the tool, the page or the run subject that supplies them. A human key the person may say instead (a PRO number) is a second, optional parameter beside the id (`shipment_get` takes `shipmentId` or `proNumber`). |
| Search text | `query`, never `search`, `q` or `term`. |
| Paging | `limit` with its `minimum` and `maximum` declared and marked `toolschema.Clamped`, so a call past the bound is read as the bound with a note rather than refused. A read over a cursor connection (a repository listing through `dbhelper.CursorList` or `querybuilder.ApplyCursorFilters`, which ignore `Offset`) pages with `after`, the `nextCursor` of the previous page, and returns `hasMore` and `nextCursor`, as the GraphQL list queries page with `after` and `endCursor` (`cursorPageSchema`, `readCursorWindow`). Each cursor is bound to the tool and its other arguments (`cursorScope`), so one reused under a changed query, filter, status or sort is refused, and an `offset` sent to a cursor list is refused rather than ignored. A read over an offset query (a repository that applies `.Offset(SafeOffset())`, or a slice in memory) keeps `offset` and returns `hasMore` and `nextOffset` (`pageSchema`). A tool never takes both. |
| Free text | `reason` for why a change is made (a cancel, a rejection, a void); `note` for text kept on the record (a hold's note); `body` for a message sent to someone. `comment` and `memo` stay where they are the record's own field. Never `notes` or `<x>Reason`. |
| Dates | `agenttoolschema.Date` for a day, `DateTime` for an instant and `LocalDateTime` for a wall-clock time read in the place's or organization's zone. Each declares its format, which the runtime asserts before the tool parses it. |
| Status | A tool narrowing by status takes `status` as a list of the domain's values, all in one call. |
| Filters | The generated list tools take `filters` of `{field, operator, value, values, days}` from the entity's filter catalog; a tool that takes `status` itself leaves it out of `filters`. |

A tool bound to a GraphQL input (`internal/api/toolcontract`) keeps the input's field names, and
a parameter the input names differently says why in the binding.

## Descriptions

- The first sentence says what the tool does, in at most 160 characters; it is all `find_tools`
  and the prompt's catalog show.
- Then when to use it rather than its neighbour, where its ids come from, and what it returns
  that a model would otherwise go and look for.
- Do not restate policy the runtime enforces (who approves, what is held after outside text);
  say what the tool changes and who sees it.
- The keyword ranker indexes a name as written and in reading order, with its search terms
  (`agentsearch.NameTokenSet`), so a request reads "list driver settlements" for
  `settlement_list_driver`. Ties break on the name in reading order (`agenttoolcatalog`), so a
  namespace, which every tool in it shares, never decides which of two equally good answers
  comes first.
- Search terms are the words people use for the tool that its name and description lack. They
  weigh as much as the name, so a term that names another area's records ("customer", "rate")
  pulls the tool into unrelated questions; a page-bound tool's terms name its page.
- Edit a description against traces and `deskbench/findings.yaml`, never against a held-out
  scenario or selection case (see "Labels and splits" in [desk-bench.md](desk-bench.md)).

## Renaming a tool

1. Write the mapping (old name to new name; two old names may map to one) and apply it across
   the repository as whole words: Go, agent templates and starters, seeds, bench agents and
   scenarios, the eval suites, write coverage, tool contract bindings, the client's tool
   titles and fixtures, i18n catalogs (keys and translations together), and docs. Leave history
   alone: old migrations, `findings.yaml`, the harness review's audit.
2. Add a migration that calls `agent_rename_tool(old, new)` for each pair. It renames the tool
   in saved agent definitions (tool lists, disabled lists, tiers), earned trust, tool rule
   overrides and memories, keeps one entry where two tools merged, and quarantines captured
   eval cases that held the old name.
3. Regenerate everything the names reach (`docs/engineering/generated-artifacts.md`): prompt
   snapshots, the catalog snapshot, the safety doc, write coverage, the product guide catalog,
   `task i18n`, and the embedding fixture with the floors and split ledgers after it.
4. Record the tool count and the held-out selection figures before and after the rename in the
   pull request, so a drop is seen and never tuned away.
