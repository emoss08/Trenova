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

Each conversation opens in the middle, and anything an agent produces (a table, a draft, a document) opens in the workspace beside it. Changes an agent wants to make to your records are not applied on their own: in your conversation they wait in the approval box at its foot, and they are also listed on [Decisions](/desk/decisions).

## Tasks

### Ask an agent a question
Keywords: chat with AI, ask the assistant, start a conversation
1. Open [Desk](/desk).
2. The ask box shows which agent it will ask. To ask a different one, select the agent's name in the box and pick another, or select a card under **Agents**; use **Search agents** when there are many.
3. Type your question and send it, or select one of the starter questions under the box.
4. Keep asking follow-ups in the conversation. Open anything the agent produces in the workspace beside it.

### Answer a change an agent proposes in the conversation
Keywords: approve in chat, review agent change, preview before approving, what will this change, approval box, tell the agent no
1. While a change waits on you, the approval box stands where you would type, at the foot of the conversation (on the Desk and in the floating assistant alike). Its first line names the change, the record it is about and the amount it comes to, with **Permanent** beside it when it cannot be undone; under that, one sentence says what it would do. Select **Details** to see each record before and after, any message or amount it would send or move, and why the agent asked. Anything that would stop the change, or that moved since it was proposed, stays in view above the buttons. When several wait, it shows the oldest first and says which one it is ("1 of 3"); the next takes its place once you decide.
2. Select **Approve** once the preview has loaded (or press ⌘Enter / Ctrl+Enter; Enter alone never approves), **Reject** to turn it down, or **Modify** (the pencil beside the buttons) to change the values first; the dialog shows what your values would do as you type and confirms with **Approve with changes**. A change over a list of records says how many it covers on its first line; open **Details**, untick any to leave out and approve with **Approve the kept records**.
3. To turn it down and say what to do instead, select **Tell the agent** (or press Esc), type your note and select **Send to the agent**. The agent reads it and answers in the conversation, often with a corrected proposal.
4. To keep talking first, select **Decide later** (the clock beside the buttons, or Alt+L). The box folds into a pill above the message box saying how many decisions wait; select it to bring the approval box back.
5. If the box says **Changed since it was proposed**, it can only be rejected; ask the agent again for a fresh proposal.

### Start a conversation without asking a question yet
Keywords: new chat, blank conversation
1. Open [Desk](/desk) and select **Conversations** at the top.
2. Select **New conversation**, then search for and pick the agent.

### Go back to an earlier conversation
Keywords: find chat, previous conversation, conversation history
1. Open [Desk](/desk).
2. Select a conversation under **Where you left off**, or find it in the left list with **Search conversations**.

### Move, shrink or hide the assistant on any page
Keywords: assistant covers the screen, move chat button, hide AI button, assistant too big, resize assistant, assistant in the way
1. To move it, drag the assistant button. While you drag, the four corners it can go to are outlined and the one it will land in is highlighted and named; let go anywhere in that quarter of the screen. The panel opens in the same corner.
2. To hide it, hover over the button and select **Hide the assistant button**. A thin tab stays on the edge of the screen; select it, or press ⌘J (Ctrl+J), to open the assistant. The tab turns amber when a decision is waiting.
3. To resize the open panel, drag its free corner (the one away from the screen edge), or focus that corner and use the arrow keys. Double-click the corner to go back to the standard size.
4. The same choices are under **Position and size** at the top of the open panel: pick a corner, turn **Hide the button when closed** on or off, or select **Reset size**.

### Organize conversations
Keywords: pin chat, delete conversation, export transcript, rename conversation
1. Open the conversation from [Desk](/desk).
2. Use **Pin conversation** (or **Unpin conversation**) to keep it at the top, **Download transcript** to save it, or **Delete conversation** and confirm with **Delete conversation**.
3. Use **Hide the workspace** or **Show the workspace** to make room for the conversation.

## Notes
Needs read access to the assistant. If no agents are available, an administrator has to connect an AI provider and enable an agent in [AI control](/admin/agent-control); people who can manage agents see **Open AI control** on the Desk.

When an agent in your own conversation proposes a change, the approval box shows what the change would do before you answer: a sentence worked out from your records as they are now, and under **Details** each record it would touch with its values before and after, the message it would send as it would go out, and any amounts it would move. Select **Show all changes** for the rest of a long change. **Approve** waits until this has loaded, and stays off if the record was edited after the agent proposed the change (**Changed since it was proposed**); reject it and ask again. Values you may not see read **Hidden by your data access**. If the change moved while you were reading it, nothing is recorded and the box shows it again so you can approve what is there now. An email draft in the workspace shows the message as it would really go out, with its real recipients.

The conversation keeps one line for each change ("Proposed: … · Approved by you 8:52 PM", or "Waiting — decide below" while it waits), which opens onto what it would do or did; the only buttons that decide are in the approval box. A change the agent asks you to look at again opens in the approval box, and in the workspace beside it select **Decide in the approval box** to bring it up. Deciding in the box or on [Decisions](/desk/decisions) updates both.

When an agent in your own conversation asks to make several changes as one plan, or several changes of the same kind at once, the approval box takes one answer for all of them: approve them all or select **Reject all**. Under **Details** each step or change shows what it would change, and a step that starts from a record an earlier step changes says so; a step that would not go through is named above the buttons. Several changes of one kind are read only once you open **Details**, and approving them without opening it is recorded as approved without reviewing what they change. You need only access to the assistant and to that agent, and each change still runs only if your own permissions allow it.

A long conversation eventually becomes read-only; select **Start a new conversation** to carry on. A conversation also becomes read-only when its agent is turned off, removed, set to run on its own, or no longer available to you, and the note in place of the message box says which, and whether an administrator can change it.
