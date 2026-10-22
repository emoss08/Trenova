# The desk bench

The desk bench (`trenova desk`) drives the Desk's agents through real turns, on this checkout's
code, and writes down everything the harness did: every model request and reply, every tool
call with its arguments, verdict and result, every refusal, retry and regrounding, every
proposal with its preview, every decision and the agent's follow-up. It exists so a person, or
an agent like Claude Code, can find where the harness misleads a model (a tool that is missing,
a description that is read wrongly, a schema that keeps refusing calls, a prompt rule that
fires on the wrong question) before a user does, fix it, and see whether the fix held.

It is a development tool. It refuses to run in production or staging, and it approves real
writes against whatever database the configuration points at.

## How a turn runs

Nothing is simulated. A message goes through `assistantturnservice.StartTurn` and
`assistantjobs.StartTurnWorkflow`, the same calls the Desk's handler makes, as a person (by
default `admin@trenova.app`) with the Desk surface, and is answered by `AssistantTurnWorkflow`
on Temporal: prepare, the agent loop, the tool activities, finish. Decision follow-ups, queued
messages and delegated tasks run exactly as they do for a person.

Four things make that observable and repeatable:

- **Its own worker, on its own namespace.** The command builds the worker's graph
  (`bootstrap.UnscheduledWorkerOptions`: every queue's workers, none of the start-up schedule
  and reconcile workflows) and runs it in-process against the Temporal namespace `deskbench`,
  which it registers when missing, keeping closed runs for an hour. The local dev server holds
  every run in memory until retention lets it go, and three days of bench runs filled its 1 GiB
  limit until it stopped answering polls; a run's files under `.deskbench/` hold all it needs, so
  an older namespace's longer retention is shortened on the next run. A dev worker on `default` running older code never answers
  a bench turn, and a bench run never touches the dev worker's work. Edit a tool description,
  run the bench again, and the new description is what the model reads. The edition's
  worker-only options are left out, since they add schedules of their own.
- **A recorder on the model router.** `agentports.CompletionService` is decorated
  (`deskbench.Recorder`) so every chat, streamed and structured call is copied as it was sent and
  as it came back: the system prompt, the messages, the tools offered, thinking, tool calls,
  tokens, latency, cost, retries, fallbacks and errors. Calls are tied to a turn by their usage
  attribution (`ThreadID`, `OwnerID`); a call that carries neither (the scope guard's
  classifier) is listed beside the turn it ran during.
- **A watcher on the workflow starter.** `kernelports.WorkflowStarter` is decorated
  (`deskbench.TurnWatcher`) so every `AssistantTurnWorkflow` started on a bench conversation, by
  the bench or by the application (a decision follow-up), is seen the moment it starts. The
  bench follows each one's stream from the first frame, waits for its result, and keeps
  waiting a few seconds for another turn the last one started.
- **No learning.** `agentports.AgentReflectionScheduler` is replaced with one that does
  nothing, so a bench conversation is never looked back over. A conversation goes quiet after
  ten minutes, so a run longer than that fed what its early cases taught to the cases after them:
  a rejected hold taught "this person does not want a hold for missing paperwork", and the case
  that asks for that hold failed on every run after. Before a run, the tidy-up
  (`benchLeftovers`) also retires the memories earlier bench conversations taught and dismisses
  what they suggested; memories from people's own conversations are never touched. Reflection
  itself is covered by its own tests, not by the bench. For the same reason
  `agentports.AgentTrustService` is wrapped so the bench's approvals, its end-of-case
  rejections and its failed executions are not outcomes for earned autonomy: its streaks
  promoted `billing_transfer` to AutoExecute on Billing exceptions, and the case that
  approves that proposal found it had already run.

The worker's own log goes to `worker.log` in the run directory, so a tool's error that never
reached the model is there too.

## Commands

Run from `services/tms`, with Postgres, Redis and Temporal up (`task docker-up`). The API and
the dev worker do not need to be running.

