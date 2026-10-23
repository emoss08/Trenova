# Tariff: Trenova's rating language

This is the language reference for Tariff, the rating language `trenova-rules` implements. It
replaces the expr-flavoured formula language. Nothing has launched, so nothing here is held
back by compatibility: every formula, seed, catalog template, corpus case, prompt and doc is
rewritten.

For the syntax at a glance, see the [cheat sheet](tariff-cheat-sheet.md).

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
  declaration, enum variant or record field, and is shown in hover, completion, signature help
  and the inputs form. The standard library's doc comments are catalog messages, translated
  like the rest of the product; doc comments in an organisation's libraries and tariffs are shown
  as their author wrote them.
- **Identifiers:**
  - names are camelCase (`extraDropFee`);
  - types and enum variants are PascalCase (`Money`, `Delivery`);
  - charge codes are UPPERCASE (`FSC`), matching the accessorial charge catalog's 10-character
    codes.
- **Keywords:** `tariff library condition use input let var const fn type enum private charge
  note if else match for in where break continue return minimum maximum round require fail fails
  test given expect and or not true false none`. `as`, `to`, `up`, `down`, `half`, `even`,
  `cents`, `total`, `lines`, `applies`, `best`, `worst`, `option`, `property`, `vary`, `assume`,
  `nondecreasing`, `nonincreasing` and the unit names are contextual: they mean something only
  where the grammar expects them, and are ordinary names elsewhere (`fn billed(weight: Weight, to:
  Weight)`, `let option = 2`). `assume` followed directly by `(` is a call (`assume(x)`
  calls a function named `assume`); with a space before the parenthesis it starts the
  statement, so `assume (shipment.weight ?? 0 lb) > 10 lb` is an `assume`.
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
  - date: `@2026-10-22`; date-time: `@2026-10-22T08:00-05:00`;
  - time of day: `@17:30`, `@07:30:15` (hours `00` to `23`, seconds optional; no date and no
    offset).

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
   one dimension, dates, date-times, times of day, durations, and the two **ordered enums**,
   `FreightClass` and `Weekday`, which order by their declared variants (`Class50 < Class55 <
   … < Class500`, `Mon < Tue < … < Sun`). Every other enum, schema or user, has no order, and
   the checker says which enums do. Text has no order for these operators: a text range (see
   "Ranges") is the one place text is ordered, so a band of zip codes is written
   `zip in "30000"..="31999"`, never `zip >= "30000"`.
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
| `Time` | `@17:30`, `@07:30:15` | a wall-clock time of day, no date and no offset; `DateTime.time` is the stop-local time |
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

### Times of day

Business hours, cut-offs and delivery windows are times on a dock's clock, not instants, so
they have a type of their own:

- **Values.** `Time` runs from `@00:00` to `@23:59:59`, to the second. `stop.arrival?.time` (any
  `DateTime`'s `.time`) is the time the date-time shows in its own offset: the stop's local
  time, whatever offset the host sent.
- **Order.** Times compare, form ranges and match as ranges in wall-clock order within one day:
  `at.time in @08:00..@17:00`, `match at.time { ..@08:00 => … @08:00..@17:00 => … @17:00.. => … }`.
  A `match` on a `Time` is exhaustive once its ranges reach from `@00:00` to `@23:59:59`.
- **Arithmetic.** `Time - Time` is the `Duration` between them on the same day, negative when
  the right side is later (`@17:00 - @08:30` is `8.5 h`, `@08:00 - @09:00` is `-1 h`). There
  is no `Time + Duration`: it would have to wrap at midnight or leave the day, and neither is a
  time of day. Move a date-time instead (`at.addMinutes(90)`) and read its `.time`.
- **No `Date.at(time)`.** A date and a time of day name an instant only together with an
  offset, and the offset a stop's clock shows on a given date (daylight saving included) is the
  time zone database's knowledge, which the module does not carry. Rather than guess one, the
  language keeps the two apart: compare `at.time` with a `Time`, and build date-times from the
  date-times the record gives (`at.addDays(n)`, `at.addHours(n)`, `at.addMinutes(n)`).
- **Interpolation** writes `17:30`, or `07:30:15` when the seconds aren't zero.

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
- `input name: Type = default where constraint`: a value the contract or template supplies.
  Inputs are only declared at file level; their doc comments label the Studio's inputs form, and
  the optional `where` constraint bounds the values a contract may set (see "Input constraints").

A name may not shadow a schema root, an input, a function or another name in scope, and a
tariff's own code may not name anything `bill` (the bill so far, below). That last rule is
today's `let` rule, kept.

## Statements

A tariff's body is a sequence of statements, run top to bottom. A block is `{ … }`, and every
block is a scope.

| Statement | Meaning |
|---|---|
| `let`, `var`, `const` | bindings (above) |
| `name = expr`, `name += expr`, `name -= expr` | reassign a `var` in scope; the type cannot change |
| `charge "Label" as CODE (options) = expr` | add a line to the bill (below); tariffs only |
| `best "Label" { option "A" { … } option "B" { … } }`, `worst "Label" { … }` | bill the cheapest (or dearest) of several methods (below); written as the value of a `let`, `var` or assignment (`let linehaul = best "Linehaul" { … }`) it also gives the kept option's subtotal; tariffs only |
| `note "Text {interpolated}"` | add a note to the bill: no amount, shown on the rate trace, the receipt and the invoice's notes |
| `review "Text {interpolated}"` | bill as usual, and ask for the rating to be approved before it is used, with this reason (below); tariffs only |
| `if cond { … } else if cond { … } else { … }` | conditional; as a statement the branches need no value |
| `if let x = optional { … } else { … }` | bind an optional when present |
| `match value { pattern => { … } … }` | as a statement, arms are blocks |
| `for x in list { … }` | iterate a list or a range of `Int`s, in order |
| `for x in list where cond { … }` | iterate only the items `cond` holds for |
| `break`, `continue` | leave or skip to the next iteration of the innermost `for` |
| `require cond else "message"`, `require let x = optional else "message"` | refuse to rate unless it holds |
| `fail "message"` | refuse to rate |
| `return expr` | functions only |
| `minimum`, `maximum` (each also `on freight`), `round` | total clauses, last in a tariff (below) |
| `test "name" { … }`, `property "name" { … }` | a test or a property (below); file level only |

There is no `while` and no recursion. A `for` walks a list or a range whose length is known
when the loop starts, so every formula terminates. A function body is the same statement
language without `charge`, `note`, `review`, `best`, `worst`, total clauses, `test` or
`property`.
Functions compute values; tariffs bill them.

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

### Ranges

A range is written after `in`, as a `match` pattern, as an input's `where` constraint, as a
property's `vary` values, or (of `Int`s) after `for … in`. Its ends are values of one type:

- **Ordered values:** numbers, money, quantities, rates, percents, dates, date-times, times of
  day, and the ordered enums (`rating.date.weekday in Mon..=Fri`,
  `shipment.freightClass in Class50..=Class85`).
- **Text**, in Unicode scalar order, character by character:
  `lane.origin.zip in "30000"..="31999"`, `lane.destination.zip3 in "300".."320"`. This is how
  a zip or postal range is written; `<` and `>` stay refused on text, and their error points
  here. Because the comparison is character by character, `"2"` falls inside `"10".."300"`
  (it sorts after `"1"` and before `"3"`), so the checker warns (`text-range`) when a range's two literal ends
  have different lengths: zip ranges are only right when every code has the ends' length.
  The empty text `""` is the smallest text, so `.."300"` starts there.

A text range is never walked (`for`) or varied (`vary`): there is no next text to step to.

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
- A range pattern's ends are literals, or variants of an ordered enum: `"300".."320"` on text,
  `@08:00..@17:00` on a time, `Class50..=Class85` on a `FreightClass`, `Mon..=Fri` on a
  `Weekday`.
- A `match` must be exhaustive: the checker names the values no arm covers. A range of an
  ordered enum covers the variants from its start to its end, so `Mon..=Fri` and `Sat | Sun`
  cover a `Weekday`; ranges on a `Time` cover it once they reach from `@00:00` to `@23:59:59`;
  text ranges never cover all text without an open end (`"500"..`) or `_`.
- Overlapping ranges are an error with both arms named, on every type that has ranges: text
  ranges and text literals inside them, time ranges, and variant ranges that share a variant.
  The weight-break table can't double-count, which is the same rule the matrix coverage check
  applies to rate matrices.

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
  quantities, dates, date-times, times of day and the ordered enums (`FreightClass`,
  `Weekday`): `shipment.commodities.max(c => c.freightClass ?? Class50)` is the highest class.
  Text has no order here.
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
| `Date`, `DateTime` | `year`, `month`, `day`, `weekday` (`Weekday`: `Mon` … `Sun`), `weekend`, `holiday`; `DateTime` also `hour`, `minute`, `date`, `time` (the `Time` on its own clock) | `addDays(n: Int)`; `DateTime` also `addHours(n: Int)`, `addMinutes(n: Int)` |
| `Time` | `hour` (0 to 23), `minute` | |
| `FreightClass` | `number` (the class as a `Number`: `Class77_5.number` is `77.5`) | |
| `Duration` | `value` | `in(unit)` and the quantity methods |
| `List<T>` | `len`, `isEmpty` | the collection methods above |

Durations come from subtracting date-times, dates (in whole days) or times of day (on one day,
negative when the right side is later).

`holiday` is true when the date (for a `DateTime`, its calendar date in its own offset, as
`weekend` reads it) is the date of one of `rating.holidays`, the organisation's holidays. It is
the one property that reads the record, so it is written where the record can be read: a
tariff's or condition's body and tests. A function, a constant and a default can't read it (a
function sees only its parameters), and the checker refuses `.holiday` there (`closed-function`
in a function, `not-constant` in a constant or default), saying to pass the answer in: `fn
fee(holiday: Bool)` called as `fee(holiday: stop.arrival?.holiday ?? false)`. For the same
reason `std.time` has no `holiday` function. Each `.holiday` is metered as one step per holiday
it compares.

