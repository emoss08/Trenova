---
path: /admin/agent-control
aliases: [AI settings, AI agents, agent setup, LLM providers, model providers, embedding providers, embeddings, semantic search, search by meaning, retrieval, re-index, indexing budget, pgvector, automation agents, agent proposals, agent memory, sub-agents, agent delegation, agent extensions, extension marketplace, web search, internet search, Exa, AI safety, tool rules, agent autonomy, AI quality, agent evaluation, golden set, eval cases, agent regression, AI audit trail, AI audit log, agent audit, AI compliance, AI decisions log, hash chain, tamper evidence, AI export, traces, tracing, trace id]
related:
  - /admin/document-intelligence
  - /admin/inbound-mailboxes
  - /admin/audit-logs
  - /admin/roles
---

## What it's for
AI control is the one place for everything AI in the organization. A rail down the left side
holds ten sections: **Overview**, **Agents**, **Providers**, **Extensions**, **Memory**,
**Retrieval**, **Safety**, **Quality**, **Activity** and **Audit trail**. Providers say where AI work goes (the
model endpoints Trenova calls and which AI tasks each one handles), agents say what AI may do
(their instructions, tools, autonomy and trigger), extensions add abilities that work only for
agents, such as searching the web, using the organization's own account with the vendor, memory
holds the standing instructions and facts agents read, retrieval shows whether agents find
memories, documents and inbound email by meaning and what indexing them costs, safety shows what
each tool and agent can do without a person, quality says how well each agent does its work, activity shows what agents did:
their runs, the changes they proposed, multi-step plans, replays and exceptions, and the audit
trail keeps a signed record of every run, model call, tool call and decision for compliance to
read, check and export.

**Overview** shows whether AI can work at all (a banner warns when no provider is connected or a
task has no provider), a strip of figures for providers and agents that are on, proposals
awaiting a decision, runs in the last 24 hours, model calls, spend, response time and tokens, the
**Organization-wide** switches, and every agent at a glance. Administrators use this page to set
up providers and agents; reviewers use **Activity** to approve or reject what agents propose.

## Tasks

### Connect an AI provider
Keywords: add LLM, model endpoint, OpenAI, API key, gateway, self-hosted model
1. Open [AI control](/admin/agent-control) and select **Providers** in the rail.
2. Select **New provider**.
3. Optionally pick a **Deployment** under **Start from a preset** to fill in the endpoint and
   output settings for a known deployment.
4. Under **Endpoint**, fill in **Name**, **Protocol**, **Model**, **Base URL** and the API key if
   the provider needs one.
5. Under **Routing**, choose the AI tasks in **Handles these tasks** and set **Priority** (lower
   runs first; providers behind it act as fallbacks). Turn on **Trusted for financial work** only
   for a provider that may take tasks that read sensitive records.
6. Leave **Enabled** on and select **Save**.
7. Back on the provider's card, select **Test** to check the endpoint answers and honours JSON
   schemas.

### Set up an embedding provider
Keywords: embeddings, embedding model, semantic search, search by meaning, vector search, Voyage, Gemini embeddings, OpenAI embeddings, nomic-embed-text, retrieval
1. Open [AI control](/admin/agent-control) and select **Providers** in the rail.
2. Select **New provider** and, under **Start from a preset**, pick one of the embedding presets:
   Voyage AI, Gemini, OpenAI, or Ollama for a model on your own hardware. The preset fills in the
   endpoint, the model, and the embedding task.
3. Under **Routing**, **Handles these tasks** shows **Embedding** ticked. An embedding model serves
   nothing else, so leave the other tasks for a separate provider. Anthropic has no embedding
   endpoint, so the task cannot be ticked on an Anthropic provider.
4. Under **Embedding**, check **Dimensions** matches the vector size the model returns and set
   **Input style** to how the endpoint tells a stored document from a search query.
