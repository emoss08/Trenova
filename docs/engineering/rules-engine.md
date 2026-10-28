# Rules engine (`trenova-rules`)

Some rules have to give the same answer in the browser and on the server, to the cent:

- Tariff, the rating language, which the Formula Studio checks as you type and the server rates
  with;
- how a charge is split between payers;
- how a report number is rounded and written;
- how specific a rate agreement lane is;
- which rate matrix band a quantity falls in.

Each of these used to be written twice, once in Go and once in TypeScript, and the copies drifted.
For example, `1.005` came out as "1.01" on the server and "1.00" in the browser, and the two
halves of a half-cent split disagreed. They now live in one Rust crate,
`services/crates/rules`. It compiles to **one WebAssembly module**: the TMS runs it in wazero,
and the web app loads the same bytes lazily.

Read this before changing anything in the crate, the Go host (`services/tms/pkg/rules`), the
browser loader (`client/packages/shared/src/rules`), or a caller of any of them. Rust
conventions are in [rust.md](rust.md).

## Layout

| Path | What |
|---|---|
| `services/crates/rules/src/abi.rs` | the three exports: `rules_alloc`, `rules_call`, `rules_free` |
| `src/ops.rs` | the operation table and the request/response envelope |
| `trenova-module-kit` (`services/crates/module-kit`) | `export_abi!`, operation dispatch, and `message!` and `Message`: user-facing text as English ICU templates plus arguments, shared with `trenova-edi` |
| `src/span.rs` | byte spans inside the crate; UTF-16 `Range` on the wire |
| `src/wire.rs` | `Dec`: exact decimals on the wire |
| `src/tariff/syntax/` | lexer, error-tolerant parser, syntax tree, outline, formatter, tokens for highlighting, and loading a file with the libraries it imports; `tariff.format`, `tariff.tokens`, `tariff.setName` |
| `src/tariff/check/` | types, units, inference, flow narrowing, exhaustiveness, modules, inputs, diagnostics with fixes, and the catalog; `tariff.check`, `tariff.catalog` |
| `src/tariff/eval/` | values with units, the metered interpreter, the bill and its total clauses, rate tables, the trace, the test runner and the test scaffold; `tariff.evaluate`, `tariff.test`, `tariff.scaffoldTest` |
| `src/tariff/editor/` | completion, hover, signature help, go to definition, rename and inlay hints; the six editor operations |
| `src/tariff/stdlib/` | the standard library, written in Tariff (`ltl`, `fuel`, `time`, `accessorials`) and compiled in, with its docs as catalog templates |
| `src/tariff/units.rs`, `types.rs` | exact unit algebra and dimensions; the types the checker infers and wire values decode against |
| `src/tariff/schema.rs`, `context.rs` | the data roots, schema enums, table and charge code shapes; the context every file is read with |
| `src/tariff/docs.rs`, `diagnostic.rs` | docs as the author's words or a `Message`; findings and their wire shape |
| `src/tariff/wire.rs` | Tariff values on the wire, decoded against their types and encoded canonically |
| `src/allocation/`, `src/report/`, `src/lane/`, `src/matrix/` | the other rules |
| `services/crates/rules/testdata/` | the regression corpora every implementation detail answers to: one file per area for the other rules, and `tariff/syntax/`, `tariff/check/`, `tariff/eval/` and `tariff/editor/` with one corpus per topic |
| `services/tms/pkg/rules/` | the Go host: the embedded `rules.wasm` and typed calls per area, on the shared runtime in `pkg/wasmmodule` |
| `client/packages/shared/src/rules/` | the browser loader: the same `rules.wasm`, `loadRules()`, synchronous access once loaded, typed calls per area |

## One artifact, two hosts

The crate has no imports: no clock, no randomness, no I/O and no WASI. So a single
`wasm32-unknown-unknown` build runs unchanged in wazero and in the browser. Agreement is
therefore not a property tests have to maintain: both sides run the same bytes.

`task rust:wasm-rules` builds the module and installs it in two places, each with a
`rules.wasm.sha256` sibling:

- `services/tms/pkg/rules/rules.wasm` (`//go:embed`);
- `client/packages/shared/src/rules/rules.wasm`.

The build uses:

- the `wasm` profile (release, `opt-level = "s"`);
- `--locked`;
- `--remap-path-prefix` for the workspace and the cargo home;
- `wasm-opt` at a pinned binaryen version.

CI rebuilds the module and fails when either committed copy differs from it.

The module is 2,342,318 bytes (2.2 MiB), 879,918 bytes gzipped (`gzip -9`). The browser
fetches it lazily, never on the startup path, and the service worker caches it as a hashed
asset. Most of the weight is the crate's own code (Tariff's checker, interpreter, editor
features and standard library), Rust's collections and JSON. `opt-level = "z"` would shrink it
at a cost in speed, which matters on the server's report and rating paths, so the profile stays
at `"s"`.

### The ABI

The protocol, in order:

1. The host asks `rules_alloc(len)` for a buffer.
2. It writes a UTF-8 JSON request into the buffer.
3. It calls `rules_call(ptr, len)`. The call returns a `u64`: the response's address in the high
   32 bits and its length in the low 32.
4. The host reads the response.
5. It frees both buffers with `rules_free(ptr, len)`.

The envelope, request and response:

```json
{"op": "tariff.check", "input": { ... }}
{"ok": { ... }}
{"error": {"code": "malformed" | "unknown_op" | "invalid_input", "detail": "..."}}
```

`ok` means the operation ran, whatever it found: a file full of errors is still a successful
check.

`error` means the request was unusable. That is always a bug in the caller. Its detail is
developer English, never shown to a user and never translated.

A panic inside the module traps. The host treats a trap as a bug: it discards that instance and
reports an internal error. No input, however malformed, may make the crate panic.

### Wire conventions

- **Decimals are exact.** An input decimal is a JSON number or a JSON string, read from its text
  and never through `f64` (`wire::Dec`). An output decimal is always a JSON string, because
  `JSON.parse` would turn a number back into a float.
- **Tariff values** (input values, the record, the bill, the trace, a test's values) are
  JSON typed by tag where JSON has none, as `formula-language.md` "Wire values" specifies and
  `src/tariff/wire.rs` implements:
  - `Bool` and `Text` are JSON `true`/`false` and strings;
  - `Int` and `Number` are `{"$n": "1200"}`;
  - money, quantities, rates and percents are `{"$q": "2.35", "u": "$/mi"}`,
    `{"$q": "1200", "u": "lb"}`, `{"$q": "18", "u": "%"}`, `{"$q": "450", "u": "$"}`;
  - `Date` and `DateTime` are `{"$date": "2026-10-22"}` and
    `{"$date": "2026-10-22T08:00:00-05:00"}` (RFC 3339 with the offset the value carries);
  - `Time` (a time of day, no date and no offset) is `{"$time": "17:30:00"}`, always written
    with seconds; `"17:30"` is read too;
  - an enum variant is a JSON string (`"Delivery"`), read as the declared enum;
  - `none` is JSON `null`;
  - lists and records are arrays and objects; a record key never starts with `$`.

  A value is decoded against the type it must have, and anything else is refused with its JSON
  path (`shipment.stops[0].weight`): a bare JSON number where a number is needed, a unit of
  another dimension, a variant the enum lacks, a missing or unknown record field, `null` where
  the type is not optional, a fraction where an `Int` is needed, nesting past 64 levels. A
  record field whose type is optional may be left out. Encoding is canonical: decimals
  normalised (`"2.5"`, never `"2.50"`), date-times with seconds and their offset, times with
  seconds, and every object's keys sorted by name.
- **Text positions** are UTF-16 code units: `{"from": 3, "to": 6}`, half-open. Inside the crate
  everything is a byte `Span`, converted at the edge by `span::Utf16Index`.
- **User-facing text** is a `Message`: `{"message": "unknown name {0}", "args": ["foo"]}`. The
  template is English ICU with positional placeholders; it is also the catalog key. The browser
  passes it to `translate(message, ...args)`; Go passes it to `shared/i18n`. An argument is
  text, a count (for plurals) or a nested `Message` (a phrase such as a value's description or a
  band label), which the host translates first and substitutes. Templates are
  declared with `message!("...")`, and `task i18n` extracts them into the Go scope and the web
  app's `web` bundle.
- **Names** on the wire are camelCase.

## Tariff

Tariff is the rating language. [formula-language.md](formula-language.md) is its reference: the
files, types, units, the bill, the data roots, the standard library and the wire values. This
section is the module's side: what each `tariff.*` operation takes and answers, the gate in
front of every run, the on-demand table protocol, the budgets and the error codes. Every shape
below is copied from the operation's module (`src/tariff/<area>/ops.rs`, whose header states
it) or from a corpus case.

The corpora are under `testdata/tariff/`, one file per topic: `syntax/*.txt` (read by
`src/tariff/syntax/tests/`), `check/*.json` (`src/tariff/check/tests.rs`; `check/ops.json`
pins `tariff.check`'s and `tariff.catalog`'s responses), `eval/*.json`
(`tests/tariff_eval_corpus.rs`, which runs each case through `tariff.evaluate` or `tariff.test`)
and `editor/*.json` (`src/tariff/editor/tests.rs`). A corpus response is matched by inclusion:
a case names only the keys it pins.

### What every operation shares

- **Strict input.** Every input object refuses a key it does not know: `{"programs": [],
  "extra": 1}` is `invalid_input`, as is a `kind` the operation does not take.
- **`kind`** is optional wherever a source is: `tariff`, `library` or `condition`. With it, the
  loader refuses a file of another kind (`file-kind`); without it the file's header decides.
