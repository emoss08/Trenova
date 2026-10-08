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

**Overview** opens with one sentence from Nova saying what is true now: how many agents are on and
working, how many proposals wait on a person, a provider that can't connect and any task with no
provider, each linked to where to act on it. Below it are the week's figures (model calls with the
days they failed, median response, tokens and spend), a notice for a provider that is failing,
**Tune-ups** drawn from the last 30 days of runs, **Usage by feature**, the agents at work and every agent at a glance, and the **Organization-wide**
settings. With no provider connected, it walks through connecting one. Administrators use this page to set
up providers and agents; reviewers use **Activity** to approve or reject what agents propose.

## Tasks

### Connect an AI provider
Keywords: add LLM, model endpoint, OpenAI, API key, gateway, self-hosted model, Ollama, vLLM
1. Open [AI control](/admin/agent-control) and select **Providers**.
2. Select **New provider** and pick the vendor, under **Hosted** or **On your network**. With no
   provider yet, pick it from **Connect a model to wake your agents**.
3. Under **Connection**, check **Name**, **Kind** and **Base URL**. A base URL on your own network
   needs **Private network**; select **Turn on Private network** when the editor asks.
4. Under **Model**, select **Fetch models** to pick from what the endpoint actually serves, or
   type the **Model ID**.
5. Paste the **Key** under **API key** when the provider needs one.
6. Under **What it handles**, choose the tasks. **When you save** lists every task whose provider
   changes.
7. Select **Test draft**. When it connects, the save button reads **Add and turn on**; otherwise
   **Add provider** saves it off until it passes.

### Order providers and route tasks
Keywords: provider priority, fallback, routing, which provider, reorder providers, assign task
1. Open [AI control](/admin/agent-control) and select **Providers**.
2. **The chain** lists providers top to bottom; each task goes to the first one that is on,
   assigned and, for tasks marked with a shield, **Trusted**. Drag a row to reorder, or open it and
   select **Move up** or **Move down**.
3. Under **Routing**, select a cell to assign or unassign a task. **Goes to** shows where each task
   lands, and a task nothing takes says what happens instead.
4. Providers that are off keep their place in line; select **Show** to list them.

### Limit what a provider may spend or how long it waits
Keywords: timeout, concurrency, monthly cap, spend limit, budget per provider, price per million tokens
1. Open the provider in **Providers** and select **Edit connection**.
2. Under **Limits and price**, set **Timeout** and **Concurrent calls**. A call past the timeout,
   or past the concurrent limit, moves to the next provider.
3. Set **Monthly spend cap** and choose **At the cap**: **Hand to next** passes work on, **Stop**
   fails it. Set **Input price** and **Output price** so spend can be counted.
4. Select **Save changes**.

### Replace a provider's API key
Keywords: rotate key, new API key, key rotation, expired key, leaked key
1. Open the provider in **Providers** and select **Edit connection**.
2. Under **API key**, paste the new key into **Replace key**.
3. Leave **Keep the old key working for 24 hours** on so calls fall back to the old key while
   other systems switch over, then select **Save changes**.

### Set up an embedding provider
Keywords: embeddings, embedding model, semantic search, search by meaning, vector search, Voyage, Gemini embeddings, OpenAI embeddings, nomic-embed-text, retrieval
1. Open [AI control](/admin/agent-control) and select **Providers**.
2. Select **New provider** and pick one of the embedding vendors: Voyage AI, Gemini, OpenAI, or
   Ollama for a model on your own hardware. It fills in the endpoint, the model and the
   **Embedding** task.
3. An embedding model serves nothing else, so leave the other tasks under **What it handles** for a
   separate provider. Anthropic has no embedding endpoint.
4. Under **Advanced**, check **Vector size** matches what the model returns and set **Embedding
   input** to how the endpoint tells a stored document from a search query.
5. Paste the key, select **Test draft**, then **Add and turn on**. The test asks for one embedding
   and fails when the model returns a different size.

### Let agents search the web
Keywords: web search, internet, Exa, look up regulations, ELD rules, hours of service, current information, extension marketplace
1. Open [AI control](/admin/agent-control) and select **Extensions**.
2. Open **Web search**, under **Featured** or from **All extensions**.
3. Paste the organization's Exa API key under **Connection** and select **Turn on**. The key is
   stored encrypted and never shown again; select **Replace** to change it later.