5. Enter the API key, optionally the **Input price, USD per million tokens**, and select **Save**.
6. Select **Test** on the provider's card. The test asks for one embedding and fails when the
   model returns a different size than **Dimensions**.

### Let agents search the web
Keywords: web search, internet, Exa, look up regulations, ELD rules, hours of service, current information, extension marketplace
1. Open [AI control](/admin/agent-control) and select **Extensions** in the rail.
2. On the web search card, select **Set up**.
3. Paste the organization's Exa API key and select **Save changes**. The key is stored encrypted
   and never shown again; leave the field blank later to keep it.
4. Select **Test connection** to check the key works.
5. Turn the extension on, and under **Available to** choose **Every agent** to give all agents,
   the assistant included, the web search tools, or **Agents you choose** to add them only to
   the agents you pick under **Choose tools** on the **Agents** section. Then select **Save
   changes**.
6. Optionally change the search depth, the results per search, the daily request limit and the
   sites agents never receive results from.

### Set up search by meaning
Keywords: semantic search, search by meaning, retrieval, vector search, embeddings, index documents, index email, re-index, indexing budget, pgvector, keyword only, words only
1. Set up an embedding provider first (see the task above). Until one is routed, agents search by
   keywords only.
2. Open [AI control](/admin/agent-control) and select **Retrieval** in the rail.
3. Read the notice at the top, if there is one. It says why agents are searching by keyword only
   and what fixes it: install pgvector 0.8 or newer and run the command it shows on the server,
   route the Embedding task (**Open Providers** goes there), or change the settings on this page
   (**Go to the settings**). With none shown, search by meaning is working.
4. Read the figures: **Indexed**, **Pending**, **Failed**, **Cost this month** against the
   indexing budget, and **Last run**.
5. In **Settings**, turn **Memories**, **Documents** and **Inbound email** on or off, set the
   **Monthly indexing budget (USD)**, and use **Pause indexing** to stop indexing for a while.
   Then select **Save settings**. A source turned on is indexed within the hour.
6. In **Sources**, each source shows how far it is indexed. To embed a source again after its
   text or the model changed, select **Re-index** on its row; the dialog shows what it could cost
   at most before you confirm with **Re-index**.
7. Select **Show failures** on a source to list the items that could not be indexed, with the
   error for each, in the table at the bottom. **Show every source** lists them all again. Select
   a row to read the whole error.

### Create an agent
Keywords: new agent, build agent, automation, scheduled agent, agent template, data access, agent sees amounts, agent pay access, restricted fields
1. Open [AI control](/admin/agent-control) and select **Agents** in the rail.
2. Select **New agent**.
3. Optionally choose a starter in **Start from** to fill in instructions, tools and a trigger you
   can change freely.
4. Fill in **Name**, **Description** and **System instructions**. Add hard lines the agent must
   not cross under **Never**, pressing Enter after each.
5. Under **Tools**, select **Choose tools**, pick what the agent may look up and change, then
   select **Done**. Set **Data access** to **Restricted** only for an agent that needs amounts and
   pay, such as invoice totals, balances, rates or a driver's net pay; at **Internal** its tools
   leave them out and say so. In chat it never sees more than the person asking, and only
   someone whose own role reaches restricted fields can give an agent **Restricted**.
6. Under **Autonomy**, set the **Ceiling**: **Propose only**, **Act with approval** or **Act
   automatically**. Turn on **Shadow mode** or **Simulation** to try the agent without it
   changing anything.
7. Under **When it runs**, pick **Chat**, **Scheduled**, **Event** or **Continuous** and fill in
   the schedule or events it asks for.
8. Optionally set a **Monthly budget**, **Runs per day** and a **Preferred provider**, leave
   **Enabled** on, and select **Save**.

