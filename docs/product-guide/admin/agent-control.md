---
path: /admin/agent-control
aliases: [AI settings, AI agents, agent setup, LLM providers, model providers, automation agents, agent proposals, agent memory, sub-agents, agent delegation]
related:
  - /admin/document-intelligence
  - /admin/inbound-mailboxes
  - /admin/audit-logs
  - /admin/roles
---

## What it's for
AI control is the one place for everything AI in the organization. A rail down the left side
holds five sections: **Overview**, **Agents**, **Providers**, **Memory** and **Activity**.
Providers say where AI work goes (the model endpoints Trenova calls and which AI tasks each one
handles), agents say what AI may do (their instructions, tools, autonomy and trigger), memory
holds the standing instructions and facts agents read, and activity shows what agents did: their
runs, the changes they proposed, multi-step plans, replays and exceptions.

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
4. For **Specific roles**, pick the roles in **Roles**. On a saved agent, **Suggested roles**
   lists every role with what it could make of the agent: "Can use all of its tools", the
   resources it is missing, or "Can't use the assistant". Select **Add** beside one to choose it.
5. Select **Save**. From then on only people holding one of the chosen roles, or a role that
   inherits one, see the agent in the Desk and the assistant, and only they see and decide what
   it proposes.

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
Opening the page needs read access to AI control. Each section in the rail appears only for
people who may read it (agents, AI providers, agent runs, agent proposals, agent exceptions,
agent memory); a section someone cannot open is left out. The organization-wide switches need
update access to AI control, deciding proposals needs update access to agent proposals, and
**Test** on a provider needs manage access to AI providers.

An agent asks only the agents listed under **Can ask**; with none listed it works with its own
tools alone. Only agents people talk to can ask or be asked, and an agent that was asked cannot
hand the task on. The agent asked works as the person in the conversation, with its own tools and
approvals, so it can never do more than that person could.

Who can use an agent is set under **Who can use it**. An agent open to everyone that holds tools
reaching restricted or confidential data shows a warning there; each person can still only do
what their own permissions allow. An agent limited to specific roles with none chosen can be used
by nobody. Roles chosen while an agent is open to everyone are kept for when it is limited again.
A system agent is always open to everyone and cannot be limited to roles. Someone who loses
access to an agent keeps their conversations with it, read-only. Setting who can use an agent
needs update access to both agents and roles. The same grants can be managed from a role's page
on [Roles](/admin/roles), under **Agents**.

Removing a provider stops any task routed only to it until another provider is assigned.
Removing an agent keeps its existing conversations but they cannot be continued, and its schedule
stops. On **Runs**, right-clicking a finished run offers **Replay against the current agent**; the
comparison appears under **Evaluations**, where **Open comparison** shows what the replay would
have done beside the original.
