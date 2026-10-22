# Tariff: Trenova's rating language

This is the language reference for Tariff, the rating language `trenova-rules` implements. It
replaces the expr-flavoured formula language. Nothing has launched, so nothing here is held
back by compatibility: every formula, seed, catalog template, corpus case, prompt and doc is
rewritten.

The engine stays as it is: the `trenova-rules` crate, one WebAssembly module, exact decimals,
metered evaluation, no clock, no randomness, no I/O, and the browser and the server agreeing.
The language is what changes.

## Why a language of our own

A rating formula is a small program with a specific job: read a shipment, apply a contract and
bill charges. The current language is a general expression evaluator, which leaves five gaps:

- **It cannot itemise the bill.** One expression gives one number. Breakdown lines are separate
  expressions that the server reconciles, with a "Rounding" remainder line when they disagree.
- **Its numbers carry no meaning.** `baseRate * totalWeight` type-checks whether the rate is per
  pound or per hundredweight. That mistake costs 100× and nothing catches it.
- **It has no structure for a real tariff.** There are no weight breaks, no reusable functions,
  no named constants, no shared libraries, and no way for a formula to refuse to rate with a
  reason.
- **Inputs, policy and tests live outside it.** Inputs are declared in a form. Minimum,
  maximum and rounding are policy fields. Test cases are database rows. Each is one more place
  the formula's meaning can drift.
- **Its idioms are expr's.** `#`, `.field`, `#acc`, pipes and the ternary are hard to read and
  hard for Nova to write correctly.

Tariff is a small, statically typed, pure scripting language whose types are freight: money,
rates, weights, distances, durations, percents, lanes and stops. A formula reads like the rate
sheet it implements, and the checker refuses one that bills the wrong thing.

## A first look

```tariff
/// Truckload linehaul with fuel, extra drops and accessorials.
tariff "Dry van TL"

use std.fuel.{ doeSurcharge }
use org.accessorials.{ hazmat, detention }

/// Paid per loaded mile.
input rate: Money/mi = $2.35/mi
input minimumCharge: Money = $450
input extraDropFee: Money = $75

let miles = shipment.distance
let linehaul = rate * miles
let drops = shipment.stops.count(s => s.kind == Delivery)

charge "Linehaul" as LH = linehaul
charge "Fuel surcharge" as FSC = doeSurcharge(fuel.price, miles)

if drops > 1 {
  charge "Extra drops" as XDROP = extraDropFee * (drops - 1)
}

if shipment.hazmat {
  charge "Hazmat" as HAZ = hazmat(shipment)
}

for stop in shipment.stops {
  let wait = stop.dwell - 2 h
  if wait > 0 h {
    charge "Detention, stop {stop.sequence}" as DET = detention(wait)
  }
}

minimum minimumCharge
round to cents

test "a two-drop load bills one extra drop" {
  given shipment.stops = [pickup(), delivery(), delivery()]
  expect charge(XDROP) == $75
}
```

## Files

A Tariff file is one of three kinds, named on its first line:

