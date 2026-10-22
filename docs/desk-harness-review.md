# Desk harness review

Status: **Phases 1–3 complete (2026-10-18).** The deliverables are:

- the audit summary (below, and §1);
- research findings (§2);
- analysis (§3);
- the recommendation (§4);
- a phased plan (§5);
- an eval plan (§6);
- references (§7).

## Executive summary

**Diagnosis.** Tool count and instruction load are real factors, but they are not the main cause.
In order of evidence, the causes are:

1. **The harness tells the model things that are wrong or incomplete.** The bench records more
   findings about misleading descriptions and result shapes (33) than about tool selection itself
   (16), and 58 more were product bugs.
2. **The tool surface overlaps.** Shipments can be read five ways; 29 job agents hold overlapping
   tool sets.
3. **Retrieval runs on text that states no goal.** Preselection ranks the raw message, or for
   unattended runs a boilerplate sentence. That is why LoadMonitor opens without the dispatch
   board.
4. **Model behaviour differs partly because of the harness's adapters.** Luna was not sent
   `strict:false`, Ollama is never given a context size, and Haiku 5.5 may hit its
   history-editing check.

**What is right in the hypothesis:**

- About 20 tool definitions per turn is past the measured comfort zone for Haiku-class models.
- The prompt never states the goal of the request.
- Prompt caching probably misses on most turns.

**Recommendation.** Desk should become one agent runtime, built from these parts:

- **Job profiles:** today's agents, kept as the unit of permissions, autonomy, budget and
  schedule.
- **A stable, cached prefix:** a platform core, a new domain primer, a namespace map and the job
  profile.
- **A deferred tool catalog:** found through a *written* need, with `defer_loading` and
  `tool_reference` on Anthropic and `allowed_tools` / `tool_search` on OpenAI, so the cache and
  thinking blocks stay valid.
- **A per-request task frame:** the goal, the definition of done, what needs approval, and a
  playbook. It is built deterministically for unattended runs, and from the per-turn classifier
  in chat.
- **A smaller, consolidated tool surface.**

Code-as-action is added later as a second mode for read-heavy and batch work, gated by evals for
each model. Delegation stays only for parallel, read-only fan-out. `dispatch`, approvals, the
ledger and taint stay exactly as they are.

**Order of work:**

1. Measure, and fix correctness. Bench the unattended runs, which are not benched at all today.
2. Make the prefix stable and loading deferred: the biggest gain for the least risk.
3. Add the task frame and the domain primer.
4. Consolidate the tools.
5. Move to a single front door.
6. Add code mode.

Success is measured as pass^4, tool-selection accuracy, steps, tokens and cost per completed task,
on Haiku 5.5, the default model, at low and medium effort. Larger models are run only on the
scenarios Haiku fails, to tell a model's limit from a harness defect.

This is an architecture review of Desk, Trenova's agent harness. It is not a change
proposal for any one file. Numbers come from the source at the commit this was
written on; where a number is an estimate, the method is given.

## Audit summary

This summary describes the harness and what the evidence in the repository says about it. It
makes no recommendations yet; those wait for the research.

1. **Desk is not one agent with a flat tool list.**
   - It has 649 tools: 431 writes, 212 reads, and 6 the runtime answers itself.
   - It has 29 agent templates.
   - A router picks an agent before each turn, using keyword and embedding matching over tool
     metadata.
   - Delegation is one level deep.
   - An agent that holds more than 12 tools gets progressive disclosure: 8 preselected tools plus
     `find_tools`.
2. **The "too many tools" hypothesis has already been half-acted on, so the problem has moved.**
   Every chat specialist is disclosed, and a turn shows about 20 tool definitions, not 60–70. The
   open questions are now three:
   - **Preselection quality.** The hybrid ranker's CI floor is top-1 0.78 and recall@5 0.91.
     Preselection always fills 8 slots, so irrelevant heavy tools ride along.
   - **Fixed overhead.** About half of every tool list is 10 conversational and memory "core"
     tools, about 5.6k tokens with `_why`.
   - **Discovery cost.** A model that does not find the right tool spends budget on `find_tools`
     rewordings. gpt-6-luna does this most.
3. **The bench evidence points less at tool count than at the harness telling the model wrong
   or incomplete things.** Of 189 findings:
   - 58 were product bugs or missing tools;
   - 33 were descriptions or result shapes that misled the model;
   - 14 were argument or schema errors;
   - 16 were discovery or selection proper.

   Model-specific clusters are real and differ in kind. Luna loops, over-searches and fills
   parameters with placeholders. Haiku states ungrounded figures and makes mistakes within a
   turn.
4. **The prompt is careful but generic.**
   - About 2.5k tokens of shared rules go to every agent.
   - Nothing states the goal of the request in hand.
   - There is no domain model or glossary.
   - Per-request injection covers tools and memories only.
5. **Context before the question:** about 10–20k tokens on the first call in a clean case, and
   about 12–27k in realistic ones.
   - Tool schemas are 1.5 to 2 times the prompt.
   - `_why` alone is 15–25% of the tool payload.
   - The palette, the "quick" surface, sends the largest prompt (about 21k).
6. **Prompt caching probably misses on most turns** for agents holding more than 12 tools. The
   tool list changes between turns and within them, and tools come first in the cached prefix.
   This is not yet measured; the existing `ai.cacheDiagnostics` setting would settle it.
7. **The provider layer has concrete defects that would look like "model X is unreliable":**
   - Ollama never sends `num_ctx`;
   - sampling settings reach only OpenAIChat;
   - the final tool-free answer may be refused on Anthropic (unverified);
   - the tool error flag is lost on every protocol but Anthropic;
   - every habit fix is tuned to the two models ever benched.
8. **Evals are rich, but they cannot yet answer this review's questions across providers.**
   - Only two models have ever been benched, and only Haiku since 2026-10-15.
   - Cache tokens are dropped from the rollups, which biases Claude against OpenAI.
   - 142 bench rubrics are never scored.
   - Nothing measures which tool a model chose against a label.
9. **Write safety is strong and worth keeping as it is.** Any new interaction pattern must go
   through the same `dispatch` chokepoint: policy-in-code tiers, preview-digest approvals,
   at-most-once steps, and taint.

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

### 1.2 Tools

Method: every registered tool was built the way `agenttoolcatalog/contract_test.go` builds them,
and dumped with its description and its schema exactly as sent to the model
(`toolschema.ForModel`). Sizes are compact-JSON characters; tokens are characters ÷ 4.

**Registration and totals.** Tools join through Uber FX value groups: `agent_tools` for action
tools (`AgentTool.Execute`) and `agent_query_tools` for reads (`AgentQueryTool.Query`). They are
folded into one immutable, ranked `agenttoolcatalog.Catalog`.

| Set | Tools |
|---|---|
| Action registry (every one has effect `change`) | 431 |
| Query registry: 199 `lookup`, 11 `present`, plus `open_page` and `find_in_trenova` | 212 |
| Answered by the runtime: `find_tools`, `ask_user`, `publish_artifact`, `request_decision`, `withdraw_proposal`, `delegate_task` | 6 |
| **Distinct tools a Desk model can be shown** | **649** |

Other facts about the catalog:

- The tools use 114 permission resources.
- The older import-assistant loop (`shipmentimportassistantservice/tools.go`) defines 17 more
  specs. 7 of them reuse registry names, `create_shipment` among them.
- Leading verbs: `list` 116, `get` 60, `update` 40, `create` 21, `record` 18, `delete` 17,
  `cancel` 16, and more than 50 other verbs.

**How an agent's tools are assigned.** Tools are held per agent and loaded per turn:

1. **Held.** An agent holds 10 core tools, its stored `ToolNames`, and each held tool's
   companions and prerequisites (`toolset.go:607`, `agentdefinition/core.go`). A template only
   seeds a new agent; the list is then edited per agent in AI Control. The stored selection is
   capped at 64 (`MaxTools`), but companions take the shipped templates to 66–72 held tools.
2. **Narrowed by permission.** `permittedWith` (`toolset.go:723`) drops every tool the person may
   not use, and every extension tool the organization has not turned on.
3. **Loaded for the turn** (`newToolSet`, `toolset.go:152-216`):
   - an agent holding **≤12** non-core tools sends all of them and has no `find_tools`;
   - an agent holding more ranks its tools against the request (`RankHybrid`: keyword plus
     embedding, reciprocal-rank fusion) and loads the top **8**, with their families. It adds the
     tools carried over from the last three turns, `find_tools`, and the conversational runtime
     tools;
   - every other held tool is listed in the system prompt under "Callable after find_tools", one
     first sentence each;
   - `find_tools` loads up to 6 tools per call, at most 4 times per turn.
4. **Decorated.** `withRationale` adds a `_why` object to **every** tool's schema (about 600
   characters, about 150 tokens).

Templates (starter tools / held, including core and families):

| Template | Starter | Held | Disclosed |
|---|---|---|---|
| Billing, Dispatch, Customer, Compliance, Settlements, Master data, Workforce | 55–56 | 65–72 | yes |
| IntakeDesk | 46 | 58 | yes |
| FuelTax, Receivables, Formula, LoadMonitor, BooksKeeper, ReportAnalyst, Factoring, DispatchAssignment | 20–33 | 32–43 | yes |
| ShipmentIntake, Import, BillingException, Detention, EDI | 13–17 | 23–28 | yes |
| ServiceFailure, CashApplication, InsightAnalyst, LoadEntryCheck, CarrierRisk, Credential, CustomerUpdate | 7–12 | 17–22 | no |
| **GeneralAssistant on the Desk** | **0** | **10 (core only)** | no, and it has no `find_tools` |
| **GeneralAssistant on the palette** | 0 | **212** (every read, through `HoldingReads`) | yes |

21 of 29 templates are disclosed, as is every chat specialist. 70 tools (51 actions and 19
queries) are on no template, each withheld with a reason.

**What a request carries** (keyword-only replay, since that is what runs without an embedding
provider):

| Agent | Tool definitions sent | Tool tokens | "Callable after" list in the system prompt |
|---|---|---|---|
| General assistant, Desk | 11 | ≈7.3k | – |
| General assistant, palette | 19–21 | 10.4–11.2k | ≈6.7k |
| Dispatch / Billing assistant | 20–23 | 11–13.5k | ≈1.7–1.8k |
| ReportAnalyst / EDIDesk / ShipmentIntake | 20–29 | 13.7–18.9k | ≈0.2–0.4k |

So "8 preselected" means about 20 tool definitions on the wire. The core tools are about half of
them, at about 5.6k tokens with `_why`.

**Size distribution, all 643 registry tools** (characters):

| | Median | p90 | p99 | Max |
|---|---|---|---|---|
| Description | 282 | 537 | 872 | 899 (lint cap 900) |
| Schema | 463 | 1,522 | 5,172 | 11,156 |
| Name + description + schema | 804 | 2,084 | 5,846 | 11,809 |

- The whole catalog is about **179k tokens**, or about 276k with `_why`.
- The palette's 212 reads, sent at once, would be about 95k.
- The largest tools are `draft_edi_connection` (11.8k characters), `apply_edi_connection_draft`
  (11.6k), `create_shipment` (7.3k), `create_report` and `update_report` (6.5k each), and
  `propose_formula` (6.4k).
- Among the runtime tools, `ask_user` is 4.8k characters. Its 1,306-character description is over
  the 900-character cap, which applies only to registry tools.

**Quality issues.**

1. **Overlap.** A shipment can be read through `list_shipments`, `search_shipments`,
   `get_shipment`, `find_shipment_records` (in fact a matcher for a new load), and the generic
   `compose_table_view` / `run_report` / `preview_report`. Workers have six read tools.
2. **Eleven single/batch pairs differ by a trailing "s"**, for example `get_invoice` /
   `get_invoices` and `post_driver_settlement` / `post_driver_settlements`. The system prompt
   needs a paragraph to stop per-record loops (`prompt.go:1243-1275`).
3. **"find" means three unrelated things**: load tools, search the product guide, match a new
   load's records.
4. **Inconsistent naming.**
   - `search_worker` is singular, the other search tools plural.
   - `web_search` and `web_read` put the noun first.
   - A record is created under four verbs: `create_`, `record_`, `add_` and `log_`.
   - `propose_formula` applies the change to the editor at once, while 91 tools that really do
     file proposals never say "propose" in their name.
5. **Inconsistent schemas.**
   - Pagination comes in three styles: offset (59 tools), cursor (12), `limit` only (28).
   - 95 of the 99 `limit` parameters have no `maximum`; the bound is in the prose only.
   - Move ids are spelled two ways (`moveId` and `shipmentMoveId`).
   - Free text has six names (`reason`, `notes`, `note`, `comment`, `body`, `memo`).
   - Search text is `query` or `search`.
   - Date ranges come in six pairs.
6. **The generic list filter is untyped.** The 37 generated list tools share
   `filters[{field, operator, value, values, days}]`, in which `value` is a string. Each tool
   therefore spells out its fields and allowed values in prose, plus a dates sentence repeated in
   34 tools. That is why the list tools sit at the 900-character cap.
7. **Policy prose in descriptions.** 101 descriptions say a person decides, and 107 contain
   "never". This restates what the policy and the "What a change you call does" prompt section
   already say.
8. **Preselection always fills all 8 slots, by design** (`catalog.go:226-227`). Irrelevant heavy
   tools ride along:
   - EDIDesk, asked "where is shipment S-1001", loads both EDI-draft tools (about 6k tokens).
   - On the palette, "run the on-time delivery report" loads `set_stop_schedule`, a page-bound
     tool that means nothing off its own page.
9. **Effect is the reliable read/write field, not the registry.** The 11 `present` tools in the
   query registry change page state and run automatically.
10. **Small bugs.**
    - `FirstSentence` cuts at the first full stop, so `record_employment_verification` shows
      "…(49 CFR 391."
    - `ai-tool-safety.md` says 646 tools and lacks the three web-watch tools added on 2026-10-18.
      Either its render test has not yet run on that commit, or it excludes those tools; I have
      not resolved which.

### 1.3 System prompt and instructions

These figures come from the 90 committed prompt snapshots
(`domain/agentdefinition/testdata/prompts/*.golden`) and from five prompts rendered through the
real `OpenTurn` with the real tool registry. Token counts are characters ÷ 4. That is a floor:
JSON schemas tokenize denser than prose, so treat the tool figures as low by perhaps 10–30%.

**Assembly.** `BuildSystemPromptParts` (`domain/agentdefinition/prompt.go:334`) builds the prompt
in two parts, and the prompt version is `agent-definition/v9`. The stable part is meant to be
cacheable. The volatile part changes from turn to turn. A few more blocks ride on the user
message (`agentruntime/turn.go:579-602`).

The stable part, in order:

| Section | Scope | ~Tokens |
|---|---|---|
| Identity and the administrator's description | per agent | 85–155 |
| **Boundaries**: tenant, act only through tools, proposals, no code, the list of untrusted tags | static | 416 |
| **Working with records**: lookup not recall; never invent ids; send only the parameters you know; do not calculate; overdue first; propose as the person said it | static (a variant is used when the agent holds `analyze_data`) | 741 (≈1,000 with the variant) |
| **Tools**, grouped by what a call does, with recipes | per agent, only when the agent holds ≤12 tools | 110–300 |
| **Agents you can ask** and the `<delegate_agents>` roster | per agent and person | ~200 + 35 per delegate |
| **Artifacts** | static | 305 |
| **Trenova itself**: product blurb, `find_in_trenova` and `open_page` rules | static | 489 |
| **Output**: conversational, inline (palette), report, or delegated | per surface | 63–360 |
| **Remembering** | static | 248 |
| **Organization instructions**: the template's text, with `{{organization}}`, `{{user.name}}` and similar filled in | per agent | 58 (General) to 934 |
| **Never**: up to 20 guardrails set by an administrator | per agent | 0 on the starter templates |
| **About this organization**: the nightly organization profile | per organization | 150–540 |

The volatile part:

- runtime context: the organization, the date, how the run started, the surface, the person and
  their profile, `<subject_context>`, `<page_context>` with the product guide's summary of the
  page, `<page_view>`, `<page_draft>`, pinned facts and mentions (81–280 tokens, and hundreds
  more with a page view or draft);
- memories chosen for this message (0–6,000 tokens by default; 16,000 at most);
- the **disclosed-tool catalog**, on agents that hold more than 12 tools. It names the loaded
  tools and then lists *every other held tool* with its first sentence: about 1,750 tokens on
  Dispatch, 2,550 on Billing, and **about 7,000 on a palette question** (195 tools);
- proposals awaiting a decision (about 245 tokens).

On the user message: the `Now:` line, `<records_in_play>` (100–600 tokens), decisions made out
of view, attachment text (up to 24,000 characters), and slash-command directives.

The shared rules that every agent carries, before any agent-specific text, come to about 2,540
tokens: Boundaries, Working with records, Artifacts, Trenova itself, Output and Remembering.

**What the prompt does well.**

- Every rule says why it exists, and the comments in `prompt.go` record the incident behind each
  one.
- There is no shouting in capitals.
- The stable and volatile split is deliberate. For example, the date is day-only so that the
  prefix stays cacheable.
- Records in play are kept off the saved history, so a stale status or ETA is never replayed.

Representative lines:

> Anything about this organization's records is a lookup, not a recall. Look it up, every
> time, even when you are confident. You do not know this organization's data.

> When the person names exactly what to change — this driver on that tractor, this amount,
> that time — propose it as they said it… The approval is where they weigh it, so do not ask
> them to confirm first.

**What it does not do.**

1. **Nothing states the goal of this request.** No intent, no task frame, no plan, no success
   criteria. The scope classifier's category is used only to refuse, and is never passed to the
   prompt. The agent matcher adds nothing to the prompt either.
2. **No domain model.** Nothing defines a move, a stop, a billing queue item, an EDI 210 or a
   settlement. The domain reaches the model through:
   - a one-paragraph product blurb;
   - the organization profile;
   - the business rules inside each template's instructions;
   - the tool descriptions.

   The product guide (`docs/product-guide/`) is reachable only through `find_in_trenova`.
3. **Per-request injection exists, but only for tools and memories.** The tools are ranked
   (hybrid lexical plus semantic) and the memories are chosen for the message. Learned
   `Procedure` memories are the nearest thing to playbooks. Nothing retrieves domain or workflow
   guidance for the request in hand.
4. **Generic text goes to agents it does not apply to.** "Working with records" (741 tokens) goes
   to the Desk's General assistant, which holds no record tools.
5. **The same text is said twice.**
   - `publish_artifact`'s description repeats the Artifacts section almost word for word.
   - `remember` repeats Remembering.
   - `find_in_trenova` and `open_page` repeat Trenova itself.
   - `find_tools` repeats the catalog preamble.
   - About 300 tokens of prose restate rules the runtime already enforces: stripping ids, code
     and tables; grounding; the `find_tools` cap.
6. **Tensions.**
   - The General assistant is told to "change nothing", yet it holds `remember`, `guide_profile`,
     `wait_until` and `flag_for_manual_review`.
   - On the palette, the surface line says "keep it short" while Output says "write every row up
     to about 30".
   - Boundaries tells every agent to use `withdraw_proposal`, a tool that does not exist until a
     proposal is filed.
7. **The list of untrusted tags in Boundaries is incomplete.** It leaves out
   `<records_in_play>`, `<pinned_facts>`, `<delegate_agents>`, `<file_text>` and both memory tags.

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
6. **Haiku 5.5's thinking-binding check may not be handled** (found while checking Anthropic's
   current API reference in Phase 2). Haiku 5.5 is the default model.
   - **What the API does.** Haiku 5.5 runs the same history-editing check as Opus 5.5 and Sonnet
     5.5. A thinking block sent back after any change to `system`, `tools` or earlier messages is
     a 400. This is enforced by default for accounts created on or after 2026-08-31, and on older
     accounts only for requests that opt in.
   - **What the adapter does.** It sets `bindsPrefix` (`drop_block` plus the beta header) only for
     Opus ≥5.5, Sonnet ≥5.5 and Fable ≥5.1 (`modeladapter/anthropicmodels.go:64-80`).
   - **How a turn hits it.** Desk changes `tools` within a turn in two places:
     - `find_tools` loads tools;
     - `offerWithdrawal` adds `withdraw_proposal` after the first proposal
       (`withdrawproposal.go:71-80`). That happens even on small agents.

     After either change, Desk replays the turn's thinking blocks.
   - **Expected result.** A Haiku 5.5 turn on a new account would get a 400 at that point, and the
     router does not retry a 400.
   - **Status: unverified.** The bench account probably predates the cut-off, which would explain
     why no finding records it. The test is one bench run on an account created after 2026-08-31
     with a turn that files a proposal and then keeps working.
7. **Habit fixes exist only for the two benched models.** Every one targets Haiku 5.5 or
   gpt-6-luna. Gemini, Llama, Qwen, DeepSeek and Ollama have code paths but no bench coverage.

### 1.5 Context spent before the person's message is read

First model call, measured on prompts rendered through the real runtime, with short profiles,
no memories and no delegates:

| Case | Tools sent | System prompt | Tool schemas | `_why` on every tool | **Total** |
|---|---|---|---|---|---|
| Desk, General assistant (core tools only) | 10 | ≈3,280 | ≈5,440 | ≈1,500 | **≈10,250** |
| Desk, Dispatch assistant (holds 50, disclosed) | 22 | ≈5,750 | ≈10,620 | ≈3,300 | **≈19,700** |
| Desk, Billing assistant (disclosed) | 19 | ≈6,435 | ≈8,140 | ≈2,850 | **≈17,450** |
| Billing as a delegate of Dispatch | 20 | ≈5,260 | ≈8,480 | ≈3,000 | **≈16,780** |
| Palette quick question (General assistant holding all 206 reads) | 18 | ≈10,450 (≈6,825 of it the catalog) | ≈7,790 | ≈2,700 | **≈20,950** |

In production, add full profiles (about 750), memories (0–6,000), delegates and records in play.
That gives roughly **12–17k tokens for a General-assistant turn and 21–27k for a specialist**,
before the question.

Where it goes:

1. **Tool schemas outweigh the prompt by 1.5 to 2 times.** The 10 core tools that every chat turn
   carries cost about 5,400 tokens:
   - `ask_user` 1,218;
   - `wait_until` 851;
   - `remember` 631;
   - `publish_artifact` 600;
   - `flag_for_manual_review` 566;
   - `recall_memory`, `guide_profile`, `find_in_trenova`, `open_page` and `cancel_wait` for the
     rest.
2. **Core tools are about half of every disclosed turn's tool list.** On Dispatch, 22 tools are
   sent: the 10 core tools, the 8 preselected for the question, `find_tools`, and tools carried
   over. Every specialist turn opens with only 8 domain tools in view, and with 10 conversation
   and memory tools beside them. Not one of the 10 core tools is a domain tool.
3. **`_why` costs about 150 tokens per tool**, on every tool and every call: 1,500 to 3,300 tokens
   per request. It is valuable for the person reading the transcript, but it is the most
   compressible block of all (see Phase 3).
4. **The palette, the "quick" surface, sends the largest prompt in the system.** Most of it is the
   195-line catalog, and it sits in the volatile, uncached part.

**Prompt caching probably misses for most agents.** This is an inference from code, not a
measurement. Anthropic matches its prefix in the order tools, then system, then messages. So any
change to the tool list discards the cached system prompt too. The tool list changes in three
places:

- **Between turns**, on any agent holding more than 12 tools: the 8 preselected tools are
  re-ranked from each question.
- **Within a turn**, when `find_tools` loads tools.
- **Within a turn**, when the first proposal adds `withdraw_proposal`
  (`withdrawproposal.go:74-80`).

The cache-mark comment at `modeladapter/anthropic.go:726-746` holds only for agents with 12 or
fewer tools. `ai.cacheDiagnostics` already exists and would confirm or refute this in one day of
traffic: look for `tools_changed`.

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

1. Money's class cap is Automatic, so 7 money tools can run unattended in an untainted run:
   - `post_customer_payment`;
   - `match_bank_receipt`;
   - `correct_charge_code`;
   - adding and removing carrier and driver settlement adjustments.

   (`vet_carrier` is Ask first; an earlier draft wrongly listed it here.)
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

### 1.8 Things that surprised me

1. The Desk's **General assistant holds no domain reads** and has no `find_tools`. It only
   answers across the system when asked from the command palette. There it holds all 212 reads,
   including page-bound tools such as `set_field_value` and `propose_formula`, which mean
   nothing off their own pages.
2. **The cheapest surface is the most expensive.** A palette quick question carries a
   6.7–7k-token catalog of every read tool, in the uncached part of the prompt.
3. **`_why` is the largest single line item that nobody counts.** It is about 150 tokens on
   every tool and every call, and it is invisible to the description-size lint.
4. **Several runtime guards ask the model again after its reply**: grounding, artifact
   placement, command hold, empty and loop retries. Refused calls (schema, validation, would-fail,
   duplicates, unparsed JSON) spend the same 12-call budget as useful work. Two bench findings
   ran out of budget this way.
5. **Two of the clearest wins in the bench history were not about the model.** DB-008 made
   `search_shipments` actually search the customer and city it claimed to, and its calls fell
   from 107 to 44. DB-080 sent `strict:false` on Responses. Both were harness changes, and they
   fixed more than any prompt edit did.
6. **Descriptions are pressed against a 900-character lint cap** because one untyped list filter
   pushes per-tool field documentation into prose, across 37 tools.

### 1.9 Questions before Phase 2

The research focus and the eval matrix depend on these answers:

1. **Which models and providers matter in production?** Is it the Claude family, OpenAI's
   gpt-6 line, Gemini, or self-hosted open models on Ollama or vLLM? Only Haiku 5.5 and
   gpt-6-luna have been benched. If self-hosting small models matters, the Ollama defects and
   small-model behaviour rise to the top.
2. **Are the specialist desks a product commitment, or are they an implementation detail?**
   The answer decides whether Phase 3 can propose one front-door agent that routes internally,
   instead of agents a person picks.
   - "Operate Trenova fully through natural language" reads like one front door.
   - Today the Desk's General assistant cannot do it, and the palette can only read.
3. **Which surface matters most:** Desk conversations with specialists, palette questions, or
   unattended event and scheduled runs? Their budgets and failure costs differ.
4. **Is there evidence beyond `findings.yaml`?** For example, `ai_work_units`, scorecard tool
   verdicts, thumbs-down reasons, or a sample of real failing conversations. Could you turn on
   `ai.cacheDiagnostics` for a day? I cannot run the bench here, because it needs a seeded
   database and provider keys.
5. **How much appetite is there for code-as-action?** The E2B sandbox already exists for
   `analyze_data`. A typed Trenova SDK whose writes still went through `dispatch` (so proposals,
   previews and the ledger stay intact) is a real candidate for read-heavy and multi-step work. If
   it is off the table for product or security reasons, I will not spend research time on it.
6. **What are the targets for latency and cost?** For example, a p50 time to first token, or a
   cost ceiling per turn. These decide how far to push routing and caching against accuracy.

### 1.10 Answers from the product owner (2026-10-18)

1. **Models.** Claude and OpenAI matter most. Self-hosted small models (Ollama, vLLM) have no
   real users. They exist because Trenova can be self-hosted, and they may be removed.
2. **Specialist desks are not a product commitment.** They exist so that work driven by a job
   description is relatable: an agent is given a job description and does the work a person in
   that job would. A single front door that routes internally is open for consideration.
3. **Priority of surfaces:**
   1. unattended event and scheduled runs;
   2. Desk conversations;
   3. palette questions.
4. **Evidence.** There is none beyond `findings.yaml`. `ai.cacheDiagnostics` can be turned on,
   and heavier desk bench runs can be done on the owner's machine.
5. **Code-as-action is on the table.**
6. **Latency and cost: the lower the better.** Haiku 5.5 is the default because it is fast and
   cheap. Opus and Fable are options, at a higher cost.

What this changes for the rest of the review:

- **Unattended runs come first.** Nobody is there to rephrase, approve straight away or notice a
  wrong turn, so reliability measured over repeated runs (pass^k) matters more than a single
  pass.
- **The target model is a small, fast one.** Every recommendation has to work on Haiku 5.5
  and a comparable OpenAI model, not only on frontier models.
- **The provider work narrows** to Anthropic Messages (with its platforms) and the OpenAI APIs.

## 2. Research findings

**Method.**

- Five parallel research tracks read primary sources: arXiv HTML and PDFs, benchmark data files, and vendor documentation and engineering posts.
- I re-checked the numbers this review relies on against the primary source myself:
  - Anthropic's tool search and programmatic tool calling docs;
  - IFScale's published results CSV;
  - the Haiku 4.5 rows of "The Bitter Lesson of Tool Calling";
  - the Haiku 4.5 tool-count result in the MCP architecture-patterns paper;
  - the Claude Haiku 5.5 launch page;
  - the advisor post.
- Anthropic API facts also come from the Claude API reference bundled with Claude Code (cached 2026-10-06), and only where it agrees with the live docs.

**Evidence labels** used below:

- **[PR]** peer-reviewed;
- **[PP]** preprint;
- **[BM]** benchmark or leaderboard data;
- **[VD]** vendor documentation;
- **[VB]** vendor blog, often reporting internal evals that are not public;
- **[AN]** practitioner anecdote.

Everything in this section is a finding from a source. Reasoning about Desk is kept to Section 3.

### 2.1 Tool count, overlap and description quality

