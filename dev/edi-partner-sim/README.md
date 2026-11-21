# EDI simulator

`edi-sim` plays Trenova's trading partners. Each partner is data in `partners/<name>/`
([PARTNER-FORMAT.md](partners/PARTNER-FORMAT.md)): its identity, envelope, transport, batching,
acknowledgment behaviour, the guides it judges Trenova's documents against
([GUIDE-FORMAT.md](partners/GUIDE-FORMAT.md)) and how it writes what it sends.

| Command | What |
| --- | --- |
| `edi-sim serve` | the long-running partner endpoint: every partner at once over AS2 and SFTP, with a control API and an event stream |
| `edi-sim conform` | runs each partner's Lading project over generated messages and judges every outbound transaction with the oracle; no Trenova needed |
| `edi-sim run` | provisions each partner in a running Trenova and plays its scenarios ([SCENARIO-FORMAT.md](partners/SCENARIO-FORMAT.md)) over the real transport, judging both sides |
| `edi-sim soak` | holds many seeded happy-path conversations at once and measures them |

## Conforming the partners' Lading projects

```bash
task conform -- --count 200            # from dev/edi-partner-sim; CI runs --count 50
go run ./cmd/edi-sim conform --replay retailer:17
```

`conform` needs no Trenova. For each partner it checks the Lading project and runs its tests,
then plays `--count` seeded cases through `pkg/lading`, the wasm module the TMS runs:

- **A shipper** tenders a generated 204 (original, change, replace or cancel). The case reads it
  through the project, then writes what Trenova sends back from what the map read: the 990
  (accept, decline, decline with a reason), a 214 for every stop arrival and departure the
  partner's flow sends on, and the 210.
- **A carrier** is tendered Trenova's 204 and answers with its 990, 214s and 210 (some declined,
  stray, misfooted or billing off the tendered rate). Each answer is read through the map and
  then stepped through the flow, from the instance the tender opened or, for a load the flow
  never saw, from none, so a flow's own reject or hold counts as it would in Trenova.

What Trenova's side holds is drawn from the case's seed the way a real tenant's data varies
(`internal/conform/variety.go`): event, response and invoice times carry seconds and land on
local midnight, the last minute of a day, a daylight saving shift or a time whose UTC date is
another day; a time is held in the stop's zone, the organisation's or UTC; names, streets,
cities, BOLs, POs and notes come lowercase, title case, with accents, with one of the
partner's separator characters, exactly at the partner guide's width or past it; a shipment
may lack its BOL, weight or pieces and a bill-to its address; invoices add fuel, stop-off,
detention and lumper lines with odd cents that sum to the total, a per-mile rate, a weight
and a delivery date; and Trenova's tender to a carrier moves its windows and takes odd cents.
Each varies once where Trenova holds it once: one BOL per shipment, one city per stop.

Every message Trenova writes is judged by the oracle against the partner's `guide/` files, its
envelope against `partner.yaml`, and the 997 or 999 the partner would answer against its reject
rules. A message the map declares `not sent` is reported as not sent: nothing goes out and no
acknowledgment is awaited. A message a rule holds is reported held and not judged, even when
the crate's own checks would also have stopped it: in Trenova both come back as reasons and
nothing is sent.

