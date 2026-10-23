# Tariff cheat sheet

A one-page quick reference for writing formula templates, formula libraries and accessorial
conditions in Tariff. The full reference, with every rule and the grammar, is
[formula-language.md](formula-language.md). Every `tariff` block on this page is a complete file
that checks clean and whose tests pass: `tests/tariff_cheat_sheet.rs` in `services/crates/rules`
runs them.

## A complete tariff

```tariff
tariff "Per mile with fuel"

/// Truckload linehaul billed per loaded mile.

use std.fuel.{ stepPercent }

/// Paid per loaded mile.
input rate: Money/mi = $2.35/mi
/// The diesel price fuel is pegged to.
input fuelPeg: Money/gal = $1.20/gal
input minimumCharge: Money = $450

let miles = shipment.distance
let linehaul = rate * miles

charge "Linehaul" (quantity: miles, rate: rate)

let price = fuel.price ?? fail "No fuel price on {rating.date}"
charge "Fuel surcharge" as FSC = linehaul * stepPercent(price, peg: fuelPeg, step: $0.05/gal, perStep: 0.5%)

minimum minimumCharge
round to cents

test "a short load bills the minimum" {
  given shipment.distance = 50 mi
  given fuel.price = $1.20/gal
  expect total == $450
}

test "fuel steps half a percent per five cents over the peg" {
  given shipment.distance = 500 mi
  given fuel.price = $1.30/gal
  expect charge("Linehaul") == $1_175
  expect charge(FSC) == $11.75
}

test "no fuel price refuses to rate" {
  given fuel.price = none
  expect fails
}
```

The order is always: header, doc, `use`s, `input`s, then bindings and charges top to bottom,
total clauses (`minimum`, `maximum`, `round`) last, and tests at the end.

## Files

| First line | What it is | Holds | Gives |
|---|---|---|---|
| `tariff "Name"` | a formula template | inputs, bindings, charges, total clauses, tests | an itemised bill |
| `library "name"` | a formula library | `fn`, `const`, `type`, `enum`, tests | functions other files import |
| `condition "Name"` | an accessorial's apply-when rule | inputs, bindings, functions, one final `Bool` | true or false |

## Writing values

| You write | It is |
|---|---|
| `1_200`, `2.35` | `Int`, `Number` (exact decimals, `_` groups digits) |
| `500 lb`, `2 cwt`, `2.5 mi`, `4 h`, `30 min`, `48 ft`, `48 in` | quantities with a unit |
| `$450` | money, in the rating's currency |
| `$2.35/mi`, `$18.50/cwt`, `$65/h` | rates: money per unit |
| `18%` | a percent (multiplies as 0.18) |
| `@2026-10-22`, `@2026-10-22T08:00-05:00` | a date, a date-time |
| `"Stop {stop.sequence}"` | text, with `{…}` interpolation (`{{` for a brace) |
| `true`, `false`, `none` | booleans, and the empty value of an optional |
| `[a, b]`, `{ from: 0 lb, rate: $40/cwt }` | a list, a record |
| `// note`, `/// doc` | a comment; a doc comment shown in hover and the inputs form |

Units, by dimension:

| Dimension | Units |
|---|---|
| Weight | `lb`, `cwt`, `kg`, `t` (metric tonne), `ton` (short ton) |
| Distance | `mi`, `km` |
| Length | `in`, `ft`, `m` |
| Volume | `ft3`, `m3`, `gal`, `l` |
| Duration | `min`, `h`, `d` |
| Temperature | `°F`, `°C` (compare and subtract only) |
| Count | `pallet`, `piece`, `stop` |

## Units and money do the arithmetic

| Expression | Result |
|---|---|
| `500 lb + 2 cwt` | `700 lb`; `+`, `-` and comparisons need one dimension |
| `$18.50/cwt * 1_200 lb` | `$222`, converting units exactly |
| `$2.35/mi * 812 mi` | `$1_908.20` |
| `$1_000 * 18%`, `$450 / 3` | money |
| `0.5% * 3` | `1.5%`; a percent times a number stays a percent |
| `$450 / 10 mi` | a rate, `Money/mi` |
| `(1_240 lb).in(cwt)` | `12.4 cwt`; `.value` gives the bare number, `12.4` |
| `$0.185/lb.in(cwt)` | `$18.50/cwt` |
| `1 cwt == 100 lb`, `$426 == $426.00` | `true`; equality compares by value |

The checker refuses money added to a weight, a rate per mile times a weight, and a rate per
hundredweight times a distance, and says what each side is.

