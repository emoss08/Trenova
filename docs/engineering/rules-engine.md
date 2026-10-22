# Rules engine (`trenova-rules`)

Some rules have to give the same answer in the browser and on the server, to the cent:

- the formula language, which the Formula Studio checks as you type and the server prices with;
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
| `src/message.rs` | `message!` and `Message`: user-facing text as English ICU templates plus arguments |
| `src/span.rs` | byte spans inside the crate; UTF-16 `Range` on the wire |
| `src/wire.rs` | `Dec`: exact decimals on the wire |
| `src/formula/syntax/` | lexer, parser (error-tolerant), AST, formatter |
| `src/formula/catalog.rs` | every function, built-in and word operator, with its shape and docs; disabled and reserved names |
| `src/formula/check/` | static types, diagnostics and fixes, `let` rules, nullable guards, the Studio's analysis |
| `src/formula/eval/` | values, the metered interpreter, built-in functions, dates and durations, rate table lookups, traces |
| `src/allocation/`, `src/report/`, `src/lane/`, `src/matrix/` | the non-formula rules |
| `services/crates/rules/testdata/` | the regression corpora every implementation detail answers to (`conformance.json` and one file per area) |
| `services/tms/pkg/rules/` | the Go host: the embedded `rules.wasm`, the wazero runtime and instance pool, typed calls per area |
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
{"op": "formula.check", "input": { ... }}
{"ok": { ... }}
{"error": {"code": "malformed" | "unknown_op" | "invalid_input", "detail": "..."}}
```

`ok` means the operation ran, whatever it found: a formula full of errors is still a successful
check.

`error` means the request was unusable. That is always a bug in the caller. Its detail is
developer English, never shown to a user and never translated.

A panic inside the module traps. The host treats a trap as a bug: it discards that instance and
reports an internal error. No input, however malformed, may make the crate panic.

### Wire conventions

- **Decimals are exact.** An input decimal is a JSON number or a JSON string, read from its text
  and never through `f64` (`wire::Dec`). An output decimal is always a JSON string, because
  `JSON.parse` would turn a number back into a float.
- **Value trees** are the formula environment, results and trace values. They are JSON, with
  numbers read exactly. Two kinds have their own encoding:
  - a date is `{"$date": "2026-01-02T03:04:05-05:00"}`: RFC 3339 with the offset the value
    carries;
  - a duration is `{"$duration": "<nanoseconds>"}`.

  Object keys starting with `$` are reserved for these tags.
- **Text positions** are UTF-16 code units: `{"from": 3, "to": 6}`, half-open. Inside the crate
  everything is a byte `Span`, converted at the edge by `span::Utf16Index`.
- **User-facing text** is a `Message`: `{"message": "unknown name {0}", "args": ["foo"]}`. The
  template is English ICU with positional placeholders; it is also the catalog key. The browser
  passes it to `translate(message, ...args)`; Go passes it to `shared/i18n`. Templates are
  declared with `message!("...")`, and `task i18n` extracts them into the Go scope and the web
  app's `web` bundle.
- **Names** on the wire are camelCase.

## The formula language

The language keeps expr-lang's syntax and the behaviour the conformance corpus pins, so every
formula in the standard catalog, the seeds and the docs still means what it did. It is owned
here now, and these choices are deliberate.

### Values and types

The static types are:

- `Number`, with `Integer` as a static subtype for whole numbers;
- `String`, `Boolean`, `Date`, `Duration`;
- `Array`, `Object`, `Nil`;
- `Any`, for values whose type is not known statically.

**Number** is one exact decimal: `rust_decimal`, 28 significant digits. No float exists
anywhere; the old engine's `float64` paths are gone.

- `Integer` is a static fact about a number, not a second runtime type. A number is `Integer`
  when it is an integer literal (decimal, hex `0x`, octal `0o` or binary `0b`, `_` separators),
  a field or input declared `integer`, the result of `+`, `-` or `*` on integers, or the result
  of `int`, `len`, `count`, `findIndex`, `findLastIndex` or `indexOf`.
- `%`, the bit functions, `..` bounds and `take` need `Integer` operands. The checker refuses
  others, and the evaluator also refuses a fractional value it meets at run time.
- `/` divides exactly, to 28 significant digits. Division by zero is an error: a warning when
  the divisor is a literal zero, a run-time error otherwise.
- `**`, `^` and `pow` are exact for integer exponents. Fractional exponents and `sqrt` use
  `rust_decimal`'s maths, which is correct to the last digit it can hold.
- Overflow is a run-time error, never a wrap.

**Dates** carry an instant and a UTC offset. The server sends each date in the offset of the
stop it came from.

- `date(text)` reads these forms: RFC 3339, `2006-01-02T15:04:05`, `2006-01-02T15:04`,
  `2006-01-02 15:04` and `2006-01-02`. A form without an offset is UTC.
- `duration(text)` reads Go duration syntax: `1h30m`, with the units `ns`, `us`, `µs`, `ms`, `s`,
  `m` and `h`.
- A date adds or subtracts a duration, and a date minus a date is a duration. Both kinds
  compare.
- A duration adds, subtracts, and multiplies or divides by a number.
- Methods on a date: `Year()`, `Month()`, `Day()`, `Hour()`, `Minute()`, `Second()`,
  `Weekday()` (0 is Sunday), `YearDay()` and `Unix()`.
- Methods on a duration: `Hours()`, `Minutes()` and `Seconds()`.

There is no time zone database in the module. `timezone()` and `date(text, layout, zone)` are
not part of the language.

**Equality and comparison** between different static kinds is an error, wherever the value
came from. `origin.zip == 75201` is refused (`origin.zip` is text, so the comparison is always
false). The old engine let it through because relation objects were untyped maps.

**Maps iterate in key order.** `keys`, `values`, `toPairs` and printing all sort by key, so a
formula's output never depends on hash order.

**Text** is Unicode. `len`, indexing and slicing count characters (Unicode scalar values), not
bytes.

### Syntax

The grammar, with its precedence:

| Precedence | Operators |
|---|---|
| 0 | `\|` pipe; the right side must be a call, and `a \| f(b)` is `f(a, b)` |
| 10 | `or`, `\|\|` |
| 15 | `and`, `&&` |
| 20 | `== != < > <= >= in matches contains startsWith endsWith` |
| 25 | `..` |
| 30 | `+ -` |
| 60 | `* / %` |
| 100, right-associative | `** ^` |
| 500 | `??` |
| unary 50 | `not`, `!` |
| unary 90 | `-`, `+` |

More of the grammar:

- `not` may prefix `in`, `matches`, `contains`, `startsWith` and `endsWith`.
- **Chained comparisons:** `0 < x <= 5000` means `0 < x and x <= 5000`. A parenthesised
  comparison does not chain.
- **Conditionals:** the ternary `a ? b : c`, and `if a { b } else { c }`, which must be
  parenthesised when used as an operand.
- **Lists:** literals `[1, 2]` and maps `{a: 1, "b c": 2}`. Trailing commas are allowed in
  lists, maps and calls.
- **Access:** members `a.b`, optional members `a?.b`, `a?.[i]`, indexes (negative indexes count
  from the end), slices `a[i:j]` and the range `1..3`.
- **Predicates:** `{ ... }` or a bare expression in a predicate argument. Inside one, `#` is the
  element, `.field` its field, `#index` its position and `#acc` the accumulator of `reduce`.