| Kind | Header | Holds | Produces |
|---|---|---|---|
| Tariff | `tariff "Name"` | inputs, bindings, charges, total rules, tests | an itemised bill |
| Library | `library "name"` | `fn`, `const`, `type`, `enum`, tests | nothing; imported by tariffs and other libraries |
| Condition | `condition "Name"` | inputs, bindings, one final `Bool` expression | true or false (an accessorial's apply-when rule) |

Libraries are a new organisation-level record, versioned like templates. A tariff version pins the
versions of the libraries it imports, so editing a library never silently reprices an existing
contract. The standard library (`std`) ships with the module and is versioned with it.

## Lexical structure

- **Statements end at the end of a line.** There are no semicolons. An expression continues onto
  the next line when the line ends with an operator, a comma or an open bracket.
- **Comments:** `//` runs to the end of the line. `///` is a doc comment; it attaches to the next
  declaration and is shown in hover, completion and the inputs form.
- **Identifiers:**
  - names are camelCase (`extraDropFee`);
  - types and enum variants are PascalCase (`Money`, `Delivery`);
  - charge codes are UPPERCASE (`FSC`), matching the accessorial charge catalog's 10-character
    codes.
- **Keywords:** `tariff library condition use input let var const fn type enum charge if else
  match for in break continue return minimum maximum round to up down nearest require fail test
  given expect and or not true false none`.
- **Text:** `"double quoted"`, with `{expr}` interpolation, `{{` for a literal brace and the usual
  escapes. There are no raw strings and no single quotes.
- **Numbers:**
  - `1200`, `1_200` and `2.35`; there are no hex, octal or exponents;
  - with a unit: `500 lb`, `2.5 mi`, `4 h`, `30 min`, `48 ft`;
  - money: `$450`, `$2.35/mi`, `$18.50/cwt`;
  - percent: `18%`;
  - date: `@2026-10-22`; date-time: `@2026-10-22T08:00-05:00`.

## Type system

Types are the point of the language. They catch, before anything runs, the mistakes that
otherwise become wrong invoices:

- a rate per hundredweight multiplied by pounds;
- money added to a weight;
- an empty weight used without a fallback;
- a weight band left uncovered;
- an enum misspelt;
- a rate table read with the wrong keys.

The rules:

1. **Every expression has a static type.** The checker runs before anything is evaluated, and
   the same checker runs in the browser and on the server. A file with a type error doesn't
   save, doesn't preview and doesn't rate.
2. **Annotate the boundaries; everything else is inferred.** Annotations are required on:
   - `input` declarations;
   - function parameters and return types;
   - `type` fields;
   - library-level `const`s.

   Everywhere else the type is inferred, and the editor shows it as an inlay hint. `let x: T =
   …` is allowed, but never required.
3. **Optionals narrow by flow.** Inside `if let x = opt`, after `require let x = opt else …`,
   in the right side of `x != none and …`, and in a `match` arm that excludes `none`, the
   binding has type `T`, not `T?`. Nowhere else.
4. **Subtyping is minimal.** `Int` widens to `Number`, and `T` widens to `T?`. Nothing else is
   a subtype of anything. Records are compared by their declared type. A record literal checks
   against the type expected where it's written, and must match it exactly: no missing fields,
   no extra fields.
5. **Dimensions are types; units are values.** `Weight`, `Money/Weight` and `Duration` are
   types. `lb`, `kg` and `cwt` are units a value of type `Weight` carries.
   - In an annotation, a unit names the dimension and the display unit: `Money/cwt` is the type
     `Money/Weight`, shown per hundredweight.
   - A value in any weight unit is accepted where a `Weight` is expected, and is converted
     exactly.
6. **Built-in generics; no user generics.**
   - `List<T>` and `T?` are generic.
   - The list methods are typed by their lambda: `stops.sum(s => s.dwell)` is a `Duration`.
   - Lambdas exist only as arguments to the built-in list methods. User functions are not
     values: they can't be passed, stored or returned.
7. **User-defined types are records and enums.**
   - `type Break = { from: Weight, rate: Money/cwt, minimum: Money? }` declares a record. Two
     declared types with the same fields are still different types, and the name is in every
     error.
   - `enum Service { Standard, Guaranteed, Expedited }` declares an enum without payloads.
   - Schema enums (`StopKind`, `ShipmentStatus`, `FreightClass`) are enums of the same kind,
     generated from the schema.
8. **Equality and order.** `==` and `!=` take two values of the same type:
   - money and quantities compare by value across scale and unit (`$426 == $426.00`,
     `1 cwt == 100 lb`);
   - records and lists compare field by field and item by item;
   - enums compare by variant;
   - comparing different types is an error.

   `<`, `<=`, `>` and `>=` are defined on numbers, money, quantities of one dimension, rates of
   one dimension, dates, date-times and durations.
9. **Numbers.**
   - `Int + - *` gives an `Int`, and `Int / Int` gives a `Number`.
   - `%` takes `Int`s only.
   - Overflow past 28 significant digits is an error, never a wrap.
   - Conversions are explicit: `x.floor()`, `x.ceil()` and `x.round()` give an `Int` from a
     `Number`; `Number(i)` and `Int.parse(text)` give an `Int?`.
10. **No escape hatch.** There is no `any`, no cast and no reflection. Every value's type is
    known at check time.

What the type system leaves out, and why:

| Left out | Why |
|---|---|
| Classes, methods on user types, inheritance, traits, interfaces | They bring object-oriented design into pricing rules. Free functions in libraries cover reuse, and they are easier to read and for Nova to write. |
| User-written generics | `List<T>` and `T?` cover what pricing needs. User generics double the checker for almost no benefit. |
| User-defined units | The unit set is fixed and the conversions are exact. A hand-written conversion factor is where float-like error creeps back in. |
| Functions as values | They would make the call graph undecidable, so the acyclicity check that guarantees termination would stop working. |
| Implicit conversions | Every unit, currency and optional change is visible in the source. |

## Built-in types

| Type | Values | Notes |
|---|---|---|
| `Bool` | `true`, `false` | |
| `Int` | whole numbers | counts, sequences, indexes |
| `Number` | exact decimals | 28 significant digits; `Int` widens to `Number` |
| `Percent` | `18%` | multiplies as 0.18; prints as 18% |
| `Money` | `$450` | the rating currency (below) |
| Quantities | `Weight`, `Distance`, `Length`, `Volume`, `Duration`, `Temperature`, `Count` | carry a unit; arithmetic checks dimensions |
| Rates | `Money/mi`, `Money/cwt`, `Money/h`, `Money/pallet`, … | money per unit of a quantity |
| `Text` | `"…"` | Unicode; length and indexing count characters |
| `Date`, `DateTime` | `@2026-10-22`, `@2026-10-22T08:00-05:00` | instant plus UTC offset; date-times keep the stop's offset |
| `List<T>` | `[a, b]` | immutable |
| Records | `{ code: FSC, amount: $10 }`, `type Charge = { … }` | structural; fields are typed |
| Enums | `enum Zone { A, B, C }`, and every schema enum | variants are bare names: `Delivery`, `InTransit` |
| `T?` | a value or `none` | optional; must be handled before use |

### Units and dimensions

Every quantity has a dimension and a unit. Values are stored exactly, in a base unit per
dimension, and print in the unit they were written in. Conversions are exact: 1 lb is
0.45359237 kg, 1 cwt is 100 lb, 1 mi is 1.609344 km.

| Dimension | Units |
|---|---|
| Weight | `lb`, `cwt`, `kg`, `t` (metric tonne), `ton` (short ton) |
| Distance | `mi`, `km` |
| Length | `in`, `ft`, `m` (linear feet, dimensions) |
| Volume | `ft3`, `m3` |
| Duration | `min`, `h`, `d` |
| Temperature | `°F`, `°C` (comparison and difference only) |
| Count | `pallet`, `piece`, `stop`, and bare `Int` |

The arithmetic rules:

- `+`, `-` and comparison take the same dimension: `500 lb + 2 cwt` is `700 lb`.
- `Money × Number`, `Money × Percent` and `Money / Number` give `Money`.
- `Money/unit × quantity` gives `Money` (`$2.35/mi × 812 mi`), converting units as needed
  (`$18.50/cwt × 1_200 lb = $222`).
- `Money / quantity` gives a rate; `quantity / quantity` of one dimension gives a `Number`.
- Anything else is an error that says what was meant: "`rate` is money per hundredweight and
  `shipment.weight` is a weight, so `rate × shipment.weight` is money. `rate + shipment.weight`
  adds money to a weight."
- `.in(unit)` converts explicitly: `shipment.weight.in(cwt)`. `.value` gives the bare `Number`
  in the unit it was last written or converted to.

### Money and currency

A rating has one currency, the agreement's or else the organisation's. `$` means that currency,
and `USD 450` / `CAD 450` name one explicitly. Adding two currencies is a type error.
`convert(amount, to: CAD)` converts at the rating's stamped exchange rate, which the host passes
in with the shipment, so a rating stays reproducible.

### Optionals

A schema field that can be empty has type `T?` (`shipment.weight: Weight?`). Using one where a
`T` is needed is an error, with quick fixes for each way to handle it:

- `shipment.weight ?? 0 lb`: a fallback;
- `if let w = shipment.weight { … }`: bind it when present;
- `require let w = shipment.weight else "This rate needs the shipment's weight"`: refuse to rate
  without it, with `w` bound as a `Weight` from there on;
- `shipment.trailer?.length`: optional chaining.

## Bindings

- `let name = expr`: immutable. This is the usual form.
- `var total = $0` followed by `total += line`: mutable, for accumulating inside a loop. A `var`
  never escapes its block's scope.
- `const DIESEL_BASE = $3.50`: a compile-time constant at file level, importable from a library.
- `input name: Type = default`: a value the contract or template supplies. Inputs are only
  declared at file level; their doc comments label the Studio's inputs form.

A name may not shadow a schema root, an input, a function or another name in scope. That last
rule is today's `let` rule, kept.

## Statements

A tariff's body is a sequence of statements, run top to bottom. A block is `{ … }`, and every
block is a scope.

| Statement | Meaning |
|---|---|
| `let`, `var`, `const` | bindings (above) |
| `name = expr`, `name += expr`, `name -= expr` | reassign a `var` in scope; the type cannot change |
| `charge "Label" as CODE (options) = expr` | add a line to the bill (below); tariffs only |
| `note "Text {interpolated}"` | add a note to the bill: no amount, shown on the rate trace, the receipt and the invoice's notes |
| `if cond { … } else if cond { … } else { … }` | conditional; as a statement the branches need no value |
| `if let x = optional { … } else { … }` | bind an optional when present |
| `match value { pattern => { … } … }` | as a statement, arms are blocks |
| `for x in list { … }` | iterate a list or a range of `Int`s, in order |
| `for x in list where cond { … }` | iterate only the items `cond` holds for |
| `break`, `continue` | leave or skip to the next iteration of the innermost `for` |
| `require cond else "message"`, `require let x = optional else "message"` | refuse to rate unless it holds |
| `fail "message"` | refuse to rate |
| `return expr` | functions only |
| `minimum`, `maximum`, `round` | total clauses, last in a tariff (below) |
| `test "name" { … }` | a test (below); file level only |

There is no `while` and no recursion. A `for` walks a list or a range whose length is known
when the loop starts, so every formula terminates. A function body is the same statement
language without `charge`, `note`, total clauses or `test`. Functions compute values; tariffs
bill them.

## Expressions

The precedence, from loosest to tightest:

| Level | Operators |
|---|---|
| 1 | `??` |
| 2 | `or` |
| 3 | `and` |
| 4 | `not` |
| 5 | `== != < <= > >= in` (non-associative; `a < b < c` is an error with a fix) |
| 6 | `..`, `..=` (ranges) |
| 7 | `+ -` |
| 8 | `* / %` |
| 9 | unary `-` |
| 10 | `.field`, `?.field`, `.method(…)`, `(…)` calls, `[…]` indexes |

- `and`, `or` and `not` are the only logic spellings, and `**` is gone (use `.pow(n)`). One
  spelling per idea.
- `if cond { a } else { b }` is an expression, and replaces the ternary. `else if` chains.
- `in` tests membership in a list or range: `shipment.pickup.date.weekday in [Sat, Sun]`,
  `shipment.weight in 500 lb..2_000 lb`.
- Ranges are half-open with `..` and closed with `..=`. Either end may be left off:
  `..500 lb`, `2_000 lb..`.

### `match`

```tariff
let perCwt = match shipment.weight {
  ..500 lb          => $42.00/cwt
  500 lb..2_000 lb  => $35.50/cwt
  2_000 lb..5_000 lb => $28.10/cwt
  _                 => $22.75/cwt
}
```

- Patterns can be literals, ranges, enum variants (`Delivery`, `Dry Van | Reefer`), `none`, the
  wildcard `_`, record shapes (`{ kind: Delivery, appointment: true }`) and a guard (`x if x > 3`).
- A `match` must be exhaustive: the checker names the values no arm covers.
- Overlapping ranges are an error with both arms named. The weight-break table can't
  double-count, which is the same rule the matrix coverage check applies to rate matrices.

### Collections and lambdas

Lambdas are `x => expr`, or `(a, b) => expr` for two parameters. List methods:

- **Filtering:** `where`, `map`.
- **Counting and totals:** `count`, `sum`, `min`, `max`, `avg`.
- **Tests:** `any`, `all`, `none`.
- **Picking items:** `first`, `last`, `find`.
- **Ordering:** `sortBy`, `take`, `skip`, `reverse`, `distinct`.
- **Grouping:** `groupBy`, which gives `List<{ key, items }>`.
- **Size:** `len`, `isEmpty`.

A few examples:

```tariff
let deliveries = shipment.stops.where(s => s.kind == Delivery)
let heaviest = shipment.commodities.max(c => c.weight)
let palletCount = shipment.commodities.sum(c => c.pallets)
```

`#`, `.field` predicates, `#acc`, `reduce` and pipes are gone. Every list method above is
bounded by the list it walks.

### Functions

```tariff
/// Deadhead over the first 50 empty miles is paid at the loaded rate.
fn deadhead(empty: Distance, rate: Money/mi) -> Money {
  let paid = (empty - 50 mi).max(0 mi)
  return rate * paid
}

fn residential(fee: Money = $95) = fee
```

- **Shape:** parameters and the return type are annotated. A one-expression body may use `=`.
- **Arguments:** parameters can have defaults and be named at the call site:
  `deadhead(empty: 80 mi, rate: rate)`.
- **Purity:** functions are pure. They can't bill a charge or read a `var` from outside, so they
  can be tested and reused anywhere.
- **No recursion:** the call graph must be acyclic, and the checker refuses a cycle. Every
  formula terminates by construction, and the meter is a second line of defence.

### Methods on values

| On | Methods |
|---|---|
| `Number` and `Money` | `round(cents \| places: n)`, `roundUp(to: …)`, `roundDown(to: …)`, `roundHalfEven(…)`, `nearest($5)`, `abs`, `min(x)`, `max(x)`, `clamp(lo, hi)`, `pow(n)`, `sqrt` |
| Quantities | `in(unit)`, `value`, `min`, `max`, `clamp`, `roundUp(to: 100 lb)` (weight breaks bill to the next hundred) |
| `Text` | `upper`, `lower`, `trim`, `startsWith`, `endsWith`, `contains`, `replace`, `split`, `len`, `matches(regex)` |
| `Date` | `year`, `month`, `day`, `weekday`, `weekend`, `hour`, `minute`, `addDays(n)`, `isHoliday(calendar)` |
| `Duration` | `in(h)` and friends |

Durations come from subtracting date-times. `isHoliday(calendar)` reads a holiday calendar the
host passes in. Today only worker time-off knows about holidays, so a rating holiday calendar is
a new organisation setting.

### Refusing to rate

- `fail "No rate for {lane}"` stops the rating with that message. It is shown on the shipment,
  in the Studio, and to Nova, and is translated.
- `require cond else "message"` and `require let x = optional else "message"` are the guard
  forms.

A failed rating is a rating error the user can act on, never a silent `$0`.

## Charges and the total

- `charge "Label" as CODE = amount` adds a line to the bill.
  - **The code** is an accessorial charge code from the organisation's catalog (`FSC`, `DET`,
    `LIFT`). The checker refuses an unknown code. The code is what lands on the invoice line,
    the EDI 210 and the rate confirmation. A charge without `as CODE` is linehaul.
  - **Placement:** charges may appear inside `if`, `match` arms and `for` loops, as in the
    per-stop detention above.
  - **Labels** can interpolate (`"Detention, stop {stop.sequence}"`). Labels are user-facing,
    so they go through the catalog like any other message.
  - **Options** can follow the code: `charge "Fuel surcharge" as FSC (taxable: false) = …`.
