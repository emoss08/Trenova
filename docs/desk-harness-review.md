# Desk harness review

Status: **Phase 1 (audit) complete. Phase 2 (research) in progress.** Phase 3 (analysis) has
not started.

This is an architecture review of Desk, Trenova's agent harness. It is not a change
proposal for any one file. Numbers come from the source at the commit this was
written on; where a number is an estimate, the method is given.

## Phase 1 summary

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
5. **Habit fixes exist only for the two benched models.** Every one targets Haiku 5.5 or
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
