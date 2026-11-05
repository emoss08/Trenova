# Global search

The command palette's search (and the `GET /search/` it calls) finds records across a tenant:
shipments, customers, workers, documents and invoices. This page is how a record reaches the
search, how the search decides what a person may see, and the steps to add another kind.

## How a record reaches the search

```
Postgres table  ──(GTC, change data capture)──►  Meilisearch index  ──►  globalsearchservice  ──►  palette
```

- **GTC** (github.com/emoss08/gtc, the `gtc` container) tails the `gtc_publication` logical
  replication publication and writes each row it is routed into a Meilisearch index.
  The routing is `deploy/gtc/sinks.yaml`: one entry per table, naming the index, its
  `primary_key` (always `id`; every Trenova row has tenant columns, so Meilisearch cannot infer
  one), its `searchable_attributes` and its `filterable_attributes` (always
  `organization_id` and `business_unit_id`, which every search filters on). No Go code writes
  these indexes.
- **GTC sends the whole row.** Index a table whose every column may sit in the search index.
  A table holding text that should not (an invoice's email snapshots, remittance text, memo and
  address) is indexed through a **projection table**: only the columns search reads, kept equal
  to the record. `document_search_projections` is written by the document services and leaves
  out files attached to a conversation, because search has no per-person check to protect them;
  `invoice_search_projections` is kept by a trigger on `invoices`
  (`20261231023134_invoice_search_projections.sql`), which covers every writer. The trigger is
  `AFTER INSERT OR UPDATE OF` the projected columns, so a write touching none of them never runs
  it, and its upsert writes only when a value changed, so GTC is not sent every touch. A generated
  column in that list (`balance_due_minor`) fires it when a column it is computed from is updated. Never store a value
  that changes with the clock (days past due): it would rewrite, and re-index, every row daily.
  Read it from a stable column (the due date) instead.
- **The publication.** Locally GTC creates a `FOR ALL TABLES` publication on first start, so a new
  table is published already. A self-hosted deploy scopes it to the sink file's tables
  (`deploy/selfhost/scripts/db-bootstrap.sh`, run on every start).

## How the search decides what a person sees

`globalsearchservice.Service.Search` (`internal/modules/operations/app/globalsearchservice`):

1. **Kinds.** Each kind is an `entitySearchDefinition` (`definitions()` in `service.go`): the
   `search.EntityType`, the index (from `search.meilisearch.indexes` in config), the
   `permission.Resource` that guards it, and the index attributes a hit reads. A request naming
   a kind no definition has is refused as invalid. One `CheckBatch` asks which resources the
   person may read; a kind they may not read is never searched and returns no group. The query
   is cut to its first 200 characters (Meilisearch reads only its first ten words).
2. **Rows.** A kind whose resource the database scopes (a team or own data scope) has its hits
   checked against the database before they are shown (`restrictToVisible` in `visibility.go`);
   a check that fails hides them. A narrowed group asks the index for four times the limit
   (`fetchLimit`, at most 100), so the records the person may see are not crowded out by the
   ones they may not, is cut back to the limit once the hidden ones are gone, and reports only
   the visible hits it found.