4. Select **Test connection** to check the key works.
5. Under **Available to**, choose **Every agent** to give all agents, the assistant included, the
   web search tools, or **Agents you choose** and select the agents that get them.
6. Optionally change **Search depth**, **Daily request limit**, the results per search and the
   sites agents never receive results from. Each change is saved as you make it.

### Set up search by meaning
Keywords: semantic search, search by meaning, retrieval, vector search, embeddings, index documents, index email, re-index, indexing budget, pgvector, keyword only, words only
1. Set up an embedding provider first (see the task above). Until one is routed, agents search by
   keywords only.
2. Open [AI control](/admin/agent-control) and select **Retrieval** in the rail.
3. Read Nova's sentence at the top. It says whether agents search by meaning or by keyword
   only, how much is left to index, and what failed. Beside it, **Route Embedding** goes to
   Providers when nothing handles the Embedding task, and **Pause indexing** or
   **Resume indexing** stops or restarts the indexer. A notice under it says when the database
   needs pgvector 0.8 or newer, with the command to run on the server, or when the embedding
   provider is failing (**Open Providers** goes there).
4. Read the figures: **Indexed**, **Waiting**, **Failed**, **Cost this month** against the
   indexing budget, and **Last run**.
5. In **Sources**, turn **Memories**, **Documents** and **Inbound email** on or off with the
   switch on each row; a source turned on is indexed within the hour. To embed a source again
   after its text or the model changed, select **Re-index** on its row; the dialog shows what it
   could cost at most before you confirm with **Re-index**.
6. In **Settings**, type the **Monthly indexing budget**; it is saved when you leave the field.
   **Pause indexing** stops indexing for a while.
7. Select the failed count on a source to list the items that could not be indexed, with the
   error for each, in the table at the bottom. **Show every source** lists them all again. Select
   a row to read the whole error.

### Create an agent
Keywords: new agent, build agent, automation, scheduled agent, agent template, data access, agent sees amounts, agent pay access, restricted fields, draft agent with AI
1. Open [AI control](/admin/agent-control) and select **Agents**.
2. Select **New agent** (or press N) and pick where to start: **Desk agent**, **Scheduled report**,
   **Event watcher** or **Start blank**. The builder opens on **What should it do?**.
3. Describe the job in a sentence or two and select **Draft it** to have Nova fill in the name,
   instructions, trigger and tools, or pick one of the starts under it to set it up yourself. Drafting is offered only while a provider takes the assistant's work.
4. Under **Identity**, name the agent and say what it does in one line. Select its tile to change
   the icon and color.
5. Under **Instructions**, brief it the way you would a new hire. Add hard lines it must not cross
   under **Never**, pressing Enter after each. A line under the editor says when the instructions
   ask for something none of its tools can do, with a button to give it that tool.
6. Under **Tools and autonomy**, select **Add tools**, tick what the agent may look up and change,
   then select **Done**. Set each change to **Propose**, **Ask first** or **Automatic**, and the
   **Ceiling** no tool goes past. Set **Data access** to **Restricted** only for an agent that
   needs amounts and pay, such as invoice totals, balances, rates or a driver's net pay; at
   **Internal** its tools leave them out and say so. In chat it never sees more than the person
   asking, and only someone whose own role reaches restricted fields can give an agent
   **Restricted**.
7. Under **When it runs**, pick **Someone asks**, **On a schedule**, **Something happens** or
   **Keeps watch** and fill in the schedule or events it asks for.
8. Optionally set a **Monthly budget**, **Runs per day** and a **Preferred provider** under
   **Budget and model**.
9. Select **Try it** to ask the unsaved draft something against live records. Nothing is written,
   sent or offered to anyone, and each step says what it would have come to.
10. A new agent starts in **Shadow**. Select **Create in shadow**, or switch the mode beside its
    name to **Live** first.