- **`context`** is what a file is read with besides its source (`src/tariff/context.rs`), the
  same shape for `tariff.check`, `tariff.evaluate`, `tariff.test`, `tariff.scaffoldTest` and
  the editor operations, so a host builds it once:

  ```json
  {"libraries": [{"path": "org.fees", "version": "v3", "source": "library \"org.fees\"\nfn fee() -> Money = $5\n"}],
   "tables": [Table],
   "codes": [{"code": "DET", "description"?, "active": true}],
   "settings": {"distanceUnit": "mi" | "km"}}
  ```

  Every field may be left out. `libraries` are the organisation libraries a file may `use
  org.<name>`; `version` is echoed back in the imports `tariff.check` reports. `tables` are the
  rate tables (below); the checker reads only their shapes. `codes` absent means the host has
  not loaded them, so charge codes go unchecked (the editor's first paint); `[]` means there
  are none. `settings.distanceUnit` is `mi` (the default) or `km`, the unit `shipment.distance`
  is stored in.
- **Positions** are UTF-16 code units, half-open `{"from", "to"}`, in the source they are
  about; an editor operation's `offset` is one too.
- **A finding** has one wire shape everywhere (`WireDiagnostic` in `src/tariff/diagnostic.rs`):

  ```json
  {"range": {"from": 62, "to": 66}, "severity": "error", "code": "unknown-name",
   "message": {"message": "…", "args": […]},
   "fixes": [{"label": {"message": "Change to {0}", "args": ["`miles`"]},
              "changes": [{"range": {"from": 62, "to": 66}, "insert": "miles"}]}],
   "related": []}
  ```

  `severity` is `error`, `warning` or `info`; only an error stops a file from saving, previewing
  and rating. A fix's changes never overlap. `related` is the other places a finding refers
  to, each `{range, message}` (a duplicate import's "First imported here"). A finding inside an
  imported library also carries `"module": "org.bad"`, and its ranges are in that library's
  source.
- **A `doc`** is a string or a `Message` (`src/tariff/docs.rs`): a string is the author's words
  (a doc comment in a tariff or an organisation library, a table's name, a charge code's
  description), shown as written; a `Message` is text the module ships (the schema's and the
  built-ins' docs, and the standard library's doc comments, whose templates are the comments
  with their whitespace collapsed, held in `src/tariff/stdlib/docs.rs`), which the host
  translates.
- **Values** are wire values (see "Wire conventions").

### `tariff.check`

Checks a file with the libraries it imports, against the schema, the tables' shapes and the
charge codes. Input: `{"source", "kind"?, "context"?, "values"?}`. From `check/ops.json`, which
pins only the fields shown:

```json
{"op": "tariff.check", "input": {
  "source": "tariff \"Dry van TL\"\n\nuse org.fees.{ fee }\n\n/// Paid per loaded mile.\ninput rate: Money/mi = $2.35/mi\ninput minimumCharge: Money\n\ncharge \"Linehaul\" (quantity: shipment.distance, rate: rate)\nfor stop in shipment.stops {\n  charge \"Stop {stop.sequence}\" as DET = fee()\n}\nminimum minimumCharge",
  "context": {"codes": [{"code": "DET", "active": true}],
              "libraries": [{"path": "org.fees", "version": "v3", "source": "library \"org.fees\"\nfn fee() -> Money = $5\n"}]}}}
```

```json
{"ok": {
  "ok": true,
  "header": {"kind": "tariff", "name": "Dry van TL"},
  "diagnostics": [],
  "inputs": [
    {"name": "rate", "doc": "Paid per loaded mile.", "type": "Money/mi",
     "descriptor": {"kind": "quantity", "dimension": "Money/Distance", "unit": "$/mi"},
     "required": false, "default": {"$q": "2.35", "u": "$/mi"}, "range": {"from": 75, "to": 79}},
    {"name": "minimumCharge", "doc": null, "type": "Money", "descriptor": {"kind": "money"},
     "required": true, "default": null}
  ],
  "imports": [{"path": "org.fees", "version": "v3"}],
  "charges": [{"label": "Linehaul", "code": null}, {"label": "Stop {stop.sequence}", "code": "DET"}],
  "outline": {
    "items": [{"kind": "header", "name": "Dry van TL"}, {"kind": "input", "name": "rate"},
              {"kind": "input", "name": "minimumCharge"},
              {"kind": "charge", "name": "Linehaul", "depth": 0},
              {"kind": "charge", "name": "Stop {stop.sequence}", "code": "DET", "depth": 1},
              {"kind": "total", "name": "minimum"}],
    "folding": [{"kind": "block", "range": {"from": 216, "to": 266}}]
  }
}}
```

- `ok` is true when no module has an error finding and no contract value is refused.
- `header` is the root's `{kind, name}`, or `null` when it has none.
- `diagnostics`: the loader's and the parser's findings first, then the checker's, each sorted
  by where it starts.
- `inputs` are the root's, in declaration order: `name`, `doc` (the doc comment as written, or
  `null`), `type` (display text), `descriptor` (for building the form: `money`, `quantity` with
  `dimension` and `unit`, `list` with `item`, `optional` with `inner`, `record` with `name` and
  `fields` (each `{name, doc, type}`), `enum` with `name`, `variants` and `ordered` (true for
  `FreightClass` and `Weekday`, whose variants have an order, so a form may offer a range of
  them), and `int`, `number`, `percent`, `text`, `bool`, `date`, `dateTime`, `time`),
  `required` (no default), `default` (a wire
  value, when the default folds), `constraint` and the name's `range`.
- `constraint` is the input's `where` clause, folded: `null` when it has none,
  `{"range": {"min": wire | null, "max": wire | null, "maxInclusive": bool}}` (the minimum is
  always inclusive; an open end is `null`), or `{"oneOf": [wire, …]}`. For a `List<T>` input it
  bounds each item. A constraint that does not fold is a checker error, never sent. A range's
  ends compare as the language orders them: numbers, money and quantities across units, dates,
  date-times and times in time order, `FreightClass` and `Weekday` variants in declaration
  order, and text in Unicode scalar order, character by character (`"30000"..="39999"`), so a
  form that validates as the user types must compare text the same way (JavaScript's `<` on
  strings compares UTF-16 code units, which differs only past the Basic Multilingual Plane).
- `imports` is every library the program loads, directly or not, dependencies first, with its
  pinned `version`: the context's for an organisation library, the crate's own for `std.*`.
- `outline.items` are `{kind, name, code?, depth, range, selection}`, where `kind` is also
  `property` for a property, `choice` for a `best` or `worst` (its label as `name`) and `option`
  for each of its options (one `depth` deeper); `outline.folding` the
  foldable `{range, kind}`s. `charges` lists every `charge` of the root, nested ones included,
  with its label as written, its code and its range.

**Contract values.** `values` is `{"rule": {name: wire value}, "agreement": {…}}`, the input
values a rate agreement rule and its agreement set. Each is decoded against its input's type
with the evaluator's own codec (`src/tariff/wire.rs`), so the checker and the evaluator agree on
every value. A value that does not fit, a name that is no input, or a required input that
neither sets is an `input-value` error finding with a `path` and no range (from `check/ops.json`):

```json
{"severity": "error", "code": "input-value", "path": "rule.drops",
 "message": {"message": "{0} is not a whole number", "args": ["1.5"]}, "fixes": [], "related": []}
```

Other paths read `rule.rat` (with "did you mean `rate`"), `agreement.breaks[2].extra` (a field
`Break` does not have) and `rule.rate` (`{0} has no default, so the rule or the agreement must
set it`). A value outside the input's constraint is an `input-value` finding naming the bounds,
at path `rule.rate` (a list input's item at `rule.stops[2]`), with values written the way
interpolation writes them: `{"message": "{0} must be from {1} to {2}; {3} is outside it", "args":
["`rate`", "$0.50/mi", "$15.00/mi", "$235.00/mi"]}`. The other forms are `lo..hi` ("{0} must be
at least {1} and below {2}; {3} is outside it"), `lo..` ("{0} must be at least {1}; {2} is below
it"), `..hi` ("{0} must be below {1}; {2} isn't"), `..=hi` ("{0} must be at most {1}; {2} is
above it") and a list ("{0} must be one of {1}; {2} isn't", with {1} written `[Standard,
Guaranteed]`). The same wording refuses a default outside its constraint (code `constraint`),
a rating whose stored value falls outside it (an `input` error) and a test's `given input.x`. An `input-value` finding makes `ok` false like any
error.

**The codes.** The loader's and the parser's:

| Code | What |
|---|---|
| `syntax` | the text does not follow the grammar, often with a one-line fix |
| `limit` | past a limit (below) |
| `header` | a file that does not start with its header, a header off the first line, a second header |
| `misplaced` | a `test` or `private` inside a block, a lambda outside a list method's arguments, a `fail` that is neither a statement nor after `??` |
| `doc` (warning) | a `///` doc comment attached to nothing; the fix makes it a plain comment |
| `file-kind` | the file is not the `kind` the request asked for |
| `module-path`, `unknown-module`, `unknown-import`, `private-import`, `import-not-library` | a `use` that names no module, no declaration, a `private` one, or a file that is not a library |
| `duplicate-import`, `import-collision`, `import-cycle`, `duplicate-declaration` | a name imported twice, imported and declared, modules that import each other, a name declared twice |
| `library-item` | an input, binding or charge in a library |

The checker's are listed in the header of `src/tariff/check/mod.rs`: unknown names (with "did
you mean" fixes), `unknown-code` and `inactive-code`, `type-mismatch`, `unit-mismatch`,
`optional` (with `?? zero`, `if let` and `require let` fixes), scope rules, call rules, rate
table rules, charge and total-clause rules (a line limit or total clause whose minimum is
above its maximum is `total-order` or `charge-option`), `match` exhaustiveness and overlap, the
warnings (`unused`, `unreachable`, `no-charge`, `needless-fallback`, `bounds` for code an
input's `where` bounds make dead, `bill-order` for a `bill.charge(…)` of a line billed only
later, and the rest), `misplaced` for a `best` used as a value anywhere but the whole value of
a `let`, `var` or assignment and for `bill` or `review` outside a tariff's own code,
`unknown-option` for a `chosen` compared with a label that is no option, `constraint`
(including a test's `given` outside the bounds), and `input-value`.

### The gate

`tariff.evaluate`, `tariff.test` and `tariff.scaffoldTest` check a file before they run it
(`gate` in `src/tariff/eval/ops.rs`). A file that does not load or does not check never runs:
the answer is an evaluation error with code `check` whose message is the first error finding,
with its range when the finding is in the root file. A file that checks clean runs with the
checker's types and its declared input types, so the checker's promises hold at run time. From
`eval/errors.json`:

```json
{"status": "error", "error": {"code": "check",
  "message": {"message": "`expect` can only be used inside a `test` or `property` block"}}}
```

### `tariff.evaluate`

Rates a batch of programs in one boundary crossing: the rate engine sends a template's tariff
and every conditional accessorial's condition together. Request (the op module's header):

```json
{"context": Context, "record": {…} | null, "budget": {"steps": 5000} | null, "trace": false,
 "programs": [{"id": "rule-1", "source": "tariff \"…\" …", "kind": "tariff",
               "inputs": {"rate": {"$q": "2.35", "u": "$/mi"}}, "record": {…},
               "exchange": {"rate": "1.3502", "currency": "CAD"}}]}
```

- **Programs.** `kind` is `tariff` or `condition` (optional). Programs with the same source and
  kind share one load and one check. Each program is independent: one program's error never
  affects another's result, and results come back in request order with their `id`.
- **The record** is the record being rated, `{"shipment", "lane", "customer", "fuel",
  "rating"}` in wire values, decoded strictly against the schema: a missing or unknown root or
  field (an optional field may be left out, as `null`), or a value of the wrong type or unit, is
  a `record` error naming its path ("The record does not fit the schema at {0}: {1}"). The host
  sends `shipment.accessorials` (the user-entered additional charges only, `[{"code",
  "quantity": {"$n"}}]`), each stop's `legDistance` (a distance in the organisation's unit,
  `null` on the first stop) and `location.category`, `location.facilityType` (variant names or
  `null`) and `location.requiresAppointment`, and `rating.holidays` (`[{"date": {"$date"},
  "name"}]`: the organisation's holidays, recurring ones expanded, from the year before the
  rating date's year to the year after). `record` absent or `null` rates against the sample
  record; a program's own `record` overrides the shared one.
- **Inputs** are a map from input name to wire value. A value is decoded against the input's
  declared type; one that does not fit is an `input` error at the input's declaration, and a
  name that is no input is an `input` error with no range ("This tariff has no input named
  {0}"). An input with no value and no default refuses to rate, with the crate's message ("The
  input {0} has no value and no default") at its declaration.
- **Exchange.** `{rate, currency}` converts a rated bill into the billing currency: `rate` is
  units of `currency` per unit of the rating currency, an exact decimal (a JSON number or
  string). A rate that is not above zero is an `argument` error.

Response:

```json
{"results": [{"id": "rule-1", "status": "rated" | "refused" | "error" | "needs",
  "bill": {"lines": [{"kind": "charge", "label": "Linehaul", "code": null, "freight": true,
                      "amount": {"$q": "1908.2", "u": "$"}, "quantity": …, "rate": …,
                      "limit": null | {"kind": "minimum" | "maximum", "computed": {"$q": "1175", "u": "$"}},
                      "on": null | "freight", "range": {"from": 0, "to": 40}}],
           "notes": [{"text": "…", "range": …}], "reviews": [{"text": "…", "range": …}],
           "freightTotal": …, "accessorialTotal": …,
           "total": …, "converted": {"currency": "CAD", "rate": "1.3502", "lines": […],
           "total": …}},
  "value": true,
  "refusal": {"message": "No fuel price on …" | {"message": "…", "args": […]}, "range": …},
  "error": {"code": "divideByZero", "message": {"message": "…", "args": […]}, "range": …},
  "needs": [{"table": "Zones", "keys": ["752"]}],
  "choices": [{"label": "Linehaul", "kind": "best", "range": …, "chosen": "Flat",
               "options": [{"label": "Per mile", "range": …, "status": "lost",
                            "subtotal": {"$q": "1175", "u": "$"}, "refusal": null},
                           {"label": "Flat", "range": …, "status": "chosen",
                            "subtotal": {"$q": "900", "u": "$"}, "refusal": null},
                           {"label": "Volume", "range": …, "status": "refused",
                            "subtotal": null, "refusal": {"message": "Volume rates start at 6,000 lb", "range": …}}]}],
  "trace": [… ] | null, "lookups": […]}]}
```

- **Choices** are every `best` and `worst` the rating evaluated, in evaluation order (a choice
  inside a loop appears once per iteration), at most 200. `kind` is `best` or `worst`;
  `chosen` is the kept option's label, or `null` when every option refused. Each option's
  `status` is `chosen`, `lost` or `refused`, with its exact `subtotal` (money) or its `refusal`
  (shaped like a result's). A tariff with no choices sends `"choices": []`; a condition never has
  any. The trace also records each choice as a `choice` event (`label`, `chosen`, `subtotal`).

- **The four statuses.** `rated`: a tariff's `bill`, or a condition's `value` (true or
  false). `refused`: a `refusal`, whose `message` is the author's interpolated text (`fail`,
  `require … else`, `?? fail`), or a `Message` when the crate wrote it (an input with no value),
  with the range of the statement. `error`: an evaluation `error` (below). `needs`: the
  `needs`, cells the request did not carry (below). Every result has `trace` and `lookups`.
- **The bill.** `lines` in billing order. `kind` is `charge`, or `minimum`, `maximum` and
  `rounding` for the total clauses' adjustment lines, which carry no label (the host labels
  them in the reader's language) and no range. `freight` is true for an uncoded line;
  adjustments are uncoded, so they count as freight. `amount` is money; `quantity` and `rate`
  are the wire values a `charge … (quantity:, rate:)` gave, else `null`. `limit` is `null`
  unless the charge's `minimum:` or `maximum:` changed its amount: then `kind` names the limit
  and `computed` is the amount it replaced (rounded to cents like the line), so the quantity
  and rate no longer multiply to `amount` and the invoice says the minimum (or maximum)
  applied. `on` is `freight` for the adjustment of a `minimum … on freight` or `maximum … on
  freight` clause and `null` on every other line. A zero line is dropped and noted on the
  trace. `notes` and `reviews` are `{text, range}` in the order they ran: a `review` asks the
  host to hold the rated bill for approval with its reasons (a losing `best` option's are
  dropped, as its notes are). `freightTotal`, `accessorialTotal` and `total` are money; the lines
  always add up to the total. With `exchange`, `converted` gives each line's amount in the
  billing currency (in `lines`' order, rounded to cents with the bill's round mode), the total
  rounded with the bill's mode and increment, and any difference on the rounding line's
  converted amount, or as one more trailing entry when the bill has no rounding line, so the
  converted lines add up to the converted total.
- **The trace** is `null` unless the request sets `trace: true`. It is the root file's events
  in execution order (`src/tariff/eval/trace.rs`; a call into a library shows as the binding it
  produced), each with its `range`: `bind` and `assign` (`name`, `value`, `display`), `branch`
  (`taken`: the branch's index, or `null` when nothing ran), `arm`, `loop` (`name`,
  `iterations`), `iteration` (`name`, `index`, `value`, `display`), `charge` (`label`, `code`,
  `value`, `display`, and `limit`: `null`, or `{kind, computed}` with the exact amount a line
  limit replaced), `dropped` (a zero line), `note` and `review` (`text`), `refusal`, `choice`
  (`label`, `chosen`, and `subtotal`: the kept option's exact subtotal, the value a `let`
  binds, or `null`) and `lookup` (an index into `lookups`). It keeps at most 2,000 events (then one `{"kind": "truncated"}`), 50 iterations
  per loop, 50 items of a list and 500 characters of display text. `display` is the text
  interpolation would write.
- **Lookups** are every table read, at most 500: `{table, op, keys, match, adjusted, value,
  display, range}`. `op` is `index`, `cell`, `band` or `interpolate`; `keys` are the wire keys
  given; `match` is the matched cell's position on each axis (`{"key"}` or `{"min", "max"}`,
  `max` `null` for an open band), or `null` on a miss; `adjusted` says an overflow policy moved
  a key into its band.

### Rate tables and the on-demand protocol

The host sends each active rate matrix as a table under `context.tables`
(`src/tariff/eval/tables.rs`):

```json
{"code": "Zones", "valueType": "Money/cwt",
 "axes": [{"match": "Exact", "kind": "Zip3", "normalization": "Zip3"},
          {"match": "Range", "kind": "WeightBreak", "unit": "lb", "overflow": "ClampToTopBand"}],
 "cells": [{"at": [{"key": "752"}, {"min": "0", "max": "5000"}], "value": "18.50",
            "minimum": "75.00", "deficit": false}],
 "loaded": [["752"]]}
```

| Field | Meaning |
|---|---|
| `code`, `name` | the matrix code a tariff names (`tables.Zones`); the matrix's name, for hover |
| `valueType` | what a cell holds: `Money`, `Money/<unit>`, `Percent`, `Number` or `Int` |
| `axes` | one to four, in position order: `{match, kind, label?, unit?, normalization?, overflow?}` |
| `axes[].match` | `Exact` or `Range` |
| `axes[].kind` | Go's `ratematrix.DimensionKind`; it gives the key's type (a weight for `WeightBreak`, a distance for `Distance`, a length for `LinearFeet`, an `Int` for `PieceCount`, a `Number` for `Quantity`, the `FreightClass` enum for `FreightClass`, text otherwise), and a `FreightClass` axis reads a key as one of the eighteen NMFC classes (`85`, `"77.5"`, `"Class92_5"`, the variant) before normalising |
| `axes[].unit` | the unit a range axis's bounds are stored in (absent: `lb` for weight breaks, `mi` for distance, `ft` for linear feet; the Go host sends the organisation's distance unit); a quantity key is converted into it first |
| `axes[].normalization` | an exact axis's key rule: `None` (default), `Trim`, `Upper` (trim, upper-case) or `Zip3` (trim, upper-case, first three characters); stored and looked-up keys pass through the same rule |
| `axes[].overflow` | a range axis's rule for a key no band covers: `Error` (default) misses, `ClampToTopBand` prices anything at or past the top band's floor at the top band, `Nearest` also takes the bottom band below the first floor and the closest band in a gap |
| `cells` | `{at, value, minimum?, deficit?}`: one position per axis, `{key}` on an exact axis and `{min, max?}` on a range axis (half-open `[min, max)`, no `max` is open-ended); exact decimals |
| `loaded` | the exact-axis key combinations whose cells `cells` holds, one normalised key per exact axis |

Axes narrow the cells in position order. Cells are kept sorted by their band floors, stably, so
when overlapping bands leave several cells the one with the lowest floors wins, then the one
sent first. An exact-axis key may be a list (`tables.Zones[lane.origin.zones,
lane.destination.zones]`): the combinations are tried in order, the first axis outermost. A
malformed table (a cell without one position per axis, an exact position without a key, a band
without a floor, a value type a table cannot hold, a code sent twice) fails only the programs
that read it, with code `table`, as does reading a code the request did not send.

**Cells load on demand**, because a table can hold hundreds of thousands of cells:

- `cells` absent means the request carried none of the table's cells; `cells` with `loaded`
  absent means `cells` is the whole table; `cells` with `loaded` means it holds exactly the
  cells of those exact-key combinations (a combination listed with no cells is a real miss).
- A lookup of a combination the request did not carry is a **need**: it gives a provisional
  miss and the program keeps running, so one pass finds every need it can. A program that
  recorded needs answers `status: "needs"` instead of a bill or a refusal, its `needs`
  deduplicated: `{"table", "keys"}`, one normalised key per exact axis. A table with no exact
  axis has nothing to load by key, so the host always sends it whole.
- The host loads those cells and calls again. From `eval/tables.json`, a table whose cells for
  `A` were loaded and whose `B` cells were not:

  ```json
  {"source": "tariff \"t\"\nlet a = tables.Grid[\"A\", 1 lb] ?? $0/cwt\nlet b = tables.Grid[\"B\", 1 lb]\nnote \"{a}\"",
   "expect": {"status": "needs", "needs": [{"table": "Grid", "keys": ["B"]}]}}
  ```

The Go host's side is under "Go callers: Tariff" below: at most four loads, three of cells and
then the still-needed tables whole. `tariff.test` and `tariff.scaffoldTest` follow the same
protocol, so a test that reads a matrix runs on the server and in the browser alike.
`tariff.check` and the editor read shapes only, so a host sends them tables without cells.

### `tariff.test`

Runs a file's own `test` blocks against the sample record (`src/tariff/eval/tests.rs`). Input:
`{"source", "context"?, "names"?, "budget"?}`; `names` runs only those tests, and the budget
applies to each test. Output:

```json
{"tests": [{"name", "range", "kind": "test" | "property",
  "status": "passed" | "failed" | "error" | "needs",
  "failures": [{"range", "message", "values": [{"range", "display", "value"?}]}],
  "bill": … | null, "error": … | null,
  "cases": 24, "skipped": 0, "counterexample": … | null}],
 "needs": [{"table": "Grid", "keys": ["A"]}],
 "coverage": {"regions": [{"range", "kind", "hits"}], "reached": 18, "total": 21} | null}
```

- **Properties** are in `tests` in source order with the tests, `kind: "property"`; a test has
  `kind: "test"`, `cases` 1, `skipped` 0 and `counterexample` `null`. A property's `cases` is how
  many cases rated, `skipped` how many an `assume` skipped. A failed property's `failures`,
  `bill` and `error` are its counterexample's, and `counterexample` is:

  ```json
  {"values": [{"target": "shipment.weight", "display": "999 lb", "value": {"$q": "999", "u": "lb"}}],
   "monotone": {"target": "shipment.weight", "expression": "total", "direction": "nondecreasing",
                "before": {"at": {"display": "999 lb", "value": …}, "result": {"display": "$310.00", "value": …}},
                "after": {"at": {"display": "1000 lb", "value": …}, "result": {"display": "$250.00", "value": …}}} | null,
   "refusal": {"message": …, "range": …} | null,
   "test": "test \"heavier never bills less: 999 lb\" {\n  given shipment.weight = 999 lb\n  …\n}\n"}
  ```

  `values` holds each `vary` target with the value it had in the shrunk case (for a `monotone`
  failure, the `before` case). `monotone` is set when a `nondecreasing` or `nonincreasing`
  expectation failed; `refusal` when the case refused to rate. `test` is a `test` block that
  reproduces the case (its `given`s, the varied values as `given`s, and the property's
  `expect cond` lines; for a monotone failure two `given` sets cannot share a test, so it pins
  the `before` case's `expression` with `expect`), formatted, for the Studio's "Save as test".
- **Budgets.** Each test, and each property case, runs under the request's per-test budget, and the
  whole run shares one step meter of 32 times the per-test step limit. Once it is spent, every
  test and property not yet finished ends `error` with code `budget` ("The tests used up the
  file's step budget ({0} steps) before they finished"), with no counterexample. A case that
  errors on its own keeps its counterexample, unshrunk. Displays are the crate's interpolation
  text (`1000 lb`, `$250.00`); a listed `vary` item displays as written (`[pickup(), delivery()]`).
- **Coverage** is the root file's regions with their `hits` across every test and property case
  of the run, in source order. `kind` is `if` (a branch block of an `if` or `else if`), `else`,
  `arm` (a `match` arm), `loop` (a `for` body), `option`, `charge`, `review`, `refusal` (a
  `fail`, the refusing side of a `require` or of `?? fail`), `function` (a function body), or
  `applies` and `notApplies` (a condition's final expression, entered when a test sees it give
  `true` or `false`; both have that expression's `range`); `range` is the region's block or
  statement. `reached` counts regions with `hits > 0`. `coverage` is `null`
  when the file does not check, when any test ended in `needs`, or when `names` ran only some
  tests.

- A test starts from the sample record and the inputs' defaults, applies its `given`s in order
  (a record path, `input.name` or a table cell `tables.Code[k, …]`), runs the file, then runs its
  `expect`s with `charge(CODE)`, `charge("Label")`, `chosen("Label")`, `total`, `lines` and
  `reviews` (the review reasons, a `List<Text>`) in scope, or a condition's `applies`. A table the
  test gives cells of holds only those cells for the test (`Tables::with_given`): its lookups
  never read the request's cells and never record needs. A test in a library runs its body with
  no rating.
- A failing `expect` reports each operand's `display` and wire `value`, so the author sees what
  the tariff computed. An evaluation error makes the status `error`, with the error. `bill` is
  the test's bill, when it rated.
- A file that does not check answers `{"tests": [], "needs": [], "error": {"code": "check", …}}`.
- **Needs.** Tests read tables under the on-demand protocol: a lookup of cells the request did
  not carry is a provisional miss and the test keeps running, so one run finds every need it
  can, from its `given`s, the rating and its `expect`s. A test that recorded needs ends with
  status `needs` (no failures, bill or error, since the miss may have caused them), and the
  response's `needs` is every test's needs, each once. The host loads them and runs the tests
  again, exactly as it does for `tariff.evaluate` (from `eval/runner.json`):

  ```json
  {"tests": "tariff \"Grid\"\n…\nlet rate = tables.Grid[lane.destination.zip3, w] ?? fail \"No rate\"\n…",
   "expect": {"tests": [{"name": "to A", "status": "needs", "failures": []}, …],
              "needs": [{"table": "Grid", "keys": ["A"]}, {"table": "Grid", "keys": ["B"]}]}}
  ```

### `tariff.scaffoldTest`

"Save as test" (`src/tariff/eval/scaffold.rs`). Input: `{"source", "context"?, "record": {…} |
null, "inputs"?, "name"}`. It rates the tariff against the record and writes a `test` block that
pins the result: a `given` for every record field that differs from the sample record (stops and
commodities through the test constructors, leaving out what the constructors fill in, a stop's
location written field by field), a `given tables.Code[k, …] = value` for every cell the rating
found by `[…]`, `.cell(…)` or `.band(x)` (a band as its floor; `.interpolate` leaves its table
live), so the saved test reads no live matrix data, a `given input.x` for every input that
differs from its default, an `expect charge(CODE)` (or
`charge("Label")` for freight) per charge and `expect total`, or `expect fails "…"` when the
rating was refused. Output: `{"test": "test \"…\" {\n given …\n expect …\n}\n"}`; or `{"test":
null, "error": …}` when the tariff does not check, the record does not fit or the rating fails;
or `{"test": null, "status": "needs", "needs": […]}` when it needs cells the request did not
carry.

### `tariff.format`, `tariff.tokens` and `tariff.setName`

`tariff.format`: `{"source"}` gives `{"formatted": "…"}`, the file in the one canonical style
(`src/tariff/syntax/format/mod.rs`: two-space indentation, at most one blank line kept, numbers
and text as written, comments kept where they are, a line past 100 columns broken one item per
line). A file with syntax errors is left alone: `{"formatted": null, "diagnostics": [...]}` with
those findings. The result is checked before it is returned: it parses without errors to the
same tree with the same comments, and formatting it again changes nothing.

`tariff.tokens`: `{"source"}` gives `{"tokens": [{"from": 0, "to": 6, "kind": "keyword"}, …]}`,
every token but newlines and the end of the file, classed for highlighting: `keyword`, `name`,
`function`, `field`, `type`, `code`, `number`, `unit`, `money`, `percent`, `date` (dates,
date-times and times of day: `@2026-10-22`, `@17:30`), `text`, `escape`, `interpolation`,
`punct`, `comment`, `doc` or `error`. The lexer reads every `camelCase` word as a name; the
tokens beside it class it further, so a name before `(` is a `function` (called, or declared
with `fn`) and a name after `.` or `?.` is a `field` (`src/tariff/syntax/ops.rs`). Both read the
tokens alone, so a broken file still highlights.

`tariff.setName`: `{"source", "name"}` gives `{"source": "…", "change": {"range", "insert"}}`,
the file with its header naming `name` and the one edit that makes it, in UTF-16 code units. The
name is written as a text literal that reads back exactly (`\"`, `\\`, `\n`, `\t`, `{{`,
`}}`); a header without its name gets one after the keyword, and a file without a header gets
`tariff "…"` and a blank line at the top. A formula template's name is its header's, so the Go
host renames through it (Duplicate, Fork, Import) and the browser's name field writes the header
with it.

### The editor operations

Each takes the file being edited with its `kind` and `context` and a position or range, loads
and checks the file (it may be broken) and answers from what is known
(`src/tariff/editor/ops.rs`):

| Operation | Input | Output |
|---|---|---|
| `tariff.complete` | `{source, kind?, context?, offset}` | `{"range": {from, to}, "items": [{"label", "kind", "detail", "doc", "insert", "snippet", "sort", "range"?}]}`, in `sort` order; a snippet is written as LSP writes one (`$1` or `${1:placeholder}` tab stops, `$0` for the final cursor, `\$`, `\}` and `\\` for those characters), and a call's placeholders are its required parameters' types (`round(${1:Money})`, `trip(empty: ${1:Distance}, rate: ${2:Money/mi})`); an item's own `range` replaces the list's |
| `tariff.hover` | `{source, kind?, context?, offset, values?}` | `null`, or `{"range", "contents": {"signature", "type", "unit", "doc", "constraint"?: Message, "value"?: {"value", "display", "times"}}}`; `constraint` is an input's `where` bound in words ("from $0.50/mi to $15.00/mi"), kept apart from `doc`, which is the author's own text |
| `tariff.signature` | `{source, kind?, context?, offset}` | `{"signatures": [{"label", "doc", "parameters": [{"label": [from, to], "doc"}]}], "active": 0 \| null, "activeParameter": n \| null}` |
| `tariff.definition` | `{source, kind?, context?, offset}` | `{"targets": [{"module": null \| "std.ltl", "range", "selectionRange"}]}`; a target in a library has its ranges in that library's source |
| `tariff.references` | `{source, kind?, context?, offset, includeDeclaration?}` | `{"references": [{"range", "kind": "declaration" \| "read" \| "write", "module": null \| "org.fees"}]}`: every use of the name at `offset` in the file, in source order, and its declaration (unless `includeDeclaration` is false), which for an imported name is in the library's source with its `module`; `[]` when `offset` is on no name. `write` is a `var` assignment or a `given`/`vary` of an input |
| `tariff.actions` | `{source, kind?, context?, range: {from, to}}` | `{"actions": [{"id", "title": Message, "edits": [{"range", "insert"}], "select": {from, to} \| null}]}`: the refactorings available for the selection; `select` is the range, in the edited source, to select afterwards (the new name, to rename it) |
| `tariff.rename` | `{source, kind?, context?, offset, newName?}` | with `newName`, `{"edits": [{"range", "insert"}]}` or `{"error": Message}`; without it (prepare), `{"range", "placeholder"}` or `{"error": Message}` |
| `tariff.inlays` | `{source, kind?, context?, range: {from, to}, values?}` | `{"hints": [{"offset", "label", "kind": "type" \| "parameter" \| "value"}]}` |

**Promote to input** (`tariff.actions`, ids `promote-input` and `promote-input-all`). Offered
when the selection is, or lies inside, one literal a contract could set: a number with or
without a unit, money, a rate, a percent, text, a date or date-time, `true`/`false`, or an enum
variant. It is not offered inside a function, a test or property, a `match` pattern, an input's
default or constraint, a `const`, a type, a condition's final expression's pattern or a library.
The edits insert `input name: Type = literal` after the file's last `input` (after its `use`s, or
after the header and its doc, when there is none), with the literal's type (`$2.35/mi` is
`Money/mi`, `48 ft` is `Length`, `2` is `Int`), and replace the literal with `name`. The name
comes from where the literal is: a charge's `rate:` or `quantity:` option gives the charge label
in camelCase plus `Rate` or `Quantity`, a charge's amount the label plus `Amount` (`Flat rate`
gives `flatRateAmount`), a `let x = …` gives `x` plus `Default`, a named argument its parameter
name, anything else `value`; a name already in scope gets `2`, `3`, …. `promote-input-all` is
offered when the file has other literals with the same text and type in places promotion is
allowed, and replaces them all. Titles are `"Promote {0} to an input"` and `"Promote all {1}
uses of {0} to an input"`.

A completion item's `kind` is `keyword`, `name`, `input`, `function`, `method`, `property`,
`field`, `type`, `enum`, `variant`, `unit`, `code`, `table`, `module`, `snippet` or `value`.
`values` is the `trace` a `tariff.evaluate` of the same file returned
(`src/tariff/editor/values.rs`): hover and inlays show a binding's value from it (its last
value, and how many times it ran in a loop). A malformed trace is ignored, never a reason to
refuse the request. Signature help gives a parameter the sentences of its function's doc that
name it in backticks. Rename refuses keywords, the schema roots and the record's fields, tests
(renamed by editing their text), and a name the standard library or an organisation library
declares (rename it there); it refuses a new name of the wrong case for what it names, or one
that collides with a name already visible where the renamed name is declared or used.

