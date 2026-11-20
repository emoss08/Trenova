# Partner format

A trading partner the simulator plays lives in `partners/<name>/`:

| File | Holds |
|---|---|
| `partner.yaml` | who the partner is and how it trades (below) |
| `generate/<set>.yaml` | how the partner writes a set it **sends**: a load profile and segment templates (below) |
| `guide/<set>.yaml` | the partner's guide for a set it **receives** from Trenova, read by the oracle ([GUIDE-FORMAT.md](GUIDE-FORMAT.md)) |
| `samples.yaml` | the seeds behind `lading/samples/*.x12` (below) |
| `lading/` | the Lading project Trenova runs for this partner: the thing under test |

`internal/partner` loads a partner with `partner.Open(dir)` (or every partner with
`partner.OpenAll("partners")`). Unknown keys are errors, and every problem names its file and the
path inside it: `partners/manufacturer/partner.yaml: acknowledgment.sends: "998" is not one of
997, 999, none`.

## partner.yaml

```yaml
name: manufacturer            # must equal the folder name
display: Kestrel Valve Works  # invented; never a real company's name or guide text
role: shipper                 # shipper: tenders to Trenova (Trenova is the carrier)
                              # carrier: Trenova tenders to it (Trenova is the broker)
zone: America/Chicago         # IANA zone the partner writes times in

them:                         # the partner's envelope identity
  qualifier: ZZ               # ISA05 (and ISA07 on what it receives)
  id: KESTRELVW               # ISA06, at most 15 characters
  group: KESTRELVW            # GS02
  scac: ""                    # required when role is carrier
trenova:                      # Trenova's identity as this partner knows it
  qualifier: ZZ
  id: TRENOVA
  group: TRENOVA
  scac: TRNV                  # required when role is shipper; templates use it as {scac}

envelope:                     # how the partner writes an interchange
  version: "004010"           # GS08; ISA12 is its first five characters
  versions:                   # optional: a set written under another version, such as a
    "999": 005010X231A1       # 5010 999 from a 4010 partner. Its groups get that GS08 and ST03,
                              # and never share an interchange with another ISA12
  separators:                 # one ASCII character each, all different, never a letter,
    element: "*"              # digit or space
    segment: "~"
    component: ">"            # ISA16
    repetition: "^"           # ISA11; leave empty to write U (required once any version is
                              # 00402 or later)
  terminator: lf              # none | lf | crlf: written after every segment terminator
  squeezedIsa: false          # true drops the ISA's fixed-width padding
  usage: T                    # ISA15: T or P

transport:                    # exactly one of as2 and sftp
  as2:
    id: KESTRELVW             # the partner's AS2 identifier
    signed: true
    encrypted: true
    compressed: false
    mdn: sync                 # sync | async | none
  # sftp:
  #   fileName: "{set}_{control}.edi"   # fields: set (the first ST01), control (ISA13), group
  #                                     # (the first GS06), sender, receiver, date (CCYYMMDD) and
  #                                     # time (HHMMSS) of the interchange; {control} is required.
  #                                     # Partner.FileName(written) names a written interchange
  #   inbound: /inbound       # where the partner drops files for Trenova
  #   outbound: /outbound     # where Trenova leaves files for the partner

batching:                     # how the partner packs what it sends
  transactionsPerGroup: 1     # STs per GS (one GS per functional group ID)
  groupsPerInterchange: 1     # GSs per ISA

acknowledgment:
  sends: "997"                # what it answers Trenova's transactions with: 997 | 999 | none
  expects: "997"              # what it wants back for its own: 997 | 999 | none
  ta1: false                  # ISA14 = 1 on what it sends (it asks for a TA1)
  delay: 45s                  # how long it waits before acknowledging
  reject:
    when: errors              # oracle findings that turn AK5 to R: never | errors | any
    sets: []                  # sets it always rejects, whatever the oracle says
  never: 0                    # probability (0..1) it never acknowledges a transaction

answers:                      # carrier only (required there): how it answers a tender Trenova
  policy: accept              # sends when no script was given: accept | draw (a seeded mix of
                              # declines, rate disputes and stray statuses) | decline | none
  decline: ""                 # the reason, when the policy is decline
  pace: 2s                    # the gap before its 990 and between each later reply

sends: ["204", "997"]         # the sets it sends; every generated set needs generate/<set>.yaml
receives: ["997", "990", "214", "210"]   # the sets it receives; each wants guide/<set>.yaml
```