### Let an agent hand work to another agent
Keywords: sub-agent, delegate, deploy a sub agent, ask another agent, report builder agent, agent can't reach another agent
1. Open [AI control](/admin/agent-control) and select **Agents**.
2. Select the agent that should be able to ask for help, then **Edit**.
3. Under **Memory and handoffs**, in **Can ask**, select each agent it may hand a task to, such
   as the report analyst for an agent that builds dashboards. Only agents people talk to can ask
   others, and an agent can ask up to eight.
4. Select **Save**. From its next reply the agent can hand those agents a task, and shows their
   work step by step in the conversation.

### Choose who can use an agent
Keywords: agent access, restrict agent, agent permissions, give role an agent, who can see agent, limit agent to roles, agent missing from picker, sensitive tools
1. Open [AI control](/admin/agent-control) and select **Agents**. An agent limited to roles names
   them at the start of its row.
2. Select the agent, then **Edit**, or select **New agent** to set it while creating one.
3. Under **When it runs**, in **Who can ask it**, choose **Everyone** or **Specific roles**, and
   pick the roles. While it is open to everyone and holds tools that reach outside the company
   or into restricted records, a warning names them with **Limit to roles**.
4. Select **Save**. The agent and who can use it are saved together, so a new agent limited to
   roles is limited from the moment it exists. From then on only people holding one of the chosen
   roles, or a role that inherits one, see the agent in the Desk and the assistant, and only they
   see and decide what it proposes.

### Turn an agent on or off, run it now, or remove it
Keywords: disable agent, enable agent, start run, delete agent, shadow mode, simulation mode
1. Open [AI control](/admin/agent-control) and select **Agents**.
2. Use the switch at the end of an agent's row to turn it on or off.
3. To start a run without waiting for its trigger, select the play button on the row, or **Run
   now** on the agent. It is offered for agents that are not chat agents, while they are on.
4. Select an agent to read it. Its **Mode** is **Live**, **Shadow** or **Simulation**; **Edit**
   opens the builder.
5. To delete one, select **Remove**, type its name and confirm with **Remove agent**. Its runs and
   audit trail are kept and its open proposals are withdrawn. System agents cannot be removed.

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
2. Press and hold **Hold to pause all agents** beside Nova's sentence until it fills. Agents keep
   running and recording what they would do, but nothing they propose is offered for a decision or
   executed. Select **Resume agents** to let them surface their work again.
3. In **Organization-wide**, select **Edit** to turn **Earned autonomy** on or off and choose the
   **Clean approvals** in a row before a tool moves up a tier on that agent. Before you save, it
   lists the tools that have already earned it; they move up when you save.

### Apply a suggested tune-up
Keywords: tune-ups, suggestions, recommendations, improve AI setup, reorder providers, idle agent, leave shadow, assign task
1. Open [AI control](/admin/agent-control) on **Overview**. **Tune-ups** lists changes to how AI is
   set up that the last 30 days of runs argue for, worked out each night: let a tool people keep
   approving unchanged move up a tier (only while earned autonomy is off), put a provider ahead of
   one that keeps failing when it catches the failures, take an agent out of shadow once its
   recorded proposals match what people did, give a task nothing handles to a provider that can
   take it, and turn off a chat agent nobody has asked anything in two weeks.
2. Each row says what the runs showed and what the change gains. Select the action on the row
   (**Raise it**, **Reorder**, **Go live**, **Assign** or **Turn off**) to make the change.
3. Select the close button on the row to dismiss it (its label is **Dismiss for 30 days**); the same change is not suggested again for 30 days. Select **Undo**
   in the notice to bring it back. Operational work, such as a proposal waiting on a person, stays
   in Watchtower.

### See what agents can do without a person
Keywords: AI safety, autonomy, what can the AI do on its own, auto execute, approval, tool policy, egress, prompt injection, outside text, sensitive tools, audit agents
1. Open [AI control](/admin/agent-control) and select **Safety** in the rail. The sentence at the
   top says how many tools run without a person, how many send outside the organization (every
   one waits for approval), and how many agents everyone can use hold them. Select the count of
   tools that run to narrow the rules to them. When open agents hold sensitive tools, select
   **Review open agents** to compare the first three.
