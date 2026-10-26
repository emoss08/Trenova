# Self-serve EDI: the design

As of 2026-10-26. This is the plan for candidate R6 in
[../rust/opportunities.md](../rust/opportunities.md), widened from "an X12 parser" to the thing
the parser is for. In most TMS products the provider's EDI specialist sets up every trading
partner. Trenova should not need one: an organisation reads its partner's implementation guide,
builds the connection itself, tests it and turns it on. Every piece below exists to make that
work a person (or the Desk agent) can do in the product, checked as they go.

What is built is described in [../../engineering/edi-x12.md](../../engineering/edi-x12.md); this
note is the plan and the reasons for it.

## 1. Where EDI stands

What the audit of `modules/edi` (about 48,000 lines of Go) and `routes/edi` (about 63,000
lines of TypeScript) found:

- **Outbound templates are flat.** A template version is a list of segments, each repeating
  over its own field path (`repeatPath`). A loop is therefore emitted one segment at a time
  across all its repetitions. The starter 204 renders a two-stop tender as
  `G62 G62 N1 N1 N4 N4 G61 S5 S5`, where X12 wants each stop's `S5` loop to carry its own
  `G62`, `N1` and `N4`. The same flaw writes every `AK3` of a 997 before every `AK4`.
- **The inbound parser reads its own broken output.** `ediinboundservice/parse.go` (1,142
  lines) maps 204, 210, 214, 990, 997 and 999 by hand from a flat segment list. It has a
  "backfill" step that pairs a stop with `N1`s found before any `S5`, which only exists because
  of the renderer above. No loop is understood; nothing a partner does differently can be
  configured.
- **Every acknowledgment accepts.** `acknowledgmentPayloadForTransaction` writes `A` into
  AK5 and AK9 whatever arrived. Validation (`edix12inspect/validation.go`) checks only the
  envelope: segment order of ISA/GS/ST and the control counts.
- **The dictionary is unused.** `edi_transaction_set_definitions`,
  `edi_transaction_loop_definitions`, `edi_transaction_segment_definitions`,
  `edi_transaction_element_definitions` and `edi_code_list_definitions` have no reader and no
  writer. The inspector names 16 segments from a map in `dictionary.go`.
- **Six transaction sets, closed.** `edi.TransactionSet` is an enum of 204, 210, 214, 990,
  997 and 999, so a partner who needs a 211, 824 or 856 cannot be set up.
- **Logic that only the server can run.** Template conditions and element values use Starlark
  (`go.starlark.net`) and a Go transform pipeline, so the designer cannot preview or check a
  template without a round trip, and nothing checks a value against the element it fills.

## 2. The decisions

| Decision | Chosen | Why |
|---|---|---|
| Where the X12 core lives | `services/crates/edi`, one `wasm32-unknown-unknown` module with no imports, run by the TMS in wazero and by the browser | The designer, the inspector, the partner simulator and production read and write files with the same bytes, as `trenova-rules` does for rating |
| One module or a part of `rules.wasm` | Its own module, `edi.wasm` | EDI pages should not download Tariff, rating pages should not download X12, and an EDI change should not rebuild the rating module. The Go host and the browser loader are shared code (`pkg/wasmmodule`), not copies |
| How a partner's rules are expressed | An implementation guide as data: a base guide plus what the partner narrows | It is what a partner hands over, it is what an EDI specialist reads, and it is checkable |
| Where base definitions come from | Curated base guides for the sets Trenova supports, written from partners' published implementation guides; anything else from an SEF import, the onboarding agent reading the partner's PDF, or inference from sample files | The ASC X12 dictionary is licensed material; a partner's own guide is the authority for that partner anyway |
| How values are computed | A small typed expression language inside the crate; Starlark and the Go transforms are deleted | It runs in both hosts, it always terminates, and a map is checked against the guide and the Trenova document schema before it runs |
| Acknowledgments | Built by the crate from the validation report, never from a template | A 997 or 999 is protocol, not a partner mapping, and its AK2/AK3/AK4 loops must nest |
| Backwards compatibility | None | Trenova has not launched: the flat templates, `parse.go`, Starlark and the unused tables are deleted as their replacements land |