The summary gives each partner's cases passed and failed, the cases with a held message and the
messages not sent, and each failing rule with the seeds it failed on and the command that
replays one (`--replay <partner>:<seed>` prints the case's documents, wire and findings).
`--report` writes the whole run as JSON.

The documents Trenova's maps receive are built in `internal/conform/documents.go` by copying
the TMS's outbound builders (`ladingdoc`, `edipayloadbuild`, `edioutbound`), since importing
them would pull in the TMS's domain. `internal/conform/mirror_test.go` pins each copied rule to
the TMS function it copies, by name: when a builder changes, change the copy and its test.

## Serving the partners

```bash
task sim        # from dev/edi-partner-sim; keys, mailboxes and history persist in ./.sim-*
go run ./cmd/edi-sim serve --partner manufacturer,carrier --identity-dir ./.sim-identity
```

Requires nothing to start. To trade with Trenova, run the API and worker (`task run-watch`,
`task worker` in `services/tms`) and give each partner Trenova's side of the connection (below).

Each partner gets:

- **Its own AS2 identity** (an AS2 partner): the AS2 ID from `partner.yaml`, a keypair and
  certificate kept in `--identity-dir/<partner>/`, and the endpoint `/as2/<partner>` (`POST /as2`
  also works and routes by `AS2-To`). It honours `partner.yaml`'s `signed`, `encrypted`,
  `compressed` and `mdn` settings both ways: what it sends is signed, encrypted (AES-256 CBC),
  compressed and asks for a sync, async or no MDN as set; what it receives must be signed or
  encrypted when the partner signs or encrypts, and it answers with the MDN Trenova asked for
  (sync in the response, async posted to `Receipt-Delivery-Option`).
- **Its own SFTP mailbox.** One SFTP server (`--sftp-listen`, default `:9222`, one host key kept in
  `--identity-dir`) with one user per partner, named after the partner, all signing in with
  `--sftp-password`. Each user is confined to `--sftp-root/<partner>/` (an `os.Root`, so no path
  leaves it). An SFTP partner's folders come from `partner.yaml` (`inbound`, `outbound`, and
  `archive` beside `inbound`); every other partner gets `/inbound`, `/outbound` and `/archive`.
  The partner drops files for Trenova in its inbound folder; a file Trenova finishes writing in
  the outbound folder is received at once, and a file Trenova moves or removes from the inbound
  folder is recorded as picked up.

### What a partner does with a file from Trenova

Every file, over AS2 or SFTP, is judged by the oracle against the partner's `guide/` files and
the judgement is stored with the file. Then the partner:

- links Trenova's 997 or 999 (and TA1) to the interchange it acknowledges;
- answers a TA1 when Trenova asked for one (ISA14 = 1): `A`, or `R` with the envelope error's code;
- builds its 997 or 999 from the oracle's findings (AK3/AK4 or IK3/IK4 notes, AK5 per its
  `reject` rules), batched as the partner batches, and sends it after `acknowledgment.delay`, or
  withholds it with probability `acknowledgment.never` (seeded by `--seed`). Every delayed send
  waits in one timer heap, not a goroutine per message;
- as a carrier, answers each accepted original 204 with a 990, its 214s and a 210, spaced by
  `answers.pace`: by the next script entry given to it (`POST /control/<partner>/script`), or by
  `answers.policy` in `partner.yaml`.

### What a partner sends

`POST /control/<partner>/send` generates and delivers over the partner's transport:

```json
{ "set": "204", "seed": 37, "with": [38, 39], "variation": "change", "sparse": false,
  "delta": { "ratePercent": 12, "addStop": false, "move": { "stop": 0, "by": "24h" } },
  "load": { "maxWeight": 43500 } }
```

