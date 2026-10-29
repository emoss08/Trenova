# Rust opportunities in Trenova

As of 2026-10-20. Which parts of the codebase would materially improve as Rust, and which would not.
The implementation brief for the first port is in [imaging-worker-prompt.md](imaging-worker-prompt.md).

## 1. Summary

Rust pays off in Trenova where three kinds of work meet: byte-heavy pipelines (documents, images,
spreadsheets), logic that has to give the same answer in the browser and on the server, and code that
parses or runs untrusted input. Most of the Go backend is not that kind of work. Shipments, billing,
GraphQL, permissions, the agent runtime and the Temporal jobs spend their time on Postgres, Redis, model
providers and outside APIs, so a faster language would change little. The documented build-time pain
(`docs/engineering/graphql-executor.md`, `module-architecture.md`) is Go compile time, and Rust compiles
more slowly, not faster.

The strongest candidates, in order:

1. **A document and imaging worker** that replaces the PDFium-in-WASM pool, the pure-Go pixel loops, the
   QR decoder, and retires the legacy OCR path, which cannot run in the shipped image.
2. **The capture companion uploading what it already computes**, so the server stops re-rendering every
   scanned page.
3. **One shared calculation crate** (formula language, money allocation, lane and matrix rules, report
   rounding) compiled for both the browser and the server. The duplicated TypeScript copies already
   give different answers.
4. **The row-level security check as a native Postgres extension** (pgrx), after measuring it.
5. **Search inside Postgres** with `pg_search`, which removes Meilisearch and puts search under the same
   row-level security as every other read.

The surveys also turned up defects that have nothing to do with language choice. They are tracked
separately and outrank any port.

### How this was done

Ten read-only surveys covered the documents and imaging pipeline, compute engines, EDI and protocols,
client tooling, the agent module, Postgres and row-level security, identity and security, domain
calculations, reporting and templates, and the code outside the API (Dash, the capture companion,
simulators, CLI, self-host deployment). The key code paths were re-read by hand. Nothing was
benchmarked. Every timing and memory figure without a source is an estimate, and section 6 lists what to
measure before committing to a port.

## 2. How Rust would fit into this codebase

The repository already has most of what a Rust addition needs:

- **Rust is already here.** `desktop/capture` is a Cargo workspace of 13 crates and binaries (about 36k
  lines, edition 2024, toolchain pinned in `rust-toolchain.toml`, workspace lints in `Cargo.toml`) with
  its own CI (`.github/workflows/native-capture.yml`). New crates can copy its conventions.
- **The Go binary already runs WASM in-process.** PDFium (`klippa-app/go-pdfium/webassembly`) and libwebp
  (`gen2brain/webp`) run under `tetratelabs/wazero` while the build stays `CGO_ENABLED=0`
  (`deploy/Dockerfile.tms:17`). A Rust crate compiled to `wasm32-wasip1` loads the same way. Two wazero
  features matter here: modules are memory-isolated from the Go heap, and
  `RuntimeConfig.WithCloseOnContextDone(true)` actually stops a running module when its context ends.
- **Sidecars are an accepted pattern.** Gotenberg (Chromium) and docling-serve (Python) already run as
  HTTP services next to the TMS. A Rust service can run the same way when it needs native libraries.
- **The Postgres image is built from source.** `deploy/Dockerfile.postgres` compiles pg_cron and
  pgvector, so a pgrx extension is one more build stage.
- **The web toolchain is already Rust.** oxlint, oxfmt, Rolldown, the React Compiler through
  `oxc-transform-react` (which took the build from about six minutes to about thirty seconds per
  `docs/engineering/web-build.md`), and the Go-native TypeScript 7 compiler.

So there are four ways in, from least to most disruptive:

| Seam | Use it for | What it keeps | What it costs |
| --- | --- | --- | --- |
| Rust → `wasm32-wasip1` run by wazero inside the Go process | Pure computation: parsers, rule engines, encoders | `CGO_ENABLED=0`, one binary, cancellation | Slower than native; a boundary call per item is expensive, so APIs must be batched |
| Rust → `wasm32` in the browser | Logic the browser and server must agree on | One source of truth | Bundle weight against the 590 KB gzip first-load budget (`client/apps/web/bundle-budget.json`); load it lazily |
| Rust sidecar service | Work that needs native libraries (PDFium, SIMD) | Isolation, its own memory limit | Another container, an internal API, plaintext crossing a local socket |
| pgrx extension in Postgres | Functions on every statement's path | Native speed inside the database | Rules out managed Postgres (RDS, Cloud SQL); native code inside the security boundary |