- **`let`:** `let name = value; body` at top level, inside parentheses, inside an `if` block or
  inside a predicate.
  - Sibling scopes may reuse a name.
  - An inner scope may not redeclare a name in scope.
  - A `let` may not be named after a field, an input, a function or a built-in.
- **Literals and comments:**
  - strings `"..."` and `'...'` with escapes;
  - raw strings in backticks, where a doubled backtick is a literal one;
  - comments `//` and `/* */`;
  - identifiers: letters in any script (`\p{L}`), digits and `_`.

### Functions

`formula.catalog` lists every function. The Go schema endpoint (`GET /formula-templates/schema`)
republishes that list next to the record's variables. The set:

- **Rounding:**
  - `round(x, d?)`: half away from zero;
  - `roundUp`, `roundDown`, `roundHalfEven`: `d` between -12 and 12;
  - `roundTo(x, step)`;
  - `ceil`, `floor`, `abs`.
- **Aggregates:** `min`, `max`, `sum`, `avg`, `mean`, `median`. Each takes numbers and lists of
  numbers, flattened one level. `min` and `max` keep the first of equal values.
- **Other numbers:** `clamp`, `pow`, `sqrt`, `coalesce`.
- **Rate tables:** `lookup`, `lookupOr`, `lookup2`, `lookup2Or`, `lookupInterp`,
  `deficitWeight`. The table name must be a string literal.
