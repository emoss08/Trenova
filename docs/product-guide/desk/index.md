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
The Desk is where you talk to Trenova's AI agents about the work in front of you: a shipment, a driver, a customer, or how to do something in the app. Its **Today** page greets you with the date and one line on what the day holds (the morning briefing's headline when one was written, otherwise how many decisions are waiting on you), and below it the box you ask in. The box names the agent it will ask and types out that agent's starter questions as its placeholder: press Tab to take the one on screen, or ⌘1–3 (Ctrl+1–3) to ask it outright. The rail down the left holds **New conversation**, search, links to **Today**, **Watchtower** and **Decisions**, every conversation grouped by when it was last used (**Pinned** first, then **Today**, **Yesterday**, **Previous 7 days**, **Previous 30 days** and **Older**), **Desk settings** and **Back to Trenova**. On a phone the rail slides over the page from the menu button at the top left.

Each conversation opens in the middle, and anything an agent produces (a table, a draft, a document, a plan) opens in the workspace beside it. Changes an agent wants to make to your records are not applied on their own: in a Desk conversation they wait in a card above the message box, and they are also listed on [Decisions](/desk/decisions).

## Tasks

### Ask an agent a question
Keywords: chat with AI, ask the assistant, start a conversation
1. Open [Desk](/desk).
2. The box shows which agent it will ask. To ask a different one, select the agent's name in the box and pick another from **Recent** or **All agents**; use **Search agents** when there are many.
3. Type your question and send it, or press Tab to use the starter question on screen.
4. Keep asking follow-ups in the conversation. Anything the agent produces opens in the workspace beside it.

### Approve a change on the Desk
Keywords: approve in desk, agent wants to change, review agent change, not now, redraft, out of date
1. While a change waits on you, a card stands above the message box. It names the change, how many records it touches and the value it would set, and says whether it can be undone.
2. Select **Approve** (or press ⌘Enter / Ctrl+Enter) once it has been checked against your records. A plan of several steps approves as one.
3. Select **Review** to read the change in full in the workspace, or **Not now** to put it off; it then waits in the workspace and on [Decisions](/desk/decisions), where you can also decline it or change its values first.
4. If the card says the change is out of date, what it was drafted against has changed since. Select the redraft button on the card to ask the agent to draft it again with the records as they are now.

### Answer a change an agent proposes in the floating assistant
Keywords: approve in chat, review agent change, preview before approving, what will this change, approval box, tell the agent no
1. While a change waits on you, the approval box stands where you would type, at the foot of the conversation in the floating assistant. Its first line names the change, the record it is about and the amount it comes to, with **Permanent** beside it when it cannot be undone; under that, one sentence says what it would do. Select **Details** to see each record before and after, any message or amount it would send or move, and why the agent asked. Anything that would stop the change, or that moved since it was proposed, stays in view above the buttons. When several wait, it shows the oldest first and says which one it is ("1 of 3"); the next takes its place once you decide.
2. Select **Approve** once the preview has loaded (or press ⌘Enter / Ctrl+Enter; Enter alone never approves), **Reject** to turn it down, or **Modify** (the pencil beside the buttons) to change the values first; the dialog shows what your values would do as you type and confirms with **Approve with changes**. A change over a list of records says how many it covers on its first line; open **Details**, untick any to leave out and approve with **Approve the kept records**.
3. To turn it down and say what to do instead, select **Tell the agent** (or press Esc), type your note and select **Send to the agent**. The agent reads it and answers in the conversation, often with a corrected proposal.
4. To keep talking first, select **Decide later** (the clock beside the buttons, or Alt+L). The box folds into a pill above the message box saying how many decisions wait; select it to bring the approval box back.
5. If the box says **Changed since it was proposed**, it can only be rejected; ask the agent again for a fresh proposal.

### Start a new conversation
Keywords: new chat, blank conversation, start over
1. Select **New conversation** in the rail of [Desk](/desk), or press ⌘N (Ctrl+N).
2. Pick the agent in the box and ask your question.

