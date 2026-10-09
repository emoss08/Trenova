---
path: /desk
aliases: [AI assistant, assistant, agents, chat, ask AI, copilot, AI desk, conversations, steer the agent, steering, interrupt the agent, slash commands, at mention, dictate, compact conversation, pinned facts, snooze, hand off, handoff, chapter, delegate, artifacts, citations, morning briefing]
related:
  - /desk/decisions
  - /desk/watchtower
  - /inbox
  - /admin/agent-control
covers:
  - /desk/t/:threadId
  - /desk/agents/:agentId
  - /desk/c/:threadId/a/:slug
  - /desk/c/:threadId/a/:slug/page
---

## What it's for
The Desk is where you talk to Trenova's AI agents about the work in front of you: a shipment, a driver, a customer, or how to do something in the app. Its **Today** page greets you with the date and one line on what the day holds (the morning briefing's headline when one was written, otherwise how many decisions are waiting on you), and below it the box you ask in. The box names the agent it will ask and types out that agent's starter questions as its placeholder: press Tab to take the one on screen, or ⌘1–3 (Ctrl+1–3) to ask it outright. The rail down the left holds **New conversation**, search, links to **Today**, **Watchtower** and **Decisions**, every conversation grouped by when it was last used (**Pinned** first, then **Today**, **Yesterday**, **Previous 7 days**, **Previous 30 days** and **Older**), **Desk settings** and **Back to Trenova**. On a phone the rail slides over the page from the menu button at the top left.

Each conversation opens in the middle, and anything an agent produces (a table, a draft, a document, a plan) opens in the workspace beside it. Changes an agent wants to make to your records are not applied on their own: in a Desk conversation they wait in a card above the message box, and they are also listed on [Decisions](/desk/decisions).

While an agent replies you can steer it with a correction or queue the next question. The message box names records with @, takes slash commands, files, scans and dictation, and can share the page you came from. A conversation about a shipment, invoice or dispute becomes a case, with a ready-to-bill or ready-to-close checklist, snoozes and waiting on the customer's or carrier's reply. Conversations can also run a request on a schedule, be handed to another agent, keep pinned facts in mind, and compact themselves when they grow long. Each fact an agent looked up carries a number tying it to the step that found it.

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
Keywords: approve in chat, review agent change, preview before approving, what will this change, approval card, not now
1. The assistant on every page asks the same way the Desk does: while a change waits on you, a card stands above the message box. It names the change, how many records it touches and the value it would set, and says whether it is reversible or can't be undone.
2. Select **Approve** once it has been checked against your records (or press ⌘Enter / Ctrl+Enter; Enter alone never approves). For a few seconds after, a bar says when it goes through: select **Undo** to take it back, or **Do it now** to let it go straight away.
3. Select **Review** to read the change in full: the assistant opens the conversation on the [Desk](/desk) with the change beside it, where you can also decline it or change its values first. Select **Not now** to put it off; it waits on [Decisions](/desk/decisions), and a pill above the message box says how many changes wait on you.
4. If the card says the change is out of date, what it was drafted against has changed since. Select the redraft button on the card to ask the agent to draft it again with the records as they are now.

### Keep or undo what an agent learned
Keywords: agent learned, learned from this conversation, self-improving, memory, procedure, lesson
1. Once a conversation has been quiet for a while, the agent looks back over it when something went wrong, took several tries or was corrected. What it kept appears under the reply it learned from, marked as learned; a procedure is the steps that worked.
2. If you asked to be asked first, or the lesson would reach your team or everyone, the card asks instead. Edit the words, choose who it is for, then select **Save memory**, or **Don't save**.
3. To take back a lesson it kept, select **Undo** on the card, or pause or forget it on the Memory page like any other memory.


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
Keywords: assistant covers the screen, move chat button, hide AI button, assistant too big, resize assistant, assistant in the way, dock assistant, full screen assistant
1. The assistant opens in one of three layouts. Floating, it sits in a corner of the page; select **Dock to side** to dock it down the side of the screen instead, and the page moves over to make room for it; select **Float** to put it back in the corner. Select **Full screen** to give it the whole screen, with your conversations listed down its side, and **Shrink** (or Esc) to return to the corner.
2. To move it, drag the assistant button. While you drag, the four corners it can go to are outlined and the one it will land in is highlighted and named; let go anywhere in that quarter of the screen. The panel opens in the same corner, or docks to that side.
3. To hide it, hover over the button and select **Hide the assistant button**. A thin tab stays on the edge of the screen; select it, or press ⌘J (Ctrl+J), to open the assistant. The tab stays open and turns amber when a decision is waiting.
4. To resize the floating panel, drag its free corner (the one away from the screen edge), or focus that corner and use the arrow keys. Double-click the corner to go back to the standard size.
5. The same choices are under **Position and size** at the top of the open panel: pick a layout and a corner, turn **Hide the button when closed** on or off, or select **Reset size**.

