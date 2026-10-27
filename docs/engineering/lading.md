# Lading

Lading is the language an organisation uses to bring its own EDI to Trenova: how a trading
partner's files become loads, statuses and invoices, what Trenova sends back and when, and the
rules that decide what happens in between. It replaces the flat EDI templates, the transform
pipeline, Starlark and the hand-written X12 mappings.

This document is normative. A change to the language changes this file, the crate
(`services/crates/edi/src/lading`) and its corpus together. How X12 itself is read, validated
and acknowledged is [edi-x12.md](edi-x12.md); the editor and the studio are "The editor" below.

## Why a language

A TMS usually sells EDI as a service: a specialist maps each partner by hand in a tool only they
can use. Trenova does not. Every organisation builds its own, so the language has to be:

- **readable by a planner and precise enough for an EDI analyst**, using X12's own names
  (`B204`, `0300`, `N104`) with the guide's descriptions one hover away;
- **checked before it runs**, against the partner's guide and Trenova's documents, so a map that
  would write a date into a number, drop a mandatory element or miss a code never ships;
- **the same in the browser and on the server**: one WebAssembly module checks, previews and tests
  in the editor, and runs in production;
- **total**: every map, rule and flow step terminates, reaches nothing outside what Trenova hands
  it, and costs a bounded amount of work.

It shares Tariff's conventions wherever both have the same idea (`==`, `??`, `///` docs,
`{expr}` interpolation, `@` dates, units, `test`, `property`, `vary`, `given`, `expect`), so
someone who writes rates can read a map.

## A first look

A small project for one partner, ACME, which tenders loads to an organisation that hauls them.

```lading
partner acme "ACME Retail"

interchange {
  them ZZ "ACMERETAIL"
  us ZZ "TRNV-ACME"
  group "ACME" <-> "TRENOVA"
  version 004010
  usage test
}

zone America/Chicago

/// Accept ACME tenders without a planner.
setting autoAccept: Bool = false
/// The heaviest load accepted automatically.
setting maxWeight: Weight = 45_000 lb
```

```lading
map tenderIn
  from x12.004010.204
  to   loadTender
  use  common

key      <- B204
purpose  <- B2A01 via purposes
bol      <- L11[02 == "BM"].01
customer <- lookup Customer(N1[01 == "BT"].04)
weight   <- L301 ?? 0 lb
total    <- money(L305, partner.currency)

stops <- each 0300 as stop {
  sequence <- stop.S501
  type     <- stop.S502 via stopTypes else reject "Stop {stop.S501} has no type"
  location <- lookup Location(stop.0310.N104) learn address(stop.0310)
  name     <- stop.0310.N102.titleCase()
  address  <- address(stop.0310)
  window   <- window(stop.G62, location.zone ?? partner.zone)
}

test "a two-stop tender" {
  given file "samples/two-stops.x12"
  expect stops.count == 2
  expect stops.last.type == Delivery
}
```

```lading
library common

crosswalk stopTypes: Code<S502> <-> StopType {
  "LD", "CL", "PL" <-> Pickup
  "UL", "CU", "PU" <-> Delivery
}

crosswalk purposes: Code<B2A01> <-> TenderPurpose {
  "00" <-> Original
  "01" <-> Cancel
  "04" <-> Change
}

fn window(dates: [Segment<G62>], zone: Zone) -> Window? {
  let opens = dates.find(d => d.01 in ["37", "53", "69", "70"])
  let closes = dates.find(d => d.01 in ["38", "54"])
  Window {
    start <- at(opens.02, opens.04, zone)
    end   <- at(closes.02, closes.04, zone)
  }
}
```

```lading
rules tenderChecks for tenderIn

rule "pickup comes before delivery" {
  require stops.first.window.start < stops.last.window.start
    else decline "Delivery is scheduled before pickup"
}

rule "within what we haul automatically" {
  require weight <= partner.maxWeight
    else hold "Over {partner.maxWeight}: a planner decides"
}
```

```lading
flow tender with acme

receive 204 as tender
  via tenderIn, rules tenderChecks
  key tender.key
  acknowledge with 997
  on Original -> create
  on Change   -> update stops, commodities, refs keep customer, total
  on Cancel   -> cancel unless dispatched else hold "ACME cancelled after dispatch"

respond 990 via responseOut
  accept automatically when partner.autoAccept
  decline automatically when declined
  otherwise planner decides within 2 h else escalate "ACME tender {tender.key} needs an answer"

send 214 via statusOut on StopArrived, StopDeparted, Delivered batch within 5 min

expect 997 for every send within 30 min else resend 1 then escalate "ACME has not acknowledged"
```

## Projects and files

An organisation has one **project** per trading partner. A project is a set of text files and a
set of sample files, saved together as a **version** (see "Versions"). A file's kind is named by
its first keyword:

| Kind | Header | Holds |
|---|---|---|
| Partner | `partner name "Label"` | the interchange identity, time zone, currency and settings; exactly one per project |
| Guide | `guide name from base "Title"` | how this partner narrows a standard X12 guide |
| Map | `map name from source to target` | how one message becomes one Trenova document, or the reverse |
| Rules | `rules name for map` | what must be true of a map's document, and what happens when it is not |
| Flow | `flow name with partner` | the conversation: what arrives, what Trenova answers, what it sends and when, what it waits for |
| Library | `library name` | functions, crosswalks and constants other files `use` |

Maps, rules, flows and libraries may also declare their own `fn`, `crosswalk` and `const` after
the header. Every file may hold `test` and `property` blocks.

**Names.** File-level names (`tenderIn`, `common`, `acme`) are camelCase and unique across the
project. Guides are named with dots (`acme.204`); the built-in base guides are
`x12.<version>.<set>` (`x12.004010.204`). File paths are free (`maps/tender-in.lading`); the
studio groups them by kind.

**Samples** are the partner's real messages and Trenova documents kept with the project:
`.x12`, `.edi` and `.txt` files are X12, `.csv` and `.json` are as named, and
`.document.json` is a Trenova document for an outbound test. Tests read them by path; the studio
runs every map against them as you type; coverage and replay use them.

## Lexical structure

- **Statements end at the end of a line.** An expression continues onto the next line when the
  line ends with an operator, a comma or an open bracket, or the next line starts with `else`,
  `.` or an operator. A segment block may put several assignments on one line, separated by
  commas: `B10 { 01 <- proNumber, 02 <- key }`.