### Have an agent work each event as it happens
Keywords: event agent, check new shipments, load entry check, duplicate shipment check, service failure agent, insight analyst, EDI quarantine agent, unassigned move agent, billing hold agent
1. Open [AI control](/admin/agent-control) and select **Agents** in the rail.
2. Select **New agent**.
3. In **Start from**, choose the starter for the event: the load entry check reads each new
   shipment for duplicates, a rate that disagrees with the lane and stops out of order; the
   service failure desk works every open failure on a shipment and tells the customer when they
   asked to be told; the insight analyst checks each new insight against its records; the EDI
   desk says why a quarantined EDI file failed and what would fix it, and changes nothing. The
   dispatch coverage agent also takes a move that loses its driver, and the billing exception
   agent a billing item put on hold.
4. Check the tools, the **Ceiling** and **Data access** the starter filled in. The insight
   analyst also starts with a cap on **Runs per day**.
5. Under **When it runs**, **Event** is already chosen with the events the starter listens for.
6. Turn on **Shadow mode** to watch what it would do first, then select **Save**.

### Set up an agent for settlements or receivables
Keywords: settlements agent, driver pay agent, payroll agent, carrier settlement agent, carrier invoice matching agent, receivables agent, collections agent, accounts receivable agent, dispute agent, late charges agent, share invoice agent
1. Open [AI control](/admin/agent-control) and select **Agents** in the rail.
2. Select **New agent**.
3. In **Start from**, choose the settlements clerk to draft driver and carrier settlements, sort
   out pay that is missing or held, match carrier invoices and propose each payment, or the
   receivables assistant to apply payments and credit, handle disputes and late charges, and
   say which overdue invoices to chase first. Both are chat agents that act as the person
   talking to them, and anything that moves money waits for that person to approve it. The
   settlements clerk reads each driver's pay setup but never changes it: a pay rate, a standing
   deduction or escrow terms stay with the people who set up driver pay.
4. Leave **Data access** at **Restricted**, since both work with amounts and pay.
5. Select **Save**. To let the billing assistant and the receivables assistant pass work to each
   other, add each to the other's **Can ask** list as described below.

### Let an agent hand work to another agent
Keywords: sub-agent, delegate, deploy a sub agent, ask another agent, report builder agent, agent can't reach another agent
1. Open [AI control](/admin/agent-control) and select **Agents** in the rail.
2. Select the pencil button on the agent that should be able to ask for help.
3. In **Can ask**, select **Add an agent** and pick each agent it may hand a task to, such as the
   Report Builder for an agent that builds dashboards. An agent can ask up to eight others.
4. Select **Save**. From its next reply the agent can hand those agents a task, and shows their
   work step by step in the conversation.

### Choose who can use an agent
Keywords: agent access, restrict agent, agent permissions, give role an agent, who can see agent, limit agent to roles, agent missing from picker, sensitive tools
1. Open [AI control](/admin/agent-control) and select **Agents** in the rail. Each agent's row
   shows who can use it: **Everyone**, or how many roles it is limited to.
2. Select the pencil button on the agent, or select **New agent** to set it while creating one.
3. In **Who can use it**, choose **Everyone who can use the assistant** or **Specific roles**.
4. For **Specific roles**, pick the roles in **Roles**. **Suggested roles** lists every role with
   what it could make of the agent with the tools chosen on the form, saved or not: "Can use all
   of its tools", the resources it is missing, or "Can't use the assistant". Select **Add** beside
   one to choose it.
5. Select **Save**. The agent and who can use it are saved together, so a new agent limited to
   roles is limited from the moment it exists. From then on only people holding one of the chosen
   roles, or a role that inherits one, see the agent in the Desk and the assistant, and only they
   see and decide what it proposes.

### Turn an agent on or off, run it now, or remove it
Keywords: disable agent, enable agent, start run, delete agent
1. Open [AI control](/admin/agent-control) and select **Agents** in the rail.
2. Use the switch at the end of an agent's row to enable or disable it.
3. To start a run without waiting for its trigger, select the play button on the row. It is shown
   for agents that are not chat agents, and only works while the agent is enabled.