`edi-sim serve` acts on `acknowledgment.delay` (the acknowledgment waits that long),
`acknowledgment.never` (it is withheld with that probability, seeded) and `acknowledgment.ta1`
(the partner asks for a TA1 on what it sends); it answers a TA1 itself whenever Trenova asks for
one. A carrier answers each accepted original tender by its script, or by `answers`.

`acknowledgment.sends` must be in `sends`, and `acknowledgment.expects` in `receives`. Only the
acknowledgment sets may appear in both lists.

## generate/<set>.yaml

The partner's own way of writing a set: a profile the seeded generator draws a load from, the
purpose code for each variation it supports, and segment templates over the generated load. The
template language is deliberately small; the generator does the work in Go.

```yaml
set: "204"                    # must match the file name; only 204 is generated today

purposes:                     # B2A01 per variation; a kind left out is one the partner never sends
  original: "00"
  duplicate: "00"             # the same tender sent again, under new control numbers
  change: "04"
  replace: "05"
  cancel: "01"

duplicates: new               # new: a duplicate is the tender again under new control numbers,
                              # 30 minutes later; resend: the original's exact bytes, which a
                              # caller writes with the original's control numbers
                              # (Definition.Resends tells it to)

load:                         # the profile
  start: "2026-03-02"         # first tender date; tenders spread over `days` days
  days: 60
  number: { prefix: "KVW-", digits: 6 }   # the load number: prefix + a number unique per seed
  bol: { prefix: "8100", digits: 6 }      # references: prefix + random digits
  po: { prefix: "45", digits: 8, chance: 0.85 }   # chance: how often it is sent (omit = always)
  pickupRef: { prefix: "R", digits: 7, chance: 0.6 }
  appointment: { prefix: "APT", digits: 6, chance: 0.7 }
  payment: [{ code: PP, weight: 9 }, { code: TP, weight: 1 }]   # weighted picks
  stops: { min: 2, max: 4, extraPickup: 0.2 }   # 2..12; extraPickup: a second plant when 3+ stops
  origins:                    # the partner's own plants; city and state must be in the
    - { code: KVW-RFD1, name: ..., street: ..., city: ROCKFORD, state: IL }   # simulator's city table
  billTo: { code: KVW-AP, name: ..., street: ..., city: ROCKFORD, state: IL, zip: "61104" }
  consignees:                 # deliveries go to "<name> <CITY>" in a random table city; the code
    code: KVC-                # is code + name index + city index, so a consignee keeps its code
    names: [MIDLAND FLUID SUPPLY, ...]   # and its address across loads
  equipment:                  # weighted; reefer gives a set-point range in °F
    - { code: TF, weight: 8, lengthFt: 53 }
    - { code: RT, weight: 1, lengthFt: 53, reefer: { min: 40, max: 55 } }
  weight: { min: 6000, max: 44000 }   # pounds for the load, split across pickups and deliveries
  pieces: { min: 6, max: 26 }         # min must cover stops.max
  lines: { min: 1, max: 3 }           # commodity lines per pickup
  commodities:
    - { description: CAST IRON GATE VALVES, code: "88240" }
    - description: CORROSION INHIBITOR
      code: "48580"
      hazmat: { un: UN1760, class: "8", name: CORROSIVE LIQUID NOS }
    - { description: DAIRY, code: "0401", temperature: { min: 33, max: 38 } }   # °F it rides at
  hazmat: 0.2                 # chance a load carries one hazardous line
  temperatureMix: 0.1         # chance a load mixes commodities whose ranges don't overlap; otherwise
                              # every line's range overlaps the first one drawn
  rate: { perMileCents: { min: 215, max: 340 }, minimumCents: 45000, stopOffCents: 7500 }
  window: { leadDays: { min: 0, max: 3 }, pickupHours: { min: 6, max: 14 }, hours: 2, mph: 47, dwellHours: 1 }
  zip4: 0.5                   # chance a delivery's ZIP carries four more digits
  contact: 0.4                # chance a stop names a contact and phone
  notes: { chance: 0.35, load: [...], delivery: [...] }

segments:                     # ST and SE are written by the writer, never here
  - seg: B2
    el: ["", "{scac}", "", "{load.number}", "", "{load.payment}"]
  - seg: L11
    when: load.po             # written only when the value is present (! negates)
    optional: true            # dropped by a sparse variation
    el: ["{load.po}", PO]
  - each: stops               # repeat the body (or one seg) per item, bound to `as`
    as: stop
    on: [original, change]    # optional: only for these variation kinds
    body:
      - seg: G62
        el: ["{stop.kind|map:pickup=37,delivery=53}", "{stop.window.start|date}", G,
             "{stop.window.start|time}", "{stop.window.start|zone}"]
```