- **Comments:** `//` runs to the end of the line. `///` is a doc comment for the next
  declaration (a setting, function, crosswalk, constant, rule or test) and is shown in hover,
  completion and the settings form.
- **Names:**
  - names are camelCase (`stopTypes`, `autoAccept`);
  - types and enum variants are PascalCase (`StopType`, `Delivery`);
  - X12 references are uppercase letters and digits: a segment ID (`N1`, `G62`, `B2A`), a
    segment ID followed by a two-digit element position (`N104`, `B2A01`, `G6202`), or an
    interchange qualifier (`ZZ`, `01`);
  - loop IDs are digits with a leading zero (`0300`, `0310`), or digits followed by a letter
    (`2000A`). After a `.`, any run of digits is a loop ID or an element position, never a
    decimal (`stop.0310`, `seg.02`).
- **Keywords:** `fn let const crosswalk rule require else reject decline hold warn fix when for
  in where each as via lookup learn match if then test property given expect vary exists and or
  not true false none`. Every other word the grammar uses (the headers' `partner`, `guide`,
  `map`, `rules`, `flow`, `library`, `from`, `to`, `use`, and `interchange`, `them`, `us`, `group`, `version`, `usage`, `zone`, `currency`, `setting`,
  `receive`, `respond`, `send`, `expect`, `key`, `acknowledge`, `on`, `create`, `update`, `keep`,
  `cancel`, `unless`, `apply`, `ignore`, `accept`, `automatically`, `otherwise`, `planner`,
  `decides`, `within`, `escalate`, `batch`, `until`, `resend`, `then`, `every`, `required`,
  `unused`, `codes`, `length`, `repeat`, `fixed`, `loop`, `min`, `max`, `severity`, `syntax`,
  `at`, `code`, `samples`, `file`, `text`, `advance`, `event`, `sent`, `held`, `declined`,
  `rejected`, `escalated`, `warned`, `no`) is contextual: it means something only where the
  grammar expects it and is an ordinary name elsewhere.
- **Text:** `"double quoted"` with `{expr}` interpolation, `{{` for a literal brace and the escapes
  `\" \\ \n \t \u{...}`. `r"..."` is raw: no escapes and no interpolation, for patterns
  (`matches(r"^\d{9}$")`).
- **Numbers:**
  - `1200`, `1_200`, `12.50`;
  - with a unit: `45_000 lb`, `20 t`, `53 ft`, `30 min`, `2 h`, `-10 °F` (see "Measures");
  - money: `$1_850.00` is US dollars; another currency is `money(12.50, "CAD")`;
  - percent: `5%`;
  - date `@2026-01-10`, time `@14:30`, instant `@2026-01-10T14:30Z` or
    `@2026-01-10T08:30-06:00`, local time in a zone `@2026-01-10T08:30[America/Chicago]`.

## Types