One bench runs per namespace at a time. Every bench process registers its workers in the same
Temporal namespace under the same identity, so a second one would take the first one's turns and
run them on its own build: an `ask` during a `run` once made a fix look broken because the run's
older binary answered it. Open holds an exclusive lock on `.deskbench/<namespace>.lock` (with the
holder's pid in it) and a second bench refuses to start; pass `--namespace` to run one beside it.

```bash
trenova desk ask "whats running late today"                    # one message, full transcript
trenova desk ask "find the walmart load" "push it to friday"    # several, in one conversation
trenova desk ask --approve "mark S-1042 delivered"              # approve what it proposes
trenova desk ask --reject "use the 2pm window" "reschedule ..." # reject, with a note
trenova desk ask --provider "Haiku 5.5" "..."                   # pin one model
trenova desk ask --attach rate-con.pdf "build a shipment from this"  # attach a file to message 1
trenova desk ask --attach 2=doc_01... "hi" "here it is"         # a copy of a stored document, on message 2
trenova desk ask --thread athr_... "and the next one?"          # continue a conversation
trenova desk clean                                              # delete every [bench] conversation
trenova desk run                                                # every scenario, once
trenova desk run --repeat 3 --provider default --provider "GPT-5"  # a model matrix
trenova desk run --tag dispatch --name late --parallel 2
trenova desk agents                                             # names --agent takes
trenova desk providers                                          # names --provider takes
trenova desk findings --status all --format csv                 # export what runs have found
trenova desk route                                              # score where questions would go
task desk-ask -- "how many loads did we deliver this week"
task desk-bench -- --repeat 3
trenova desk run --provider claude-haiku-5-5@low --provider claude-haiku-5-5@medium  # effort cells
trenova desk run --provider claude-haiku-5-5@low --provider claude-haiku-5-5@low+advisor  # advisor cell
trenova desk run --split held-out --pass-k 4 --repeat 4         # one split, pass^4
trenova desk run --tag unattended                               # fired event and schedule runs
trenova desk run --tool-threshold 0                             # always disclose (8 + find_tools)
trenova desk baseline                                           # the Phase 0 matrix, deskbench/baseline.yaml
trenova desk sweep --held 8,12 --mode all,disclosed             # part of the tool-count sweep
task desk-baseline
task desk-sweep
trenova desk gate --probe                                       # project the gate's cost and time
trenova desk gate --budget .deskbench/gates/<time>-probe        # the gate run (deskbench/gate.yaml)
trenova desk gate --confirm .deskbench/gates/<time>             # confirm it on the held-out split
```

`--provider` takes a provider's name, model id or id; `default` is the organization's own
order, which is what a person who never opens the model picker gets. A provider may end in
`@effort` (`off`, `none`, `minimal`, `low`, `medium`, `high`) to override its reasoning effort for
that cell. Each `--provider` is a cell, and a cell is pinned: its calls never fail over to another
provider, so every row of the matrix measures one model at one effort. A cell may also end in
suffixes after the effort, each starting with `+`; an unknown suffix is an error.
`--tool-threshold N`
overrides `ai.toolDisclosureThreshold` for the run (`0` always discloses, a large number never
does). `--agent` takes an
agent's name, id or template and defaults to the general assistant, the agent Ask uses.
Every case is a real Desk conversation of that person's, titled `[bench] <scenario> #<repeat>`,
and it stays in the Desk once the case ends, to open and read like any other. Whatever the agent
proposed and the scenario did not decide is rejected at the end, so the decisions inbox does not
fill up, and the agent's answer to that rejection is in the conversation too. `--keep` leaves
those proposals waiting instead, to decide them in the Desk; `--delete` deletes each
conversation when its case ends; `trenova desk clean` deletes every `[bench]` conversation. A
Desk tab does not follow a bench turn live (the API reads turn streams from its own Temporal
namespace), so a conversation appears once its turns finish. A formula scenario's studio thread
is always removed at the end, since the studio keeps one thread per person and template and the
next run would otherwise replay this one's history.

### Cells: `+advisor`

`+advisor` gives the cell's model Anthropic's advisor tool, overriding the provider's own
advisor setting (AI Control → Providers, [agent-runtime.md](agent-runtime.md), "The advisor"): `claude-haiku-5-5@low+advisor` is
Haiku 5.5 at low effort that may consult Claude Opus 5.5 up to twice per model call.
`+advisor=<model>` names another advisor model; the API refuses a pair whose advisor is less
capable than the executor, with a 400 on the first call. The suffix is refused on the default
provider and on any provider that does not speak the Anthropic Messages API. The cell's label
carries it (`Haiku 5.5 (claude-haiku-5-5) @low +advisor`), so it is its own row of the matrix.

The suffix rides on the cell's pin, as the effort does: every chat call of the case's thread gets
`ChatCompletionRequest.Advisor`, and the Anthropic adapter declares `advisor_20260301` (beta
`advisor-tool-2026-03-01`) after the agent's own tools. A cell without the suffix runs with
whatever advisor its provider's setting declares. Unlike the provider setting, a cell's advisor
that the API refuses is not retried without it: the call fails, so a cell never runs quietly
without what it asked for. A call's advisor consultations are counted
apart from the executor's tokens (`advisor` on each call in `calls/`, `advisorCalls`,
`advisorPromptTokens` and `advisorOutputTokens` in `usage`) and priced at the advisor model's list
price, inside the call's cost. A call whose advisor model has no list price is left unpriced
rather than counted at the executor's rate. How the advice is kept and replayed is in
[agent-runtime.md](agent-runtime.md), "The advisor".

`desk run` compares with the previous run by default (`.deskbench/latest`), printing the
scenarios whose pass rate or refused-call rate moved, and exits non-zero when a case fails.
`--compare <dir or report.json>` compares with a particular run; `--compare ''` with none.

## Watching it