### `tariff.catalog`

Input `{}`. Everything an author can name before declaring anything, with its docs
(`src/tariff/check/catalog.rs`); the editor's reference pane and Nova's prompt read it. Every
doc in it is a `Message`.

```json
{"modules": [{"path": "std.ltl", "doc", "declarations": [{"kind": "fn" | "const" | "type" | "enum",
               "name", "signature", "doc", "fields"? , "variants"?}]}],
 "types": [{"name", "doc"}],
 "methods": [{"receiver", "name", "kind": "property" | "method", "signature",
              "params": [{"name", "type", "default"}], "returns", "doc"}],
 "tableMethods": [{"name", "doc"}],
 "units": [{"unit", "dimension", "factor", "si"}],
 "keywords": […], "contextual": ["as", "cents", "to", "up", "down", "half", "even", "total", "lines", "applies"],
 "schema": {"roots": [{"name", "doc", "type", "fields"}], "records": [{"name", "fields"}],
            "enums": [{"name", "variants", "ordered"}]}}
```

The modules are `std.ltl`, `std.fuel`, `std.time`, `std.accessorials` and `std.tiers`, in that
order. A
schema field is `{name, type, descriptor, words, doc, fields}`. `keywords` are the lexer's
reserved words, never names; `contextual` are the lexer's contextual words (keywords only where
the grammar uses them, which `tariff.tokens` classes as `keyword` there and `name` elsewhere)
and the test-only names `total`, `lines` and `applies`, so a parameter may be called `to`.
`methods` ends with the shipment's own `requested(code: CODE) -> Bool` (receiver `Shipment`). A unit's
`dimension` is its base's name, `Percent` for `%` and `Count` for a count label. The catalog is static data;
the Go host caches it per module version.