| Type | Values | X12 elements of this type |
|---|---|---|
| `Text` | any text | `AN`, `B` |
| `Code<E>` | text from the code list of element `E` (`Code<S502>`), a subtype of `Text` | `ID` |
| `Int` | whole numbers | `N0` |
| `Decimal` | exact decimals | `R`, `N1`..`N9` (read with their implied places) |
| `Bool` | `true`, `false` | |
| `Date` | a calendar date | `DT` |
| `Time` | a time of day | `TM` |
| `DateTime` | an instant with the zone it is read or written in | |
| `Zone` | an IANA time zone (`America/Chicago`) | |
| `Duration` | `30 min`, `2 h`, `1 d` | |
| `Percent` | `5%` | |
| `Money` | an amount in a currency | |
| `Weight`, `Length`, `Volume`, `Temperature` | measures with units | |
| `[T]` | lists | repeated segments, loops, element repetitions |
| `[Text: T]` | maps from text to `T`: `[key: value, ...]`, `[:]` empty (a document's `refs`) | |
| `T?` | `T` or `none` | anything not certain to be present |
| `Segment<ID>`, `Loop<ID>` | an X12 segment or loop instance | |
| records | document records (`Stop`, `Window`, `Address`) and `{ name: value }` literals | |
| enums | document enums (`StopType`, `TenderPurpose`) | |
| references | `Customer`, `Location`, `Commodity`, ... (see "Lookups") | |

- **Optionals propagate through paths.** Reading `stop.0310.N104` when the stop has no `0310`
  loop is `none`, not an error. An expression that reads something optional is optional.
- **Handling none:** `a ?? b` is `a`, or `b` when `a` is none. An assignment's `else` does the
  same, or names what happens instead (`else reject "..."`). `exists x` is `true` when `x` is not
  none. A required document field can't be assigned an optional without handling it
  (`optional-unhandled`, with fixes for `??`, `else` and `else reject`).
- **Equality and order:** `==` and `!=` compare two values of the same type; `<`, `<=`, `>`, `>=`
  compare numbers, measures in the same dimension, money in the same currency, dates, times and
  instants. `in` tests membership in a list (`d.01 in ["37", "53"]`), `matches` a raw pattern.
- **Arithmetic** is exact: `+ - * /` on numbers, measures and money in the same dimension and
  currency, `* /` by numbers, `DateTime + Duration`. Division by zero is none. Comparing or
  adding measures converts units; the result carries the left operand's unit.
- **Lambdas** are `x => expr` and appear only as arguments (`stops.count(s => s.type == Pickup)`).

### Measures

| Dimension | Units |
|---|---|
| Weight | `lb`, `kg`, `oz`, `t` (metric tonne), `ton` (short ton) |
| Length | `in`, `ft`, `yd`, `mi`, `cm`, `m`, `km` |
| Volume | `ft3`, `m3`, `gal`, `l` |
| Temperature | `°F`, `°C` (comparison and difference only) |
| Duration | `s`, `min`, `h`, `d` |

`weight(value, unitCode)`, `length(value, unitCode)`, `volume(value, unitCode)` and
`temperature(value, scale)` build a measure from a number and the X12 unit code next to it:
weights `L` (pounds), `K` (kilograms), `E` (metric tons); lengths `F`, `I`, `M`; volumes `E`
(cubic feet), `X` (cubic meters), `G` (gallons); temperatures `F`, `C`. An unknown code is none.
Trenova stores weights in pounds, lengths in feet, volumes in cubic feet and temperatures in °F;
a document field converts on assignment.

## Paths into X12

Inside a map whose source or target is a guide, X12 is addressed the way an analyst writes it:

| Path | Means |
|---|---|
| `B204` or `B2.04` | element 04 of the B2 segment at the transaction's top level |
| `L11` | the L11 segments at the top level, a list |
| `L11[02 == "BM"]` | the first L11 whose element 02 is `BM`, optional |
| `L11[02 == "BM"].01` | its element 01 |
| `0300` | the instances of loop 0300, a list |
| `stop.0310.N102` | element 02 of N1 in the stop's 0310 loop |
| `stop.G62` | the stop's G62 segments, a list |
| `C001.02` | component 02 of composite element C001 |

- Unqualified X12 references always start at the transaction's top level. Inside
  `each 0300 as stop`, the stop's segments are `stop.S501`, never a bare `S501`
  (`x12-unqualified`, with the fix).
- `[cond]` picks the first match; inside it, `01`, `02`, ... are that segment's elements.
  `.find(s => ...)`, `.filter(s => ...)`, `.count`, `.first` and `.last` work on any list.
- The path's type comes from the guide: `B204` is `Text`, `S502` is `Code<S502>`, `G6202` is
  `Date`, `L305` is `Decimal` (an `N2` read with its implied places), and an element in a
  segment or loop that may be absent is optional. A transaction that fails validation is rejected
  before any map runs, so a mandatory element of a mandatory segment is never none.
- A path the guide doesn't have is an error (`x12-unknown`), with the nearest real path
  suggested; a segment the partner's guide marks `unused` is a warning (`x12-unused`).

## Maps

```lading
map name
  from <source>
  to   <target>
  use  <library>, <library>
```

A map is **inbound** when its source is a guide, `csv { ... }` or `json` and its target a
document type; **outbound** when its source is a document type and its target a guide. A map is
pure: it reads its input, the partner's settings and Trenova's lookups, and produces one output.
When it runs, and what happens to the result, is the flow's business.

### Inbound maps

The body assigns the target document's fields:

```lading
key      <- B204
bol      <- L11[02 == "BM"].01 ?? L11[02 == "BL"].01
refs     <- refs(L11)
refs["PO"] <- N9[01 == "PO"].02
custom.po_number <- L11[02 == "PO"].01

stops <- each 0300 as stop {
  let party = stop.0310
  sequence <- stop.S501
  name     <- party.N102.titleCase()
}

when B2A01 == "05" {
  purpose <- Change
}
```

- **`field <- expr`** assigns. An assignment whose value is none leaves the field as it was, so
  later lines can fill a gap (`bol <- L11[02 == "BL"].01` after the `BM` line). A later
  assignment that is never none overwrites; overwriting a value that is never none is a warning
  (`assignment-overwritten`).
- **`else`** after an assignment gives a default (`else 0 lb`) or an outcome: `else reject "..."`,
  `else hold "..."` or `else warn "..."` (see "Rules"). A required field must be assigned on
  every path (`field-unassigned`).
- **`each <list> as x { ... }`** builds a list of records, one per item, in order. The list is
  usually a loop (`each 0300 as stop`) but may be any list (`each L5 as line`).
- **`let name = expr`** binds a value for the rest of its block. **`when cond { ... }`** assigns
  only when `cond` is true.
- **`refs["PO"] <- ...`** sets one entry of a `[Text: Text]` field.
- **`custom.<field>`** is the organisation's custom fields on the target's record (a shipment's,
  an invoice's), named as their definitions name them (`custom.po_number`) and typed from them.
  Completion after `custom.` lists them; a name the organisation has not defined is an error
  that lists the ones it has, or says where to add one.

### Outbound maps

The body says which segments to write. The runtime writes the envelope (ISA, GS, ST, SE, GE,
IEA), the control numbers and the counts.

```lading
map statusOut
  from shipmentStatus
  to   x12.004010.214

B10 { 01 <- proNumber, 02 <- key, 03 <- org.scac }
L11 when bol { 01 <- bol, 02 <- "BM" }
each refs as ref {
  L11 { 01 <- ref.value, 02 <- ref.key }
}

0205 <- each events as event {
  LX { 01 <- index + 1 }
  0210 {
    AT7 {
      01 <- event.kind via statusCodes
      02 <- event.reason ?? "NS"
      05 <- event.at
      06 <- event.at
      07 <- event.at.zoneCode()
    }
    MS1 when exists event.city { 01 <- event.city, 02 <- event.state, 03 <- event.country }
  }
}
```

- **`SEG { 01 <- expr, ... }`** writes one segment; several blocks for the same ID write several,
  in source order. **`SEG when cond { ... }`** writes it only when `cond` holds.
- **`LOOP { ... }`** writes one instance of a loop; **`LOOP <- each list as x { ... }`** writes
  one per item, with `index` counting from 0. A segment written inside a loop belongs to that
  instance, so a stop's dates and address can't drift to another stop.
- **`each list as x { ... }`** without a loop writes its segments once per item at the current
  level.
- The order blocks are written in doesn't matter: the runtime places every segment where the
  guide puts it.
- **Element values convert by the element's type:** a `Date` or `DateTime` into a `DT` is
  `CCYYMMDD` (`YYMMDD` when the element holds six); a `Time` or `DateTime` into a `TM` is
  `HHMM`; a `DateTime` is written in its own zone. A `Decimal` or `Money` into an `Nn` is
  shifted by its places with no point (`1250.50` into an `N2` is `125050`); into an `R` it keeps
  its point and drops trailing zeros. A `Code` or `Text` into an `ID` must be one of the
  element's codes (`code-not-in-list` for a literal, a runtime finding otherwise). Text that is
  too long for its element is an error at runtime; when the document field's maximum is longer
  than the element's, the checker warns (`may-exceed-length`) and offers `.truncate(n)`.
- A mandatory element no assignment fills, or a mandatory segment no block writes, is an error
  (`x12-mandatory-unwritten`). What the map writes is read back against the guide before it is
  sent; any finding is the message's.

### Other sources

```lading
map ordersCsv
  from csv {
    order:  Text
    pickup: Date "MM/dd/yyyy"
    weight: Weight lb
    origin: Text
  }
  to loadTender

stops <- each rows as row { ... }
```

- **`csv { column: Type [format] }`** reads a file with a header row. Each row is a record named
  `row` inside `each rows as row`; columns are matched by header name, case-insensitively. A
  `Date`, `Time` or `DateTime` column takes a pattern (`"MM/dd/yyyy"`), a measure a unit, a
  `Money` column a currency.
- **`json`** reads any JSON document as `input`. Paths use `.name` and `[index]` (an index past
  the end, or a missing key, is none); `each order.stops as stop` walks an array, and the list
  functions work on one, called with their parentheses (`order.stops.count()`; `.count` alone
  reads a key named `count`). Every value is typed by the conversion that reads it (`text(x)`,
  `int(x)`, `decimal(x)`, `date(x, "yyyy-MM-dd")`), since JSON carries no schema.

## Documents

The targets of inbound maps and sources of outbound maps. Each field is typed, documented and
shown in completion and hover. `?` marks an optional field; every other field must be assigned
by an inbound map.

**`loadTender`** (204):

| Field | Type |
|---|---|
| `key` | `Text`: the partner's identifier for the load, which every later message quotes |
| `purpose` | `TenderPurpose`: `Original`, `Change`, `Cancel` |
| `bol`, `proNumber`, `poNumber` | `Text?` |
| `customer` | `Customer?` |
| `serviceType`, `shipmentType`, `rating` | `ServiceType?`, `ShipmentType?`, `RatingTemplate?` |
| `payment` | `Payment?`: `Prepaid`, `Collect`, `ThirdParty` |
| `equipment` | `Text?` |
| `weight` | `Weight?` |
| `pieces` | `Int?` |
| `temperature` | `{ min: Temperature?, max: Temperature? }?` |
| `total`, `freightCharge`, `otherCharges` | `Money?` |
| `refs` | `[Text: Text]` |
| `notes` | `[Text]` |
| `stops` | `[Stop]`: `sequence: Int`, `type: StopType` (`Pickup`, `Delivery`), `location: Location?`, `name: Text?`, `code: Text?`, `address: Address?`, `window: Window`, `weight: Weight?`, `pieces: Int?`, `refs`, `notes` |
| `commodities` | `[CommodityLine]`: `commodity: Commodity?`, `description: Text`, `weight: Weight?`, `pieces: Int?`, `hazmat: Bool` |
| `charges` | `[Charge]`: `accessorial: Accessorial?`, `code: Text?`, `description: Text?`, `amount: Money` |
| `custom` | the organisation's shipment custom fields |

**`tenderResponse`** (990): `key: Text`, `decision: Decision` (`Accept`, `Decline`),
`reason: Text?` (at most 200 characters), `scac: Text?`, `respondedAt: DateTime`, `refs`.

**`shipmentStatus`** (214): `key: Text`, `bol`, `proNumber`, `shipperReference: Text?`,
`weight: Weight?`, `pieces: Int?`, `equipment: Text?`, `refs`, and `events: [StatusEvent]`:

| Field | Type |
|---|---|
| `kind` | `EventKind`: `Dispatched`, `ArrivedPickup`, `Loaded`, `DepartedPickup`, `EnRoute`, `ArrivedDelivery`, `Delivered`, `DepartedDelivery`, `Delayed`, `AppointmentSet`, `EtaChanged`, `Exception` |
| `reason` | `Text?`: the partner's reason code, kept as sent |
| `at` | `DateTime` |
| `local` | `Bool?`: the time is the clock where it happened (AT707 `LT` or none); Trenova reads it again in the matched stop's zone |
| `stop` | `Int?`: the stop sequence |
| `city`, `state`, `country` | `Text?` |
| `latitude`, `longitude` | `Decimal?` |
| `equipment` | `Text?` |
| `eta` | `DateTime?` |

**`freightInvoice`** (210): `number: Text`, `date: Date`, `deliveryDate: Date?`, `key: Text?`,
`bol`, `proNumber: Text?`, `weight: Weight?`, `billTo: Party?` (`name`, `address`), `total: Money`,
`lines: [InvoiceLine]` (`sequence: Int`, `code: Text?`, `accessorial: Accessorial?`,
`description: Text?`, `amount: Money`, `rate: Decimal?`, `weight: Weight?`), `shipments:
[InvoicedShipment]` (`proNumber`, `bol`, `poNumber: Text?`, `deliveryDate: Date?`,
`amount: Money?`), `refs`.

**Records** used across documents: `Address` (`line1: Text`, `line2: Text?`, `city: Text`,
`state: Text?`, `postal: Text?`, `country: Text?`), `Window` (`start: DateTime`, `end: DateTime?`),
`Party` (`name: Text`, `address: Address?`).

Acknowledgments (997, 999, TA1) are not documents: Trenova writes them itself from its reading of
what it received ([edi-x12.md](edi-x12.md)).

## Lookups

A partner's codes for customers, locations and the rest become Trenova records through the
organisation's mapping profiles:

```lading
customer <- lookup Customer(N1[01 == "BT"].04)
location <- lookup Location(stop.0310.N104) learn address(stop.0310)
```

- `lookup Kind(code)` is a `Kind?` reference with the record's `id`, `code` and `label` (and a
  location's `zone`). The kinds are `Customer`, `Location`, `Commodity`, `Accessorial`,
  `ServiceType`, `ShipmentType`, `RatingTemplate` and `ServiceFailureReason`.
- A code with no mapping makes the lookup none and records it as **unmapped**. A message whose
  document has unmapped codes waits for someone to map them; once mapped, the mapping is kept
  for this partner and every waiting message resumes on its own.
- **`learn <expr>`** gives Trenova what it needs to suggest a match for an unmapped code (an
  address, a name). The person mapping it sees the suggestions; a suggestion is never applied
  without them.
- Lookups are answered by the host. A run that needs codes it was not given answers with the
  codes it needs, and the host runs it again with them, so a map makes one batched round trip
  however many lookups it has.

## Functions, crosswalks and constants

```lading
/// The appointment window from a stop's dates.
fn window(dates: [Segment<G62>], zone: Zone) -> Window? {
  let opens = dates.find(d => d.01 in ["37", "53", "69", "70"])
  Window {
    start <- at(opens.02, opens.04, zone)
    end   <- at(dates.find(d => d.01 in ["38", "54"]).02, opens.04, zone)
  }
}

fn zip5(postal: Text) -> Text = postal.digits().take(5)

const heavy = 40_000 lb
```

- **`fn name(param: Type, other: Type = default) -> Type = expr`**, or a block of `let` lines
  ending in its value. Functions may call each other but not recursively (`call-cycle`).
- A **record literal** is `Type { field <- expr ... }` (or `{ name: expr, ... }` when the type is
  known from where it goes). It is none when a required field is none.
- **`const name = expr`** is a value computed once.

```lading
crosswalk charges: AccessorialCode <-> Code<L108> {
  "FUEL"      <-> "FUE"
  "DETENTION" <-> "DET"
  "LUMPER"     -> "LUM"
  "MSC"       <-  "ZZZ"
  else         -> "MSC"
}
```

- **`crosswalk name: A <-> B { ... }`** pairs values. `x via name` reads left to right when `x`
  is an `A`, right to left when it is a `B`. `<->` works both ways, `->` only left to right,
  `<-` only right to left; a pair may list several values on either side (`"LD", "CL" <->
  Pickup`), and the first is the one written. `else -> v` (or `else <- v`) answers anything
  unpaired in that direction; without it an unpaired value is none.
- When `A` and `B` are the same type, `via name` reads left to right and `via name.back` right to
  left.
- **Checked both ways:** writing an enum through a crosswalk that doesn't cover every variant is
  an error (`crosswalk-incomplete`); reading an element through one that misses a code the
  partner's guide allows is a warning (`crosswalk-misses-code`), with a fix that adds the pair.

### The standard library

Every function is also a method on its first argument: `postal.zip5()` is `zip5(postal)`.

| Area | Functions |
|---|---|
| Text | `trim`, `upper`, `lower`, `titleCase`, `take(n)`, `drop(n)`, `truncate(n)`, `pad(n, with: " ", left: false)`, `replace(old, new)`, `split(sep)`, `join(list, sep)`, `digits`, `letters`, `clean` (removes characters X12 can't carry and the partner's separators), `startsWith`, `endsWith`, `contains`, `matches(pattern)`, `length`, `isBlank` |
| Numbers | `round(places = 0)`, `floor`, `ceil`, `abs`, `min`, `max`, `int(x)`, `decimal(x)`, `text(x)` |
| Money | `money(amount, currency)`, `amount(money)`, `currency(money)` |
| Measures | `weight(value, code)`, `length(value, code)`, `volume(value, code)`, `temperature(value, scale)`, `in(measure, unit)` (`weight.in(kg)`), `value(measure, unit)` |
| Dates and times | `at(date, time, zone)`, `date(text, pattern)`, `time(text, pattern)`, `dateTime(text, pattern, zone)`, `date(dt)`, `time(dt)`, `zone(dt)`, `inZone(dt, zone)`, `zoneCode(dt)` (`CT`, `ET`, `LT`, ...), `zoneFromCode(code)`, `addBusinessDays(date, n)`, `now()` |
| Lists | `count`, `first`, `last`, `isEmpty`, `find(f)`, `filter(f)`, `map(f)`, `any(f)`, `all(f)`, `sum(f)`, `min(f)`, `max(f)`, `sortBy(f)`, `distinct`, `take(n)`, `drop(n)`, `flatten`, `get(i)` |
| X12 | `refs(segments)` (an L11 or N9 list as qualifier to value), `address(loop)` (N3 and N4 of a name loop as an `Address`), `party(loop)` (N1, N3, N4 as a `Party`), `elements(segment)` (every element as text) |
| Addresses | `zip5`, `zip9`, `stateCode` (a US or Canadian state or province name or code to its code), `countryCode(text, iso: 2)`, `phone(text)` (E.164) |
| Checks | `checkDigit(text, "mod10")`, `isDate(text, pattern)`, `isNumber(text)` |

## Rules

```lading
rules tenderChecks for tenderIn

rule "a consignee has a postal code" {
  for stop in stops where stop.type == Delivery {
    require exists stop.address.postal
      else reject at stop.address.postal "The consignee's postal code is missing"
  }
}

rule "ACME sends nine-digit zips" {
  for stop in stops where stop.address.postal.matches(r"^\d{9}$") {
    fix stop.address.postal <- stop.address.postal.zip5()
  }
}

rule "hazmat needs eyes" {
  warn "Hazmat on board: check placards" when commodities.any(c => c.hazmat)
}
```

A rules file belongs to one map and runs on the document that map produced (inbound), or the
document about to be written (outbound), before Trenova acts on it. It sees the libraries its map
uses and may `use` more on its header, as a flow or a library may. A function or crosswalk from a
library the file does not use is an error that names the library, with a fix that adds the
`use`. Inside a rule:

- **`require cond else outcome`**: the outcome happens when `cond` is false (or none).
- **`outcome when cond`**: the outcome happens when `cond` is true.
- **`for x in list where cond { ... }`** runs its statements per item.
- **`fix path <- expr`** changes the document; the change is recorded with the rule, the old
  value and the new.
- **`let`** binds as in a map.

| Outcome | Inbound | Outbound |
|---|---|---|
| `reject [at path] "message"` | the transaction is rejected in Trenova's 997 or 999, with an element note at the X12 element `path` was read from; nothing is created | the message is not sent; it fails with the message |
| `decline "message" [code "X"]` | a valid message Trenova will not act on: a 990 decline, when the flow declines automatically, or a decline proposal for a planner | not allowed |
| `hold "message"` | the document is created and held in the exception queue until a person releases it | the message is held unsent until a person releases or edits it |
| `warn "message"` | a note on the message | a note on the message |

Messages may interpolate (`"Over {partner.maxWeight}"`). A rule that reads nothing it can't be
sure of and can never fail is a warning (`rule-always-passes`): the partner's guide may already
require what it checks.

## Guides

```lading
guide acme.204 from x12.004010.204 "ACME Load Tender Guide rev 3.2"

/// ACME's load number; every later message quotes it.
B2 { 04 required }
B2A { 01 codes "00", "01", "04", "05" }
L11 max 10 { 02 codes "BM", "PO", "CR", "IA" }
NTE unused
loop 0300 min 2 max 12
0300.S5 { 02 codes "LD", "UL" }
0300.G62 { 01 codes "69", "70", "37", "38", "53", "54"; 04 required }
0300.0310.N1 { 01 codes "SH", "CN"; 04 required }
0300.0310.N4 { 03 length 5..9 }
severity invalid-code warning
```

A partner guide narrows its base (a built-in `x12.*` guide or another partner guide):

- **segments:** `SEG required`, `SEG unused`, `SEG max n`, `SEG min n`, and a block of element
  lines `{ 01 ...; 02 ... }` (lines or `;` separate them);
- **elements:** `required`, `unused`, `codes "a", "b"`, `length a..b`, `repeat n`, `fixed "v"`;
- **loops:** `loop ID required`, `loop ID unused`, `loop ID min n max n`;
- **syntax notes:** `SEG syntax P0304 R0102`;
- **severity:** `severity <finding> error|warning|info` changes how a finding kind is reported
  for this partner (the kinds are those in [edi-x12.md](edi-x12.md)).

A segment line names its loops (`0300.0310.N1`), or the segment's ID alone when the base guide
has it in only one place (`S5` for `0300.S5`); an ID found in several loops is an error
(`guide-ambiguous-path`) that lists them.

Only narrowing is allowed: making something optional mandatory or unused, shortening a length,
restricting codes, lowering a maximum, adding a syntax note. Anything else is an error
(`guide-widens`), with the base's rule shown beside it. The onboarding agent drafts a guide from
the partner's PDF; each line it writes carries the page it came from, shown on hover.

## The partner

```lading
partner acme "ACME Retail"

interchange {
  them ZZ "ACMERETAIL"
  us ZZ "TRNV-ACME"
  group "ACME" <-> "TRENOVA"
  version 004010
  separators "*" "~" ">" "^"
  usage test
}

zone America/Chicago
currency "USD"

/// Accept ACME tenders without a planner.
setting autoAccept: Bool = false
/// The heaviest load accepted automatically.
setting maxWeight: Weight = 45_000 lb in 1 lb..80_000 lb
```

- **`interchange`** is the envelope: the partner's and the organisation's ISA qualifier and ID,
  the GS application codes (theirs `<->` ours), the X12 version, the separators Trenova writes,
  and `usage test` or `usage production` (ISA15). A project is certified before it may be
  `production` (see "Versions").
- **`zone`** is the zone for times that carry none of their own and have no stop location to
  take one from. **`currency`** is the default currency (`partner.currency`).
- **`setting name: Type = default [in range]`** declares a value a planner can change on the
  partner's page without a new version. Its doc comment is its label. Setting types are `Bool`,
  `Int`, `Decimal`, `Text`, `Percent`, `Money`, `Duration` and the measures. Code reads them as
  `partner.name`.
- `partner` also has `id`, `name` and `code`; `org` has `name`, `scac` and `zone`.

## Flows

A flow is the conversation with a partner about one thing (a tender, an invoice) from its first
message to its last. Trenova runs one instance per `key`.

```lading
flow tender with acme

receive 204 as tender
  via tenderIn, rules tenderChecks
  key tender.key
  acknowledge with 997
  on Original -> create
  on Change   -> update stops, commodities, refs keep customer, total
    hold when tender.total != current.total "ACME changed the rate"
  on Cancel   -> cancel unless dispatched else hold "ACME cancelled after dispatch"

respond 990 via responseOut
  accept automatically when partner.autoAccept
  decline automatically when declined
  otherwise planner decides within 2 h else escalate "ACME tender {tender.key} needs an answer"

send 214 via statusOut on StopArrived, StopDeparted, Delivered, EtaChanged batch within 5 min until Delivered

send 210 via invoiceOut on InvoicePosted

expect 997 for every send within 30 min else resend 1 then escalate "ACME has not acknowledged {message.set} {message.control}"
```

**`receive <set> as name`** handles each message of that set from the partner:

- **`via map[, rules rules]`** reads it; the rules' outcomes come with the document.
- **`key expr`** is the instance the message belongs to. Every later message with the same key
  joins it.
- **`acknowledge with 997|999|none`** is the functional acknowledgment Trenova sends for the
  group the message came in.
- **`on <purpose> -> action`** for a document with a `purpose`, or a single `-> action` for one
  without. The actions are:
  - `create`: a new load from the document;
  - `update fields... [keep fields...] [hold when cond "message"]`: apply only the listed fields
    of the document to the existing load, never the kept ones, and hold instead when `cond` holds
    (`current` is the load as it is now);
  - `cancel [unless state else action]`: cancel, or do `action` when the load is in `state`
    (`dispatched`, `delivered`, `invoiced`);
  - `apply`: hand the document to Trenova's normal handling for its set (a 990 to the tender it
    answers, a 214 to the shipment's tracking, a 210 to carrier invoice matching);
  - `hold "message"`, `ignore`.
- A `reject` from the rules rejects the transaction whatever the action; a `hold` holds after the
  action.

**`respond <set> via map`** answers a received tender:

- `accept automatically when cond` and `decline automatically when cond`, read in order;
  `declined` is true when the rules declined, and `held` when they held;
- `otherwise planner decides [within duration else escalate "message"]`, or `otherwise accept`,
  or `otherwise decline "message"`;
- the response's document is built by Trenova (`key`, `decision`, `reason`, `respondedAt`) and
  written by the map.

**`send <set> via map on events [where cond] [batch within duration] [until event]`** writes a
message when something happens in Trenova. The events are `TenderOffered`, `TenderAccepted`,
`TenderDeclined`, `TenderCancelled`, `Dispatched`, `StopArrived`, `StopDeparted`, `Delivered`,
`EtaChanged`, `ServiceFailure`, `InvoicePosted` and `InvoiceVoided`. `batch within` gathers the
events of one instance that land within the window into one message; `until` stops sending once
that event has been sent.

**One flow per message.** A received set, a response and an event go to the first flow in the
project that names them, so a second flow that receives the same set, responds with the same set
or sends on the same event is an error there (`flow-receive-taken`, `flow-respond-taken`,
`flow-event-taken`); within one flow, sending the same set twice on one event is
`duplicate-send`.

**`expect <ack> for every send within duration else resend n then escalate "message"`** watches
the partner's acknowledgment of everything this flow sends: a missing one is resent up to `n`
times, then escalated to the exception queue. A rejected one is escalated at once.

**Flows Trenova starts.** A broker tendering to a carrier starts the conversation:

```lading
flow carrierTender with swift

send 204 via tenderOut on TenderOffered key tender.key
receive 990 as response via responseIn key response.key -> apply
receive 214 as status via statusIn key status.key -> apply
receive 210 as invoice via invoiceIn key invoice.key -> apply
expect 997 for every send within 30 min else escalate "Swift has not acknowledged"
```

A flow is a function from its state and one event to its new state and the actions Trenova
takes. The crate computes it (`lading.flowStep`); the TMS runs each instance as a Temporal
workflow that feeds it events (messages, Trenova's events, a planner's decision, timers,
acknowledgments) and carries out its actions. Flow tests run the same function.

## Tests

```lading
test "a two-stop tender becomes a two-stop load" {
  given file "samples/two-stops.x12"
  given lookup Customer("ACME-BT") = "Acme Corp"
  expect stops.count == 2
  expect stops[1].window.start == @2026-01-11T16:00[America/Chicago]
  expect total == $1_850.00
  expect no holds
}

test "an arrival writes the right AT7" {
  given shipmentStatus {
    key: "L-1001"
    events: [{ kind: ArrivedDelivery, at: @2026-01-10T14:30Z, stop: 2, city: "Chicago", state: "IL" }]
  }
  expect x12 contains "AT7*X1*NS***20260110*0830*CT~MS1*Chicago*IL~"
}
```

- **`given file "path"`** reads a sample (inbound); **`given text "..."`** the given text;
  **`given <document> { ... }`** builds an outbound map's input; **`given setting name = value`**,
  **`given lookup Kind("code") = "Label"`** (an unmapped code is `given lookup Kind("code") =
  none`) and **`given now = @...`** fix what the run reads.
- **`expect cond`** must be true. **`expect x12 contains "..."`** and **`expect x12 == "..."`**
  compare the transaction a map wrote, from ST to SE, without the ISA, GS, GE and IEA around
  it. **`expect no holds`** (`rejects`,
  `declines`, `warnings`), **`expect held "message"`**, **`expect rejected`**, **`expect
  declined`** and **`expect warned "message"`** check a rules file's outcomes.
- A test in a map file runs that map; in a rules file, its map and then its rules; in a library,
  its functions.

Flow tests drive a flow with a script:

```lading
test "a rate change after acceptance is held" {
  receive file "samples/original.x12"
  respond accept
  receive file "samples/change-rate.x12"
  expect held "ACME changed the rate"
  expect sent 997
}

test "an unanswered tender escalates" {
  receive file "samples/original.x12"
  advance 2 h
  expect escalated "ACME tender L-1001 needs an answer"
}
```

`receive file|text`, `respond accept|decline "reason"`, `event Kind { fields }`, `ack accepted|
rejected` and `advance duration` are steps. Every message a test sends gets the next control
number from `0001`, as the partner would see it, so an escalation's `{message.control}` reads as
it will in production; and the load moves with the events a test raises (`Dispatched` makes it
dispatched, `Delivered` delivered, `InvoicePosted` invoiced), so a later step meets the state a
`cancel unless dispatched` checks; `expect sent <set> [accept|decline]`, `expect held`,
`expect escalated`, `expect created`, `expect updated`, `expect cancelled` and `expect no
<action>` check what the flow did.

**Properties** check a rule over many inputs:

```lading
property "everything ACME has sent still reads" {
  vary file in samples "samples/**/*.x12"
  expect no rejects
}

property "every decision writes a reservation code" {
  vary decision in Decision
  given tenderResponse { key: "P-1", decision: decision, respondedAt: @2026-01-10T14:30Z }
  expect x12 contains "B1*"
}
```

`vary name in samples "glob"`, `in [values]`, `in a..b` (24 values: both ends, landmarks and
evenly spread points, as Tariff) or over an enum; at most 256 cases. A failing case is shrunk to
the simplest one, which the tests panel shows as the counterexample.

**Coverage.** Every test records which assignments, segment blocks, rules and flow arms ran; the
studio marks the lines no test reached. **Unread data**: every element present in the project's
samples that no map reads is listed ("N9*PO appears in 37 of 40 samples and nothing reads it"),
with a fix that adds an assignment.

## Lineage

Every value a map produces remembers where it came from: the X12 elements (or CSV cells, JSON
paths, document fields) it was computed from, and the functions, crosswalks and rule fixes it
passed through. The studio shows it on hover (`stops[1].address.city ← 0300[2].0310.N401
"DALLAS" · titleCase`), the message view shows it for every received message, and a rule's
`reject at` uses it to name the element in the 997.

## Versions, replay and shadow

A project version is **Draft**, **Certified**, **Active**, **Shadow**, **Superseded** or
**Archived**. One version is active per project.

- A draft saves at any time; it can't be certified while any file has an error or any test
  fails.
- **Certify** checks readiness: no errors, every test and property passing, every flow arm
  reached by a test, every element the partner's guide makes mandatory read or written, and a
  replay with no new rejections. A certified version may switch `usage` to `production`.
- **Replay** runs a version against the partner's real messages from a period (inbound) or the
  documents Trenova sent (outbound) and compares it with the active version: how many read the
  same, which fields differ and how, what is newly held, rejected or declined.
- **Shadow** runs a certified version beside the active one on live traffic for a set time,
  sending nothing, and records where the two disagree.
- **Activate** makes a certified version active; the previous one is superseded. Messages record
  the version that read or wrote them.

## The editor

The studio (`/edi/lading/:partnerId`, `client/apps/web/src/routes/edi/lading/`) edits one
partner's project: the version picker and its actions above, the file tree, one editor per file
(`components/lading-editor/`), and the panels below. The file tree arranges the project by what uses what (`projectMap`): the partner file, each
flow with the maps and rules its clauses name (from `lading.check`'s flows), the shared guides
and libraries, the files no flow reaches, and the samples. Its header says whether the project
checks clean, each file shows its error and warning counts, a map's set and direction and the map
a rules file checks, Mod-P filters by path, and rows rename in place (a sample's rename rewrites
the quoted paths tests read it by). Adding opens a palette of every kind of file and sample (type
to filter, or a kind's letter), then that kind's form; a source file's options (a map's
direction, input, document and guide; the map rules check; the base a guide narrows) shape a
starter it previews, and the partner file is the module's own starter's (`lading.starter`).
A partner without a project gets the start form, which writes the starter for the chosen role
and sets as the first draft. Everything about the text is computed by the module in the browser,
against the organisation and custom fields from `GET /edi/lading/context/`:

- **the editor:** colour from `lading.tokens`; diagnostics with their code, the places they
  relate to and quick fixes from `lading.check` of the whole working copy, with this file as the
  editor holds it; completion (X12 paths from the guide with each segment's and element's name,
  usage and codes; document fields; settings; functions and methods; crosswalks; enum variants;
  events; and, inside the quotes of a text compared with a code element, that element's codes
  with their descriptions), asked about the text being typed and asked again after a `.` or a
  quote; hover (the guide's description, the declaration in the syntax colours, the type, the doc,
  and the last run's value with the elements it was read from and the steps it went through);
  signature help; inlay values from the last run; go to definition (F12, Mod-click, into other
  files); Find references (Shift-F12); rename (F2); the line's quick fixes and the refactorings
  (Mod-.: *Extract crosswalk*, *Extract function*, *Promote to setting*, *Move to library*),
  whose edits may reach other files and land in the working copy; F8 and Shift-F8 between
  problems and Mod-Shift-M for all of them; folding from the outline; formatting (Shift-Alt-F);
  Mod-S (`:w` in Vim) saves the draft. The status bar under it shows the file's errors and
  warnings, the cursor's line and column, Format and the Vim switch; with Vim keys on, the
  Tariff editor's mappings (`gd`, `K`, `gl`, `]d`, `grn`, `grr`, `gra`, Space as the leader) run
  Lading's own commands;
- **problems:** every finding in the project, errors first, each opening its file at the range;
- **tests:** tests and properties run on the working copy without saving, with readiness,
  failures that open their range, a property's counterexample, the elements the samples carry
  that no map reads, and the lines no test reached marked in the editor;
- **run:** one map on one sample, either here (the working copy, no lookups answered) or with
  live data (the saved version on the server, lookups answered from the organisation's records);
  the transactions, findings, outcomes, the document, the written X12, the lookups asked for and
  the unmapped codes; its values feed hover and inlays;
- **release:** the steps to live traffic (saved, replayed, certified, shadow, live) worked out
  from the version, the working copy and its replays (`releaseStages`), then the replays of each
  inbound map against the active version, or the shadow's messages, as a table with the
  selected row's differences beside it;
- **settings:** the partner settings the partner file declares, changed on the active version
  without a new version;
- **notes** for the version.

A draft is the only version that changes: the studio holds a working copy, saves it in one
write (with the version number, so a concurrent save is refused), blocks leaving the page while
it is unsaved, and offers certify only once the copy is saved and has no errors. Every other
version opens read-only, and *New draft from this version* copies it.

## Module operations

All are `lading.*` operations of `edi.wasm`, with the module's JSON envelope, `Message`s and
UTF-16 ranges:

| Op | Does |
|---|---|
| `lading.check` | every file's diagnostics, outline and symbols, the project's settings, maps, flows and readiness |
| `lading.format`, `lading.tokens` | one file |
| `lading.complete`, `lading.hover`, `lading.signature`, `lading.definition`, `lading.references`, `lading.rename`, `lading.inlays`, `lading.actions` | the editor at a position |
| `lading.run` | one map (and its rules) on one input: the output, outcomes, findings, unmapped codes, the lookups it still needs, and lineage |
| `lading.test` | every test and property, with coverage and unread data |
| `lading.flowStep` | one flow event: the new state and the actions |
| `lading.replay` | two versions of a map on a batch of inputs, compared |
| `lading.guide` | a guide compiled to the X12 guide shape |
| `lading.starter` | the starter project for a partner and its sets |
| `lading.catalog` | documents, records, enums, events, lookups and the standard library, with docs |

## Limits

A run has a budget of evaluation steps (10 million) and output size (16 MiB of X12); a flow step
1 million. Exceeding either is an error at the call, never a partial result. A project holds at
most 200 files and 2,000 samples, and a test run 256 cases per property.

## Appendix: grammar

```text
file        = header { decl | test | property }
header      = partnerHdr | guideHdr | mapHdr | rulesHdr | flowHdr | libraryHdr

partnerHdr  = "partner" name string
partnerDecl = "interchange" "{" { ("them"|"us") X12REF string | "group" string "<->" string
              | "version" NUMBER | "separators" string string string string
              | "usage" ("test"|"production") } "}"
            | "zone" zoneName | "currency" string
            | "setting" name ":" type "=" expr [ "in" expr ".." expr ]

guideHdr    = "guide" dotted "from" dotted [string]
guideDecl   = path ( "required" | "unused" | "max" INT | "min" INT
              | "syntax" X12REF {X12REF} | "{" elemRule { (";"|NL) elemRule } "}" )
            | "loop" LOOPID { "required" | "unused" | "min" INT | "max" INT }
            | "severity" kebab ("error"|"warning"|"info")
elemRule    = POS { "required" | "unused" | "codes" string {"," string}
              | "length" INT ".." INT | "repeat" INT | "fixed" string } [string]

mapHdr      = "map" name "from" source "to" target uses
source      = dotted | docType | "csv" "{" { name ":" type [string] } "}" | "json"
assign      = lvalue "<-" expr [ "else" (outcome | expr) ]
lvalue      = name { "." name | "[" expr "]" }
each        = lvalue "<-" "each" expr "as" name block
segBlock    = X12REF [ "when" expr ] "{" { POS "<-" expr [","] } "}"
loopBlock   = LOOPID ( block | "<-" "each" expr "as" name block )
when        = "when" expr block
let         = "let" name "=" expr

uses        = [ "use" name {"," name} ]
rulesHdr    = "rules" name "for" name uses
rule        = "rule" string "{" { ruleStmt } "}"
ruleStmt    = "require" expr "else" outcome | outcome "when" expr
            | "for" name "in" expr [ "where" expr ] block | "fix" lvalue "<-" expr | let
outcome     = "reject" [ "at" expr ] string | "decline" string [ "code" string ]
            | "hold" string | "warn" string

flowHdr     = "flow" name "with" name uses
flowDecl    = "receive" SET "as" name { receiveClause }
            | "respond" SET "via" name { respondClause }
            | "send" SET "via" name "on" EVENT {"," EVENT} [ "where" expr ] [ "key" expr ]
              [ "batch" "within" expr ] [ "until" EVENT ]
            | "expect" SET "for" "every" "send" "within" expr "else"
              [ "resend" INT "then" ] "escalate" string

libraryHdr  = "library" name uses
decl        = fnDecl | crosswalk | constDecl
fnDecl      = "fn" name "(" [ param {"," param} ] ")" "->" type ( "=" expr | block )
crosswalk   = "crosswalk" name ":" type "<->" type "{" { pair } [ "else" ("->"|"<-") expr ] "}"
pair        = expr {"," expr} ("<->"|"->"|"<-") expr {"," expr}
constDecl   = "const" name "=" expr

expr        = literal | name | path | expr "." name [ "(" args ")" ] | expr "[" expr "]"
            | name "(" args ")" | "lookup" TYPE "(" expr ")" [ "learn" expr ]
            | expr "via" name | expr "??" expr | "exists" expr
            | "match" expr "{" { pattern "=>" expr } [ "else" "=>" expr ] "}"
            | "if" expr "then" expr "else" expr | name "=>" expr
            | expr binop expr | ("not"|"-") expr | "[" [ expr {"," expr} ] "]"
            | "[" ":" "]" | "[" expr ":" expr {"," expr ":" expr} "]"
            | TYPE "{" { name "<-" expr } "}" | "{" { name ":" expr [","] } "}"
            | "each" expr "as" name block
```