Every run streams what is happening to `.deskbench/live.log` (truncated when a run starts, and
copied to the run's own `live.log`): each case starting and its verdict, each turn's message,
the reply and thinking as they stream, every tool call with its arguments and what came of it,
retries, refusals and regrounding, every proposal and decision. Follow it from `services/tms`:

```bash
tail -f .deskbench/live.log
```

Each line names its case, so cases run with `--parallel` stay readable. The finished
conversations are in the Desk as `[bench] …`.

## What a run writes

Each run writes to `.deskbench/runs/<time>-<ask|run>/`:

| File | What it holds |
|---|---|
| `summary.md` | pass matrix by scenario and model, the comparison, tools that were refused or failed with sample reasons, model calls worth a look (cut off, retried, fell back, unparsed arguments), every failing check |
| `transcript.md` | each case's steps and checks, then each turn as a timeline: model calls (thinking, text, the calls asked for), tool calls (verdict, timing, arguments, the result the model read), notable stream events (`retrying`, `reply_regrounded`, `run_tainted`, `refused`, `world_changed`, delegation, memory), proposals with their preview, and the reply |
| `report.json` | all of it, machine-readable |
| `calls/<case>/turnNN-callNNN.json` | one model call exactly as sent and received: messages, the tool names offered, the result |
| `prompts/<digest>.md` | each distinct system prompt, once |
| `tools/<digest>.json` | each distinct tool list with full schemas, once |
| `live.log` | the run as it happened, line by line |
| `worker.log` | the in-process worker's JSON log |
| `temporal.log` | the Temporal SDK's own log |

A call file names its system prompt and tool list by digest, so two calls with the same
`systemDigest` saw the same prompt and a changed digest after an edit is the edit taking effect.

## What a run measures

Each case, and each cell (a provider at an effort, split by scenario split), reports:

- **Tokens.** Uncached input, cache read, cache write and output, each apart. Anthropic reports
  input without the cached tokens and OpenAI with them; the bench subtracts so `in` is always
  input the cache did not serve.
- **Cost.** Priced per call at the provider's own prices, with cache reads and writes at their
  own rates and Haiku 5.5's long-prompt tier above 100K prompt tokens. A provider with no prices
  is priced from the model's list price (`aiprovider.ListPrice`) or the price reference, and the
  figure is marked `~`.
- **Work.** Model calls, tool calls, refused calls by verdict, guard re-asks by guard (the
  `reask` on each call), time to first token and time per task.
- **Cache on later calls** (`CACHED 2+`, `laterCacheReadShare`): the cache-read share of every
  model call of a turn but its first, which writes the cache. A stable prefix shows here first:
  the rebuild of context assembly is held to above 80%.
- **Opens with find** (`findFirstShare`): the share of turns that called a tool whose first call
  was `find_tools`, the step a missed preselection costs.
- **Tool selection**, against the scenario's labels: precision, recall, the position of the
  first expected call, and calls to forbidden tools. Core and runtime tools (`find_tools`,
  `withdraw_proposal`, …) are left out unless a label names them.
- **Argument validity**: the share of a trial's tool calls whose arguments the tool's schema
  accepted, that is every call but those refused as `invalid` (`ARGS VALID`). A denied,
  duplicate or over-budget call is not an argument error.
- **Unauthorized effects**, which must be none: after each case the bench reads the case's runs
  from the database and counts every change that took effect without the authority it needed
  (`effects.go`): a proposal executed below `AutoExecute` with no approval that still stands (an
  undone approval is none); one executed on its own in a tainted run where the policy holds such a
  write (it leaves the organization, it moves money with no allowance, the run read web content
  and the tool does not escalate, or the policy marked it held as tainted); a write by a fan-out
  delegate, one answering a `delegate_task` call (a task a person directed to another agent writes
  under its own approvals and is not counted); a write by a tool the scenario forbids; a write the
  step ledger settled that no proposal accounts for; and an approval given against a preview of
  another version of the record than the one the proposal pinned, version 0 included. Every query
  names the bench user's organization and business unit as well as running under its row-level
  scope. It reads
  the database rather than the runtime's own verdicts, so a runtime that misjudges its own
  authority is still caught. Each one is listed in `summary.md` under "Unauthorized effects" with
  its case, tool and rule, and the cell's `UNAUTHORIZED` column reads `UNSAFE`; a case whose
  audit could not be read counts as unaudited, which a cell is not safe with either. It is a
  measure, not a check: a case still passes or fails on its own checks.
- **Retrieval**, on disclosed turns, against the tools the turn actually used. Preselection
  recall is the share the turn opened with loaded (the 8 ranked plus the companions,
  followers, fed and carried-over tools loaded beside them; a tool declared deferred is not
  one the turn opened with). `find_tools` recall is scored
  per answer, the share of the used tools the turn did not open with that one answer
  loaded, averaged over answers; `findUnionRecall` is the same against every answer of the
  turn together.

`--repeat N` runs every scenario N times per cell. The matrix gives the mean per trial, never a
sum, with a 95% Student-t interval. A scenario's pass rate takes a Wilson interval over its
trials; a model's is the mean of its scenarios' pass rates with a t-interval over scenarios,
because one scenario's trials pass and fail together and pooling them would draw the interval
too narrow (`report.json` keeps the pooled Wilson rate as `pass`, beside `scenarioPass`); `pass^k` (`--pass-k`,
default 4) is the chance all k trials of a scenario pass, estimated as C(c,k)/C(n,k) from c
passes in n trials and averaged over scenarios, so it needs `--repeat` of at least k.
`--compare` compares means per trial, so a run with a different `--repeat` still compares. Each
model's cell is also set beside its cell in the earlier run, before → after, on pass^k, recall,
preselection recall, model and tool calls, time to first token, the cache share of later calls,
the share that opens with `find_tools` and cost per task (`CompareCells`).

### Counting cache misses

Turn on `ai.cacheDiagnostics` (`cacheDiagnostics: true` under `ai:` in your config, or
`TRENOVA_AI_CACHEDIAGNOSTICS=true`) and every Claude API call asks where its prompt stopped
matching the call before it. A changed prompt is logged at info as "the prompt stopped matching
the call before it", with `reason` (`tools_changed`, `system_changed`, `messages_changed`,
`model_changed`) and `missedTokens`. Count them in a run's worker log:

```bash
grep -c '"reason":"tools_changed"' .deskbench/runs/<run>/worker.log
grep -o '"reason":"[a-z_]*_changed"' .deskbench/runs/<run>/worker.log | sort | uniq -c
```

Within a turn the tool array is the same on every call (`withdraw_proposal` is declared from the
start, and the final answer keeps the tools with `tool_choice: none`), so `tools_changed` inside
a turn is a regression; across turns it is disclosure loading tools.

## Baseline and sweep

`trenova desk baseline` (`task desk-baseline`) runs every scenario in every cell of
`services/tms/deskbench/baseline.yaml` (Haiku 5.5, the default model, at low and medium effort,
4 trials, pass^4) and writes `baseline.md` and `baseline.json` beside
the usual run files in `.deskbench/baselines/<time>/`. Every cell must name a provider the
organization has; `trenova desk providers` lists them. `--split`, `--tag` and `--parallel` narrow
it as they do `run`. Later runs compare against it with `--compare .deskbench/baselines/<time>`.

The baseline is what the context-assembly rebuild is compared against, and that comparison is the
same model before and after, so the matrix holds only the model the product defaults to. A larger
model is not a standing cell: when Haiku fails a scenario and it is unclear whether the model or
the harness is at fault, run that scenario once on it (`trenova desk run --provider
claude-sonnet-5-5 --repeat 1`). Another model's cell goes in a spec of your own passed with
`--spec`. The report gives each cell's cost, so run one split first (`--split held-out`) and
scale from what it spent.

`trenova desk sweep` (`task desk-sweep`) sets `ai.toolDisclosureThreshold` from data. For each
held count N (8, 12, 20, 30) the scenario's agent is copied as `Bench · Sweep · <name>` holding
the tools the scenario's labels need plus the agent's others up to N, and every scenario runs
once with every held tool sent (`all`) and once disclosed (the 8 ranked plus `find_tools`). A
scenario needing more than N tools, or an agent holding fewer, is skipped at that point. Each
point is a run under `.deskbench/sweeps/<time>/<mode>-<N>/`, repeated twice (`sweep.repeat`; two
trials leave pass^4 unreported, and the threshold reads the pass rate), and `sweep.md` /
`sweep.json` at the top line them up by model. Set the threshold to the largest N at which `all` is at least as good.

## The gate

`trenova desk gate` (`task desk-gate`) runs `deskbench/gate.yaml` as one run and measures from
it, by rules the spec fixes before the run, which models' providers scripts help, which effort
each route runs best at, and whether the advisor helps on it; a second run confirms the findings
on the held-out split. Its output is a list of recommended provider settings; it changes no
setting, and an organization decides from its own use. The plan, the rules and why they are what they are, are in
[desk-bench-gate.md](desk-bench-gate.md).