## Names and types

- `let x = …` is a value. `var total = $0` then `total += …` accumulates inside a loop.
- `input rate: Money/mi = $2.35/mi` is a value the contract sets; the default is used when the
  rate agreement rule and agreement don't set it. An input with no default that nothing sets
  refuses to rate.
- `input rate: Money/mi = $2.35/mi where $0.50/mi..=$15/mi` bounds what a contract may set: a
  range (`lo..hi` excludes `hi`, `lo..=hi` includes it, either end may be left off) or a list
  (`where [Standard, Guaranteed]`). The default must satisfy it, the rule editor shows the bounds,
  and a value outside them is refused when the rule or agreement is saved.
- `const BASE: Money = $3.50` is a compile-time constant (written from literals only).
- Names are camelCase (`extraDropFee`), types and enum variants PascalCase (`Money`, `Delivery`),
  charge codes UPPERCASE (`FSC`).
- Annotate inputs, function parameters and return types, record fields and library constants.
  Everything else is inferred, and the editor shows the type as a hint.
- Types: `Bool`, `Int`, `Number`, `Percent`, `Money`, `Text`, `Date`, `DateTime`, the
  quantities `Weight`, `Distance`, `Length`, `Volume`, `Duration`, `Temperature`, `Count`, rates
  such as `Money/mi`, `List<T>`, records, enums, and `T?` for a value that may be empty.

## Empty values

A field that can be empty, such as `shipment.weight`, is `Weight?` and must be handled before
use:

| Write | When |
|---|---|
| `shipment.weight ?? 0 lb` | a fallback |
| `shipment.weight ?? fail "This rate needs a weight"` | refuse to rate without it |
| `require let w = shipment.weight else "This rate needs a weight"` | refuse, and use `w` from there on |
| `if let w = shipment.weight { … }` | only when it is there |
| `shipment.trailerType?.code` | reach into it; the result is optional too |

`??` binds loosest of all operators: `a ?? b * c` means `a ?? (b * c)`. Wrap the fallback,
`(a ?? b) * c`, when you mean the product.

## Decisions, loops and lists

```tariff
tariff "Optionals and flow"

require let weight = shipment.weight else "This rate needs the shipment's weight"

let perCwt = match weight {
  ..500 lb           => $42.00/cwt
  500 lb..2_000 lb   => $35.50/cwt
  2_000 lb..5_000 lb => $28.10/cwt
  _                  => $22.75/cwt
}

charge "Linehaul" = perCwt * weight.roundUp(to: 100 lb)

let deliveries = shipment.stops.count(s => s.kind == Delivery)
let extra = if deliveries > 1 { deliveries - 1 } else { 0 }
if extra > 0 {
  charge "Extra drops" as XDROP (quantity: extra, rate: $75)
}

for stop in shipment.stops where stop.dwell != none {
  let dwell = stop.dwell ?? 0 h
  if dwell > 2 h {
    note "Stop {stop.sequence} waited {dwell}"
  }
}

if let temp = shipment.temperature.min {
  if temp < 32 °F {
    note "Frozen load"
  }
}

if shipment.hazmat and not shipment.reefer {
  charge "Hazmat" as HAZ = $150
}

test "a 1,240 lb load bills at the next hundred pounds" {
  given shipment.weight = 1_240 lb
  expect charge("Linehaul") == $461.50
}

test "two deliveries bill one extra drop" {
  given shipment.weight = 600 lb
  given shipment.stops = [pickup(), delivery(), delivery()]
  expect charge(XDROP) == $75
}

test "no weight refuses" {
  given shipment.weight = none
  expect fails "This rate needs the shipment's weight"
}
```

- `if … { a } else { b }` is also an expression; there is no `? :`.
- `match` must cover every value, and its ranges may not overlap. `..500 lb` is below 500,
  `500 lb..2_000 lb` is from 500 up to (not including) 2,000, `..=` includes the end, and `_`
  is everything else. Arms also match enum variants (`Delivery`, `Pickup | Delivery`), `none`, and
  guards (`n if n > 3`).
- `for x in list where cond { … }` walks a list in order; `break` and `continue` work. There is no
  `while` and no recursion, so every rating ends.
- `x in [Sat, Sun]` and `w in 500 lb..2_000 lb` test membership.
- Logic is `and`, `or`, `not`. Comparisons can't be chained: write `1 < n and n < 3`.

List methods, with `x => …` lambdas:

| Method | Gives |
|---|---|
| `where(s => …)`, `map(s => …)` | a list |
| `count(s => …)`, `len`, `isEmpty` | how many, whether empty |
| `sum(c => c.pieces)` | a total; `$0`, `0 h` or `0` for an empty list |
| `min(…)`, `max(…)` | the smallest or largest value, optional (`none` when empty) |
| `avg(…)` | the average; an empty list stops the rating, so test `isEmpty` first |
| `any(…)`, `all(…)` | true or false |
| `first()`, `last()`, `find(s => …)`, `xs[0]` | one item, optional |
| `sortBy(s => …)`, `take(n)`, `skip(n)`, `reverse()`, `distinct()` | a list |
| `groupBy(s => …)` | `{ key, items }` for each group |

## Functions

```tariff
tariff "Lists and functions"

/// Deadhead over the first 50 empty miles is paid at the loaded rate.
fn deadhead(empty: Distance, rate: Money/mi) -> Money {
  let paid = (empty - 50 mi).max(0 mi)
  return rate * paid
}

fn residential(fee: Money = $95) = fee

input rate: Money/mi = $2/mi
input emptyMiles: Distance = 80 mi

let pieces = shipment.commodities.sum(c => c.pieces)
let heaviest = shipment.commodities.max(c => c.weight) ?? 0 lb
let names = shipment.commodities.map(c => c.name)
let hasHazmat = shipment.commodities.any(c => c.hazmat)

charge "Linehaul" (quantity: shipment.distance, rate: rate)
charge "Deadhead" = deadhead(empty: emptyMiles, rate: rate)

if pieces > 10 or heaviest > 5_000 lb or hasHazmat {
  charge "Liftgate" as LIFT = residential()
}

if not names.isEmpty {
  note "Commodities: {names}"
}

test "deadhead pays empty miles past 50" {
  given shipment.distance = 100 mi
  expect charge("Deadhead") == $60
  expect deadhead(empty: 40 mi, rate: $2/mi) == $0
}

test "many pieces add a liftgate" {
  given shipment.distance = 100 mi
  given shipment.commodities = [commodity(pieces: 6), commodity(pieces: 6)]
  expect charge(LIFT) == $95
  expect lines.len == 3
}
```

- Parameters and the return type are annotated; a one-expression body may use `=`.
- Arguments can be named (`deadhead(empty: 80 mi, rate: rate)`), and parameters can have
  defaults written from literals and constants.
- A function sees only its parameters, constants, types and other functions: not `shipment`,
  the inputs or the file's `let`s. Pass in what it needs. It can't bill a charge.

Methods on values:

| On | Methods |
|---|---|
| `Money` | `round()`, `roundHalfEven()`, `roundUp(to: $5)`, `roundDown(to: $1)`, `abs()`, `min(x)`, `max(x)`, `clamp(lo, hi)` |
| Quantities | `in(cwt)`, `value`, `round(to: q)`, `roundUp(to: 100 lb)`, `roundDown(to: q)`, `abs()`, `min(x)`, `max(x)`, `clamp(lo, hi)` |
| `Number` | `round(places: 2)`, `roundHalfEven()`, `floor()`, `ceil()`, `abs()`, `min(x)`, `max(x)`, `clamp(lo, hi)`, `pow(n)`, `sqrt()` |
| `Percent` | `value` (`18%.value` is `18`) |
| `Text` | `len`, `isEmpty`, `upper()`, `lower()`, `trim()`, `startsWith(t)`, `endsWith(t)`, `contains(t)`, `replace(a, b)`, `split(sep)`, `matches(pattern)` |
| `Date`, `DateTime` | `year`, `month`, `day`, `weekday` (`Mon` … `Sun`), `weekend`, `addDays(n)`; on a `DateTime` also `hour`, `minute`, `date`, `addHours(n)` |

## Charges and the bill

| Statement | Bills |
|---|---|
| `charge "Linehaul" (quantity: miles, rate: rate)` | a freight line, amount `rate × quantity`, with both shown on the invoice |
| `charge "Base" = $100` | a freight line with an amount |
| `charge "Fuel surcharge" as FSC = …` | an accessorial line, under the catalog code `FSC` |
| `note "Text {x}"` | a note on the trace, receipt and invoice; no amount |
| `minimum $450` | tops the total up with a "Minimum charge adjustment" line |
| `maximum $5_000` | brings the total down with a "Maximum charge adjustment" line |
| `round to cents` | the default; also `round half even to cents`, `round up to $5`, `round down to $1` |

