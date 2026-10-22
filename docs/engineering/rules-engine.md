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
  a field declared `integer`, `#index`, the result of `+`, `-`, `*` or `%` on integers, of a
  unary `-` or `+` on one, of a conditional whose branches are both integers, or the result of
  `int`, `len`, `count`, `findIndex`, `findLastIndex`, `indexOf`, `lastIndexOf`, a bit function
  or a date method. An input is declared `Number`, so it is never `Integer`.
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
- **Conditionals:** the ternary `a ? b : c`, its short form `a ?: c` (`a ? a : c`), and
  `if a { b } else { c }`, which must be parenthesised when used as an operand. A block may not
  be empty.
- **Lists:** literals `[1, 2]` and maps `{a: 1, "b c": 2}` (a key is a name, a string or a
  number). Trailing commas are allowed in lists, maps and calls.
- **Calls:** a function by name, `round(x, 2)`, or a method on a date or a duration,
  `pickupDate.Year()`; nothing else is called.
- `??` binds tighter than every other operator, so mixing it with one needs parentheses:
  `(a ?? 0) * 2`.
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

The defaults are in `eval` and may be lowered per request. The parser caps the source length
(32,768 UTF-16 code units), the node count (1,000) and the nesting depth (128), so no tree is
deeper than 128 nodes (see "Limits" under the static operations). On the server, wazero also
closes an instance when its call's context ends: a deadline kills the work instead of
abandoning it.

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

### The static formula operations

`formula.catalog`, `formula.check`, `formula.format`, `formula.signature` and
`formula.identifier` are the language service: the Formula Studio's editor runs them as you
type, and the server runs `formula.check` before it saves a formula. Their code is in
`src/formula/syntax/` (lexer, parser, tree, formatter, statements), `src/formula/catalog.rs` and
`src/formula/check/` (checker, schema knowledge, cursor lookups); `src/formula/ops.rs` holds the
handlers. `testdata/conformance.json` pins what the checker accepts and where it reports what it
refuses (`tests/formula_conformance.rs`); `tests/formula_*.rs` hold the rest, including every case
the old TypeScript language service was tested with.

