---
path: /desk
aliases: [AI assistant, assistant, agents, chat, ask AI, copilot, AI desk, conversations]
related:
  - /desk/decisions
  - /desk/watchtower
  - /inbox
  - /admin/agent-control
covers:
  - /desk/t/:threadId
---

## What it's for
The Desk is where you talk to Trenova's AI agents about the work in front of you: a shipment, a driver, a customer, or how to do something in the app. Its **Today** page shows an ask box with the agent it will ask and a few starter questions that agent can answer, **Your day** (a short briefing), a link to changes **Waiting on your decision**, **Agents** (every agent you can ask, the ones you use most first, with **Search agents** and a filter for agents made from a template or built by hand), and **Where you left off** with your recent conversations. **Conversations** at the top lists your conversations, grouped by agent with pinned ones first, plus **New conversation** and links to **Watchtower** and **Decisions**.

Each conversation opens in the middle, and anything an agent produces (a table, a draft, a document) opens in the workspace beside it. Changes an agent wants to make to your records are not applied on their own; they wait on [Decisions](/desk/decisions) for someone to approve.

## Tasks

### Ask an agent a question
Keywords: chat with AI, ask the assistant, start a conversation
1. Open [Desk](/desk).
2. The ask box shows which agent it will ask. To ask a different one, select the agent's name in the box and pick another, or select a card under **Agents**; use **Search agents** when there are many.
3. Type your question and send it, or select one of the starter questions under the box.
4. Keep asking follow-ups in the conversation. Open anything the agent produces in the workspace beside it.

### Start a conversation without asking a question yet
Keywords: new chat, blank conversation
1. Open [Desk](/desk) and select **Conversations** at the top.
2. Select **New conversation**, then search for and pick the agent.

### Go back to an earlier conversation
Keywords: find chat, previous conversation, conversation history
1. Open [Desk](/desk).
2. Select a conversation under **Where you left off**, or find it in the left list with **Search conversations**.

### Organize conversations
Keywords: pin chat, delete conversation, export transcript, rename conversation
1. Open the conversation from [Desk](/desk).
2. Use **Pin conversation** (or **Unpin conversation**) to keep it at the top, **Download transcript** to save it, or **Delete conversation** and confirm with **Delete conversation**.
3. Use **Hide the workspace** or **Show the workspace** to make room for the conversation.

## Notes
Needs read access to the assistant. If no agents are available, an administrator has to connect an AI provider and enable an agent in [AI control](/admin/agent-control); people who can manage agents see **Open AI control** on the Desk.

When an agent in your own conversation asks to make several changes as one plan, answer it right there with **Approve all** or **Reject all**; you need only access to the assistant and to that agent, and each change still runs only if your own permissions allow it.

A long conversation eventually becomes read-only; select **Start a new conversation** to carry on. A conversation also becomes read-only when its agent is turned off, removed, set to run on its own, or no longer available to you, and the note in place of the message box says which, and whether an administrator can change it.
