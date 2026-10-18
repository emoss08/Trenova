# Desk harness review

Status: **Phase 1 (audit) in progress.** Phases 2 (research) and 3 (analysis) have not started.

This is an architecture review of Desk, Trenova's agent harness. It is not a change
proposal for any one file. Numbers come from the source at the commit this was
written on; where a number is an estimate, the method is given.

## 1. Audit

### 1.1 Agent topology

Desk is **not** a single agent with a flat tool list. It is a set of configured agents,
a pre-turn router, in-turn tool discovery, and one level of delegation.

| Layer | What it is | Where |
|---|---|---|
| Agents | 29 shipped templates (`DispatchAssistant`, `BillingAssistant`, `GeneralAssistant`, `Receivables`, `EDIDesk`, …). An organization creates agent definitions from them in AI Control, each with its own tools, instructions, autonomy ceiling, tool budget (default 12, ceiling 64) and delegate allowlist. | `domain/agentdefinition/templates.go`, `definition.go:38-47` |
| Pre-turn router | `myAgentMatch` / `agentruntime.MatchAgent`. Before a question is sent, the question is ranked against tool names, descriptions and search terms by keyword plus embedding (reciprocal-rank fusion). The result is one of `BetterAgent`, `CurrentAgent`, `NoAgent`, `NoMatch`, and the Desk switches agents, offers a switch, or only suggests one, depending on the person's setting. No model is called. | `agentruntime/match.go`, `matchpool.go`; agent-runtime.md "Matching a question to an agent" |
| In-turn discovery | `find_tools` lets the agent load tools it holds but was not shown, and names a delegate or another agent when it finds nothing it may call. Palette questions hold every query tool the person may use, disclosed through `find_tools`. | `agenttoolcatalog/hybrid.go`, `agentruntime/discovery*`; agent-runtime.md "Quick questions from the palette" |
| Delegation | `delegate_task` hands a task to an agent on the caller's allowlist (at most 8 delegates, 3 tasks per turn, one level only). The delegate runs inline in the same workflow, as the same person, with its own tools and autonomy. | agent-delegation.md |
| The loop | `agentruntime.Service.Drive` (`runner.go:150`): a model call, then the tool calls, until the tool budget is spent or the reply has no calls. Each model call and each tool call is a Temporal activity. | `runner.go`, `jobs/agentflow` |

After the model stops calling tools, the loop runs several checks that can ask the model
again before the reply is accepted (`runner.go:240-262`):

- an empty reply is asked again (`maxReplyRetries` = 1, `replyguard.go:19`);
- `holdToCommand` (`command.go:158`): a slash-command turn (`/watch`, …) that ends without
  calling a tool that honours the command is sent back once with a correction;
- `reground` (`groundingguard.go:118`): a reply that claims something the tool results do not
  support is asked again (the grounding guard);
- `passReply` (`replypass.go:49`): the reply is corrected in place, with no new model call;
- `placeArtifacts` (`placement.go:46`): a reply that names an artifact outside a sentence, or
  invents or misses one, is written again once, then fixed mechanically.

Every one of these costs a model call when it fires. That matters for Phase 3's
efficiency question.

(Sections 1.2 to 1.5 are being filled in from parallel audits.)

### 1.6 Safety of write actions

Write safety is the strongest part of the harness. It is declared in code, checked at boot and
in CI, and the safety doc is generated from it. Nothing in this review proposes weakening it.
Any new interaction pattern (code-as-action included) has to go through the same chokepoint.

**Classes and tiers.** Each tool declares where its effect lands: none, personal, internal,
customer-visible, driver-visible, external recipient, or money (`domain/agent/toolsafety.go:11-19`).
Each call gets one of three tiers: Propose, ActWithApproval, AutoExecute
(`domain/agent/enums.go:326-328`). `agenttoolpolicy.Decide` (`decide.go:48-84`) takes the lowest
of these limits and records each reason in `HeldBy`:

- the tool or agent tier;
- the agent's autonomy ceiling;
- the tool's maximum tier;
- the class cap (customer, driver and external writes are never past ActWithApproval);
- any per-call condition.

The runtime can lower the tier further:

- taint lowers it to ActWithApproval;
- web or partner text read earlier in the turn lowers every later write to Propose
  (`agentruntime/extensions.go:116`);