Every position is UTF-16 code units, half-open, on the wire, and a byte span inside the crate.
Every user-facing text is a `Message` (`{"message": "...", "args": [...]}`); a diagnostic carries
its message flattened into it (`"message"` and `"args"` are the diagnostic's own keys).

#### `formula.catalog`

Every function, builtin, disabled and reserved name. The Go schema endpoint
(`GET /formula-templates/schema`) republishes `functions`, `builtins` and `disabled` unchanged
next to the record's variables.

Input: `{}`. Output:

```json
{
  "functions": [
    {
      "name": "round", "signature": "round(x, decimals?)",
      "description": {"message": "Rounds a number to the given decimal places (default 0, between -12 and 12)."},
      "example": {"message": "round(3.14159, 2) // 3.14"},
      "category": "rounding", "operator": false,
      "minArgs": 1, "maxArgs": 2, "returns": "Number", "numericArgs": true, "predicate": false
    }
  ],
  "builtins": [
    {
      "name": "filter", "signature": "filter(list, predicate)",
      "description": {"message": "The items the predicate holds for, in order."},
      "example": {"message": "len(filter(stops, .type == \"Delivery\"))"},
      "category": "predicate", "operator": false,
      "minArgs": 2, "maxArgs": 2, "returns": "Array", "numericArgs": false, "predicate": true
    }
  ],
  "disabled": [
    {
      "name": "now", "reason": "nondeterministic", "replacement": "ratingTime",
      "message": {"message": "{0} isn't available in formulas: a formula must price a shipment the same way every time it runs. Use {1}, the date the contract terms are read at", "args": ["`now()`", "`ratingTime`"]}
    }
  ],
  "reserved": ["abs", "all", "and", "..."]
}
```

- `functions` are the 41 entries the reference pane lists (rounding, aggregates, rate tables,
  text, and the word operators and slice with `operator: true`); `builtins` the 43 list,
  predicate, bit and date functions. Both have the same shape, and every entry has a
  description and an example now.
- `maxArgs` is `-1` for any number of arguments. `returns` is `Number`, `Integer`, `String`,
  `Boolean`, `Date`, `Duration`, `Array`, `Object` or `Any`. `category` is one of `math`,
  `rounding`, `aggregate`, `conditional`, `rateTable`, `string`, `list`, `predicate`, `bits` and
  `date`.
- `disabled` reasons are `nondeterministic`, `serialization` and `textBuilding`; `replacement` is
  left out when there is none. `message` is what the checker reports when a formula calls it.
- `reserved` (sorted) is every name a template input may not take: the keywords and every
  function, builtin and word operator.

#### `formula.check`

Input:

```json
{
  "source": "let linehaul = baseRate * totalDistance;\nlinehaul + weight * lookup(\"fuel\", 1) + extra",
  "expect": "amount",
  "schema": {
    "variables": [
      {"name": "weight", "type": "number", "description": "Total weight", "category": "shipment", "nullable": true, "computed": false},
      {"name": "totalDistance", "type": "number", "nullable": false},
      {"name": "stops", "type": "array"},
      {"name": "stops[].type", "type": "string"}
    ],
    "inputs": [{"name": "baseRate", "type": "Number"}],
    "operators": []
  },
  "tables": [{"name": "fuel", "axes": 1, "banded": false}]
}
```

- `expect`: `amount` (the default; the formula must come to a number) or `predicate` (a yes/no
  rule: true/false or a number).
- `schema.variables` is `DescribeSchema`'s variable list as published; extra keys are ignored.
  A field of a list's items is named `list[].field` (`stops[].weight`); those type `.weight`,
  `#.weight`, `stops[0].weight` and `first(stops).weight`, and without them an item's fields
  are not known (and not checked). `schema.inputs` are the template's inputs
  (`Number`/`String`/`Boolean`, any case). `schema.operators` are word operators beyond
  `startsWith`, `endsWith`, `contains` and `matches`.
- `schema: null` (or left out) means the schema has not loaded: the output says
  `pending: true` and leaves out `unknown-variable` and `unknown-field`. Functions come from
  the catalog, which is compiled in, so calls, arity, types and `let` rules are still checked.
- `tables` (optional): the rate tables the formula may read. With it, an unknown table name, a
  one-axis function on a two-axis table (or the reverse), and `lookupInterp` or `deficitWeight`
  on a table without bands are errors on the name's literal.

Output:

```json
{
  "empty": false,
  "pending": false,
  "diagnostics": [
    {
      "from": 52, "to": 58, "severity": "warning", "code": "nullable",
      "message": "{0} is empty on some shipments, and the formula would fail to rate them. Give it a fallback: {1}",
      "args": ["`weight`", "coalesce(weight, 0)"],
      "name": "weight",
      "fixes": [{"kind": "replace", "label": {"message": "Use {0}", "args": ["coalesce(weight, 0)"]}, "changes": [{"from": 52, "to": 58, "insert": "coalesce(weight, 0)"}]}]
    },
    {
      "from": 81, "to": 86, "severity": "error", "code": "unknown-variable",
      "message": "{0} isn't a shipment field or one of this template's inputs. Declare it as an input, or pick a field from the reference.",
      "args": ["`extra`"],
      "name": "extra",
      "fixes": [{"kind": "declare", "label": {"message": "Declare {0} as a {1} input", "args": ["extra", "Number"]}, "name": "extra", "type": "Number"}]
    }
  ],
  "resultType": "Number",
  "reads": ["baseRate", "totalDistance", "weight"],
  "schemaReads": ["totalDistance", "weight"],
  "inputReads": ["baseRate"],
  "functionCalls": ["lookup"],
  "bindings": [{"name": "linehaul", "type": "Number", "reads": 1, "from": 0, "to": 39, "nameFrom": 4, "nameTo": 12}],
  "statements": [
    {"index": 0, "kind": "let", "name": "linehaul", "code": "let linehaul = baseRate * totalDistance", "from": 0, "to": 39},
    {"index": 1, "kind": "result", "code": "linehaul + weight * lookup(\"fuel\", 1) + extra", "from": 41, "to": 86}
  ],
  "lookupTables": [{"name": "fuel", "function": "lookup", "from": 68, "to": 74}]
}
```

- **Acceptance.** A formula is accepted iff it has no diagnostic with `severity: "error"`. The
  browser and the server read the same answer; the server refuses anything else.
- `diagnostics` are sorted by `from`, then `to`, with duplicates (same range, code and message)
  removed. `name` is the identifier a finding is about, when it is about one. A `replace` fix's
  `changes` never overlap; a `declare` fix asks the Studio to add an input of that `type`
  (`Boolean` for a name such as `isWeekend`, `hasLiftgate`, `requiresEscort`; `Number`
  otherwise).
- `empty` is true for a formula of spaces and comments only; it then has one `empty` error.
- `resultType` is the static type of the last statement (`Integer` is a whole `Number`).
- `reads` lists every field and input read, inputs first, each in the order it first appears;
  `schemaReads` and `inputReads` split it; `functionCalls` lists the functions called (not
  methods). `bindings` are the top-level `let` lines.
- `statements` replace the old engine's `SplitStatements`: each `let` line and each value, its
  code from its first token to the last token of its value (no `;`, spaces or comments), with
  its range. They are listed for a formula that does not parse too, and a value line in the
  middle is its own statement.
- `lookupTables` replaces `ExtractLookupTableRefs`: every rate table read by a string literal,
  in source order, with the function that reads it and the literal's range (one entry per
  call: a caller that wants one per table and axis count dedupes by `name` and `function`).