### Go back to an earlier conversation
Keywords: find chat, previous conversation, conversation history, search chats
1. Open [Desk](/desk).
2. Select the conversation in the rail, where conversations are grouped by when they were last used with pinned ones first.
3. To find one by what was said in it, open search from the rail (or press ⌘K / Ctrl+K) and type a load number, customer, invoice ID or any words from the conversation. Results cover conversations, messages and what they produced.

### Move, shrink or hide the assistant on any page
Keywords: assistant covers the screen, move chat button, hide AI button, assistant too big, resize assistant, assistant in the way
1. To move it, drag the assistant button. While you drag, the four corners it can go to are outlined and the one it will land in is highlighted and named; let go anywhere in that quarter of the screen. The panel opens in the same corner.
2. To hide it, hover over the button and select **Hide the assistant button**. A thin tab stays on the edge of the screen; select it, or press ⌘J (Ctrl+J), to open the assistant. The tab turns amber when a decision is waiting.
3. To resize the open panel, drag its free corner (the one away from the screen edge), or focus that corner and use the arrow keys. Double-click the corner to go back to the standard size.
4. The same choices are under **Position and size** at the top of the open panel: pick a corner, turn **Hide the button when closed** on or off, or select **Reset size**.

### Organize conversations
Keywords: pin chat, delete conversation, export transcript, rename conversation, hide workspace, make room
1. Open the conversation from [Desk](/desk).
2. In the bar at the top, use **Pin conversation** (or **Unpin conversation**) to keep it at the top of the rail, or **Download transcript** to save it.
3. To rename it, double-click its name in the rail, type the new name and press Enter.
4. To delete it, hover over it in the rail, select **Delete conversation**, then **Delete** to confirm.
5. Select **Workspace** in the top bar (or press ⌘\\ / Ctrl+\\) to show or hide what the agent produced, or **Hide artifacts** inside it to make room for the conversation.

## Notes
Needs read access to the assistant. If no agents are available, an administrator has to connect an AI provider and enable an agent in [AI control](/admin/agent-control); people who can manage agents see **Open AI Control** on the Desk.

When an agent in your own conversation in the floating assistant proposes a change, the approval box shows what the change would do before you answer: a sentence worked out from your records as they are now, and under **Details** each record it would touch with its values before and after, the message it would send as it would go out, and any amounts it would move. Select **Show all changes** for the rest of a long change. **Approve** waits until this has loaded, and stays off if the record was edited after the agent proposed the change (**Changed since it was proposed**); reject it and ask again. Values you may not see read **Hidden by your data access**. If the change moved while you were reading it, nothing is recorded and the box shows it again so you can approve what is there now. An email draft in the workspace shows the message as it would really go out, with its real recipients.

The conversation keeps one line for each change ("Proposed: … · Approved by you 8:52 PM", or "Waiting — decide below" while it waits), which opens onto what it would do or did; the only buttons that decide are in the approval box, or on the Desk in the card above the message box. A change the agent asks you to look at again opens in the workspace beside the conversation; select **Decide in the approval box** to bring it up. Deciding in the box or on [Decisions](/desk/decisions) updates both.

When an agent in your own conversation in the floating assistant asks to make several changes as one plan, or several changes of the same kind at once, the approval box takes one answer for all of them: approve them all or select **Reject all**. Under **Details** each step or change shows what it would change, and a step that starts from a record an earlier step changes says so; a step that would not go through is named above the buttons. Several changes of one kind are read as the box appears, whether or not **Details** is open, so approving them records each one as reviewed. Past the first five, each change is read only once you open it under **Details**, and one you never opened is recorded as approved without reviewing what it changes. You need only access to the assistant and to that agent, and each change still runs only if your own permissions allow it.

A long conversation eventually becomes read-only; select **Start a new conversation** to carry on. A conversation also becomes read-only when its agent is turned off, removed, set to run on its own, or no longer available to you, and the note in place of the message box says which, and whether an administrator can change it.