### Limits

| Limit | Value | Past it |
|---|---|---|
| A file | 256 KiB | one `limit` error; nothing is read |
| Tokens | 50,000 | one `limit` error where they run out |
| Nesting depth | 128 levels | one `limit` error; the rest reads as the end of the file |
| Interpolation inside interpolation | 8 levels | a `limit` error on that text |
| A program (the root and every library it loads) | 64 modules, 1 MiB of source | a `limit` error on the `use` |
| A wire value's nesting | 64 levels | refused, as a value that does not fit |

The depth limit bounds every recursion over the tree, which is what keeps the checker, the
formatter and the evaluator within WebAssembly's stack.

### Budgets

The evaluator meters its work as it happens (`src/tariff/eval/budget.rs`), so a runaway
program stops at its limit, deterministically, in the browser and on the server alike:

| Budget | Default | Counts |
|---|---|---|
| `steps` | 1,000,000 | every node evaluated, and every element a list method or table lookup visits |
| `textBytes` | 1,048,576 | every byte of text the program builds; text read from the record is free |
| `items` | 100,000 | every list item and record field the program creates |
| `callDepth` | 64 | user functions running at once (the checker refuses recursion; this also bounds the native stack) |

A request's `budget` may lower any of them; a value above a default is ignored, so a budget is
never raised (`{"steps": 999999999999}` runs under the default). The limits apply to each
program, and each test, separately. Spending one is a `budget` error ("The tariff used up its
step budget ({0, plural, one {# step} other {# steps}}) before it finished"). On the server,
wazero also closes an instance when its call's context ends, so a deadline kills the work
instead of abandoning it.

### Evaluation errors

An evaluation error is `{code, message, range}`: a stable code the host may branch on, a
`Message`, and the range in the root file of the node that failed, or `null` before it is
placed (an unknown input). An error or refusal that leaves a call into a library is moved onto
the call, so the range is always in the source the author is looking at
(`src/tariff/eval/error.rs`).

| Code | When |
|---|---|
| `check` | the file does not load or check (the gate); the message is the first error finding |
| `internal` | the program did something its types rule out, which the gate makes unreachable for a file that checked |
| `budget` | a budget is spent |
| `input` | an input value that does not fit its declared type, or names no input |
| `record` | the record does not fit the schema |
| `divideByZero` | division or remainder by zero |
| `overflow` | a result past 28 significant digits, or a date past year 9999 |
| `conversion` | a unit conversion between dimensions, or one that does not fit |
| `argument` | an argument of the right kind with an unusable value: a rounding increment of zero, a negative square root, `avg` of an empty list, an exchange rate not above zero |
| `regex` | a `matches` pattern that does not compile or is too large |
| `table` | a rate table the request did not send, or one whose cells do not fit its shape |
| `negativeCharge` | a coded charge with a negative amount |
| `negativeTotal` | a bill whose freight lines or total come out below zero |

A refusal is not an error: it is the tariff declining to rate, with the author's words.

### Determinism

The module has no clock, no randomness and no I/O. `rating.date` is in the record the host
sends, so a tariff rates the same record the same way every time. The sample record (what a
request without a record, every test and the Studio's preview start from) is the schema's
defaults: an empty `New` shipment with one pickup and one delivery stop (no leg distances,
blank locations), no commodities and no requested accessorials, a blank lane, a sample customer,
no fuel price, and a `Sell` rating in `USD` on `2026-01-05T12:00:00Z` with no holidays
(`src/tariff/eval/record.rs`), so a test's answer never depends on the
day it runs. Wire values encode canonically and objects' keys are sorted, so the same answer
is the same bytes.

### Fuzz targets

`fuzz/` (cargo-fuzz, outside the services workspace, run with `cargo +nightly fuzz run <target>`)
holds four targets:

- `tariff_parse`: arbitrary text parsed and loaded as a program: a tree every time, the tokens
  covering the text in order, every diagnostic and fix inside it;
- `tariff_format`: formatted text parses to the same tree and formats to itself, or the file is
  refused;
- `tariff_check`: arbitrary text through `tariff.check`, always one JSON envelope;
- `tariff_evaluate`: arbitrary text rated against the sample record through `tariff.evaluate`
  with the trace on: one result for the one program, every time.

No input, however malformed, may make the crate panic.

## Operations

The other areas document their operations here: the input and output, with an example, and the
`testdata/` file that pins them.

### Charge allocation (`allocation.*`)

A shipment's freight and accessorials, or an order's charges, can be billed to several
customers. Each charge is divided by its allocation rows and every payer's invoice is the sum
of its shares. The server bills with these operations and the shipment form previews the same
invoices, so each takes every charge of one shipment, order or form in **one call**.
Pinned by `testdata/allocation.json`.

**Shared shapes.** A *charge* is

```json
{"kind": "Freight" | "Accessorial" | "OrderCharge",
 "total": "1000.01",
 "method": "Flat" | "PerUnit" | "Percentage", "amount": "75", "unit": 2,
 "allocations": [<row>, null, ...]}
```

Its total is `total` when given (freight, an order charge), or else its accessorial line priced
against the request's `base` (the shipment's freight; zero when absent):

- `Flat`: `amount × max(unit, 1)`;
- `PerUnit`: `amount × unit`, and zero when `unit < 1`;
- `Percentage`: `base × amount / 100`;
- anything else, or no method: zero.

A *row* is the GraphQL charge allocation:
`{"id"?, "billToCustomerId", "method": "Percent" | "Amount", "percent"?, "amount"?, "sequence"?}`.
`null` rows are skipped but keep their positions. Here and in every area below, a `null` list or
string reads as empty, as GraphQL writes an unset one, and a decimal that is `null`, missing or a
blank string is "not filled in"; any other text must be a decimal, or the request is
`invalid_input`. So is a charge without a valid `kind` in `allocation.split`, and arithmetic past
28 significant digits, which no stored column reaches.

**Ordering.** Rows are read by `sequence` (missing is 0), then by `id`. Rows without an ID (not
saved yet) sort before saved rows of the same sequence, where Go's empty ID sorts, and keep their
request order among themselves, so the browser and the server agree once the rows are saved.

#### `allocation.split`

Input: `{"defaultPayerId": "cus_1", "base": "1000", "charges": [<charge>...]}`.

```json
{"defaultPayerId": "cus_first", "charges": [{"kind": "Freight", "total": "1000.01", "allocations": [
  {"id": "chal_b", "billToCustomerId": "cus_second", "method": "Percent", "percent": 40, "sequence": 1},
  {"id": "chal_a", "billToCustomerId": "cus_first", "method": "Percent", "percent": 60, "sequence": 0}]}]}
```

Output: per charge its exact `total` and either `shares` (in split order) or `error` (a
`Message`); then the payers and whether billing is split, both `null` when any charge failed.

```json
{"charges": [{"total": "1000.01", "shares": [
   {"allocation": 1, "allocationId": "chal_a", "billToCustomerId": "cus_first", "method": "Percent",
    "amount": "600.01", "percent": "60", "partial": true},
   {"allocation": 0, "allocationId": "chal_b", "billToCustomerId": "cus_second", "method": "Percent",
    "amount": "400", "percent": "40", "partial": true}]}],
 "payers": [
   {"payerId": "cus_first", "freightAmount": "600.01", "accessorialAmount": "0", "totalAmount": "600.01"},
   {"payerId": "cus_second", "freightAmount": "400", "accessorialAmount": "0", "totalAmount": "400"}],
 "isSplit": true}
```

`allocation` is the row's index in that charge's `allocations`; an unallocated charge's one share
has neither `allocation` nor `allocationId`, and `method: null`.

The rules (`chargeallocation.ResolveShares`, `ResolveOrderChargeShares`, `decimalutils.AllocatePercent`):

- No rows: the whole charge, rounded half to even to cents, to the default payer at 100%.
- Every row must use the same method ("A charge must be split either by percent or by amount,
  not both"), that method must be `Percent` or `Amount` ("Allocation method must be Percent or
  Amount"), and every row must name a payer ("Bill-to customer is required").
- Percent rows must each be in (0, 100] and total 100 at six places (half away from zero);
  otherwise "Percent allocations for this charge must total 100". Every share but the last is
  `total × percent / 100` rounded half to even to cents; the last is the rounded total minus the
  others.
- Amount rows are rounded half to even to cents and must add up to the rounded charge; otherwise
  "Amount allocations for this charge must add up to {0}", with the unrounded total written by
  `StringFixed(2)` (half away from zero). An amount share's percent is shopspring's
  `amount × 100 / total` (the quotient rounded half away from zero to 16 places, then to 6), or
  `null` when the charge is zero. A percent share carries its row's percent; an unallocated
  share 100.
- A share is `partial` when it is not the whole rounded charge.
- Payers: freight shares sum into `freightAmount`, accessorial and order-charge shares into
  `accessorialAmount`. The default payer leads, even with nothing to pay; the others follow by
  ID. `isSplit` is true when any share is partial or goes to another payer, or more than one
  payer is billed.

#### `allocation.remainder`

What a split editor shows while somebody types, and whether the rows can be saved. Input:
`{"base"?, "charges": [<charge>...]}`; a charge with neither `total` nor `method` has an unknown
total. Output, per charge:

```json
{"charges": [{"method": "Percent", "allocated": "60", "remaining": "40", "complete": false,
  "over": false, "mixedMethods": false, "duplicatePayers": false, "missingPayers": false,
  "problem": {"message": "Percents must add up to 100"}, "valid": false}]}
```

- `method` is the first row's (`Percent` when it is not `Amount`); `mixedMethods` when any row
  differs from it.
- Percent rows: `allocated` is their exact sum, `remaining` is `100 − allocated`; `complete` when
  the sum rounds to 100 at six places and there is a row; `over` when it rounds above 100.
- Amount rows: `allocated` is their exact sum, `remaining` is `total − allocated` (`null` when
  the total is unknown); `complete` when their cents add up to the total's cents; `over` when
  they add up to more.
- `problem` is the first of: "Split every row by percent or every row by amount", "Each payer
  may appear only once", "Every row needs a payer", "Percent must be greater than 0 and at most
  100" / "Amount must be greater than zero", "Percents must add up to 100" / "Amounts must add
  up to the charge". No rows is no problem (the whole charge goes to the default payer), and
  amount rows against an unknown total are not judged on their sum. `valid` is
  `problem == null`.

#### `allocation.chargeTotal`

`{"base": "2450", "charges": [{"method": "Flat", "amount": 150, "unit": 2}, {"method": "Percentage", "amount": 10}]}`
→ `{"totals": ["300", "245"], "sum": "545"}`: each charge's exact total by the rules above, and
their sum (`AdditionalChargesTotal`).

#### `allocation.rebalance`

Amount splits that went stale because their charge changed (`FindStaleAmountSplits`), converted
to percents that keep each payer's proportion (`ConvertAmountSplitsToPercent`). Input as
`allocation.remainder`. Output per charge, and how many were converted:

```json
{"charges": [{"stale": true, "allocatedTotal": "2850", "chargeTotal": "3000",
  "percents": ["52.631579", "47.368421"]}], "converted": 1}
```

- A charge is stale when every row is an amount row and their cents do not add up to the
  charge's cents, exactly when `allocation.split` would refuse it. Otherwise
  `{"stale": false, "allocatedTotal": null, "chargeTotal": null, "percents": null}`.
- `percents` is aligned with the request's `allocations` (`null` for a `null` row). Every row but
  the last by sequence gets shopspring's `amount × 100 / sum` rounded half away from zero to six
  places, and the last gets the rest of 100. A stale split whose rows sum to zero or less cannot
  be converted: `percents` is `null` and it does not count as converted.

#### Where Go and TypeScript disagreed

| Rule | Go | TypeScript | Now |
|---|---|---|---|
| Money | exact decimals | floats and `Math.round(x × 100)` | exact decimals |
| `PerUnit` with no units | 0 | `amount × max(unit, 1)` | 0 (Go) |
| Unknown method | 0 | the amount | 0 (Go) |
| Missing unit | 0 | 1 | 0 (Go: the field's zero value; the form defaults it to 1 anyway) |
| Mixed methods, bad percents, amounts that do not add up | refused | not checked; shares computed anyway | refused (Go) |
| Rows without a payer | not reachable (validation refuses them first) | dropped from the split | refused with "Bill-to customer is required" |
| Row order | sequence, then ID | sequence, then position | sequence, then ID, unsaved rows by position |
| Percent complete | rounds to 100 at six places | rounds to 100 at four places | six places (Go): `33.3333333 × 3` is complete |
| Amount complete | `RoundBank(sum) == RoundBank(total)` | `Math.round` cents | cents of each share (see the fix below) |
| Duplicate payers | allowed, merged into one share | refused by the editor | refused by `allocation.remainder` only; `allocation.split` merges them as Go does |

Two Go rules were wrong and are fixed here. The Go callers now run the module, so the server
follows the fixed rules too (`decimalutils.AllocatePercent` and the resolver's amount check are
deleted):

- **The last percent share** was `total − others` before rounding, then rounded half to even.
  With a sub-cent total that loses a cent: 50/50 of `0.015` gave `0.01 + 0.00`, but the payer
  totals should add up to the charge's `0.02`. The last share is now `RoundBank(total) − others`,
  which is the same answer for every whole-cent total.
- **Amount rows** were accepted when their unrounded sum rounded to the charge, but each share is
  then rounded on its own: `0.005 + 0.005` passed for a `0.01` charge and billed nobody. Their
  cents must now add up.

The invariants are property tests in `tests/rules_allocation.rs`: shares always add up to the
rounded total, and every share but the last is within half a cent of its exact share. The last
share absorbs the others' rounding, so it is within one cent for two payers and within
`n × ½` cent for `n`; a largest-remainder rule would bound it by one cent for any `n`, but it
would move cents between payers on existing invoices, so Go's "remainder on the last row" stays.

**What replaces what.** `chargeLineTotal` → `allocation.chargeTotal`; `splitCharges` and
`summarizeBillingByPayer`'s sums → `allocation.split`; `allocationRemainder` and
`allocationsValid` (and the messages of `chargeAllocationsRefinement`) →
`allocation.remainder`; `AdditionalCharge.Total`/`AdditionalChargesTotal`,
the share builders behind `ResolveShares`/`ResolveOrderChargeShares` and
`AllocatePercent`/`SumEquals` → `allocation.chargeTotal` and `allocation.split`;
`FindStaleAmountSplits`/`ConvertAmountSplitsToPercent` → `allocation.rebalance`. The Go names
in that list that still exist (`ResolveShares`, `ResolveOrderChargeShares`, `AdditionalChargeTotals`,
`RebalanceAmountSplits`) are thin calls into `pkg/rules`; the rest are deleted. They stay native: `toMinorUnits` (money in cents for the credit memo
dialog and invoice line comparison, not an allocation rule), and the GraphQL shaping helpers
`nestChargeAllocations`, `toChargeAllocationInput`, `hasAnyAllocations`. Grouping allocations
by the charge they target (`TargetID`, the kind filters in `ResolveShares`) and payer
validation (`ValidateAllocations`: the customer exists, is active and belongs to the tenant)
stay in Go, since they need the shipment and the database.

### Report formatting (`report.*`)

How reports are built, run and exported is in [reporting.md](reporting.md). Every place a report
is shown renders values through the column's display contract
(`format.Resolved` in keel analytics). The browser (the builder preview, the result grid, the
Desk) formats through this module; the server (CSV, PDF, XLSX, digests) through keel's `format`
package. `testdata/report.json` pins both: the Rust tests run it here, and
`TestServerFormattingMatchesTheBrowsersContract` (`internal/infrastructure/reporting/render`)
runs its typed cases against keel. Change a case and both hosts move together. The cells where
keel still disagrees (the sign of a negative that rounds to zero, the space after a currency
code, `CA$`, a number too large to read exactly) are listed in that test's `knownDivergences`;
the test fails on any new disagreement and on a listed one that keel has since fixed.

#### `report.format`

Works on columns, because the server formats up to 100,000 rows and the browser a virtualised
grid. Each column has its resolved `display` (every field defaults to Go's zero value, and a
missing display renders raw), or a `resolve` input (as `report.resolve`) to resolve it in the
same call; a `valueType`; and its `values`.

```json
{"columns": [
  {"display": {"style": "currency", "decimals": 2, "grouping": true, "currency": "USD",
               "negative": "parentheses", "rules": [{"op": "gt", "value": 10000, "tone": "warning"}]},
   "values": [1082.0672643987443142, -1.005, null, 50000]},
  {"resolve": {"type": "epoch", "spec": {"dateStyle": "date"}}, "values": [1772587800]}]}
```

```json
{"columns": [
  {"values": [{"text": "$1,082.07"}, {"text": "($1.01)"}, {"text": ""},
              {"text": "$50,000.00", "tone": "warning"}]},
  {"values": [{"date": "1772587800"}]}]}
```

**Values.** `null` is empty; `true`/`false` are booleans; a JSON number is an exact decimal (one
too large to read is shown as written); an array or object reads as empty. A JSON string is text,
except that in a column whose `valueType` is `int`, `decimal` or `epoch` a string holding a
decimal is that number (GraphQL sends decimals as strings so they stay exact), and in a `bool`
column `"true"` and `"false"` are booleans. Both hosts know a column's catalog type, which is
the `valueType`.

**Dates.** A number in a date-style column comes back as `{"date": "<epoch seconds>"}` (the
integer part, truncated toward zero, as Go's `IntPart`), with its tone, for the host to write in
the report's time zone: Go's layouts on the server, `Intl` in the browser. The module has no time
zone data. A Go `time.Time` value never reaches the module: the host formats it directly.
A date bucket's value is the instant its period starts in the organization's zone (the compiler
emits `date_trunc(field, to_timestamp(col), zone)`, and drill-through inverts it in that zone),
so a host that wrote it in another zone would show the wrong period: the browser takes the zone
from the result (`ReportPreview.timezone`) and passes it to `reportCellText`.

**The order of decisions** (`Resolved.String`): empty → `nullText`; style `""` → the raw value
(a decimal normalised, a boolean as `true`/`false`, text as is); a number in a column with a
usable band → its range; a number in a date column → a date; any other number → the numeric
style; a boolean → the boolean style (whatever the column's style); text → `prefix + text +
suffix`, or `nullText` when empty.

**Numbers.**

- A percent is shifted two places; a duration formats as below.
- Compact notation picks T, B, M or K from the **unrounded** magnitude (`999999` with no places
  is "1,000K").
- The digits are rounded half away from zero to `decimals` (clamped to 0..10) exactly as
  written: `1.005` → "1.01", `2.675` → "2.68".
- The whole part is grouped by thousands when `grouping`.
- Assembly is `prefix + symbol + digits + compact suffix + % + suffix`. The currency symbols are
  `USD` and empty `$`, `CAD C$`, `MXN MX$`, `EUR €`, `GBP £`, `AUD A$`, `JPY ¥`; any other code
  is written followed by a space.
- A negative value is wrapped in parentheses when `negative` is `parentheses`, else prefixed with
  a minus. The sign is decided before rounding, so `-0.001` is "-0.00" or "($0.00)", as in Go.

**Durations.** The value times `durationUnit` (seconds, minutes, hours, days) is the seconds:

- `clock` (the default): `H:MM`;
- `compact`: "1d 2h 10m";
- `verbose`: "1 day 2 hours 10 minutes";
- `decimalHours`: hours to `max(decimals, 1)` places, grouped, then " h".

Clock, compact and verbose use whole seconds. A negative duration is signed after the prefix and
suffix are added.

**Bands** (`format.Band`). The value of a banded cell is its range's lower bound. A fixed
`width` gives `[value, value + width)`. Explicit `edges` give the range of the last edge at or
below the value (a value below the first edge is in the first range), and the top range is
open: "$5,000.00+". Both ends render through the column's display, with a trailing suffix written
once: "10,000 – 15,000 lbs". Bounds are exact decimals.

**Tones** (`ToneFor`). Rules are tried in order and the first match gives the tone: `gt`, `gte`,
`lt`, `lte`, `eq`, and `between` (inclusive). Only numbers are toned, compared exactly; a percent
column's values are compared times 100, so a threshold of 12 means 12%.

#### `report.resolve`

`{"columns": [{"type": "decimal", "hint": "money", "dateStyle"?, "band"?, "spec"?}]}` →
`{"displays": [<resolved display>]}` (`format.Resolve` on the server):

- **Baseline:** text, grouping on, USD, minus, standard notation, `dateTime`, `yesNo`, seconds,
  clock. `epoch` is date, `bool` is bool, `int` is a number with 0 places, `decimal` a number with
  2, and only number types take the hint:
  - `money`: currency, 2 places, parentheses;
  - `percent`: 1 place;
  - `duration`: 0 places;
  - `distance`: 1 place, " mi";
  - `weight`: 0 places, " lbs";
  - `count`: 0 places.
- A bucket's `dateStyle` applies to a date column.
- A band that is not empty is carried, even a malformed one, so validation can report it.
- **Spec:** a `style` different from the baseline's restyles first:
  - currency resets 2 places and parentheses;
  - percent resets 1 place;
  - duration resets 0 places;
  - number clears the suffix.

  Then every set field overrides. Decimals are clamped to 0..10, the currency is upper-cased
  (ASCII), and rules replace the baseline's.

Output decimals in a display (band width and edges, rule values) are strings, like every output
decimal.

#### Where Go and TypeScript disagreed

| Rule | Go | TypeScript | Now |
|---|---|---|---|
| Rounding | `StringFixed`, exact, half away from zero | `toFixed` on floats: `1.005` → "1.00", `2.675` → "2.67" | exact (Go) |
| Rule and band arithmetic | `float64` | `number` | exact decimals: a `0.1` band from `0.2` ends at `0.3` |
| A band that does not validate (one edge, a negative width, width and edges) | rendered anyway: one edge reads "500.00+" for every value, a negative width with no edges indexes past the end and panics | ignored | ignored (TS): Go was wrong |
| A band on a date, text or boolean column | the number renders as its range | only numeric styles | its range (Go) |
| A number in a text or boolean column | formatted as a number | written raw | as a number (Go) |
| A boolean in a numeric column | its boolean style | written raw | its boolean style (Go) |
| An empty string in a numeric column | `nullText` | `""` | `nullText` (Go) |
| A date inside a decimal epoch column | `IntPart` (truncated) | `new Date(seconds × 1000)` with the fraction | truncated (Go), formatted by the host |
| Objects and arrays | empty, so `nullText` | `JSON.stringify` | empty (Go) |
| Currency upper-casing | ASCII | Unicode | ASCII (Go) |

**What replaces what.** `formatReportValue` and `reportValueTone` → `report.format`;
`resolveReportDisplay` → `report.resolve` (or `resolve` inside `report.format`). On the server,
resolution and formatting are keel's (`format.Resolve`, `format.Formatter`). They stay native in
the browser:

- dates (Go time layouts; `formatReportDate` with `Intl` in the browser), fed by the `date` cells;
- `reportNumericValue` (float coercion for charts) and `isNumericDisplay` (a one-line style
  check for alignment).

### Lanes (`lane.*`)

A rate agreement rule names a scope for its origin and its destination, stores one lane key, and
the most specific matching rule prices a shipment once priority is equal. The lane editor scores
and keys lanes as they are typed, so it must build what the engine stores to the character.
Pinned by `testdata/lane.json`.

Rules are read in the GraphQL / Go JSON shape (`originScopeType`, `originScopeValue`,
`originCity`, the same for `destination`, `direction`, `status`, `priority`, `effectiveFrom`,
`effectiveTo` and the condition fields); other fields are ignored, so either host passes its rows
as they are. An empty or missing scope type is `Any`, the default a saved rule takes.

#### `lane.specificity`

`{"rules": [<rule>...]}` → `{"scores": [1843200, 1025023]}` (`ComputeSpecificity`):

- **Geography:** `(origin weight + destination weight) × 1024`. The weights are Any 0,
  Country 300, State 500, Zone 600, Radius 650, CityState 700, Zip3 800, Zip5 900 and
  Location 1000; an unknown type weighs 0.
- **Conditions,** each counted once when the rule names it:
  - commodities 512, freight classes 256;
  - tractor types, trailer types or equipment classes 128, once together;
  - service types 64, shipment types 32;
  - a weight bound 16, a distance bound 8;
  - service models 4, hazmat 2, temperature control 1.

  They sum to 1023, below one step of geography, which a compile-time assertion keeps true.
- A list counts when it is not empty (`null` is empty). A bound counts when it is not `null`,
  missing or blank.

#### `lane.keys`

```json
{"rules": [{"direction": "Bidirectional", "originScopeType": "Zip3", "originScopeValue": "60601",
            "destinationScopeType": "State", "destinationScopeValue": "ust_ga"}],
 "shipments": [{"origin": {"postalCode": "60601", "city": "Chicago", "countryIso3": "USA"},
                "destination": {"stateId": "ust_ga"}}]}
```

```json
{"rules": [{"origin": "Z3:606", "destination": "ST:ust_ga", "key": "Z3:606>ST:ust_ga",
            "stored": "Z3:606>ST:ust_ga", "reverse": "ST:ust_ga>Z3:606",
            "segments": [{"text": "Z3:606"}, {"text": "ST:ust_ga", "scope": "State", "id": "ust_ga", "prefix": "ST:", "suffix": ""}]}],
 "shipments": [{"originKeys": ["Z5:60601", "Z3:606", "CT:USA", "ANY"],
                "destinationKeys": ["ST:ust_ga", "ANY"],
                "candidates": ["Z5:60601>ST:ust_ga", "Z5:60601>ANY", "Z3:606>ST:ust_ga", "Z3:606>ANY",
                               "CT:USA>ST:ust_ga", "CT:USA>ANY", "ANY>ST:ust_ga", "ANY>ANY"]}]}
```

- **A rule's ends** (`Scope.Key`):
  - `ANY`, `LOC:<id>`, `ZN:<id>`, `ST:<id>`;
  - `Z5:<postal>`, `Z3:<first three of postal>`;
  - `CS:<state>|<CITY>`, `CT:<COUNTRY>`.

  An end has no key (`null`) when it is a radius, an unknown type, has no value (a city-state
  needs both), or is a postal prefix shorter than three characters. `key` is
  `origin>destination` or `null`; `stored` is the key or `RADIUS`, the sentinel a geospatially
  matched lane stores; `reverse` is the swapped key of a bidirectional rule. `segments` is the
  key parsed as `lane.parse` does.
- **Text:**
  - a postal code is trimmed, cut at the first dash unless the dash leads, and upper-cased, so
    inner spaces stay;
  - a city is folded to single spaces and upper-cased;
  - a country is trimmed and upper-cased.

  Upper-casing is Unicode's one-to-one mapping, as Go's `strings.ToUpper`: `ß` stays `ß`. A
  postal prefix is cut at a character boundary where Go would cut a byte.
- **A shipment end's keys** (`Place.MatchKeys`), most specific first:
  - location, then postal code and prefix;
  - city-state (both needed), then every non-empty zone;
  - state, then country;
  - always ending with `ANY`.

  `candidates` is the cross product, origin major (`LaneKeyCandidates`).

#### `lane.parse`

`{"keys": ["CS:us_tx|DALLAS>ZN:rzn_sw", "Z3:606>CT:USA"]}` →
`{"keys": [{"segments": [{"text": "CS:us_tx|DALLAS", "scope": "CityState", "id": "us_tx", "prefix": "CS:", "suffix": "|DALLAS"}, {"text": "ZN:rzn_sw", "scope": "Zone", "id": "rzn_sw", "prefix": "ZN:", "suffix": ""}]}, {"segments": [{"text": "Z3:606"}, {"text": "CT:USA"}]}]}`.

Splits each key at `>`. `LOC:`, `ZN:`, `ST:` and `CS:` ends reference a record (a city-state's
is the state before `|`); every other end, and one with an empty ID, is literal. A display
replaces `id` with the record's name: `prefix + name + suffix`.

#### `lane.coverage`

`{"rules": [<rule>...]}` → `{"issues": [{"index": 1, "kind": "duplicate", "other": 0, "message": {"message": "This lane is written exactly as narrowly as another one, so which of the two applies is decided by a tie-break rather than by you."}}]}`.

The engine orders matching rules by priority (higher first), agreement priority, specificity,
effective date and ID. Two active lanes (an empty status is active) that are reachable by the
same key (a bidirectional lane by its reverse too) with the same specificity, during overlapping
effective windows (`[effectiveFrom, effectiveTo)`, open-ended when `effectiveTo` is `null`),
never both apply:

- with different priorities the lower one is `shadowed`: "Another lane covers the same geography
  with a higher priority, so this one can never apply.";
- with equal priorities the later one is a `duplicate`.

Radius lanes are not judged. A lane gets at most one issue (shadowed before duplicate), and
`other` is the first lane it collides with. Lanes are bucketed by key and specificity, so only
identical lanes are compared.

This rule has no Go twin; it lives in the crate because it is built entirely from the crate's
keys and specificity, it would otherwise be the last copy of the key scheme in TypeScript, and
the server can run it over an import or a save to warn about dead lanes. Moving it fixed one
TypeScript bug and two blind spots:

- TypeScript flagged the *later* lane as shadowed even when the later lane had the higher
  priority. The lower-priority lane is now the shadowed one, wherever it sits.
- It compared lanes whose effective windows do not overlap (an amendment and its predecessor).
- It missed a bidirectional lane meeting a lane written the other way round.

#### Where Go and TypeScript disagreed

| Rule | Go | TypeScript | Now |
|---|---|---|---|
| An end with no value (`Location` with `""`) | no key: the rule stores `RADIUS` | an empty end: `">ANY"` | no key (Go) |
| A postal prefix shorter than three | no key | `Z3:60` | no key (Go) |
| Spaces inside a postal code | kept: `Z5:606 01` | removed: `Z5:60601` | kept (Go; the stored key must match) |
| A leading dash in a postal code | kept | the code becomes empty | kept (Go) |
| Spaces inside a city | folded to one | kept | folded (Go) |
| A whitespace-only value | keyed, `Z5:` | keyed, `Z5:` | keyed, as Go stores it |

**What replaces what.** `laneSpecificity` and `ComputeSpecificity` → `lane.specificity`;
`laneKeyPreview`, `ApplyLaneKey`, `ReverseLaneKey`, `Scope.Key`, `CityStateKey`, `LaneKey`,
`Place.MatchKeys` and `LaneKeyCandidates` → `lane.keys`; `laneKeyScopeIds` and the parsing in
`laneKeyStringDisplay` → `lane.parse`; `findCoverageIssues` → `lane.coverage`; `normalizePostal`
and the weight and scale constants → inside those operations. They stay native: `laneKeyDisplay`
and `laneSegmentsDisplay` (substituting translated record names into `segments`),
`laneEndDisplay` (translated labels around the ends `lane.keys` returns) and `laneDisplayLabel`
(translated labels only), `IsEffectiveAt` and
`MatchesDayOfWeek` and the rest of rule matching (server-side, against shipment facts). The Go
names in this list are deleted; `rategeo.MatchLane`, `rategeo.ScopeKeys` and
`rateagreement.ApplyLanes` call the module in their place.

### Rate matrices (`matrix.*`)

Pinned by `testdata/matrix.json`. Dimensions are
`{"position": 0..3, "matchMode": "Exact" | "Range", "kind": "Zone", "label": ""}`. Cells are the
GraphQL cell (`id`, `d0Key`..`d3Key`, `d0Min`..`d3Max`; other fields ignored). A blank or `null`
bound is no bound, and `null` entries are skipped. A position outside 0..3 is `invalid_input`. A
dimension's display label is its trimmed `label`, or else the English name of its kind (Zone,
Postal prefix, Postal code, State, Country, Weight break, Distance, Pieces, Linear feet, Freight
class, Equipment, Service, Custom, Quantity). A host that translates passes the translated name
as `label`.

**Bands** are half open, `[min, max)`, and a missing bound is no bound:
`matrix::band::contains(min, max, value)` implements it. Tariff's table lookups
(`src/tariff/eval/tables.rs`) use the same rule for the cells the host sends, and for
interpolation's segments; a table band always has a floor (a cell without one is a malformed
table), since sorting by floor is what makes the lowest band win where bands overlap.

A rate is read by a tariff, from a matrix sent as a Tariff table (above); there is no lookup
operation of its own.

#### `matrix.coverage`

`{"dimensions": [...], "cells": [...]}` → `{"issues": [{"kind", "position", "message", "bands"?, "slices"?}]}`.
This rule has no Go twin; the editor runs it (and the server can, before publishing a tariff).
The kinds:

- **`empty`**: "This matrix has no axes, so nothing can ever look a rate up in it." or "This
  matrix has no rates yet, so every lane pointing at it prices nothing."
- **`duplicate`**: "Two rates sit at {0}. The engine will take the one entered first and never
  mention the other.", reported once per coordinate. The coordinate is every axis's key or
  `min..max`, joined by " · ".
- For each banded axis, within each **slice** (the cells that agree on every other axis):
  - **`gap`** below the lowest band when it starts above zero: "{0} has no band below {1}, so
    anything smaller finds no rate.";
  - **`unbounded`** when nothing reaches upward forever: "{0} stops at {1}. Anything at or above
    that finds no rate — give the top band no upper bound to catch it.";
  - **`gap`** between bands: "{0} jumps from {1} to {2} with nothing in between.";
  - **`overlap`**: "{0} bands {1} and {2} overlap, so the same shipment matches both."

Bands are swept by lower bound (none first), each compared with the furthest reach so far.
Numbers are written exactly with English thousands grouping. An overlap's band arguments are
nested messages, the band labels ("{0} – under {1}", "{0} and over", "under {0}", "Any") with
their bounds as arguments, so a host that translates the warning translates each label too;
`bands` also gives both bands' bounds as decimals. The same finding in several slices is
reported once, with `slices` listing the other axes' coordinates (absent for a one-axis
matrix).

Moving it from TypeScript fixed three things. Its messages are TypeScript's English unchanged.

- TypeScript collected the bands of every cell together, so a zone-by-weight matrix whose zones
  break at different weights showed false overlaps, and a gap in one zone was hidden by another.
  Each slice is now checked on its own.
- It compared only neighbouring bands, so a wide band covering two narrow ones produced a false
  gap and a missed overlap.
- It skipped every band after an unbounded one or before a floorless one, missing their
  overlaps.

**What replaces what.** `ContainsQuantity` → `matrix::band::contains` (deleted from Go, as are
`SelectCell` and `ratematrix.LookupCell`: tariffs read matrices as tables);
`findMatrixCoverageIssues` → `matrix.coverage`. They stay native: the sheet
projection (`buildMatrixGrid`, `sliceBuckets`, `bucketKeyOf`, `cellCoordinate`,
`coordinateKey`), `axisLabel` and `bandLabel` for headers, which only lay cells out and label
them. A dimension's `KeyNormalization` and `RangeOverflow` stay settings of the matrix in Go;
the Tariff host sends them as each table axis's `normalization` and `overflow`
(`rating/app/tariff/tables.go`).

## The Tariff server API

The Studio checks and tests Tariff in the browser with the same module the server uses, so only
these facts cross HTTP (all under `/api/v1`; every range is the module's `{from, to}` in UTF-16
code units of the source):

- **`GET /formula-templates/context`** returns `{context, libraries, tablesStamp}`. `context` is the
  organisation's `rules.TariffContext` (settings, table shapes, accessorial codes and the source
  of every importable library, the latest approved version of each); `libraries` lists them as
  `{path, libraryId, name, description, versionNumber}`. The browser passes `context` to its own
  `tariff.check`, `tariff.test` and editor operations, and reads the language itself from
  `tariff.catalog`.
- **`POST /formula-templates/context/cells`** (formula-template read) takes `{needs, whole?}`,
  the `needs` a browser `tariff.test` answered (at most 200, each a table code and at most four
  keys), and answers the cells of those combinations, or with `whole: true` the tables they
  name entire, read at the organisation's rate table stamp:

  ```json
  {"stamp": "12:40:1767225600",
   "tables": [{"code": "Grid", "cells": [{"at": [{"key": "A"}, {"min": "0"}], "value": "20"}],
               "loaded": [["A"]]}]}
  ```

  A table the organisation does not have is left out; a whole table has `loaded: null`. It is
  a read (write coverage `read-only`, allowed on a read-only plan). The context endpoint's
  `tablesStamp` is the same stamp: the browser keeps loaded cells per stamp
  (`lib/tariff-table-cells.ts`), and an answer at another stamp means a matrix changed, so it
  reads the context again.
- **`POST /formula-templates/test`** (the preview) takes `{source, inputs?, shipmentId?}`, with
  `inputs` a map of wire values, and answers:

  ```json
  {
    "valid": true,
    "diagnostics": [],
    "inputs": [{"name": "rate", "doc": "...", "type": "Money/mi", "descriptor": {"kind": "quantity", "unit": "$/mi"}, "required": true}],
    "imports": [{"path": "org.accessorials", "libraryId": "fl_...", "versionNumber": 3}],
    "charges": [{"label": "Linehaul", "code": null, "range": {"from": 120, "to": 171}}],
    "bill": {"lines": [], "notes": [], "freightTotal": "865.2", "accessorialTotal": "0", "total": "865.2"},
    "refusal": null,
    "error": null,
    "trace": [],
    "lookups": [],
    "tests": [{"name": "...", "range": {"from": 300, "to": 340}, "status": "passed", "failures": []}]
  }
  ```

  - `diagnostics` are `tariff.check`'s findings, with the latest approved version of each
    imported library. A source with an error finding is not rated: `bill`, `refusal`, `error` and
    `trace` are null, and `tests` is empty.
  - Exactly one of `bill`, `refusal` (`{message, range}`, the source's `fail` or `require`) or
    `error` (`{code, message, range}`) is set. `error.code` is the module's evaluation code, or
    `timeout` when the preview's two seconds ran out, or `invalid` when the inputs do not fit
    the declarations.
  - Without `shipmentId` the preview rates the sample record; with one it rates that shipment's
    record, which needs `shipment:read`.
  - `tests` are the file's own `test` blocks, run by `tariff.test`.
  - A source over 64 KiB or more than 100 inputs is a 400 validation problem.
- **`POST /formula-templates/scaffold-test`** takes `{source, inputs?, shipmentId, name}` and
  answers `{test}`: a `test` block written by `tariff.scaffoldTest` from that shipment's facts,
  for the person to paste into the source ("Save as test").
- **Saving** a template (create, update, import, rollback, fork, a schedule of billed content)
  or a library runs `tariff.check` (kind `tariff` or `library`) and `tariff.test`. An error
  finding refuses the save with a validation error on `source` carrying the finding's range, so
  the editor underlines it; the template stores the checker's `inputs` (each with its display
  `type` and structured `descriptor`), `imports` and `testCount`. Submitting needs a clean
  check and every test passing; approving pins each `org.*` import to the library version
  approved at that moment and runs the dependents gate (below).
- **`/formula-libraries`** (REST writes; reads are GraphQL `formulaLibraries`, `formulaLibrary`,
  `formulaLibraryVersions`, `formulaLibraryReviews`) holds an organisation's libraries, each a
  `library "org.<name>"` file whose first line must name it.

## Hosts

Both hosts load the same committed bytes and speak the ABI above; neither knows anything about
an operation beyond its name. Typed calls per area wrap the generic call in modules of their
own beside it (Go: next to `pkg/rules`; browser: beside `runtime.ts`), each decoding into its
own types.

### The Go host (`services/tms/pkg/rules` on `pkg/wasmmodule`)

```go
err := rules.Call(ctx, "allocation.split", input, &output)  // sonic in and out
raw, err := rules.CallRaw(ctx, "tariff.catalog", []byte("{}")) // pre-encoded input, raw `ok` out
info, err := rules.Version(ctx)                              // wasmmodule.VersionInfo{Version, Operations}
```

The package functions use `rules.Default()`, the process-wide `*rules.Runtime`.
`rules.New(wasmmodule.Options{PoolSize, RecycleMemoryBytes, MemoryLimitBytes})` builds another
(tests do). Everything below is `pkg/wasmmodule`, which `pkg/x12` ([edi-x12.md](edi-x12.md)) runs
on too: a module package is a `wasmmodule.Spec` and typed calls on a `Runtime` embedding
`*wasmmodule.Runtime`. The API is generic-free, so typed callers build on it without spreading generic
code. `CallRaw` is for input that is already JSON, such as a cached rate table: the request is
spliced, not re-encoded, and the output comes back as the raw JSON of `ok`.

**Compiling.** The module is compiled once per process, lazily and safely under concurrency
(`sync.OnceValues`): by the warm-up, or by the first call. A compile failure is kept for the
life of the process, and every call then fails with `ErrUnavailable`. The runtime uses wazero's
compiler where the platform has one and its interpreter otherwise, with
`WithCloseOnContextDone(true)`.

**The compile cache** is `os.TempDir()/trenova-rules/<sha256 of rules.wasm>`, or
`$TRENOVA_RULES_CACHE_DIR/<sha256>`, so a restart or another test binary does not compile
again, and a new module never reads an old one's entries. wazero runs the machine code it finds
there, so the directory must be private to the process's user: created `0700`, not a symlink,
not writable by group or others, owned by the user. Otherwise the host compiles in memory and
logs a warning.

**The instance pool.** At most `GOMAXPROCS` instances (`Options.PoolSize`), created as calls
need them. An instance runs one call at a time. A caller waits for a free one under its own
context. An instance is discarded and replaced after:

- a trap;
- its context closing it mid-call;
- an unreadable response;
- growing its linear memory past 64 MiB (`Options.RecycleMemoryBytes`), since linear memory
  never shrinks.

Each instance is also capped at 256 MiB (`Options.MemoryLimitBytes`): an allocation past it
traps instead of growing. A request over 64 MiB (`wasmmodule.MaxRequestBytes`) is refused before
it runs.

**Timeouts.** The host adds none of its own: the caller's context bounds both the wait and the
run. A context that has already ended fails before anything compiles or waits. A deadline that
passes mid-call closes the instance (wazero checks the context while the code runs), so the
work stops instead of being abandoned. The evaluator's budgets bound a program inside that.

**Errors** are sentinels, wrapped with the operation's name:

| Error | Means | The caller |
|---|---|---|
| `ErrRequest`, as `*wasmmodule.RequestError{Op, Code, Detail}` | the module refused the request (`malformed`, `unknown_op`, `invalid_input`), or the host did (`too_large`) | has a bug: return an internal error, never show the detail |
| `ErrTrap` | the module panicked or answered with something that is not an envelope; logged at error with the operation, never the payload; the instance is gone | returns an internal error and never retries |
| `ErrContextDone` | the context ended while waiting for an instance or running; wraps `ctx.Err()` | treats it as any cancelled work |
| `ErrUnavailable` | the module failed to compile or instantiate, or the runtime was closed | returns an internal error; start-up already logged why |

Typed callers map these to `errortypes` at their own edge.

**Metrics.** `pkg/wasmmodule` cannot import the metrics package, so it reports through a small
`wasmmodule.Observer`, implemented by `metrics.WasmModule` (`metrics.NewWasmModule`, one per
module; the EDI module's are `trenova_edi_module_*`) (README in
`internal/infrastructure/observability/metrics`):

| Metric | Labels |
|---|---|
| `trenova_rules_calls_total` | `op`, `outcome` (`ok`, `request_error`, `trap`, `context_done`, `unavailable`) |
| `trenova_rules_call_duration_seconds` | `op`; includes the wait for an instance |
| `trenova_rules_instances_in_use` | |
| `trenova_rules_instances_discarded_total` | `reason` (`trap`, `context_done`, `memory`) |

`op` is the operation's name. A name the module does not know, or one not shaped like
`area.name`, is counted as `unknown`, so a caller bug cannot grow the label set.

**Warm-up.** `infrastructure.WasmModulesModule` (`internal/bootstrap/infrastructure/wasmmodules.go`,
in the core options both the API and the worker start with) gives each default runtime (rules
and edi) the logger and its metrics. At start it warms each in the background: compile,
instantiate and call its version operation, within two minutes. Start-up never waits for it. A failure is logged at error
level and every call reports it as `ErrUnavailable`.

### Go callers

Every Go caller of the allocation and lane rules goes through a typed wrapper in
`pkg/rules` (`allocation.go`, `lane.go`; Go calls no `matrix.*` operation, which
only the browser's matrix editor runs). The wrappers send decimals exactly (`decimal.Decimal` marshals as a JSON
string), read output decimals back from strings, and check that the module answered for as many
items as it was asked about. User-facing messages decode into `rules.Message`, whose arguments
may be nested messages: `shared/i18n` translates an argument that is an `i18n.Localizer` in the
reader's locale, so `Message.ValidationError` and `Message.BusinessError` translate all the way
down. No Go code computes these rules any more.

| Package | Op | One call per |
|---|---|---|
| `operations/domain/shipment`: `ResolveShares`, `ResolveOrderChargeShares`, `BillsWholeChargeToPayer` | `allocation.split` | shipment or order: every charge and its allocation rows in one request |
| `operations/domain/shipment`: `AdditionalChargeTotals`, `ChargeTotalsByCharge` | `allocation.chargeTotal` | shipment: every accessorial line priced at once |
| `operations/domain/shipment`: `RebalanceAmountSplits` | `allocation.rebalance` | shipment edit |
| `rating/domain/rateagreement`: `ApplyLanes` | `lane.keys`, `lane.specificity` | save: every rule of the agreement, the amendment or the imported sheet |
| `rating/domain/ratezone`: `RateZone.StampMembers`, through `rategeo.ScopeKeys` | `lane.keys` | zone save: every member |
| `places/domain/rategeo`: `MatchLane` (from `rateengine.RateContext.LaneKeys` and `tariff.RecordSource.LaneZones`) | `lane.keys` | rating request: the bare ends' keys for zone membership, then the candidate lanes once the zones are known |

The allocation callers above them (invoice create, consolidate and reconciliation, the billing
queue's preview, review, reassign and payer share, invoice adjustments, the charge allocation
repository, the operations allocation check, billing readiness and transfer plans, the order
service and the GraphQL shipment and charge allocation mappers) take a `context.Context` and
return an error, and make one call per shipment, invoice or request, never one per charge.
Storage-bound validation (`ValidateAllocations`, `TargetID`, grouping by charge) stays in Go.

**Report rendering.** The CSV, PDF and XLSX renderers and the delivery digest format on the
server with keel's `format` package, column by column over each batch; the module is not called
from Go for reports.

**Lanes on save.** `rateagreement.ApplyLanes` stamps `LaneKey` and `SpecificityScore` on a batch
of rules before validation and before any insert: the service's create, update and amend plans,
the rate import's staging (which then reports a row whose end has no key), and the seeds. A rule
written without it is refused by `BeforeAppendModel` (`ErrLaneNotApplied`), as a zone member
without its match key is (`ErrMatchKeyNotApplied`). `lane.coverage` is not run on the server:
rate agreement validation has no warning channel, only errors, and dead or duplicate lanes are
legal. The lane editor runs it and shows each issue on its lane.

**What stays in Go.** Dates (Go layouts and zones), `Resolved.ExcelFormat` with the currency
symbols Excel number formats need, `Spec.Validate`, `Band.Validate`, and `Band.Bounds`,
`IsLowest` and `IsWhole`, which the query compiler and drill-through use to filter rows rather
than to label them.

### Go callers: Tariff

The typed calls are in `pkg/rules/tariff.go` (`CheckTariff`, `EvaluateTariffs`, `TestTariff`,
`FormatTariff`, `TariffCatalog`, `ScaffoldTariffTest`) and the wire values in
`pkg/rules/tariff_value.go`; `pkg/tarifftypes` holds what the TMS stores (`Inputs`, `InputDecl`,
`Import`, the decimal `Bill`) and compares (`DeclsEqual`, `ImportsEqual`, `ValuesEqual`). The host
of the language is `internal/modules/rating/app/tariff`. Authoring (`tariff.AuthoringModule`)
owns `check.go`, `test.go`, `catalog.go`, `resolver.go`, `libraries.go`, `describe.go` and
`errors/`; rating owns the record (`record/`, and `recordsource.go`, which adds the lane's zones
and the fuel price), the tables (`tables.go`, over `pkg/ratetablecache`), codes, settings, the
rater and the context source, behind `tariffports.ContextSource`, `tariffports.Rater` and
`tariffports.RecordBuilder`.

**The checker.** `tariff.Checker.CheckSource` assembles the context and makes one `tariff.check`
call. Libraries come two ways:

- `LibrariesLatest` (drafts, saves, the preview): the latest approved version of each `org.*`
  library the source imports, found by reading the source's `use org.<name>` lines and resolved
  transitively in batched rounds, one query per round (at most 32 libraries and 8 rounds);
- `LibrariesPinned` (approved versions, billing): exactly the versions an approved version
  pinned, read in one query by `(library, version)` and memoised per request.

Tables, codes and settings come from `ContextSource`, which sends table shapes only (cells
load on demand, below). `SourceCheck.Imports` turns the module's imports into pins (`{path,
libraryId, versionNumber}`). `RunTests` and `TestChecked` (a source that does not check runs no
tests; `SourceCheck` carries the tenant its context was built for) run `tariff.test`, and
`ScaffoldTest` runs `tariff.scaffoldTest`, both loading the cells the module `needs` exactly as
the rater does (below), so saving, submitting, readiness, the preview and "Save as test" all
pass a tariff that reads a matrix. `TableCells` serves the browser's loads
(`POST /formula-templates/context/cells`, under "The Tariff server API").
`Catalog` caches `tariff.catalog` per module version.

**The rater.** `tariff.Rater.Rate` (the `tariffports.Rater` that `rateengine/pricing.go` and
the preview, backtest and impact call) rates with one `tariff.evaluate` request, repeated only
to load cells: the template's tariff as program `tariff`, with its inputs filtered to the
declared ones and the exchange rate, and every conditional accessorial's condition as program
`condition:<id>`, all against one record from `RecordSource` and one context. A condition that
does not come to `true` withholds its charge, and one that does not rate is logged. A tariff's
`error` is returned as a business error carrying its code (an `input` error as a validation
error on `inputs`); a refusal comes back in the result.

**Cells on demand.** `Tables.Shapes` sends a table with no exact axis whole (its cells read once
per lookup stamp and cached in `pkg/ratetablecache`), and a table with an exact axis with
`cells` absent and `loaded: []`. When a result answers `needs`, the rater loads them and asks
again: `Tables.LoadCells` reads each needed combination's cells (from the cache, else one query
per table for every missing combination's stored keys, then cached per combination, an empty
answer included) and appends them and the combinations to `cells` and `loaded`. The rater loads
cells after each of its first three evaluations that need them; after a fourth it loads each
still-needed table whole (`LoadWhole`, `loaded` absent), and a fifth evaluation that still needs
cells is `ErrUnresolvedNeeds`. The loop is one `cellLoad` (`rating/app/tariff/cells.go`) that the
rater, the test runs and "Save as test" share; `ContextSource.TableStamp` gives the
organisation's lookup stamp the browser keeps its loaded cells by.

**The record.** `RecordSource.Build` fills the record from a shipment the rating hydrator
loaded (`ratinghydration`: one batched read per relation for every shipment in a batch,
stop locations' categories included). `shipment.accessorials` are the shipment's own
additional charges only (never system-generated or tariff lines, which would feed back on a
re-rate), their codes from the loaded charge or the organisation's catalog; each stop's
`legDistance` is its move's stored distance when the move has two stops, `0` when it starts
where the previous move ended, else `null`; `rating.holidays` are the organisation's holidays
(not blackouts) from the year before the rating date to the year after, recurring ones
expanded into each year, read once per organisation and year through
`ratetablecache.Memoize`, the same per-batch memo the tables use.

**Reviews hold billing.** A rated bill's `reviews` (`tarifftypes.Bill.Reviews`) apply the
charges but hold the shipment out of billing: `RatingDetail.AwaitsReview` is true until a
person approves through `approveShipmentRatingReview` (`shipmentratingreviewservice`, billing
queue update permission, refused to agents, audited on the shipment), which stores a
`RatingReviewApproval` naming the reviews it approved. A re-rate keeps the approval while the
reviews are the same, holds again when they change and clears the hold when there are none.
While held, approving the queue item (by hand, in bulk or automatically), every invoice path
and invoice runs refuse (`shipment.RatingReviewHoldError`), and the queue item carries a
`ratingReviewHold` and a `RatingAwaitingReview` check, as detention holds do.

**The resolver.** `TemplateResolver.ResolveBilling` (the `tariffports.Resolver` rating calls)
returns a template's billing version as of a date with its source, stored input declarations,
pins and pinned library sources, through the per-request `effectiveversioncache`; a template
never approved bills nothing, an active template whose versions predate snapshots bills its
row. `ResolveLibraries` serves `tariffports.LibraryResolver`.

**Templates and libraries.** `formulatemplateservice` and `formulalibraryservice` save through
the checker (kind `tariff` and `library`), refuse on the first error finding with its range,
store the derived inputs, imports and test count, and version on a material change (source or
imports). Approval runs the readiness gates again, pins the imports, and for a template runs the
**dependents gate**: when the declared inputs differ from the approved version's, every distinct
set of stored inputs on the rules, agreements and shipments that use the template
(`rateagreementports.TemplateInputsLister`, grouped in SQL, at most 2,000 sets) is checked with
`tariff.check`'s `values`, and approval is refused, naming an example and a count, when one no
longer fits. The billing control's fallback template is checked with no inputs.

**Choices.** `RateResult.Choices` carries the module's `choices`; the rate engine maps them
with `tarifftypes.ChoicesFrom` (each option's exact subtotal, in the tariff's rating currency,
or its refusal in the reader's language) onto the quote's trace (`ratetypes.Trace.Choices`),
`RatedShipment.Choices` and the shipment's `RatingDetail.Choices`, which GraphQL serves as
`ShipmentRatingDetail.choices`. The Studio preview answers the module's own `choices`, ranges
included. A rating that stops on an `input` error (a stored value outside a constraint added
later) is a validation error on `inputs` with the module's bounds message; the quote records it
in the reader's language.

**Properties and coverage.** Properties are tests: `TariffTestRun.Passed` is false when one
fails, so save-for-review, submit and approval refuse it, naming each failure through
`tariff.FailingTests` (the varied values, the monotone pair or the refusal). Readiness for
templates and libraries adds a `coverage` check that warns and never blocks, and a `coverage`
summary (`tariff.SummarizeCoverage`: reached, total and up to 50 unreached regions with their
range, line and source line). Nova's `SummarizeTests` carries the same.

**The preview, backtest and impact** rate a candidate source through `tariffports.Rater` with a
two-second timeout per rating (the backtest's past shipments come from
`tariffports.RatedShipments`, the record of a shipment from `tariffports.RecordBuilder`).

**Errors.** `tariff/errors` maps the module's answers:

| Module outcome | Becomes |
|---|---|
| an error finding of `tariff.check` | validation on `source` with the finding's range (`CheckRefusal`); an `input-value` finding is validation on its `path` |
| a test that fails or errs at submit or approval | validation naming the failing tests |
| an evaluation error (`TariffEvalError`) | the preview reports it as `error`; "Save as test" as `Evaluation`, an `EvaluationError` validation on `source` with its code and range; rating as the rater's business error carrying the code |
| a refusal (`TariffRefusal`) | the preview reports it as `refusal`; rating records its text, a crate-written refusal in the reader's locale (`i18n.FromContext`), as the rating's refusal (`rateengine`) |
| `ErrRequest`, `ErrTrap`, `ErrUnavailable` | 500, logged; the detail is never shown |
| `ErrContextDone` | the context's own error, or `timeout` when the preview's deadline ran out |

`api/helpers/classifier.go` classifies the two error types as validation, so REST answers a
`validation-error` problem (HTTP 400) whose error carries `range`, and GraphQL a field error;
messages translate in the reader's locale through `shared/i18n`.

**Nova** reads the language with `formula_describe_language` (the catalog plus the
organisation's context, `tariff.Findings` and `tariff.SummarizeTests` turning the module's UTF-16
ranges into lines and character columns, and cutting excerpts, through
`stringutils.ByteOffsetOfUTF16`, so text after `°F`, `é` or an emoji is placed right and never
split), checks with `formula_check`, runs tests with `formula_run_tests`, rates with
`formula_preview`, and proposes whole files with `formula_propose` and `formula_library_propose`,
which are shown only when the source checks clean and has at least one test, all passing.

### The browser loader (`client/packages/shared/src/rules`)

```ts
const rules = await loadRules();                   // load.ts: Promise<RulesRuntime>
rules.call("tariff.check", { source, context });   // runtime.ts: synchronous, JSON in and out
rules.callRaw("tariff.tokens", sourceJson);        // JSON text in, JSON text out
rulesSync().version();                             // throws RulesNotLoadedError before the load
const runtime = useRules();                        // hooks/use-rules.ts: suspends until loaded
```

**Loading.** `loadRules()` returns one memoised promise. A failed load is forgotten, so the
next caller tries again. It fetches `new URL("./rules.wasm", import.meta.url)`, which Vite
emits as `assets/rules.<hash>.wasm`, never inlined (it is far over `assetsInlineLimit`) and
never in the startup bundle. When the server sends `application/wasm`, the module compiles
while it downloads (`instantiateStreaming`). Otherwise it is compiled from the whole file, as
it is when streaming refuses a response it has not read. A network or HTTP failure is a
`RulesLoadError`; a corrupt file is a `WebAssembly.CompileError`. Passing bytes
(`loadRules(bytes)`) skips the network; tests do this.

**Synchronous access.** A loaded `RulesRuntime` calls the ABI directly, with `TextEncoder` and
`TextDecoder`:

- `call(op, input)` returns the parsed output as `unknown`, for a typed wrapper to narrow;
- `callRaw(op, json)` takes and returns JSON text;
- a request error throws `RulesRequestError {op, code, detail}`;
- a trap throws `RulesTrapError`, after the runtime has already replaced the instance from the
  compiled module.

An instance whose memory grew past 64 MiB is replaced too. `rulesSync()` returns the loaded
runtime or throws `RulesNotLoadedError`. `isRulesLoaded()`, and `subscribeRules` with
`getRulesSnapshot` (for `useSyncExternalStore`), report readiness.

**`useRules()`** (`@trenova/shared/hooks/use-rules`) returns the runtime. Until the module has
loaded, it suspends the component with React's `use()` on the memoised promise, so render code
under a `Suspense` boundary (with the screen's skeleton) can then call it synchronously. Once
loaded it never suspends. A failed load reaches the error boundary, and the next render that
asks loads again.

**Preload.** `main.tsx` calls `preloadRulesWhenIdle()` (`rules/preload.ts`) after it renders.
It waits for the load event and then for idle time (`requestIdleCallback`, with a ten-second
timeout; two seconds after load without it), then imports the loader and loads the module. A
screen that needs it later finds it ready. A failure is logged and left to the next caller.

**Tests.** Both Vitest setups (`client/packages/shared/src/test-setup.ts`,
`client/apps/web/src/test-setup.ts`) run `await loadRules(RULES_WASM_BYTES)` before any test,
so tests call `rulesSync()` and render `useRules()` without waiting. The bytes come from
`rules/test-module.ts`, which Vite reads from disk at transform time (`rules.wasm?inline`;
both configs list `**/*.wasm` in `assetsInclude`). A test of loading itself takes a fresh
loader with `vi.resetModules()`.

**The service worker** answers `assets/rules.<hash>.wasm` from its asset cache after the first
fetch, like any content-hashed build file. The cached response keeps `application/wasm`, so it
compiles while it streams, and it is dropped when a release no longer ships it. It is not among
the files the worker caches at install, since the page loads it on idle.

**Hosting.** The page's Content-Security-Policy needs `'wasm-unsafe-eval'` in `script-src`. It
does not depend on `'unsafe-eval'`, which can then be removed. `connect-src 'self'` covers the
fetch. The file must be served as `application/wasm`:

- Caddy does from Go's MIME table (`deploy/Caddyfile`, which has the CSP);
- the Vite dev server does;
- Cloud's Worker (`trenova-cloud-deploy`, `edition/cloudflare/web/worker.ts`) must add the same
  `script-src` source.

### Browser callers

The typed wrappers beside the loader are `tariff.ts` (with `tariff-wire.ts` for Tariff's wire
values, whose unit table, `tariff-units.generated.ts`, `build-wasm.sh install` writes from the
crate's `SIMPLE_UNITS` through `tests/client_units.rs`, a test that fails when the committed
copy drifts, since formatting a value must not wait for the module), `allocation.ts` (with `allocation-form.ts` for zod), `report.ts`, `lane.ts` and
`matrix.ts`; `wire.ts` shapes form values for them, sending a value that is not a decimal yet
("abc", `NaN`, "1.") as "not filled in" so a half-typed form never refuses the request. Decimal
outputs stay strings in every client type and become numbers only where a formatter or a chart
needs one. Messages are shown with `translateMessage(message, t)`
(`@trenova/shared/i18n/message`), which translates nested messages first; `task i18n-check`
reports `t(x.message, ...x.args)` and a literal handed to it.

Each screen calls once per question for everything it shows, and waits in one of two ways:

- **Render** calls `useRules()` under a `Suspense` boundary with the screen's skeleton (its own
  `LazyComponent` fallback, or the lazy section or Desk card it is already inside). The idle
  preload has almost always finished first, so the skeleton rarely shows.
- **Validation** goes through `rulesResolver` (`apps/web/src/lib/rules-resolver.ts`): the form's
  resolver is asynchronous, awaits `loadRules()` (importing the loader only then) and validates
  with the schema built from the loaded module. The shared zod schemas, which also parse server
  data, carry no allocation rule; `withShipmentAllocations` and `withChargeAllocations` add it
  for a form, and run even while other fields fail so a split dialog shows its problem.

| Screen | Operation | One call covers | Waits by |
|---|---|---|---|
| Shipment form, document draft review (`shipment-panel.tsx`, `document-shipment-draft-review-dialog.tsx`) | `allocation.remainder` | the freight and every additional charge, per validation | `rulesResolver` |
| Order add-charge, billing queue reassign-charge dialogs | `allocation.remainder` | the charge being split, per validation | `rulesResolver` |
| Charge split editor's remainder line (`components/billing`) | `allocation.remainder` | the charge being split | `ChargeSplitRemainderSkeleton` |
| Shipment charge summary, additional charges list, billing by payer | `allocation.remainder`, `allocation.chargeTotal`, `allocation.split` | the freight; every listed charge; every charge of the shipment | the shipment form's lazy sections (`ShipmentFormSkeleton`) |
| Additional charge dialog | `allocation.chargeTotal` | the charge being edited | the shipment form's lazy sections |
| Shipment board money line, billing queue charge lines, Desk shipment card | `allocation.chargeTotal` | every charge of the shipment, a percentage one against the freight | `MoneyLineSkeleton`, `BillingQueueChargeLinesSkeleton`, the card's skeleton |
| Report result grid (builder preview, explore, dashboards, drill-through) | `report.format` | 200 rows of every column, formatted the first time a row of the page is read and kept for that result (memoised on the module, columns and rows); the totals row | `ResultGridSkeleton` |
| Report chart and KPI | `report.format` | every plotted value; an axis tick Recharts picks later is formatted once and kept | `ReportChartSkeleton` |
| Report builder column format and band editors | `report.resolve`, `report.format` (with `resolve`) | the column's display and its sample; every previewed band | `ColumnFormatEditorSkeleton`, `BandPreviewSkeleton` |
| Rate agreement lane editor | `lane.keys`, `lane.specificity`, `lane.coverage` | every lane of the agreement, again only when a field they read changes | `LaneEditorSkeleton` |
| Lane key labels (`useLaneKeyLabels`: import rate sheet, simulation) | `lane.parse` | every distinct key on screen | `LaneKeyTableSkeleton`, `SimulationPanelSkeleton` |
| Desk rate agreement card | `lane.keys` | every lane read so far, again when a page arrives | the card's skeleton |
| Tariff editor in both Studios (`components/formula-editor`): highlighting, lint, completion, hover, signature help, definition, rename, inlays, folding, format | `tariff.tokens`, `tariff.check`, `tariff.complete`, `tariff.hover`, `tariff.signature`, `tariff.definition`, `tariff.rename`, `tariff.inlays`, `tariff.format` | the document at the cursor or the visible range; one check per text, kind and context (an LRU of four), shared by the linter, the folding and the Studio | `StudioSkeleton` (the Studio's provider), and `TariffEditor`'s own `useRules()` |
| Template and library Studios: outline, problems, tests panel, readiness, reference pane, "Save as test" on the sample load | `tariff.check`, `tariff.test`, `tariff.catalog`, `tariff.scaffoldTest` | the source once per text (`useTariffWorkbench`); the tests once the source has rested 400 ms, again after each load of the table cells they need (`useTariffTests`: three loads by combination, then the tables whole, through `POST /formula-templates/context/cells`, kept per stamp); the catalog once per loaded runtime | `StudioSkeleton` |
| Nova's proposal card | `tariff.check`, `tariff.test`, `tariff.format` | the proposed source | the Studio's boundary |
| Desk formula template card, every read-only source (`TariffCode`) | `tariff.tokens`, `tariff.check` | the source, once per text | the card's skeleton |
| Rate matrix grid editor | `matrix.coverage` | the whole matrix, on each change of its cells or axes; each axis labelled as the screen names it | `MatrixGridEditorSkeleton` |

Tests read the corpora through `rules/__tests__/corpus.ts`, so a wrapper or screen test asserts
the case the crate is pinned to, never what the code returned when the test was written.

The Studios and the read-only views of a source are Tariff's browser callers, and no Tariff is
lexed, parsed, checked, evaluated or formatted in TypeScript; [formula-language-service.md](formula-language-service.md) has the
editor's side.