Code mode is judged per cell of the spec's `codeMode` rule on Phase 5a's paired scenarios
(`mode` and `pair`; a scenario's `mode`, `discrete`, `script` or `script_writes`, overrides the
provider's Scripts settings for its case, in development and test only), by that phase's rule: on the held-out pairs, no pass^k loss in script mode
for any work kind; cost per completed task `costCut` lower or pass^k `passGain` higher; the
controls within `controlWithin`; and no unauthorized effect on the adversarial pairs. Script
halves are left out of every other figure the gate reads, because the product runs no script
until an organization turns Scripts on for a provider. Where the rule holds, the gate recommends
"Scripts on" for that model's provider, and "Scripts may propose writes on" only where the
batch-write adversarial pairs ran and showed no unauthorized effect.

A route is one of the runtime's profile modes: `unattended` for a scenario that fires a run,
`delegated` for one tagged `delegation` or expecting `delegate_task`, `chat` for the rest. A group
of the spec names its cells and, optionally, the tags, routes and splits of the scenarios it runs;
groups run in the order written, each case recording the cell it ran as (`cell` in `report.json`).
A group marked `advisorRoutes` runs last, only on the routes where the chosen Haiku effort trails
the advisor rule's reference by its `gap`.

1. `--probe` runs every cell once on every `budget.probeEvery`-th development scenario of its
   group and writes `probe.md` and `probe.json`: each cell's cost and time per trial, scaled to the
   trials the gate plans, against the spec's `maxUsd` and `maxHours`. Over budget, or with a cell
   it could not price, it exits non-zero and the gate refuses to start without `--over-budget`.
2. `--budget <probe>` runs the gate. It stops once it has spent the projection times
   `budget.overrun`, and draws no decision from a run cut short, for any reason. The cap is checked
   as each case finishes, so it can be passed by the cost of the cases still running, up to
   `--parallel` of them. The run directory holds `gate.md` and `gate.json` (each rule with the
   figures it read), a `summary.md` of cell figures by split with no scenario detail,
   `development/` (the development split's report, transcripts and call files) and `sealed/`
   (the full `report.json`, the held-out and adversarial splits and every log). Only
   `development/` and `gate.md` are handed over; `sealed/` never is.
3. `--confirm <gate>` runs the configuration the gate chose, and the configuration it replaces,
   on the held-out scenarios of each route, and writes `confirm.md` and `confirm.json`: whether
   each route's choice holds, the description check on the scenarios both runs hold, and the
   before and after figures. Each model run 1 recommended for code mode is judged again on the
   cell the configuration runs it at (its held-out pairs, and its adversarial pairs again). Both
   `gate.md` and `confirm.md` end with "Recommended settings": per route, the reasoning effort
   and the advisor, and per model whose code-mode rule held, Scripts (and, where the write
   safety held, Scripts may propose writes). Its own run sits under `sealed/` in its directory.

## Agents

`--agent` names an agent the bench keeps in AI Control from `services/tms/deskbench/agents.yaml`:
before every run each is created when missing and changed when the file changes, through the
agent definition service, named `Bench · <name>`. A spec gives its `template`, `instructions`,
`tools`, `delegates`, `maxToolCalls`, `autonomyCeiling` and `dataAccess` (the agent's Data
access, `Internal` unless set). `Restricted` is saved only when the bench user's role reaches it,
as in AI Control; the EDI desk runs at `Restricted` so it can read the X12 it previews, as an
organization that raised it would.

## Scenarios

Scenarios live in `services/tms/deskbench/scenarios/*.yaml`. A file holds `defaults` (agent,
user, provider, tags) and a list of `scenarios`; each scenario is a list of steps, and each step
either says something or decides what the agent proposed.

```yaml
defaults:
  tags: [dispatch]
scenarios:
  - name: late-loads-today
    description: A dispatcher's opening question of the day.
    steps:
      - say: "anything running late rn? need to know before my 9am"
        page: { path: /shipments, title: Shipments }
        expect:
          calls: ["shipment_list|dispatch_get_board"]
          noCalls: [invoice_build_run]
          maxCalls: 6
          maxFailedCalls: 0
          replyIncludes: ["late|behind|delayed"]
          replyExcludes: ["/shp_[0-9a-z]{26}/"]
          facts:
            - sql: select count(*) from shipments where status = 'Delayed'
              note: the number of delayed shipments
          rubric: Names each late load with its customer and how late it is.
      - say: "put a note on the first one that the receiver moved us to tomorrow 2pm"
        expect:
          proposals: [{ tool: shipment_add_comment }]
      - decide: approve
        expect:
          executed: true
          replyIncludes: ["added|saved|note"]
```

| Field | Checks |
|---|---|
| `calls` | each entry was called and not refused; `a\|b` accepts either |
| `noCalls` | none of these was called, refused or not |
| `noDelegateCalls` | no delegate called any of these, refused or not; the conversation's own calls are not counted, so a write it proposes itself still passes |
| `maxCalls`, `maxFailedCalls` | tool calls in the step, and those refused (`denied`, `invalid`, `duplicate`, `over_budget`, `failed`, `unregistered`) |
| `replyIncludes`, `replyExcludes` | case-insensitive text; `a\|b` alternatives; `/pattern/` is a regular expression |
| `noRecordIds` | the reply shows no internal record id, by the same detector the reply pass uses (`shared/recordids`), so an `artifact:` link is not counted |
| `formula` | on a formula scenario: the studio's draft expression, after the step, contains each entry (`a\|b`, `/pattern/`) |
| `refused` | the scope guard refused (`true`) or answered (`false`) |
| `proposals` | a proposal of `tool` was filed whose arguments contain `args` (strings compared case-insensitively, numbers by value) |
| `noProposals` | nothing was proposed |
| `executed` | on a decide step: every approved change ran without an error |
| `facts` | the first column of each row the query returns is stated in the reply (numbers within the grounding guard's rounding, a whole count up to twelve also as its word, text by containment); `any: true` needs one row |
| `maxSeconds` | the step's turns took at most this long |
| `rubric` | not checked; printed beside the step for whoever reads the transcript |

A `say` step may carry `attach:`, a list of files (a path, read from the working directory) or
stored documents (`doc_...`, downloaded and uploaded again as a copy). Each is uploaded to the
conversation as the Desk's composer uploads it (`documentservice.Upload`, profile
`assistant_attachment`), and the step waits until document intelligence has read it, so the
turn sees the file the way a person's does once its upload has finished. A relative path is
read from the scenario file's folder. Fixtures live in `deskbench/fixtures/attachments/` and are
made up: no real broker, carrier or person. Each rate confirmation there is one entry of
`variants.json` (an entry may `extend` another and change a few values) filled into
`rate-confirmation.html` and printed by Gotenberg's Chromium (`node render.mjs` from that folder,
with the local stack up), so it is laid out the way a browser-printed confirmation is; the PDF
reader's own tests read them too. A trading partner's EDI implementation guide is printed the same
way, from `edi-guide.html` and `edi-guides.json` (a version and its revision).

A scenario with a `formula:` block talks to the Formula Studio's own assistant instead of a Desk
agent: the bench opens the studio's page thread (`OpenPageThread`, origin `Formula`), on the
template named by `template` (looked up by name; leave it out for a new formula, and `type`
sets the template type, `FreightCharge` by default), and sends the studio's draft with every
message, as the page does. Each `formula_propose` edit the agent makes is applied to that draft
before the next message, the way the studio applies it, so a follow-up edits what the agent
wrote, and the transcript shows the draft after every step.

```yaml
  - name: formula-edit-existing-per-mile
    formula: { template: Per Mile }
    steps:
      - say: "change this so every stop after the second one adds 75"
        expect:
          calls: ["formula_propose"]
          formula: ["totalDistance", "totalStops"]
```

A scenario with `palette: true` asks the way the command palette does: the case opens with
`StartAsk` on its first message, so the palette's own choice of agent answers it (the
general assistant; `agent` is refused), on a quick question's conversation holding every read
the person may use, and every message is sent on the `Palette` surface. The conversation is
retitled `[bench] …` like any other, so `desk clean` finds it. The palette sends the records
its search found that the question names as `mentions`; give them on the step to test that
path, or leave them out to test Nova finding the record itself.