3. **Hits.** `mapHit` turns an index document into a `GlobalSearchHit` (`<kind>Hit`): the title,
   the subtitle, the `href` from the record link (`recordHref`, which reads the product guide's
   record paths), and metadata. Every metadata key must be classified in `hitMetadata`
   (`enrich.go`): `own()` for a value of the hit's own record, `linkedField(kind, …)` for a
   linked record's field shown through its reference view, `figureOf(kind)` for a figure about a
   linked record that needs that kind's read. An unclassified key is dropped from every response,
   and `TestEveryHitMetadataKeyIsClassified` fails. A person who reads a kind sees every field of
   its records (field ceilings apply only to a record reached through another, and to agents),
   so `own()` needs no field check; a field in a protected group (`protectedfield`, such as a
   worker's licence number) must never be an `own()` value.

The client never decides access, but it never offers what the server will not return either:
the palette's tabs and `@` filters show only the kinds the person may read
(`useReadableRecordTypes`, from `PALETTE_SEARCH_RESOURCES`), and while the permissions load
their place is held by placeholders. The records a person opened recently are kept in memory
only (`stores/recent-records-store.ts`), cleared when the session ends (`clearSessionCache`),
and offered only for kinds the person may still read. Previewing a record warms its queries but
never a document's page: the server logs each drawn page as the document being viewed.

## Adding a kind of record to the search

Server (`services/tms`):

1. **What to index.** If every column of the table may sit in the index, index the table. If not,
   add a projection table holding only the searched and displayed columns, kept by a trigger
   on the record (one-row copy; see "A column copied from one related row" in
   [backend/data-access.md](backend/data-access.md)) and backfilled in its migration, with a
   cascading foreign key to the record. Never index text a search result does not show.
2. **The kind.** Add `EntityType<Kind>` to `shared/types/search/entity.go`. The definition
   (step 4) is what makes the search accept it.
3. **The index name.** Add a field to `MeilisearchIndexConfig` and its default in
   `internal/infrastructure/config/searchconfig/search.go`, and the `indexes:` entry in
   `config/config.yaml`, `config/config.example.yaml` and `deploy/selfhost/config/config.yaml`.
4. **The definition.** Add an `entitySearchDefinition` (kind, label, index, guarding resource,
   attributes) to `definitions()`, a `<kind>Hit` to `mapHit`, and the hit's metadata keys to
   `hitMetadata`. Amounts in minor units come back from Meilisearch as floats (`minorValue`).
   Add the index to `indexConfig()` in the package's tests, and a test that finds the kind and
   one that finds none without its read (`invoice_test.go`).
5. **The record link.** The hit's `href` needs the kind in `client/apps/web/src/config/record-links.ts`
   (then `pnpm --filter @trenova/web guide:generate`); a hit with no `href` is dropped.

GTC (`deploy/gtc/sinks.yaml`):

6. Add a `public.<table>:` entry under `meilisearch.tables` with `index`, `primary_key: id`,
   `searchable_attributes` (what a person types) and `filterable_attributes`
   (`organization_id` and `business_unit_id`, and nothing the service does not filter on).

Client (`client/apps/web/src`):

7. `globalSearchEntityTypes` in `services/global-search.ts`. The typecheck then asks for:
   - the kind's icon in `PALETTE_ENTITY_ICONS`, its guarding resource in
     `PALETTE_SEARCH_RESOURCES` and its words in `paletteEntityText`
     (`components/command-palette/palette-entities.ts`), and its place in
     `PALETTE_RECORD_TYPES`;
   - a case in every switch over `PaletteRecord.entityType`: `recordStatus` (`palette-status.ts`),
     `recordFields` (`palette-fields.ts`), `RecordContext` (`palette-row.tsx`), the actions
     (`palette-actions.ts`), `recordSub` (`record-link.tsx`), `SourceRecord`
     (`source-records.tsx`), the preview body (`palette-preview.tsx`) and its prefetch
     (`preview-queries.ts`).
8. Its `@` aliases in `searchEntityOptions` (`search-entity-filter.ts`), and the page that lists
   the kind in `PAGE_RECORD_TYPES` (`route-command-data.ts`).
9. Run `task i18n` for the new words.

Bring it up locally:

10. `task db-migrate` (the projection and its backfill), then `task db-stamp`.
11. `docker compose up -d --force-recreate --no-deps gtc` in `dev/`. GTC reads the sink file only
    at start; the log shows `meilisearch index ready … index=<name>`. Not `restart`: the sink file
    is a single-file bind mount, and an edit that writes a new file (`sed -i`, most editors)
    leaves the mount on the old one, so the restarted container cannot start (exit 127,
    `error mounting … /config/sinks.yaml: no such file or directory`).
12. Backfill the table into its index: `POST localhost:8095/backfill {"table":"public.<table>"}`.
    Progress is in `GET localhost:8095/api/stats`, the count in Meilisearch's
    `GET localhost:7700/indexes/<name>/stats` (`Authorization: Bearer` the
    `MEILI_MASTER_KEY` from `dev/compose.yml`). Rows written before the restart are not streamed
    until the backfill runs.

## Gotchas

- GTC's sink file has no `sortable_attributes`, so a kind browsed with nothing typed comes back
  in Meilisearch's own order, not newest first. Sorting needs that setting added to GTC
  (`internal/infrastructure/config/sinks.go` in github.com/emoss08/gtc) and a `Sort` on
  `customerports.SearchRequest`.

- Search returning empty groups with a 401 in the API log means `search.meilisearch.apiKey` in
  `config/config.yaml` does not match `MEILI_MASTER_KEY` in `dev/compose.yml`.
- Never run the GTC binary with a flag inside its container (`docker exec gtc /app/gateway
  --help`): it ignores flags and starts a second instance.
- A record kind a Nova answer may cite but the palette does not search (no index) can still be
  previewed: add it to `PALETTE_ENTITY_ICONS` only, with its preview, and leave it out of
  `globalSearchEntityTypes`.