An entry is either a `seg` with `el` (its elements in position order) or a `body` of entries. An
element is a text, or a list of texts for a composite. Text outside braces is literal; `{path}`
reads the view and `|filter` transforms it. Trailing empty elements and components are dropped,
and any separator inside a value is written as a space.

Filters: `date` (CCYYMMDD), `date6` (YYMMDD), `time` (HHMM), `zone` (the X12 time code of the
time's zone: CT, ET, MT, PT), `dollars` (cents as 2145.00), `upper`, `lower`, `trunc:N`, `zip5`,
`zpad:N` (zero-padded, sign kept: -010), `map:a=b,c=d` (an unmapped value is an error).

The 204 view:

| Path | Value |
|---|---|
| `scac`, `purpose`, `kind` | Trenova's SCAC, the variation's purpose code, the variation kind |
| `load.number` `bol` `po` `note` `payment` | text (empty when not drawn) |
| `load.weight` `pieces` `miles` `stopCount` | whole numbers (pounds) |
| `load.rate` `total` | cents (rate per mile, total charge) |
| `load.tenderedAt`, `load.hazmat` | time, yes/no |
| `billTo.*`, `stop.party.*` | `code name street city state zip country contact phone` |
| `equipment.code` `lengthFt` `reefer` `temp` | `temp` is the set point in °F, absent when not a reefer |
| `stops[]` | `sequence kind pickup delivery party window.start window.end weight pieces pickupRef appointment note lines[] temperatures[]` |
| `stop.lines[]` | `number description code weight pieces hazmat.un hazmat.class hazmat.name temperature.min temperature.max` (`temperature` absent without one) |
| `stop.temperatures[]` | `min max` in °F: a pickup's distinct line ranges; a delivery's, the ranges of the lines dealt to it in turn (every delivery gets at least one) |

`Load.TemperatureConflict()` is true when the stops' ranges have no common temperature: what a
single-compartment reefer can't hold.

Opening a partner renders every supported variation (full and sparse) for seed 1, so a template
naming a value that does not exist fails when it loads, naming the entry
(`segments[9].body[3].el[1]: stop.windw is not a field`).

## Variations

A send is a seed plus a variation. The same seed and variation always write the same bytes.

| Field | Effect |
|---|---|
| `kind` | `original`, `duplicate`, `change`, `replace` or `cancel`: picks the purpose code |
| `sparse` | drops every entry marked `optional` |
| `delta.ratePercent` | the total times (100 + n)% |
| `delta.addStop` | one more delivery after the last, with half its freight and a stop-off charge |
| `delta.move` | `{ stop: n, by: 24h }` moves stop n's window (0 is the last stop) |

Each kind is sent at a fixed time after the load was tendered (original 0, duplicate 30 min,
change 3 h, replace 4 h, cancel 6 h), which dates the ISA and GS.

## samples.yaml

```yaml
- file: tender-change.x12     # written to lading/samples/<file>
  set: "204"
  seed: 37
  control: 503                # the ISA and GS control numbers
  variation: { kind: change, delta: { ratePercent: 12 } }
- file: tender-batch.x12
  set: "204"
  seed: 13
  with: [14, 15]              # more seeds in the same interchange, batched as the partner batches
  control: 610
  variation: { kind: original }
```

`Definition.Batch(set, sends, controls)` writes several seeded sends the same way, dated at the
latest send; a batch that the partner's batching splits into several interchanges is an error in
`samples.yaml`.

The checked-in samples are the generator's output: `TestSamplesAreTheGeneratorsOutput` fails when
one differs. After changing a profile or template, rewrite them and the Lading tests that quote
their values:

```bash
go test ./internal/partner -run TestSamples -update
```