4. To change an agent, select the pencil button and edit it, then **Save**.
5. To delete one, select the trash button and confirm with **Remove agent**. System agents cannot
   be removed.

### Approve or reject what an agent proposed
Keywords: agent decisions, pending proposals, review agent changes, approve plan, preview agent change
1. Open [AI control](/admin/agent-control), select **Activity** in the rail and then
   **Proposals**.
2. Right-click a pending proposal and choose **Approve**, **Approve with changes** or **Reject**.
3. Read **What changes** in the dialog: each record the change would touch and its values before
   and after, worked out from the records as they are now. Anything you may not see reads
   **Hidden by your data access**. **Approve and run** stays off until it has loaded, and for a
   change whose record was edited since it was proposed (**Changed since it was proposed**).
4. Give a **Reason** (required when rejecting) and confirm with **Approve and run** or **Reject**.
   If the change moved while the dialog was open, nothing is recorded and the dialog shows it
   again; read it and confirm.
5. With **Approve with changes**, the record the change is about cannot be edited, and what your
   values would do is shown as you type.
6. For a multi-step plan, select **Plans** instead and use **Approve all** or **Reject all**; the
   dialog shows every step's changes in order.

### Pause every agent at once
Keywords: kill switch, stop AI, shadow mode, earned autonomy
1. Open [AI control](/admin/agent-control) on **Overview**.
2. In **Organization-wide**, turn on **Pause all agents**. Agents keep running and recording what
   they would do, but nothing they propose is offered for a decision or executed.
3. In the same panel, turn **Earned autonomy** on or off and choose the **Promotion threshold**
   (clean approvals in a row before a tool moves up a tier on that agent).

### See what agents can do without a person
Keywords: AI safety, autonomy, what can the AI do on its own, auto execute, approval, tool policy, egress, prompt injection, outside text, sensitive tools, audit agents
1. Open [AI control](/admin/agent-control) and select **Safety** in the rail. Safety has two
   tables under it in the rail, **Tool rules** and **By agent**.
2. Read the figures at the top: **Tools that run without a person** (tools that change records
   and, on at least one agent, can run without anyone approving), **Tools that send outside the
   organization**, and **Open agents with sensitive tools** (agents everyone can use that hold
   tools reaching restricted data or leaving the organization).
3. Select **Tool rules**. Every tool is listed with **Who sees it**, its **Max tier**, what it
   **Needs** of the person using it, whether it **Reads outside content**, and whether it
   **Runs without a person** on at least one agent. Use the search box to find a tool by name, or
   select **Filter** to narrow by **Who sees it**, **Max tier**, **Needs**, **Kind**,
   **Reads outside content** or **Runs without a person**, and **Sort** to order by any of them.
   **Display** shows hidden columns such as **Name** and **Kind** and changes the row density.
4. Select a row to open its rule: the **Rationale**, **How far it may go** and any
   **Record condition**. A tool marked **Depends on the call** goes further for some calls than
   others; its rule says why.
5. Select **By agent**, then **Add an agent** and choose one; add up to ten to compare them in one
   table. Nothing is read until you add an agent. Each agent shows **Who can use it** and its
   **Ceiling**, and the table lists every tool the agents hold with what happens
   **Before outside text** and **After outside text**: **Runs on its own**,
   **Depends on the call**, **Needs approval**, **Proposes only** or **Simulated**. Select
   **Filter** to narrow to one **Agent**, an answer, or a **Held by** reason. Select the cross
   beside an agent's name to take it out of the comparison.
6. The **Held by** chips say which limit stops a tool going further, such as the agent's
   ceiling, where the work goes, or that the run has read outside text. A tool whose tier was
   earned shows **Tier earned**, and one still earning shows how many clean approvals it needs
   for the next tier. Select a row to read both answers beside the tool's rule.