- **Text:**
  - `upper`, `lower`, `trim`, `trimPrefix`, `trimSuffix`, `hasPrefix`, `hasSuffix`;
  - `indexOf`, `lastIndexOf`;
  - `replace`, `split`, `splitAfter`, `join`;
  - `len`, `string`, `float`, `int`, `type`.
- **Lists and maps:**
  - `first`, `last`, `get`, `take`;
  - `keys`, `values`, `toPairs`, `fromPairs`;
  - `reverse`, `uniq`, `concat`, `flatten`, `sort`, `sortBy`.
- **Predicates:**
  - `all`, `none`, `any`, `one`, `count`;
  - `filter`, `map`, `find`, `findIndex`, `findLast`, `findLastIndex`;
  - `groupBy`, `reduce`.
- **Bits**, on integers: `bitand`, `bitor`, `bitxor`, `bitnand`, `bitshl`, `bitshr`,
  `bitushr`, `bitnot`.
- **Dates:** `date`, `duration`.

`now`, `toJSON`, `fromJSON` and `repeat` are disabled, each with a reason and a replacement
(`now` becomes `ratingTime`). `toBase64`, `fromBase64` and `timezone` are not part of the
language.

### Budgets

The evaluator meters its work, so a runaway formula stops with an error instead of burning a
CPU:

- every node evaluated, and every element a built-in visits, costs a step;
- every byte of text built counts against a text budget;
- every list element created counts against an items budget.

The defaults are in `eval` and may be lowered per request. The parser caps the source length,
the node count and the nesting depth. On the server, wazero also closes an instance when its
call's context ends: a deadline kills the work instead of abandoning it.

### Errors

A static problem is a diagnostic. It has:

- a range, a severity and a stable `code`;
- a `Message`;
- fixes.

The browser and the server get the same diagnostics, and the server refuses any formula with an
error-severity one.

A run-time problem is an evaluation error. It has:

- a stable `code`;
- a `Message`;
- the range of the node that failed;
- for a missing value, the fields involved and the guards that would fix them.

## Operations

Each area documents its operations here: the input and output, with an example, and the
`testdata/` file that pins them.

<!-- formula.catalog, formula.check, formula.format, formula.signature, formula.identifier -->

### `formula.evaluate`

Evaluates a batch of formulas against one environment in one boundary crossing: the server sends
a template's main expression and every breakdown line for a shipment together. The environment
is read once and the rate tables are indexed once; each program runs under its own budget. The
host runs `formula.check` first; `schema` is accepted for that and ignored here.

```json
{"op": "formula.evaluate", "input": {
  "programs": [
    {"id": "expression", "source": "lookup(\"fsc\", fuelType) * miles + linehaul", "expect": "amount"},
    {"id": "linehaul", "source": "baseRate * miles", "expect": "amount"},
    {"id": "rush", "source": "isRush and miles > 100", "expect": "predicate"}
  ],
  "env": {"baseRate": 2.5, "miles": 120, "linehaul": 300, "fuelType": "DIESEL", "isRush": true,
          "pickup": {"$date": "2026-01-02T08:00:00-06:00"}, "wait": {"$duration": "1800000000000"}},
  "tables": [{"name": "fsc", "axes": [{"match": "Exact"}],
              "cells": [{"at": [{"key": "DIESEL"}], "value": "0.35"}]}],
  "trace": {"statements": false, "lookups": true},
  "budget": {"steps": 200000}
}}
```