- a write outside the agent's business hours is held (`changelimits.go:101`);
- the organization's tool rules apply, and if they cannot be read the tool falls back to
  Propose (`dispatch.go:910-933`).

A tool whose policy is inconsistent fails at boot (`agenttoolpolicy/validate.go:110-181`).

**Counts** (from the generated `docs/engineering/ai-tool-safety.md`, which lists 646 tools):

| Kind | Automatic | Ask first | Propose | Total |
|---|---|---|---|---|
| Reads | 215 | 0 | 0 | 215 |
| Writes | 102 | 194 | 135 | 431 |

The 102 automatic writes break down as 87 internal, 7 money, 5 personal, and 3 that classify
each call. 13 tools are marked "a person always decides".

**What dispatch checks, in order** (`dispatch.go:95-339`):

1. The person's permission.
2. The tool's whole schema. Every object is closed and every enum names its source.
3. How many records the call touches (500 by default; a bigger call is refused).
4. The tier.
5. The tool's own validator, which runs at every tier.
6. A duplicate of a proposal that is still pending is refused.
7. A baseline preview. A write that would fail is not filed, and the model is told why.
8. Simulation or the daily tool budget, for a call that runs automatically.

**Approvals are bound to what the person saw.** The preview digest is SHA-256 over these
inputs (`domain/agent/preview.go:368-440`):

- the proposal and the tool;
- the parameters as they would run;
- coverage and the withheld count;
- the target record's version;
- the changes, filtered by what the reader may see.

At decision time the change is previewed again:

- a target record that changed since filing is refused as stale;
- a digest that no longer matches is a conflict;
- an approver's edits can never point the write at another record;
- the write executes as the approver, with the approver's permissions
  (`proposalexecutor/executor.go:209-214,765-790`);
- typing "approved" decides nothing; `request_decision` opens the approval box.

**Retries are at most once.** Each call is claimed in `agent_run_steps` before it runs, under a
key hashed from the owner, tool, canonical arguments and an ordinal (`stepkey.go:46-58`):

- a settled claim replays its outcome;
- a call whose outcome is unknown is reported to the model as unknown;
- an unwritable ledger refuses the call (`agentruntime/guard.go:64-206`).

**Taint.** A run opens tainted when it starts from outside content: an email, a document, an EDI
file, an attachment, a tainted memory, or an inherited conversation or delegator. While it is
tainted, every customer, driver, external and money write waits for a person. Tool output is
fenced in `<untrusted_data>` (`fence.go:14-33`). The red team checks 15 cases against 14
invariants (`agentredteam/invariants.go`).

**Write coverage.** `writecoverage.yml` has 1,170 entries: 643 GraphQL mutations and 527 REST
writes.

- 459 are covered by a tool.
- 711 are exempt, each with a reason. The largest groups are configuration (270),
  agent administration (107) and security (88).
- 0 are pending.

So every write meant for an agent has a tool. Taken together, Desk's write surface is
431 write tools.

**Gaps worth a decision** (they are not the subject of this review, but they are recorded here):

1. Money's class cap is Automatic, so 7 money tools (for example `vet_carrier` and settlement
   adjustments) can run unattended in an untainted run.
2. The preview digest is optional. A decision without one goes through and is recorded as
   unreviewed (`agentdecisionservice/preview.go:97-109`). Rows of a batch past the first 5 are
   approved unreviewed.
3. The scope classifier fails open after 3 s. The comment at `agentguard/scope.go:46` says
   "fails closed".
4. Approving in AI Control checks `agent_proposal:update`, but the audit row records
   `OpApprove`.

**Where safety costs efficiency.** This is relevant to Phase 3:

- Every refused call spends the tool budget (12 per turn by default). Refusals include schema
  failures, validation failures, would-fail previews, duplicates and unparsed JSON.
  `findings.yaml:212` and `:322` record runs that ran out of budget this way.
- The guards that rewrite a reply (grounding, artifact placement, loop or empty retry, command
  hold) each cost a model call, and they can stack on one turn.
- Closed schemas catch an invented field early, but each catch is a round trip.

### 1.7 Evals, traces and recorded failures