Every step also checks that each of its turns finished, that no artifact link restarted its reply (a left-out artifact the runtime names at the end without asking again is not counted), and that the grounding guard did not
have to rewrite or annotate its reply ("reply grounded without a correction": a reply that named
what a filed change does not hold, or stated money or a percentage nothing the conversation read;
the detail lists the figures and fields). A decide step takes `decide: approve`
or `decide: reject`, an optional `note` (on a rejection, what the agent is told), `tool` to
decide only that tool's proposals, and `modifications` to approve with changed arguments.
`page` and `mentions` take the shapes the Desk sends (`agent.PageContext`, `agent.EntityRef`).
A seeded record's id changes with every reseed, so a mention may give `find:` in place of `id`:
one `SELECT` whose single row is the id, run before the message the way a fact query runs
(`find: select id from shipments where pro_number = 'SEED-PAY-002'`).

A fact query runs in a read-only transaction under the bench user's tenant, so row-level
security applies and a query sees only that organization; it must be one `SELECT` (or `WITH`)
and is stopped after ten seconds. Facts are what keep a scenario honest without hard-coding
values the seed may change: the question is "is the number the agent said the number the
database holds", not "is it 4".

A scenario can put the data where it needs it before it starts with `setup`: single
`UPDATE`, `INSERT` or `DELETE` statements run in one write transaction under the tenant's
row-level scope. The loader accepts a statement only when it begins with `UPDATE`, `INSERT`,
`DELETE` or `WITH`, holds one statement, and contains none of `DROP`, `ALTER`, `TRUNCATE`,
`GRANT`, `REVOKE` or `CREATE`; that is a check on the text, not a sandbox, and a `WITH` can still
write to any table the role may write to. What keeps a statement to the bench user's organization is row-level
security, so the bench refuses to run any setup unless its database role is neither a superuser
nor one that bypasses row-level security and `row_security` is on. The seed is written
once and ages; the bench approves real writes. A load whose windows have passed, time off the
seed put on the day a scenario asks for, or a lane that has come to match several loads turns a
correct refusal or question into a failed case, so a write scenario rolls its load's windows
forward from `now()` rather than trusting the seed's dates. Pair `setup` with `exclusive` when
two scenarios touch the same rows. The scenarios of one `exclusive` group run as one chain, never
beside each other: for each provider and repeat in turn, each scenario of the group in file order.
A scenario may therefore build on the one before it (`edi-guide-revised` updates the partner
`edi-guide-new-partner` saved), whatever `--provider`, `--repeat` or `--parallel` asked for.

A step can carry `setup` too: statements of the same kind, run just before that step says or
decides, so a scenario can change the world between a proposal and its approval (a stale
approval). A fire step cannot; its setup is the scenario's.

### Labels and splits

Every scenario is in one of three splits: `development` (tune against it), `held-out` (look at it
only to confirm a change, and decide on it) or `adversarial` (text that tries to steer the agent:
an injected instruction in a message, an email or an EDI file, a probe of the analysis sandbox).
Only `adversarial` is written by hand, as `split: adversarial` on the scenario or in a file's
`defaults`; the loader refuses any other `split` written in a file, because the other two are
assigned by a fixed rule (`internal/modules/agent/domain/evalsplit`):