```json
{"ok": {"results": [
  {"id": "expression", "value": "342",
   "lookups": [{"function": "lookup", "table": "fsc", "keys": ["DIESEL"], "value": "0.35",
                "match": {"matchedKey": "DIESEL"}}]},
  {"id": "linehaul", "value": "300", "lookups": []},
  {"id": "rush", "value": true, "lookups": []}
]}}
```

**Programs.** `expect` is `amount` or `predicate`. Results come back in request order with the
program's `id`, and either a `value` or an `error`, never both.

- An **amount** is a decimal string. Anything else is an error, as the Go engine's `toDecimal`
  and `rejectBooleanAmount` decided: true or false is `booleanAmount`, nil is `nullResult`, text,
  a date, a list or a map is `nonNumericResult`.
- A **predicate** is `true`, `false` or a decimal string; the host reads a number as true when it
  is not zero. A predicate never reads rate tables: every lookup in it is zero (the Go engine's
  `StubLookup`), so a yes-or-no condition never loads a tenant's tables.
- A program that does not parse comes back with a `syntax` error at the first diagnostic and is
  not run; the other programs still run.

**The environment** is a value tree (see Wire conventions): an object of fields and inputs,
nested objects for relations, lists for stops and commodities. Numbers are read exactly from
their JSON text; a JSON string is always text, never a number. Dates are `{"$date": ...}` in the
offset of the stop they came from; durations `{"$duration": "<nanoseconds>"}`. A field the host
leaves out reads as nil, the same as one sent as `null`. Objects nest at most 32 levels.

**Tables.** `tables` absent means the context loaded none: every lookup fails with
`lookupUnavailable`. `tables: []` means none exist: every lookup fails with `tableNotFound`. Each
table is one rate matrix, encoded once by the host from the stamped matrix and cached as bytes:

| Field | Meaning |
|---|---|
| `name` | the matrix code a formula names |
| `axes` | one or two axes, in position order: `{match, kind?, normalization?, overflow?}` |
| `axes[].match` | `Exact` or `Range` (Go's `ratematrix.MatchMode`) |
| `axes[].kind` | Go's `DimensionKind`; only `FreightClass` changes how keys read; default `Custom` |
| `axes[].normalization` | `None` (default), `Trim`, `Upper` (trim, upper-case) or `Zip3` (trim, upper-case, first three characters) |
| `axes[].overflow` | `Error` (default), `ClampToTopBand` or `Nearest` |
| `cells` | `{at, value}`: `at` has one `{key?, min?, max?}` per axis; `value` an exact decimal |

A range axis matches half-open bands `[min, max)`; a band without `max` is open-ended; a cell
without `min` on a range axis is ignored, as Go ignored it. A freight-class axis reads `85`,
`"77.5"` and `"Class92_5"` as the NMFC class before normalising. A table with no axis or more
than two, a cell whose `at` does not have one entry per axis, or a name sent twice for the same
number of axes is a request error. A name may be both a one-axis and a two-axis table: `lookup`
reads one, `lookup2` the other.

The lookups are `matrix_lookup.go`'s, in exact decimals: `lookupOr` and `lookup2Or` fall back
only on a miss (`lookupMiss`); a missing table, the wrong shape or an unusable key always
surfaces. `lookup2` narrows to the cells whose row matches (overlapping row bands are all
candidates), takes the first whose column matches, and on a range column retries once with the
column moved into a band by its overflow policy. `lookupInterp` interpolates between band floors
and holds the edge values outside them; `deficitWeight` bills the next break's minimum when that
is cheaper at the next break's rate.

**Traces.** `trace.statements` adds `statements` to each result, from one pass over the program
(the Go engine compiled it once per statement):
`{index, kind: "let" | "result", name?, code, from, to, value?, error?}`, with `from` and `to` in
UTF-16 units. The last statement's `value` is the program's result as above; other values are
shown as the Go engine's `traceValue` showed them: decimals as strings, dates in UTC to the
second, durations as `1h30m0s`, text past 500 characters ellipsised, lists and maps to 3 levels
and 50 items (`"[N items]"`, `"{N fields}"`), nil map entries left out. A failed `let` is shown
with its error and the trace goes on; the program's result is the first failure.

`trace.lookups` adds `lookups`, every call in order, for the receipt (Go's `LookupTrace`; the
scope is the program's `id`): `{function, table, keys, value?, match?, error?}`. `match` is
`{matchedKey?, bandMin?, bandMax?, adjusted?}`: the normalised key of an exact table, the band of
a range table, `row × col` of a two-axis cell (a band reads `500–1000` or `500+`), the helper's
name for `lookupInterp` and `deficitWeight`; `adjusted` says an overflow policy moved the key. A
miss `lookupOr` absorbed is recorded with its error.

**Budgets.** Each program runs under `budget` (`steps`, `textBytes`, `items`), which may lower the
defaults and never raise them (raising one is a request error):

| Budget | Default | Counts |
|---|---|---|
| `steps` | 1,000,000 | every node evaluated and every element a built-in visits |
| `textBytes` | 1,048,576 | every byte of text an operation builds |
| `items` | 100,000 | every list item and map entry a formula creates |

**Errors.** An evaluation error is `{code, message, args?, from?, to?, fields?, guards?}`; the
message is a `Message` template with positional arguments, where the kind of a value is an ICU
`select` argument (`nil`, `boolean`, `text`, `number`, `date`, `duration`, `list`, `map`). For
`missingField`, `fields` are the field paths read without a value and `guards` the edit that keeps
the formula pricing for each (`coalesce(weight, 0)`, `origin?.zip ?? 0`, `.weight ?? 0` inside a
predicate, `coalesce(pickupDate, ratingTime)` for a date).

| Code | When |
|---|---|
| `syntax` | the program does not parse, or has no statements |
| `missingField` | a value the formula needs is nil |
| `unknownName` | a function, `#name` or method that does not exist |
| `type` | an operand or argument of the wrong kind, or `==` between different kinds |
| `argument` | an argument of the right kind with an unusable value (`round(x, 13)`, `date("x")`) |
| `index` | an index outside a list or text |
| `divideByZero` | `/` or `%` by zero |
| `overflow` | a number past 28 significant digits, a date past 9999, a duration past 292 years |
| `nonFinite` | a negative number to a fractional power, zero to a negative power |
| `regex` | a `matches` pattern that does not compile |
| `budget` | a budget is spent, or values nest past 32 levels |
| `lookupMiss`, `tableNotFound`, `tableShape`, `lookupKey`, `lookupUnavailable` | the rate table failures above |
| `booleanAmount`, `nullResult`, `nonNumericResult` | an amount or predicate that is not a number |

**Where it differs from the Go engine, deliberately.**

- Numbers are exact: `0.1 + 0.2` is `0.3`; division by zero is `divideByZero`, where float64 gave
  ±Inf or NaN; an unused `let` that divides by zero fails the formula.
- A nil reaching an operator is `missingField` naming the one field that caused it; Go named
  every nil field the formula referenced whenever anything failed.
- `==` between different kinds is an error at run time too; Go answered false.
- Rounding decimals, `%`, bits, ranges, slices and `take` need whole numbers; Go truncated 3.9.
- `int(text)` reads decimal text and truncates (`int("12.7")` is 12); Go's `Atoi` refused it.
- `type()` answers `number` for every number (one number type); dates and durations answer
  `date` and `duration`. `string()` writes nil as `nil`, a date in RFC 3339 and maps in key order.
- `indexOf`, `lastIndexOf`, `len`, indexes and slices count characters, not bytes.
- `upper` and `lower` use full Unicode case mapping.
- `min` and `max` take lists, flattened one level, like the other aggregates.
- A date minus a date that does not fit a duration is `overflow`; Go saturated.
- `duration()` applies the fraction in integer arithmetic; Go went through float64.
- `date(text)` reads only the five forms above; `date(text, layout, zone)` and `timezone()` are
  gone. Map keys built by `groupBy` and `fromPairs` are text (`1` becomes `"1"`).

The cases are in `testdata/evaluation.json` (ported from the Go engine's tests, each naming the
Go test it comes from) and the `value`s of `testdata/conformance.json`.

<!-- allocation.*, report.format, lane.*, matrix.* -->

## Hosts

<!-- The Go host: pool, timeouts, metrics, compile cache. The browser loader: loading, synchronous access, preload, tests. -->