The codes:

| Code | Severity | What |
|---|---|---|
| `syntax` | error | the text does not follow the grammar (message as the editor always worded it, often with a one-line fix: `;` after a `let`, `==` for `=`, `()` after a piped function), an escape the language does not know, a number that does not fit, or a pointer outside a predicate |
| `limit` | error | past a parser limit (below) |
| `empty` | error | nothing but spaces and comments |
| `unknown-variable` | error | a name that is no field, input or `let`; a "did you mean" replace fix within two edits, and a declare fix |
| `unknown-field` | error | a field a record or a list's items do not have, at the first unknown part |
| `unknown-function` | error | a function the catalog does not have (`toBase64`, `fromBase64` and `timezone` are gone), a method a date or duration does not have, or a value called |
| `disabled-function` | error | `now`, `toJSON`, `fromJSON`, `repeat`; `now()` has a fix to `ratingTime` |
| `arity` | error | too few or too many arguments, on the parentheses |
| `type` | error | an operand or argument of the wrong kind, `%`/bits/`..`/`take` on a number that can hold a fraction (fix: `int(x)`), comparing different kinds, a field read from text, a method on a value that has none, a predicate given to a function that takes values (fix: `sum(map(list, .field))`) |
| `nullable` | warning | a field that can be empty read without a guard (`??`, `coalesce`, `?.`, a `nil` test that guards it, including `x != nil && x ...`), once per field, with the guard as a fix: `coalesce(weight, 0)`, `(shipmentType?.code ?? "")` for a field under an object, `(.weight ?? 0)` for an item's field; dates fall back to `ratingTime`, text to `""`, flags to `false` |
| `let-unused` | warning | a `let` nothing reads |
| `let-duplicate` | error | a name already bound in scope |
| `let-before-definition` | error | a `let` read above the line that defines it |
| `let-shadows` | error | a `let` named after a field, an input or a function |
| `let-last` | error | the formula ends on a `let`; a fix returns its name |
| `result-type` | error | the formula does not come to what `expect` asks |
| `discarded` | warning | a value line before the last, worked out and thrown away |
| `divide-by-zero` | warning | a divisor that is a literal zero |
| `rate-table` | error | a table name that is not a string literal, or (with `tables`) an unknown table or one of the wrong shape |

Messages reuse the TypeScript language service's English templates wherever a check carried
over, so their translations in `i18n/messages.*.json` still apply. An inner phrase that the old
service translated separately (a value's description such as ``"`x` is a number"``, a type's
name) is passed as an English argument, because `Arg` has no nested message yet.

#### `formula.format`

Input `{"source": "a+b", "width": 80, "operators": []}` (`width` defaults to 80, `operators` to
none). Output `{"ok": true, "text": "a + b", "changed": true}`, or
`{"ok": false, "reason": "empty" | "syntax" | "comments"}` when the formula is left alone: it is
empty, it has a syntax diagnostic (so a part would be lost), or it holds a comment (which the tree
does not keep). One statement per line, a single space around every operator, a blank line
between statements kept, a long line broken where it reads best (a sum one term per line with the
operator leading, a call one argument per line, a ternary one case per line, a `let` after its `=`
when the value then fits); the author's parentheses, literals and operator spellings are kept.
Widths count UTF-16 code units. Formatting its own output changes nothing.

#### `formula.signature`

