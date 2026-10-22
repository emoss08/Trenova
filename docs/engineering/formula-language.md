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
tariff "Dry van TL"

/// Truckload linehaul with fuel, extra drops, hazmat and detention.

use std.fuel.{ stepPercent }
use std.accessorials.{ detention }
use org.accessorials.{ hazmat }

/// Paid per loaded mile.
input rate: Money/mi = $2.35/mi
/// Fuel is priced off this peg, one step per 5 cents.
input fuelPeg: Money/gal = $1.20/gal
input minimumCharge: Money = $450
input extraDropFee: Money = $75

let miles = shipment.distance
let linehaul = rate * miles
let drops = shipment.stops.count(s => s.kind == Delivery)

charge "Linehaul" (quantity: miles, rate: rate)

let price = fuel.price ?? fail "No fuel price on {rating.date}"
charge "Fuel surcharge" as FSC = linehaul * stepPercent(price, peg: fuelPeg, step: $0.05/gal, perStep: 0.5%)

if drops > 1 {
  charge "Extra drops" as XDROP (quantity: drops - 1, rate: extraDropFee)
}

if shipment.hazmat {
  charge "Hazmat" as HAZ = hazmat(miles)
}

for stop in shipment.stops where stop.dwell != none {
  let owed = detention(stop.dwell ?? 0 h, rate: $65/h)
  if owed > $0 {
    charge "Detention, stop {stop.sequence}" as DET = owed
  }
}

minimum minimumCharge
round to cents

