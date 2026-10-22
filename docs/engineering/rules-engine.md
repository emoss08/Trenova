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

### `formula.evaluate`

Evaluates a batch of formulas against one environment in one boundary crossing: the server sends
a template's main expression and every breakdown line for a shipment together. The environment
is read once and the rate tables are indexed once; each program runs under its own budget. The
host runs `formula.check` first. `schema` is the same as `formula.check` takes, and only its
`operators` matter here, since the grammar needs them: a published word operator runs the
two-argument function of its name written infix (`zip hasPrefix "75"`), and one with no such
function is `unknownName`.

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
  ±Inf or NaN; an unused `let` that divides by zero fails the formula. A result keeps 28
  significant digits: past that its last digit is rounded (`1 / 3 * 3` is
  `0.9999999999999999999999999999`), and a whole part past 28 digits is `overflow`.
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