Desk has more eval machinery than most production harnesses. It does not yet measure the
three things this review is about across providers: which tool a model chose, whether the task
succeeded, and steps and tokens per task.

**What exists**

| Tool | What it measures | Size | Paths |
|---|---|---|---|
| Desk bench (`trenova desk run`) | Real turns on a seeded database through the production workflow. Deterministic checks: `calls`, `noCalls`, `maxCalls`, `maxFailedCalls`, `proposals`, `executed`, `replyIncludes/Excludes`, `facts`, `db`, `maxSeconds`. A provider matrix (`--provider A --provider B --repeat N`). | 134 scenarios, 202 steps, 20 files | `services/tms/deskbench/scenarios/`, `internal/deskbench/checks.go` |
| Rubrics in the bench | 142 of the 202 steps carry a rubric. **None is scored**: each is printed beside the step. | – | `desk-bench.md:203` |
| Routing bench (`trenova desk route`) | The pre-turn agent matcher. No model is called. | 33 labelled questions | `deskbench/routing/desk.yaml` |
| Eval gate (CI) | `find_tools` / preselection **ranking**. It does not measure what a model chooses. | 557 keyword cases, 631 hybrid cases (557 plus 74 paraphrases) | `app/agentevalgate`, `evals/toolselection.floors.json` |
| Quality suites (production) | Replays of captured cases against the live agent. The score is 0.7 deterministic (toolChoice 0.35, arguments 0.25, proposals 0.20, …) plus 0.3 from an LLM judge; a case passes at 0.8. Regression is measured against the median of recent runs. | Cases captured from decided proposals and replies given a thumbs-up | `agentscoring/scoring.go`, `agentqualityservice/judge.go` |
| Red team | 15 injection cases against 14 invariants. The live baseline was **never measured** (`live-baseline.json`: `"cases":0`). | 15 cases | `app/agentredteam` |
| Findings log | Every problem the bench has found. | 189 findings: 176 fixed, about 12 open, 1 won't fix | `deskbench/findings.yaml` |

**Ranking floors in CI** (`evals/toolselection.floors.json`). This number matters most for
the tool-count hypothesis:

| Ranker | Cases | Top-1 | Recall@5 |
|---|---|---|---|
| keyword | 557 | 0.98 | 1.00 |
| hybrid (keyword + embedding, includes paraphrases) | 631 | **0.78** | **0.91** |

A disclosed turn opens with 8 preselected tools. So on paraphrased requests, roughly 1 in 10
turns can start without the right tool in view: recall@5 is 0.91, and recall@8 is unmeasured
but must be at least that. Those turns must recover through `find_tools`.

**Which models were tested.** Only two models have ever been benched: `claude-haiku-5-5` (named
in 119 findings) and `gpt-6-luna` (named in 83). From 2026-10-15 on, only Haiku was run. The
latest full-bench pass rate on record is 83 of 107 (`603e746dd`). Up to DB-157, both models ran
as a matrix, and the findings split evenly between them: 48 Haiku only, 43 Luna only, 40 both.

**What the findings say, by cause.** These counts are the audit's own classification, one
primary category per finding.

| Primary cause | Findings | Haiku only | Luna only | Both |
|---|---|---|---|---|
| A product or backend bug, or a missing tool, that the bench exposed | 58 | 28 | 6 | 12 |
| A description or result shape that misled the model | 33 | 14 | 7 | 11 |
| Bench infrastructure, a scenario or seed data that went stale | 24 | 6 | 2 | 5 |
| Tool discovery or selection | 16 | 6 | 8 | 2 |
| Grounding, invented figures, reply form | 15 | 9 | 1 | 2 |
| Argument or schema errors | 14 | 7 | 5 | 2 |
| Loops, running out of budget | 13 | 1 | 10 | 2 |
| Context or prompt | 9 | 5 | 2 | 2 |
| Provider or adapter | 7 | 3 | 2 | 2 |

**How failures differ by model.**

- **gpt-6-luna** over-searches. It rewords `find_tools` until the budget is spent, fills every
  optional parameter, and sends placeholders: `"x"`, `"n/a"`, zeros for pieces and
  temperatures, and ids made up by swapping a prefix. Part of the cause was the Responses
  adapter not sending `strict:false` (DB-014, DB-080).