- **Uncoded charges are freight**: each is its own freight line on the invoice. A freight line
  may be negative (a discount) or zero; a zero line is dropped from the bill.
- **Coded charges are accessorials**: `as CODE` must name an active code in your accessorial
  charges, and the amount must be positive. When the tariff bills a code, nothing else (a
  contract accessorial, the fuel surcharge program) bills it again on that shipment.
- Charges can sit inside `if`, `match` arms and `for` loops, never inside a function.
- Total clauses come last, in the order `minimum`, `maximum`, `round`, and apply to the whole
  bill.

## Choosing the cheaper method

```tariff
tariff "Lower of per mile or flat"

/// Paid per loaded mile.
input rate: Money/mi = $2.35/mi where $0.50/mi..=$15/mi
input flatRate: Money = $900
input volumeRate: Money/lb = $0.12/lb

best "Linehaul" {
  option "Per mile" {
    charge "Linehaul" (quantity: shipment.distance, rate: rate)
  }
  option "Flat" {
    charge "Flat rate" = flatRate
  }
  option "Volume" {
    let weight = shipment.weight ?? 0 lb
    require weight >= 6_000 lb else "Volume rates start at 6,000 lb"
    charge "Volume" = volumeRate * weight
  }
}

test "a long run takes the flat rate" {
  given shipment.distance = 500 mi
  expect chosen("Linehaul") == "Flat"
  expect total == $900
}

test "a short run bills per mile" {
  given shipment.distance = 100 mi
  expect chosen("Linehaul") == "Per mile"
  expect charge("Linehaul") == $235
}
```

- `best` keeps the option whose charges add up to the least, `worst` the most; a tie goes to the
  option written first. The kept option's lines join the bill; the others show on the trace.
- An option that refuses (`require`, `fail`, `?? fail`) is out of the running; when every option
  refuses, the rating refuses with the first one's reason.
- Each option is its own scope, and may not assign a `var` from outside the choice.
- `chosen("Linehaul")` in a test is the kept option's label, or `none`.

## Refusing to rate

A rating that can't be priced says why; it never bills a silent `$0`.

- `fail "No rate for {lane.destination.zip3}"`
- `require shipment.distance > 0 mi else "This rate needs a distance"`
- `require let w = shipment.weight else "This rate needs a weight"`
- `value ?? fail "…"`

The message is shown on the shipment, in the Studio and to Nova.

## What you can read

| Root | Highlights |
|---|---|
| `shipment` | `weight: Weight?`, `pieces: Int?`, `distance`, `duration`, `linearFeet`, `volume`, `density?`, `hazmat`, `reefer`, `temperature.min?`, `temperature.max?`, `freightClass?`, `highestClass?`, `serviceType.code`, `trailerType?.code`, `stops`, `commodities`, `pickup?`, `delivery?` |
| a stop | `sequence`, `kind` (`Pickup`, `Delivery`, …), `status`, `appointment`, `window.start?`, `window.end?`, `arrival?`, `departure?`, `dwell?`, `pieces?`, `weight?`, `location.zip3` |
| a commodity | `name`, `freightClass?`, `weight`, `pieces`, `dimensions`, `volume?`, `density?`, `hazmat`, `stackable` |
| `lane` | `origin` and `destination` (`city`, `state`, `zip`, `zip3`, `country`, `zones`), `interstate`, `crossBorder` |
| `customer` | `name`, `code` |
| `fuel` | `price: Money/gal?`, `priceDate?`, `index?` |
| `rating` | `date`, `mode` (`Sell` or `Buy`), `sellTotal?`, `currency` |
| `tables` | your organisation's active rate matrices, by code |

Hover any field in the Studio for its type and unit; the **Reference** tab lists them all.

## Rate tables

```tariff
tariff "Zone rates"

let weight = shipment.weight ?? fail "This rate needs a weight"
let perCwt = tables.Zones[lane.origin.zones, lane.destination.zones] ?? fail "No zone rate for this lane"

charge "Linehaul" = perCwt * weight

if let band = tables.Breaks.band(weight) {
  note "Weight break from {band.min}"
}

test "a zone pair bills its rate" {
  given shipment.weight = 1_000 lb
  given lane.origin.zones = ["A"]
  given lane.destination.zones = ["B"]
  expect total == $200
}

test "a lane with no zone rate refuses" {
  given shipment.weight = 1_000 lb
  given lane.origin.zones = ["Z"]
  expect fails "No zone rate for this lane"
}
```