## 3. The crate

```
services/crates/edi/src/
  abi.rs, ops.rs, message.rs   three exports, the operation table, user-facing messages (as in trenova-rules)
  x12/separators.rs            separators from the ISA, fixed-width or squeezed
  x12/lexer.rs                 segments, elements, repetitions and components as borrowed slices with byte spans
  x12/envelope.rs              ISA/GS/ST nesting, control numbers and counts
  x12/guide/                   the guide model, its check and compiled form, and the built-in base guides
  x12/structure.rs             the loop tree, built by walking the guide
  x12/validate.rs              element, segment, loop and syntax-note checks, as X12 error codes
  x12/ack.rs                   997 and 999 bodies from a validation report
  x12/write.rs                 the writer: segments to text with a profile's separators
```

### Reading

1. **Separators** come from the ISA: the element separator is the byte after `ISA`, and the
   component separator is the byte after the sixteenth element. The segment terminator is the
   byte after that, plus a trailing CR, LF or CRLF when present. An ISA whose padding was
   squeezed (a sample copied out of a PDF) is read by counting separators instead of offsets.
   The repetition separator is ISA11 from version 00402 on.
2. **The lexer** yields each segment as a borrowed slice with its byte span; elements,
   repetitions and components are split on demand. Nothing is copied until a value goes on
   the wire.
3. **The envelope** checks ISA/IEA, GS/GE and ST/SE pairing, control numbers in header and
   trailer, the counts in IEA01, GE01 and SE01, and duplicate control numbers. Interchange
   problems carry TA1 note codes, group problems AK9/AK905 codes, transaction problems
   AK502/IK502 codes.
4. **The loop tree.** The guide's structure is a tree of segment uses and loops; a loop is
   recognised by its first segment. For each segment the walker looks forward in the current
   loop for a use with that ID or a child loop it starts; failing that, a repeat of the current
   loop when the segment is its trigger; failing that, the parent loop, up to the transaction.
   A segment found nowhere is unrecognised (AK304 code 1) when the guide does not define it,
   out of sequence (code 7) when it is defined elsewhere, and not in the set (code 6) when the
   set never uses it. The walk is linear in the number of segments.
5. **Validation** runs on the tree:
   - segments: mandatory missing (3), loop over its repeat (4), over max use (5), and in a
     999 a "not used" segment present (I4) or one under the partner's minimum (I8);
   - elements: mandatory missing (1), too many elements (3), too short (4), too long (5), an
     invalid character (6), a code not in the list (7), a bad date (8) or time (9), too many
     repetitions (12) or components (13), and in a 999 a "not used" element present (I10) or
     a code the partner excludes (I6);
   - syntax notes: paired (`P`), required (`R`), exclusion (`E`, code 10), conditional (`C`)
     and list conditional (`L`) (code 2 when a required element is missing).

   Each finding carries the segment's position in the transaction, its loop ID, the element
   and component position, the X12 code and a message for people.

### Guides

A guide names its standard, version, transaction set and functional group, and holds:

- a **segment dictionary**: each segment's elements in order, with the element number, name,
  type (`AN`, `ID`, `DT`, `TM`, `N0`–`N9`, `R`), minimum and maximum length, usage
  (mandatory, optional, conditional), repeat count, components for a composite, code list and
  syntax notes;
- a **structure**: the tree of segment uses and loops, each with usage and maximum use or
  repeat.

A partner guide is a guide with `derivedFrom` naming its base. It may only narrow: make an
optional segment or element mandatory or not used, lower a maximum, shorten a length, restrict
a code list, fix a value, and add notes. `x12.checkGuide` reports anything that widens the
base, so a partner guide can never accept what the base standard rejects.

Base guides ship compiled into the crate for 004010 204, 210, 214, 990 and 997, and 005010
999 (`X231A1`). Code lists are included only where they are short and stable (acknowledgment
codes, ISA qualifiers, usage indicators); a list a partner restricts comes from that partner.

