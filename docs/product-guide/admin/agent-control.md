---
path: /admin/agent-control
aliases: [AI settings, AI agents, agent setup, LLM providers, model providers, embedding providers, embeddings, semantic search, automation agents, agent proposals, agent memory, sub-agents, agent delegation, agent extensions, extension marketplace, web search, internet search, Exa, AI safety, tool rules, agent autonomy, AI quality, agent evaluation, golden set, eval cases, agent regression]
related:
  - /admin/document-intelligence
  - /admin/inbound-mailboxes
  - /admin/audit-logs
  - /admin/roles
---

## What it's for
AI control is the one place for everything AI in the organization. A rail down the left side
holds eight sections: **Overview**, **Agents**, **Providers**, **Extensions**, **Memory**,
**Safety**, **Quality** and **Activity**. Providers say where AI work goes (the model endpoints
Trenova calls and which AI tasks each one handles), agents say what AI may do (their
instructions, tools, autonomy and trigger), extensions add abilities that work only for agents,
such as searching the web, using the organization's own account with the vendor, memory holds the
standing instructions and facts agents read, safety shows what each tool and agent can do without
a person, quality says how well each agent does its work, and activity shows what agents did:
their runs, the changes they proposed, multi-step plans, replays and exceptions.

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

### Create an agent
Keywords: new agent, build agent, automation, scheduled agent, agent template
1. Open [AI control](/admin/agent-control) and select **Agents** in the rail.
2. Select **New agent**.
3. Optionally choose a starter in **Start from** to fill in instructions, tools and a trigger you
   can change freely.
4. Fill in **Name**, **Description** and **System instructions**. Add hard lines the agent must
   not cross under **Never**, pressing Enter after each.
5. Under **Tools**, select **Choose tools**, pick what the agent may look up and change, then
   select **Done**.
6. Under **Autonomy**, set the **Ceiling**: **Propose only**, **Act with approval** or **Act
   automatically**. Turn on **Shadow mode** or **Simulation** to try the agent without it
   changing anything.
7. Under **When it runs**, pick **Chat**, **Scheduled**, **Event** or **Continuous** and fill in
   the schedule or events it asks for.
8. Optionally set a **Monthly budget**, **Runs per day** and a **Preferred provider**, leave
   **Enabled** on, and select **Save**.

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
Keywords: agent decisions, pending proposals, review agent changes, approve plan
1. Open [AI control](/admin/agent-control), select **Activity** in the rail and then
   **Proposals**.
2. Right-click a pending proposal and choose **Approve**, **Approve with changes** or **Reject**.
3. Give a **Reason** (required when rejecting) and confirm with **Approve and run** or **Reject**.
4. For a multi-step plan, select **Plans** instead and use **Approve all** or **Reject all**.

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

Removing a provider stops any task routed only to it until another provider is assigned.
Removing an agent keeps its existing conversations but they cannot be continued, and its schedule
stops. On **Runs**, right-clicking a finished run offers **Replay against the current agent**; the
comparison appears under **Evaluations**, where **Open comparison** shows what the replay would
have done beside the original.