### Check how well an agent is doing
Keywords: AI quality, agent score, regression, satisfaction, thumbs down, golden set, evaluation cases, nightly sweep, eval budget, agent got worse
1. Open [AI control](/admin/agent-control) and select **Quality** in the rail. Quality lists its
   tables under it in the rail: **Agents**, **Suite runs**, **Worst-rated answers**,
   **Golden set** and **Settings**.
2. Read the figures at the top: **Satisfaction** (the share of rated answers that were thumbs
   up), **Ratings**, **Quality score** (how the agents score against their golden sets),
   **Regressions**, and **Eval spend this month** against the monthly budget.
3. In **Agents**, each agent is listed with its satisfaction against the window before, a line of
   its recent suite scores, and how its **Last run** went: **Completed**, **Skipped** (nothing
   about the agent or its cases changed), **Budget stopped** or **Failed**. An agent whose score
   fell shows **Regressed**. Select **Filter** to narrow by **Last run** or **Regressed**, and
   **Sort** to order by **Satisfaction** or **Quality score**.
4. Select an agent's row to open it. **Quality over time** shows its scores. To score it now
   instead of waiting for the nightly sweep, select **Run suite now**. Select **Its suite runs**
   or **Its worst-rated answers** to open those tables narrowed to the agent; **Show every agent**
   widens them again.
5. In **Suite runs**, each run shows its **Status**, **Quality score**, **Cases** and
   **What changed** about the agent since the run before, such as its instructions, tools or
   model. Select a run to read it, then **See the cases** to list what each case scored; select a
   case to read the reply and the judge's note. **Back to suite runs** returns to the runs.
6. **Worst-rated answers** lists the answers people rated down in the last 30 days, most disliked
   first. Select one to read the question, the answer and **Why**. Select **Open the
   conversation** to read one you were part of.
7. Keep the cases the agents are scored against in **Golden set**: **Activate** a candidate
   captured from a decided proposal, **Add case** to write one by hand, or **Quarantine** a case
   that is no longer fair.
8. In **Settings**, turn **Run the nightly sweep** on or off, choose the **Hour it starts**, set
   **Most cases per agent**, the **Nightly budget (USD)** and **Monthly budget (USD)**, the
   **Regression threshold (points)**, and whether to **Have a judge read a sample**. Then select
   **Save settings**.

### Read what agents did on the audit trail
Keywords: AI audit trail, agent audit log, who approved, what did the agent do, AI compliance, tool calls, model calls, AI decisions, evaluations
1. Open [AI control](/admin/agent-control) and select **Audit trail** in the rail, then **Trail**.
2. Read the figures at the top: **Chain** says whether the trail is **Signed** with a key held
   outside the database or **Unsigned**, **Sealed through** is the last row the trail has sealed,
   and **Last verified** is what the last check found.
3. Choose the range at the left of the bar above the table: **Last 24 hours**, **Last 7 days**
   (where it starts), **Last 30 days**, **Last 90 days**, or pick days on the calendar and select
   **Apply**. Select **Every agent** to narrow to one agent, pick a person in **Anyone** to see the
   work done for them or decided by them, and turn on **Include evaluations** to show replays
   beside live work.
4. The table lists each event with **When**, **Agent**, **What** (the event and the tool),
   **Outcome**, **Person**, **Record**, **Tier**, **Model**, **Cost**, **Tainted** and **Trace**.
   Select **Filter** to narrow by **What**, **Tool**, **Outcome**, **Record**, **Record type**,
   **Tier**, **Model**, **Tainted** or **Trace**.
5. Select a row to read the event in full: **Who**, **What**, **Why** (the tier, what set it and
   what held it back), **Changed** (the record, its versions, and the audit log entries
   **Matched by time**), **Provenance**, **Model**, **Arguments**, **Trace** and **Chain**.