## 3. Candidates, ranked

Each candidate gives what is there today, what Rust changes, what it costs, and what to fix in Go first.

### R1. Document and imaging worker (done)

The imaging worker (`services/imaging`, `docs/engineering/imaging-worker.md`) does every
document and image computation; the TMS calls it through `kernelports.Imaging`.

**Today.** Nearly every document operation goes through four PDFium WASM instances with 1,024 MB of
linear memory each (`internal/infrastructure/config/config.go:1736-1762`). The pool is warmed in the
background because compiling the module is slow (`pdfrender/pdfreader/reader.go:51`). Capture
inspection, thumbnails, the page viewer, legacy extraction and the docling pre-check all queue on it.

- **The page viewer renders on every request.** `GetPageImage` (`modules/documents/app/documentservice/service.go:806-850`)
  downloads and decrypts the whole document, renders one page at 150 DPI, resizes it with Lanczos and
  encodes WebP. Caching is ETag only, and the Desk prefetches the next page.
- **Legacy OCR churns codecs and cannot run in the shipped image.** A scanned page is rendered,
  PNG-encoded, PNG-decoded, passed through four `disintegration/imaging` filters, thresholded by a
  per-pixel `img.At()` loop (`documentanalysis/contentbuilding.go:103-125`), PNG-encoded again,
  written to a temp file, and handed to a `tesseract` subprocess (`activities.go:912-1029`). The TMS
  image does not include tesseract (`docs/self-hosting/README.md:398`).
- **QR decoding runs on full pages.** `gozxing` with `TRY_HARDER` over a 150 DPI page builds a full
  luminance copy (`infrastructure/captureimaging/inspector.go:162-184`). This is likely the most
  expensive pure-Go step per captured page.
- **Text profiling opens every PDF a second time.** Before docling runs, `pdfTextProfile`
  (`conversion.go:674-699`) opens each PDF in PDFium and reads every page's text only to mark which
  pages have native text.

**With Rust.**

- Native PDFium through `pdfium-render`, with no 1 GB WASM heaps and no copy out of linear memory.
- SIMD resizing (`fast_image_resize`) and threshold and luma loops over contiguous buffers.
- QR decoding with `rxing` or `rqrr`.
- Decode once and reuse, replacing the PNG round trips.
- No OCR in the worker. Docling already runs OCR (RapidOCR, or a vision model on the GPU profile), so
  the legacy extraction engine and its Tesseract path are deleted rather than ported.

Run it as a sidecar ("imaging worker") that takes plaintext bytes over a local API and returns
thumbnails, page images, page-text maps and QR results. A lighter first step is to move only the pure
pixel work (resize, threshold, QR, JPEG encode) into a wazero module and keep PDFium where it is.

**What improves.** Peak memory per job, throughput on the page viewer and capture batches (up to 1,000
pages each), one extraction engine instead of two, and one shared imaging library with the desktop
crate (R2).

**Costs.** A new service and its API. PDFium has to be linked or shipped as a shared library. The
encryption envelope stays in Go, so plaintext crosses a local socket.

**Fix in Go first.**

- The encryption envelope base64-encodes the file twice (`modules/kernel/app/encryptionservice/service.go:196-214`).
  Stored files are about 1.78× their plaintext size, and a decrypt passes the blob through string
  copies, two base64 decodes and a JSON parse, roughly 7–9× the file size in heap before any PDF work.
  A binary envelope, or chunked AES-GCM that can stream, fixes this in any language.
- Add a server-side cache for rendered pages.

### R2. Capture companion uploads its own results, plus a shared protocol contract

**Today.** The Rust scan helper already builds a 240 px thumbnail and an 1,100 px view for every page
(`desktop/capture/crates/capture-imaging/src/preview.rs:17-22`) and keeps them in its local spool. The
upload sends only the PDF and a few headers (`crates/capture-client/src/api.rs:418-437`). The server
then reads each page back from storage, renders it through PDFium, rebuilds an almost identical
thumbnail (quality 72 instead of 70), measures ink coverage for blank detection and decodes QR codes
(`modules/documents/app/captureprocessing/processing.go:263-299`, four workers).