### Find a conversation in the assistant
Keywords: assistant history, earlier chat in assistant, conversations list, search assistant
1. In the floating or docked assistant, select **Conversations** at the top. The ones a change is waiting on are listed first, then today, yesterday and earlier days; type in the search to find one by its title or its agent.
2. Full screen, the same list runs down the side. Select **Search** (or press ⌘K / Ctrl+K) to search it, and press Esc to clear the search. Press ⌘\ (Ctrl+\) to fold the list away.
3. To carry a conversation over to the Desk, select **Conversation actions** beside its title and then **Open in Desk**. **Download transcript** and **Delete conversation** are there too.

### Organize conversations
Keywords: pin chat, delete conversation, export transcript, rename conversation, hide workspace, make room
1. Open the conversation from [Desk](/desk).
2. In the bar at the top, open **More actions** and use **Pin conversation** (or **Unpin conversation**) to keep it at the top of the rail, or **Download transcript** to save it.
3. To rename it, double-click its name in the rail, type the new name and press Enter.
4. To delete it, hover over it in the rail, select **Delete conversation**, then **Delete** to confirm.
5. Select **Workspace** in the top bar (or press ⌘\\ / Ctrl+\\) to show or hide what the agent produced, or **Hide artifacts** inside it to make room for the conversation.

### Steer an agent while it is replying
Keywords: steer, steering, interrupt, redirect, change direction, correct the agent mid-reply, add detail while working, send while working
1. While the agent works, a light runs round the message box and a line at its top says what the agent is doing.
2. Type your correction or extra detail in the box.
3. Press Enter, or select the round arrow button. The agent reads it at its next step and carries on with it in mind.
4. To stop the reply altogether, empty the box and select **Stop** (the square button).

### Queue a follow-up for after the reply
Keywords: queue message, send after, follow-up, line up questions, wait for reply, send next
1. While the agent works, type the next question.
2. Press ⌥Enter (Alt+Enter), or select **Queue a follow-up** beside the send button. It is sent on its own once the current reply finishes.
3. Waiting messages are listed above the box in the order they will go. Select **Hide** or **Show** to fold the list.
4. On a waiting message, use **Edit**, **Move up**, **Move down** or **Remove**. Select **Send it now** to send it straight away, or **Steer the reply with it now** to hand it to the reply under way.
5. If the last reply was stopped or failed, the list pauses. Select **Send next** to carry on.

### Name a shipment, customer or other record with @
Keywords: at mention, @ mention, reference a load, tag a shipment, point the agent at a record, link invoice
1. In the message box, type @ followed by part of a load number, customer, invoice, driver or carrier.
2. Narrow the list with **All**, **Shipments**, **Customers**, **Invoices**, **Drivers** or **Carriers** along its top.
3. Use ↑ ↓ to move, and Enter or Tab to insert the record. Esc closes the list.
4. The record shows in your message and goes with it, so the agent looks up that exact record. Delete the @name from the text to drop it.

### Use a slash command
Keywords: slash command, shortcut, /status, /quote, /report, /explain, quick question
1. In an empty message box, type /. The list shows **Commands** first, then the agent's starter questions.
2. Use ↑ ↓ to pick one, then Enter or Tab. A starter question is sent as it is.
3. For a command that needs details, such as /status (a PRO or shipment number), /quote (origin and destination city) or /report (a report name), type each one after the command. The grey hints show what is still missing, and **Desk will be asked** shows the full question.
4. Press Enter once every detail is filled in. Press Esc to send what you typed as plain text instead.
5. /explain asks the agent to explain the page you came from.

### Attach files to a message
Keywords: attach, upload, add a document, send a PDF, rate confirmation, BOL, spreadsheet, drag and drop
1. Select **Attach files** (the plus at the bottom left of the box), then **Upload from computer**. You can also press ⌘U (Ctrl+U), or drag files anywhere onto the Desk and drop them on the box.
2. Each file shows above the text while it uploads. Select **Remove** to take one off, or **Try again** if an upload failed.
3. Type your question and send it. If you send files with no text, the agent is asked what is in them.