- **The total** is the sum of the charges. These clauses apply to it, in order:
  - `minimum amount`: if the total falls short, the difference is billed as a "Minimum charge"
    line (code `MIN`), so the invoice still adds up.
  - `maximum amount`: caps the total, with a negative "Maximum charge adjustment" line.
  - `round to cents`, `round up to $5`, `round half even to cents`: rounds each charge. Any
    remainder goes on a "Rounding" line, so the lines always sum to the total. This is today's
    remainder rule, made visible in the formula.
- A total below zero is a rating error, as today.

The old separate breakdown lines and the min, max and rounding policy fields are gone. The
formula is the bill.

## Data

The record being rated is exposed in typed, unit-aware namespaces built from the schema
registry. Each field gets a unit, and optional fields are `T?`.

| Root | Holds |
|---|---|
| `shipment` | `weight: Weight?`, `pieces: Int?`, `distance: Distance`, `linearFeet: Length`, `volume: Volume`, `hazmat: Bool`, `reefer: Bool`, `temperature: { min: Temperature?, max: Temperature? }`, `status`, `serviceType`, `equipment`, `stops: List<Stop>`, `commodities: List<Commodity>`, `pickup: Stop`, `delivery: Stop`, `freightClass` |
| `lane` | `origin` and `destination` (`city`, `state`, `zip`, `zip3`, `country`, `zone`), `interstate: Bool`, `crossBorder: Bool` |
| `customer` | `name`, `code`, plus the custom fields the organisation marks as rateable |
| `fuel` | `price: Money/gal?`, `priceDate: Date?`, `index: Text?` |
| `rating` | `date: DateTime` (today's `ratingTime`), `currency`, `mode: Sell \| Buy` |
| `tables` | the organisation's rate tables, by name |

A `Stop` has `sequence: Int`, `kind: Pickup | Delivery`, `arrival: DateTime?`,
`departure: DateTime?`, `dwell: Duration` (zero when it is not known), `appointment: Bool` and
`location`.

### Rate tables

A rate table reads like a lookup on a value:

- `tables.Zones[lane.origin.zip3, lane.destination.zip3]` gives a `Money?` for a one- or two-axis
  lookup;
- `tables.Zones.get(…) ?? fail "No zone rate for {lane}"` refuses to rate on a miss;
- `tables.LtlBreaks.band(shipment.weight)` gives the matching band's `{ min, max, rate }`;
- `.interpolate(x)` and `.deficit(weight)` replace `lookupInterp` and `deficitWeight`.

The checker types each table from its definition: axes, keys, value type with units, bands. So
`tables.Zones["X", "Y", "Z"]` and a weight passed to a distance axis are errors, and completion
lists a table's real keys.

## Modules

- `use std.fuel` imports the standard library: `fuel` (DOE-indexed surcharges, per-mile and
  percent schedules), `ltl` (deficit rating, class lookups, density), `accessorials` (common
  ones as functions), `time` (business hours, holidays) and `geo` (zones).
- `use org.accessorials.{ liftgate, residential }` imports the organisation's own libraries by
  name.
- **Visibility:** everything a library declares is exported unless marked `private`.
- **Pinning:** each import is pinned per tariff version, as above.

## Tests

Tests live in the file they test and run in the Studio on every keystroke, in CI for the
standard library, and on save.

```tariff
test "a 1,200 lb load bills at the 500–2,000 lb break" {
  given shipment.weight = 1_200 lb
  given shipment.distance = 412 mi
  expect charge(LH) == $426.00
  expect total == $511.20
}

test "a load with no weight refuses to rate" {
  given shipment.weight = none
  expect fails "This rate needs the shipment's weight"
}
```

- `given` overrides fields of a sample shipment built from the schema's defaults. Per-file
  helpers such as `pickup()` and `delivery()` build stops.
- `expect` compares exactly, with a unit-aware equality: `$426` equals `$426.00`.
- **They replace the Studio's test-case rows**, which become a view of the tests in the source.

## Conditions

```tariff
condition "Residential delivery"

lane.destination.zone == Residential and shipment.weight ?? 0 lb < 10_000 lb
```

A condition is a tariff with no charges and one `Bool` result. It replaces the accessorial
`ApplyCondition` expression, with the same checker and the same Studio.

## The editor

The module answers, from the same parse:

- diagnostics with quick fixes (as today);
- completion, with types, units, doc comments, schema enums and table keys;
- hover, showing type, unit, doc and the last preview's value;
- go to definition, including into libraries;
- rename across the file;
- inlay hints, showing inferred types and each `let`'s previewed value;
- signature help;
- a fixed formatter, gofmt-style;
- folding, and an outline of inputs, functions, charges and tests;
- the test results, run in the browser.

Syntax highlighting moves into the module too, as a token stream for CodeMirror's highlighter,
so the last TypeScript copy of the lexer goes away.

## What stays the same

- **The engine.** One `trenova-rules` crate, one `wasm32-unknown-unknown` module, the same ABI,
  hosts, pools, metrics and build.
- **Exact decimals end to end, and metering.** Steps, text bytes and items, plus wazero closing
  the call on deadline.
- **Determinism.** No clock (`rating.date` is passed in), no randomness, map order by key.
- **Diagnostics.** They are `Message`s with nested arguments, translated in both hosts, with
  UTF-16 ranges.
- **One call per rating** on the server, and the batch shapes for predicates.

## What it costs

- **The crate's front end and checker are new.** The lexer, the parser, a real type checker
  (inference, units, optionals, exhaustiveness, overlap) and a module resolver. The evaluator is
  adapted rather than rewritten: values gain units, and it gains `var`, `for`, functions, charges
  and tests.
- **Schema work in Go.** Fields gain units and optional types. Rate tables publish their shapes.
  Accessorial codes and organisation libraries become new inputs to a check.
- **Storage.**
  - A template stores one source file. Breakdown definitions, the policy fields and the
    test-case rows are dropped.
  - Libraries become a new versioned record.
  - Contract accessorials store a `condition` file.
- **The Studio.**
  - The inputs form comes from the `input` declarations, and the test panel from the `test`
    blocks.
  - The rate trace shows charges.
  - The new editor features need wiring: completion, go-to, rename, inlays, the outline.
- **Nova.** Its prompt gains a language reference and examples, and every proposal goes through
  the checker and the tests before it is shown.
- **Rewrites.** The standard catalog, the seeds, the docs, the corpus and every test that writes
  a formula.

## Decisions

1. **Name.** The language is called Tariff. A freight tariff is the published rate document,
   which is what these files are.
2. **Syntax.** Rust- and TypeScript-flavoured: braces, no semicolons, `let`/`var`/`fn`/`match`,
   `x => …` lambdas.
3. **Units and currency** are part of the type system.
4. **Structure.** The formula is the itemised bill; breakdown lines and policy fields are
   removed.
5. **Inputs, tests and conditions** live in the source.
6. **No recursion and no `while`.** Termination holds by construction.
7. **Organisation libraries**, with versions pinned per tariff version.

## How it is built

The work goes in one PR, in this order:

1. The spec is frozen: this doc plus a grammar appendix.
2. The crate: lexer and parser, checker, evaluator changes, the module resolver, the editor
   operations and the standard library. The corpus is rewritten in Tariff.
3. Go:
   - the schema gains units and optionals;
   - table shapes, accessorial codes and libraries are published;
   - storage changes: templates become one source, libraries get a table, conditions are stored
     as source;
   - rating reads charges;
   - receipts, the HTTP contract and Nova's tools are updated.
4. The client:
   - the Studio, with inputs from source, tests, outline, completion, go-to, rename and inlays;
   - the token highlighter;
   - the rate trace;
   - a library editor;
   - the accessorial condition editor.
5. Seeds, the standard catalog, Nova's prompt, translations and the docs, with this doc becoming
   the language reference.

## Appendix A: grammar

The normative syntax, in EBNF:

- `{ x }` repeats zero or more times, `[ x ]` is optional, and `|` separates alternatives.
- `NL` is a statement-ending newline.
- `name` is a camelCase identifier, `TypeName` a PascalCase one and `CODE` an UPPERCASE one.
- A newline does not end a statement in three cases:
  - inside `( … )`, `[ … ]` or a record literal's `{ … }`;
  - after a binary operator, `,`, `=` or `=>`;
  - before a line that starts with a binary operator, `.`, `?.` or `??`.

  So chains and long sums wrap without a continuation character.

```ebnf
file        = header NL { doc } { item } ;
header      = ( "tariff" | "library" | "condition" ) string ;
doc         = "///" text NL ;
item        = { doc } ( use | input | const | typeDecl | enumDecl | fnDecl | test | stmt ) ;

use         = "use" modulePath [ "." "{" name { "," name } [ "," ] "}" ] [ "as" name ] NL ;
modulePath  = ( "std" | "org" ) "." name { "." name } ;

input       = "input" name ":" type [ "=" expr ] NL ;
const       = [ "private" ] "const" name [ ":" type ] "=" expr NL ;
typeDecl    = [ "private" ] "type" TypeName "=" recordType NL ;
enumDecl    = [ "private" ] "enum" TypeName "{" TypeName { "," TypeName } [ "," ] "}" NL ;
fnDecl      = [ "private" ] "fn" name "(" [ param { "," param } [ "," ] ] ")"
              [ "->" type ] ( block | "=" expr NL ) ;
param       = name ":" type [ "=" expr ] ;

type        = baseType [ "?" ] ;
baseType    = TypeName [ "<" type ">" ]                       (* Bool, Int, List<T>, Weight, … *)
            | "Money" "/" ( unit | TypeName )                 (* Money/cwt, Money/Weight *)
            | recordType ;
recordType  = "{" field { "," field } [ "," ] "}" ;
field       = name ":" type ;

test        = "test" string block ;
block       = "{" { stmt } "}" ;

stmt        = letStmt | varStmt | assign | charge | note | ifStmt | matchStmt | forStmt
            | require | fail | return | "break" NL | "continue" NL | total
            | given | expect | expr NL ;
letStmt     = "let" name [ ":" type ] "=" expr NL ;
varStmt     = "var" name [ ":" type ] "=" expr NL ;
assign      = name ( "=" | "+=" | "-=" ) expr NL ;
charge      = "charge" string [ "as" CODE ] [ "(" option { "," option } ")" ] "=" expr NL ;
option      = name ":" expr ;
note        = "note" string NL ;
ifStmt      = "if" condition block { "else" "if" condition block } [ "else" block ] ;
condition   = expr | "let" name "=" expr ;
matchStmt   = "match" expr "{" { arm } "}" ;
arm         = pattern [ "if" expr ] "=>" ( block | expr ) NL ;
pattern     = alt { "|" alt } ;
alt         = literal | rangePat | TypeName | "none" | "_" | name | recordPat ;
rangePat    = [ literal ] ( ".." | "..=" ) [ literal ] ;
recordPat   = "{" name ":" pattern { "," name ":" pattern } [ "," ] "}" ;
forStmt     = "for" name "in" expr [ "where" expr ] block ;
require     = "require" condition "else" string NL ;
fail        = "fail" string NL ;
return      = "return" expr NL ;
total       = ( "minimum" | "maximum" ) expr NL
            | "round" [ "half" "even" | "up" | "down" ] "to" ( "cents" | money ) NL ;
given       = "given" lvalue "=" expr NL ;                    (* tests only *)
expect      = "expect" ( expr | "fails" [ string ] ) NL ;     (* tests only *)
lvalue      = name { "." name } ;

expr        = coalesce ;
coalesce    = orExpr { "??" ( orExpr | "fail" string ) } ;
orExpr      = andExpr { "or" andExpr } ;
andExpr     = notExpr { "and" notExpr } ;
notExpr     = "not" notExpr | compare ;
compare     = range [ ( "==" | "!=" | "<" | "<=" | ">" | ">=" | "in" | "not" "in" ) range ] ;
range       = [ sum ] ( ".." | "..=" ) [ sum ] | sum ;
sum         = product { ( "+" | "-" ) product } ;
product     = unary { ( "*" | "/" | "%" ) unary } ;
unary       = "-" unary | postfix ;
postfix     = primary { "." name [ args ] | "?." name [ args ] | "[" expr { "," expr } "]" | args } ;
args        = "(" [ arg { "," arg } [ "," ] ] ")" ;
arg         = [ name ":" ] ( expr | lambda ) ;
lambda      = name "=>" expr | "(" name "," name ")" "=>" expr ;
primary     = literal | name | TypeName [ "." name ] | "(" expr ")" | list | record
            | ifExpr | matchExpr ;
ifExpr      = "if" condition "{" expr "}" { "else" "if" condition "{" expr "}" } "else" "{" expr "}" ;
matchExpr   = "match" expr "{" { pattern [ "if" expr ] "=>" expr NL } "}" ;
list        = "[" [ expr { "," expr } [ "," ] ] "]" ;
record      = "{" name ":" expr { "," name ":" expr } [ "," ] "}" ;

literal     = number [ unit ] | money | percent | string | date | dateTime
            | "true" | "false" | "none" ;
money       = ( "$" | CURRENCY ) number [ "/" unit ] ;        (* $450, CAD 450, $2.35/mi *)
percent     = number "%" ;
number      = digit { digit | "_" } [ "." digit { digit | "_" } ] ;
unit        = "lb" | "cwt" | "kg" | "t" | "ton" | "mi" | "km" | "in" | "ft" | "m"
            | "ft3" | "m3" | "min" | "h" | "d" | "°F" | "°C" | "pallet" | "piece" | "stop"
            | unit "/" unit ;                                 (* lb/ft3 for density *)
date        = "@" digit4 "-" digit2 "-" digit2 ;
dateTime    = date "T" digit2 ":" digit2 [ ":" digit2 ] ( "Z" | ( "+" | "-" ) digit2 ":" digit2 ) ;
string      = '"' { char | escape | "{" expr "}" | "{{" | "}}" } '"' ;
```

A keyword is never an identifier. Unit names are contextual: `lb` is a unit only directly after
a number, or after `/` in a money literal or a type. Elsewhere it is an ordinary name.