Input `{"source": "round(totalWeight, 2)", "position": 19, "operators": []}`. Output
`{"name": "round", "active": 1, "from": 5, "to": 21}`: the innermost call around the cursor whose
function the catalog has (builtins included; not methods or operators), which argument the cursor
is in (from 0; for a pipe the piped value is argument 0), and the call's parentheses (an unclosed
call runs to the end of the text). `null` when the cursor is in no such call, on its `(`, or past
its `)`. The editor reads the signature and description from `formula.catalog` by `name`.

#### `formula.identifier`

Input `{"source": "customer.name + 1", "position": 3, "schema": {...}}` (`schema` optional).
Output `{"name": "customer.name", "kind": "variable", "from": 0, "to": 13}`, or `null` when the
cursor is on no name. Dotted paths are one name; strings and comments never count. `kind` is
`function` (a call), `method` (a call on a value, `pickupDate.Year()`), `field` (after a `.` that
joins it to no name: an item's `.weight`, or `stops[0].weight`), `let` (bound by a `let` anywhere
in the formula), `pointer` (`#`, `#acc`, `#index`), and with a schema `input`, `variable` or
`unknown`; without one every other name is a `variable`.

#### Limits

| Limit | Value | Past it |
|---|---|---|
| Source length | 32,768 UTF-16 code units | one `limit` error from that point; nothing is parsed |
| Nodes | 1,000 (as the old engine allowed) | one `limit` error; the rest reads as the end of the text |
| Nesting depth | 128 levels (brackets, operands, calls, `else if` chains, the left spine of a long chain) | one `limit` error; the rest reads as the end of the text |

Every tree is therefore at most 128 nodes deep, which is what lets the checker, the formatter and
the evaluator recurse within WebAssembly's 1 MiB stack (`tests/formula_properties.rs` checks the
deepest inputs on a 1 MiB thread). A suggestion is never sought for a name longer than 64
characters, so no step is quadratic in the length of the text: checking a 1,000-node formula
takes under 2 ms in an unoptimised test build, and a small fraction of that optimised.

`fuzz/` holds cargo-fuzz targets for `formula.check` (with the signature and identifier lookups)
and `formula.format` (which must give text that parses and formats to itself).

<!-- formula.evaluate and the rate table format -->

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

The rules (`shipment.ResolveShares`, `ResolveOrderChargeShares`, `decimalutils.AllocatePercent`):

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

Two Go rules were wrong and are fixed here; `decimalutils.AllocatePercent` and the resolver's
amount check should follow when the Go callers move to the module:

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
`ResolveShares`/`ResolveOrderChargeShares`/`AllocatePercent`/`SumEquals` → `allocation.chargeTotal`
and `allocation.split`; `FindStaleAmountSplits`/`ConvertAmountSplitsToPercent` →
`allocation.rebalance`. They stay native: `toMinorUnits` (money in cents for the credit memo
dialog and invoice line comparison, not an allocation rule), and the GraphQL shaping helpers
`nestChargeAllocations`, `toChargeAllocationInput`, `hasAnyAllocations`. Grouping allocations
by the charge they target (`TargetID`, the kind filters in `ResolveShares`) and payer
validation (`ValidateAllocations`: the customer exists, is active and belongs to the tenant)
stay in Go, since they need the shipment and the database.

### Report formatting (`report.*`)

Every place a report is shown (the builder preview, the result grid, CSV, PDF, the XLSX text
fallback) renders values through the column's display contract (`reportfmt.Resolved`). Pinned
by `testdata/report.json`.

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

**Bands** (`reportfmt.Band`). The value of a banded cell is its range's lower bound. A fixed
`width` gives `[value, value + width)`. Explicit `edges` give the range of the last edge at or
below the value (a value below the first edge is in the first range), and the top range is
open: "$5,000.00+". Both ends render through the column's display, with a trailing suffix written
once: "10,000 – 15,000 lbs". Bounds are exact decimals.

**Tones** (`ToneFor`). Rules are tried in order and the first match gives the tone: `gt`, `gte`,
`lt`, `lte`, `eq`, and `between` (inclusive). Only numbers are toned, compared exactly; a percent
column's values are compared times 100, so a threshold of 12 means 12%.

#### `report.resolve`