- `tables.CODE[key, …]` reads a rate matrix by its code, with one key per dimension. It gives
  the matrix's value type, optional: a missing cell is `none`.
- An exact dimension (zone, zip, state, class) takes a key or a list of keys tried in order; a
  banded one (weight, distance, pieces) takes a quantity.
- `tables.CODE.cell(…)` adds the cell's minimum and deficit flag; `tables.CODE.band(x)` gives
  the matching band (`min`, `max`, `value`); `.interpolate(x)` interpolates between bands.
- The checker knows each matrix's dimensions, so the wrong number of keys, or a weight for a
  distance dimension, is an error.

## Standard library

Import with `use std.ltl.{ deficit, Break }`, or the whole module with `use std.ltl`.

| Module | Gives |
|---|---|
| `std.ltl` | `Break`, `Rated`, `deficit(breaks, actual:, allow: true)`, `billed(weight, to: 100 lb)`, `densityClass(density)` |
| `std.fuel` | `stepPercent(price, peg:, step:, perStep:)`, `perMile(price, peg:, mpg:)` |
| `std.time` | `weekend(at)`, `afterHours(at, open: 8, close: 17)`, `businessDuration(start, end, open: 8, close: 17)` |
| `std.accessorials` | `detention(dwell, free: 2 h, rate:, increment: 15 min)`, `extraStops(stops, included:, fee:)` |

Weight breaks are an input the rule edits as a table:

```tariff
tariff "LTL weight breaks"

use std.ltl.{ deficit, Break }

/// The contract's weight breaks.
input breaks: List<Break> = [
  { from: 0 lb, rate: $40/cwt, minimum: $95 },
  { from: 500 lb, rate: $32/cwt, minimum: none },
  { from: 1_000 lb, rate: $25/cwt, minimum: none },
]

let weight = shipment.weight ?? fail "This rate needs a weight"
let rated = deficit(breaks, actual: weight)

charge "Linehaul" (quantity: rated.weight, rate: rated.rate)

minimum rated.minimum ?? $0
round up to $1

test "a heavy load rates at the deficit weight when cheaper" {
  given shipment.weight = 900 lb
  expect charge("Linehaul") == $250
}
```

## Tests

| Write | Does |
|---|---|
| `test "name" { … }` | a test, at the end of the file |
| `given shipment.weight = 1_240 lb` | sets a field of the sample shipment |
| `given input.rate = $2.10/mi` | sets an input |
| `given shipment.stops = [pickup(), delivery(appointment: true)]` | builds stops; also `splitPickup()`, `splitDelivery()` |
| `given shipment.commodities = [commodity(pieces: 6, weight: 500 lb)]` | builds commodities |
| `expect total == $511.20` | the bill's total |
| `expect charge("Linehaul") == $365.30`, `expect charge(FSC) == $11.75` | one line's amount, by label or code |
| `expect lines.len == 3` | the whole bill's lines |
| `expect fails`, `expect fails "This rate needs a weight"` | that it refuses (with that message) |

- A test reads the bill, the sample shipment, and the file's functions and constants. It
  can't read the tariff's own `let`s: check the charge they feed instead.
- The sample shipment is one pickup and one delivery with no commodities, and every input has its
  default.
- A version with a failing test can't be submitted for review. **Save as test** in the preview
  writes a test from a real shipment.

## Properties

A property checks a rule over a range of shipments instead of one:

```tariff
tariff "Per hundredweight"

/// Paid per hundred pounds.
input rate: Money/cwt = $18/cwt where $5/cwt..=$60/cwt
input minimumCharge: Money = $95

let weight = shipment.weight ?? fail "This rate needs a weight"
charge "Linehaul" = rate * weight.roundUp(to: 100 lb)
minimum minimumCharge

property "heavier never bills less" {
  vary shipment.weight in 1 lb..40_000 lb
  expect total nondecreasing in shipment.weight
}

property "never below the minimum" {
  vary shipment.weight in 1 lb..40_000 lb
  vary input.rate
  expect total >= $95
}
```

| Write | Does |
|---|---|
| `vary shipment.weight in 1 lb..40_000 lb` | tries many weights: both ends, every literal, default and band edge of that kind in the file (and one step either side), and evenly spread values |
| `vary input.rate` | tries values across the input's `where` constraint |
| `vary service in [Standard, Expedited]`, `vary shipment.hazmat` | tries each listed value; a `Bool` or enum alone tries all of them |
| `assume (shipment.weight ?? 0 lb) > 0 lb` | skips the cases where it's false; it can't read the bill |
| `expect total >= $95` | must hold in every case |
| `expect total nondecreasing in shipment.weight` | must never fall as the weight grows (`nonincreasing`: never rise) |