| Finding | Evidence | Strength |
|---|---|---|
| Accuracy falls as the catalog grows, mildly for frontier models and steeply for small ones. Claude-4-Sonnet: 62.3% with only the tools needed, 62.4% with 218 tools, 44.2% with all 552. GPT-4o fell from 0.94 to 0.76 and Llama-3.1-8B from 0.86 to 0.09 as the catalog went from 49 to 741 tools. | MCPVerse, arXiv 2508.16260; LongFuncEval, arXiv 2505.10570 | [PP] |
| The only small-Claude data: **Claude Haiku 4.5's tool-selection accuracy falls below 90% between 10 and 15 tools in context** (91% at 10, 87% at 15). Sonnet 4 stays at 90% or above up to 20 tools. | Rodrigues & Vas, arXiv 2606.30317. The abstract confirms the threshold; the per-count figures are from the body. | [PP], production telemetry from one organization |
| Sensitivity to tool count depends on the model and must be measured per model. Moving from a domain's own tools to the full pool cost Claude-3.5-Sonnet 28 points (97.6% → 69.2%) and GPT-4.1 3 points (98.1% → 94.7%). | MCP-Zero, arXiv 2506.01056 | [PP] |
| Showing more tools hurts the *choice* even when the right tool is in view: Sonnet 4.6 chose correctly 100% of the time with 1 tool shown and 87.1% with 5. But fixed top-k sets miss hard queries: on ToolBench, top-5 covered 64.7% of queries and top-20 covered 77.3%. | Repantis et al. (Meta), arXiv 2605.24660 | [PP] |
| **Overlap matters more than count.** "Some implementations successfully manage more than 15 well-defined, distinct tools while others struggle with fewer than 10 overlapping tools." Merging semantically redundant tools raised GPT-4o's correct-selection rate by 5–22%. | OpenAI, *A practical guide to building agents*; ToolScope, arXiv 2510.20036 | [VB], [PP] |
| Vendors put their soft limits in the same place: | | |
| • OpenAI: "aim for fewer than 20 functions available at the start of a turn". | OpenAI function-calling guide | [VD] |
| • Google: an active set of 10–20 tools. | Gemini function-calling guide | [VD] |
| • Anthropic: Claude's choice of tool "degrades once you exceed 30–50 available tools". Use tool search from 10 tools or 10k tokens of definitions, and keep 3–5 tools always loaded. | Anthropic tool search docs | [VD] |
| Description quality is worth 5–15 points. | | |
| • Removing parameter descriptions cost GPT-4o 14.5 points on BFCL-executable. | Play2Prompt, ACL Findings 2025 | [PR] |
| • Self-refined documentation took GPT-4o-mini from 35 to 47 correct paths, past GPT-4o with raw documentation at 37. | DRAFT, ICLR 2025 | [PR] |
| • Tool-use examples raised accuracy on complex parameters from 72% to 90%. | Anthropic, *Advanced tool use* | [VB] |
| **Longer is not better.** Adding to MCP descriptions gained a median 5.85 points but added 67% more execution steps and made results worse in 16.7% of cases. Short, targeted additions did as well. | Hasan et al., arXiv 2602.14878 | [PP] |
| Wording alone steers selection between near-equivalent tools by more than 10×. | Faghih et al., EMNLP 2025 | [PR] |
| **Models are bad at noticing that the tool they need is missing** (BFCL "Miss Func", 2026-04-12 snapshot): Sonnet 4.5 65%, Opus 4.5 64%, **Haiku 4.5 42.5%**, GPT-4.1 32.5%, GPT-5.2 18%, GPT-5-mini 17%. This is the skill `find_tools` depends on. | BFCL v4 leaderboard data; Patil et al., ICML 2025 | [BM], [PR] |
| Native function calling beats tools described in the prompt by a wide margin for small models: Haiku 4.5 scores 68.7% native against 25.3% prompted. | BFCL v4 | [BM] |
| Schema validity has converged across models (≥97.5% for gpt-4o-mini and up). What separates them is choosing the right tool, the right parameters and the right dependencies. | MCP-Bench, ICLR 2026 | [PR] |
| About a third of gpt-4o's τ-bench failures are wrong arguments. Weaker models invent ids about 4× as often. | τ-bench, arXiv 2406.12045 | [PP/BM] |

### 2.2 Tool retrieval and dynamic loading

| Finding | Evidence | Strength |
|---|---|---|
| Tool retrieval is harder than ordinary retrieval. A query and the tool it needs share almost no words (ROUGE-L 0.06, against 0.31–0.34 for NQ and MS-MARCO). The best embedding model reaches nDCG@10 of about 47, BM25 about 22, and rerankers help little. Using retrieved tools instead of the right ones cost 11.4 points of pass rate. | ToolRet, ACL 2025 | [PR] |
| Benchmarks built from synthetic queries flatter retrievers. With queries written in user personas, BM25 retrieval scored *below* putting all 2,000 tools in context (59% against 65%). | HumanMCP, arXiv 2602.23367 | [PP] |
| **A search query the model writes beats retrieval on the raw user message.** | | |
| • A model-written capability request reached about 95%, against 71.6% (single turn) and 65.1% (multi-turn) when retrieving on the raw query. | MCP-Zero | [PP], one model, 48 tools |
| • Extracting the intents from a message before retrieving raised nDCG@5 from 0.65 to 0.78 for single-tool queries and from 0.53 to 0.72 for multi-tool ones. | Re-Invoke, EMNLP 2024 Findings | [PR] |
| • Breaking a query down and rewriting it raised Recall@5 by 46–56%. | Toolshed, arXiv 2410.14594 | [PP] |
| When the agent finds its own tools, **retrieval errors are about half of Claude's failures**. Cutting k from 5 to 1 lost 14.7 points; raising k to 10 added nothing. | LiveMCPBench, arXiv 2508.01780 | [BM/PP] |
| **Anthropic tool search keeps the prompt cache.** "The API excludes deferred tools from the system-prompt prefix … appends a `tool_reference` block inline … The prefix is untouched, so prompt caching is preserved." | Anthropic tool search docs | [VD], verified |
| • A custom search (embedding or hybrid) can return `tool_reference` blocks from an ordinary `tool_result`. | same | |
| • It supports up to 10,000 deferred tools and **Claude Haiku 5.5**. | same | |
| • It is not billed separately. | same | |
| On Anthropic's internal MCP evals, tool search raised Opus 4 from 49% to 74% and Opus 4.5 from 79.5% to 88.1%, and cut tool tokens by about 85%. | Anthropic, *Advanced tool use* | [VB] |
| OpenAI has two equivalents: | OpenAI function-calling and tool-search guides | [VD] |
| • `allowed_tools` limits which tools can be called "but not modify the list … to maximize savings from prompt caching". | | |
| • Client-executed `tool_search` (gpt-5.4 and later) injects loaded tools at the end of the context. | | |
| Practitioners report the same thing. | | |
| • Manus: the KV-cache hit rate is "the single most important metric". "Avoid dynamically adding or removing tools mid-iteration": it breaks the cache, and calls to tools no longer defined confuse the model. | Manus engineering post | [AN] |
| • Archestra modelled 77% savings from progressive disclosure and measured 13% on real transcripts. Conversation tokens dominate after a few turns. | Archestra | [AN] |
| • A source-code study of 11 coding agents found deferred tool loading is the dominant pattern. | arXiv 2609.00006 | [PP] |

### 2.3 Code as the action

| Finding | Evidence | Strength |
|---|---|---|
| CodeAct (Python as the action) reached 74.4% against 52.4% for JSON on M3ToolEval (gpt-4-1106), with about 30% fewer turns. It won on 12 of 17 models. It **lost** for claude-instant-1 (20.7 against 31.7) and scored about zero for small open models. On single atomic calls (API-Bank) it won on only 8 of 17. | Wang et al., ICML 2024 | [PR], 82 tasks, no writes |
| The newest head-to-head is a wash on average: 77.0 for programmatic calling against 78.6 for JSON over 14 models. | "The Bitter Lesson of Tool Calling", arXiv 2608.06370 (verified from the HTML) | [PP], small ablations (n = 31–52) |
| • **Claude Haiku 4.5 gained overall (81.9 → 85.8) but lost on dependent chains (88.5 → 73.1, n = 52) and under a flood of schemas (87.1 → 77.4).** | same | |
| • GPT-4o fell from 81.9 to 55.0 and GPT-4.1 to 62.1, mostly from syntax errors. | same | |
| Anthropic's own measurement: | Anthropic programmatic tool calling docs | [VD], verified |
| • **"On τ²-bench … where each turn makes one or two sequential tool calls, programmatic tool calling left scores unchanged and cost roughly 8% more. Sequential single-call workflows do not benefit."** | | |
| • On a 75-tool project-management benchmark it cut billed input tokens by about 38% with no change in accuracy. | | |
| Code-as-action degrades faster over repeated trials on transactional work. On τ²-bench retail (DeepSeek-V3.2), pass^4 was 47.0 for CodeAct against 58.8 for ReAct. On a PostgreSQL write domain it scored 36.2 against 52.4. | CodeDelegator, arXiv 2601.14914 (abstract verified; numbers from the body) | [PP], one lab |
| Every vendor uses the same security model, and none of them provides approval. | Anthropic PTC docs; Cloudflare Code Mode (Sep 2025, Feb 2026); OpenAI programmatic tool calling docs | [VD], [VB] |
| • The sandbox has no network and no credentials. | | |
| • Every effect returns through a dispatcher on the host. | | |
| • Calls are tagged as coming from code. | | |
| • Anthropic: "Do not rely on `allowed_callers` as a security boundary." | Anthropic PTC docs | |
| • OpenAI recommends direct calls for "writes or approval-sensitive actions". | OpenAI docs | |
| • Anthropic programmatic calling is not eligible for zero data retention. A pending call times out after about 4 minutes, so a person's approval cannot block inside the script. | Anthropic PTC docs | |
| **Staging effects and checking authority at commit time is what keeps approval sound.** | | |
| • CaMeL plans in code with capability tracking. It solved 77% of tasks securely (84% undefended) at about 2.8× the tokens. | arXiv 2503.18813 | [PP] |
| • Cordon's effect outbox cut risky commits from 45 of 45 to 0 of 45. | arXiv 2606.17573 | [PP] |
| • Agents commit on authority that has gone stale (207 of 216 invalidated runs committed anyway) unless a fail-closed check runs at commit. | arXiv 2607.10487 | [PP] |

### 2.4 Routing, multiple agents and planning

| Finding | Evidence | Strength |
|---|---|---|
| Whether multiple agents help depends on the task's structure. Against a single agent, results range from +80.8% on decomposable financial reasoning to **−70% on sequential planning**, and tool-heavy tasks pay a coordination cost. A single agent did 67.7 successes per 1k tokens; centralized multi-agent did 21.5. | Kim et al. (Google), arXiv 2512.08296 v3 (abstract verified) | [PP] |
| Across 1,642 traces from 7 multi-agent frameworks, 41–86.7% of tasks failed. System design accounted for about 44% of failures, misalignment between agents about 32%, and verification about 24%. Repeating steps (15.7%) and not knowing when to stop (12.4%) were the largest single causes. | MAST, arXiv 2503.13657 | [PP] |
| Multi-agent research used about 15× the tokens of chat. Anthropic lists "domains that require all agents to share the same context" as a poor fit. | Anthropic, *Multi-agent research system* | [VB] |
| A multi-agent system "compiled" into one agent with a skill library scored −2 to +4 points at about 54% fewer tokens. Choosing a skill stays above 95% up to 20 skills, falls to about 20% at 200, and **confusable neighbours cost 17–63 points**. Hierarchical selection recovers most of it: 72–85% against 45–63%. | Li, arXiv 2601.04748 | [PP], synthetic |
| A supervisor that paraphrases what its sub-agents said loses information (a "telephone" effect). Forwarding their answers verbatim improved results by about 50%. | LangChain, *Benchmarking multi-agent architectures* | [VB] |
| Re-planning after each observation was the largest single gain (29.6% → 44.2% on WebArena-Lite). A plan fixed before any observation is cheap but cannot react to what it finds. | Plan-and-Act, ICML 2025; ReWOO | [PR] |
| A model asked to correct itself without an outside signal does not reliably improve. Correction works with reliable external feedback. | Huang et al., ICLR 2024; Kamoi et al., TACL 2024 | [PR] |
| Cascades between a small and a large model. | | |
| • Router papers (FrugalGPT, RouteLLM, Hybrid LLM) are single-turn only. A cascade degrades quickly once its judge's error passes 0.2. | RouterBench | [BM] |
| • The advisor tool lets Haiku consult Opus mid-task: BrowseComp rose from 19.7% to 41.2%, at 85% less per task than Sonnet alone (verified). | Anthropic, *The advisor strategy* | [VB] |

### 2.5 Long context and instruction load

| Finding | Evidence | Strength |
|---|---|---|
| **Instruction density hurts small models exponentially.** Accuracy at 10 / 50 / 100 / 250 instructions (verified against the results CSV): | IFScale, NeurIPS 2025 | [PR], synthetic task, no Haiku 4.5 or 5.5 |
| • claude-3.5-haiku: 98.0 / **78.0** / **43.4** / 16.6 | | |
| • claude-sonnet-4: 100 / 98.0 / 94.4 / 77.2 | | |
| • gpt-4.1: 98.0 / 98.8 / 95.4 / 74.0 | | |
| • Small models fail mostly by leaving instructions out. Earlier instructions are followed better. | | |
| Real agent system prompts average 1,723 words and 11.9 constraints, and fewer than 30% of instructions are followed perfectly. **Constraints about tools and conditional constraints are the hardest.** | AgentIF, arXiv 2505.16944 | [PP] |
| Length alone costs accuracy, even with perfect retrieval and even with blank padding: drops of 13.9–85% at 30k tokens. Having the model recite the relevant facts before solving mitigates it. | Du et al., arXiv 2510.05381 | [PP] |
| On needles that share no words with the question, effective length is 8k for GPT-4o and 4k for Claude 3.5 Sonnet. | NoLiMa, ICML 2025 | [PR] |
| Every model family did better on a focused prompt of about 300 tokens than on the full 113k-token history. | Chroma, *Context rot* | [VB] |
| When a request's details are spread over several turns, models score 39% worse on average and their unreliability rises 112%. Restating the whole task in one message recovers 95% of single-turn performance. | Laban et al., arXiv 2505.06120 | [PP] |

### 2.6 Reliability across repeated runs, and how models differ