### Check that the audit trail has not been changed
Keywords: verify audit trail, hash chain, tamper evidence, audit integrity, signing key
1. Open [AI control](/admin/agent-control) and select **Audit trail** in the rail.
2. Select **Verify now**. The check runs in the background and the button shows **Verifying…**
   until its result is stored.
3. Read **Last verified**: **Verified** means every row still matches its chain, **Mismatch**
   means a row was changed or removed and says at which row, and **Key missing** means a row names
   a signing key that is no longer configured.

### Export the AI audit trail
Keywords: download audit trail, AI audit export, CSV, JSON, auditor, compliance export, SHA-256
1. Open [AI control](/admin/agent-control), select **Audit trail** in the rail and narrow the
   trail if you want to export part of it.
2. Select **Export trail…**.
3. Choose the **Format**, CSV or JSON, set **From** and **To**, and leave **Use current filters** on to
   carry the agent, person, evaluations and table filters into the file, or turn it off to export
   every row in the range.
4. Select **Export**. A small export downloads at once and shows its rows, size and SHA-256;
   **Download again** fetches it again. A large one is written in the background: you are
   notified when it is ready, and **Open exports** shows it.
5. Select **Exports** in the rail to see every export with its status, **Rows**, **Size**,
   **SHA-256**, **Chain** (**Complete** or **Filtered**) and when it **Expires**. Select
   **Download** on your own export to download it.

### Find the trace of an agent's work
Keywords: trace id, tracing, OpenTelemetry, Tempo, Jaeger, span
1. Open [AI control](/admin/agent-control) and select **Activity** in the rail, then **Runs** or
   **Proposals**; or open **Audit trail**.
2. The **Trace** column shows the start of the trace id. Select the copy button beside it to copy
   the whole id; where a tracing backend is configured, the id is a link that opens the trace.
3. To find every row of one trace, select **Filter**, choose **Trace** and paste the id.

### Record something every agent should know
Keywords: agent memory, standing instruction, fact, correction, retire memory
1. Open [AI control](/admin/agent-control) and select **Memory** in the rail.
2. Select **New memory**.
3. Choose the **Kind** (**Instruction**, **Fact** or **Correction**), write the **Memory**, and
   optionally set **Until**.
4. To make it about one record, pick a **Kind of record** and the record; leave it empty for
   something every agent should know. Then select **Save**.
5. To stop agents reading an entry, right-click it and choose **Retire**; **Restore** brings it
   back.

## Notes
An embedding provider turns memories, documents and mail into vectors so agents can find them by
what they mean rather than by the exact words; until one is set up, agents search by keywords
only. Social security, card and bank account numbers are masked before a document is sent.
Changing the embedding model or its size re-indexes everything that was embedded. Providers with
the same model and size back each other up; a provider with a different model is never used in
their place.

Retrieval is read with read access to AI providers; changing its settings or re-indexing needs
update access to AI providers. Only chunks whose text changed are embedded again on a re-index,
so the cost shown before one is the most it could cost, and a re-index when nothing changed costs
nothing. The monthly indexing budget covers indexing for the calendar month (UTC); when it is
spent indexing pauses, and it resumes on its own the next month or as soon as the budget is
raised. What agents' searches cost counts against each agent's own monthly budget. When the
Embedding task is routed to a different model, every source is indexed under the new model
beside the old one, and searches move to it only when all of it is done; **Sources** shows how far
that has come.

An agent that has read content written outside the organization (an inbound email, an
extracted document, an EDI file, a bank receipt, a file attached in chat, or a memory such a
run wrote) never sends anything to a customer, driver or outside address, or moves money, on
its own: that change waits for a person's approval whatever tier the tool has, and the
proposal is marked as having read outside content. A suggested memory drawn from ratings of
one agent is kept for that agent alone once approved.

A memory holds up to 4,000 characters. Each prompt carries only as much memory as the agent's
**Memory in the prompt** setting allows (6,000 tokens unless changed), starting with what is
recorded about the record the conversation is about. When the organization keeps close to 5,000
active memories, the Memory section warns that fewer of them reach each prompt; retire what no
longer holds.