- `vary`, `given` and `assume` come first; ranges need both ends.
- The same file always runs the same cases (at most 256 per property).
- A failure is shrunk to the simplest case, or for `nondecreasing`, to the two neighbouring
  values that are out of order ("999 lb bills $310.00, 1000 lb bills $250.00"). **Save as test**
  writes it as an ordinary `test`, so the bug stays fixed.
- Properties block review like tests.
- **Coverage:** the Studio marks the lines no test or property reaches (an unreached `if` branch,
  `match` arm, option, charge or refusal) and counts them in the tests panel.

## Values to check in a test

```tariff
tariff "Literals"

charge "Freight" = $18.50/cwt * 1_200 lb

test "units convert exactly" {
  expect 1_200 + 2.35 == 1_202.35
  expect 500 lb + 2 cwt == 700 lb
  expect 1 cwt == 100 lb
  expect 4 h + 30 min == 270 min
  expect 48 in == 4 ft
  expect $426 == $426.00
  expect total == $222
  expect $2.35/mi * 812 mi == $1_908.20
  expect $1_000 * 18% == $180
  expect 0.5% * 3 == 1.5%
  expect 18%.value == 18
  expect $0.185/lb.in(cwt) == $18.50/cwt
  expect (1_240 lb).in(cwt).value == 12.4
  expect @2026-10-22.weekday == Thu
  expect @2026-10-24.weekend
  expect "Stop {2} of {3}" == "Stop 2 of 3"
}
```

## Conditions and libraries

An accessorial's apply-when rule is a condition file ending in one `Bool`:

```tariff
condition "Residential delivery"

/// The zip3s this contract treats as residential.
input residentialZip3s: List<Text> = []

lane.destination.zip3 in residentialZip3s and (shipment.weight ?? 0 lb) < 10_000 lb
```

A formula library holds functions, constants, types and enums for templates to import with
`use org.<name>.{ … }`. Everything is exported unless marked `private`, and a library
constant needs its type:

```tariff
library "accessorials"

/// Bills a liftgate at the delivery.
fn liftgate(fee: Money = $85) -> Money = fee

const RESIDENTIAL_FEE: Money = $95

type Surcharge = { label: Text, amount: Money }

enum Service { Standard, Guaranteed, Expedited }

test "the liftgate defaults to $85" {
  expect liftgate() == $85
}
```

A template version pins the library versions it imports, so publishing a library never reprices
an existing contract.

## Common checker errors

| The checker says | Fix |
|---|---|
| "… may be empty, so it can't be used as … here" | handle the optional: `?? fallback`, `?? fail "…"`, `require let`, `if let` |
| "… adds money per hundredweight to a weight" | multiply the rate by the quantity it is priced per, or convert with `.in(…)` |
| "… multiplies money per mile by a weight" | use the quantity the rate is per (`shipment.distance` for `Money/mi`) |
| "A charge's amount must be money" | the expression is still a rate or a quantity: multiply by the quantity |
| "… isn't in the accessorial charge catalog" | add the code under accessorial charges, or fix its spelling |
| "This match on … doesn't cover …" | add the missing ranges, or a `_` arm |
| "This arm overlaps an arm above it" | make the ranges meet without overlapping (`..500 lb`, `500 lb..`) |
| "Comparisons can't be chained" | `1 < n and n < 3` |
| "… isn't a variant of `StopKind`. Did you mean …?" | the suggested spelling |
| "… which a function can't read" | pass the value in as a parameter |
| "… which a test can't read" | expect on the bill (`total`, `charge(…)`, `lines`) instead of a `let` |
| "… calls itself through …" | remove the cycle; functions can't recurse |
| "A library's constant is part of its interface, so … needs a type" | `const NAME: Money = …` |
| "… takes 2 keys (…), but 1 is given" | give one key per matrix dimension |
| "`rate` must be from $0.50/mi to $15.00/mi; … is outside it" | the default or the contract value is outside the input's `where` bounds |
| "A choice needs at least two options" | add another `option`, or drop the `best` |
| "… falls from … at … to … at …: it must not fall as … grows" | a property found the inversion: fix the break, then **Save as test** |
| "A property can't expect a refusal" | narrow the property with `assume` or its ranges |

Hover a diagnostic in the Studio for its quick fixes; most of these have one.