test "a two-drop load bills one extra drop" {
  given fuel.price = $3.20/gal
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
- **Keywords:** `tariff library condition use input let var const fn type enum private charge
  note if else match for in where break continue return minimum maximum round require fail fails
  test given expect and or not true false none`. `as`, `to`, `up`, `down`, `half`, `even`,
  `cents`, `total`, `lines` and the unit names are contextual: they mean something only where the
  grammar expects them, and are ordinary names elsewhere (`fn billed(weight: Weight, to: Weight)`).
- **Text:** `"double quoted"`, with `{expr}` interpolation, `{{` for a literal brace and the usual
  escapes. There are no raw strings and no single quotes. Interpolation writes any value that
  can't be empty: numbers, percents, money, quantities, dates, enums (as their variant), and
  lists and records of such values, as `[a, b]` and `{ name: x, weight: 5 lb }`. An optional,
  or a list or record holding one, must be handled first (`?? fallback`, `if let`).
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
   save, doesn't preview and doesn't rate: `tariff.evaluate`, `tariff.test` and
   `tariff.scaffoldTest` check it first and run nothing, answering with its first finding
   (code `check`). A file that checks clean runs the way the checker promised.
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
| `Percent` | `18%` | multiplies as 0.18; prints as 18%; `.value` is the bare number, 18 |
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
| Volume | `ft3`, `m3`, and the liquid units `gal`, `l` |
| Duration | `min`, `h`, `d` |
| Temperature | `°F`, `°C` (comparison and difference only) |
| Count | `pallet`, `piece`, `stop`, and bare `Int` |
| Quotients | any two of the above, for densities and rates of use: `lb/ft3`, `mi/gal` |

The arithmetic rules:

- `+`, `-` and comparison take the same dimension: `500 lb + 2 cwt` is `700 lb`.
- `Money × Number`, `Money × Percent` and `Money / Number` give `Money`.
- A percent scaled by a number is a percent, in either order: `0.5% × 3` is `1.5%`, and
  `Percent / Number` is a `Percent`. `Percent ± Percent` is a `Percent`; `Percent × Percent` is
  an error.
- `Money/unit × quantity` gives `Money` (`$2.35/mi × 812 mi`), converting units as needed
  (`$18.50/cwt × 1_200 lb = $222`).
- `Money / quantity` gives a rate; `quantity / quantity` of one dimension gives a `Number`.
- Anything else is an error that says what was meant: "`rate` is money per hundredweight and
  `shipment.weight` is a weight, so `rate × shipment.weight` is money. `rate + shipment.weight`
  adds money to a weight."
- `.in(unit)` converts explicitly: `shipment.weight.in(cwt)`. The unit is written by name
  (`cwt`, `lb/ft3`) and must measure the value's dimension. On a rate it may also be a unit of
  the rate's denominator, which converts the denominator, or a whole rate unit written with
  `$`: `$0.185/lb.in(cwt)` and `$0.185/lb.in($/cwt)` are both `$18.50/cwt`. The type keeps its
  dimension and takes that display unit. `.value` gives the bare `Number` in the unit it was
  last written or converted to.

### Money and currency

A rating has exactly one currency: the contract's (the rule's, else the agreement's) or, when no
contract applies, the customer's billing currency. A money literal is written `$450` whatever that
currency is; `$` means "money in the rating currency", and a formula cannot mix currencies because
it can name only one. When the billing currency differs, the host passes the exchange rate in
force on the rating date and the module converts the finished bill: each line exactly, rounded the
way the bill rounds, with any difference on the rounding line, so the converted lines still add
up to the converted total. The rate is recorded on the bill. There is no `convert` and no currency literal inside the language.

### Optionals

A schema field that can be empty has type `T?` (`shipment.weight: Weight?`). Using one where a
`T` is needed is an error, with quick fixes for each way to handle it:

- `shipment.weight ?? 0 lb`: a fallback;
- `if let w = shipment.weight { … }`: bind it when present;
- `require let w = shipment.weight else "This rate needs the shipment's weight"`: refuse to rate
  without it, with `w` bound as a `Weight` from there on;
- `shipment.trailerType?.code`: optional chaining.

## Bindings

- `let name = expr`: immutable. This is the usual form.
- `var total = $0` followed by `total += line`: mutable, for accumulating inside a loop. A `var`
  never escapes its block's scope.
- `const DIESEL_BASE = $3.50`: a compile-time constant at file level, importable from a library.
  Constants and defaults (of inputs and of function parameters) are compile-time values:
  computed from literals and other constants only, never from the record, an input, a
  binding or another parameter.
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
- `in` tests membership in a list or range: `rating.date.weekday in [Sat, Sun]`,
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

Lambdas are `x => expr`, or `(item, index) => expr` for two parameters, the index counted from
0. List methods:

- **Filtering:** `where`, `map`.
- **Counting and totals:** `count`, `sum`, `min`, `max`, `avg`.
- **Tests:** `any`, `all` (write `not xs.any(…)` for "none"; `none` is the empty value).
- **Picking items:** `first`, `last`, `find`.
- **Ordering:** `sortBy`, `take`, `skip`, `reverse`, `distinct`.
- **Grouping:** `groupBy`, which gives `List<{ key, items }>`.
- **Size:** `len`, `isEmpty`.

A few examples:

```tariff
let deliveries = shipment.stops.where(s => s.kind == Delivery)
let heaviest = shipment.commodities.max(c => c.weight)
let palletCount = shipment.commodities.sum(c => c.pieces)
```

The rules the checker and the evaluator share:

- **`min` and `max`** with a lambda give the lambda's smallest or largest value, never the item
  that has it (`heaviest` above is a `Weight?`); without one, the smallest or largest item.
  Either way the result is optional, `none` for an empty list.
- **`sum`** of an empty list is the zero of its type: `$0`, `0 h`, `0`.
- **`avg`** is not optional: a `Number` for numbers (whole numbers too), else the values' own
  kind (`Money`, `Weight`). An empty list has no average, so `avg` of one stops the rating with
  an `argument` error; test `isEmpty` first when the list may be empty.
- **`sortBy`, `min` and `max`** order by a value that has an order: numbers, percents, money,
  quantities, dates and date-times. Text has no order.
- **Indexing** a list or a text, `xs[i]`, gives `T?`: `none` when `i` is outside it (negative
  or past the end), so it is handled like any optional (`xs[0] ?? fallback`, `xs[0]?.field`).

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
  `deadhead(empty: 80 mi, rate: rate)`. A default is a compile-time value (literals and
  constants), so it can't read the other parameters.
- **Purity:** functions are pure and closed. They can't bill a charge, and they see only their
  parameters, constants, other functions, types and imports: not the schema roots, the inputs, a
  file's `let`s or any `var`. So a function can be tested and reused anywhere.
- **No recursion:** the call graph must be acyclic, and the checker refuses a cycle. Every
  formula terminates by construction, and the meter is a second line of defence.

### Methods on values

A property is written without parentheses (`weight.value`, `date.weekday`, `list.len`); a method
with them (`amount.roundUp(to: $5)`).

| On | Properties | Methods |
|---|---|---|
| `Percent` | `value` (the bare number: `18%.value` is `18`) | |
| `Number` | | `round(places: Int = 0)`, `roundHalfEven(places: Int = 0)`, `floor()`, `ceil()` (both give `Int`), `abs()`, `min(x)`, `max(x)`, `clamp(lo, hi)`, `pow(n: Int)`, `sqrt()` |
| `Money` | | `round()` (to cents, half up), `roundHalfEven()`, `roundUp(to: Money)`, `roundDown(to: Money)`, `abs()`, `min(x)`, `max(x)`, `clamp(lo, hi)` |
| Quantities | `value` | `in(unit)` (on a rate, also a unit of its denominator: `in(cwt)`),  `round(to: q)`, `roundUp(to: q)` (weight breaks bill to the next hundred: `roundUp(to: 100 lb)`), `roundDown(to: q)`, `abs()`, `min(x)`, `max(x)`, `clamp(lo, hi)` |
| `Text` | `len`, `isEmpty` | `upper()`, `lower()`, `trim()`, `startsWith(t)`, `endsWith(t)`, `contains(t)`, `replace(a, b)`, `split(sep)`, `matches(pattern)` (a linear-time regular expression) |
| `Date`, `DateTime` | `year`, `month`, `day`, `weekday` (`Weekday`: `Mon` … `Sun`), `weekend`; `DateTime` also `hour`, `minute`, `date` | `addDays(n: Int)`; `DateTime` also `addHours(n: Int)` |
| `Duration` | `value` | `in(unit)` and the quantity methods |
| `List<T>` | `len`, `isEmpty` | the collection methods above |

Durations come from subtracting date-times (or dates, in whole days).

### Refusing to rate

- `fail "No rate for {lane}"` stops the rating with that message. It is shown on the shipment,
  in the Studio, and to Nova, and is translated.
- `require cond else "message"` and `require let x = optional else "message"` are the guard
  forms.

A failed rating is a rating error the user can act on, never a silent `$0`.

## Charges and the bill

A tariff's result is a **bill**: an ordered list of charge lines, notes, and a total.

```tariff
charge "Linehaul" (quantity: miles, rate: rate)          // amount = rate × quantity
charge "Fuel surcharge" as FSC = linehaul * fuelPercent
charge "Detention, stop {stop.sequence}" as DET (quantity: billable, rate: $65/h)
```

- **`charge "Label" [as CODE] [(quantity: q, rate: r)] [= amount]`** adds a line.
  - With `quantity` and `rate` the amount may be left out: it is `rate × quantity`, and both are
    printed on the invoice line (its quantity, unit price and rate unit). With an amount and no
    options the line shows the amount only.
  - **Uncoded charges are freight.** They sum to the shipment's freight charge, and each becomes
    its own freight line on the invoice, described by its label. A tariff that bills "Linehaul"
    and "Deficit weight" shows both.
  - **Coded charges are accessorials.** `as CODE` names an active code in the organisation's
    accessorial charge catalog (`FSC`, `DET`, `LIFT`; the checker refuses an unknown or inactive
    code). The line becomes a system-generated additional charge owned by the tariff (replaced on
    every re-rate), an accessorial invoice line carrying the code, and an EDI 210 charge whose
    code is the catalog code.
  - **Amounts.** A coded charge must be positive. An uncoded charge may be negative (a discount
    line) or zero; a zero line is dropped from the bill and noted on the trace. A bill whose
    freight lines sum below zero, or whose total is below zero, is a rating error.
  - **Labels** interpolate, and are user-facing text: shown as written on the invoice, which is
    the customer's document, and translated in the Studio's own chrome only.
  - Charges may appear inside `if`, `match` arms and `for` loops, never inside a function.
- **`note "…"`** adds a line with no amount, shown on the trace, the receipt and the invoice notes.
- **Total clauses** come last, in this order, and apply to the whole bill:
  - `minimum amount`: when the total falls short, the difference is billed as an uncoded
    "Minimum charge adjustment" line;
  - `maximum amount`: when the total exceeds it, an uncoded negative "Maximum charge adjustment"
    line brings it down;
  - `round to cents` (half up; the default when no `round` clause is written),
    `round half even to cents`, `round up to $5`, `round down to $1`: every line is rounded, and
    any difference between the rounded lines and the rounded total goes on an uncoded "Rounding"
    line, so the lines always add up to the total. The step may be any money expression (an
    input, so a contract sets it); it must be more than `$0`, which the checker enforces for a
    literal and the rating for any other value (an `argument` error).
- **One owner per code.** When a tariff bills a code that something else would also generate on
  the same shipment (a contract accessorial, the fuel surcharge program, a detention policy), the
  tariff's line wins, the other is not generated, and the trace says so. Fuel can no longer be
  billed twice.

The template's old breakdown lines, its minimum, maximum and rounding policy fields, and the rate
engine's discount, absolute-minimum, deficit-distance, guardrail and agreement-rounding steps are
gone: each is a line or a clause in the tariff, where it can be read, tested and changed.

## Contracts and inputs

A tariff's `input`s are its contract interface: they are what a rate agreement sets.

- **Where values come from**, first match wins: the rate agreement rule's input values, then the
  agreement's (contract-wide values such as a minimum), then the input's default. An input with
  no default that nothing sets refuses to rate, naming the rule and the input.
- **Rate agreement rules and agreements store `inputs`**: a map from input name to a value in the
  wire form. Saving checks every value against the tariff's declarations (type, unit, record
  shape) through `tariff.check`, so a rule cannot be saved with a missing or mistyped input.