- **Haiku** states figures that no tool returned (DB-149, still open), files a corrected
  proposal next to the original one (DB-056, critical), and under-uses a capability it holds
  (DB-175: `analyze_data` was used in 2/4, 0/4 and 0/4 runs).
- **Both models**: most failures were the harness telling the model something false. DB-008
  is the canonical case. `search_shipments` said it matched customer and city, but only matched
  PRO and BOL numbers. Both models searched "freshhaul", found nothing, and opened shipments one
  by one until the budget ran out. After the fix, `search_shipments` calls fell from 107 to 44
  and 5 Luna cases flipped to pass.

**What production can measure today.**

- Traces: an `invoke_agent` root per turn, `chat {model}` spans with cache-normalized tokens,
  and `execute_tool` spans with their outcome.
- The `ai_work_units` rollup, per turn, by model and provider: model calls, tool calls, tool
  failures (split between model-caused and tool-caused), tokens and cost.
- `assistant_tool_outcomes_total{tool,verdict}`. It has **no model or provider label**.

So steps, tokens and cost per task can already be read by model. Tool-selection accuracy cannot,
because there are no labels. Task success is only proxied by turn status, proposal decisions and
thumbs.

**Measurement defects that bias any comparison across providers:**

1. Cache tokens are dropped. `ChatCompletionResult` has no cache fields
   (`ports/agentports/agentcompletion.go:148-180`), and the Anthropic adapter reports only
   uncached `input_tokens` (DB-002: a Haiku turn reads "4 in"). OpenAI's input count includes
   its cached tokens. Both the bench matrix and `ai_work_units.input_tokens` therefore undercount
   Claude's input heavily, relative to OpenAI. The trace spans normalize this; the rollups do not.
2. The bench matrix sums tokens across repeats instead of averaging them, and has no cost column
   (the local Anthropic provider has no prices set).
3. `--provider default` can fail over mid-run, so one row may mix models.
4. 13 findings are marked fixed while their `fixedIn` still says "to verify in a run". Two
   scenarios had their call limits raised instead of the behaviour being fixed (DB-118: 8 → 10;
   DB-121: 8 → 11).

### 1.4 Provider layer

Paths in this section: MA = `services/tms/internal/infrastructure/agentcompletion/modeladapter/`,
CR = `…/agentcompletion/completionrouter/`, RT = `services/tms/internal/modules/agent/app/agentruntime/`.

**Protocols, not vendors.** There are four adapters (`MA/adapter.go:199-206`):

| Protocol | Covers | Notes |
|---|---|---|
| `AnthropicMessages` | Claude direct, Bedrock, Vertex, Foundry, Claude Platform on AWS and on Google Cloud | Official Go SDK; requests are signed in-house; wire requests pinned in `testdata/anthropic_wire` |
| `OpenAIResponses` | OpenAI `/v1/responses` | Encrypted reasoning is replayed; `strict:false` is always sent (DB-080) |
| `OpenAIChat` | vLLM, SGLang, LM Studio, llama.cpp, OpenRouter, Groq, Together, Fireworks, NIM, DeepSeek, and **Gemini through its OpenAI-compatible endpoint** | There is no native Gemini adapter |
| `Ollama` | Native `/api/chat` | Ollama has no tool call ids, so two calls to the same tool cannot be told apart |

**How a provider is chosen.** Candidates are the organization's providers assigned to the
`AssistantChat` task, ordered by priority. The thread's model picker, or else the agent's
`PreferredProviderID`, moves one provider to the front, and an explicit pick pins it
(`CR/chat.go:69-71`).

Failure handling:

- A provider is rested for 1 minute after 3 failures, and the rest is shared through Redis.
- A reply that dies partway is restarted on the next provider, at most twice.
- An empty reply counts as a failure.
- A refusal never fails over.

**The system prompt is the same for every provider and model.** Per model, only the following
differ:

- reasoning settings: Anthropic adaptive effort or a thinking budget; `reasoning_effort` on
  Chat; reasoning effort plus summary on Responses; `think` on Ollama;
- the max-token floors;
- schema stripping: `toolschema.ForModel` strips `x-*` keys; `ForPortableModel` also strips
  `examples` and custom formats, for Gemini;