### Writing and acknowledging

The writer turns segments into text with a profile's separators: ISA fields padded to their
fixed widths, trailing empty elements trimmed, and SE01, GE01 and IEA01 counted. `x12.acknowledge`
builds a 997 or a 999 from a validation report: one AK2 loop per transaction, an AK3 or IK3 loop
per segment in error with its AK4 or IK4 elements inside it, AK5 or IK5 with `A`, `E` or `R`,
and AK9 with `A`, `E`, `P` or `R` and the three counts.

### Bounds

One call per interchange. Inputs are capped at the host's 64 MiB request size; a transaction
reports at most 1,000 findings (then says it was cut short); loops nest at most 16 deep. Every
walk is linear in the input, so a hostile file costs at most its own length.

### Fuzzing

`cargo-fuzz` targets for the lexer and envelope, the loop walker against every base guide, and
the guide check, run for a bounded time in CI. A property test writes random trees with the
writer and reads them back to the same tree.

## 4. Maps: what lets an organisation build its own

A **map** connects a guide to a Trenova document type (load tender, shipment status, freight
invoice, tender response). It is written in both directions with one language:

- an **inbound map** reads the loop tree into a document: `stops <- each 0300` with
  `name <- 0310.N1[N1-01 = "SH"].N1-02`, a qualifier picking the repetition it wants;
- an **outbound map** builds the tree from a document: `0300 <- each stops`, so a stop's loop
  carries its own segments by construction.

Expressions are typed and total: paths into the tree or the document, the functions that
today's transforms provide (trim, pad, case, substring, formats for dates, times, decimals,
phones, states and postal codes, qualifiers, conditionals, coalesce), code tables, partner
settings and lookups into the organisation's mapping profiles. A map is checked before it runs:
a `DT` element filled from text, a code the partner's guide does not allow, a mandatory element
no rule fills, a document field the target type does not have. The starter maps for the six sets
are written in this language and replace `parse.go`, the flat renderer, the transforms and
Starlark.

## 5. The designer

The designer loads `edi.wasm` and works locally:

- paste a partner's sample and see its loop tree, the guide's findings and the mapped document
  at once;
- edit a guide or a map with completion of segment paths and codes from the guide, hover with
  element names and usage, and diagnostics with fixes, following the Tariff editor
  ([../../engineering/formula-language-service.md](../../engineering/formula-language-service.md));
- start a guide from an SEF file, from the onboarding agent's reading of the partner's PDF (its
  `requirements[]` become the narrowing), or from a batch of samples (segments seen, their
  maximum use and the codes in use);
- keep certification tests: a sample file and the document it must map to, or a document and
  the X12 it must produce. They run in the browser as you type and on the server before a
  version can be activated.

## 6. Order of work

Each step ships complete, with its docs, tests and generated artifacts.

1. **The crate's reading core**: separators, lexer, envelope, guides and base guides, the loop
   walker, validation, the writer and acknowledgments, the WebAssembly build, fuzzing; the
   shared Go module host (`pkg/wasmmodule`) and the EDI host (`pkg/x12`); `edix12inspect`
   reads through the crate, inbound files are validated against their guide, acknowledgments
   report what was wrong and reject what is, and the unused dictionary tables are dropped.
2. **Maps and the inbound path**: the map language, its checker and evaluator, the starter
   inbound maps, per-profile inbound maps; `parse.go`'s hand mapping is deleted.
3. **Outbound on the tree**: outbound maps and envelope writing in the crate; the flat
   renderer, transforms, Starlark and the template segment table are deleted.
4. **The designer on the module**: the browser loader, the guide and map editors, golden
   tests, SEF import, guide inference and the agent tools (validate a file, explain a finding,
   propose a map change, draft tests from samples).
5. **An open transaction-set catalog**: any set a guide defines is received, validated,
   acknowledged and archived, and mapped when Trenova has a document type for it.