- **The rule editor is the tariff's inputs form**, built from the declarations: labels from doc
  comments, unit-aware fields for quantities and rates, a table editor for list inputs such as
  weight breaks.
- **The rule loses its pricing columns.** `Rate`, the breaks table, `AllowDeficitRating`,
  `DiscountPercent`, `AbsoluteMinCharge`, `MinBillableDistance`, `MinCharge`, `MaxCharge`,
  `RoundingMode`, `FreightClassSource`, `FixedFreightClass`, `DensityScaleID` and `RateMatrixID`
  are removed; a rule is a lane, its eligibility, its dates and priority, a tariff and that
  tariff's inputs. The agreement loses `DefaultMinCharge`, `DefaultMaxCharge`, `RoundingMode` and
  `RoundingPrecision`. Currency and the buy-side margin guardrails stay.
- **Weight breaks are an input**: `input breaks: List<Break>` with `std.ltl.Break`, edited as a
  table in the rule form and priced by `std.ltl.deficit`.
- **Shipments keep the tariff and its inputs.** Adopting a contract writes the template and the
  resolved input values onto the shipment (replacing `BaseRate`), so every re-rate reapplies the
  whole contract, not just a rate.
- **Matrices are tables.** A rate matrix loses its formula template and rounding and gains a value
  type; tariffs read it by code (below).