`{"columns": [{"type": "decimal", "hint": "money", "dateStyle"?, "band"?, "spec"?}]}` →
`{"displays": [<resolved display>]}` (`reportfmt.Resolve`):

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
`resolveReportDisplay` and `reportfmt.Resolve` → `report.resolve` (or `resolve` inside
`report.format`); `Resolved.String`, `ToneFor`, `Band.Bounds` and `bandLabel` → `report.format`.
They stay native:

- dates (Go time layouts; `formatDate` and `dateParts` with `Intl`), fed by the `date` cells;
- `reportfmt.ExcelFormat` (Go only; nothing in the browser writes XLSX number formats);
- `Spec.Validate` and `Band.Validate`, which are save-time checks with developer messages;
- `Band.IsWhole` and `IsLowest`, which the query compiler and drill-through use;
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
and `laneKeyStringDisplay` (substituting translated record names into `segments`),
`laneEndDisplay` and `laneDisplayLabel` (translated labels only), `IsEffectiveAt` and
`MatchesDayOfWeek` and the rest of rule matching (server-side, against shipment facts).

### Rate matrices (`matrix.*`)

Pinned by `testdata/matrix.json`. Dimensions are
`{"position": 0..3, "matchMode": "Exact" | "Range", "kind": "Zone", "label": ""}`. Cells are the
GraphQL cell (`id`, `d0Key`..`d3Key`, `d0Min`..`d3Max`; other fields ignored). A blank or `null`
bound is no bound, and `null` entries are skipped. A position outside 0..3 or more than four
lookup values is `invalid_input`. A dimension's display label is its trimmed `label`, or else
the English name of its kind (Zone, Postal prefix, Postal code, State, Country, Weight break,
Distance, Pieces, Linear feet, Freight class, Equipment, Service, Custom, Quantity). A host that
translates passes the translated name as `label`.

**Bands** are half open, `[min, max)`, and a missing bound is no bound:
`matrix::band::contains(min, max, value)` is the one implementation (`ContainsQuantity`), and
the formula evaluator's band lookups should call it.

#### `matrix.select`

One lookup and its candidate cells (`SelectCell`):

```json
{"dimensions": [{"position": 0, "matchMode": "Exact"}, {"position": 1, "matchMode": "Range", "kind": "WeightBreak"}],
 "cells": [{"id": "rmc_1", "d0Key": "SE", "d1Min": 0, "d1Max": 1000}, {"id": "rmc_2", "d0Key": "SE", "d1Min": 1000}],
 "values": [{"keys": ["SE", "SOUTH"]}, {"quantity": "1000"}]}
```

→ `{"cell": {"index": 1, "id": "rmc_2", "preference": 0, "matchedKeys": ["SE", "1000", "", ""]}}`,
or `{"error": {"code": "dimension_value_missing", "position": 1, "message": {"message": "No value was supplied for the matrix dimension {0}", "args": ["Weight break"]}}}`,
or `{"error": {"code": "no_matching_cell", "message": {"message": "No matrix cell matches the supplied values"}}}`.

- Every exact axis needs a key and every banded axis a quantity; the first axis without one, in
  dimension order, fails the lookup. An axis with another match mode constrains nothing, as in
  Go.
- A cell matches when its key on every exact axis is one of that axis's keys and its band on
  every banded axis contains the quantity.
- The keys are ordered most preferred first. `preference` is the sum of the matched keys'
  positions, and the lowest wins, then the lowest cell ID, then the earlier cell in the request
  (unsaved cells, which sort before saved ones). Go's `sort.Slice` left that last tie
  unspecified.
- `matchedKeys` gives per position the key matched, or the quantity normalised.

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
the English band labels ("{0} – under {1}", "{0} and over", "under {0}", "Any"), and `bands`
gives both bands' bounds, because a `Message` argument cannot be a `Message`: a host that
translates rebuilds the labels from `bands`. The same finding in several slices is reported
once, with `slices` listing the other axes' coordinates (absent for a one-axis matrix).

Moving it from TypeScript fixed three things. Its messages are TypeScript's English unchanged.

- TypeScript collected the bands of every cell together, so a zone-by-weight matrix whose zones
  break at different weights showed false overlaps, and a gap in one zone was hidden by another.
  Each slice is now checked on its own.
- It compared only neighbouring bands, so a wide band covering two narrow ones produced a false
  gap and a missed overlap.
- It skipped every band after an unbounded one or before a floorless one, missing their
  overlaps.

