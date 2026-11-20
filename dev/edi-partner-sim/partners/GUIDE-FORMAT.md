# Partner guide format

A partner's `guide/<set>.yaml` is the partner's own implementation guide for one transaction set
it **receives** from Trenova. The simulator's oracle (`internal/oracle`) reads it and judges
every transaction Trenova sends that partner. It is written independently of the `edi` crate's
guides on purpose: the crate must not judge its own output.

```yaml
set: "214"                 # transaction set ID (ST01)
version: "004010"          # GS08 prefix it applies to (004010, 005010X…)
group: QM                  # functional identifier code (GS01)
title: "Shipment Status Guide, v3.2"

segments:                  # every segment the structure uses, ST and SE included
  ST:
    name: Transaction Set Header
    elements:
      - { pos: 1, name: Transaction Set Identifier Code, type: ID, min: 3, max: 3, usage: M, fixed: "214" }
      - { pos: 2, name: Transaction Set Control Number, type: AN, min: 4, max: 9, usage: M }
  B10:
    name: Beginning Segment for Transportation Carrier Shipment Status Message
    elements:              # in position order; positions not listed are "not used"
      - pos: 1
        name: Reference Identification
        type: AN           # AN ID DT TM N0..N9 R
        min: 1
        max: 30
        usage: M           # M mandatory, O optional, X conditional, N not used (present = finding)
      - { pos: 2, name: Shipment Identification Number, type: AN, min: 1, max: 30, usage: M }
      - pos: 3
        name: Standard Carrier Alpha Code
        type: ID
        min: 2
        max: 4
        usage: M
        codes: [TRNV, TRNX] # allowed values; omitted = any value of the type
        fixed: null        # a value the element must equal
        repeat: 1          # repetitions allowed (repetition separator), default 1
  L11:
    name: Business Instructions and Reference Number
    elements:
      - { pos: 1, name: Reference Identification, type: AN, min: 1, max: 30, usage: M }
      - { pos: 2, name: Reference Identification Qualifier, type: ID, min: 2, max: 3, usage: M, codes: [BM, PO, QN] }
  LX:
    name: Assigned Number
    elements:
      - { pos: 1, name: Assigned Number, type: N0, min: 1, max: 6, usage: M }
  AT7:
    name: Shipment Status Details
    elements:
      - { pos: 1, name: Shipment Status Code, type: ID, min: 2, max: 2, usage: X, codes: [X3, AF, X1, D1] }
      - { pos: 2, name: Shipment Status or Appointment Reason Code, type: ID, min: 2, max: 2, usage: X }
      - { pos: 3, name: Shipment Appointment Status Code, type: ID, min: 2, max: 2, usage: X, codes: [AA, AB] }
      - { pos: 4, name: Shipment Status or Appointment Reason Code, type: ID, min: 2, max: 2, usage: X }
      - { pos: 5, name: Date, type: DT, min: 8, max: 8, usage: M }
      - { pos: 6, name: Time, type: TM, min: 4, max: 8, usage: M }
      - { pos: 7, name: Time Code, type: ID, min: 2, max: 2, usage: O, codes: [CT, ET, MT, PT, UT] }
    syntax:                # X12 syntax notes over this segment's positions
      - P0102              # paired: 01 and 02 both present or both absent
      - P0304
      - R0103              # required: at least one of 01, 03
      - E0103              # exclusion: not more than one of 01, 03
      - C0706              # conditional: if 07 then 06
  MS1:
    name: Equipment, Shipment, or Real Property Location
    elements:
      - { pos: 1, name: City Name, type: AN, min: 2, max: 30, usage: X }
      - { pos: 2, name: State or Province Code, type: ID, min: 2, max: 2, usage: X }
      - { pos: 3, name: Country Code, type: ID, min: 2, max: 3, usage: O }
    syntax:
      - L010203            # list conditional: if 01 then at least one of 02, 03
  SE:
    name: Transaction Set Trailer
    elements:
      - { pos: 1, name: Number of Included Segments, type: N0, min: 1, max: 10, usage: M }
      - { pos: 2, name: Transaction Set Control Number, type: AN, min: 4, max: 9, usage: M }

structure:                 # the transaction, in order
  - { seg: ST, usage: M }
  - { seg: B10, usage: M }
  - seg: L11               # the heading's references: the BOL and the PO
    usage: O
    max: 10
    elements:              # narrows L11 for this use only (see "Per-use overrides")
      - { pos: 2, codes: [BM, PO] }
  - loop: "0205"           # loop ID; its first entry is the trigger segment
    usage: O
    max: 999               # loop repeats
    body:
      - { seg: LX, usage: M }
      - loop: "0210"
        usage: M
        max: 10
        body:
          - { seg: AT7, usage: M }
          - { seg: MS1, usage: O }
      - seg: L11           # the status's stop: only a stop number
        usage: O
        elements:
          - { pos: 2, codes: [QN] }
  - { seg: SE, usage: M }
```

A composite element lists its `components` instead of a type; each component takes the same keys
as an element (no `components` or `repeat` of its own):

```yaml
  SV1:
    elements:
      - pos: 1
        name: Composite Medical Procedure Identifier
        usage: M
        components:
          - { pos: 1, name: Product/Service ID Qualifier, type: ID, min: 2, max: 2, usage: M, codes: [HC] }
          - { pos: 2, name: Product/Service ID, type: AN, min: 1, max: 48, usage: M }
```

## Per-use overrides

A segment is defined once under `segments`, but a guide often narrows it differently where it is
used: a 214's heading L11 takes `BM` and `PO`, the L11 inside loop 0205 only `QN`. A segment use
in `structure` may carry `elements:`, each naming a `pos` the segment lists and any of `usage`,
`min`, `max`, `codes`, `fixed` and `repeat`. The override applies to that use only, and it may
only narrow:

- `usage` may tighten (`O` to `M` or `N`) but never relax a mandatory element;
- `min` and `max` stay within the segment's own;
- every code is one of the segment's codes (when it lists any), and `fixed` agrees with a fixed
  value the segment already has;
- `repeat` is at most the segment's;
- a composite takes only `usage` and `repeat`: narrow its components in the segment.

The guide check rejects an override of a position the segment does not list, a position
overridden twice, and `elements:` on a loop.

## Rules

- A segment use's `max` is its max use within one loop iteration (default 1); a loop's `max`
  is its repeat count (default 1).
- Dates (`DT`) are CCYYMMDD or YYMMDD by `max`; times (`TM`) HHMM, HHMMSS or HHMMSSDD by length.
- `N0`..`Nn` are implied-decimal integers (an explicit point is a finding); `R` is a decimal.
- Findings use the X12 codes: segment AK304 (1 unrecognized, 2 unexpected, 3 mandatory missing,
  4 loop over max, 5 over max use, 6 not in set, 7 out of sequence) and element AK403
  (1 mandatory missing, 2 conditional required missing, 3 too many elements, 4 too short,
  5 too long, 6 invalid character, 7 invalid code, 8 invalid date, 9 invalid time,
  10 exclusion violated, 12 too many repetitions, 13 too many components), plus `I10` for a
  not-used element that is present.