- **The fallback template** (the billing control's) runs with its defaults. **The buy side** rates
  with `rating.mode == Buy` and `rating.sellTotal` set.

## Accessorials and conditions

```tariff
condition "Residential delivery"

/// The zip3s this contract treats as residential.
input residentialZip3s: List<Text> = []

lane.destination.zip3 in residentialZip3s and (shipment.weight ?? 0 lb) < 10_000 lb
```

- A **condition** file has inputs, bindings, functions and one final `Bool` expression. It
  replaces a contract accessorial's `ApplyCondition`; the row stores the condition's source, and
  `ConditionLanguageVersion` and the accessorial's unused `FormulaTemplateID` are removed, as is
  the never-evaluated `AccessorialCharge` template type.
- Every condition of one contract application is checked in one module call, on the sell side
  and, now, on the buy side.
- A contract accessorial row keeps its data pricing (method, amount, rate unit), and its free
  units and maximum amount are now honoured when it is applied.
- A tariff can bill any accessorial directly with `charge … as CODE`; the one-owner-per-code rule
  decides between the two.

## Fuel

Fuel surcharge programs stay the place surcharges are configured, and still add their line after
the tariff has rated, unless the tariff billed the program's code itself. Inside a tariff,
`fuel.price` is the index price **as of the rating date** (today it is the latest price), and
`std.fuel` computes surcharges in the language for contracts that price fuel themselves.

## Data

The record being rated is exposed in typed, unit-aware roots. Units are the ones the domain
stores: weights in pounds, temperatures in °F, lengths in feet, volumes in cubic feet, and
distances in the organisation's stored distance unit (miles or kilometres), which the unit system
now carries instead of assuming miles.

| Root | Fields |
|---|---|
| `shipment` | `proNumber: Text`, `status: ShipmentStatus`, `weight: Weight?` (commodity lines, else the shipment's weight), `pieces: Int?`, `distance: Distance`, `duration: Duration`, `linearFeet: Length`, `volume: Volume`, `density: Weight/Volume?`, `hazmat: Bool`, `reefer: Bool`, `temperature: { min: Temperature?, max: Temperature? }`, `freightClass: FreightClass?`, `highestClass: FreightClass?`, `serviceType: { code: Text, description: Text }`, `shipmentType`, `tractorType`, `trailerType` (each `{ code, description }?`), `stops: List<Stop>`, `commodities: List<Commodity>`, `pickup: Stop?`, `delivery: Stop?` |
| `lane` | `origin` and `destination`, each `{ city, state, zip, zip3, country: Text, zones: List<Text> }`; `interstate: Bool`; `crossBorder: Bool` |
| `customer` | `name: Text`, `code: Text` |
| `fuel` | `price: Money/gal?`, `priceDate: Date?`, `index: Text?` |
| `rating` | `date: DateTime` (in the organisation's time zone), `mode: RatingMode` (`Sell`, `Buy`), `sellTotal: Money?`, `currency: Text` |
| `tables` | the organisation's active rate matrices, by code |

- A `Stop` has `sequence: Int`, `kind: StopKind`, `status: StopStatus`, `appointment: Bool`,
  `window: { start: DateTime?, end: DateTime? }`, `arrival: DateTime?`, `departure: DateTime?`,
  `dwell: Duration?` (present when both arrival and departure are), `pieces: Int?`,
  `weight: Weight?` and `location: { city, state, zip, zip3, country, timezone: Text }`.
- A `Commodity` has `name: Text`, `freightClass: FreightClass?`, `weight: Weight`, `pieces: Int`,
  `dimensions: { length: Length?, width: Length?, height: Length? }`, `volume: Volume?`,
  `density: Weight/Volume?`, `hazmat: Bool` and `stackable: Bool`.
- The schema enums (`ShipmentStatus`, `StopKind`, `StopStatus`, `FreightClass`, `RatingMode`)
  are enums of the language; `FreightClass` variants are `Class50` through `Class500`.
- Volume also has the liquid units `gal` and `l`, for fuel prices.

### Rate tables

Every active rate matrix with one to four dimensions is a table, named by its code. A matrix
declares its **value type** (`Money`, `Money/cwt`, `Money/mi`, `Percent`, `Number`, …), and each
dimension's kind gives its key type: zones, zips, states, countries, freight classes, equipment
and service types are `Text` (or the enum) and match exactly; weight breaks, distance, pieces and
linear feet are quantities in bands.

- `tables.Zones[lane.origin.zip3, lane.destination.zip3]` gives the value type, optional
  (`Money/cwt?`): a miss is `none`, so `?? fail "…"` refuses to rate with a reason.
- Any identifier may follow `tables.`, in the matrix code's own spelling: `tables.ZONES`,
  `tables.fuelTiers`.
- An exact axis's key may be a list, tried in order: `tables.Zones[lane.origin.zones,
  lane.destination.zones]` takes the first pair of zones that has a cell.
- `tables.Zones.cell(…)` gives `{ value, minimum: Money?, deficit: Bool }?`, the cell's value
  with its minimum charge and deficit flag.
- `tables.Breaks.band(weight)` gives the matching band `{ min, max, value }?`, and
  `.interpolate(x)` interpolates between bands.
- The checker types each table from its definition, so a wrong number of keys, a weight given to
  a distance axis or an unknown table is an error, and completion lists a table's real keys.
  Normalisation (trim, upper case, zip3) and range overflow keep their per-dimension settings.

## Modules and the standard library

- `use std.ltl` or `use std.ltl.{ deficit, Break }` imports from the standard library, which is
  written in Tariff, ships inside the module and is versioned with it.
- `use org.accessorials.{ residential }` imports from the organisation's libraries, by name.
- Everything a library declares is exported unless marked `private`. A tariff version pins the
  version of every library it imports; publishing a library version never reprices a contract.

| Module | Declares |
|---|---|
| `std.ltl` | `type Break = { from: Weight, rate: Money/cwt, minimum: Money? }`; `type Rated = { weight: Weight, rate: Money/cwt, minimum: Money?, deficit: Bool }`; `fn deficit(breaks: List<Break>, actual: Weight, allow: Bool = true) -> Rated` (today's deficit rating: the cheaper of the actual weight's break and the next break's minimum weight); `fn billed(weight: Weight, to: Weight = 100 lb) -> Weight`; `fn densityClass(density: Weight/Volume) -> FreightClass` (the NMFC density scale) |
| `std.fuel` | `fn stepPercent(price: Money/gal, peg: Money/gal, step: Money/gal, perStep: Percent) -> Percent`; `fn perMile(price: Money/gal, peg: Money/gal, mpg: Number) -> Money/mi` |
| `std.time` | `fn weekend(at: DateTime) -> Bool`; `fn afterHours(at: DateTime, open: Int = 8, close: Int = 17) -> Bool`; `fn businessDuration(start: DateTime, end: DateTime, open: Int = 8, close: Int = 17) -> Duration` |
| `std.accessorials` | `fn detention(dwell: Duration, free: Duration = 2 h, rate: Money/h, increment: Duration = 15 min) -> Money`; `fn extraStops(stops: List<Stop>, included: Int, fee: Money) -> Money` |

## Tests

Tests live in the file they test. The Studio runs them on every change, saving runs them, and a
version with a failing test cannot be submitted for review.

```tariff
test "a 1,240 lb load bills at the next hundred pounds" {
  given shipment.weight = 1_240 lb
  given shipment.distance = 412 mi
  expect charge("Linehaul") == $365.30
  expect total == $511.20
}

test "a load with no weight refuses to rate" {
  given shipment.weight = none
  expect fails "This rate needs the shipment's weight"
}
```

- `given` overrides a field of the sample record, built from the schema's defaults (a one-stop
  pickup and delivery, no commodities) and the inputs' defaults; `given input.rate = $2.10/mi`
  overrides an input.
- In tests only, `pickup(…)`, `delivery(…)`, `splitPickup(…)` and `splitDelivery(…)` build a
  `Stop` and `commodity(…)` a `Commodity`; every argument is named and optional, and a stop's
  `sequence` defaults to its place in the list. After the `given`s, `shipment.pickup`,
  `shipment.delivery` and each stop's `dwell` are worked out again unless a `given` set them.
- `expect` takes a `Bool`; `charge("Label")` or `charge(CODE)` reads a line's amount (`Money?`),
  `total` the bill's total, `lines` the whole list (`{ label: Text, code: Text?, amount: Money,
  kind: LineKind }`, where `LineKind` is `Charge`, `Minimum`, `Maximum` or `Rounding`); `expect fails` (optionally with the message)
  expects a refusal.
- Tests replace the Studio's test-case rows and recorded scenarios: "Save as test" in the preview
  writes a `test` block from a real shipment's facts into the source.

## Module operations

The crate's `formula.*` operations are replaced by `tariff.*`, with the same envelope, budgets,
`Message`s and UTF-16 ranges as every other operation:

| Operation | Does |
|---|---|
| `tariff.check` | parses and checks a file with the libraries it imports, against the schema, the tables' shapes and the organisation's accessorial codes; returns diagnostics with fixes, the declared inputs (name, type, default, doc), the outline, symbols and inferred types (for inlay hints) and the imports it pinned |
| `tariff.evaluate` | rates a batch of programs (a tariff or condition with its libraries, input values and record) and returns, per program, the bill (charges with label, code, amount, quantity, rate and source range; notes; adjustments; total) or the refusal, plus lookups and per-statement values for the trace |
| `tariff.test` | runs a file's tests against the sample record; returns each test's result, the failing expectation and the values involved |
| `tariff.format` | formats a file (one canonical style) |
| `tariff.tokens` | the token stream for highlighting |
| `tariff.complete`, `tariff.hover`, `tariff.signature`, `tariff.definition`, `tariff.rename`, `tariff.inlays` | the editor's language features at a cursor or for a range |
| `tariff.scaffoldTest` | writes a `test` block from a real shipment's record and its bill, for the Studio's "Save as test" |
| `tariff.catalog` | the standard library's modules, types and functions, and every built-in method, with their docs |

### Wire values

Values cross the boundary as JSON, typed by tag where JSON has no type of its own, and numbers are
always decimal strings read exactly:

| Value | Wire form |
|---|---|
| `Bool`, `Text` | JSON `true`/`false`, a JSON string |
| `Int`, `Number` | `{"$n": "1200"}` |
| Money, quantities, rates, percents | `{"$q": "2.35", "u": "$/mi"}`, `{"$q": "1200", "u": "lb"}`, `{"$q": "18", "u": "%"}`, `{"$q": "450", "u": "$"}` |
| `Date`, `DateTime` | `{"$date": "2026-10-22"}`, `{"$date": "2026-10-22T08:00:00-05:00"}` |
| Enum variants | a JSON string (`"Delivery"`), read as the declared enum |
| `none` | JSON `null` |
| Lists, records | JSON arrays and objects (record keys never start with `$`) |

Input values on rules and agreements, the record the host sends, the bill and the trace all use
these forms, so a value never loses its unit or its precision between Postgres, Go, the module
and the browser.

## Storage and server changes

- **Formula templates** keep their name, versions, review and scheduling, and store one `source`
  instead of `expression`, `variableDefinitions`, `breakdownDefinitions`, the policy fields,
  `type` and `languageVersion`. A version also stores the inputs and pinned imports `tariff.check`
  derived, for contract forms and queries. The test-case table and the scenario columns are
  dropped.
- **Formula libraries** are a new organisation-level record with the same versioning and review as
  templates (`formula_libraries`, `formula_library_versions`), with GraphQL reads, REST writes,
  permissions, audit, agent tools and an editor.
- **Rate agreement rules and agreements** gain `inputs` and lose the pricing columns above; the
  breaks table is dropped. **Rate matrices** gain `valueType` and lose their template, rounding
  and currency (a table's money is in the rating currency). Density scales are removed: the NMFC
  scale is `std.ltl.densityClass`, and a contract's own scale is a `match` or a table.
  **Shipments** replace `baseRate` with `tariffInputs`. **Contract accessorials** store a
  `condition` source.
- **The rate engine** resolves the rule, gathers the input values and makes one `tariff.evaluate`
  call; the bill becomes the shipment's freight lines and tariff-owned additional charges, the
  quote's linehaul, fuel and accessorial amounts, the carrier assignment's base, fuel and
  accessorials on the buy side, and the shipment's rating detail (which now shows the bill).
- **Invoices** get one freight line per uncoded charge; **EDI 210** sends each accessorial line's
  charge code and `400` for freight lines.
- **Accessorial charge codes** are unique per organisation, case-insensitively, and match
  `^[A-Z][A-Z0-9_]{2,9}$`, so `as CODE` always names one charge.
- **Manual rates.** A shipment rated by hand (rate locked, or the manual rating method) has one
  freight line, its typed amount; it runs no tariff. A tendered rate from an EDI 204 or a document
  is a manual rate.
- **The schema endpoint** publishes the roots above with types and units; the Studio's preview
  returns the bill, the trace and the test results; Nova's tools read and propose tariff source,
  run `tariff.check` and `tariff.test` before showing anything, and its prompt carries this
  reference.

## The editor

The Studio edits one source file. Everything else is a view of it, computed by the module:

- diagnostics with quick fixes, completion (types, units, docs, schema enums, table keys,
  accessorial codes), hover (type, unit, doc, the last preview's value), signature help, go to
  definition (into libraries), rename, inlay hints (inferred types and each binding's previewed
  value), folding and the outline (inputs, functions, charges, tests);
- highlighting from `tariff.tokens`, so no TypeScript lexer remains;
- the inputs panel, from the `input` declarations, holding the preview's sample values;
- the tests panel, from the `test` blocks, run in the browser on every change;
- the bill preview: each charge with its code, quantity and rate, the notes and the adjustments.

## What stays the same

- **The engine.** One `trenova-rules` crate, one `wasm32-unknown-unknown` module, the same ABI,
  hosts, pools, metrics and build.
- **Exact decimals end to end, and metering.** Steps, text bytes and items, plus wazero closing
  the call on deadline.
- **Determinism.** No clock (`rating.date` is passed in), no randomness, map order by key.
- **Diagnostics.** `Message`s with nested arguments, translated in both hosts, with UTF-16 ranges.
- **One call per rating** on the server, and one per contract application for conditions.

## Decisions

1. **Name.** The language is Tariff; a freight tariff is the published rate document, which is
   what these files are. The product keeps calling them formula templates.
2. **Syntax.** Rust- and TypeScript-flavoured: braces, no semicolons, `let`/`var`/`fn`/`match`,
   `x => …` lambdas, one spelling per idea.
3. **Types.** Static, inferred inside bodies, annotated at boundaries; units and money are types;
   records and enums are the only user types; no generics, classes or escape hatches.
4. **The formula is the bill.** Uncoded lines are freight, coded lines are accessorials, total
   clauses are in the source, and one owner per code.
5. **The contract is the inputs.** Rules and agreements set a tariff's inputs; their pricing
   columns are removed.
6. **Inputs, tests and conditions live in the source.**
7. **No recursion and no `while`.** Termination holds by construction.
8. **Organisation libraries**, versions pinned per template version, and a standard library
   written in Tariff.

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
- A `match` arm's guard is an expression that ends at the arm's `=>`: in `x if ready => 1`,
  `ready =>` is the guard's last value and the arrow, never a lambda.

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
charge      = "charge" string [ "as" CODE ] [ "(" option { "," option } ")" ] [ "=" expr ] NL ;
                                                              (* "= expr" may be left out only with quantity: and rate: *)
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
            | "round" [ "half" "even" | "up" | "down" ] "to" ( "cents" | expr ) NL ;
given       = "given" lvalue "=" expr NL ;                    (* tests only *)
expect      = "expect" ( expr | "fails" [ string ] ) NL ;     (* tests only *)
lvalue      = name { "." name } ;                             (* shipment.weight, input.rate *)

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
money       = "$" number [ "/" unit ] ;                       (* $450, $2.35/mi, $18.50/cwt *)
percent     = number "%" ;
number      = digit { digit | "_" } [ "." digit { digit | "_" } ] ;
unit        = "lb" | "cwt" | "kg" | "t" | "ton" | "mi" | "km" | "in" | "ft" | "m"
            | "ft3" | "m3" | "gal" | "l" | "min" | "h" | "d" | "°F" | "°C" | "pallet" | "piece" | "stop"
            | unit "/" unit ;                                 (* lb/ft3 for density *)
date        = "@" digit4 "-" digit2 "-" digit2 ;
dateTime    = date "T" digit2 ":" digit2 [ ":" digit2 ] ( "Z" | ( "+" | "-" ) digit2 ":" digit2 ) ;
string      = '"' { char | escape | "{" expr "}" | "{{" | "}}" } '"' ;
```

A keyword is never an identifier. Unit names are contextual: `lb` is a unit only directly after
a number, or after `/` in a money literal or a type. Elsewhere it is an ordinary name, except
as the argument of `.in(…)`, which reads `cwt` or `lb/ft3` as a unit. `$` directly followed
by `/` and a unit is the money unit, so a whole rate unit is written `.in($/cwt)`. `in` directly
after a number is inches unless what follows starts a value: `48 in` is a length, and in
`5 in [5, 6]` it is the membership keyword.