### Refusing to rate

- `fail "No rate for {lane}"` stops the rating with that message. It is shown on the shipment,
  in the Studio, and to Nova, and is translated.
- `require cond else "message"` and `require let x = optional else "message"` are the guard
  forms.

A failed rating is a rating error the user can act on, never a silent `$0`.

## Charges and the bill

A tariff's result is a **bill**: an ordered list of charge lines, notes, review reasons, and a
total.

```tariff
charge "Linehaul" (quantity: miles, rate: rate, minimum: $450)   // amount = rate × quantity, at least $450
charge "Fuel surcharge" as FSC = bill.freight * fuelPercent
charge "Detention, stop {stop.sequence}" as DET (quantity: billable, rate: $65/h)
```

- **`charge "Label" [as CODE] [(quantity: q, rate: r, minimum: lo, maximum: hi)] [= amount]`**
  adds a line. Each option is written at most once, by name, in any order.
  - With `quantity` and `rate` the amount may be left out: it is `rate × quantity`, and both are
    printed on the invoice line (its quantity, unit price and rate unit). With an amount and no
    options the line shows the amount only.
  - **Line limits.** `minimum: lo` makes the line bill at least `lo`, and `maximum: hi` at most
    `hi` (both money). The amount (the written one, or `rate × quantity`) is worked out exactly,
    then raised to `lo` when it falls short, then lowered to `hi` when it exceeds it; a `lo`
    above `hi` is an error (the checker's when both fold, else the rating's `argument` error).
    The line keeps its label, code, quantity and rate and bills the limited amount, and records
    which limit applied and the amount it replaced (`limit` on the wire), so the invoice and the
    trace can say "minimum applied" next to the quantity and rate that no longer multiply to
    it. One line, not a line and an adjustment, is the shape because a limit belongs to the
    charge: a coded charge's minimum stays that accessorial (an adjustment line would be
    uncoded, so freight), `charge(…)`, `bill.charge(…)` and `best` see the amount the customer
    is billed, and the one-owner-per-code rule still sees one line. The limits apply before the
    zero and sign rules below, so a line whose amount comes out at `$0` with a `minimum:` bills
    the minimum; put the charge under an `if` when it applies only sometimes.
  - **Uncoded charges are freight.** They sum to the shipment's freight charge, and each becomes
    its own freight line on the invoice, described by its label. A tariff that bills "Linehaul"
    and "Deficit weight" shows both.
  - **Coded charges are accessorials.** `as CODE` names an active code in the organisation's
    accessorial charge catalog (`FSC`, `DET`, `LIFT`; the checker refuses an unknown or inactive
    code). An unknown code is reported once, at its first use, with every other use as related
    places. It suggests a code spelled nearly the same, or else a catalog code whose description
    matches the charge's label ("`FSC` isn't in the accessorial charge catalog. `FUEL` is "Fuel
    surcharge"; did you mean `FUEL`?"), with a fix that writes it at every use, and a second fix
    that bills the line as freight instead by removing `as CODE`. The Studio adds **Add CODE to
    accessorial charges**, which opens the catalog's create form with the code filled in and the
    charge's label as its description. The line becomes a system-generated additional charge owned by the tariff (replaced on
    every re-rate), an accessorial invoice line carrying the code, and an EDI 210 charge whose
    code is the catalog code.
  - **Amounts.** A coded charge must be positive. An uncoded charge may be negative (a discount
    line) or zero; a zero line is dropped from the bill and noted on the trace. A bill whose
    freight lines sum below zero, or whose total is below zero, is a rating error.
  - **Labels** interpolate, and are user-facing text: shown as written on the invoice, which is
    the customer's document, and translated in the Studio's own chrome only.
  - Charges may appear inside `if`, `match` arms and `for` loops, never inside a function.