Luma and thresholding are written twice, with the same coefficients and different thresholds
(`capture-imaging/src/raster.rs:125-170` against `captureimaging/inspector.go:116` and
`contentbuilding.go:103`).

**With Rust.** Extend `put_page` to upload the thumbnail, an ink score and QR codes the helper decodes
(`rxing`). The server falls back to its own render only when those are missing. The device is paired
and authenticated, and cover-sheet tokens are still looked up per tenant, so trusting these values
carries little risk.

**Contract drift.** Protocol types are mirrored by hand. `capture-protocol/src/api.rs:1-7` says every
type mirrors a Go type, and constants such as `MAX_PAGE_BYTES` and the header names are copied
(`api.rs:13-34`, `api/handlers/capturehandler/device.go:17-20`). The seal digest has golden vectors on
the Rust side only (`capture-protocol/src/manifest.rs:60-72`); the Go tests compute expected values with
Go's own function (`captureflow/service_test.go:296,406`). And `native-capture.yml:36-48` runs only
on changes under `desktop/capture/**`, so a Go-side protocol change never runs the Rust tests. Check in
shared JSON fixtures and golden digests that both test suites read, and widen the CI path filter to
`modules/documents/domain/capture/**` and `capturehandler/**`.

**What improves.** Server CPU and PDFium contention for every scanned page, faster intake, and no silent
protocol drift. Effort is small, because the Rust code already exists.

### R3. One shared calculation crate for the browser and the server

**Status.** Done. `services/crates/rules` (`trenova-rules`) holds all five rule sets in the
table below, compiled to one `wasm32-unknown-unknown` module that the TMS runs in wazero and the
web app loads lazily. expr-lang is gone from the TMS, the browser's parser, checker and formatter
are deleted, and every Go and TypeScript copy named below is deleted. The expr-flavoured formula
language itself has since been replaced by Tariff, a statically typed rating language the crate
owns ([formula-language.md](../../engineering/formula-language.md)), with its corpora under
`testdata/tariff/` and a metered, exact evaluator. See
[rules-engine.md](../../engineering/rules-engine.md). The rest of this section is the analysis
as it was written.

**Today.** Several rule sets live twice, once in Go and once in TypeScript, held together by tests on
one side or by nothing.

| Logic | Server (Go) | Browser (TS) | How they are kept equal | Known drift |
| --- | --- | --- | --- | --- |
| Formula language parse and check | `expr-lang/expr` v1.17.8 plus `rating/app/formula` | `formula-editor/expr-parser.ts` (1,071 lines), `expr-check.ts` (1,656) | 73-case `testdata/conformance.json` | Precedence table hand-copied from expr's `operator.go` |
| Charge allocation | `decimalutils/allocate.go`, `operations/domain/shipment/allocationresolver.go:153` (half-to-even) | `packages/shared/src/lib/charge-split.ts:10` (`Math.round(v*100)`, half-up, floats) | TS tests only | Half-cent cases differ |
| Report value formatting | `pkg/reportfmt/format.go:214` (`shopspring/decimal`) | `packages/shared/src/lib/report-format.ts:242` (`toFixed`) | None | `1.005` → Go "1.01", JS "1.00"; `2.675` → Go "2.68", JS "2.67" |
| Lane specificity | `rategeo.Weight`, rate agreement weights | `packages/shared/src/lib/rate.ts` ("must stay in step", lines 7-12) | None | Unknown |
| Matrix quantity bands | `ratematrix.ContainsQuantity` | `packages/shared/src/lib/rate-matrix.ts` | None | Unknown |

The formula evaluator also has a runtime defect. On timeout it abandons the goroutine running the
expression, which keeps using CPU (`rating/app/formula/engine/engine.go:527-563`). Each evaluation also
deep-clones its environment (`engine.go:517`), and results pass through `float64` before becoming
decimals (`engine.go:656-710`, `matrix_lookup.go:292,315`).