**What replaces what.** `ContainsQuantity` → `matrix::band::contains`; `SelectCell` →
`matrix.select`; `findMatrixCoverageIssues` → `matrix.coverage`. They stay native: the sheet
projection (`buildMatrixGrid`, `sliceBuckets`, `bucketKeyOf`, `cellCoordinate`,
`coordinateKey`), `axisLabel` and `bandLabel` for headers, which only lay cells out and label
them. `DensityScale.ClassFor` and the formula lookups' `KeyNormalization` and `RangeOverflow`
stay in Go with their callers; `ClassFor` uses the same half-open rule as `contains`.

## Hosts

Both hosts load the same committed bytes and speak the ABI above; neither knows anything about
an operation beyond its name. Typed calls per area wrap the generic call in modules of their
own beside it (Go: next to `pkg/rules`; browser: beside `runtime.ts`), each decoding into its
own types.

### The Go host (`services/tms/pkg/rules`)

```go
err := rules.Call(ctx, "allocation.split", input, &output)  // sonic in and out
raw, err := rules.CallRaw(ctx, "formula.evaluate", encoded)  // pre-encoded input, raw `ok` out
info, err := rules.Version(ctx)                              // rules.VersionInfo{Version, Operations}
```

The package functions use `rules.Default()`, the process-wide `*rules.Runtime`.
`rules.New(rules.Options{PoolSize, RecycleMemoryBytes, MemoryLimitBytes})` builds another
(tests do). The API is generic-free, so typed callers build on it without spreading generic
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
traps instead of growing. A request over 64 MiB (`rules.MaxRequestBytes`) is refused before
it runs.

**Timeouts.** The host adds none of its own: the caller's context bounds both the wait and the
run. A context that has already ended fails before anything compiles or waits. A deadline that
passes mid-call closes the instance (wazero checks the context while the code runs), so the
work stops instead of being abandoned. The evaluator's budgets bound a formula inside that.

**Errors** are sentinels, wrapped with the operation's name:

| Error | Means | The caller |
|---|---|---|
| `ErrRequest`, as `*rules.RequestError{Op, Code, Detail}` | the module refused the request (`malformed`, `unknown_op`, `invalid_input`), or the host did (`too_large`) | has a bug: return an internal error, never show the detail |
| `ErrTrap` | the module panicked or answered with something that is not an envelope; logged at error with the operation, never the payload; the instance is gone | returns an internal error and never retries |
| `ErrContextDone` | the context ended while waiting for an instance or running; wraps `ctx.Err()` | treats it as any cancelled work |
| `ErrUnavailable` | the module failed to compile or instantiate, or the runtime was closed | returns an internal error; start-up already logged why |

Typed callers map these to `errortypes` at their own edge.

**Metrics.** `pkg/rules` cannot import the metrics package, so it reports through a small
`rules.Observer`, implemented by `metrics.Rules` (README in
`internal/infrastructure/observability/metrics`):

| Metric | Labels |
|---|---|
| `trenova_rules_calls_total` | `op`, `outcome` (`ok`, `request_error`, `trap`, `context_done`, `unavailable`) |
| `trenova_rules_call_duration_seconds` | `op`; includes the wait for an instance |
| `trenova_rules_instances_in_use` | |
| `trenova_rules_instances_discarded_total` | `reason` (`trap`, `context_done`, `memory`) |

`op` is the operation's name. A name the module does not know, or one not shaped like
`area.name`, is counted as `unknown`, so a caller bug cannot grow the label set.

**Warm-up.** `infrastructure.RulesModule` (`internal/bootstrap/infrastructure/rules.go`, in the
core options both the API and the worker start with) gives the default runtime the logger and
the metrics. At start it warms the runtime in the background: compile, instantiate and call
`rules.version`, within two minutes. Start-up never waits for it. A failure is logged at error
level and every call reports it as `ErrUnavailable`.

### The browser loader (`client/packages/shared/src/rules`)

```ts
const rules = await loadRules();                  // load.ts: Promise<RulesRuntime>
rules.call("formula.check", { source, schema });  // runtime.ts: synchronous, JSON in and out
rules.callRaw("formula.evaluate", cachedJson);    // JSON text in, JSON text out
rulesSync().version();                            // throws RulesNotLoadedError before the load
const runtime = useRules();                       // hooks/use-rules.ts: suspends until loaded
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