- **`note "…"`** adds a line with no amount, shown on the trace, the receipt and the invoice notes.
- **`review "…"`** asks for the rating to be approved before it is used, with its interpolated
  text as the reason: `review "Hazmat over 500 miles: check the permit cost"`. It may be written
  wherever a `note` may (a tariff's top level and the blocks inside it, options included), never
  in a function, test, library or condition. The rating still succeeds and bills as usual; the
  bill carries every review reason reached, in order (`reviews`, shaped exactly like `notes`),
  and the host holds a rated bill that has any for approval instead of applying it. A losing
  `best` option's reviews are dropped with its lines and notes. A refused rating has no bill,
  so no reviews.
- **Total clauses** come last, in this order, each written at most once:
  1. `minimum amount on freight`: when the freight lines (the uncoded charge lines) add up to
     less, the difference is billed as an uncoded "Minimum charge adjustment" line;
  2. `maximum amount on freight`: when the freight lines add up to more, an uncoded negative
     "Maximum charge adjustment" line brings them down;
  3. `minimum amount`: when the whole bill's total (the freight adjustments included) falls
     short, the difference is billed as an uncoded "Minimum charge adjustment" line;
  4. `maximum amount`: when the total exceeds it, an uncoded negative "Maximum charge
     adjustment" line brings it down;
  5. `round to cents` (half up; the default when no `round` clause is written),
     `round half even to cents`, `round up to $5`, `round down to $1`: every line is rounded, and
     any difference between the rounded lines and the rounded total goes on an uncoded "Rounding"
     line, so the lines always add up to the total. The step may be any money expression (an
     input, so a contract sets it); it must be more than `$0`, which the checker enforces for a
     literal and the rating for any other value (an `argument` error).

  The freight clauses come first because a whole-bill clause must see the freight the customer
  is actually billed: `minimum $400 on freight` then `minimum $450` tops freight up to $400 and
  only then the whole bill to $450. An adjustment line records the scope it applied to (`on` on
  the wire: `freight` for the freight clauses, `null` for the whole bill's); the host labels it
  in the reader's language. Every limit compares exact amounts: lines, line limits and clauses
  are all worked out before anything is rounded, and rounding (step 5) runs last, as always. So
  the rounded amounts can sit off a limit by the rounding: each line moves by at most half a
  cent, the total moves to its step, and the rounding line, which is freight, takes up the
  difference, so the billed freight can differ from a freight limit by up to half a cent per line
  of the bill, plus the total's step. A freight minimum above a whole-bill maximum (or any minimum above the maximum of
  its scope) satisfies nothing and is an error when both fold.
- **One owner per code.** When a tariff bills a code that something else would also generate on
  the same shipment (a contract accessorial, the fuel surcharge program, a detention policy), the
  tariff's line wins, the other is not generated, and the trace says so. Fuel can no longer be
  billed twice.

The template's old breakdown lines, its minimum, maximum and rounding policy fields, and the rate
engine's discount, absolute-minimum, deficit-distance, guardrail and agreement-rounding steps are
gone: each is a line or a clause in the tariff, where it can be read, tested and changed.

### Choosing between methods

Contracts often bill "the lower of" two methods: per mile or flat, LTL or volume, the actual
weight or the next break. `best` writes that down and keeps the losing methods visible:

```tariff
best "Linehaul" {
  option "Per mile" {
    charge "Linehaul" (quantity: shipment.distance, rate: rate)
  }
  option "Flat" {
    charge "Flat rate" = flatRate
  }
  option "Volume" {
    require (shipment.weight ?? 0 lb) >= 6_000 lb else "Volume rates start at 6,000 lb"
    charge "Volume" = volumeRate * (shipment.weight ?? 0 lb)
  }
}
```

- **What runs.** Each `option` block runs in order, in its own scope: it reads every name in scope
  where the `best` is written, and its own `let`s end with it. It may not assign a `var` declared
  outside the `best` (the checker refuses it), so a losing option leaves no trace in the rest of
  the file.
- **What is compared.** An option's subtotal is the exact sum of the charge lines it billed
  (freight and coded, before rounding). `best` keeps the option with the lowest subtotal, `worst`
  the one with the highest (the buy side's "greater of", a floor price); a tie goes to the option
  written first. The chosen option's lines and notes join the bill where the `best` is written;
  the others are dropped. Total clauses still apply to the whole bill afterwards.
- **As a value.** Written as the whole value of a `let`, `var` or assignment, a choice bills
  exactly as the statement does and also gives the kept option's subtotal, as `Money` (the
  example below). That is the only place a choice gives a value: it bills lines, so it stands where a statement
  does, and the binding names what it billed. Anywhere else inside an expression (`linehaul * 2`
  around it, a charge's amount, a condition, a lambda) is an error (`misplaced`): bind it first.
  When every option refuses, the rating refuses, so the binding is never reached without a
  value.
- **Refusing options.** An option that refuses (`fail`, `require`, `?? fail`) is out of the
  running: that is how an option says "I don't apply to this shipment". When every option refuses,
  the rating refuses with the first option's message. An evaluation error inside an option stops
  the rating, as anywhere else: errors are bugs, not choices.
- **The trace.** Every evaluation of a `best` or `worst` is recorded with each option's subtotal
  or refusal and which one won, shown in the Studio's preview and on the shipment's rating detail;
  its `choice` event carries the kept option's `subtotal`, the value a `let` binds.
- **Rules.** A `best` has a label, unique among the file's choices, and at least two options, each
  with a label unique within it. It may appear wherever a `charge` may, including inside `if`,
  `for`, `match` arms and another option, as a statement or as the value of a `let`, `var` or
  assignment there. An option with no `charge` on any path is a warning: it bills `$0`, so
  `best` always picks it.
- **In tests**, `chosen("Linehaul")` is the label of the option the last evaluation of that choice
  kept, as `Text?` (`none` when the choice was not reached or every option refused):
  `expect chosen("Linehaul") == "Flat"`. Comparing it (`==`, `!=`, `in [...]`) with a label that
  is no option of that choice is an error (`unknown-option`), with the nearest option's label as
  a fix.

The value form, pricing fuel on whichever linehaul was kept:

```tariff
let linehaul = best "Linehaul" {
  option "Per mile" { charge "Linehaul" (quantity: shipment.distance, rate: rate) }
  option "Flat" { charge "Flat rate" = flatRate }
}
charge "Fuel surcharge" as FSC = linehaul * fuelPercent
```

### Reading the bill so far

A charge often depends on what was billed before it: fuel on the linehaul a `best` kept, a
discount on the freight. `bill` reads the bill being built, at the line where it is read:

```tariff
best "Linehaul" {
  option "Per mile" { charge "Linehaul" (quantity: shipment.distance, rate: rate) }
  option "Flat" { charge "Flat rate" = flatRate }
}
charge "Fuel surcharge" as FSC = bill.freight * 32%
if bill.charge("Flat rate") != none {
  note "Billed at the flat rate, with {bill.charge(FSC) ?? $0} of fuel"
}
```

| Read | Type | Is |
|---|---|---|
| `bill.freight` | `Money` | the exact sum of the uncoded lines billed so far |
| `bill.accessorials` | `Money` | the exact sum of the coded lines billed so far |
| `bill.total` | `Money` | both |
| `bill.charge("Label")`, `bill.charge(CODE)` | `Money?` | the sum of the lines billed so far with that label (as interpolated) or code; `none` when there are none |

- **So far, in source order.** Statements run top to bottom, so a read sees exactly the lines
  billed before it, and the same file always reads the same amounts. Amounts are exact, before
  any rounding, and after each line's own `minimum:` and `maximum:`, so fuel worked out after a
  floored linehaul uses the floored amount. Zero lines are not on the bill, and total clauses
  have not run yet: their adjustments are never read.
- **Inside a `best` option**, `bill` reads the bill outside the choice plus the lines that option
  has billed so far: never another option's lines, so each method is priced as if it were the
  one kept.
- **Where.** `bill` is read in a tariff's top level and the blocks inside it. A function can't
  read it (pass the amount in), and neither can a condition or a library, which have no bill.
  A test reads the finished bill instead, through `total`, `lines`, `charge(…)` and `reviews`.
  Nothing in a tariff may be named `bill`.
- **Order warnings.** `bill.charge("Label")` or `bill.charge(CODE)` naming a label or code the
  tariff bills only later in source order is a warning (`bill-order`: "`FSC` is billed after
  this line, so this reads `none`"), with the line that bills it as a related place. A charge in
  a `for` loop around the read may bill before it, on an earlier iteration, and a charge in an
  option of a choice that is over before the read may have joined the bill, so neither is
  later. A label or code the tariff never bills is a warning too (`not-billed`), as in tests.

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
- **Input constraints** (below) bound what a contract may set.
- **The fallback template** (the billing control's) runs with its defaults. **The buy side** rates
  with `rating.mode == Buy` and `rating.sellTotal` set.

### Input constraints

A `where` clause bounds the values an input accepts, so a fat-fingered contract value is refused
when it is saved instead of billed:

```tariff
/// Paid per loaded mile.
input rate: Money/mi = $2.35/mi where $0.50/mi..=$15/mi
input freeStops: Int = 2 where 0..=10
input service: Service = Standard where [Standard, Guaranteed]
```

- **Forms.** A constraint is a compile-time value (literals and constants, like a default):
  - a range of the input's type, for an ordered type (numbers, `Int`, money, quantities, rates,
    percents, dates, date-times, times of day, the ordered enums) or text: `lo..hi` includes
    `lo` and excludes `hi`, `lo..=hi` includes both, and either end may be left off (`1..`,
    `..=$15/mi`, `where @00:00..=@23:59`, `where "000"..="999"`, `where Class50..=Class100`);
  - a list of the input's type, for any type with equality: the value must be one of them.
  For an optional input (`T?`) the constraint applies when a value is set; `none` always passes.
  For a `List<T>` input of an ordered or comparable `T`, it applies to every item. Record inputs
  take no constraint.
- **The default must satisfy it**; the checker refuses one that does not.
- **Contract values.** `tariff.check`'s `values` validation refuses a rule or agreement value
  outside the constraint with an `input-value` finding that names the bounds: "`rate` must be
  from $0.50/mi to $15.00/mi; $235.00/mi is outside it". `tariff.check` reports each input's
  constraint, so the rule editor shows the bounds and validates as the user types.
- **At rating time** a value outside the constraint (a value saved before the constraint was
  added) stops the rating with an `input` error naming the input, the value and the bounds.
- **In tests and properties**, a `given input.x` outside the constraint is an error (`constraint`)
  naming the bounds, since no contract can set that value, so a test built on it proves nothing.
  For a list constraint it suggests the nearest listed value, with a fix that writes it. A
  property's `vary input.x` with no range of its own varies over the constraint.
- **What the bounds decide.** Where an input is read by its own name, the checker knows its
  values, and warns (`bounds`) about code the bounds make dead. It is a warning because the code
  runs; the bill is unaffected.
  - A comparison with a compile-time value (`==`, `!=`, `<`, `<=`, `>`, `>=`, `in [...]`,
    `in lo..hi`) whose answer the bounds fix is "never true" or "always true", quoting the
    constraint: with `input rate: Money/mi where $0.50/mi..=$15/mi`, `rate > $20/mi` is never
    true and `rate >= $0.50/mi` always is; with `input zip: Text where "30000"..="39999"`,
    `zip in "40000"..` is never true.
  - Against a list constraint, a value outside the list names the values the input takes, and
    suggests the nearest of them with a fix: `service == "Guarenteed"` is never true, because
    `service` is only ever `"Standard"` or `"Guaranteed"`; did you mean `"Guaranteed"`?
  - A `match` arm whose pattern (a value, a range or a variant) the bounds rule out is "never
    matched".
  An input read through another binding (`let s = service`), or compared with a value computed
  at run time, is not judged.

## Accessorials and conditions

```tariff
condition "Residential delivery"

/// The zip3s this contract treats as residential.
input residentialZip3s: List<Text> = []

lane.destination.zip3 in residentialZip3s and (shipment.weight ?? 0 lb) < 10_000 lb

test "a residential zip3 applies" {
  given input.residentialZip3s = ["752"]
  given lane.destination.zip3 = "752"
  expect applies
}

test "a heavy load does not" {
  given input.residentialZip3s = ["752"]
  given lane.destination.zip3 = "752"
  given shipment.weight = 12_000 lb
  expect not applies
}
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
- **What the user asked for.** `shipment.accessorials` lists the accessorials entered on the
  shipment (liftgate, residential, inside delivery), each `{ code: Text, quantity: Number }`, and
  `shipment.requested(CODE)` is whether one of them has that code: `if shipment.requested(LIFT)
  { charge "Liftgate" as LIFT = $85 }`. The code is written bare, like `charge(CODE)` in a test,
  and the checker judges it against the accessorial charge catalog exactly as it judges `charge
  … as CODE` (`unknown-code`, reported once with every use, with the same fixes). The host lists
  only the additional charges a user entered, never the lines a tariff or another program
  generated, so a tariff that bills `LIFT` doesn't see its own line on the next re-rate.
  `requested` is a method of the shipment record (`shipment.requested(…)`, or a binding that
  holds it).
- **Tests of a condition** read its result as `applies`: the final `Bool` expression's value, in
  the condition's `test` blocks only. `expect applies` and `expect not applies` are ordinary
  `expect`s, so `expect applies == (lane.interstate)` works too. A test whose condition refuses
  fails each `expect applies` with the refusal, as a tariff's test fails on `total`; `expect
  fails` expects the refusal. Coverage counts both outcomes (below), so a condition whose tests
  only ever see it apply shows its "doesn't apply" region unreached.

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
| `shipment` | `proNumber: Text`, `status: ShipmentStatus`, `weight: Weight?` (commodity lines, else the shipment's weight), `pieces: Int?`, `distance: Distance`, `duration: Duration`, `linearFeet: Length`, `volume: Volume`, `density: Weight/Volume?`, `hazmat: Bool`, `reefer: Bool`, `temperature: { min: Temperature?, max: Temperature? }`, `freightClass: FreightClass?`, `highestClass: FreightClass?`, `serviceType: { code: Text, description: Text }`, `shipmentType`, `tractorType`, `trailerType` (each `{ code, description }?`), `stops: List<Stop>`, `commodities: List<Commodity>`, `pickup: Stop?`, `delivery: Stop?`, `accessorials: List<{ code: Text, quantity: Number }>` |
| `lane` | `origin` and `destination`, each `{ city, state, zip, zip3, country: Text, zones: List<Text> }`; `interstate: Bool`; `crossBorder: Bool` |
| `customer` | `name: Text`, `code: Text` |
| `fuel` | `price: Money/gal?`, `priceDate: Date?`, `index: Text?` |
| `rating` | `date: DateTime` (in the organisation's time zone), `mode: RatingMode` (`Sell`, `Buy`), `sellTotal: Money?`, `currency: Text`, `holidays: List<{ date: Date, name: Text }>` |
| `tables` | the organisation's active rate matrices, by code |

- A `Stop` has `sequence: Int`, `kind: StopKind`, `status: StopStatus`, `appointment: Bool`,
  `window: { start: DateTime?, end: DateTime? }`, `arrival: DateTime?`, `departure: DateTime?`,
  `dwell: Duration?` (present when both arrival and departure are), `pieces: Int?`,
  `weight: Weight?`, `legDistance: Distance?` and `location: { city, state, zip, zip3, country,
  timezone: Text, category: LocationCategory?, facilityType: FacilityType?, requiresAppointment:
  Bool }`.
  - `legDistance` is the distance from the previous stop, in the organisation's distance unit
    like `shipment.distance`; it is `none` on the first stop, and on a stop whose leg has not
    been measured.
  - `location.category` and `location.facilityType` are what the location is (a `Warehouse`, a
    `Port`) and what kind of facility (`ColdStorage`, `CrossDock`), `none` when the location
    doesn't say. `location.requiresAppointment` is the location's own rule that every visit
    needs an appointment, as opposed to the stop's `appointment`, which says this visit has one.
- A `Commodity` has `name: Text`, `freightClass: FreightClass?`, `weight: Weight`, `pieces: Int`,
  `dimensions: { length: Length?, width: Length?, height: Length? }`, `volume: Volume?`,
  `density: Weight/Volume?`, `hazmat: Bool` and `stackable: Bool`.
- The schema enums (`ShipmentStatus`, `StopKind`, `StopStatus`, `FreightClass`, `RatingMode`,
  `LocationCategory`, `FacilityType`, `Weekday`) are enums of the language; `FreightClass`
  variants are `Class50` through `Class500` (`Class77_5` is class 77.5), and `Weekday`'s are
  `Mon` through `Sun`. These two are ordered by their variants, lowest class and Monday first;
  the others have no order. `LocationCategory` is `Terminal`, `Warehouse`,
  `DistributionCenter`, `TruckStop`, `RestArea`, `CustomerLocation`, `Port`, `RailYard` or
  `MaintenanceFacility`, and `FacilityType` is `CrossDock`, `StorageWarehouse`, `ColdStorage`,
  `HazmatFacility` or `IntermodalFacility`, in the places module's order.
- `shipment.accessorials` is the accessorials a user entered on the shipment, in the order
  entered, each with its catalog `code` and `quantity`; it never holds a line a tariff, a fuel
  program or another rule generated. `shipment.requested(CODE)` tests it by code (see
  "Accessorials and conditions").
- `rating.holidays` is the organisation's holidays from 1 January of the year before the rating
  date's year to 31 December of the year after, recurring holidays expanded to their date in
  each of those years, in date order. It holds holidays only, never blackout dates. `date.holiday`
  reads it (see "Methods on values").
- Volume also has the liquid units `gal` and `l`, for fuel prices.
- `bill` is not part of the record: a tariff's own code reads it as the bill it is building
  (`bill.freight`, `bill.accessorials`, `bill.total`, `bill.charge(…)`; see "Reading the bill
  so far"), and `tariff.catalog` lists it after `tables` with its fields and docs.

### Rate tables

Every published, unarchived rate matrix with one to four dimensions is a table, named by its
code. A matrix declares its **value type** (`Money`, `Money/cwt`, `Money/mi`, `Percent`, `Number`,
…), and each dimension's kind gives its key type: zones, zips, states, countries, freight
classes, equipment and service types are `Text` (or the enum) and match exactly; weight breaks,
distance, pieces and linear feet are quantities in bands; a **postal range** axis (`ZipRange`) is
`Text` matched against ranges.

**Versions.** A matrix is versioned like a formula library. Editors change its draft; submitting
the draft sends it to review, and a reviewer approving it publishes it as the next numbered
version, which is immutable. A tariff reads a table at a version: a formula template version pins
`code → version` for every table its check reads (as it pins library imports), so rating with a
template version reads the pinned versions' cells, and publishing a matrix version never reprices
a contract until a new template version is approved. Editing and testing a draft in the Studio
read each table's newest published version, and the Studio shows which one. An archived matrix is
no longer served to new checks; versions already pinned keep rating.

**Postal ranges.** A `ZipRange` axis's cells hold an inclusive range of postal codes, `{"from",
"to"}`, both ends trimmed and upper-cased and of one length `n`. A key matches a range when it has
at least `n` characters and its first `n` characters fall between the ends, compared character by
character in Unicode code point order (`"10001-1234"` matches `100`–`149`; `"9"` matches no range
of length 3). The axis is keyed, so it takes part in on-demand loading like an exact axis; it
takes no normalisation and no overflow policy (a key no range holds is a miss), and ranges on one
axis never overlap, which the server and `matrix.coverage` refuse.

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
- In tests, `given tables.Zones["A", "B"] = $20/cwt` sets a cell, so a test does not depend on
  the table's live cells (see "Tests").
- `tariff.check` reports `tables: [{ code, version }]`, every table the file and the modules it
  imports read, with the version the context held. The server turns these into the pins a
  template version stores; the context's `TableShape` carries `version` for that.

On the wire a table axis is `{ "match": "Exact" | "Range", "kind", "label", "unit"?,
"normalization"?, "overflow"? }`; a postal range axis is `{ "match": "Range", "kind": "ZipRange" }`.
A cell's `at` holds `{ "key" }` on an exact axis, `{ "min", "max"? }` on a band and `{ "from", "to"
}` on a postal range.

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
| `std.time` | `fn weekend(at: DateTime) -> Bool`; `fn afterHours(at: DateTime, open: Time = @08:00, close: Time = @17:00) -> Bool` (a weekend, before `open` or from `close` on, on `at`'s own clock); `fn businessDuration(start: DateTime, end: DateTime, open: Time = @08:00, close: Time = @17:00) -> Duration` (the weekday time between `open` and `close`) |
| `std.accessorials` | `fn detention(dwell: Duration, free: Duration = 2 h, rate: Money/h, increment: Duration = 15 min) -> Money`; `fn extraStops(stops: List<Stop>, included: Int, fee: Money) -> Money` |
| `std.tiers` | graduated (progressive) pricing, each tier pricing only the part of a distance or weight inside it: `type DistanceTier = { from: Distance, rate: Money/mi }`; `type DistanceStep = { from: Distance, to: Distance, distance: Distance, rate: Money/mi, amount: Money }`; `fn byDistance(tiers: List<DistanceTier>, distance: Distance) -> Money`; `fn distanceSteps(tiers: List<DistanceTier>, distance: Distance) -> List<DistanceStep>`; and for weight `type WeightTier = { from: Weight, rate: Money/cwt }`, `type WeightStep = { from: Weight, to: Weight, weight: Weight, rate: Money/cwt, amount: Money }`, `fn byWeight(tiers: List<WeightTier>, weight: Weight) -> Money`, `fn weightSteps(tiers: List<WeightTier>, weight: Weight) -> List<WeightStep>` |

### Graduated tiers

`std.ltl`'s breaks price the whole weight at one break's rate. A graduated rate prices each part
at its own tier's rate, like income tax brackets: "the first 100 mi at $3/mi, the next 400 at
$2.50/mi, beyond at $2/mi" is three `DistanceTier`s, and 250 mi bills $300 + $375.

```tariff
tariff "Graduated linehaul"

use std.tiers.{ DistanceTier, distanceSteps }

/// Each tier prices the miles from its start up to the next tier's start.
input mileTiers: List<DistanceTier> = [
  { from: 0 mi, rate: $3/mi },
  { from: 100 mi, rate: $2.50/mi },
  { from: 500 mi, rate: $2/mi },
]

for step in distanceSteps(mileTiers, shipment.distance) {
  charge "Linehaul, {step.from} to {step.to}" (quantity: step.distance, rate: step.rate)
}

test "250 miles bills the first 100 at $3 and the next 150 at $2.50" {
  given shipment.distance = 250 mi
  expect total == $675
  expect lines.len == 2
}
```

Each tier the distance reaches is its own line, with its quantity and rate on the invoice;
`charge "Linehaul" = byDistance(mileTiers, shipment.distance)` bills the same total as one line.

- A tier runs from its `from` up to the next tier's `from`; the highest tier has no end. Tiers
  are read in order of `from`, however the list is written, and the lowest tier also prices
  anything below its start (as `std.ltl.deficit` prices a weight below every break): its step
  starts at zero, so a list whose first tier starts at `1 mi` still prices a 0.5 mi load.
- The steps leave out every tier the distance or weight doesn't reach, so each step's
  `distance` (or `weight`) is above zero; a zero or negative quantity has no steps and bills
  `$0`. `amount` is the step's `rate × distance`, exact; the bill rounds it with every other
  line.
- `byDistance` is the steps' total. No tiers at all refuses to rate ("There are no distance
  tiers to price 250 mi with").
- `WeightTier`, `WeightStep`, `byWeight` and `weightSteps` are the same for weight, priced per
  hundredweight. There are two families rather than one because a tier's rate unit is part of
  its type, and Tariff has no user generics.

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
  pickup and delivery, no commodities, no requested accessorials, no holidays) and the inputs'
  defaults; `given input.rate = $2.10/mi` overrides an input. `given shipment.accessorials =
  [{ code: "LIFT", quantity: 1 }]` and `given rating.holidays = [{ date: @2026-12-25, name:
  "Christmas Day" }]` set what a rating reads through `shipment.requested(…)` and `.holiday`.
- In tests only, `pickup(…)`, `delivery(…)`, `splitPickup(…)` and `splitDelivery(…)` build a
  `Stop` and `commodity(…)` a `Commodity`; every argument is named and optional, and a stop's
  `sequence` defaults to its place in the list. A stop builder takes each field of a `Stop`
  (`legDistance: 120 mi`), and each field of its `location` directly: `delivery(zip3: "752",
  category: Warehouse, facilityType: ColdStorage, requiresAppointment: true)` sets those fields
  of the stop's location and leaves the rest blank. Giving `location:` whole and one of its
  fields in the same call is an error (`duplicate-argument`). A built stop's `legDistance` is
  `none` unless given, and its location has no category or facility type and requires no
  appointment. After the `given`s, `shipment.pickup`, `shipment.delivery` and each stop's
  `dwell` are worked out again unless a `given` set them.
- **Table cells.** `given tables.Code[key, …] = value` sets one cell of a rate table for the
  test. The checker holds it to the table's shape: one key per axis, each of the axis's key type
  (one key, not a list, on an exact axis), and a value of the table's value type, or the whole
  cell as `{ value: …, minimum: Money?, deficit: Bool }` for a tariff that reads `.cell(…)` (a
  bare value has no minimum and no deficit).
  - Once a test gives any cell of a table, that table holds only the cells the test gave: a
    lookup of any other is a miss (`none`), never the live cell and never a request for cells
    the host must load. A table the test gives no cell reads the live cells the request carries,
    as a rating does.
  - On a range axis the key is a band floor: `given tables.Breaks[1_000 lb] = $28/cwt` is the
    band from 1,000 lb up to the next floor any `given` of that table writes on that axis, or
    open-ended when none is higher, and every key in the band reads it. So `[0 lb]`, `[500 lb]`
    and `[1_000 lb]` give the bands 0–500 lb, 500–1,000 lb and 1,000 lb and up. Exact keys pass
    through the axis's normalisation, and the axis's overflow policy still applies to a key no
    given band holds.
  - The same cell given twice keeps the later value. Keys and values are worked out like any
    `given`'s value, before the rating.
  - A test (or property) of a tariff or condition is warned (`live-table`, at its name) for
    each table the file reads, in its body or in the test itself, that the test gives no cell
    of: the test passes or fails with the matrix's live data, so editing a rate matrix can break
    it. Giving the cells the test reads silences it.
- `expect` takes a `Bool`; `charge("Label")` or `charge(CODE)` reads a line's amount (`Money?`),
  `total` the bill's total, `lines` the whole list (`{ label: Text, code: Text?, amount: Money,
  kind: LineKind }`, where `LineKind` is `Charge`, `Minimum`, `Maximum` or `Rounding`), and
  `reviews` the review reasons (`List<Text>`, in order: `expect reviews.isEmpty`,
  `expect reviews == ["Hazmat over 500 miles: check the permit cost"]`); `expect fails`
  (optionally with the message) expects a refusal. These read the finished bill; the tariff's
  own code reads the bill so far through `bill`. In a condition's tests, `applies` is the
  condition's result: `expect applies`, `expect not applies`.
- Tests replace the Studio's test-case rows and recorded scenarios: "Save as test" in the preview
  writes a `test` block from a real shipment's facts into the source.

### Properties

A test checks one shipment. A **property** checks a rule over a whole range of shipments, which is
where pricing bugs hide: the weight-break inversion where 999 lb bills more than 1,000 lb, the
short load that bills below the minimum, the discount that goes negative.

```tariff
property "heavier never bills less" {
  vary shipment.weight in 1 lb..40_000 lb
  expect total nondecreasing in shipment.weight
}

property "never below the minimum" {
  vary shipment.distance in 1 mi..3_000 mi
  vary input.rate
  given fuel.price = $3.20/gal
  expect total >= $450
}
```

- **`vary target in values`** makes `target` take many values. The target is a record path or
  `input.name`, exactly as `given` takes, or a new `camelCase` name, bound for the rest of the
  property (the only form a library may use, to test its functions). The values are:
  - a range of an ordered type with both ends written: `1 lb..40_000 lb`, `$1/mi..=$5/mi`,
    `@2026-01-01..@2027-01-01`, `@06:00..=@20:00`, `Class50..=Class100`;
  - a list of values: `[Standard, Expedited]`, `[[pickup(), delivery()], [pickup(), delivery(),
    delivery()]]`.

  `in values` may be left out for a `Bool` (both values), an enum (every variant), an optional
  enum or `Bool` (also `none`), and an input with a range or list constraint (the constraint).
- **`given`** sets a field, an input or a rate table's cell for every case, as in a test (a
  table one `given` sets a cell of holds only the given cells in every case). **`assume cond`** skips the
  cases where `cond` is false; it reads the record, inputs, `vary` names, functions and constants,
  and runs before the rating, so it cannot read the bill.
- **`expect cond`** must hold in every case. **`expect value nondecreasing in target`** (or
  `nonincreasing`) must hold as the varied `target` grows, with every other varied value fixed:
  `value` is any ordered expression the case can compute (`total`, `charge(FSC) ?? $0`,
  `lines.len`), and `target` must be varied over a range in this property. `expect fails` is not
  allowed: a property states what holds when the tariff rates; narrow it with `assume` or the
  ranges instead.
- **Cases.** The engine picks the values deterministically: the same file always runs the same
  cases.
  - For a range, it takes both ends; every *landmark* inside the range (a literal, constant,
    input default or set value of the same dimension anywhere in the file or its libraries, and
    every band edge of a range-keyed rate table axis of that dimension), each with its neighbours
    one *step* either side; and evenly spread points, until the axis has 24 values. The step is 1
    in the last decimal place the range's ends are written with (`1 lb..40_000 lb` steps by
    1 lb, `$0.50/mi..$15/mi` by $0.01/mi), a day for dates, a minute for date-times and
    times of day, and one variant for an ordered enum (`vary shipment.freightClass in
    Class50..=Class100`). Text has no step, so a text range can't be varied over: list the
    values instead.
  - For a list, every item.
  - With one `vary`, every value is a case. With several, every combination when there are at
    most 256, otherwise each axis's landmarks against the other axes' ends, filled up to 256 with
    combinations drawn from a generator seeded by the property's name.
- **Failures shrink.** When an `expect` fails, the engine looks for the simplest failing case: for
  each axis in turn it searches, by halving, for the smallest value from the range's start that
  still fails, keeping the others fixed (an item list keeps its first failing item). For a
  `nondecreasing` failure it reports the two neighbouring values whose results are out of order,
  narrowed by halving the gap between them ("999 lb bills $310, 1000 lb bills $250"). A case
  that refuses to rate fails the property with the refusal's message.
- **Results.** A property passes, fails with its counterexample (each varied value, the failed
  expectation's values, the case's bill) or ends in error, like a test. The Studio shows the
  counterexample and can save it as an ordinary `test`, so the bug stays fixed.
- **Rules.** Properties live at file level in tariffs and libraries. A property needs at least one
  `vary`; its ranges must be finite, with compile-time ends, and an input's range must lie inside
  the input's constraint. Its `vary`s, `given`s and `assume`s come first, before its other
  statements; a `given` may not read a varied name, and a path is varied at most once and never
  also given; a rate table can't vary (`vary-target`), only take given cells. Its `expect`s
  follow the test rules (they read the bill, the record and the file's functions and
  constants), so a saved counterexample always checks. At most 256 cases run per property,
  under the file's budget.

Properties are tests: the Studio runs them on every change, and a version with a failing property
cannot be saved for review or submitted.

### Coverage

Every test and property case records which parts of the file it ran. `tariff.test` reports the
file's **regions** with how many times each ran: each block of an `if`, `else if` and `else`
(statement or expression), each `match` arm, each `for` body, each `option`, each `charge`, each
`review`, each refusal (`fail`, the refusing side of `require` and of `?? fail`), each function
body, and a condition's two outcomes: two regions on its final expression, `applies` and
`notApplies`, entered when a test or case sees the condition give `true` or `false` (a refusal
enters neither). The Studio marks the lines of regions no test reached, shows "Tests reach 18 of
21 regions" in the tests panel, and lists unreached regions as a readiness warning before submit.
Coverage counts the root file's regions only; a library has its own.

## Module operations

The crate's `formula.*` operations are replaced by `tariff.*`, with the same envelope, budgets,
`Message`s and UTF-16 ranges as every other operation:

| Operation | Does |
|---|---|
| `tariff.check` | parses and checks a file with the libraries it imports, against the schema, the tables' shapes and the organisation's accessorial codes; returns diagnostics with fixes, the declared inputs (name, type, default, doc), the outline, symbols and inferred types (for inlay hints) and the imports it pinned |
| `tariff.evaluate` | rates a batch of programs (a tariff or condition with its libraries, input values and record) and returns, per program, the bill (charges with label, code, amount, quantity, rate, the limit that applied and source range; notes; review reasons; adjustments with their scope; total) or the refusal, plus lookups and per-statement values for the trace |
| `tariff.test` | runs a file's tests and properties against the sample record; returns each one's result, the failing expectation and the values involved, a property's counterexample, and the file's coverage |
| `tariff.format` | formats a file (one canonical style) |
| `tariff.tokens` | the token stream for highlighting |
| `tariff.complete`, `tariff.hover`, `tariff.signature`, `tariff.definition`, `tariff.references`, `tariff.rename`, `tariff.inlays` | the editor's language features at a cursor or for a range |
| `tariff.actions` | the refactorings available for a selection, each a set of edits: today "Promote to input" |
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
| `Time` | `{"$time": "17:30:00"}` (`HH:MM` or `HH:MM:SS` is read; seconds are always written) |
| Enum variants | a JSON string (`"Delivery"`), read as the declared enum |
| `none` | JSON `null` |
| Lists, records | JSON arrays and objects (record keys never start with `$`) |

Input values on rules and agreements, the record the host sends, the bill and the trace all use
these forms, so a value never loses its unit or its precision between Postgres, Go, the module
and the browser. The record is decoded strictly against "Data": no field unknown, and every
field present except an optional one, which may be left out as `null`. For the newest fields:

```json
{"shipment": {"accessorials": [{"code": "LIFT", "quantity": {"$n": "1"}}],
              "stops": [{"legDistance": null,
                         "location": {"category": "Warehouse", "facilityType": null,
                                      "requiresAppointment": true, "…": "…"}, "…": "…"},
                        {"legDistance": {"$q": "412", "u": "mi"}, "…": "…"}], "…": "…"},
 "rating": {"holidays": [{"date": {"$date": "2026-12-25"}, "name": "Christmas Day"}], "…": "…"}}
```

`legDistance` is in the organisation's distance unit, like `shipment.distance`; an enum is its
variant's name, and `null` is none.

## Storage and server changes

- **Formula templates** keep their name, versions, review and scheduling, and store one `source`
  instead of `expression`, `variableDefinitions`, `breakdownDefinitions`, the policy fields,
  `type` and `languageVersion`. A version also stores the inputs and pinned imports `tariff.check`
  derived, for contract forms and queries. The test-case table and the scenario columns are
  dropped.
- **Formula libraries** are a new organisation-level record with the same versioning and review as
  templates (`formula_libraries`, `formula_library_versions`), with GraphQL reads, REST writes,
  permissions, audit, agent tools and an editor.
- **Rate matrices** are versioned with the library review flow and statuses: the draft is version
  `0` of `rate_matrix_dimensions` and `rate_matrix_cells`, each published version a
  `rate_matrix_versions` row with its own dimension and cell rows keyed by `(matrix,
  version_number)`, copied from the draft in one `INSERT … SELECT` on approval. Reviews are
  `rate_matrix_reviews`; `rate_matrices` carries the review state, `published_version_number` and
  `has_draft_changes`. Templates and template versions store `tables` (the pins), and
  `formula_template_version_tables` indexes them for "who pins this matrix". The server's table
  cache keys published cells by `(tenant, matrix, version)` with no invalidation, and serves the
  newest published versions under a stamp that moves when one is published or archived; a rating
  batch reads its cells by `(matrix, version)` in bounded, batched queries. Postal range cells
  keep the range's end in `dN_key_to` beside its start in `dN_key`. GraphQL reads
  `rateMatrixVersions`, `rateMatrixVersion`, `rateMatrixReviews`, `rateMatrixUsages` and
  `FormulaTemplate.tables`; REST writes `submit`, `approve`, `reject`, `request-changes`,
  `archive` and `restore` under `/rate-matrices/:id/`.
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
- the tests panel, from the `test` and `property` blocks, run in the browser on every change, with
  a property's counterexample and a "Save as test" that writes it as a `test`;
- coverage in the gutter: the lines of regions no test reached, with the count in the tests panel;
- references: every place a name is declared, read or written, highlighted at the cursor and
  listed on request;
- refactorings from `tariff.actions`: **Promote to input** turns a selected literal (`$2.35/mi`)
  into an `input` declaration with that default and puts the new name in its place, ready to
  rename, optionally replacing every equal literal in the file;
- the bill preview: each charge with its code, quantity, rate and any limit that applied, the
  notes, the review reasons and the adjustments.

Every editor operation answers from one call on a file that may not parse, in UTF-16 positions.
A `doc` on the wire is a string (the author's words) or a `Message` (a built-in's or the standard
library's, translated by the host). Signature help gives a parameter the sentences of its
function's doc that name it, and none when they would only repeat the function's doc. Rename
refuses a name imported from a library (rename it in the library), keywords, schema roots and
tests, and refuses a new name that would collide with one in scope.

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
9. **Properties are tests.** Deterministic, landmark-driven cases with shrinking, so a property
   fails the same way in the browser, on the server and in review.
10. **"Lower of" is a construct.** `best` and `worst` choose between whole methods and record the
    losers, instead of each contract hand-coding comparisons.
11. **Inputs carry their bounds.** A contract value outside them is refused when saved, and the
    form shows them.

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
item        = { doc } ( use | input | const | typeDecl | enumDecl | fnDecl | test | property | stmt ) ;

use         = "use" modulePath [ "." "{" name { "," name } [ "," ] "}" ] [ "as" name ] NL ;
modulePath  = ( "std" | "org" ) "." name { "." name } ;

input       = "input" name ":" type [ "=" expr ] [ "where" expr ] NL ;
                                                              (* the constraint: a range or a list *)
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
property    = "property" string block ;
block       = "{" { stmt } "}" ;

stmt        = letStmt | varStmt | assign | charge | note | review | ifStmt | matchStmt
            | forStmt | choose NL | require | fail | return | "break" NL | "continue" NL
            | total | given | vary | assume | expect | expr NL ;
letStmt     = "let" name [ ":" type ] "=" expr NL ;
varStmt     = "var" name [ ":" type ] "=" expr NL ;
assign      = name ( "=" | "+=" | "-=" ) expr NL ;
charge      = "charge" string [ "as" CODE ] [ "(" option { "," option } ")" ] [ "=" expr ] NL ;
                                                              (* options: quantity:, rate:, minimum:,
                                                                 maximum:; "= expr" may be left out
                                                                 only with quantity: and rate: *)
option      = name ":" expr ;
note        = "note" string NL ;
review      = "review" string NL ;                            (* tariffs only *)
choose      = ( "best" | "worst" ) string "{" { option } "}" ;
option      = "option" string block ;
ifStmt      = "if" condition block { "else" "if" condition block } [ "else" block ] ;
condition   = expr | "let" name "=" expr ;
matchStmt   = "match" expr "{" { arm } "}" ;
arm         = pattern [ "if" expr ] "=>" ( block | expr ) NL ;
pattern     = alt { "|" alt } ;
alt         = literal | rangePat | TypeName | "none" | "_" | name | recordPat ;
rangePat    = [ rangeEnd ] ( ".." | "..=" ) [ rangeEnd ] ;
rangeEnd    = literal | TypeName [ "." TypeName ] ;             (* a variant of an ordered enum *)
recordPat   = "{" name ":" pattern { "," name ":" pattern } [ "," ] "}" ;
forStmt     = "for" name "in" expr [ "where" expr ] block ;
require     = "require" condition "else" string NL ;
fail        = "fail" string NL ;
return      = "return" expr NL ;
total       = ( "minimum" | "maximum" ) expr [ "on" "freight" ] NL
            | "round" [ "half" "even" | "up" | "down" ] "to" ( "cents" | expr ) NL ;
given       = "given" lvalue [ "[" expr { "," expr } "]" ] "=" expr NL ;
                                                              (* tests and properties; the keys
                                                                 only after tables.Code, one per
                                                                 axis: a cell *)
vary        = "vary" ( lvalue | name ) [ "in" expr ] NL ;     (* properties only *)
assume      = "assume" expr NL ;                              (* properties only *)
expect      = "expect" ( expr [ ( "nondecreasing" | "nonincreasing" ) "in" lvalue ]
                       | "fails" [ string ] ) NL ;            (* tests and properties; the
                                                                 monotone form and no "fails"
                                                                 in properties *)
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
primary     = literal | name | TypeName [ "." TypeName ] | TypeName "." name args
            | "(" expr ")" | list | record | ifExpr | matchExpr | choose ;
                                                              (* StopKind.Delivery, Int.parse(t);
                                                                 Class77_5.number is a postfix
                                                                 field of the variant; a choose
                                                                 only as the whole value of a
                                                                 let, var or assignment *)
ifExpr      = "if" condition "{" expr "}" { "else" "if" condition "{" expr "}" } "else" "{" expr "}" ;
matchExpr   = "match" expr "{" { pattern [ "if" expr ] "=>" expr NL } "}" ;
list        = "[" [ expr { "," expr } [ "," ] ] "]" ;
record      = "{" name ":" expr { "," name ":" expr } [ "," ] "}" ;

literal     = number [ unit ] | money | percent | string | date | dateTime | time
            | "true" | "false" | "none" ;
money       = "$" number [ "/" unit ] ;                       (* $450, $2.35/mi, $18.50/cwt *)
percent     = number "%" ;
number      = digit { digit | "_" } [ "." digit { digit | "_" } ] ;
unit        = "lb" | "cwt" | "kg" | "t" | "ton" | "mi" | "km" | "in" | "ft" | "m"
            | "ft3" | "m3" | "gal" | "l" | "min" | "h" | "d" | "°F" | "°C" | "pallet" | "piece" | "stop"
            | unit "/" unit ;                                 (* lb/ft3 for density *)
date        = "@" digit4 "-" digit2 "-" digit2 ;
dateTime    = date "T" digit2 ":" digit2 [ ":" digit2 ] ( "Z" | ( "+" | "-" ) digit2 ":" digit2 ) ;
time        = "@" digit2 ":" digit2 [ ":" digit2 ] ;             (* @17:30, @07:30:15 *)
string      = '"' { char | escape | "{" expr "}" | "{{" | "}}" } '"' ;
```

A keyword is never an identifier. `best`, `worst`, `option` and `review` start their construct
only when a string follows on the same line, and `on` and `freight` are keywords only after a
total clause's amount; elsewhere all of them are ordinary names (`bill` is a name too, one a
tariff may not declare). Unit names are contextual: `lb` is a unit only directly after
a number, or after `/` in a money literal or a type. Elsewhere it is an ordinary name, except
as the argument of `.in(…)`, which reads `cwt` or `lb/ft3` as a unit. `$` directly followed
by `/` and a unit is the money unit, so a whole rate unit is written `.in($/cwt)`. `in` directly
after a number is inches unless what follows starts a value: `48 in` is a length, and in
`5 in [5, 6]` it is the membership keyword.