**With Rust.** One crate (`trenova-rules`) holding the formula grammar, checker and evaluator plus
decimal money allocation, lane specificity, matrix bands and report rounding. It compiles twice: to
`wasm32` for the browser (lazy-loaded in the Formula Studio and rate screens) and to `wasm32-wasip1`
for wazero on the server.

- **One grammar.** The conformance corpus becomes a regression suite, not the only guard against
  drift.
- **Timeouts that stop the work.** A metered interpreter, or wazero's close-on-context-done, ends a
  runaway formula instead of abandoning it.
- **Exact decimals end to end**, with no per-call environment clone.
- **Small modules.** Rust WASM modules run to hundreds of KB, where a Go-to-WASM build of expr-lang
  would be megabytes.

**Costs.** Leaving expr-lang means owning the language: its semantics, error messages and every
existing formula's behavior. Rating runs on every shipment write, so server calls need batch APIs. A
call per formula is fine; a call per report cell over 100k rows is not. Report formatting needs
locale data, which argues for keeping `Intl` in the browser and sharing only the rounding.

**Cheaper alternative.** Do the TypeScript maths in integer minor units or decimal strings, and add
shared golden fixtures that both test suites consume. Fixing the report rounding and charge-split drift
this way first is the right move whether or not the crate follows.

### R4. The row-level security check as a pgrx extension

**Today.** `trenova_rls.claims()` (`internal/infrastructure/postgres/migrations/20261231007040_row_level_security.tx.up.sql:79-125`)
is PL/pgSQL, `SECURITY DEFINER`, with `SET search_path`. Each call:

- parses the `trenova.scope` setting with `string_to_array`
- looks up the signing key in `trenova_rls.scope_keys` (an SPI query)
- rebuilds HMAC-SHA256 by hand from two `sha256()` calls, hex-encodes it and compares it as text
- checks expiry with a regex

`org_id()`, `bu_id()` and `user_id()` each call it again. Every policy wraps them as
`(SELECT trenova_rls.org_id())`, so they run once per statement per occurrence, not per row. But
nothing caches the result: the standard policy (`standard_expr`, lines 256-267) verifies twice per
table reference, read-scope policies add more, and policies that reference other tables pull in theirs.
A five-table join verifies the same token about ten times, and nothing is reused across
statements. Estimated at 10–30 µs per call
(unmeasured), that adds roughly 0.1–0.4 ms to a typical statement.

**With Rust.** A pgrx `claims()` that verifies once per transaction (keyed by the token and the local
transaction id), keeps key pads in backend memory with invalidation, and skips SPI and the GUC
save/restore. Later calls take well under a microsecond. The cache lives in process memory, not in a
setting, so it cannot be forged.

**Costs.** Managed Postgres is ruled out. The tenant boundary would run in native code, so the
extension needs careful review and fuzzing.

**Try in SQL first.**

1. Measure with `track_functions = 'pl'` and `EXPLAIN ANALYZE`.
2. Fully qualify names and drop `SET search_path`, likely the largest overhead.
3. Return org and business unit from one row-returning function, so one InitPlan serves both columns.
   Confirm it is still indexable.

**Related, possibly larger, not a Rust issue.** No operator in the repo is `LEAKPROOF`. Under row-level
security, a user condition that is not leakproof usually cannot become an index condition. The query
builder emits exactly those: `ILIKE '%v%'` against trigram indexes (`pkg/querybuilder/querybuilder.go:318-326`),
`@@ websearch_to_tsquery` against about 80 tsvector GIN indexes (`:767`, `:807`), `ST_DWithin`
(`:1052`). In `enforce` mode these searches may scan the whole tenant. Check with
`SET ROLE trenova_tenant; EXPLAIN …`.

### R5. Search inside Postgres with `pg_search`

**Today.** GTC streams five tables into Meilisearch (`deploy/gtc/sinks.yaml`), and global search runs
one HTTP query per index, then enriches results from Postgres
(`modules/operations/app/globalsearchservice/service.go:136-180`). Meilisearch sits outside row-level
security, so every scope rule has to be re-applied in Go after the search. Searches also fetch whole
documents, including the full OCR text of documents, because no `AttributesToRetrieve` is set
(`infrastructure/meilisearch/client.go:98-101`).