| Finding | Evidence | Strength |
|---|---|---|
| pass^k is the probability that *all* k trials succeed, and it sits far below pass^1. | | |
| • gpt-4o on τ-bench retail: pass^1 61%, pass^8 below 25%. | τ-bench | [BM] |
| • On τ² (pass^1 → pass^4): Opus 4.5 retail 79.6 → 51.8; Sonnet 4.5 retail 72.4 → 39.5. | τ² leaderboard v1.0.1 | [BM] |
| • **GPT-5.2 airline: 83 → 72 at high reasoning, 52.5 → 22 with reasoning off.** Reasoning effort is the largest single lever on OpenAI models. | τ² leaderboard v1.0.1 | [BM] |
| Capability gains bring only small gains in reliability. Smaller models often match larger ones on consistency. Sensitivity to paraphrase is the main difference in robustness. Incorrect charges and refunds are the most common safety violation. | Rabanser et al., ICML 2026 | [PR] |
| A per-step success rate of 0.95 halves end-to-end success by about 13 steps; 0.99 lasts about 69 steps. Models condition on their own earlier mistakes; thinking models avoid this, and a sliding context window helps. | Sinha et al., ICLR 2026 | [PR] |
| Claude Haiku 5.5 (released 2026-10-07) reports Terminal-Bench 4.0 39.2% (Haiku 4.5 0%, GPT-6 Luna 16.4%) and OSWorld 2.1 offline 72.4%. **It publishes no τ², BFCL or MCP tool-use scores, and no independent tool-calling evaluation exists yet.** | Anthropic launch page (verified) | [VB] |

### 2.7 What the providers recommend

**Anthropic** ([VD], [VB]):