Opening the page needs read access to AI control. Each section in the rail appears only for
people who may read it (agents, AI providers, agent runs, agent proposals, agent exceptions,
agent memory); a section someone cannot open is left out. **Safety** appears for people who may
read agents, and **Quality** for people who may read the evaluation suite. **Worst-rated answers**
and satisfaction need read access to agent feedback, **Run suite now** needs create access to the
evaluation suite, and changing **Settings** needs update access to both the evaluation suite and
AI control, because the budgets are spend. The organization-wide switches need
update access to AI control, deciding proposals needs update access to agent proposals, and
**Test** on a provider needs manage access to AI providers. Viewing **Extensions** needs read
access to agent extensions, and turning one on, changing its settings or testing it needs update
access. An agent searches the web for someone only when their role has the web research
permission.

Search queries and the addresses of pages agents read are sent to the extension's vendor. Queries
that contain Trenova record IDs, email addresses or phone numbers are refused before they leave
Trenova. Once an agent has read web content, every change it asks for in the rest of that reply
waits for a person's approval, whatever the agent's autonomy. Each extension counts its requests
against the daily limit, which resets at midnight UTC, and the card shows today's requests and
this month's cost.

An agent asks only the agents listed under **Can ask**; with none listed it works with its own
tools alone. Only agents people talk to can ask or be asked, and an agent that was asked cannot
hand the task on. The agent asked works as the person in the conversation, with its own tools and
approvals, so it can never do more than that person could.

Who can use an agent is set under **Who can use it**. An agent open to everyone that holds tools
reaching restricted or confidential data, or whose work leaves the organization, shows a warning
there as soon as the form says so, before it is saved; each person can still only do what their
own permissions allow. An agent limited to specific roles with none chosen can be used
by nobody. Roles chosen while an agent is open to everyone are kept for when it is limited again.
A system agent is always open to everyone and cannot be limited to roles. Someone who loses
access to an agent keeps their conversations with it, read-only. Setting who can use an agent
needs update access to both agents and roles; someone without update access to roles can still
save the rest of an agent as long as they leave who can use it as it was. The same grants can be managed from a role's page
on [Roles](/admin/roles), under **Agents**.

Every night, at the hour chosen in **Settings** (in the organization's timezone unless another
is chosen there), each agent with active cases is replayed against a sample of its golden set
with every write simulated. The sample always includes the cases the agent failed most recently
and is otherwise spread across where the cases came from; it is the same sample for the same run.
An agent is skipped when nothing about it (its instructions, tools, model or provider) or its
cases changed since its last run, unless that run is older than the rerun limit. A run stops
when the nightly or monthly budget is spent, and says so. When an agent's score falls below the
median of its recent runs by more than the threshold, the run is marked **Regressed**, a
Watchtower item is raised (critical when a case failed a hard check), and the people who can
update AI control are told once, with what changed.

The **Audit trail** section appears for people with read access to the AI audit trail; reading
agent runs does not grant it. **Verify now** needs the same read access. **Export trail…** and
**Download** need export access to the AI audit trail, and only the person who asked for an export
can download it; the file can be downloaded until it expires, seven days after it was written
unless the organization's configuration says otherwise. Requesting and downloading an export are
written to the audit log. The audit log entries shown with an event appear only for people who may
read the audit log. Arguments show only what the reader may see on that record; anything above it
reads `[withheld]`, and confidential values were never recorded. Events reach the trail within a
minute of happening. How long they are kept is set on
[Data retention](/organization/data-retention).

Removing a provider stops any task routed only to it until another provider is assigned.
Removing an agent keeps its existing conversations but they cannot be continued, and its schedule
stops. On **Runs**, right-clicking a finished run offers **Replay against the current agent**; the
comparison appears under **Evaluations**, where **Open comparison** shows what the replay would
have done beside the original.
