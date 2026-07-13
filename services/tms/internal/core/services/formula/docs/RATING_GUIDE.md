<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Rating Guide — Formula Templates & Rate Tables

A practical, user-facing guide to how Trenova turns the facts of a shipment into a charge —
the formulas you write, the lookup tables they draw from, and the controls that keep a bad
formula from ever reaching an invoice.

- [01 · How rating works](#01--how-rating-works)
- [02 · Shipment variables](#02--shipment-variables)
- [03 · Writing expressions](#03--writing-expressions)
- [04 · Rate tables](#04--rate-tables)
- [05 · Rate breakdown](#05--rate-breakdown)
- [06 · Guardrails](#06--guardrails)
- [07 · Approval workflow](#07--approval-workflow)
- [08 · Scheduled versions](#08--scheduled-versions)
- [09 · Backtesting](#09--backtesting)
- [10 · Testing on real data](#10--testing-on-real-data)
- [11 · Worked examples](#11--worked-examples)
- [12 · Quick reference](#12--quick-reference)

---

## 01 · How rating works

Everything in this guide is built from four pieces. Learn these and the rest of the engine
falls into place.

| Piece | What it is |
|---|---|
| **Formula template** | A saved, reusable pricing rule. A shipment goes in, a dollar amount comes out. |
| **Expression** | The math inside a template — the actual formula, written in a small, safe language. |
| **Variables** | Facts about the shipment your expression can use: distance, weight, stop count, and more. |
| **Rate table** | A tenant-owned lookup table of numbers — fuel surcharges, lane rates, weight breaks — that formulas pull from. |

### What happens when a shipment is rated

1. You assign a formula template to the shipment.
2. The engine gathers the shipment's variables (distance, weight, hazmat status, and so on) and any rate tables your formula references.
3. It picks the correct version of the template for that shipment's ship date (see [Scheduled versions](#08--scheduled-versions)).
4. It evaluates the expression, produces the freight charge, and clamps it to your [guardrails](#06--guardrails) if you set any.
5. The number — plus a full [breakdown](#05--rate-breakdown) of how it was reached — is saved on the shipment.

> **Good to know.** Only a template in the **Active** state can rate a shipment. Drafts and
> templates under review never touch a live charge — that is the whole point of the
> [approval workflow](#07--approval-workflow).

---

## 02 · Shipment variables

When a formula runs against a shipment, these values are available by name. Most are computed
for you by rolling up the shipment's moves, stops, and commodities — you don't have to add them
up yourself.

### Distance, stops & dimensions

| Variable | Type | What it means |
|---|---|---|
| `totalDistance` | number | Total miles across every move on the shipment. |
| `totalStops` | number | Count of stops across all moves. |
| `totalWeight` | number | Total weight (uses the shipment weight, or sums the commodities). |
| `totalPieces` | number | Total piece count. |
| `totalLinearFeet` | number | Linear feet consumed, from commodity dimensions. |
| `weight` / `pieces` | number | The shipment-level weight and piece values as entered. |

### Temperature & hazmat

| Variable | Type | What it means |
|---|---|---|
| `hasHazmat` | true / false | Whether any commodity is hazardous. |
| `requiresTemperatureControl` | true / false | Whether a temperature range is set. |
| `temperatureMin` / `temperatureMax` | number | The requested temperature range, in degrees. |
| `temperatureDifferential` | number | Max minus min — the width of the temperature window. |

### Charges & identifiers

| Variable | Type | What it means |
|---|---|---|
| `baseRate` | number | The base rate recorded on the shipment. |
| `freightChargeAmount` | number | The current freight charge, if already set. |
| `otherChargeAmount` | number | Accessorial / other charges total. |
| `currentTotalCharge` | number | The shipment's current total charge. |
| `proNumber` | text | The PRO number — useful in conditions, not math. |

> **Custom variables.** Need something the shipment doesn't carry — a national diesel price, a
> lane code, a customer discount tier? Define it as a **variable** on the template, give it a
> default, and supply the real value at rating time. Custom variables are how outside data (like
> today's fuel price) gets into a formula.

---

## 03 · Writing expressions

Expressions read like arithmetic. You combine variables, numbers, operators, and a set of
built-in functions. **The result must be a number** — that number becomes the charge.

### Operators

- **Math:** `+` `-` `*` `/` `%` (modulo) and `**` (power).
- **Compare:** `>` `>=` `<` `<=` `==` `!=`.
- **Logic:** `&&` (and), `||` (or), `!` (not).
- **Choose a value:** `condition ? whenTrue : whenFalse` — the ternary. This is how you branch.

```js
// Per-mile rate with a hazmat uplift — 25% more when the load carries hazmat
totalDistance * 2.15 * (hasHazmat ? 1.25 : 1.0)
// 420 miles, hazmat → $1,128.75
```

### Built-in functions

| Function | Does |
|---|---|
| `round(x)` / `round(x, places)` | Rounds to a whole number, or to `places` decimals (−12 to 12). |
| `ceil(x)` · `floor(x)` | Round up / round down to a whole number. |
| `abs(x)` | Absolute value. |
| `min(a, b)` · `max(a, b)` | The smaller / larger of two numbers. |
| `clamp(x, lo, hi)` | Forces `x` to stay between `lo` and `hi`. |
| `sum(...)` · `avg(...)` | Total / average of any number of values. |
| `pow(base, exp)` · `sqrt(x)` | Power and square root. `sqrt` rejects negatives. |
| `coalesce(a, b, ...)` | The first value that isn't empty. `coalesce(discountRate, 0)` falls back to 0. |
| `lookup(table, key)` / `lookupOr(table, key, default)` | Read a value from a [rate table](#04--rate-tables). Covered next. |

```js
// Weight-based with a floor and a ceiling:
// $0.14 per pound, but never below $95 or above $4,000
clamp(totalWeight * 0.14, 95, 4000)
// 12,500 lb → $1,750.00
```

> **The result must be a number.** A formula that produces text, or that could divide by zero and
> yield an infinite value, is rejected — at save time when we can catch it, and safely at rating
> time if the data causes it. Reach for `coalesce` and `clamp` to keep results well-behaved.

---

## 04 · Rate tables

Some pricing is really a *table*, not a formula: fuel surcharge bands, per-lane flat rates,
weight breaks. Cramming those into an expression means a tower of nested `? :` conditions that
only the author can read and no one dares edit.

A **rate table** moves that data out of the formula and into a table an ops person can maintain.
Your formula just asks it a question with `lookup`. Manage them under
**Billing → Configuration Files → Rate Tables**.

### Two kinds of table

- **Exact** — matches a **text key** to a value. Best for lane rates, customer tiers, accessorial
  codes — anything with named buckets.
- **Range** — matches a **number** to the band it falls in. Best for fuel scales and weight breaks.
  Each band's low end is **included**, its high end is **excluded**; the last band can be open-ended.

### A Range table: fuel surcharge

Keyed by the national average diesel price. A price of exactly `3.50` lands in the third band.

**Rate table · key `fuel_surcharge` · type Range**

| From (incl.) | To (excl.) | Surcharge |
|---:|---:|---:|
| 0.00 | 3.00 | 0.00 |
| 3.00 | 3.50 | 0.12 |
| 3.50 | 4.00 | 0.18 |
| 4.00 | — | 0.25 |

### An Exact table: lane rates

**Rate table · key `lane_rate` · type Exact**

| Match key | Rate |
|---|---:|
| `ATL-MIA` | 1,450.00 |
| `ATL-JAX` | 980.00 |

### Reading a table from a formula

Call `lookup("table_key", value)`. For a Range table the value is a number; for an Exact table
it's the text key. If a lane might be missing from the table, use `lookupOr` to supply a fallback
instead of erroring.

```js
// Use the contracted lane rate if we have one, otherwise price it at $2.10 / mile
lookupOr("lane_rate", laneCode, totalDistance * 2.10)
// laneCode "ATL-MIA" → $1,450.00
```

> **How keys and safety work.** A table's **key** (like `fuel_surcharge`) is the name formulas use
> — it starts with a letter and contains only letters, digits, and underscores. When you save a
> template, Trenova checks that every table you reference actually exists in your organization, so
> a typo is caught immediately. `lookup` and `lookupOr` are reserved words — you can't name a
> variable after them.

---

## 05 · Rate breakdown

"Why was this load $1,432?" is a question auditors and customers ask constantly. A single total
can't answer it. **Breakdown definitions** let you name the parts of a charge and have each one
calculated and saved alongside the total.

Each breakdown item is just a name, a friendly label, and its own expression — evaluated against
the same shipment. Add up to 20. They're recorded on the shipment's rating detail, so the
composition travels with the load.

**Breakdown definitions on a freight template**

| Name | Label | Expression |
|---|---|---|
| `base` | Line haul | `totalDistance * ratePerMile` |
| `fuelSurcharge` | Fuel surcharge | `base * lookup("fuel_surcharge", fuelPrice)` |
| `hazmatFee` | Hazmat handling | `hasHazmat ? 75 : 0` |

On a rated shipment, that produces a line-itemized trace like this:

| Component | Amount |
|---|---:|
| Line haul | 903.00 |
| Fuel surcharge | 162.54 |
| Hazmat handling | 75.00 |
| **Total** | **1,140.54** |

> **Robust by design.** A breakdown component that fails to calculate never breaks the rating —
> the error is recorded against that one line, and the main charge still computes. Breakdowns
> explain the total; they don't gate it.

---

## 06 · Guardrails

Set an optional **minimum** and **maximum charge** on a template. Whatever the expression
produces, the final charge is clamped into that range. If a clamp happens, the rating detail
records the raw result and which bound was hit — so nothing is silently swallowed.

- **Minimum charge** — protects against a formula (or a tiny shipment) producing $0 or a few
  dollars that wouldn't cover cost.
- **Maximum charge** — catches a runaway result (a bad multiplier or a divide-by-near-zero) before
  it becomes a $400,000 line item.

```js
// With min $150 and max $10,000 set on the template.
// Expression yields $92 for a very short move…
totalDistance * 2.10
// raw $92.00 → clamped to minimum $150.00
```

Guardrails are limits, not pricing. Keep them wide — they're a safety net for mistakes, not a
substitute for getting the formula right.

---

## 07 · Approval workflow

Templates that set prices deserve a review step. Every template has a status, and moving between
them is permissioned — separate *submit*, *approve*, and *reject* rights let you require a second
set of eyes.

```
Draft  ──submit──▶  In Review  ──approve──▶  Active  ⇄  Inactive
                        │
                        └──reject──▶  Draft
```

- **Submit** moves a Draft into In Review, with a note for the reviewer.
- **Approve** promotes it to Active — the only state that can rate a shipment.
- **Reject** sends it back to Draft with a required comment explaining why.
- **Active ⇄ Inactive** lets you retire or restore a template without re-approval.

> **Material edits reset approval.** If you change a pricing-relevant field — the expression,
> variables, breakdowns, guardrails, schema, or type — on a template that's Active or In Review,
> it drops back to **Draft** and must be re-approved. You can't quietly edit a formula out from
> under an approval.

---

## 08 · Scheduled versions

Every save creates a version. You can give a version an **effective date** — the moment it
becomes the one used for rating — instead of flipping templates by hand at midnight on
January 1st.

Rating then chooses the right version by the **shipment's ship date** (falling back to when the
shipment was created). A load that ships in December is priced with December's rates even if you
rate it in January, and a load that ships after the increase gets the new rates automatically.

1. Open a template's version history and pick the version to schedule.
2. Set its activation date — it must be in the future, and the template must be Active.
3. From that date on, matching shipments rate against that version. Clear the schedule anytime.

> **Why ship date, not today.** Pricing by ship date keeps historical loads reproducible.
> Re-rating an old shipment always gives the rate that was in force when it moved — not whatever
> is current.

---

## 09 · Backtesting

The scariest failure mode in rating is a change that silently doubles — or halves — your prices.
**Backtesting** runs a candidate formula against your recently rated shipments (up to 500) and
shows exactly how each total would move, plus the aggregate impact.

The candidate can be an edited expression you're working on, or an earlier version you're
considering rolling back to. Every shipment is priced both ways and compared.

**Backtest · candidate vs. current · 4 of 312 shown**

| PRO | Current | Candidate | Change |
|---|---:|---:|---:|
| P100482 | 1,140.54 | 1,201.20 | +5.3% |
| P100489 | 2,310.00 | 2,310.00 | 0.0% |
| P100501 | 640.00 | 705.60 | +10.3% |
| P100510 | 3,980.00 | 3,712.00 | −6.7% |

The summary rolls this up: how many shipments changed, how many went up versus down, the current
and candidate totals, and the largest single increase and decrease. A per-shipment error (bad
data on one old load) is reported on its row and never stops the run.

---

## 10 · Testing on real data

The expression tester normally runs on sample values you type. It can also run against a
**real shipment by number** — building the exact same environment production rating would, so
what you see in the tester is what you'd get on the invoice.

- Toggle **Use real shipment** and pick the shipment.
- The tester resolves its true variables — distance, weight, hazmat, everything — no guessing.
- You get the result, the resolved variables, and any breakdown components, all from live data.

> **Permission-aware.** Testing against a shipment requires permission to read that shipment — the
> tester can't be used to peek at data you otherwise couldn't see.

---

## 11 · Worked examples

### 1 · Simple per-mile with a floor

A dependable starting point: rate by the mile, never go below a minimum.

```js
// Set Minimum charge = 250 on the template
totalDistance * ratePerMile
// 420 mi × $2.15, min $250 → $903.00
```

### 2 · Line haul + fuel surcharge + hazmat, itemized

Combines a rate table, a custom `fuelPrice` variable, and breakdown components so the invoice
explains itself.

```js
totalDistance * ratePerMile
  * (1 + lookup("fuel_surcharge", fuelPrice))
  + (hasHazmat ? 75 : 0)
// 420 mi × $2.15, diesel $3.85, hazmat → $1,140.54
```

Pair it with the breakdown definitions from [section 05](#05--rate-breakdown) to split that total
into line haul, fuel, and hazmat.

### 3 · Contracted lanes, with a per-mile safety net

Honor negotiated lane rates where they exist, and never fail on a lane that isn't in the table.

```js
// Set a Maximum charge to catch outliers
max(
  200,
  lookupOr("lane_rate", laneCode, totalDistance * 2.10)
)
// Unlisted lane, 300 mi → $630.00
```

---

## 12 · Quick reference

### Functions

| Call | Returns |
|---|---|
| `round(x[, places])` | Nearest whole number, or to *places* decimals. |
| `ceil(x)` · `floor(x)` · `abs(x)` | Round up · round down · absolute value. |
| `min(a,b)` · `max(a,b)` | Smaller · larger of two numbers. |
| `clamp(x, lo, hi)` | Keep *x* within [lo, hi]. |
| `sum(...)` · `avg(...)` | Total · average of the arguments. |
| `pow(base, exp)` · `sqrt(x)` | Power · square root. |
| `coalesce(...)` | First non-empty value. |
| `lookup(table, key)` | Value from a rate table (errors if missing). |
| `lookupOr(table, key, default)` | Value from a rate table, or the default. |

### Most-used variables

| Variable | Is |
|---|---|
| `totalDistance` | Total miles |
| `totalWeight` · `totalPieces` | Weight · piece count |
| `totalStops` | Stop count |
| `hasHazmat` | true / false |
| `requiresTemperatureControl` · `temperatureDifferential` | Reefer flags |
| `baseRate` · `freightChargeAmount` · `otherChargeAmount` | Existing charge fields |

### Rules of thumb

- A formula must produce a **number**. Branch with `condition ? a : b`.
- Move tiered pricing into a **rate table** instead of stacking `? :`.
- Set a **minimum charge** as a floor; keep the maximum as a safety catch.
- Only **Active** templates rate shipments; material edits reset approval.
- **Backtest** before you activate a change; **test on a real shipment** while you write it.
