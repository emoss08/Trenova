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
- **A recorder on the model router.** `serviceports.CompletionService` is decorated
  (`deskbench.Recorder`) so every chat, streamed and structured call is copied as it was sent and
  as it came back: the system prompt, the messages, the tools offered, thinking, tool calls,
  tokens, latency, cost, retries, fallbacks and errors. Calls are tied to a turn by their usage
  attribution (`ThreadID`, `OwnerID`); a call that carries neither (the scope guard's
  classifier) is listed beside the turn it ran during.
- **A watcher on the workflow starter.** `serviceports.WorkflowStarter` is decorated
  (`deskbench.TurnWatcher`) so every `AssistantTurnWorkflow` started on a bench conversation, by
  the bench or by the application (a decision follow-up), is seen the moment it starts. The
  bench follows each one's stream from the first frame, waits for its result, and keeps
  waiting a few seconds for another turn the last one started.
- **No learning.** `serviceports.AgentReflectionScheduler` is replaced with one that does
  nothing, so a bench conversation is never looked back over. A conversation goes quiet after
  ten minutes, so a run longer than that fed what its early cases taught to the cases after them:
  a rejected hold taught "this person does not want a hold for missing paperwork", and the case
  that asks for that hold failed on every run after. Before a run, the tidy-up
  (`benchLeftovers`) also retires the memories earlier bench conversations taught and dismisses
  what they suggested; memories from people's own conversations are never touched. Reflection
  itself is covered by its own tests, not by the bench. For the same reason
  `serviceports.AgentTrustService` is wrapped so the bench's approvals, its end-of-case
  rejections and its failed executions are not outcomes for earned autonomy: its streaks
  promoted `transfer_to_billing` to AutoExecute on Billing exceptions, and the case that
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
trenova desk ask --thread athr_... "and the next one?"          # continue a conversation
trenova desk clean                                              # delete every [bench] conversation
trenova desk run                                                # every scenario, once
trenova desk run --repeat 3 --provider default --provider "GPT-5"  # a model matrix
trenova desk run --tag dispatch --name late --parallel 2
trenova desk agents                                             # names --agent takes
trenova desk providers                                          # names --provider takes
trenova desk findings --status all --format csv                 # export what runs have found
task desk-ask -- "how many loads did we deliver this week"
task desk-bench -- --repeat 3
```

`--provider` takes a provider's name, model id or id; `default` is the organization's own
order, which is what a person who never opens the model picker gets. `--agent` takes an
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
| `transcript.md` | each case's steps and checks, then each turn as a timeline: model calls (thinking, text, the calls asked for with their rationale), tool calls (verdict, timing, arguments, the result the model read), notable stream events (`retrying`, `reply_regrounded`, `run_tainted`, `refused`, `world_changed`, delegation, memory), proposals with their preview, and the reply |
| `report.json` | all of it, machine-readable |
| `calls/<case>/turnNN-callNNN.json` | one model call exactly as sent and received: messages, the tool names offered, the result |
| `prompts/<digest>.md` | each distinct system prompt, once |
| `tools/<digest>.json` | each distinct tool list with full schemas, once |
| `live.log` | the run as it happened, line by line |
| `worker.log` | the in-process worker's JSON log |
| `temporal.log` | the Temporal SDK's own log |

A call file names its system prompt and tool list by digest, so two calls with the same
`systemDigest` saw the same prompt and a changed digest after an edit is the edit taking effect.

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
          calls: ["list_shipments|search_shipments|get_dispatch_board"]
          noCalls: [build_invoice_run]
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
          proposals: [{ tool: add_shipment_comment }]
      - decide: approve
        expect:
          executed: true
          replyIncludes: ["added|saved|note"]
```

| Field | Checks |
|---|---|
| `calls` | each entry was called and not refused; `a\|b` accepts either |
| `noCalls` | none of these was called, refused or not |
| `maxCalls`, `maxFailedCalls` | tool calls in the step, and those refused (`denied`, `invalid`, `duplicate`, `over_budget`, `failed`, `unregistered`) |
| `replyIncludes`, `replyExcludes` | case-insensitive text; `a\|b` alternatives; `/pattern/` is a regular expression |
| `noRecordIds` | the reply shows no internal record id, by the same detector the reply pass uses (`shared/recordids`), so an `artifact:` link is not counted |
| `formula` | on a formula scenario: the studio's draft expression, after the step, contains each entry (`a\|b`, `/pattern/`) |
| `refused` | the scope guard refused (`true`) or answered (`false`) |
| `proposals` | a proposal of `tool` was filed whose arguments contain `args` (strings compared case-insensitively, numbers by value) |
| `noProposals` | nothing was proposed |
| `executed` | on a decide step: every approved change ran without an error |
| `facts` | the first column of each row the query returns is stated in the reply (numbers within the grounding guard's rounding, text by containment); `any: true` needs one row |
| `maxSeconds` | the step's turns took at most this long |
| `rubric` | not checked; printed beside the step for whoever reads the transcript |

A scenario with a `formula:` block talks to the Formula Studio's own assistant instead of a Desk
agent: the bench opens the studio's page thread (`OpenPageThread`, origin `Formula`), on the
template named by `template` (looked up by name; leave it out for a new formula, and `type`
sets the template type, `FreightCharge` by default), and sends the studio's draft with every
message, as the page does. Each `propose_formula` edit the agent makes is applied to that draft
before the next message, the way the studio applies it, so a follow-up edits what the agent
wrote, and the transcript shows the draft after every step.

```yaml
  - name: formula-edit-existing-per-mile
    formula: { template: Per Mile }
    steps:
      - say: "change this so every stop after the second one adds 75"
        expect:
          calls: ["propose_formula"]
          formula: ["totalDistance", "totalStops"]
```

Every step also checks that each of its turns finished. A decide step takes `decide: approve`
or `decide: reject`, an optional `note` (on a rejection, what the agent is told), `tool` to
decide only that tool's proposals, and `modifications` to approve with changed arguments.
`page` and `mentions` take the shapes the Desk sends (`agent.PageContext`, `agent.EntityRef`).

A fact query runs in a read-only transaction under the bench user's tenant, so row-level
security applies and a query sees only that organization; it must be one `SELECT` (or `WITH`)
and is stopped after ten seconds. Facts are what keep a scenario honest without hard-coding
values the seed may change: the question is "is the number the agent said the number the
database holds", not "is it 4".

A scenario can put the data where it needs it before it starts with `setup`: single
`UPDATE`, `INSERT` or `DELETE` statements run in one write transaction under the tenant's
row-level scope (nothing that touches the schema or grants is accepted). The seed is written
once and ages; the bench approves real writes. A load whose windows have passed, time off the
seed put on the day a scenario asks for, or a lane that has come to match several loads turns a
correct refusal or question into a failed case, so a write scenario rolls its load's windows
forward from `now()` rather than trusting the seed's dates. Pair `setup` with `exclusive` when
two scenarios touch the same rows.

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

- A turn the application starts long after the last one (a wait resolving, a scheduled
  request) is not followed; the bench stops waiting a few seconds after the last turn ends.
- Attachments are not sent: a scenario talks, it does not upload files.
- Approved writes are real. Run against a seeded development database, and `task db-reset`
  when a run has moved the data too far.