`variation` is `original` (default), `duplicate`, `change`, `replace` or `cancel`; `with` adds
seeds to the same send, packed per the partner's batching. `load` narrows what the seed draws
(the [scenario format](partners/SCENARIO-FORMAT.md#what-a-scenario-needs-from-the-load) lists
its keys); a later variation of a load must carry the same `load` as its original, or it varies
a different load. A partner that resends (`duplicates:
resend`, the legacy partner) answers `duplicate` with the original's exact bytes and control
numbers; it refuses one for a seed it never sent. `{"payload": "ISA..."}` sends raw X12. The
call returns once delivery finished, with each message, its MDN and its state.

## Control API

| Method | Path | Purpose |
| --- | --- | --- |
| `GET`  | `/control/partners` | every partner: identity, AS2 ID, URL and certificate PEM, SFTP host, port, user, password, host key and folders, Trenova's side as configured |
| `GET`  | `/control/<partner>` | one partner, the same |
| `POST` | `/control/<partner>/trenova` | Trenova's side: `certificatePem`, `inboundUrl` (the profile's **Inbound URL**), `as2Id` (default: `trenova.id` from `partner.yaml`) |
| `POST` | `/control/<partner>/send` | generate and send (above) |
| `GET`  | `/control/<partner>/messages` | both directions, newest last: raw X12, oracle judgement, ack sent or received and its status, TA1, MDN, and `documents` (below); `?direction=sent\|received&set=&key=&limit=` |
| `GET`  | `/control/<partner>/messages/<id>` | one message |
| `GET`  | `/control/<partner>/conversations` | messages grouped by load number (204 B2-04, 990 B1-02, 214 B10-02, 210 B3-03; acks join their document's), with the sets seen and the run scenario attached to the load, if any |
| `GET`  | `/control/<partner>/conversations/<key>` | one conversation as the viewer draws it: its `rows` (each transaction, Trenova-side event and clock jump in time order, with its verdict and the scenario's mark), its `messages`, and the attached `run` and `scenario` |
| `GET`/`POST` | `/control/<partner>/script` | a carrier's queued answers: `{"answers":[{"key":"<load>","decline":"…","deltaCents":0,"misfooted":false,"stray":false,"delay":0.3,"silent":false,"seed":0}],"replace":false}`; an entry without `key` answers the next tender |
| `GET`/`POST` | `/control/<partner>/acks` | acknowledgment overrides: `{"overrides":[{"set":"214","key":"<load>","action":"withhold\|reject"}],"replace":false}`; each withholds, or rejects with AK5/IK5 R, the partner's acknowledgment of the next matching transaction |
| `GET`  | `/control/<partner>/mailbox` | the partner's SFTP folders and their files |
| `POST` | `/control/reset` | forget history, scripts and pending sends (`{"partner":"x"}` for one); keys, Trenova's side and control numbers stay |
| `GET`  | `/control/events` | server-sent events of every change (`?partner=`); reconnect with `Last-Event-ID` to replay what was missed |
| `GET`/`POST` | `/control/runs` | the runs kept (newest first) and the current one, each with its tally; `POST` starts one: `{"kind":"run\|soak\|conform","number":12,"seed":0,"api":"…"}` |
| `GET`/`POST` | `/control/runs/<id>` | one run with every scenario and step; `POST {"status":"passed\|failed"}` finishes it |
| `POST` | `/control/runs/<id>/scenarios` | adds or updates a scenario: `{"partner","slug","name","seed","status","replay","keys":[],"steps":[…]}` (`steps` left out keeps the ones posted) |
| `POST` | `/control/runs/<id>/steps` | records one step: `{"partner","scenario":"<slug>","step":{"index","kind","text","status","why","keys","matches":[{"messageId","groupControl","control","key"}],"events":[{"key","at","label","detail","tone":"trenova\|hold\|bad","who"}],"gap"}}` |
| `GET`  | `/healthz` | liveness |

Events are `message.received`, `message.scheduled`, `message.sent`, `message.failed`,
`ack.scheduled`, `ack.sent`, `ack.failed`, `ack.withheld`, `ack.received`, `ta1.scheduled`,
`ta1.sent`, `ta1.received`, `answer.scheduled`, `mdn.received`, `mdn.sent`, `mdn.failed`,
`file.picked_up`, `message.loaded` (a message read from `--report`) and `reset`; each carries
the message as it now stands. `run.started`, `run.scenario`, `run.step`, `run.finished` and
`run.reset` carry `run: {runId, scenario, keys}` instead, naming the loads the change touches.

Every message holds one `documents` entry per transaction (and per TA1): its set, ST and GS
control numbers, the ISA control number, the load numbers it is about, a title read from the
X12 (`Original tender`, `Change: rate +12%`, `X3 arrived pickup`, `Accepted 204 501`), its
elements split by the partner's own separators with the ISA/GS and GE/IEA lines, the oracle's
findings for it (segment index, element position, X12 code, rule and guide file), and the link
between an acknowledgment and what it answers: `answers` on the 997, 999 or TA1, `answeredBy`
(with the AK5/IK5 code) on the transaction. Each document has a number, kept across restarts,
that the viewer's links use.

History is kept in memory, `--history` messages per partner (oldest dropped first), and the
last 20 runs. With `--state`, a JSON snapshot is written every two seconds when something
changed and on shutdown; a restart restores the history, the runs, Trenova's side, the scripts,
the remembered originals and the control numbers, and delivers the acknowledgments and answers
that were still waiting.

## Seeing a run

`edi-sim serve` serves the run viewer at `/` (`http://localhost:9210/` with `task sim`): the
partners, each partner's conversations by load number, the selected conversation as a ladder
(the partner's lifeline, Trenova's lifeline, time downward, each transaction an arrow and each
acknowledgment a dashed one, Trenova's planner and flow events on its lifeline) with the
scenario's marks in the **Expected** column, and the selected message: its verdict, control
numbers, transport and MDN, what answers it, its X12 with the oracle's findings beside the
element they name, the scenario's steps and the command that runs the case again. The page is
plain HTML, CSS and JavaScript embedded in the binary; it uses Geist from Google Fonts and falls
back to system fonts offline.

It follows `/control/events`, so a running conversation grows on the ladder as messages and
steps arrive; if the stream drops it says so and reconnects, replaying what it missed. Links
keep the view: `#manufacturer`, `#manufacturer-KVW-48213`, `#manufacturer-KVW-48213-13` (the
last number is the transaction's document number). Rows are focusable; Enter selects one.

**Run progress.** `edi-sim run` and `edi-sim soak` post their progress to the simulator they
play against (`--sim`) as they go: the run when it starts and finishes, each scenario with its
planned steps, and each step as it ends, with the transactions it matched (the message, GS and
ST control numbers the expectation consumed), the Trenova-side events it saw (planner accepted
or declined, dispatched, arrived, delivered, invoiced, flow held, released or open, shipment
status, reconciled) and the clock jumps `advance` made. The viewer attaches a conversation to
the latest scenario that touched its load number and puts each step's mark on the rows it
matched, so the check marks align with the right transactions. A soak conversation appears as a
scenario of its own, its milestones (first acknowledgment, 990, 214, 210) as steps. When the
simulator does not take the progress, the run says so once and carries on.

**A finished run.** `edi-sim serve --report run-report.json` (or a `conform --report` file)
loads that run into the viewer: its messages into the history and its scenarios, steps and
marks as a run marked `from <file>`. A conform case becomes one scenario per seed, its replay
`edi-sim conform --replay <partner>:<seed>`; conform has no clock, so its messages carry the
report file's time. A soak report holds timings, not the wire, and is refused. Loading the same
report again into a `--state` that already holds it does nothing.

```bash
task sim                                                     # terminal 1, then open http://localhost:9210/
task run -- --trenova-cli ../../services/tms/build/trenova   # terminal 2: watch it play
go run ./cmd/edi-sim serve --report run-report.json          # or look at a finished run
```

### Where the SFTP files land

With `task sim` each partner's mailbox is `./.sim-sftp/<partner>/`:

```bash
ls ./.sim-sftp/manufacturer/outbound      # what Trenova delivered over SFTP
ls ./.sim-sftp/retailer/tamberlane/to-carrier
curl -s localhost:9210/control/retailer/mailbox | jq
```

## Playing scenarios against Trenova

`edi-sim run` needs the simulator (`task sim`), the API and the worker (`task run-watch` and
`task worker` in `services/tms`) and a seeded database (`task db-seed`: the run dispatches with
the tenant's workers and tractors and, for the carrier partner, copies a recent shipment's
customer and lane). Steps that advance a flow's clock need a built CLI:

```bash
(cd ../../services/tms && go build -o ./build/trenova ./cmd/cli)
task run -- --trenova-cli ../../services/tms/build/trenova --keep-going
go run ./cmd/edi-sim run --partner manufacturer --scenario happy-path -v \
  --trenova-cli ../../services/tms/build/trenova
```

For each partner it signs in (`--email`, `--password`), then idempotently creates or refreshes
the partner (code `EDISIM-<NAME>`), its communication profile (AS2 with both certificates, or
SFTP at the simulator's mailbox with its host key pinned; Trenova's AS2 keypair persists in
`--identity-dir/trenova`), the carrier and its EDI channel for a carrier partner, and its Lading
project from `partners/<p>/lading`: a draft holding every file and sample, tested, replayed
against the active version when there is one, certified and activated. A project whose files
are already active is left alone. When certification fails the run prints why (failing tests,
errors) and skips that partner's scenarios. Last it gives the simulator the profile's inbound
URL (`--trenova-public-url` when the simulator reaches Trenova at another host).

`--deadline-scale 0.3` multiplies every wait a scenario states (its `within`s and the 3m
default) for a quick pass; the scenarios keep their deadlines as written. A step that fails
ends its scenario and the rest are reported skipped. Each scenario's result line prints as the
scenario ends (with `-v`, its steps and wire first), and the summary ends with the time spent
waiting on expectations and on Trenova actions, and how much of it went to steps that failed.

Every load number is keyed by the run (`--run 1..899`, default the one after the last, kept in
`.sim-run`), so runs never collide in Trenova; `--run N` replays a run's exact numbers. Waits
follow the simulator's event stream and poll Trenova's API with backoff, each up to its
deadline. The run writes `run-report.json` (per scenario: steps with pass or fail and timings,
each conversation's wire log on both sides with the oracle's findings, Trenova's state after
each action, milestones and the seed) and prints a summary with the command that replays each
failure; it exits non-zero on any failure. `--keep-going` runs every scenario and partner after
one fails.

`edi-sim soak --partner all --conversations 200 --rate 2 --concurrency 16` provisions the same
way, then holds seeded happy-path conversations (tender, accept, dispatch, deliver, invoice; a
tender out for the carrier) and reports, per partner, the p50/p90/p99/max time to the first
997, 990, 214 and 210, the conversations that never reached one (lost), duplicated documents,
and flows still open after `--settle`. It writes `soak-report.json`.

## Setting the simulator up from its guide

`specs/manufacturer-implementation-guide.txt` is the manufacturer partner
(`partners/manufacturer`, Kestrel Valve Works) written up as its implementation guide: its AS2
and SFTP endpoints, ISA/GS identifiers, separators, the documents it exchanges, its 997s, its 204
rules and codes, and a sample 204 that is exactly what it sends for seed 37
(`internal/sim/spec_test.go` holds the two together). With the simulator running, attach the
guide to a Desk conversation with an agent that handles EDI and ask for the connection to be set
up from it. The draft should be ready with no decisions; after approving it, enter your AS2
private key and certificate and the manufacturer's certificate (from
`/control/manufacturer`) on the communication profile, then turn the profile on. Copy the
profile's **Inbound URL** from its **Transport** tab and give it to the simulator with your
certificate:

```bash
curl -s localhost:9210/control/manufacturer/trenova -d @- <<JSON
{"inboundUrl": "http://localhost:8080/…", "certificatePem": "$(sed 's/$/\\n/' trenova.pem | tr -d '\n')"}
JSON
curl -s localhost:9210/control/manufacturer/send -d '{"set":"204","seed":37}' | jq
```

It sends nothing until it has the inbound URL. The guide names no password or key, so none is
drafted.

## Flags

`serve`: `--partners` (folder), `--partner a,b`, `--listen`, `--public-url` (how Trenova reaches
the simulator: AS2 URLs and async MDN returns; default `http://localhost:<the --listen port>`), `--identity-dir`, `--sftp-listen`, `--sftp-host`,
`--sftp-password`, `--sftp-root`, `--state`, `--history`, `--seed`, `--workers`, `--report`
(a run or conform report to show in the viewer).

`conform`: `--partners`, `--partner`, `--seed`, `--count`, `--workers`, `--report`, `--replay
<partner>:<seed>`, `-v`.

`run`: `--api`, `--sim`, `--trenova-public-url`, `--email`, `--password`, `--partners`,
`--partner`, `--scenario`, `--run`, `--run-file`, `--identity-dir`, `--trenova-cli`,
`--trenova-dir`, `--keep-going`, `--deadline-scale`, `--report`, `-v`. `soak` takes the same connection, partner,
run and CLI flags (`--api`, `--sim`, `--trenova-public-url`, `--email`, `--password`,
`--partners`, `--partner`, `--run`, `--run-file`, `--identity-dir`, `--trenova-cli`,
`--trenova-dir`, `-v`) and `--conversations`, `--rate`, `--concurrency`, `--within`, `--settle`,
`--report`.