**With Rust.** ParadeDB's `pg_search` is a Rust (pgrx) extension that adds BM25 search, built on
Tantivy, to Postgres. Moving the five indexes into Postgres would put search under the same row-level
security and read-scope policies as everything else, and would remove the Meilisearch container and its
GTC sink. GTC stays for the Redis caches and the `tca:events` stream.

**Costs.** Managed Postgres is ruled out here too. Typo tolerance and ranking would need checking
against Meilisearch's, and the `LEAKPROOF` question in R4 applies to its operator as well.

**Fix now regardless.** Set `AttributesToRetrieve`.

### R6. An X12 core crate (in progress)

The `trenova-edi` crate (`services/crates/edi`, one WebAssembly module run by wazero through
`pkg/x12`, and by the browser next) is the only X12 reader: separators, envelopes, a loop walker
driven by base and partner guides, element and syntax-note checks, 997/999 written from the
reading, a writer, proptest and three cargo-fuzz targets (`docs/engineering/edi-x12.md`).
`edix12inspect` and the inbound pipeline read through it, rejected transactions are not routed,
and acknowledgments report real errors. The browser loader, maps in place of `parse.go` and
the flat templates, and the designer follow (`docs/notes/edi/self-serve-edi.md`, order of work).

### R7. A local sandbox for code mode (done)

Scripts and data analysis run in Monty, compiled to WebAssembly by the `trenova-scripts` crate and
run in wazero inside the worker (`docs/engineering/agent-code-mode.md`, §5). E2B is gone: no vendor
account, no remote round trip per read, and no pandas, numpy or matplotlib, whose place the SDK's
datasets, statistics helpers and chart specs take.

### R8. `capture-imaging` in the Dash driver portal

**Today.** Dash sends raw phone photos, 3–8 MB each over cellular, through a file input straight into
`FormData` (`client/packages/shared/src/lib/portal.ts:29-34`). There is no resize, compression,
deskew, multi-page PDF or HEIC support, and HEIC is not in the server's MIME allow-list (`config.go:1247-1262`).

**With Rust.** Compile `capture-imaging` to WASM and run it in a Web Worker. A BOL photo becomes the
same black-and-white CCITT G4 PDF the scanners produce, tens of KB, and feeds the same server pipeline.
Edge detection and deskew (for example `imageproc`) would be new code.

**Alternative.** For a plain resize and JPEG, `createImageBitmap` plus `OffscreenCanvas` in a Worker is
enough and adds nothing to the bundle. Rust only earns its place if output should match the scanners.

### R9. Spreadsheet imports

**Today.** Rate, fuel-purchase and factoring-remittance imports parse uploads with excelize and load
the whole first sheet into memory as `[][]string` (`pkg/rateimport/sheet.go`). A 32 MiB upload
(`rateimportservice/service.go:35`) expands into hundreds of MB of XML and far more in Go strings.
`infrastructure/officedoc/xlsx.go` shows the bounded, streaming pattern.

**With Rust.** `calamine` under wazero parses several times faster in a fraction of the memory, and
untrusted zip and XML stay inside a memory-isolated module.

**Do in Go first.** Follow officedoc's pattern: stream rows and cap the row count. After that the Rust
gain is marginal unless import volume grows.

### R10. The i18n extractor

**Today.** `i18n/tools` is the last code in the repo that uses Babel. `task i18n-check` parses about
5,600 client files twice (`extract-ts.mjs:132`, `fragments.mjs:529`); the codemod parses them a third
time (`codemod.mjs:247`). ICU validation is three regexes (`validate.mjs`, 67 lines), and CLAUDE.md
already warns that a bare brace breaks a diagnostic message.

**With Rust.** An oxc-based binary that parses once for extraction and fragment detection, plus a real
ICU MessageFormat parser.

**Cheaper step.** Switch to the `oxc-parser` Node binding the other client scripts already use
(`client/scripts/loading-skeletons.mjs:13`).

### R11. A server-sent events gateway (only at larger scale)

**Today.** The realtime broker (`infrastructure/realtimebroker`) is well built: sharded Redis Streams, a
bounded publish queue, and slow readers closed instead of waited on. Each connection costs roughly
30–40 KB (goroutine stack, a 256-frame channel, bufio buffers; estimated). At the configured 20,000
connections per replica (`config.go:75-107`), that is about 0.6–0.8 GB.