- **Tools.**
  - Prefer fewer, task-shaped, namespaced tools (`search_contacts`, not `list_contacts`).
  - Return readable names rather than ids.
  - Offer a `response_format` of concise or detailed; in one example this took a result from 206 to 72 tokens.
  - Return actionable errors.
  - "Provide extremely detailed descriptions. This is by far the most important factor."
  - "Ask for an explanation, not reasoning": a tool parameter that asks for reasoning can trigger a billed `reasoning_extraction` refusal on Opus and Sonnet 5.5. (Desk's `_why` already asks for an explanation.)
- **Prompts.**
  - Write at the "right altitude".
  - Use the "smallest set of high-signal tokens".
  - Dial back "CRITICAL/MUST" language.
  - Keep history append-only.
  - Deliver harness notices as mid-conversation system messages, never inside a `tool_result`.
- **Haiku 5.5 specifics.**
  - At `low` effort in long agent prompts it is "more likely to skip a search, stop early, or skip a check". Moving from `low` to `medium` roughly halved early stopping and more than doubled output tokens.
  - Pricing is $0.10/$0.50 per million tokens when the prompt is ≤100k tokens, and **$0.50/$2.50 above**. The tier is set by the whole prompt, cache reads included.
  - Its tokenizer counts about 30% more tokens than Haiku 4.5's for the same text.
  - Its thinking blocks run the history-editing check: a block sent back after a change to `system`, `tools` or earlier messages is a 400 for accounts created on or after 2026-08-31.
- **Caching.**
  - The order is tools → system → messages, and any change to the tool definitions invalidates everything.
  - Changing `tool_choice` keeps the tools and system caches and invalidates only the messages cache.
  - `tool_choice: none` is the documented way to get a text-only answer.
  - Reads cost 0.1× the input price (0.05× on Opus and Sonnet 5.5); 5-minute writes cost 1.25×.
  - The minimum cacheable prefix is 512 tokens.
- **Evals** (*Demystifying evals for AI agents*, 2026-01-09).
  - Grade the outcome, not the path.
  - Report pass^k.
  - Start from 20–50 tasks drawn from real failures.
  - Isolate trials.
  - Give the judge an "Unknown" option, with one judge per dimension.
  - Read the transcripts.

**OpenAI** ([VD]):

- Fewer than 20 functions at the start of a turn.
- Use `allowed_tools` rather than changing the list, so the cache survives.
- Strict mode on the Responses API.
- On GPT-6 Astra and GPT-6.1 Sol, tool calling requires the Responses API.
- Pass reasoning items back unchanged; reusing reasoning raised τ-bench retail from 73.9% to 78.2%.
- "Combine functions that are always called in sequence."
- The OpenAI Evals platform becomes read-only on 2026-10-31 and shuts down on 2026-11-30.

**Google** ([VD]): an active set of 10–20 tools.

### 2.8 Progressive disclosure and per-request context

| Finding | Evidence | Strength |
|---|---|---|
| Agent Skills disclose in three levels: about 100 tokens of name and description per skill always in context, a SKILL.md under about 5k tokens read when relevant, and bundled files read only when needed. No accuracy figures are published. | Anthropic, *Agent Skills* | [VB] |
| A playbook updated by itemized deltas (ACE) raised AppWorld from 42.4 to 59.4. Rewriting the whole playbook "collapsed" it from 18,282 to 122 tokens and dropped accuracy below the no-playbook baseline. Workflow memory added 24.6% relative on Mind2Web and 51.1% on WebArena. | ACE, ICLR 2026; Agent Workflow Memory, arXiv 2409.07429 | [PR], [PP] |
| Anthropic's context guidance favours a **hybrid**: some context retrieved up front, the rest found by the agent just in time. | *Effective context engineering* | [VB] |

## 3. Analysis

Sections 3 to 6 are my reasoning. Each claim points back to the audit (§1) or the research (§2).
Where something is a hypothesis, it says so, and §6 says how to test it.

### 3.0 Two more audit facts, found while analysing

1. **No unattended run is benched.** The desk bench drives chat turns only, through `StartTurn`.
   There are no event-driven or scheduled scenarios (`internal/deskbench/scenario.go`,
   `deskbench/scenarios/*.yaml`), yet these are the owner's first-priority surface.
2. **How unattended runs pick their first tools.** An unattended run's whole request is
   boilerplate (`agentdefinition/runinput.go`): "An event started this run: `<kind>`. It concerns
   `<label>`… Follow your instructions: look up what you need, act through your tools…". The
   8 preselected tools are ranked against that text. I reproduced the keyword ranking for every
   non-chat template, using a scratch overlay test with no change to the repo:
   - **Event-driven runs** rank well enough: the event name (`billing_queue.item_exception`)
     carries most of the signal.
   - **EDIDesk**, given a quarantined file, loads `draft_edi_connection` and
     `apply_edi_connection_draft`. These are the two largest tools in the catalog, about
     6k tokens together.
   - **LoadMonitor**, the one scheduled template, opens with `list_watchtower_items`,
     `email_customer`, `notify_driver`, `list_service_failures`, `log_check_call`,
     `record_stop_actual`, `add_shipment_comment` and `evaluate_service_failures`. That is four
     writes. It does *not* open with `get_dispatch_board`, `get_shipment`,
     `get_shipment_tracking` or `get_worker_hos`, although its instructions start with "work
     through what the dispatch board shows". Its first step has to be `find_tools`, the skill
     small models are weakest at (§2.1, BFCL Miss Func: Haiku 4.5 42.5%).
   - With embeddings configured the order may differ. The point stands: the request carries no
     goal, so the ranking has nothing to rank against.

### 3.1 Is the diagnosis right?

**Partly. Tool count and instruction load are real factors, but they are not the main cause.
The evidence points at four causes, in this order.**

1. **The harness misleads the model, and the evidence for this is strong.** The bench's own
   history has more cases of a description or result shape misleading the model (33 findings)
   than of discovery or selection (16), and 58 more were product bugs (§1.7).
   - The biggest single win on record, DB-008, was making a tool do what its description said.
     It cut 107 calls to 44.
   - The literature agrees: description quality is worth 5–15 points, wording alone steers
     selection by more than 10×, and longer descriptions can hurt (§2.1).
   - So the cheapest reliability gain is still tools whose descriptions are true, whose names
     are distinct and whose results are complete. That is the work the team already does through
     the findings log; it needs to be systematic rather than found one finding at a time.

2. **The tool surface overlaps.** The OpenAI guide says, and ToolScope and the skill-library
   results measure, that overlap and confusability matter more than count (§2.1, §2.4). Desk has
   heavy overlap:
   - shipments can be read five ways;
   - eleven single/batch pairs differ by a trailing "s";
   - "find" means three different things;
   - there are 29 job agents whose tool sets overlap heavily, for example Billing, Receivables and
     Factoring (§1.2).

   The router and the in-turn ranker are both choosing among confusable neighbours. That is where
   the skill-library study saw 17–63 points lost.

3. **Retrieval runs on text that carries no goal.** Preselection ranks the raw user message, or
   for unattended runs a boilerplate sentence (§3.0).
   - Raw messages share about 6% of their wording with the tools they need (§2.2, ToolRet).
     Retrieving on a written need instead of the raw message was worth more than 20 points in
     MCP-Zero.
   - The CI floor for hybrid ranking already shows the effect: top-1 0.78 and recall@5 0.91 on
     paraphrases, against 0.98 and 1.0 on non-paraphrases. The non-paraphrase cases reuse the
     catalog's own words, so they flatter the ranker.
   - When preselection misses, recovery depends on `find_tools`. Recognising that a needed tool
     is missing is a skill small models do badly (Haiku 4.5 42.5%, GPT-5-mini 17%).

4. **Behaviour differs by model, partly because of the harness's own adapters.**
   - Luna over-searched partly because the Responses adapter did not send `strict:false`
     (DB-080).
   - Ollama is never given a context size.
   - Sampling reaches only one protocol.
   - Haiku 5.5's history-editing check may turn the harness's own tool-list edits into 400s
     (§1.4 item 6).

   Some of what reads as "model X is unreliable" is the harness treating model X differently.

**Where the hypothesis is right:**

- **Tool count.** About 20 definitions go out per turn, 10 of them generic core tools. For a
  Haiku-class model that is past the measured 90% line (between 10 and 15 tools for Haiku 4.5;
  OpenAI's soft ceiling is 20). Showing more tools lowers choice accuracy even when the right one
  is in view (§2.1).
- **Instruction load.** About 2.5k tokens of shared rules carry 16–24 "never" and 12–14
  "do not" statements per agent, plus the template's own instructions. For a small model, IFScale
  measures omissions rising quickly past a few dozen directives. AgentIF finds constraints about
  tools the hardest to follow (§2.5).

So the load does cost accuracy. But cutting the load without fixing causes 1–3 would move the
failures elsewhere rather than remove them.

**The prompt hypothesis is right.** Nothing tells the model what this request is for: no goal, no
definition of done, no plan (§1.3). For unattended runs the goal is fully known before the run
starts, from the trigger, the event and the template. Today it is spread through the job
description, and the request itself is boilerplate. Restating the whole task in one place
recovered 95% of the loss from multi-turn underspecification (§2.5). A task frame is the
single-message version of that.

**The efficiency hypothesis is right, with three specific causes:**

- prompt caching probably misses on most turns (§1.5);
- refused calls and guard re-asks spend the same budget as useful work (§1.6);
- missed preselection forces extra `find_tools` steps (§3.0).

At Haiku prices, latency and steps matter more than dollars (§3.4).

### 3.2 Is a flat list of tool calls the right interface?

**No. But the answer is not to replace discrete tools; it is to change what the model is shown
and when.** I compared four patterns against the evidence and against Desk's constraints. Those
constraints are:

- every write goes through `dispatch`;
- the default model is small;
- the main workload is unattended and sequential, and ends in proposals.

| Pattern | What the evidence says for Desk's workload | Verdict |
|---|---|---|
| **One flat list of everything** | Collapses at Desk's scale: MCPVerse lost 18 points at full scale, and the 600+ tools would be about 179k tokens. | Rejected |
| **More multi-agent** (orchestrator plus specialists, deeper delegation) | −39% to −70% on sequential tasks, a penalty on tool-heavy tasks, 3–5× the tokens, and handoffs lose context. MAST, Kim et al. and Anthropic's own "poor fit" list all describe this workload (§2.4). | Rejected as the main pattern. Keep one-level delegation for **parallel, read-only fan-out** only. |
| **Code over a typed SDK, replacing tools** | Helps fan-out, aggregation and filtering (−37% to −98% tokens). Neutral to negative on sequential, approval-gated transactional work: τ² unchanged at +8% cost; worse pass^4. **The nearest small Claude model regressed on dependent chains (Haiku 4.5: 88.5 → 73.1).** Writes need an outbox and re-checking at commit (§2.3). | Rejected as a replacement. **Adopted as a second mode** for read-heavy sweeps and batch writes, behind evals for each model. |
| **One runtime with a stable, deferred catalog, a written search need, a per-request task frame, and fewer, consolidated tools** | Supported by tool search (+25 points on Opus 4 at −85% tokens; the cache is preserved), model-written queries (+20 points), a per-request focus (§2.5), and overlap reduction (+5–22 points). Keeps `dispatch` unchanged. | **Adopted as the core.** |

MCP is not the question here. It is a transport, and nothing in the evidence favours moving
Desk's in-process tools behind it.

### 3.3 System prompt design

What should be static, what should be injected per request, and how the model learns the domain:

| Layer | Content | Where it goes | Changes |
|---|---|---|---|
| **Platform core** | Boundaries, how to treat records, output contract, approvals | `system`, first, cached | per release |
| **Domain primer (new)** | A glossary and entity model (about 1–2k tokens): shipment → moves → stops; assignment; tender; billing queue → invoice → posting → adjustment; settlement; detention; EDI 204/210/214/990; the record id prefixes. | `system`, cached | per release |
| **Tool map (new shape)** | The tool namespaces and what each covers, one line each, in a stable order: "`shipments_*`: find, read, create and change loads…". This replaces today's per-turn list of every unloaded tool. | `system`, cached | per release or per agent |
| **Job profile** | Today's template instructions: persona, priorities, what stays a person's decision, organization guardrails | `system`, cached per agent | per agent version |
| **Organization and person profiles** | As now, but the person profile moves out of the volatile part | `system`, cached per org / person | daily |
| **Task frame (new)** | The goal of *this* request, the success criteria, the definition of done, what needs approval, the budget, the relevant playbook ids, and the subject records. **Unattended:** built deterministically from trigger, event kind and subject (an event → playbook table). **Chat:** built from the per-turn classifier, which today only refuses. | after the cache breakpoint, as a mid-conversation system message on models that take one, else a block in the user turn | per request |
| **Playbooks (new, Skills pattern)** | Step-by-step procedures for recurring jobs: "bill a delivered load", "cover a move", "handle a quarantined 210". Each has a one-line index entry in `system` and is loaded on demand by a `read_playbook` tool or through the task frame. Learned `Procedure` memories feed them as itemized delta updates, as in ACE (§2.8). | loaded per request | as learned |
| **Records in play, memories, page** | As today, but kept out of the `system` prefix | messages | per turn |

What this changes:

1. **The goal is stated, so the model need not infer it.** For unattended runs this is pure
   gain. The information already exists in the trigger, the event kind and the template; it
   only needs writing down in one place.
2. **Rules the runtime already enforces leave the prompt** (about 300 tokens; §1.3). Tool rules
   go into the tool's own description, not the shared rules, and only when a tool needs one. The
   aim is a small number of rules that hold everywhere, with the critical ones first, since
   models follow earlier instructions better (§2.5).
3. **The domain is taught once, in a cached block**, instead of being implied by hundreds of
   tool descriptions.
4. **The per-turn classifier already runs on every turn** (`agentguard/classifier.go`), with a
   3-second budget, and outputs only a category and an eight-word reason. Extending its schema to
   also return the request's goal and a written search need costs no extra call. That one output
   feeds both the task frame and preselection, which turns retrieval on the raw message into
   retrieval on a written need (the MCP-Zero and Re-Invoke effect). It needs care: today the
   classifier fails open, and it must stay advisory.

### 3.4 Token and step efficiency

Ranked by expected gain against risk. The cost figures use the Haiku 5.5 and Opus 5.5 list prices
(§2.7).

1. **Make the prompt cache hold.**
   - Send a stable, name-sorted superset of each agent's held tools on every call.
   - Mark everything except a small core `defer_loading`.
   - Answer `find_tools` with `tool_reference` blocks, a documented custom-search path that
     preserves the prefix and works on Haiku 5.5.
   - Always declare `withdraw_proposal` (deferred), instead of adding it after the first proposal.
   - Force the final answer with `tool_choice: none`, not `tools = nil`.
   - **What it saves.** Take a turn of six model calls over a 20k-token prefix:
     - Haiku 5.5: input cost falls from about $0.012 to about $0.0035, a 70% saving.
     - Opus 5.5: about $0.48 to about $0.12.
     - Time to first token falls on every call, which matters more at Haiku prices.
   - **It also removes the Haiku 5.5 history-editing risk** (§1.4 item 6), because nothing edits
     `tools` mid-turn any more. Of every item here, this has the best gain for its risk.
2. **Cut the fixed overhead in every request.**
   - Keep three or four tools always loaded (`find_tools`, `ask_user` where a person is present,
     and the one or two most-used domain reads for the profile), and defer the rest of today's
     10 core tools.
   - Shrink `_why` from three 200-character properties to one optional short `why` string. It is
     about 150 tokens per tool today (§1.5), and one string keeps its audit value at a fraction
     of the cost.
   - Never put the catalog in the volatile part (the palette's ~7k tokens).
3. **Stop refusals and guard re-asks from sinking turns.**
   - Count invalid-argument refusals against a separate small allowance (for example 3), not the
     main budget, so a corrected call does not shorten the task. The repeat guard still stops
     loops.
   - Send grounding and placement re-asks as harness system messages.
   - Measure each guard's fire rate and the extra calls it causes.
4. **Write the goal down, so preselection lands and the first `find_tools` disappears** (§3.3).
   On LoadMonitor alone, this probably removes one or two steps from every scheduled run.
5. **Respect Haiku 5.5's 100k-token price cliff in long unattended runs.**
   - Use context editing (`clear_tool_uses`, server-side, which keeps thinking valid) or tighter
     caps on list results.
   - Offer a `response_format: concise` option on large reads.
   - Alarm on prompts over 90k tokens.
6. **Tune effort per route.**
   - Haiku 5.5 at `low` stops early in long prompts. Sweep `low` against `medium` on the bench
     before choosing.
   - On OpenAI models, reasoning effort was the largest single lever on pass^k (GPT-5.2 airline
     pass^4: 22% at none, 72% at high).
7. **Escalate instead of upgrading.** Use Anthropic's advisor tool (Haiku executor, Opus
   advisor, `max_uses` capped) on the hard decisions, rather than running whole agents on Opus.
   The evidence is a vendor post (BrowseComp roughly doubled at 85% less than Sonnet alone), so
   test it on the bench.

### 3.5 What else matters

- **Multi-provider.**
  - On OpenAI, the equivalents are `allowed_tools` (keeps the cached list and restricts what can
    be called) and client-executed `tool_search` (gpt-5.4 and later) for `find_tools`.
  - GPT-6 Astra and GPT-6.1 Sol need the Responses API for tool calling.
  - The tool error flag should reach OpenAI and other protocols as structured output, not only
    as text.
  - **Ollama:** remove it, or mark it unsupported for agents. It has no users, at least two
    defects, and no coverage on the bench. Keep the OpenAI-compatible protocol for gateways, and
    label agent support as "unbenched".
- **Evaluation.** The priority surface has no bench coverage (§3.0). Rubrics are not scored.
  Two models were ever benched. Cache tokens are dropped. These must be fixed before any of
  the above can be shown to work (§6).
- **Observability.**
  - Add cache-read ratio and prompt size per call.
  - Add a `model` and a `provider` label to the tool-outcome counter.
  - Add first-correct-tool position and the number of `find_tools` calls before it.
  - Record refusals made before dispatch in the ledger.
- **Latency.**
  - Unattended runs already skip the scope classifier, which runs for up to 3 seconds before
    each chat turn and is not on the run path.
  - If the classifier's job widens to producing the task frame (§3.3), its 3-second budget
    applies to it too. Caching cuts time to first token on every model call.
- **Safety, for anything new.**
  - Code mode must reach effects only through a broker that calls `dispatch`.
  - Writes from code are outbox intents, folded into one batch proposal and re-previewed when
    the person decides (§2.3). Desk's stale-version refusal at decision time already provides
    this check for discrete writes.
  - Taint labels are attached by the broker, not trusted to the model.
  - Separately, decide on the two safety gaps already found: money tools can run automatically,
    and an approval without a digest is accepted (§1.6).
- **Cost is now a quality lever, not a constraint.** At $0.10 per million input tokens, a Haiku
  turn costs fractions of a cent. Spend tokens where they buy reliability: the domain primer,
  playbooks, effort, the advisor. Measure cost per *completed* task, not per call.

## 4. Recommendation

**Move to one agent runtime with job profiles, a stable cached prefix, a deferred catalog found
through a written need, a per-request task frame, and a smaller, de-duplicated tool surface. Add
a code mode for read-heavy and batch work later, gated by evals. Keep delegation only for
parallel, read-only fan-out. Keep `dispatch` and the approval model exactly as they are.**

How each part maps to today:

1. **Job profiles replace "agents" as the unit the model sees, but not as the product unit.**
   An agent definition keeps what makes it valuable as a product: its job description,
   permissions, autonomy ceiling, budget, schedules, events and allowlist. What changes is the
   context built from it:
   - The job description becomes the profile's instructions.
   - Its tools become the profile's non-deferred core plus its deferred namespaces.
   - Its routine work becomes playbooks.
   - A single front door for chat (§1.10 answer 2) is then a profile switch inside one runtime,
     not a hand-off between agents:
     - The router's confusion between neighbouring desks becomes a soft default, because any
       profile can still find another namespace's tools through `find_tools`, subject to the
       person's permissions and the profile's tier ceilings.
     - Unattended runs pick their profile deterministically, from the event or the schedule.
2. **A stable prefix.** Each call's prefix is built from the same parts in the same order:
   - platform core, domain primer, tool map, job profile and organization profile in `system`;
   - one sorted tool array, mostly deferred;
   - a cache breakpoint;
   - then the task frame, records in play and history in `messages`.

   The tools change only through `tool_reference` (Anthropic) or `allowed_tools` / `tool_search`
   (OpenAI).
3. **Discovery through a written need.** Preselection ranks on the task frame's need, not the raw
   message:
   - unattended runs: the need comes from the playbook;
   - chat: the need comes from the classifier.

   `find_tools` stays as the agent's own search, answering with tool references. Its answers
   carry namespaces. Its quality is measured as its own eval (§6).
4. **A consolidated tool surface.** This reaches the same functionality with fewer, distinct
   tools:
   - Merge single/batch pairs into one tool that takes a list.
   - Merge `list_*`/`search_*` overlaps into one read per entity, with typed filters.
   - Namespace names by module (`shipments_`, `billing_`, …).
   - Settle one convention per concept: pagination, ids, notes, date ranges.
   - Replace the untyped list filter with per-tool typed filters, so the field documentation
     leaves the prose.
   - Rewrite descriptions by trace-driven iteration (DRAFT/Play2Prompt style), scored on a
     held-out part of the bench.

   Every existing capability survives. Write coverage (`writecoverage.yml`) and the contract
   tests are the guard that nothing is lost.
5. **Code mode, later and gated.**
   - A read-only Trenova SDK runs in the existing E2B sandbox, with no egress except an RPC
     binding to a Go broker that calls `dispatch`.
   - It is offered for fan-out, reconciliation and report-style work.
   - Write intents go to an outbox that becomes one batch proposal, re-previewed at commit.
   - It is enabled per model only where the bench shows a gain, since Haiku 4.5 regressed on
     chains (§2.3).
   - A stronger model can write reusable scripts for recurring scheduled runs, which Haiku then
     runs with parameters (the LATM idea).

**Alternatives rejected, and why:**

- **Keep the 29 agents and only tune descriptions.** It leaves the overlap, the uncached prefix,
  the boilerplate inputs to unattended runs, and the cost of misrouting in place.
- **A deeper orchestrator with specialist sub-agents.** The evidence is against it for
  sequential, tool-heavy work, and it costs 3–5× the tokens (§2.4).
- **Code-as-action everywhere.** The model-specific regressions and the null result on τ² rule it
  out, and writes would become harder to govern (§2.3).
- **The provider's own BM25 or regex tool search.** Desk's own hybrid ranker already handles
  permissions, families and hand-offs. Use the custom `tool_reference` path instead, which keeps
  that logic.
- **Fine-tuning, or tools as tokens (ToolGen).** Not possible on hosted models, and it cannot
  follow a catalog that changes.
- **Logit masking (Manus).** Hosted APIs do not expose it. `allowed_tools` and `defer_loading`
  are the hosted equivalents.

**What to keep unchanged.** These are working, and the plan builds on them:

- the `dispatch` chokepoint, policy-in-code tiers, preview-digest approvals and the step ledger;
- taint;
- argument repair that reports back to the model;
- concurrent reads;
- records in play and world-change notices;
- the bench, the findings discipline, prompt snapshot tests, wire tests and the eval gate;
- Temporal durability.

## 5. Phased plan

Each phase lists what it changes and how it is checked. The checks refer to the eval plan in §6.
No phase removes a capability.

**Phase 0: measure, and fix correctness (1–2 weeks).**

- **Measurement.**
  - Carry cache read and write tokens through `ChatCompletionResult`, the bench and
    `ai_work_units`, and price the bench providers.
  - Average the bench matrix rather than summing it, and add pass^k (k = 4).
  - Add `model` and `provider` labels to the tool-outcome counter.
  - Turn on `ai.cacheDiagnostics` for a day, as the owner offered, and count `tools_changed`.
- **Correctness.**
  - Test the Haiku 5.5 binding risk on an account created after 2026-08-31. If it reproduces,
    set `bindsPrefix` for Haiku 5.5 as a stopgap.
  - Use `tool_choice: none` for the final answer.
  - Declare `withdraw_proposal` from the start of the turn.
  - Give Ollama `num_ctx`, or remove Ollama.
- **Coverage of the priority surface.** Add unattended-run scenarios to the bench: event and
  scheduled, for at least LoadMonitor, Billing exception, Detention, Intake, EDI and Dispatch
  assignment.
- **Baseline.** Run the matrix on Haiku 5.5 (low and medium effort), Sonnet 5.5, gpt-6-luna and
  gpt-6.1-sol, 4 trials each.

**Phase 1: a stable prefix and deferred loading (2–3 weeks). Expected to be the biggest single
gain.**

- Send a sorted, stable tool array per agent, with a small non-deferred core and the rest
  `defer_loading`.
- `find_tools` answers with `tool_reference` blocks on Anthropic, and with `allowed_tools` or
  client `tool_search` on OpenAI.
- Move the catalog listing into a stable namespace map in `system`.
- Move the person profile to the stable part.
- Shrink `_why` to one optional string.
- **Check:** cache-read ratio above 80% on the second and later calls of a turn; no loss in pass^k
  or tool-selection accuracy; steps per task and time to first token fall.
- **Built (2026-10-19, PR #822; `_why` already gone in #815).** See §8.3.

**Phase 2: the task frame and the domain primer (2–3 weeks).**

- Build a task frame for unattended runs from an event → playbook table.
- Extend the classifier's schema to emit the goal and a need for chat turns.
- Rank preselection on the need.
- Add the domain primer.
- Move enforced rules out of the prompt, and move tool-specific rules into tool descriptions.
- Deliver steering, world-change and guard notices as mid-conversation system messages where the
  model supports them.
- **Check:** recall@8 of preselection against the tools actually used; how many runs begin with
  `find_tools`; pass^k on the new unattended scenarios.
- **Built (2026-10-19, the task-frame PR stacked on #822).** See §8.3.

**Phase 3: consolidate the tool surface (4–6 weeks, rolling).**

- Work module by module:
  - merge the overlaps;
  - namespace the names;
  - make the conventions consistent;
  - type the filters;
  - rewrite descriptions against traces.
- Keep a held-out split of the bench so description edits are not fitted to the scenarios they
  are scored on.
- **Check:** tool count down, write coverage unchanged, the contract tests passing; selection
  accuracy and argument validity up on held-out cases.

**Phase 3 progress.** The held-out split is in place before any tool changes: scenarios and
`find_tools` cases are held out by a fixed rule, a quarter of each desk, chat and unattended apart
(`desk-bench.md`, "Labels and splits"). The held-out figures below are the eval gate's `find_tools`
ranking by keyword and meaning over the held-out cases, read from the gate's log after each PR.
Argument validity on held-out scenarios is read from a bench run (`trenova desk run --split
held-out`) when one is made for a PR.

| PR | Modules | Tools before → after | Write coverage (tools / exempt / pending) | Held-out recall@5 | Held-out top-1 |
|---|---|---|---|---|---|
| Held-out split | none (measurement only) | 649 → 649 | 459 / 710 / 0, unchanged | 0.92 (161 cases) | 0.81 |
| Shipments, orders, import page | `shipment`, `order`, `intake` (55 tools namespaced) | 649 → 648 | 459 / 710 / 0, unchanged | 0.9193 → 0.9193 | 0.8137 → 0.8137 |
| Billing queue and invoices | `billing`, `invoice` (47 tools namespaced; 7 single/batch pairs merged) | 648 → 641 | 459 / 710 / 0, unchanged | 0.9193 → 0.9193 | 0.8137 → 0.8137 |
| Settlements and pay | `settlement` (56 tools namespaced; 4 single/batch pairs merged) | 641 → 637 | 459 / 710 / 0, unchanged | 0.9193 → 0.9193 | 0.8137 → 0.8137 |
| Receivables | `receivable` (22 tools namespaced; no merges) | 637 → 637 | unchanged, 0 pending | 0.91 → 0.91 | 0.81 → 0.81 |
| Every remaining tool | 23 more namespaces, `accounting` to `worker` (450 tools namespaced; no merges) | 637 → 637 | unchanged, 0 pending | 0.9141 → 0.9202 | 0.8098 → 0.8344 |
| Overlapping tools | `worker`, `inbox`, `accounting`, `agent` (5 merges) | 637 → 632 | unchanged, 0 pending | 0.9202 → 0.93 | 0.8344 → 0.82 |
| Typed reads through GraphQL: reference data | `equipment_list_manufacturers`, `equipment_list_types`, `document_list_types`, `location_list_categories` and seven `masterdata` lists read their screen's persisted operation | 632 → 632 | unchanged, 0 pending | 0.93 → 0.93 | 0.82 → 0.82 |
| Typed reads through GraphQL: parties | `customer_list`, `location_list`, `carrier_list` | 632 → 632 | unchanged, 0 pending | 0.93 → 0.93 | 0.82 → 0.82 |
| Typed reads through GraphQL: equipment | `equipment_list_tractors`, `equipment_list_trailers` | 632 → 632 | unchanged, 0 pending | 0.93 → 0.93 | 0.82 → 0.82 |
| Typed reads through GraphQL: workers | `worker_list` | 632 → 632 | unchanged, 0 pending | 0.93 → 0.93 | 0.82 → 0.82 |
| Typed reads through GraphQL: time off | `worker_list_time_off` | 632 → 632 | unchanged, 0 pending | 0.93 → 0.93 | 0.82 → 0.82 |
| Typed reads through GraphQL: one worker's records | `worker_list_checklists`, `worker_list_training`, `worker_list_performance_reviews` | 632 → 632 | unchanged, 0 pending | 0.93 → 0.93 | 0.82 → 0.82 |
| Typed reads through GraphQL: injuries and drug tests | `worker_list_injuries`, `worker_list_dot_tests` | 632 → 632 | unchanged, 0 pending | 0.93 → 0.93 | 0.82 → 0.82 |
| Typed reads through GraphQL: leave and previous employers | `worker_list_leave_cases`, `worker_list_employment_verifications` | 632 → 632 | unchanged, 0 pending | 0.93 → 0.93 | 0.82 → 0.82 |

The shipment row is measured on the held-out set the split PR recorded (161 cases), so both
figures read the same cases; the module's merges moved three cases between splits, which the
ledger now records. On the development cases the same PR moved hybrid top-1 from 0.7702 to
0.7745 and recall@5 from 0.9149 to 0.9128 (one case, an accounting request whose ranking moved
with the renamed tools' embeddings), and keyword recall@5 over every case from 0.9287 to 0.9319.

The billing row reads the same 161 cases. On the development cases it moved hybrid top-1 from
0.7745 to 0.7766 and recall@5 from 0.9128 to 0.9149, and keyword top-1 from 0.8830 to 0.8872 with
recall@5 from 0.9255 to 0.9213 (two cases). Two causes found on the way are fixed for every
later module: keyword ties broke on the alphabetical name, so a namespace prefix (`billing_`)
decided which of two equal answers came first, and they now break on the name in reading order;
and a name that carries another area's noun (`billing_transfer_shipments`) name-matched every
question about that noun. The merges moved two cases between strata; one, a miss in both, joined
the held-out set, so the gate's own held-out hybrid top-1 reads 132 of 163 (0.8098) where it read
132 of 162, and its floor is written as 0.80.

The settlement row reads the same 161 cases. Held-out keyword ranking is unchanged as well
(recall@5 0.9503, top-1 0.9130). On the development cases hybrid top-1 moved from 0.7766 to
0.7787 and keyword recall@5 from 0.9213 to 0.9234, while keyword top-1 fell from 0.8872 to
0.8809: three requests ("hold the driver's pay", "post every approved carrier settlement",
"generate this pay period's settlements for our carriers") had matched words the old names or
the batch tools' generic search terms ("every", "once") carried. Settlement records come in two
kinds, so the namespace declares the qualifiers `driver` and `carrier`, and a name is indexed in
reading order too, which gave the list tools back their plural.

The receivables row reads the hybrid gate's 163 held-out cases. Held-out keyword top-1 fell
from 1.00 to 0.993: one of 143 cases now misses, measured with and without the search term below,
and the held-out floor is written as 0.99 rather than tuned against. On the development cases the
singular `receivable_list_collection_worklist` lost the plural the old name carried and fell to fifth
for "who should collections call first today"; its search term "collections worklist" gives it
back. Bank receipts and customer payments keep their own names in reading order through the
namespace's `Records` (`receivable_get_bank_receipt` reads `get_bank_receipt`), so the Cash
application agent's subject read, cards and record links are unchanged.

The last row namespaces every tool still outside one, so the ratchet now holds at zero and a new
tool without a namespace fails it. Kinds of record a namespace names on its own (tractors,
trailers, commodities, journal entries, fiscal periods, tenders and two dozen more) are its
`Records`, so cards, subject reads and record links read as before. The hybrid figures above
read the gate's 163 held-out cases. By keyword, held-out recall@5 fell from 1.00 to 0.993 (one
case: "shop carriers for a rate on this lane" no longer finds `carrier_shop` in the top five)
and top-1 from 0.993 to 0.951 (seven cases); none is tuned against, and the floors are written
as 0.99 and 0.95. On the development cases six tools lost a word their old name carried and get
it back as a search term (`agent_create_memory` "remember", `agent_retire_memory` "forget
memory", `dispatch_create_tender` "tender to carriers", `rate_quote` "quote shipment",
`ledger_create_manual_journal` "draft manual journal", `fuel_recalculate_jurisdiction_miles`
"move jurisdiction miles"). Three development requests still rank a different tool first,
because that tool's new name gained one of the request's words; the expected tool lost nothing,
so no term was added for them. Model-facing text that still named `remember` now names
`agent_create_memory`, and a red-team script that called the removed name was corrected. What
remains of Phase 3 is the merges: overlapping tools found while writing the rename map (worker
search into the worker list, single and batch pairs, carrier intelligence acknowledge and
resolve).

The merges row folds five tools into the tool that reads or writes the same records under the
same permission: `worker_search` into `worker_list`, which ranks the closest name first;
`worker_list_expiring_credentials` into `worker_list_credentials`, which lists one worker's
credentials or what falls due across the organization, soonest first; `inbox_search_inbound_messages`
into `inbox_list_inbound_messages`, whose query ranks mail by what it says under the list's
filters; `accounting_get_sync_state` into `accounting_list_sync_records` through `documentIds`;
and `agent_request_review` into `agent_raise_exception`, which takes evidence. The held-out sets
grew by two cases each as the merged tools' cases moved between strata. By keyword, held-out
recall@5 holds at 0.99 and top-1 at 0.95 over 145 cases. With meaning, held-out recall@5 rose to
0.93 and top-1 fell to 0.82 over 165, written as the floor rather than tuned against; among the
misses, "reply to the inbound email from the broker" now ranks the inbox list above
`inbox_send_reply`, because the list carries the search tool's mail vocabulary. On the development
cases, the merged tools' text was corrected where a merge cost a request its answer: the exception
tool's first sentence says it flags a case for a person to review, the credential list's "medical card
expiry" term, which drew requests to record a new card, became "about to lapse", the sync record list dropped a "reach" that pulled in EDI
deliveries, and the worker list takes back "find a driver". "Look up David Park as a driver by
name" still misses with meaning, because "park" draws the wait tool first. Pairs that look alike
but differ in what they do stay apart: generating one driver's settlement and the period's batch,
accepting a carrier invoice match with or without its variance, acknowledging and resolving a
carrier finding, and confirming proposed accounting mappings against setting one.

The reference data row is the first of the typed-reads modules
([agent-graphql-reads.md](engineering/agent-graphql-reads.md)): eleven lists now read the
persisted operation their page sends, through the read executor, with typed parameters derived
from the record's filter configuration in place of the generic `filters`, and the cursor paging
the paging fix put in place. Each description now names its typed parameters, as the repository
lists' did. Over 165 held-out cases, hybrid recall@5 holds at 0.93 and top-1 at 0.82; by keyword,
over 145, recall@5 holds at 0.99 and top-1 at 0.95. The floors are unchanged, and nothing was tuned
against the held-out cases.

The parties row converts the customer, location and carrier lists the same way. Their
descriptions now name every filterable field of the record, the carrier's forty-odd among them,
and held-out ranking is unchanged: hybrid recall@5 0.93 and top-1 0.82 over 165 cases, keyword
0.99 and 0.95 over 145.

The equipment row converts the tractor and trailer lists. Converting the tractor list found that
the repository tools made a check the screen does not: a tractor's driver was named only to a
reader who could read workers. An agent read now gates a nested record by its own kind's read
permission and shows only its id where that is refused. Held-out ranking is unchanged.

The workers row converts `worker_list`, whose kind is narrowed by a team scope. A Postgres test
reads both paths as a person whose worker grant reaches only their team and gets the same two
workers from each. The list keeps the repository row's rules as named hooks: the closest name
first for a name search, and a NonCompliant credential folded into `canBeAssigned`. Its profile
filters (endorsement, CDL class, compliance, credential dates) are derived from the profile's own
fields. Held-out ranking is unchanged.

The time off row converts `worker_list_time_off`, whose default window reads leave that overlaps
the next two weeks, including leave that began earlier and is still running, which the screen's
start-date bounds could not say. `WorkerPTOEntriesInput` gains an additive `overlapping` field that
the resolver passes to the repository's existing overlap filter (`gqlschema-diff`: one safe change),
with a Postgres test that leave begun before the window is read only when it overlaps. Converting
it also found that `worker_pto` declared no field sensitivities, so its Restricted default would
have withheld every field at the default agent data access; the operational fields are now
declared Internal and a leave's reasons stay Restricted. Held-out ranking is unchanged.

The one worker's records row adds the third tool shape: a list whose root field returns every
record at once, keyed by the worker, whose parameters are the operation's own variables
(`workerId`, `includeClosed`) rather than a table's filters. Checklists, training assignments and
performance reviews read new `Desk*` operations on the worker's own root fields, and the checklist
and review lists read the active templates alongside, through operations the generator proves,
only when the reader may read templates, as the repository tools did. A training assignment's
course is a linked record and shows its reference view. A Postgres test reads each list under
row-level security against the service read the repository tool made, with and without closed
records, and finds nothing of another tenant's worker. Descriptions are unchanged, so the
embedding fixture is current; held-out ranking is unchanged (keyword 0.99 and 0.95 over 145, hybrid
0.93 and 0.82 over 165).

The injuries and drug tests row converts two of the partial reads. `workerInjuries` and
`workerDotTests` take the worker as optional, with the repository tools' filters (`caseYear`,
`recordableOnly`, `openOnly`, `limit`) added to the schema (`gqlschema-diff`: eight safe changes),
so one worker's cases or the whole log read through the screen's root field. Neither kind declared
field sensitivities: injuries now keep what happened to the person Restricted while the posted log
is Internal, and, by the owner's decision, a drug or alcohol test's result, substances and
collection details are Restricted where they had been Internal. Leave cases and employment
verifications stay on their repository path: their GraphQL types have no kind, so the field guard
would not gate them, and they convert with `@kind` and field sensitivities after reference views.
The drug test description now says the result can be withheld, so the embedding fixture is
re-recorded; held-out ranking is unchanged (keyword 0.99 and 0.95 over 145, hybrid 0.93 and 0.82
over 165).

The leave and previous employers row converts the last two of those reads. Their GraphQL types had
no kind, so the field guard would have shown a leave reason or a previous employer's findings at
any ceiling; `WorkerLeaveCase` now declares `@kind(resource: "worker_leave")` and
`WorkerEmploymentVerification` `@kind(resource: "qualification")`, the qualification file declares
that it authorizes the verifications it carries, both kinds declare which fields are Internal, and
the REST response guard names both structs. `qualification` was not given worker as its parent:
the permission engine reads a child through its parent's grant, so worker read would have granted
every qualification record. A Postgres test reads each list for one worker and for the
organization, against the repository tool's service read, and checks that an agent at Internal is
never shown a leave reason, a finding or a drug and alcohol answer. Held-out ranking is unchanged
(keyword 0.99 and 0.95 over 145, hybrid 0.93 and 0.82 over 165).

**Phase 4: a single front door and job profiles (3–4 weeks).**

- One runtime: an agent definition supplies a profile.
- Chat routing becomes a default profile with discovery across namespaces.
- Delegation is restricted to read-only fan-out, and returns structured, verbatim results.
- **Check:** the routing bench becomes a profile-selection bench, with its confusion matrix
  reported; end-to-end pass^k on cross-desk requests ("bill it and email the customer").
- **Built (2026-10-19, PR #821).** See §8.3.

**Phase 5: code mode for reads, then batch writes (4–6 weeks).**

- A read-only SDK through a broker, in the existing E2B sandbox.
- Next, write intents to an outbox that becomes a batch proposal, re-previewed at commit.
- Turned on per chat provider in AI Control: "Scripts", and "Scripts may propose writes", which
  requires it. Both are off by default and read at dispatch from the provider that answered,
  beside the organization's own opt-in (plan capability, E2B setting, permission).
- **Check:** paired scenarios in discrete mode and code mode, mixing fan-out reads, batch writes
  and τ-style sequential controls; adversarial cases (injection, sandbox escape, stale approval).
  The bar is zero unauthorized commits. The bench gate measures them and recommends the
  settings per model; it changes none ([desk-bench-gate.md](engineering/desk-bench-gate.md)).
- **Built (2026-10-20, PRs #837, #838, #849); settings on the provider since 2026-10-21.** See
  [agent-code-mode.md](engineering/agent-code-mode.md) and
  [agent-code-mode-writes.md](engineering/agent-code-mode-writes.md).

**Phase 6: tune the models (ongoing).**

- Set effort per route on the provider; the bench gate recommends one.
- The advisor (Haiku with Opus) on hard steps is a provider setting ("Advisor"), off by
  default; the gate recommends it per route where it helps, and the API's own refusal of an
  advisor is recorded on the provider and the call answered without it
  ([agent-runtime.md](engineering/agent-runtime.md), "The advisor").
- Run the full matrix every quarter, and whenever a provider ships a model, as measurement:
  its output is a recommended setting, never a switch.

## 6. Eval plan

**The set.** Build "Desk tasks" from three sources:

1. **The existing 134 bench scenarios (202 steps)**, kept, with their rubrics scored for the
   first time.
2. **New unattended-run scenarios.** Each is an event or schedule fired against seeded state, at
   least 30 at first, covering every event-driven template and LoadMonitor. Grade the outcome:
   database state, proposals filed with their arguments, records flagged. Do not grade the path.
3. **Real failures.** Every finding in `findings.yaml` with a reproducible scenario, plus 20–50
   requests from production threads with thumbs-down or rejected proposals, anonymized.

Split the set three ways:

- a **development** set, for iterating on descriptions and prompts;
- a **held-out** set, for decisions;
- an adversarial set: injection through emails or EDI, sandbox probes, stale approvals, money
  writes.

Label each task's expected tool set, the tools it must not call, and its success state.

**Metrics, per task, and per provider × model × effort:**

| Metric | Definition | Source |
|---|---|---|
| Task success, pass^1 and pass^4 | Outcome checks pass in all k trials | Bench checks (`db`, `proposals`, `executed`, `facts`) and the scored rubric |
| Tool-selection accuracy | Precision and recall of tools called against the labelled expected set; position of the first correct tool; calls to forbidden tools | Bench trajectory |
| Retrieval quality | Recall@8 of preselection, and recall of each `find_tools` answer, against the tools a passing trial used | Instrument `newToolSet` and `resolveFind` |
| Steps | Model calls per task; tool calls; refused calls by verdict; guard re-asks by guard | `ai_work_units`, bench record |
| Tokens | Uncached input, cache read, cache write and output, per task; peak prompt size | Fixed usage fields |
| Cost per completed task | Total cost ÷ trials that passed | Priced bench providers |
| Latency | Time to first token, and time per task | Existing metrics |
| Safety | Unauthorized commits (must be 0), tainted writes held, refusals on in-scope tasks | Red team plus the adversarial set |

**Comparisons to run, in the matrix:**

- **Model.** Haiku 5.5 at `low` / `medium`, Sonnet 5.5, Opus 5.5 as the ceiling, gpt-6-luna,
  gpt-6.1-sol.
- **Interventions,** one at a time against the baseline:
  - deferred and cached loading;
  - the task frame;
  - the written need;
  - a slim core;
  - `_why` as one string;
  - consolidated tools;
  - code mode.
- **Tool-count sweep.** Held tools at 8, 12, 20 and 30, crossed with each model, to set
  `disclosureThreshold` from data instead of the current 12.

**Rules:**

- Run 4 trials per cell.
- Report means with confidence intervals.
- Never ship a prompt or description change on the development set alone.
- Read transcripts from every failing cell.
- Keep LLM judges to rubric dimensions, one judge per dimension with an "Unknown" option,
  calibrated on 50 human-labelled transcripts before their scores count.
- Run the full matrix weekly while Phases 1–4 are under way, then monthly. Do not build on the
  OpenAI Evals platform, which shuts down on 2026-11-30.

**Cheap first steps:**

1. The tool-count sweep and the cache metrics need no new code paths beyond measurement.
2. The unattended scenarios reuse the bench's seeded database and agent sync.
3. A cassette-backed version of each passing trial gives a fast regression test that needs no
   model.

## 7. References

**Papers and benchmarks:**

- Wang et al., *Executable Code Actions Elicit Better LLM Agents* (CodeAct), ICML 2024. https://arxiv.org/abs/2402.01030
- Patel et al., *The Bitter Lesson of Tool Calling*, 2026. https://arxiv.org/abs/2608.06370
- Fei et al., *CodeDelegator*, 2026. https://arxiv.org/abs/2601.14914
- Debenedetti et al., *Defeating Prompt Injections by Design* (CaMeL), 2025. https://arxiv.org/abs/2503.18813
- Costa et al., *Securing AI Agents with Information-Flow Control* (FIDES), 2025. https://arxiv.org/abs/2505.23643
- Beurer-Kellner et al., *Design Patterns for Securing LLM Agents against Prompt Injections*, 2025. https://arxiv.org/abs/2506.08837
- Chen et al., *Cordon: Semantic Transactions for Tool-Using LLM Agents*, 2026. https://arxiv.org/abs/2606.17573
- Santos-Grueiro, *Commit-Time Authorization for LLM Agents*, 2026. https://arxiv.org/abs/2607.10487
- Shi et al., *Progent*, 2025. https://arxiv.org/abs/2504.11703
- Cai et al., *Large Language Models as Tool Makers*, ICLR 2024. https://arxiv.org/abs/2305.17126
- Shi et al., *Retrieval Models Aren't Tool-Savvy* (ToolRet), ACL 2025. https://arxiv.org/abs/2503.01763
- Gan & Sun, *RAG-MCP*, 2025. https://arxiv.org/abs/2505.03275
- Fei, Zheng, Feng, *MCP-Zero*, 2025. https://arxiv.org/abs/2506.01056
- Chen et al., *Re-Invoke*, EMNLP 2024 Findings. https://arxiv.org/abs/2408.01875
- Lumer et al., *Toolshed*, 2024. https://arxiv.org/abs/2410.14594
- Lumer et al., *ScaleMCP*, 2025. https://arxiv.org/abs/2505.06416
- Laddha et al., *HumanMCP*, 2025. https://arxiv.org/abs/2602.23367
- Sengupta et al., *ToolDreamer*, 2025. https://arxiv.org/abs/2510.19791
- Mo et al., *LiveMCPBench*, 2025. https://arxiv.org/abs/2508.01780
- Lei et al., *MCPVerse*, 2025. https://arxiv.org/abs/2508.16260
- Kate et al., *LongFuncEval*, 2025. https://arxiv.org/abs/2505.10570
- Rodrigues & Vas, *MCP Server Architecture Patterns for LLM-Integrated Applications*, 2026. https://arxiv.org/abs/2606.30317
- Repantis et al., *How Many Tools Should an LLM Agent See?*, 2026. https://arxiv.org/abs/2605.24660
- Paramanayakam et al., *Less is More*, DATE 2025. https://arxiv.org/abs/2411.15399
- Wang et al., *MCP-Bench*, ICLR 2026. https://arxiv.org/abs/2508.20453
- Luo et al., *MCP-Universe*, 2025. https://arxiv.org/abs/2508.14704
- Fang et al., *Play2Prompt*, ACL Findings 2025. https://arxiv.org/abs/2503.14432
- Qu et al., *From Exploration to Mastery* (DRAFT), ICLR 2025. https://arxiv.org/abs/2410.08197
- Guo et al., *Learning to Rewrite Tool Descriptions* (Trace-Free+), 2026. https://arxiv.org/abs/2602.20426
- Hasan et al., *MCP Tool Descriptions Are Smelly!*, 2026. https://arxiv.org/abs/2602.14878
- Faghih et al., *Tool Preferences in Agentic LLMs are Unreliable*, EMNLP 2025. https://arxiv.org/abs/2505.18135
- Lin et al., *Hammer*, ICLR 2025. https://arxiv.org/abs/2410.04587
- Liu et al., *ToolScope*, 2025. https://arxiv.org/abs/2510.20036
- Wang et al., *MCPTox*, AAAI 2026. https://arxiv.org/abs/2508.14925
- Patil et al., *The Berkeley Function Calling Leaderboard*, ICML 2025. https://proceedings.mlr.press/v267/patil25a.html ; leaderboard https://gorilla.cs.berkeley.edu/leaderboard.html
- Yao et al., *τ-bench*, 2024. https://arxiv.org/abs/2406.12045
- Barres et al., *τ²-bench*, 2025. https://arxiv.org/abs/2506.07982 ; leaderboard https://taubench.com
- Rabanser et al., *Towards a Science of AI Agent Reliability*, ICML 2026. https://arxiv.org/abs/2602.16666
- Sinha et al., *The Illusion of Diminishing Returns*, ICLR 2026. https://arxiv.org/abs/2509.09677
- Kwa et al. (METR), *Measuring AI Ability to Complete Long Tasks*, 2025. https://arxiv.org/abs/2503.14499
- Cemri et al., *Why Do Multi-Agent LLM Systems Fail?*, 2025. https://arxiv.org/abs/2503.13657
- Kim et al., *Towards a Science of Scaling Agent Systems*, 2025–26. https://arxiv.org/abs/2512.08296
- Li, *When Single-Agent with Skills Replace Multi-Agent Systems and When They Fail*, 2026. https://arxiv.org/abs/2601.04748
- Erdogan et al., *Plan-and-Act*, ICML 2025. https://arxiv.org/abs/2503.09572
- Xu et al., *ReWOO*, 2023. https://arxiv.org/abs/2305.18323
- Kim et al., *An LLM Compiler for Parallel Function Calling*, ICML 2024. https://arxiv.org/abs/2312.04511
- Huang et al., *Large Language Models Cannot Self-Correct Reasoning Yet*, ICLR 2024. https://arxiv.org/abs/2310.01798
- Kamoi et al., *When Can LLMs Actually Correct Their Own Mistakes?*, TACL 2024. https://arxiv.org/abs/2406.01297
- Ong et al., *RouteLLM*, 2024. https://arxiv.org/abs/2406.18665
- Hu et al., *RouterBench*, 2024. https://arxiv.org/abs/2403.12031
- Jaroslawicz et al., *How Many Instructions Can LLMs Follow at Once?* (IFScale), NeurIPS 2025. https://arxiv.org/abs/2507.11538 ; data https://distylai.github.io/IFScale/results.csv
- Qi et al., *AgentIF*, 2025. https://arxiv.org/abs/2505.16944
- Du et al., *Context Length Alone Hurts LLM Performance Despite Perfect Retrieval*, 2025. https://arxiv.org/abs/2510.05381
- Modarressi et al., *NoLiMa*, ICML 2025. https://arxiv.org/abs/2502.05167
- Liu et al., *Lost in the Middle*, TACL 2024. https://arxiv.org/abs/2307.03172
- Laban et al., *LLMs Get Lost in Multi-Turn Conversation*, 2025. https://arxiv.org/abs/2505.06120
- Zhang et al., *Agentic Context Engineering* (ACE), ICLR 2026. https://arxiv.org/abs/2510.04618
- Wang et al., *Agent Workflow Memory*, 2024. https://arxiv.org/abs/2409.07429
- Barbaste et al., *Harness Engineering: a source-code study of eleven systems*, 2026. https://arxiv.org/abs/2609.00006

**Vendor documentation and engineering posts:**

- Anthropic, *Tool search tool*. https://platform.claude.com/docs/en/agents-and-tools/tool-use/tool-search-tool
- Anthropic, *Programmatic tool calling*. https://platform.claude.com/docs/en/agents-and-tools/tool-use/programmatic-tool-calling
- Anthropic, *Define tools*. https://platform.claude.com/docs/en/agents-and-tools/tool-use/define-tools
- Anthropic, *Prompt caching*. https://platform.claude.com/docs/en/build-with-claude/prompt-caching
- Anthropic, *Tool use with prompt caching*. https://platform.claude.com/docs/en/agents-and-tools/tool-use/tool-use-with-prompt-caching
- Anthropic, *Preserved thinking*. https://platform.claude.com/docs/en/build-with-claude/preserved-thinking
- Anthropic, *Context editing*. https://platform.claude.com/docs/en/build-with-claude/context-editing
- Anthropic, *Pricing*. https://platform.claude.com/docs/en/about-claude/pricing
- Anthropic, *Prompting best practices*. https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices
- Anthropic, *Introducing advanced tool use*, 2025-11-24. https://www.anthropic.com/engineering/advanced-tool-use
- Anthropic, *Code execution with MCP*, 2025-11-04. https://www.anthropic.com/engineering/code-execution-with-mcp
- Anthropic, *Writing effective tools for agents*, 2025-09-11. https://www.anthropic.com/engineering/writing-tools-for-agents
- Anthropic, *Effective context engineering for AI agents*, 2025-09-29. https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents
- Anthropic, *Building effective agents*, 2024-12-19. https://www.anthropic.com/engineering/building-effective-agents
- Anthropic, *How we built our multi-agent research system*, 2025-06-13. https://www.anthropic.com/engineering/multi-agent-research-system
- Anthropic, *Equipping agents for the real world with Agent Skills*, 2025-10-16. https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills
- Anthropic, *The "think" tool*, 2025 (updated 2025-12-15). https://www.anthropic.com/engineering/claude-think-tool
- Anthropic, *Demystifying evals for AI agents*, 2026-01-09. https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents
- Anthropic, *The advisor strategy*, 2026-04-09. https://claude.com/blog/the-advisor-strategy
- Anthropic, *Claude Haiku 5.5*, 2026-10-07. https://www.anthropic.com/news/claude-haiku-5-5
- OpenAI, *Function calling* (`allowed_tools`, strict mode). https://developers.openai.com/api/docs/guides/function-calling
- OpenAI, *Tool search*. https://developers.openai.com/api/docs/guides/tools-tool-search
- OpenAI, *Programmatic tool calling*. https://developers.openai.com/api/docs/guides/tools-programmatic-tool-calling
- OpenAI, *Prompt caching*. https://developers.openai.com/api/docs/guides/prompt-caching
- OpenAI, *Reasoning models*. https://developers.openai.com/api/docs/guides/reasoning
- OpenAI, *GPT-5 prompting guide* (archived). https://developers.openai.com/cookbook/examples/gpt-5/gpt-5_prompting_guide
- OpenAI, *A practical guide to building agents*. https://cdn.openai.com/business-guides-and-resources/a-practical-guide-to-building-agents.pdf
- OpenAI, *Evaluation best practices*. https://developers.openai.com/api/docs/guides/evaluation-best-practices
- Google, *Gemini function calling*. https://ai.google.dev/gemini-api/docs/function-calling
- Cloudflare, *Code Mode: the better way to use MCP*, 2025-09-26. https://blog.cloudflare.com/code-mode/
- Cloudflare, *Code Mode: give agents an entire API in 1,000 tokens*, 2026-02-20. https://blog.cloudflare.com/code-mode-mcp/

**Practitioner posts:**

- Manus, *Context Engineering for AI Agents: Lessons from Building Manus*, 2025-07-18. https://manus.im/blog/Context-Engineering-for-AI-Agents-Lessons-from-Building-Manus
- Cognition, *Don't Build Multi-Agents*, 2025-06-12. https://cognition.com/blog/dont-build-multi-agents
- LangChain, *Benchmarking Multi-Agent Architectures*, 2025-06-10. https://www.langchain.com/blog/benchmarking-multi-agent-architectures
- Chroma, *Context Rot*, 2025. https://www.trychroma.com/research/context-rot
- Archestra, *Progressive disclosure for MCP tools: modeled vs measured*, 2026. https://archestra.ai/blog/progressive-disclosure-for-mcp-tools

## 8. Decisions and open questions (2026-10-18, after the review)

**Decisions made by the product owner:**

1. **Remove Ollama.**
2. **Close the write-safety gaps** in §1.6:
   - money tools that run automatically;
   - approvals accepted without a preview digest.
3. **Remove `_why`** from every tool schema.

**What a code scan found each decision touches.** These need settling before implementation.

1. **Ollama is used for more than chat.**
   - The adapter has an embeddings path (`modeladapter/embed_ollama.go`).
   - **The CI eval gate records its embedding baselines with Ollama**
     (`agentevalgate/embeddings.go`: `ollama pull nomic-embed-text`, `TRENOVA_EVAL_OLLAMA_URL`).
     The hybrid floors in `toolselection.floors.json` (top-1 0.78, recall@5 0.91) were measured
     against those vectors.
   - Removing Ollama means choosing another embedding model for the gate and re-recording the
     floors, or keeping a test-only embedder.
   - Ollama is also referenced in the seed providers, the provider catalog and prober, the
     pricing reference, the `ForPortableModel` schema path, tracing attributes, the web
     provider editor and its brand marks, the self-hosting docs and the self-host `config.yaml`
     (`ai.privateNetworkProviders`).
2. **The money gap is a trade-off against unattended automation.**
   - Capping the Money class at Ask first would make every customer-payment post and bank
     match wait for a person. That is the core of what the Cash application desk runs
     unattended, and unattended runs are the owner's first priority.
   - The alternative keeps those tools automatic, but only inside per-call conditions:
     - an exact match only;
     - an amount ceiling;
     - a per-run and per-day cap;
     - never after outside content.
   - That needs a decision on which bounds apply to which of the 7 tools.
3. **Unreviewed approval is deliberate today, so closing it is a design change, not a bug fix.**
   The client says so itself (`preview-gate.ts`): a batch "proposal they never opened goes
   without [a digest] and is recorded as approved unreviewed, which is the truth".
   - Approvals without a digest also arrive through the REST handlers (`agentproposalhandler`,
     `agentplanhandler`), which API keys and integrations use, and through plans.
   - Choices:
     - (a) require a digest everywhere, so bulk approval must show each preview;
     - (b) require a digest by class (money, customer, driver, external, tainted) and allow
       unreviewed approval only for internal writes;
     - (c) keep it and only surface it.

     I recommend (b).
4. **`_why` is a product feature, not only a token cost.**
   - The Desk renders it as the "why" toggle on each step (`desk-citations.tsx`, `StepWhyToggle`;
     `activity.ts`), and it is saved with the steps.
   - Removing it removes that UI, or needs a replacement. Anthropic's thinking display
     `"updates"` returns short progress notes between tool calls at no extra prompt cost, but
     only on Fable 5.1, Opus 5.5 and Sonnet 5.5, not Haiku 5.5. Haiku 5.5 can return summarized
     thinking. The model's own text before a call is another source.
   - The quality judge's "rationale" is its own field and is unaffected.

### 8.1 Constraint update: Trenova is not in production

The owner confirmed that Trenova has no production deployments. Migrations, saved data and
backward compatibility are therefore not constraints. What is built now sets the design. This
changes the plan in four ways.

1. **Delete the compatibility layer instead of building around it.** The agent module carries:
   - about 28 `workflow.GetVersion` or `fx.Supports` gates, over at least 19 change ids;
   - their old branches;
   - activities kept registered only for those branches (agent-runtime.md, "Waiting on in-flight
     executions");
   - 27 `@deprecated` GraphQL fields;
   - deprecated queries kept so "a tab still on the previous release" works
     (`myAgentRecommendation`).

   Every change to the loop today pays for a new gate and a new old branch. Before launch:
   - remove the gates and their old branches;
   - reset the development Temporal namespace when workflow code changes shape;
   - drop the deprecated fields and queries.

   Put versioning back as a launch requirement. This makes the restructuring in §5 much
   cheaper, because most of it is loop and context-assembly code.
2. **Rename and merge tools freely.** Tool names, parameter names and schemas can change without
   aliases:
   - namespaced names;
   - merged single/batch pairs;
   - one convention for ids, pagination, notes and dates;
   - typed filters.

   Agent definitions, seeds and bench fixtures are regenerated. No saved data has to survive.
   Write coverage and the contract tests remain the check that no capability is lost.
3. **Merge the phases.** §5 phased the work partly to protect running executions and stored
   conversations, and that reason is gone. The revised order:
   1. **Baseline first** (Phase 0). It is still needed: without it, nobody can tell whether the
      restructure helped. That includes the measurement fixes, unattended-run scenarios on the
      bench, and the 4-trial matrix.
   2. **One rebuild of context assembly** (Phases 1, 2 and 4 together):
      - job profiles in one runtime;
      - the stable cached prefix with the domain primer and namespace map;
      - the deferred catalog through `tool_reference` and `allowed_tools`;
      - the task frame;
      - no `_why`;
      - `tool_choice: none` for the final answer;
      - no mid-turn changes to `tools`.

      Behind it, the bench is compared against the baseline.
   3. **Tool surface consolidation** (Phase 3), module by module, with no aliases.
   4. **Code mode** (Phase 5), a provider setting off by default, measured by eval for each model.
   5. **Model tuning** (Phase 6).
4. **Evals cannot draw on production traffic.** No production threads, thumbs or decided
   proposals exist to mine. The eval set (§6) is built from:
   - the bench's 134 scenarios;
   - the findings log;
   - new unattended-run scenarios;
   - the owner's own use of a seeded environment.

   The held-out split matters more for that reason: without real traffic, the risk of fitting
   descriptions to the bench is higher.

**Decisions still open:**

1. **Ollama embeddings.** Removing Ollama includes its embeddings. The CI eval gate then needs
   another embedding model to record its floors, for example the embedding model the
   organization's retrieval settings use. The floors are re-recorded once.
2. **The money tools:** a person on every call, or automatic inside per-call bounds (§8 item 2).
3. **The preview-digest rule:** required everywhere, required by class (recommended), or only
   surfaced (§8 item 3).
4. **The step "why" toggle:** removed with `_why`, or fed from thinking summaries and progress
   notes where the model provides them (§8 item 4).

### 8.2 Decided (2026-10-18): specifications to build from

**1. Remove Ollama from the application, and keep it only for recording eval fixtures.**

- **Application.** Remove:
  - the `Ollama` provider kind and its adapters (`modeladapter/ollama.go`, `embed_ollama.go`);
  - their registration in `modeladapter/adapter.go`;
  - its branches in the completion router, the prober, the pricing reference, the vendor list
    and tracing attributes;
  - the `ForPortableModel` callers that existed for it;
  - the provider editor's option and brand mark;
  - the development seed;
  - the self-hosting docs and the `config.yaml` comments.

  `ai.privateNetworkProviders` stays for OpenAI-compatible servers on a private network.
- **CI and evals.** CI reads recorded fixtures (`evals/embeddings/nomic-embed-text.json`) and
  needs no Ollama. Only the two recorders call Ollama, and both go through the application's
  adapter registry today:
  - `agentevalgate/embeddings_record_test.go`;
  - `retrievalservice/record_fixture_test.go` via `testutil/retrievaltest`.

  Move a minimal Ollama embedding client into test-only code (for example
  `internal/testutil/ollamaembed`), and point both recorders at it. Nothing in the fixtures or
  floors changes.

**2. Money tools run automatically only within bounds.**

- **Change the default.** The Money class cap becomes Ask first. A money tool may run
  automatically only when its per-call condition shows the call is within bounds. A new money
  tool is therefore safe by default.
- **Bounds that apply to every money tool:**
  - never automatic in a tainted run, or after outside content (as today);
  - a per-call amount ceiling;
  - a per-run cap and a per-day cap, both counted in calls and in total amount;
  - the existing per-tool daily execution budgets (`agentbudgetservice`) count the calls.
- **Bounds for each tool:**

  | Tool | Automatic only when |
  |---|---|
  | `post_customer_payment` | The remittance matches open invoices of the same customer exactly (amount and reference), the whole payment is applied, and it is within the ceiling |
  | `match_bank_receipt` | The receipt matches one posted payment exactly, by amount and reference |
  | `correct_charge_code` | The invoice is not yet posted and the corrected total is within the ceiling of the original |
  | Add carrier or driver settlement adjustment | The settlement is not yet approved and the line is within the ceiling |
  | Remove carrier or driver settlement adjustment | The line being removed was added by an agent and the settlement is not yet approved |

- **Configuration.** The ceilings and caps are organization settings, editable in AI Control,
  with conservative defaults.
- **Explaining holds.** A call outside its bounds is filed as a proposal, with `HeldBy` naming the
  bound it exceeded. That keeps the explanation the policy engine already gives.
- **Tests.** Add the bounds to `agenttoolpolicy` validation, and regenerate the safety doc.

**3. A preview digest is required by class.**

- **When it is required.** Approving a proposal requires the digest of the preview the
  approver was shown when either holds:
  - the proposal's class is money, customer-visible, driver-visible or external recipient;
  - the proposal is tainted.

  Internal and personal writes may still be approved without one, and are recorded as
  unreviewed, as today.
- **Where it applies.** Enforce it in `agentdecisionservice` (`preview.go`), so every path
  inherits it:
  - single and batch GraphQL decisions;
  - plans (`agentplanservice`);
  - the REST handlers (`agentproposalhandler`, `agentplanhandler`);
  - decisions in a conversation;
  - decision follow-ups.
- **Batch approval in the Desk.** The batch approval in the Desk must show the preview of every
  gated item before it can approve them together. `batchPreviewDigests` then has a digest for
  each of them.
- **Integrations.** An integration calling the REST API fetches the preview first and echoes its
  digest.
- **Refusal.** Refuse with an error that says the change must be reviewed first.

**4. Remove `_why`, and replace the "Why this step?" toggle with the model's own reasoning.**

- **Server side.** Remove:
  - `rationale.go` and `withRationale`;
  - `liftRationales`;
  - the stored rationale on steps;
  - the `StepRationale` type in the client;
  - the `StepWhyToggle` in `desk-citations.tsx`.
- **The replacement.** Show the reasoning summary already saved on the assistant message that
  made the call (`conversation.Message.Reasoning`, `ReasoningTrace`), using the existing
  `reasoning-disclosure.tsx` component. Hide it when the model returned none.
- **Cost.** Thinking is billed the same whatever is displayed. The Anthropic adapter already asks
  for summarized thinking, and Responses returns reasoning summaries. The feature therefore costs
  no prompt tokens, and drops about 150 tokens per tool on every call.
- **Unattended runs.** Their reasoning stays in the trajectory for audit.

### 8.3 Built (2026-10-19): the context-assembly rebuild

Phases 1, 2 and 4 were built together, as §8.1 item 3 set out, in three stacked PRs: job profiles
(#821), the stable prefix (#822), then the task frame. `docs/engineering/agent-runtime.md`
describes the design as built ("Job profiles in one runtime", "What a provider can cache",
"Notices are system messages", "The task frame and the domain primer").

**Job profiles (Phase 4).**

- One runtime: `agentruntime.profileFor` gives every turn a profile, in one of four modes:
  conversation, unattended, delegated, or read-only fan-out. The agent definition supplies what
  the profile holds.
- A conversation turn can reach the tools other agents in the organization hold for the person.
  They are found through `find_tools` when the turn's own tools match weakly, and each one runs
  under the tier its holder would give it (`DecideInput.HolderCeiling`).
- `delegate_task` is read-only fan-out:
  - several calls in one step run in parallel (`workflow.Go`);
  - a delegate's writes are refused;
  - the delegate hands back its tool results verbatim, as structured findings
    (`DelegateReport.Findings`).
- The routing bench reports profile selection as a confusion matrix, with recall and precision
  for each profile (`deskbench.ProfileConfusion`).

**Stable prefix and deferred loading (Phase 1).**

- Each turn declares one name-sorted tool array and never changes it.
  - The core and the preselected tools are sent in full.
  - Every other tool the turn holds or reaches is declared deferred.
  - Deferred schemas travel by name through Temporal.
- `find_tools` loads tools by reference: `tool_reference` blocks on Anthropic, and a client
  `tool_search` on OpenAI Responses (gpt-5.4 and later, on OpenAI's platform). Other providers
  are sent the tools loaded so far.
  - `allowed_tools` was rejected because it keeps every schema in context.
- The per-turn catalog listing is replaced by a namespace map built from the permission
  registry's categories.
- The person profile is in the stable part.
- `tool_choice: none` for the final answer was confirmed from Phase 0.

**Task frame and primer (Phase 2).**

- **Unattended runs** are framed from an event → playbook table.
  - Every one of the 31 event kinds has a playbook, and so does the load monitor's schedule.
  - Other scheduled or continuous runs, and runs that pick up a wait, get a generic playbook.
  - Playbooks are written in words, never tool names.
- **Chat turns** are framed by the scope classifier, which now also returns a `goal` and a
  written `need`.
  - It stays advisory, and still fails open.
  - The shared verdict store keeps only the category.
- **Preselection** ranks on the need, followed by the message.
- **The domain primer** (about 560 tokens) sits in the cached shared part.
- **Notices.** Steering, world-change and guard notices, and the final-answer note, travel as
  `RoleSystem`.
  - On Anthropic models that accept them, they are sent as mid-conversation `system` messages.
  - Elsewhere they are developer messages (Responses) or user text.
  - The person's words are never sent inside a system message.
- **Rules removed from the shared prompt** are the ones the runtime enforces or a tool's
  description already states:
  - the `withdraw_proposal` explanation;
  - the code-block and internal-id mechanics;
  - the duplicated search warning.
- The fenced-tag list now names every tag in use.
- `PromptVersion` is v11.

**Measured here, offline.**

- **Prompt size.** Across the 90 prompt snapshots, the mean system prompt went from about 3,770
  tokens on master to about 3,480 (characters ÷ 4). That is after adding the primer and the task
  frame.
  - Delegated prompts fell most: 3,490 to 2,790.
  - Chat prompts went from 4,190 to 4,080.
  - Unattended prompts went from 3,380 to 3,430, because they now carry a playbook.
- **The shared part is now the same bytes on every turn of an agent for a person.** It used to
  name the tools each turn loaded. Two tests check this:
  `TestRun_APromptsToolSectionIsTheSameForEveryQuestion` and
  `TestRun_ADisclosedTurnNeverChangesItsToolArray`.
- The Anthropic wire golden for a deferred turn shows `defer_loading` on every unloaded tool,
  the cache mark on the last tool sent in full, and `tool_reference` blocks in the result that
  loaded tools.
- **Suites.** The unit suites pass: agentruntime, modeladapter, agentdefinition, agentguard,
  assistantservice, agentflow, agentredteam, agentevalgate, agenttoolcatalog contract tests,
  and the bench's own tests.
  - Red-team case 08 now expects a delegate's writes to be refused.

**Not measured yet.** These need the live bench, which needs a seeded database and provider keys,
neither available where this was built:

- the cache-read share on a turn's second and later calls (target above 80%);
- pass^k;
- tool-selection accuracy;
- steps per task;
- time to first token;
- preselection recall@8;
- the share of turns that open with `find_tools`.

The bench reports each of them: "CACHED 2+", "OPENS WITH FIND" and recall in `summary.md`, and
`--compare` gives the per-cell changes. To measure them against the Phase 0 baseline, run on a
seeded development database:

```
cd services/tms
task db-reset
trenova desk baseline          # once on master, if no Phase 0 baseline is kept
trenova desk run --provider claude-haiku-5-5@low,claude-haiku-5-5@medium --repeat 4 \
  --compare .deskbench/baselines/<phase-0 time>
trenova desk route --compare <phase-0 routing run>
```
