# Scenario format

A scenario is a conversation between one partner and a running Trenova, written as the steps
each side takes and the checks between them. It lives in `partners/<partner>/scenarios/*.yaml`;
`edi-sim run` plays it ([README](../README.md#playing-scenarios-against-trenova)). Unknown keys
are errors, and every problem names the file and the step: `steps[3].expect.wire: "991" is not
one of [204 990 214 210 997 999 TA1]`.

```yaml
name: a rate change after acceptance is held for a planner, then released
description: optional prose shown in the report
within: 3m                  # how long a step waits by default (3m when left out)
steps:
  - send: { set: "204", as: t1 }
  - expect: { ack: t1 }
  - trenova: { decide: t1, decision: accept }
  - expect: { wire: "990", about: t1, elements: { B1-04: A } }
  - send: { set: "204", variation: change, of: t1, delta: { ratePercent: 12 } }
  - expect: { flow: t1, status: Held, why: changed the rate }
  - trenova: { release: t1 }
```

Each step is a mapping with exactly one of `send`, `expect`, `trenova`, `advance`, `script` and
`acks` (and an optional `name`). Steps run in order; the first failure ends the scenario and the
rest are reported as skipped.

## Conversations and seeds

A conversation is named when it starts (`as: t1` on an original 204, `tender: c1` for a carrier)
and every later step names it. Its key is the load number on the wire: the generated 204's
B2-04 for a shipper, the B2-04 of Trenova's 204 for a carrier. A batch (`count: 25`) starts a
group `b` of conversations `b#1` … `b#25`; a step about `b` holds for every member.

Seeds are offsets within the scenario, 1 to 39 (left out: the next free one). The runner adds
the run's base, `run × 1000 + scenario index × 40`, so a run never reuses a load number from
another run and `--run N` replays the same numbers. A partner holds at most 25 scenarios.

## send

The partner generates and sends a 204 over its own transport (`POST /control/<p>/send`).

| Key | Meaning |
|---|---|
| `set` | `"204"`, the only set a shipper partner generates |
| `as` | an original's conversation name |
| `of` | the conversation a later variation is about |
| `variation` | `original` (default), `duplicate`, `change`, `replace`, `cancel`; the partner must list it in `generate/204.yaml`'s `purposes` |
| `seed`, `count` | the seed offset and how many tenders go in one send (packed as the partner batches) |
| `sparse` | drop every optional segment |
| `delta` | `ratePercent`, `addStop`, `move: { stop: 0, by: 24h }` (stop 0 is the last) |
| `load` | an original's needs from the load it draws (below); later variations keep them |

After an SFTP partner's send the runner asks Trenova to poll the profile, as a person would.

### What a scenario needs from the load

A seed draws a load from `generate/204.yaml`'s ranges, and a partner's Lading rules hold some
of what it draws (a retailer load over its van limit, crossing reefer ranges). A scenario whose
expectation depends on such a property states it instead of trusting its seed:

```yaml
- send: { set: "204", as: t1, load: { maxWeight: 43500 } }      # a load the rules take
- send: { set: "204", as: t1, load: { minWeight: 44000 } }      # one they hold for a planner
```

| Key | Narrows the draw to |
|---|---|
| `minWeight`, `maxWeight` | a gross weight in that range (lb), within the partner's own |
| `stops` | exactly that many stops, within the partner's range |
| `hazmat` | `true`: a hazardous line on the first pickup; `false`: no hazardous commodity |
| `oneRange` | temperature-controlled lines one reefer setting holds (no crossing ranges) |

Every other property is drawn as the seed would draw it. A need the partner's generator
cannot meet (`maxWeight: 5000` for loads of 9,000 lb and up, `hazmat: true` for a partner with
no hazardous commodity) stops the run before it starts, naming the file and the step. The
need belongs to the conversation: a change, replacement or cancellation of `t1` varies the
same load. `edi-sim soak` draws its loads with the partner's happy path's needs.
`internal/partner/need_test.go` plays every shipped scenario's tenders, across runs, through
the partner's Lading rules and fails when one would be held, declined or rejected against the
scenario's expectation.

## expect

Each expectation waits up to its `within` (or the scenario's) and fails with what it saw `edi-sim run --deadline-scale 0.3` multiplies every wait for a quick pass.

**`wire: <set>`, `about: <conversation>`**: a transaction Trenova sent that the partner received
(or, for `997`/`999`/`TA1`, Trenova's acknowledgment of something the partner sent). Every
expectation of a set takes the next transactions about the conversation that no earlier
expectation took, so two `expect: { wire: "990" }` steps need two 990s.

| Key | Meaning |
|---|---|
| `accepted` | the partner's guide (the oracle) accepts it; `true` unless set to `false` |
| `elements` | element checks, `{ B1-04: A, N9-02: "NO TRUCKS" }`; a composite takes `SEG-NN-C` |
| `count` | how many transactions this step takes (default 1) |
| `statuses` | 214s only: the AT7-01 codes that must have arrived, in any order, across as many 214s as it takes |
| `total` | 210s only: B3-07 equals `tendered` (the latest tender, deltas applied; for a carrier, the rate Trenova tendered), `original`, a percentage of the original (`+12%`) or an amount (`1250.00`) |
| `resent` | a transaction whose body repeats an earlier one (a resend) |
| `none` | nothing new arrives for the whole window |

**`ack: <conversation>`**: Trenova's 997/999 for what the partner sent: the conversation's last
send, or with `set: "214"` every message of that set the partner sent about it (a carrier's
answers). `accepted: true` (default) wants every one accepted (AK5 A or E); `accepted: false`
wants at least one rejected. Trenova's acknowledgment must also pass the partner's guide.

**`flow: <conversation>`**: the Lading flow instance keyed by the conversation, read through
`/edi/flows/instances/`. `status` is `Open`, `Held`, `Escalated` or `Closed`; `why` is a
case-insensitive part of its reason; `count` is how many instances carry the key.

**`reconciled: <conversation>`**: every document Trenova sent about the conversation has the
partner's acknowledgment, and Trenova's own message records show it `Accepted`.

**`trenova: { shipment: <conversation>, … }`**: the shipment the conversation built (the flow's
shipment, or its transfer's): `exists: false`, `stops`, `status` (`New`, `Assigned`,
`InTransit`, `Completed`, `Invoiced`, `Canceled`, …) and `total` as for a 210.

## trenova

What a person does in Trenova, through the public API, against the conversation's flow and
shipment:

| Key | Does |
|---|---|
| `decide: t1, decision: accept\|decline, reason: …` | approves or rejects the tender's transfer; the flow answers with a 990 |
| `dispatch: t1` | assigns a worker and tractor to every unassigned move |
| `arrive: t1, stop: 2` / `depart: t1, stop: 2` | records a stop's actual arrival or departure (stops counted from 1 across the moves) |
| `deliver: t1` | arrives at and departs every stop not yet done, in order |
| `invoice: t1` | creates the invoice from the shipment and posts it |
| `release: t1` | releases a held flow instance |
| `tender: c1, rate: "1850.00"` | carrier partners: creates a shipment (the customer, service type and lane of a recent shipment) and spot-tenders its move to the partner's carrier by EDI; waits for the 204 to reach the partner |

## advance

`advance: { about: t1, by: 2h }` runs `trenova edi flows advance --instance <id> --by 2h --json`
for the conversation's flow instance: the flow's timers (answer windows, acknowledgment waits,
status batches) behave as if that much time passed. It needs `--trenova-cli`.

## script

`script: { answers: [ { decline: NO CAPACITY }, { deltaCents: 15000 }, {} ] }` queues a carrier
partner's answers to the next tenders (`POST /control/<p>/script`); `about` ties an answer to a
known conversation. An answer may `decline`, bill `deltaCents` more, send a `misfooted` 210, add
a `stray` 214 for an invented load, `delay` its 990 (a probability), or stay `silent`.

## acks

`acks: { about: t1, set: "214", action: withhold|reject }` makes the partner withhold, or reject
with AK5/IK5 R, its acknowledgment of the next transaction of that set about the conversation
(`POST /control/<p>/acks`). Each step queues one; two steps cover a document and its resend.
