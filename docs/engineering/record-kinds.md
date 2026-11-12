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
| Which kinds exist, their id prefixes, nouns and guarding resource | `recordkind/recordid.go` (one kind per resource) and `recordkind/recordkind.go` (the rest) | `generate-record-kinds.mjs` → `@trenova/shared/types/generated/record-kinds` (`RecordKind`, `RecordKindIDPrefix`) |
| Where a record opens | `client/apps/web/src/config/record-links.ts` | product guide → `services/tms/pkg/productguide/catalog_gen.json` |
| Which kinds the Desk draws as a card | `client/apps/web/src/routes/desk/_components/artifacts/cards/registry.tsx` | product guide (`deskCard` on the record link) |
| The kind of record each agent subject is | `agent/subjectrecord.go` (`subjectRecordKinds`) | `AgentSubjectRecordKind` |
| The kinds the @ search offers and their tabs | `assistantservice/assistantthreads/mentions.go` (`mentionKinds`) | `MentionKindTab`, `MentionKindsAskedFor` |

Read from those, never copied:

- **Page context and mentions.** `agentpagecontext.IsKnownPageEntityType` accepts any kind
  the permission registry knows except `assistant_thread`.
- **Subjects.** Their prefixes and nouns are read from the permission
  registry, and their articles from `inflect.WithArticle`.
- **Memory.** It recognises a record id by the registry's prefix table.
- **Mention search.** It checks the kind's own resource
  (`recordkind.Kind.Resource()`).
- **Desk tables.** `opensRecords` asks the catalog whether the table's kind
  has a Desk card.
- **Client.** `agentSubjectPath`, the composer's mention tabs, glyphs and
  labels, and `RECORD_LINKS` are typed against the generated kinds.

## Adding a kind of record

1. Declare it on the server. A resource whose records are one kind gets a
   prefix in `recordIDPrefixes` (`recordid.go`). Any other kind gets a
   `recordkind.Kind` constant and a `recordKinds` entry (`recordkind.go`). Then add the
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
`RecordKind` (client) or `recordkind.Kind` (server), and add it to
`recordkinds_test.go`, rather than writing another list of strings.

## Sharing a record

Sharing is one resource and one dialog for every kind of record. There is no per-kind share
table, endpoint or dialog; an invoice, a shipment and a worker's case are shared the same way.

- **Resource.** `record_shares` (`agent/domain/recordshare`): one row per record and recipient,
  with the sharer, the note, the `path` inside the app the link opens (the tab or the Desk page
  the sharer was on), and a count and dates that move each time it is shared again. `path` is
  validated as an address inside the app (`recordshare.ValidPath`, and a check constraint),
  because the email joins it to the app's own origin.
- **Service.** `recordshareservice` reads the record with the sharer's own access
  (`deskrecords.Labeler`), so nobody shares what they cannot open; offers and accepts only
  teammates whose own role lets them read that kind (`kind.Resource()`), since a share sends
  a link and never grants access; records the shares, audits them on the record's resource,
  then tells each recipient in the app (`record_shared`) and by email. Both messages are
  document templates any organization can reword (`notification.record_shared`,
  `record.share.email`), told what the record is in words (`RecordTitle`), not per kind.
- **API.** GraphQL `recordShares(kind, id)`, `recordShareCandidates(kind, id, query)` and
  `shareRecord(input)`; each checks read on the kind's own resource. Agents share an invoice
  through `invoice_share` and `invoice_list_share_candidates`, which call the same service.
- **Taking a share back.** `unshareRecord(input)` marks the share revoked (`revoked_at`,
  `revoked_by_id`); the row is never deleted, so who could read a case through a share, and
  until when, stays on record. Every read of standing shares filters on `revoked_at IS NULL`:
  the dialog's list, `SharedWith`, and the case's sight (`caseownerquery.InSight`), so the
  recipient stops reading the case at once. Sharing with the same person again clears the mark.
  The sharer may take their own share back; the case's owner, and anyone who may give cases to
  people (`Standing.CanAssign`), may take back any. `RecordShare.removable` tells the dialog which
  rows get **Remove**. It is audited on the record's resource like the share was.
- **Client.** `ShareDialog` (`components/share/share-dialog.tsx`) takes the record
  (`kind`, `id`, a title) and the `path` to open. Use it for any new share button. With no
  record it offers the link alone, for a page that is no one record.

- **A share brings the record's case into view.** Desk cases have one owner, and other
  people's cases are hidden from a reader who does not see every case. A record shared with
  someone is the exception: its case is in their sight to read and ask about, and its owner
  still decides it. `caseowner.Sight.Viewer` carries the reader, `caseownerquery.InSight` lets
  a row through when a share for them exists (one probe of `uq_record_shares_recipient`), and
  `caseownerservice.Standing`/`Standings` set `CanSee` the same way, with `CanWork` unchanged.
  This is the only thing a share adds, and it adds nothing to what a role permits: the
  recipient must still be able to read that kind of record.

A kind becomes shareable by being a record kind: nothing else is registered.

