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

<!-- formula.evaluate and the rate table format -->

<!-- allocation.*, report.format, lane.*, matrix.* -->

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