**With Rust.** A tokio/axum gateway that speaks the documented key layout (`realtimebroker/keys.go`) at
a fraction of the memory per connection.

**When.** Only well beyond 20,000 connections per replica. Presence and authentication are tied into the
Go gateway (`kernel/app/realtimeservice/gateway.go`), so the gateway is not a clean cut.

### R12. A specified canonical form for the AI audit chain

**Today.** Audit rows are hashed over JSON in whatever bytes sonic happens to emit
(`modules/agent/domain/aiaudit/canonical.go`, `shared/jsonutils/canonical.go`): its string escaping,
invalid-UTF-8 handling and byte-order key sort (RFC 8785 sorts by UTF-16 code units). Yet external
auditors are expected to re-hash exported rows (`docs/engineering/ai-audit-trail.md:132-158`). A sonic
upgrade could silently change the bytes and break verification of old rows.

**With Rust.** An RFC 8785 (JCS) encoder and chain verifier shipped as one crate, used by the server
(through wazero) and as an offline auditor CLI.

**Not language-bound.** The value is the spec, and a Go JCS implementation works too. Separately, the
chain is signed with symmetric HMAC; Ed25519 signatures would let third parties verify exports.

### R13. PII scrubbing (only if profiling shows it)

`shared/piiscrub` runs six RE2 patterns plus Luhn, ABA and IBAN checks over embedding inputs
(`agentcompletion/completionrouter/embed.go:105`) and training-export text
(`aianonymizer/anonymizer.go`), and the anonymizer adds about twelve more regex passes. Rust's
`RegexSet` matches all patterns in one pass and is much faster per byte. Volume is modest today, so act
only if it shows up in a profile.

### Candidates at a glance

| # | Candidate | Main gain | Seam | Effort (rough) | Prerequisite |
| --- | --- | --- | --- | --- | --- |
| R1 | Document and imaging worker | Memory, throughput, one imaging library | Sidecar (done) | Large | Binary encryption envelope; page cache |
| R2 | Capture companion uploads results | Server CPU per scanned page | Existing desktop crate | Small | Shared JSON fixtures; CI path filter |
| R3 | Shared calculation crate | One answer in browser and server; hard timeouts | Browser WASM plus wazero | Large | Fix TS rounding with shared fixtures first |
| R4 | pgrx `claims()` | Per-statement overhead under RLS | Postgres extension | Medium | Measure; SQL fixes |
| R5 | `pg_search` | Search under RLS, one fewer service | Postgres extension | Medium | — |
| R6 | X12 core crate | Robust, guide-driven parsing | wazero plus browser WASM (reading done) | Medium | — |
| R7 | Local code-mode sandbox | No vendor, lower latency | wazero (done) | Medium | — |
| R8 | Imaging in Dash | Small uploads over cellular | Browser WASM | Small–medium | PWA basics |
| R9 | Spreadsheet parsing | Memory, sandboxing | wazero | Small | Go limits and streaming first |
| R10 | i18n extractor | One parse, real ICU checks | CLI | Small | Try `oxc-parser` first |
| R11 | SSE gateway | Memory per connection | Sidecar | Medium | Only beyond 20k per replica |
| R12 | JCS canonical crate | Verifiable audit exports | wazero plus CLI | Small | — |
| R13 | PII scrubbing | Regex throughput | wazero | Small | A profile that shows it |

## 4. Where Rust will not help