### Scan paper into a message
Keywords: scan, scanner, paperwork, scan a BOL, scan a POD, Trenova Capture
1. In an open conversation, select **Attach files**, then **Scan from Capture**.
2. Under **Computer**, pick the computer your scanner is connected to. Optionally choose the **Scanner**, **Scan settings** and **Document type** (or leave **Let Desk decide**).
3. Select **Start scan** and put the pages in the scanner.
4. The scan shows above the text and counts pages as they arrive. Select **Done scanning** when the last page is in, or **Cancel scan** to drop it.
5. If no computer is set up, select **Download Trenova Capture** or **My scanners** to set one up on [My scanners](/capture/devices).

### Dictate a message
Keywords: voice, speak, talk to the agent, dictation, microphone, speech to text
1. Select **Dictate** (the microphone beside the send button).
2. Speak. Your words appear in the box as they are recognised, and a timer shows how long it has been listening.
3. Select **Stop dictating** when you are done. Check the text, then send it.

### Share the page you came from
Keywords: share page, what I'm looking at, page context, stop sharing, explain this page
1. The chip in the message box names the page you had open before coming to the Desk. If nothing is shared, it reads **No page**.
2. Select the chip to see the page. Turn **Share this page with Desk** on to send it with each message, or off so the agent does not see it.
3. With a page shared, select **Explain what's on this page** to ask the agent about it.