2. Read the figures: **Tools that run without a person**, **Tools that send outside the
   organization** and **Open agents with sensitive tools**. **Who sees the work** lays out every
   tool that changes something by the widest audience its work reaches; select an audience to
   narrow the rules to it, and **Clear** to show them all again.
3. Under **View**, select **Tool rules**. Every tool is listed with **Who sees it**, its
   **Max tier**, what it **Needs** of the person using it, whether it **Reads outside content**,
   the **Agents** that hold it, and whether it **Runs without a person** on at least one agent.
   Use the search box to find a tool by name, or **Filter** and **Sort** by any of them.
4. Select a row to open the tool: **Most it may do**, **Needs**, **Reads outside content** and
   **Who sees its work**, then every agent holding it with what that agent does before and after
   reading outside text, and **The whole rule**. **Audit trail** opens the audit trail.
5. To hold a tool lower for your organization, select **Change tool rule**. Choose the
   **Most freedom any agent gets** (choices looser than the tool's own rule are closed) and
   whether to **Treat what it returns as outside text**, and write a **Reason**; a change to the
   most freedom cannot be saved without one. **Who's affected** lists every agent holding the tool
   and how its answer moves before you save. Select **Save changes**: the change, its reason and
   the agents it moved go to the audit trail, and the open tool shows
   **Your organization's rule**. Choosing what the tool declares returns it to its own rule.
6. Under **View**, select **By agent**, then **Pick an agent**; select **Compare another** to
   compare up to four. Nothing is read until you pick one. Each agent shows who can use it and its
   ceiling, warns when everyone can use it, and offers **Limit to roles**. The table lists every
   tool the agents hold with what happens **Before outside text** and **After outside text**:
   **Runs on its own**, **Depends on the call**, **Needs approval**, **Proposes only** or
   **Simulated**. The **Held by** chips say which limit stops a tool going further; a tool whose
   tier was earned shows **Tier earned**.

### Check how well an agent is doing
Keywords: AI quality, agent score, regression, satisfaction, thumbs down, golden set, evaluation cases, nightly sweep, eval budget, agent got worse
1. Open [AI control](/admin/agent-control) and select **Quality** in the rail. The sentence at the
   top says how the agents score against their golden sets, how many of the answers people rated
   they liked, and which agent fell furthest after its last change. Select its name, or the
   button beside the sentence, to open that agent.
2. Read the figures: **Satisfaction** (the share of rated answers that were thumbs up),
   **Ratings**, **Quality score**, **Regressions** still open, and **Eval spend this month**
   against the monthly budget.
3. Under **View**, choose **Agents**, **Suite runs**, **Worst rated**, **Golden set** or
   **Document extraction**.
4. In **Agents**, each agent is listed with its **Quality score** and a line of its recent runs,
   the **Change** against its recent median, and how its **Last run** went: **Completed**,
   **Skipped** (nothing about the agent or its cases changed), **Budget stopped** or **Failed**.
   **Display** shows **Satisfaction** and **Ratings**. Select **Filter** to narrow by
   **Last run** or **Regressed**, and **Sort** to order by **Quality score**.
5. Select an agent to open it: its score and the change, **Satisfaction**, the
   **Regression threshold**, the **Judge**, **Cases per night**, **What changed** and the answers
   rated **Lowest rated**. Select **Run suite now** to score it without waiting for the night,
   **Open agent** to edit it, or **All runs** to list its suite runs; **Show every agent** widens
   them again.
6. In **Suite runs**, each run shows its **Status**, **Cases asked** and **Quality score**. Select
   a run to read its **Score**, **Change**, **Cost** and **What changed**, with a square for every
   case: red ones scored below the bar. **See the cases** lists what each case scored; select a
   case to read the reply and the judge's note. **Back to suite runs** returns to the runs.
7. **Worst rated** lists the answers people rated down in the last 30 days, most disliked first.
   Select one to read **The question**, **The answer** and **What they said**. Select
   **Add to golden set** to score the agent against it every night, **Write a memory** to tell
   the agents what was wrong, or **Open the conversation** to read one you were part of.
8. Keep the cases the agents are scored against in **Golden set**: **Activate** a candidate
   captured from a decided proposal, **Add case** to write one by hand, or **Quarantine** a case
   that is no longer fair.
9. In **Agents**, select **Sweep settings** to change the **Nightly sweep**: when it should
   **Start at**, the **Cases per agent**, and when to **Rerun an unchanged agent after**; turn the
   **Judge model** on and choose its **Share of answers**; set the **Regression threshold** and
   the **Fewest cases to compare**; and the **Budget** **Per night** and **Per month**. Select
   **Save changes**. Changing these needs permission to update AI control and the golden set.

### Try a new document extraction model on real documents
Keywords: shadow traffic, shadow model, candidate model, fine-tuned model, compare extraction models, A/B test extraction, new extraction provider, model rollout
1. Register the new model as an AI provider assigned to **Document extraction**, with a
   priority after the provider that extracts documents today, so nothing is routed to it.
2. Open [AI control](/admin/agent-control), select **Quality** in the rail, then **Document
   extraction**, and choose **Shadow**.
3. Select **Edit settings**, turn on **Shadow production extraction**, choose the **Candidate
   provider**, and set the **Share of extractions (%)** and the **Most per 24 hours**. Select
   **Save**. From then on that share of documents is also read by the candidate; its answer is
   kept but never used, and the calls spend from the evaluation budget.
4. When someone creates a shipment from a shadowed document's draft, both answers are scored
   against what they confirmed. Read the figures: **Candidate accuracy** against **Production
   accuracy** on the same documents, how often the candidate did **Better or worse**, how many
   documents were **Shadowed**, and the **Cost**. **Accuracy by field** puts the candidate's
   biggest shortfall first.
5. Select a row in the table to read one document field by field: what a person confirmed,
   what the candidate read, and what production read.
6. To stop, select **Edit settings**, turn **Shadow production extraction** off and select
   **Save**. Move the candidate ahead of the current provider only when it beats production
   here and in an evaluation run.

### Roll out a new document extraction model gradually
Keywords: gradual rollout, canary, promote extraction model, serve new model, fine-tuned model in production, stop rollout, rollout guard, rollback extraction model
1. Try the model first under **Shadow** (see above), and roll it out only once it reads at least
   as well as production there.
2. Open [AI control](/admin/agent-control), select **Quality** in the rail, then **Document
   extraction**, and choose **Rollout**. Changing the rollout needs permission to update AI
   providers.
3. Select **Edit settings**, turn on **Serve the candidate**, choose the **Candidate provider**,
   and set the **Share of documents (%)**. Start small, such as 5 percent. The two guard
   allowances say how far the candidate may fall behind production before the rollout stops on
   its own: **Stop below production's accuracy by (pts)** and **Stop above production's unusable
   answers by (pts)**. Select **Save**. From then on the candidate reads that share of documents
   and its answers fill their shipment drafts; when it cannot answer, production reads the
   document instead.
4. Read the figures: **Candidate accuracy** on the documents it read against **Production
   accuracy** on the rest, how often its answers were **Unusable answers** beside production's,
   and how many documents were **Sent to the candidate** and **Kept on production**. **Guards**
   says how much evidence each guard still needs before it can act. **Accuracy by field** puts
   the candidate's biggest shortfall first.
5. Raise the share in steps while the candidate holds up. The same documents stay with the
   candidate, so each step adds documents.
6. To stop at any time, select **Stop rollout**; every document goes back to production at once.
   If a guard stops the rollout, the page says which one and why, and the people who can change
   AI providers are notified. Save the settings with the rollout on to start a new
   comparison.
7. Once the candidate has served a large share without a guard stopping it, give it the highest
   document extraction priority in **Providers** and turn the rollout off.

### Check whether a document extraction model is getting worse
Keywords: extraction accuracy over time, model drift, accuracy dropped, provider accuracy trend, weekly accuracy, extraction getting worse, accuracy alert
1. Open [AI control](/admin/agent-control), select **Quality** in the rail, then **Document
   extraction**, and choose **Accuracy**.
2. Scroll to **Accuracy by provider over time**. Each AI provider that read documents in the
   last 12 weeks has a row: its **Weekly accuracy** line, **Last week**, the four weeks
   before it, the **Change** between them, and a **Status**.
3. **Drifting** means last week's accuracy fell more than the allowed points below that
   provider's own previous four weeks. **Not enough data** means too few fields were confirmed
   to judge; the help on the panel says how many are needed.
4. When a provider drifts, the people who can update AI providers are notified on Monday. To
   respond, compare it with another provider under **Shadow**, or lower its document extraction
   priority in **Providers** so another provider reads documents first.

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

### Read or download what an agent run did
Keywords: run transcript, agent run log, what did the agent say, download transcript, background run, scheduled run
1. Open [AI control](/admin/agent-control), select **Activity** in the rail and then **Runs**.
2. Select a run to open it: its **Status**, what it was **Started by**, the **Model** and its
   **Summary**.
3. Open **Transcript** to read what the agent said and thought and each tool it called, with what
   it sent and got back. A long run keeps its opening and its end; the stretch left out between
   them is counted where it fell.
4. Select **Download** to save the transcript as a Markdown file, laid out the way a downloaded
   conversation is, with the number of messages left out stated in it.

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
3. Choose the **Kind** (**Instruction**, **Fact**, **Correction** or **Procedure**), write the
   **Memory**, and optionally set **Until**.
4. To make it about one record, set **About** to the kind of record and pick the record; leave it
   on **Every agent** for something every agent should know. Then select **Save memory**.
5. To stop agents reading an entry, open it and select **Retire memory**, or right-click it and
   choose **Retire**; **Restore** brings it back.

### Let agents learn from their work
Keywords: self-improving agents, learning, reflection, look back, lessons, procedures, what the agent learned
1. Open [AI control](/admin/agent-control) on **Overview**. In **Organization-wide**, select
   **Edit** and turn **Learn from their work** on or off for every agent, then save. It is on unless someone turned it off.
2. To change it for one agent, open the agent in **Agents** and turn **Learns from its work** on
   or off, then select **Save**.
3. Select **Memory** in the rail. **What agents learned** lists the latest times an agent looked
   back over its work: what made it look (a tool that worked after failing, a person correcting
   it, a proposal changed or refused, a reply rated unhelpful, a long task) and each lesson it
   kept, offered or turned down, with the reason.
4. A lesson shared beyond one person waits under **Nova suggests**, with the memory it would
   retire. Select **Approve** to keep it as written, **Edit** to reword it first, or **Dismiss**.
5. Open any memory in the list to see **Where it came from**: why it was kept, what made the
   agent look back, what was said, the conversation or run it was drawn from, and the memory it
   replaces or that has replaced it.

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

An agent looks back over a conversation once it has been quiet for ten minutes (or after an hour
of steady back and forth), and over a background run once every proposal it raised is decided.
It does so only when something in the work is worth learning from, so most conversations cost
nothing. A lesson is a **Procedure** (the steps that worked, naming the tools), a **Fact** no
record holds, or, from a person's own words, an **Instruction** that was not already saved. It is
kept for the person in the conversation unless they may create agent memories, follows their
saving preference, and replaces an older memory rather than contradicting it. A lesson for a team
or the whole organization, and anything learned from work that read outside content, is only
ever offered. Retiring or editing a learned memory works like any other.

A memory holds up to 4,000 characters. Each prompt carries only as much memory as the agent's
**Memory in the prompt** setting allows (6,000 tokens unless changed), starting with what is
recorded about the record the conversation is about. When the organization keeps close to 5,000
active memories, the Memory section warns that fewer of them reach each prompt; retire what no
longer holds.

Opening the page needs read access to AI control. Each section in the rail appears only for
people who may read it (agents, AI providers, agent runs, agent proposals, agent exceptions,
agent memory); a section someone cannot open is left out. **Safety** appears for people who may
read agents, and **Quality** for people who may read the evaluation suite. **Worst rated**
and satisfaction need read access to agent feedback, **Run suite now** and **Add to golden set**
need create access to the evaluation suite, and changing **Sweep settings** needs update access to
both the evaluation suite and AI control, because the budgets are spend. The organization-wide switches need
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

Every night, at the hour chosen in **Sweep settings** (in the organization's timezone unless another
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
