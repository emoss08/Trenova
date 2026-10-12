# Record kinds

A kind of record (`shipment`, `formula_template`, `billing_queue_item`) is
named in many places: where it opens, what a page or an @ mention may name,
which agent subject it is, which @ search tab it sits under, whether the Desk
draws it as a card. Each of those used to be a hand-kept list, and they
drifted. A record link the server did not know made every message asked from
its page fail with `context.entityType: Unknown record type`. An invoice
dispute offered by the @ search was refused when sent.

Each fact now has one owner, and every other list is generated from it or held
to it by the typecheck, a generator, or a test.

## Who owns what

| Fact | Owner | Reaches the other side by |
|---|---|---|
| Which kinds exist, their id prefixes, nouns and guarding resource | `permission/recordid.go` (one kind per resource) and `permission/recordkind.go` (the rest) | `generate-record-kinds.mjs` → `@trenova/shared/types/generated/record-kinds` (`RecordKind`, `RecordKindIDPrefix`) |
| Where a record opens | `client/apps/web/src/config/record-links.ts` | product guide → `services/tms/pkg/productguide/catalog_gen.json` |
| Which kinds the Desk draws as a card | `client/apps/web/src/routes/desk/_components/artifacts/cards/registry.tsx` | product guide (`deskCard` on the record link) |
| The kind of record each agent subject is | `agent/subjectrecord.go` (`subjectRecordKinds`) | `AgentSubjectRecordKind` |
| The kinds the @ search offers and their tabs | `assistantservice/mentions.go` (`mentionKinds`) | `MentionKindTab`, `MentionKindsAskedFor` |

Read from those, never copied:

- **Page context and mentions.** `agent.IsKnownPageEntityType` accepts any kind
  the permission registry knows except `assistant_thread`.
- **Subjects.** Their prefixes and nouns are read from the permission
  registry, and their articles from `stringutils.WithArticle`.
- **Memory.** It recognises a record id by the registry's prefix table.
- **Mention search.** It checks the kind's own resource
  (`RecordKind.Resource()`).
- **Desk tables.** `opensRecords` asks the catalog whether the table's kind
  has a Desk card.
- **Client.** `agentSubjectPath`, the composer's mention tabs, glyphs and
  labels, and `RECORD_LINKS` are typed against the generated kinds.

## Adding a kind of record

1. Declare it on the server. A resource whose records are one kind gets a
   prefix in `recordIDPrefixes` (`recordid.go`). Any other kind gets a
   `Kind…` constant and a `recordKinds` entry (`recordkind.go`). Then add the
   entity that mints it to the minting test in `recordid_test.go`.
2. Run `pnpm --filter @trenova/graphql codegen`, which runs
   `generate-record-kinds.mjs`.
3. If it opens on a page, add it to `record-links.ts`. Then run
   `pnpm --filter @trenova/web guide:generate`.

That is all a page needs: the assistant hears about the record from its own
page with no further change. Each further feature is one entry in that
feature's table:

- **Desk card:** an entry in `cards/registry.tsx`, then regenerate the guide.
  See [desk-artifacts.md](desk-artifacts.md).
- **@ mention:** a `mentionKinds` entry plus its query in
  `conversationrepository/mentions.go`, then regenerate. The composer then
  fails to typecheck until the kind has a glyph, and its tab has a label.
- **Agent subject:** a `Subject…` constant, its `subjectRecordKinds` entry and
  the GraphQL enum value, then regenerate.
- **Memory subject:** a `MemoryRecordKind` named after the kind.

## What catches a mistake

- **Unknown kind in a record link:** `RECORD_LINKS` fails to typecheck, and
  `guide:generate` refuses it.
- **Desk card with no record link:** `guide:generate` refuses it.
- **Unknown kind in a subject or mention entry:** `generate-record-kinds.mjs`
  refuses it.
- **A Go table, or the catalog, naming a kind the registry doesn't know:**
  `recordkinds_test.go` in `agent/domain/agent` and
  `TestMentionKindsAreKindsOfRecordGuardedByTheirResource` fail.
- **Stale generated files:** `codegen:check` and `guide:check` fail in CI.

A new table of kinds belongs to one of the owners above. Key it by
`RecordKind` (client) or `permission.RecordKind` (server), and add it to
`recordkinds_test.go`, rather than writing another list of strings.
