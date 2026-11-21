# X12 reading, validation and acknowledgments

Trenova reads, checks and acknowledges X12 in one Rust crate, `trenova-edi`
(`services/crates/edi`), compiled to one WebAssembly module, `edi.wasm`. The TMS runs it in
wazero through `services/tms/pkg/x12`; the designer will run the same bytes in the browser. The
goal is an EDI an organisation builds itself: the partner's implementation guide is data, the
checks follow it, and nothing about a partner lives in Go code. The plan for the rest (maps, the
designer) is [docs/notes/edi/self-serve-edi.md](../notes/edi/self-serve-edi.md).

Read this before changing the crate, `pkg/x12`, `edix12inspect`, the inbound pipeline's parsing
(`ediinboundservice/parse.go`), how acknowledgments are written, or a starter template's numbers.

## Layout

| Path | What |
|---|---|
| `src/abi.rs`, `src/ops.rs` | `edi_alloc`, `edi_call`, `edi_free` (through `trenova-module-kit`'s `export_abi!`) and the operation table |
| `src/x12/separators.rs` | separators from the ISA: fixed 106 bytes, or squeezed (read by position, with a finding); line endings after the terminator |
| `src/x12/lexer.rs` | segments with byte spans, repetitions and components |
| `src/x12/envelope.rs` | ISA/GS/ST nesting, every envelope check, at most `MAX_SEGMENTS` (2,000,000) |
| `src/x12/structure.rs` | the loop walker: places each segment in the guide's tree |
| `src/x12/element.rs`, `guide/syntax.rs` | element checks (type, length, codes, dates, times, implied decimals) and syntax notes `P R E C L` |
| `src/x12/finding.rs` | every finding kind, its level, its 999/997/TA1 codes and default severity (`finding_kinds!`) |
| `src/x12/read.rs` | `x12.read`: the whole reading in one call |
| `src/x12/ack.rs`, `src/x12/write.rs` | `x12.acknowledge` (997/999 from a reading) and `x12.write` (character set, ISA padding, counts, separator cleaning) |
| `src/x12/charset.rs`, `charset/table.rs` | the X12 character set and `fold`, which writes text in it: each character outside it as its closest ASCII form, or dropped |
| `src/x12/guide/` | the guide model, the compact text form, the base guides, compiling, and `check` for partner guides |
| `src/x12/guide/data/` | `envelope.dict`, `004010.dict`, `005010.dict`; guides for 004010 204, 210, 214, 990, 997 and 005010 999 |
| `tests/x12_properties.rs`, `fuzz/` | proptest (write then read round trips; any text reads); cargo-fuzz targets `x12_read`, `x12_guide`, `dispatch` |
| `services/tms/pkg/x12/` | the Go host: the embedded `edi.wasm`, typed calls, the wire types, `ParseNumber` |
| `services/tms/pkg/wasmmodule/` | the wazero runtime, pool, compile cache and metrics observer shared with `pkg/rules` |

## Operations

| Op | Does |
|---|---|
| `edi.version` | the module version and its operations |
| `x12.read` | separators, envelopes, segments, loop placement, findings and an acknowledgment verdict per group, for a whole file or a fragment |
| `x12.acknowledge` | a complete 997 or 999 interchange per group, from what `x12.read` found |
| `x12.write` | X12 text from segments: every value in the X12 character set (with a finding), ISA widths, SE01/GE01/IEA01 counts, trailing empty elements trimmed, a separator inside a value written as a space (with a finding) |
| `x12.guides`, `x12.guide` | the base guides, and one guide in full |
| `x12.checkGuide` | a partner guide's problems, including where it widens its base |

One call reads one file: the reading carries every transaction, so a caller never loops over
the module per transaction.

## Guides

A **guide** is a segment dictionary plus a structure tree: segments with usage and maximum use,
loops with a repeat count, each loop identified by its first segment. The base guides are the
published X12 sets, written in the compact text form described in `guide/text.rs` (one element
per line, so they review against the published standard). A **partner guide** is JSON with
`derivedFrom` naming its base. It may only **narrow** the base: make an optional element
mandatory or not used, shorten lengths, restrict codes, lower repeats, add syntax notes, and set
`severities` to raise or lower a finding kind for that partner. `x12.checkGuide` reports every
place it widens the base, so a guide that would accept what the standard rejects is caught
before it is used.

The walker matches a segment against the current loop, then its parents, never against a loop's
own trigger as a repeat inside that loop (the `find` floor in `structure.rs`). A segment it
cannot place is classified: not in this set, in the set but out of place, not used by this
partner, or past its maximum.

## Findings

Every finding has a kind, a level (file, interchange, group, transaction, segment, element), a
severity and the codes the acknowledgments use: TA1 (interchange), AK905, AK502/IK502, AK304 and
IK304, AK403 and IK403, with 997 fallbacks for the 999-only codes. Its text is a `message!`
template, extracted into the Go scope and the web bundle (i18n `RUST_ROOTS`), and localized on
the server with `i18n.FromContext(ctx)` (`edix12inspect.FindingDiagnostic`). A transaction keeps at
most `MAX_FINDINGS_PER_TRANSACTION` (1000) findings and says how many it left out.

## Inbound

`ediinboundservice` reads a received file once with `x12.read`, choosing each transaction's
guide (the partner's, else the base for its set and version). Then:

- a file that is not X12, has no interchange or no transactions is **quarantined**;
- a transaction with an error is **rejected**: its message is `Failed` with the findings as
  diagnostics, and it is not routed (a file whose only transaction is rejected is quarantined);
- each functional group gets its acknowledgment from the reading, written by `x12.acknowledge`,
  with the idempotency key `inbound-ack:<fileID>:<version>:<groupIndex>`, so a retried file
  sends each acknowledgment once.

Numbers are read by their X12 type: `x12.ParseNumber(value, type)` applies an `Nn` element's
implied decimal places unless the value has an explicit point (which the crate reports as a
warning). The element's type comes from the guide (`X12Element.DataType`), never from a
hard-coded position.

## Acknowledgments

A 997 or 999 is written by the crate from the reading of what it answers, never from a template.
An acknowledgment profile (`edi_partner_document_profiles` for set 997 or 999) therefore has no
template: `template_id` is null, a check constraint keeps it so, and a template for 997 or 999
cannot be created. `GenerateDocument` and `PreviewDocument` for an acknowledgment need the
source message (`sourceMessageId`).

## Outbound, until maps land

Outbound documents still render from templates (`edix12`). An `Nn` element is written with the
`format_decimal` transform and `implied: true`, which shifts by the places and drops the point;
the starter 204 and 210 use it for every N2 charge, and dates go through `format_date`. A new
starter element with a numeric X12 type uses the same helpers in `templates/base204.go`
(`amount`, `repeatAmount`, `date`).

## A flow instance opens only when a step leaves something to keep

`edilading.Flows.Dispatch` steps a key with no instance from an empty state (`{}`, so Lading's
`instance.exists` is false) and stores the instance only when the step leaves something to keep:
a received or sent document, a load, a pending send, a batch, a timer, or an action beyond the
acknowledgment (`opens`, `lading.FlowStarted`). A message the flow rejects or ignores opens
nothing and links to no instance. The new row goes in with the step's state through
`CreateInstance` (`CreateOrSkip` on `uq_edi_flow_instances_key`); a delivery that loses that race
gets `ErrFlowInstanceExists`, reads the winner and steps again from its state, as a version
conflict on `SaveInstance` does. Never create an instance before its first step.

## Temperatures on a tender

A loadTender's `Stop.temperature` and `CommodityLine.temperature` land on the shipment's stops and
commodity lines (`temperature_min`/`temperature_max`, whole °F, as the shipment's own), and
`FromLoadTender` writes them back out. The shipment's range is never typed twice: when any live
stop or line has a range, `shipmentnormalize.Temperature` derives it on every save
(`temperaturerange.Load`): the range they all share, or the span from the coldest to the warmest
bound when they share none (a frozen and a chilled drop), so a multi-temperature load still reads
as reefer to rating, the dispatch board and quick filters. With no line range the shipment's range
is taken as entered, and a tender's load-level `temperature` fills it.

## Advancing a flow's clock in development

A Lading flow's timers (a planner's answer window, an acknowledgment wait, a batch) are due
times stored on its instance and swept each minute (`DispatchDueFlowTimers`). To play a
conversation without waiting, move one instance's clock forward:

```sh
trenova edi flows advance --instance edfi_01J... --by 2h      # or --to-due
trenova edi flows advance --instance edfi_01J... --by 30m --json
```

It moves every stored time the flow compares against now back by the duration (the instance's
`wake_at`, and in its state `awaitingResponse`, `responseDue` and each `pending` and `batches`
`due`; `lading.ShiftFlowState`), records an `advanced` event in the instance's history, and
wakes that one instance through `DispatchFlowTimer`, the sweep's own path. It prints the status,
why the flow holds, the next due time, the actions taken and the messages sent; `--json` is
for the EDI simulator. It finds the instance's tenant under a system scope, then works under
that tenant's scope. It refuses to run unless `app.env` is `development`, and has no route or
GraphQL field. A new time field in the Rust `FlowState` that the flow compares against now
belongs in `ShiftFlowState` too, or an advance leaves it behind.

## Build and checks

`task rust:wasm-edi` builds the module (profile `wasm`, `-Os`, the shared reproducible build in
`services/scripts/wasm-module.sh`) and installs `services/tms/pkg/x12/edi.wasm` with its
`.sha256`; `task rust:wasm-edi-check` fails when the committed copy differs. The WebAssembly
modules job in `rust.yml` runs the check and clippy for `wasm32-unknown-unknown`; the fuzz job
runs the three targets. The Go host's tests run with the TMS.

## The rules that bite

- **The module is the only X12 reader.** Go never splits segments, counts elements or checks
  envelopes; it asks `x12.read` and maps the reading. A second parser drifts from the first.
- **Rebuild the module with the crate.** A Rust change without `task rust:wasm-edi` leaves Go
  tests running the old bytes; the check job fails the push.
- **Positions and types come from the guide.** Reading an amount by a fixed position, or
  parsing an `N2` literally, is how the 210 read B306 as the amount and `125000` as $125,000.
- **Partner guides narrow.** Never accept a partner guide `x12.checkGuide` reports as widening
  its base.
- **Acknowledgments never have templates**, and an acknowledgment is never built in Go.
- **Only the X12 character set goes on the wire.** X12.6's basic and extended sets together are
  printable ASCII (U+0020 to U+007E); a partner's translator rejects anything else with AK403
  code 6. Records hold names as people typed them (Peñasco, Société Générale, an en dash, `½`,
  curly quotes), so every value is folded by `charset::fold` before it is written: an accented
  letter or typographic mark becomes its closest ASCII form (`ñ` n, `ß` ss, `Æ` AE, `–` -, `½`
  1/2, a non-breaking space a space), and a character with none (CJK, emoji, combining marks,
  controls) is dropped. A Lading map folds in `write_elements`, before the element's length
  check, and reports a `written-in-character-set` warning naming the element; when only the
  folding pushed a value past the element's maximum it is cut to the maximum, and a value that
  folds to nothing is written as absent, so a required element is reported missing.
  `x12.write` folds too (`valueWrittenInCharacterSet`), before it pads or cuts the ISA and before
  it replaces separators, since a fold can produce one (`›` becomes `>`). There is no
  per-partner character set: the guide model has none, and printable ASCII is what every
  partner's translator takes. Reading is liberal: a character outside the set in an `AN` or `ID`
  element is `characterOutsideCharacterSet`, AK403 6 as a **warning**, so the transaction is
  accepted with errors, not rejected; a partner's guide raises it with
  `severity character-outside-character-set error`. Lading's `clean()` folds the same way. The
  fold's table is generated from Unicode's NFKD with marks removed plus hand entries
  (`charset/table.rs` says which); it adds about 14 KB to `edi.wasm`, where a normalization
  crate would carry every script's decompositions only to drop most of them.
