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

(Sections 1.2 to 1.5 and 1.7 are being filled in from parallel audits.)

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