### Choose which model answers
Keywords: model, switch model, AI model, change provider, faster model, reasoning model
1. Select the model chip near the send button. It reads **Auto** unless you picked a model.
2. Pick **Auto** (**Recommended**: your organization's order decides, and it keeps working if a provider goes down), or a specific model. Use **Search models…** to find one.
3. Models that failed their last check are listed but cannot be chosen. Your administrators decide which models are offered.

### Ask a different agent from a conversation
Keywords: switch agent, another agent, different assistant
1. Select the agent's name in the message box.
2. Pick one from **Recent** or **All agents**, or type in **Search agents**.
3. A new conversation starts with that agent. Anything you had typed is asked there.

### See how full the conversation is and compact it
Keywords: context window, conversation too long, memory full, compact, summarize conversation, tokens, usage limits
1. In a conversation, the ring beside the model chip fills as the conversation grows. From three quarters full it shows a percentage, then turns amber and red.
2. Select the ring to open **Context window**. **See detailed breakdown** shows what takes the space.
3. Under **Compact conversation**, select **Compact** to summarize older turns. You can also type /compact in the box.
4. While it compacts, the box takes nothing new. Select **Cancel** at the top of the box to stop it.
5. Use the switch below to let the conversation compact itself as it nears full.
6. **Usage limits** in the same panel shows how much of your and the agent's daily and monthly allowance is used, and when each resets.

### Read what a compaction kept
Keywords: compacted, summary, what did it forget, auto-compacted, earlier messages
1. Where the conversation was compacted, a line runs across it with a **Compacted** (or **Auto-compacted**) pill saying how many earlier messages were summarized.
2. Select the pill. **What the agent carries forward** lists the summary, and **Kept in full** lists what was not summarized, such as **Pinned facts** and **Pending approvals**.
3. Messages above the line are dimmed, because the agent now reads them only through the summary.

### Pin a fact for the whole conversation
Keywords: pin fact, remember this, keep in mind, always remember, context note
1. In the **Keeping in mind** bar above the message box, select **Pin a fact**.
2. Type the fact, for example the invoice date or a customer's rule, and press Enter. Esc cancels.
3. Every agent in the conversation keeps it in mind for every reply, including after the conversation is compacted.
4. Select **Unpin** (×) on a fact to remove it.

### See or cancel what an agent is waiting on
Keywords: waiting, wait until, parked, notify me when, truck arrives, reply from carrier, follow up later, cancel wait, picked up after a wait
1. Ask the agent to pick a task back up later, for example "tell me when the truck reaches the consignee" or "check back when the carrier replies".
2. Above the message box, a list shows what the agent is waiting on: the kind of wait (an arrival, a departure, a reply, an appointment, free time, drive time or a time), the agent's description, and when it is due or gives up.
3. Nothing runs while it waits. When the event happens, the agent comes back to the conversation on its own, and a note marks the place: **Picked up after a wait**, or **Picked up after the wait ran out** if it gave up first.
4. Select **Cancel the wait** (×) to drop one you no longer need.

### Make a conversation a case about a shipment, invoice or dispute
Keywords: case, cases, link conversation to shipment, attach invoice, open a case, dispute case, about this load
1. Open the conversation from [Desk](/desk).
2. In the bar at the top, select the case button (its tip reads **Make this a case**).
3. Under **Make this a case about**, choose **Shipment**, **Invoice** or **Dispute**, search for the record (shipments by PRO or BOL number, invoices and disputes by invoice number or customer), and pick it.
4. A card for the case now stands above the message box. It names the record, which you can select to open, and shows where the case stands.

### Take a case to ready to bill or ready to close
Keywords: case, ready to bill, ready to close, checklist, next step, POD, rate confirmation, paperwork, billing checklist, what's left
1. Open the case's conversation from [Desk](/desk). The card above the message box shows the checklist: **Ready to bill** for a shipment, **Ready to close** for an invoice, with how many steps are done out of how many.
2. Select the checklist name to open it. Each step shows whether it is done, holding the record up, or waiting on something outside your control, with a line saying why (for example, which carrier has not confirmed the rate confirmation).
3. Select the button for the next step, for example **Request POD**, **Chase the rate confirmation**, **Mark ready to invoice** or **Send invoice**. It asks the case's agent to take that step, worded as your own message. If that agent can't take it but another agent you may use can, the same button hands the step to that agent right here in the conversation; point at the button to see which agent takes it. If no agent you may use can take it, the button opens the page where you do it yourself. A step that is holding the record up has its own button in the open checklist too.
4. For a step your team added that a person ticks off, select its box in the checklist to tick it or untick it.

### Snooze a case
Keywords: case, snooze, remind me later, put away, hide until, until appointment, until ETA, tomorrow morning, next week
1. Open the case's conversation and select **Case** in the bar at the top.
2. Under **Snooze**, pick **In 3 hours**, **Tomorrow morning** or **Next week**. Each one shows the time it ends.
3. For a shipment you can also pick **Until the next appointment**, which moves when the appointment does and ends when the truck reaches that stop, or **Until the ETA**, which follows the truck's estimate.
4. To choose your own time, select **Pick a time…**, set **Snooze until**, then select **Snooze**.
5. To bring it back early, open the menu again and select **Wake it now**.

### Wait for the customer or carrier to reply on a case
Keywords: case, waiting on customer, waiting on carrier, follow up when they answer, park until reply, waiting on a reply
1. Open the case's conversation and select **Case** in the bar at the top.
2. Under **Wait for a reply from**, pick the record's customer or one of its carriers.
3. The case shows **Waiting on the customer** or **Waiting on the carrier**. When that party's reply comes in, the agent picks the case back up on its own.

### Move a case to another record or stop treating it as a case
Keywords: case, wrong shipment, change record, unlink, remove case, move case
1. Open the case's conversation and select **Case** in the bar at the top.
2. To point it at a different record, select **Move to another record** and pick the record.
3. To make it an ordinary conversation again, select **Stop treating this as a case**. The conversation stays; it is just no longer about the record.

### Choose which steps a case checklist shows
Keywords: case checklist, ready to bill checklist, ready to close checklist, required steps, optional step, turn off step, customer does not want notification, billing checklist settings
1. On [Desk](/desk), open **Settings** from the bottom of the sidebar and select **Case checklists** under **Organization**. You need read access to billing control to see it, and update access to change it.
2. Choose **Ready to bill** (shipments) or **Ready to close** (invoices), then **Every customer** for your organization's checklist.
3. Under **Steps you set**, choose **Required**, **Optional** or **Off** for each step. A required step keeps the record from being ready; an optional one is shown and ticked but never blocks; one that is off is not shown.
4. Steps under **Always required** follow rules kept elsewhere, such as the customer's billing profile or billing control, and stay required.
5. Select **Save changes** in the bar at the bottom, or **Discard** to drop your changes.

### Put a case checklist's steps in order
Keywords: reorder checklist, step order, move step, drag step
1. In **Settings** › **Case checklists**, open the checklist and select **Reorder**.
2. Drag a step by its handle, or focus the handle and press the up or down arrow to move it one place. Steps that are always required can be moved too.
3. Select **Done**, then **Save changes**.

### Add a step of your own to a case checklist
Keywords: custom checklist step, add step, manual tick, document step, lumper receipt
1. In **Settings** › **Case checklists**, open the checklist and select **Add a step**.
2. Give it a **Name** and choose **What ticks it**: **A person on the case**, who ticks it on the case, or **A document on file**, which ticks it once an accepted copy of the chosen **Document type** is attached to the shipment.
3. Under **When it's the next step**, optionally set the **Button** and **What it asks the agent**. The preview shows the button as the case will draw it.
4. Select **Done**, then **Save changes**. A checklist can have up to 12 steps of your own.

### Give a customer its own case checklist
Keywords: customer checklist, per customer steps, customer bills differently
1. In **Settings** › **Case checklists**, select **Add a customer** and find the customer. Its checklist starts as a copy of your organization's.
2. Change its steps, then select **Save checklist**.
3. To send the customer back to your organization's checklist, open it and select **Remove this checklist**, then select it again to confirm. **Go back to the default** does the same for your organization's own checklist.

### Hand a conversation to another agent
Keywords: hand off, handoff, transfer, pass to another agent, switch agent keep context
1. Open the conversation from [Desk](/desk) and select the hand-off button in the bar at the top (its tip reads **Hand off to another agent**).
2. Under **Hand off to**, pick the agent. The agents your current agent already works with are listed first.
3. This starts a new conversation with that agent. It carries over a summary of this conversation, its pinned facts and its pinned artifacts.
4. A card in this conversation says who it was handed to and what went with it (**Carried over**). Select **Open their conversation** to go there. The new conversation has a matching card with **Open the earlier conversation**. Both conversations stay open.

### Have an agent run a request on a schedule
Keywords: schedule, recurring, every morning, every weekday, every Monday, daily report, run automatically
1. In a conversation on [Desk](/desk), send a message that starts with when it should run, then what to ask. For example: "every weekday at 7:30am, list shipments still not dispatched for today". You can also type /schedule, then the when and the request.
2. Instead of an answer, a **Scheduled** card appears: **Results will post into this conversation**. The card shows the request, how often it runs and when it runs next.
3. On the card, select **Pause** to stop it for now, **Resume** to start it again, or **Delete** to remove it. A deleted one shows **Schedule deleted**.

### Follow a task an agent handed to another agent
Keywords: delegate, asked another agent, sub-agent, agent asked agent
1. When the agent you are talking to hands part of the work to another agent, the reply shows a step headed "Asked" and that agent's name, with the task beside it.
2. While the other agent works, its steps appear beneath it. When it finishes, the step says how it ended, what it made and anything **Waiting for your approval**.
3. Open the step to read the **Task**, **What it did** and what it found.

### Take back an approval on the Desk
Keywords: undo approval, approved by mistake, cancel approved change, do it now
1. Right after you approve a change, a bar with a countdown takes the card's place. It says what was approved and when it goes through.
2. Select **Undo** within those few seconds to take it back; the change waits on you again. Select **Do it now** to let it go straight away.

### Open what an agent produced
Keywords: artifacts, workspace, agent output, table the agent made, switch artifact, find artifact, pin artifact, all artifacts
1. Open the conversation from [Desk](/desk). Select **Workspace** in the top bar to open the workspace beside it. The number on the button is how many things the conversation has produced.
2. The open item sits in front of a small stack of the others. Select it (**Switch artifact**) to fan the stack out and pick another, or use **Previous** and **Next** to step through them.
3. To see everything the conversation produced, select **All artifacts** at the bottom of the fanned stack, or press ⌘J (Ctrl+J). Type in **Search artifacts** to find one by its title, the tool that made it or the turn it came from, or narrow the list with **All**, **Pinned** or a kind such as a table, record or document. Press Enter to open the first match.
4. Select **Pin to conversation** at the bottom of the workspace to keep an item at the top of the conversation. Select it again to unpin it.
5. Select **Copy link** to copy a link that opens the conversation with this item beside it, or **Open on its own page** to read it full width in a new tab. On that page, **Open in conversation** takes you back.
6. Select **Hide artifacts** to close the workspace.

### Compare versions of a table or report
Keywords: version history, what changed, earlier version, read again, changed cells
1. When an agent reads the same table again later in the conversation, the result becomes a new version of the same item rather than a separate one. A version number shows on the item.
2. Select the version number (**Versions**) to list every version. Each one says what changed from the version before it, such as rows added, rows gone or values changed. The newest is marked **Latest**.
3. Pick an earlier version to view it. Cells that changed since the version before are highlighted.

### Filter, sort and export a table
Keywords: export CSV, download table, spreadsheet, sort rows, filter results, Excel
1. Open the table in the workspace beside the conversation.
2. Type in the filter box (**Filter rows**) to show only the rows that contain your text. Select a column heading to sort by it, select it again to reverse the order, and a third time to clear the sort.
3. Select a row, or the arrow at the end of it (**Open this record**), to open the record it is about.
4. Select **Export CSV** at the bottom of the workspace to download the table. A report shown as bars has **Download CSV** and **Open full report** under it instead.
5. If the table says it is showing only some of the rows, ask the agent for the rest.

### Read and edit a document an agent wrote
Keywords: agent report, write-up, edit document, rewrite paragraph, shorten, simplify, restore version, document history
1. Open the document in the workspace. Its sources are listed at the end, and the numbers in the text say which source each statement rests on. Point at a number to see the source, and select **Open artifact** to open what it came from.
2. To rewrite one passage, select some text in a paragraph while in **Read**, then choose **Shorter** or **Plainer**, or type an instruction in **Ask to change…** and press Enter. The new wording shows under the old one. Select **Accept** to keep it, **Try again** for another attempt, or **Keep original** to drop it.
3. To edit by hand, select **Edit**, click any paragraph and type. Select the save button to save your changes as a new version, or **Discard** to throw them away. **Done** leaves editing when nothing has changed.
4. Every saved change is a new version. Select the version number (**Versions**) to open an earlier one, then **Restore** to make it the latest again, or **Back to latest** to return.

### Download or copy a document
Keywords: export document, PDF, Word, docx, Markdown, copy text, print report
1. Open the document in the workspace and select **Export**.
2. Choose **Download as PDF**, **Word document**, **Copy as Markdown** or **Copy text**.

### Review and send an email an agent drafted
Keywords: draft email, agent email, send message, edit draft, email approval
1. Open the email draft in the workspace. It shows who it goes to, the subject and the message, and **Why this wording** when the agent explained its choice.
2. While it waits on you, change the subject or the message as needed. If the message comes from your organization's template, the draft says so and the text is written when the email is sent.
3. Select **Send for approval** to send your version on for approval, or **Copy** to copy the subject and message. Once it is decided, the draft shows **Sent for approval**, **Sent** or **Set aside**.

### Create a shipment from a document an agent read
Keywords: rate confirmation, extract document, scanned load, read fields, create load from PDF, document intelligence
1. Upload a document in the conversation. When the agent reads it, the workspace shows the page with a box over every value it found, and the list of fields with how sure it is of each one. Fields that need a look are counted and marked.
2. Point at a field to light up where it was found on the page, and the other way round. Use the page numbers to move through a document with several pages.
3. Select **Fix fields** to ask the agent to check the doubtful fields against the document again.
4. Select **Create shipment from this**. The agent drafts the shipment, and it waits for your approval like any other change. Once it is approved, the item shows **Shipment created**. If the document is already attached to a shipment, it shows **Already a shipment** instead.

### Approve billing items from a Desk table
Keywords: billing queue, approve invoices, bulk approve, ready to bill, invoice review, assign biller
1. Ask an agent for your billing queue. The table shows each item's current status, and it stays up to date as the queue changes.
2. Tick the items you want, or use **Select all**. A bar shows how many are selected, how many are ready, how many have no biller and how many need you.
3. Select the approve button to approve the ready ones. If some have no biller, the bar asks first. Choose to assign yourself and approve them all, or approve only the ones that already have a biller. Select **Back** to change your mind.
4. For a few seconds after, the bar says how many were approved. Select **Undo** before the count runs out to stop it. Any invoice that could not be approved is reported once the approval finishes.
5. To work through items that need you, select the review button on the bar, or select a row to open the item.

### Review one billing item
Keywords: billing item, invoice check, proof of delivery, rate con mismatch, hold invoice, post invoice
1. Open the item from a billing queue table in the conversation. It shows who it bills, its status and how long it has been in the queue.
2. Under **Before it can be approved**, read the five checks: biller, charges against the rate con, proof of delivery, bill-to and terms, and not a duplicate. For a check with no biller, select **Assign to me** or **Someone else…**. For any other flagged check, pick one of the choices offered under it.
3. Under **Charges**, compare each line with the rate con. A line that was removed or adjusted can be put back with **Undo**. Under **Shipment**, see the lane and the paperwork, and open the shipment.
4. Select **Approve** once every check is clear, or **Hold** and pick **Waiting on paperwork**, **Customer dispute** or **Rate question**. A held item has **Release** to take it off hold.
5. An approved item has a post button with its total. Posting sends the invoice to the customer and can't be undone.
6. Use **Previous item** and **Next item** to move through the queue, and **Back to the queue** to return to the table. **Activity** lists what has happened to the item. Select **Show earlier** for older entries.

### Check where a fact in a reply came from
Keywords: citation, citations, source, step number, how did the agent know, reasoning, why did it do that, web sources
1. A reply carries small numbers after the facts the agent looked up. Point at a number to see the step behind it: what the agent did, what it found and how long it took.
2. Select **Why this step?** to read the agent's own reasoning (**Saw**, **Because**, **Instead of**). Select **Hide reasoning** to close it. If the step produced a table or record, select its name, or the number itself, to open it in the workspace.
3. When the agent searched the web, the reply names the site it cites. Point at it to list every page it cites, and select one to open it in a new tab. Under the finished reply, select the sources count to see the search and every page it found.
4. To hide the numbers, turn off **Source numbers** in Desk settings.

### Copy, pin or listen to a reply
Keywords: copy answer, bookmark reply, chapter, chapters, read out loud, text to speech
1. Hover over a reply in a Desk conversation to show its actions.
2. Select **Copy reply** to copy its text.
3. Select **Pin as chapter** to mark the reply as a chapter of the conversation. It then shows its chapter number. Select **Unpin chapter** to remove it.
4. Select **Read aloud** to hear the reply, and **Stop reading** to stop. This button only appears when your browser can read text aloud.

### See or change what an agent can do
Keywords: agent permissions, agent tools, ask first, agent limits, budget, business hours, turn agent off, what can this agent do
1. In a conversation, select the agent's name in the top bar, or open **More actions** and select **What this agent can do**.
2. **Look things up** lists what the agent can read, and **Make changes** lists what it can change. Each is set to **Allowed** or **Off**. A change can also be set to **Ask first**, so it waits for your approval. A row with a lock says why some choices are not available.
3. **Hands off to** shows which agents take questions outside this agent's work.
4. **Limits** shows **Requests today**, this month's budget, the **Largest single change** it may make at once, and whether it may **Only change things during business hours**.
5. People who manage agents can switch the agent **On** or **Off** and change any of these. Every change saves straight away. Everyone else sees the page but cannot change it.
6. Select **Back to conversation** to return.

### Search the whole Desk
Keywords: find conversation, find message, find artifact, find decision, search desk, recent searches
1. On [Desk](/desk), select **Search** in the rail or press ⌘K (Ctrl+K).
2. Type in **Search chats, messages, artifacts…**. Results are grouped as **Conversations**, **Messages**, **Artifacts** and **Decisions**.
3. Narrow the results with **All**, **Chats**, **Messages**, **Artifacts** or **Decisions**, or press Tab to move between them.
4. Use the arrow keys and Enter, or select a result. A conversation or message opens its conversation, an artifact opens the conversation with it beside it, and a decision opens [Decisions](/desk/decisions).
5. With nothing typed, the search offers **Recent searches** and recent conversations and artifacts.

### Change how the Desk looks and behaves
Keywords: desk preferences, dark mode, text size, send with enter, default agent, start page, turn off confetti, dictation, scanner
1. On [Desk](/desk), select **Desk settings** at the bottom of the rail.
2. Pick a section and change what you need. Every change applies straight away:
   - **Appearance**: **Theme**, **Conversation width**, **Motion** and **Text size**.
   - **Conversation**: **Send with** (**Enter** or **⌘ Enter**), **Source numbers**, and **Open Desk to** (**Today** or **Last conversation**).
   - **Composer**: **Start new conversations with** a chosen agent or **The last one I used**, **Share the page you came from**, **@ mentions**, **Slash commands**, **Dictation** and **Suggested questions**.
   - **Files & scanning**: **Drop files** (**Anywhere** or **On the composer**), **Scan with** and **Scan settings**.
   - **Artifacts**: **Open new artifacts** (**Right away** or **When I click**) and **New artifact dot**.
   - **Agent & approvals**: **Approval celebration** (**Confetti**, **Subtle** or **Off**) and **Working border**.
3. Select **Reset to defaults** to put everything back.

## Notes
Needs read access to the assistant. If no agents are available, an administrator has to connect an AI provider and enable an agent in [AI control](/admin/agent-control); people who can manage agents see **Open AI Control** on the Desk.

When an agent in your own conversation proposes a change, the card above the message box shows it the same way on the Desk and in the floating assistant: what it would set and whether it can be undone, checked against your records as they are now. **Approve** waits until that check has loaded. If the change was drafted against records that have changed since, the card says it is out of date and offers to redraft it instead. A plan of several steps, or several changes of one kind, is approved with one answer. You need only access to the assistant and to that agent, and each change still runs only if your own permissions allow it.

The conversation marks each decided change with a line ("Approved"), and deciding on the card or on [Decisions](/desk/decisions) updates both.

A long conversation eventually becomes read-only; select **Start a new conversation** to carry on. A conversation also becomes read-only when its agent is turned off, removed, set to run on its own, or no longer available to you, and the note in place of the message box says which, and whether an administrator can change it.

- Enter sends and Shift+Enter starts a new line. In **Desk settings**, **Send with** can be set to **⌘ Enter** instead. **@ mentions**, **Slash commands** and **Dictation** can each be turned off there.
- Only words can steer a reply under way. A message carrying files always waits until the reply finishes. Once a steering message has been handed to the agent, it can't be edited or taken back; stopping the reply is the only way to undo it. Waiting messages are kept with the conversation, so they survive a reload or another tab.
- You can't change the agent or the model while an agent is replying.
- A message can carry up to 5 files, each up to 25 MB: PDF, PNG, JPG, HEIC, TIFF, CSV, Excel, Word, text and email (.eml) files. Password-protected PDFs and other file types are refused before they upload. Scanning works only inside an open conversation, not from the box on Today.
- @ search lists only records you are allowed to read. Dictation isn't available in every browser; where it isn't, the microphone says so.
- A conversation keeps up to 12 pinned facts of up to 200 characters each. Compacting keeps pinned facts, pending approvals and, where it can, the latest turns in full. A conversation set to compact itself does so at 85% full.
- An agent can wait on up to ten things in one conversation. A wait gives up after a day unless the agent sets a longer limit (at most a week); either way the agent picks the task back up and is told it ran out. If the thing has already happened, the agent acts on it right away.
- A case is in one of four states, shown on its card, in its menu and beside the conversation in the rail. **Working**: nothing is put off. Waiting: on something such as **Waiting on the customer**, **Waiting on the carrier** or **Waiting on a reply**. **Snoozed**: put away until a time, the next appointment or the ETA. **Settled**: the record is closed. A shipment settles once it is invoiced or canceled, not when it delivers, because the ready-to-bill work comes after delivery; an invoice once it is paid or voided; a dispute once it is resolved or withdrawn.
- A case's state is worked out each time you look, so a payment recorded elsewhere settles its invoice's case on its own. Snoozed and settled cases sit on their own **Snoozed** and **Settled** shelves in the rail; a snoozed case comes back on time, or sooner when something the agent was waiting on arrives. A conversation started from a shipment, invoice or dispute is a case from the start.
- Which steps a case checklist shows, their order, and whether each is required or optional, are set in **Settings** › **Case checklists** on the Desk; a customer can have its own. A ticked step belongs to the record, so everyone working a case about the same shipment or invoice sees it.
- If a record the agent was working with changes elsewhere during its turn, a note headed **Changed while the agent was working** lists what changed, and the agent reads those records again before relying on them.
- An agent can hand a task only to the agents chosen for it in [AI control](/admin/agent-control), and only ones you may use. A case step you hand to another agent with its button needs only that you may use that agent. That agent works as you, its changes still wait for your approval, and it cannot pass the work on again. Hand-off goes to another agent, not to another person.
- Anything an agent produces lands in the workspace: tables, reports, record cards, documents, email drafts, plans, read documents, decisions and billing items. With **Open new artifacts** set to **Right away**, a new item opens as it arrives; otherwise a dot on **Workspace** marks something new.
- Billing actions on a billing item are your own and need billing queue permissions. The agent never approves or posts an invoice on its own.
- Desk settings and chapters are kept in this browser; they do not follow you to another browser or computer.
- Changing what an agent can do needs permission to manage agents.