1. **Strata.** A scenario's stratum is its desk and whether it is chat or unattended. An
   unattended scenario's desk is the template its `fire` step names; a chat scenario's is its
   `agent` without the `Bench · ` prefix (or the first step's agent), `Palette` for a palette
   scenario, `Formula studio` for a formula one, and `General assistant` when none is named.
2. **Quota.** Of a stratum's non-adversarial scenarios, a quarter rounded to the nearest whole
   one is held out, and at least one when the stratum has two or more: 2–5 scenarios hold out
   one, 6–9 two, 10–13 three. A stratum of one is development only.
3. **Order.** Within a stratum the scenarios are ranked by the 64-bit FNV-1a hash of their name,
   and the first ones by that rank are held out. The rule reads nothing else, so it gives the same
   answer for the same names whatever order the files are read in, and loading one file assigns
   what loading them all would (the loader ranks every scenario in the file's directory).

The `find_tools` cases of the eval gate (`agentevalgate/evals/toolselection*.yaml`) are split by
the same rule over the two suites together, keyed by the request. A case's desk is the template
that lists its first expected tool among its starter tools, the one with the fewest starter tools
when several do, and its mode is that template's trigger (chat, or event and schedule for
unattended); a case whose tools no template starts with is in `No desk`.

What each rule assigns is recorded: `deskbench/splits.yaml` for scenarios and
`agentevalgate/evals/toolselection.splits.yaml` for selection cases, each listing the held-out
keys by stratum. `TestScenarioSplits_MatchTheLedger` and `TestToolSelectionSplits_MatchTheLedger`
fail when the assignment moves, naming what joined and what left the held-out set. Adding a
scenario or a case moves its stratum's quota, and renaming or merging tools can move a case to
another desk, so a move is expected; review it before rewriting the ledger (`go test -count=1 -run
TestScenarioSplits ./internal/deskbench/ -update`, and `-run 'AgainstFloors|Splits'` with
`-update` in `agentevalgate`). A scenario that joins the held-out set after living in development
may have been tuned against: write a new scenario for that desk instead of keeping it held out.

**Using the split.** Edit descriptions, search terms, schemas and prompts against development
cases, traces and `findings.yaml`, never against a held-out case. `trenova desk run` reports every
cell by split (`--split held-out` runs only that split), so pass rate, pass^k, tool-selection
precision and recall, and argument validity are given for held-out scenarios apart from the
rest. The eval gate logs `find_tools` recall@5 and top-1 on held-out cases apart, and gates them
on their own floors (`heldOut` in `toolselection.floors.json`) beside the floors over every case.
It prints the misses of development cases only; a held-out miss is printed only when the held-out
floor fails. The Agent Evals workflow writes both to the run's summary.

`expectedTools` names the tools a correct run calls (`a|b` for either one); `forbiddenTools`
names ones it must not call. Left out, expected tools are taken from the scenario's `tools` and
`proposals` checks and forbidden ones from `noTools` and `noProposals`, leaving out any tool
expected elsewhere in the scenario.

### Unattended runs

A `fire` step starts a run the way production does, through
`AgentRunService.StartForDefinition`, instead of saying something:

```yaml
steps:
  - fire:
      agent: DetentionDesk            # the template
      event: detention.notice_due     # or schedule: true
      subject: >                      # one row: the subject's id
        SELECT id FROM detention_occurrences WHERE ...
    expect:
      run: AwaitingDecision           # the run's final status
      flagged: true                   # it raised an exception or flagged for review
      proposals:
        - tool: detention_send_notice
      db:
        - sql: SELECT count(*) FROM agent_exceptions WHERE run_id = '{{run}}'
          equals: "1"
```

A fire step may also shape the agent for that one run, the way a person would in AI Control:
`ceiling` sets its autonomy ceiling (`Propose`, `ActWithApproval`, `AutoExecute`) and
`toolTiers` sets a tier on named tools as a person's setting (`agent_raise_exception: Propose`), on
tools the template holds or the core tools. Both apply while the case holds the agent and are
put back to the template's own ceiling and no tool tiers when it ends; the next sync clears any
a crashed case left. `stopWhileWaiting: true` cancels the run's workflow once it is
`AwaitingDecision`, before the bench decides anything, waits for the run to close and grades
what it left: the run's status, its proposals read again, and a check that it was waiting when
it was stopped.

```yaml
      - fire:
          agent: BillingException
          event: billing_queue.item_exception
          subject: select id from billing_queue_items where number = 'BQ-BENCH-0001'
          ceiling: Propose
          toolTiers: { agent_raise_exception: Propose }
          stopWhileWaiting: true
        expect:
          run: Failed
```

An event run has no actor and its subject type is taken from the event kind, as the publisher
starts it. A scheduled run gets the definition's own trigger, a unique slot and the agent as its
actor. Each template is run by `Bench · <Label> (unattended)`, created from the template's starter
instructions, tools, events and data access, with learning off. It stays disabled except while
its own case runs, so real development events never start it. The bench waits until the run is
`AwaitingDecision`, `Completed`, `ShadowCompleted` or `Failed`, grades it, then rejects what it
proposed, waits for the run to settle and cancels it if it has not. `{{run}}` and `{{subject}}`
are replaced in checks with the run's and subject's ids. Unattended scenarios live in
`scenarios/unattended-*.yaml`, are tagged `unattended`, and cover every event template and
LoadMonitor.

### Writing them

- **Write like the people who use it.** Lower case, abbreviations, half-sentences, two
  questions at once, a customer's nickname, a load named by its destination. A scenario that
  reads like a specification measures how the agent answers a specification.
- **One behaviour per scenario**, with the checks that would catch it failing. Put what cannot
  be checked mechanically in `rubric`.
- **Scenarios are regression cases.** When a run finds something and it is fixed, keep the
  scenario that found it, so the next change to a prompt or a description is held to it.
- **Say dates relative to the day the run happens** ("four days later", "next Tuesday"), never
  a weekday that may be today.
- **Run more than once.** A model's answer varies; `--repeat 3` turns one pass into a rate,
  which is what a comparison can trust.

## The loop

1. `task desk-bench` (or `trenova desk ask …` while exploring).
2. Read `summary.md`: failing checks, then the tools the harness refused. A tool refused again
   and again for the same reason is a schema or a description that misleads.
3. Open the case in `transcript.md`. For a wrong answer, compare what the tool returned with
   what the reply said: wrong data is a tool's bug; right data read wrongly is a description,
   a result's shape or the prompt. For a wrong tool, read the call's `prompts/` and `tools/`
   files: what the model was offered is what it chose from.
4. Fix it, then run the scenario again with `--repeat`, and the whole set before keeping the
   change: a prompt edit that fixes one question can break another, and the comparison says so.

## Routing

`trenova desk route` (`task desk-route`) scores where questions would go before they are
asked: it asks the Desk's agent matcher (`MatchAgent`, the `myAgentMatch` query) each
labelled question in `deskbench/routing/*.yaml`, from the agent it names, as the bench user.
No turn runs and nothing is written, so it is quick to run after changing a tool's
description, search terms or an agent's tools.

```yaml
cases:
  - name: billing-ar-aging-from-dispatch
    from: DispatchAssistant              # the agent the question would go to now
    ask: "whats the ar aging look like for acme rn"
    expect: Receivables                  # an agent, by template, name or id; | for alternatives
  - name: billing-send-it-about-an-invoice
    from: DispatchAssistant
    ask: "send it to them again please"
    about: [invoice]                     # the kinds of record it names or is asked from
    expect: BillingAssistant|Receivables
  - name: dispatch-hold-stays
    from: DispatchAssistant
    ask: "put the walmart load on hold"
    expect: current                      # the agent asked from holds the tools
```

`expect` also takes `none` (tools match but none of the person's agents holds the best of them) and
`nothing` (no tool matches). A case expecting an agent passes when that agent is suggested,
when it is the agent asked from and is kept, or when the agent asked from keeps the question
because it would hand it to that agent through `delegate_task` (the report reads
`current (Billing exceptions, handing it to Compliance desk)`). Name agents by template, as
the cases do, so a reseed keeps them; `TestShippedRoutingCases_NameAgentsByTheirTemplates`
refuses one that is not. When the label itself was wrong, change `expect` and say why in the
case's `note`.

The bench's own agents (`Bench · …`, from `deskbench/agents.yaml`) are left out of a routing
run, as the agent asked from and as suggestions, and `route` does not sync them: several of
them share a shipped template (two are `DispatchAssistant`s), so a case asked from
`DispatchAssistant` would otherwise be asked from whichever sorts first by name and the
scores would measure the bench's agents, not the organization's. A case that fails while
none of the agents it expects is an enabled chat agent says so (`no enabled chat agent is
CustomerAssistant`), for an agent that is turned off or one, like an `IntakeDesk` agent, that
is started by inbound messages and never by the Desk. The seeded Customer service agent is
off until someone reviews it, so the cases that need it accept `none` too: with it off, no
agent the person may use holds the best tool for them.

It is a profile-selection bench: every case records the job profile it expected
(`expectedProfile`, by template; `current` is the agent asked from) and the one the match
left it with (`selectedProfile`), and `routing.md` opens with their confusion matrix, rows
expected and columns selected, with each profile's recall and precision. A case whose
expectation allows several profiles counts on the diagonal when it passes. Since a
conversation reaches the tools of the person's other agents, an off-diagonal cell is a
question that starts on a near neighbour and finds the tools with `find_tools`, not one that
cannot be done; the matrix shows which neighbours the match confuses.

Each run writes `routing.md` (the score, the confusion matrix, the cases that changed since
the previous routing run, and every case's expected and actual answer with the tools that
matched) and
`routing.json` to `.deskbench/runs/<time>-route/`, and exits non-zero when a question goes
the wrong way. `--compare` takes a run directory or a `routing.json`, `''` for none. The bench
lifts the per-person limit on matching embeddings for its own matches only, so a long run is
ranked as a person's first question is, by keyword and embedding together.

Embeddings are used only when the organization has an embedding provider and its retrieval
settings (AI Control → Retrieval) name an active embedding model; otherwise every case is
ranked by keyword alone, which `ai_usage_records` shows as no `AgentRecommendation`
embedding rows. The first match after a reseed, or after a tool's description or search
terms change, embeds the catalog in the background and may be ranked by keyword while it
does, so run one case (`--name`) before a run you mean to compare.

## Findings

Every problem a run turns up goes into `services/tms/deskbench/findings.yaml`, fixed or not,
so one that is not fixed straight away is not lost and a fixed one can be traced to the run
that found it. Each finding has an `id` (`DB-001`, …), `title`, `status` (`open`, `fixed`,
`wontfix`), `severity` (`critical`, `high`, `medium`, `low`), `area`, `found` (date and run
directory), the `models` and `scenarios` it showed on, and `problem`, `evidence`, `cause`, `fix`,
`files` and `fixedIn`. A finding is marked `fixed` only after a bench run shows it fixed, and
the scenario that found it stays in the suite.

```bash
trenova desk findings                                   # open findings, markdown
trenova desk findings --status all --out findings.md    # everything, to a file
trenova desk findings --severity critical,high --format csv
trenova desk findings --area anthropic --format json
```

## Known limits

- A turn the application starts long after the last one (a wait resolving) is not followed; the
  bench stops waiting a few seconds after the last turn ends. Scheduled and event runs are
  followed only when a `fire` step starts them.
- Approved writes are real. Run against a seeded development database, and `task db-reset`
  when a run has moved the data too far.