- a long tail of wire workarounds:
  - Gemini thought signatures;
  - inline `<think>` blocks from Qwen and R1 distills;
  - `<tool_call>` text markup from servers that have no tool parser;
  - streams that put every call at index 0;
  - Haiku's empty-argument replay (DB-001).

**Tool request settings.** `tool_choice` is never sent; it is a reserved key
(`aiprovider/provider.go:804-818`). No parallel-tool-calls setting is sent, so every provider's
default applies. Every tool gains an optional `_why` rationale parameter, stripped before
dispatch (`RT/rationale.go`). The error flag on a tool result (`is_error`) reaches only
Anthropic; the other protocols see a failure only in the result's text.

**Argument repair.** Before validation, `RT/arguments.go:56-92` and `pkg/toolschema/coerce.go`
quietly fix a long list of model mistakes, and tell the model what was fixed:

- renamed keys (`shipment_id` → `shipmentId`, a bare `id`);
- strings where numbers belong, and the reverse;
- `"true"` for a boolean;
- an enum value in the wrong case;
- a single value where a list belongs;
- JSON sent as a string;
- empty optional values and placeholder ids (`"x"`, `"n/a"`, `"tbd"`), which are dropped;
- relative dates, resolved in the organization's time zone;
- an id of the wrong kind, refused by its prefix.

The list shows which models made which mistakes, and in practice it is a catalog of
gpt-6-luna's and Haiku's habits.

**Prompt caching.**

- **Anthropic** gets 3 cache breakpoints, all with the 5-minute TTL: after the last tool, after
  the stable system part, and on the last message (`MA/anthropic.go:723-789`).
- **Responses** gets a `prompt_cache_key`.
- **Chat and Ollama** rely on byte-stable JSON.

The cache is broken by:

- `find_tools` adding tools mid-turn. Tools come first in Anthropic's prompt, so everything after
  them is written again.
- Per-message re-ranking of the 8 preselected tools, on agents that hold more than 12 tools.
  The tool list, and so the whole prompt, can change on every turn.
- `fitWindow` shortening this turn's older results once the request passes 85% of the window.
- History replay: tool results stay whole only in the last 3 turns, and older ones shrink to
  320 characters (`RT/messages.go:62-71`). This shrinking is stable from turn to turn.
- Failover to another provider.
- The final tool-free answer (below).

**Context.** The window comes from the provider's own setting, or else from a table keyed by
model id (`aiprovider/contextwindow.go`). Conversations compact at 85% of the window, and
replay holds at most 120 messages. A tool result is capped at 12,000 characters, and a JSON
result keeps its fields while its lists are cut (`RT/fence.go`).

**Defects and risks found:**

1. **Sampling reaches only OpenAIChat.** The temperature policy (chat 0.3, classifiers 0)
   exists to keep small models from degenerating. Ollama, where those models usually run, never
   receives it, and neither do Anthropic or Responses (`MA/sampling.go`, `MA/openaichat.go:58-68`).
2. **Ollama never sends `num_ctx`.** The runtime assumes a 128k window, but Ollama serves its own
   default and truncates silently. `ExtraBody`, where an operator could set the window, is
   accepted on every provider but applied only on OpenAIChat. This is a likely cause of local
   models being unreliable. (I verified the field is absent from the code; I have not run it.)
3. **The tool-free final answer** (`RT/runner.go:633-660`, after the budget is spent or a turn is
   stuck) sets `req.Tools = nil` while the history still holds tool calls and results.
   - The Anthropic API has historically rejected such a request ("requests which include
     tool_use or tool_result blocks must define tools"). `finalAnswer` swallows that error and
     falls back to the canned line, so the failure would be silent.
   - It also discards the tools cache prefix.
   - `tool_choice: none` with the tools kept would avoid both problems.
   - **Unverified:** no wire fixture covers this request, and I have not seen it fail.
4. **No provider-native tool search** (Anthropic's tool search tool with `defer_loading`,
   OpenAI's equivalents) is used. Discovery is entirely Trenova's own `find_tools`.
5. **Habit fixes exist only for the two benched models.** Every one targets Haiku 5.5 or
   gpt-6-luna. Gemini, Llama, Qwen, DeepSeek and Ollama have code paths but no bench coverage.