| Area | Why not |
| --- | --- |
| GraphQL executor and resolvers | The pain is Go compile time (354 s cold before sharding, 152 s after a domain-struct edit). Rust compiles more slowly. `async-graphql` would mean rewriting every resolver. |
| Agent runtime and model I/O | Network- and Temporal-bound. Streams parse fine with sonic; no partial-JSON parser is needed. |
| Token counting | Claude's tokenizer is not public. Calibrate from the provider's reported `usage.input_tokens` instead of the `runes/4` (`shared/llmtokens`) and `bytes/4` (`agentruntime/turnhistory/contextuse.go:134-143`) heuristics, which disagree. |
| Tool-argument validation | Compiled once per tool (`pkg/toolschema/validate.go:50-101`); a WASM call would cost more than the validation. |
| Retrieval | pgvector and Postgres full-text search do the work; in-memory catalogs are a few hundred items. |
| numberguard | O(citations × supported) decimal comparisons; the fix is a sorted float slice and binary search in Go. |
| Dispatch planner and assignment solver | Time goes to the scoring callback, which builds detail strings with `fmt.Sprintf` in the inner loop (`dispatchcandidateservice/score.go:177-210`). Fix in Go. |
| ETA, hours of service, mileage, telematics | I/O-bound (PC*MILER, Samsara) or small pure functions. |
| Accounting, billing, settlement, IFTA | Database-bound decimal maths. No bank-file parsers exist yet. |
| Permission engine | Cost is two Redis round trips and a JSON decode per check, not CPU. |
| Auth and crypto | Thin wrappers over Go's standard library, go-oidc and Cloud KMS. SAML does not exist yet; if it is built, XML-DSig is where library choice matters most. |
| Reporting compiler | Postgres does the work; compiling takes milliseconds. |
| Template engine | Rust template engines lack `html/template`'s context-aware escaping, so a port would weaken security. |
| Change-data-capture consumers, Redis Lua, migrations | A few events a second, scripts that run inside Redis, DDL-bound. |
| Rate limiting, idempotency, webhook signatures, SSRF guard | Tiny and Redis-bound. |
| Go code generators and lint tools | They need `go/types`. |
| Simulators, CLI, observability | Development fixtures, operator commands, configuration. |
| Browser CSV export and maps | A Web Worker and marker clustering, not WASM. |
| Python ML tooling, docling | Tied to the Python ML ecosystem; no mature Rust equivalent for layout and table extraction. |

One more note: `shared/samsara/internal/samsaraspec/types.gen.go` is 90,650 lines of oapi-codegen
output. Trimming the spec to the endpoints used would cut compile time more than any port.

## 5. Recommended order

1. **Now, in Go:** the tracked defect fixes, and the measurements in section 6.
2. **First Rust work:** R2 (small, reuses existing code) and a spike of R1's pure pixel path under wazero,
   measured against the current Go path.
3. **Shared rules:** fix report rounding and charge allocation with shared fixtures, then decide on R3 with
   the formula language as its anchor.
4. **Database:** run the R4 measurements and SQL fixes, then decide on pgrx. Evaluate R5 at the same time,
   since both rule out managed Postgres.
5. **As needs arise:** R11
   only at scale, the rest when a profile points at them.

## 6. What to measure before porting

| Question | How |
| --- | --- |
| Cost of `claims()` per statement | `track_functions = 'pl'`, then `pg_stat_user_functions` under a realistic load |
| Index use under row-level security | `SET ROLE trenova_tenant; EXPLAIN ANALYZE` on trigram, full-text and `ST_DWithin` searches |
| Where capture processing spends time | CPU profile of `captureprocessing/processing.go` over a 100-page batch |
| Peak heap per document | Heap profile of upload → decrypt → thumbnail for 5, 20 and 50 MB files |
| Page-viewer latency | p50/p95 of `GetPageImage` under concurrent Desk use |
| wazero against native for pixel work | Benchmark the same resize, threshold and QR code natively, under wazero, and in Go |
| numberguard per reply | Benchmark `CheckAmounts` on a tool-heavy turn |
| Redis round trips per GraphQL request | Count permission-cache reads per operation |
| Formula evaluation under load | Abandoned-evaluation metric (`AbandonedEvaluationFinished`) in production |

## 7. Glossary

- **wazero:** a WebAssembly runtime written in Go, already used by the TMS for PDFium and WebP.
- **wasmtime:** a WebAssembly runtime written in Rust, with fuel metering and epoch interruption.
- **pgrx:** a framework for writing Postgres extensions in Rust.
- **InitPlan:** a subquery Postgres runs once per statement and reuses.
- **LEAKPROOF:** a Postgres function marking that lets the planner apply a user's condition before the
  row-level security check, which is what allows index use.
- **JCS:** JSON Canonicalization Scheme (RFC 8785), a byte-exact JSON encoding for hashing and signing.
- **CCITT G4:** the compression scanners use for black-and-white pages.
