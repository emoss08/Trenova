---
path: /triage
aliases: [inbox, email, mail, inbound email, monitored mailbox, shared inbox, customer emails, tenders by email, queue, nova]
related:
  - /desk
  - /desk/watchtower
  - /shipment-management/shipments
  - /admin/inbound-mailboxes
---

## What it's for
Triage is where messages from your connected sources arrive. Today a source is an inbound email address; Nova reads every message, works out what kind it is (tender, rate confirmation, proof of delivery, invoice, status request, detention dispute or other) and which shipment or customer it is about, and handles whatever a rule allows. Anything left over waits for a person with its next step already worked out. Customer service, dispatch and billing staff use it to clear the queue.

Triage opens on the **Briefing**: a ring with how many messages are left for you, Nova's line about the day (who has waited longest and roughly how long the rest will take), a timeline of today with what Nova handled on one lane and what was yours on the other, and a deck of cards that each clear a group at once (**File all**, **Draft all**, **Start**, **Link**, **Review**). Under the deck sit what Nova handled on its own (**Spot-check**), corrections this week, sources that stopped receiving mail (**Fix**) and a rule Nova suggests (**Hand it to Nova**). At the end of the day the deck becomes **Carrying over**, with a button to hand what is left to the person on call.

The rail on the left holds the **Briefing**, the lanes (**Waiting on you**, **Assigned to me**, **Waiting on them**, **Snoozed**, **Handled**, **Ignored**, **Everything**), the kinds still **Waiting**, saved **Views**, your **Sources** and **Rules**. Hide or show it with the sidebar button at the left of the header, or ⌘B / Ctrl+B. On a phone the lanes run along the top and the rest of the rail opens from the menu button beside them. Each message's response target counts down beside it: what is past its target is listed under **Over target**.

An open message shows its subject, sender and thread, the kind Nova read it as, the words, its files and what happened to it. Beside it (above it on a narrow screen, in a sheet on a phone) are its **Actions**, a numbered list of steps each finished in place, and its **Information**: the shipment and customer it is about and the facts Nova read from it. A bar along the foot of the message always holds the next step.

## Tasks

### Clear a group from the Briefing
Keywords: briefing, file all, draft all, deck, what is left
1. Open [Triage](/triage). The **Briefing** is the first thing you see.
2. Read the front card in the deck: who is in the group, the time left on each, and roughly how long it takes.
3. Select its button (**File all**, **Draft all**, **Start**, **Link** or **Review**). A finished card turns into a chip, and the next card comes forward.
4. Select a name in Nova's line or a dot on the timeline to open that message.

### Work through what is waiting on you
Keywords: triage email, review inbound messages, clear the queue, next message
1. Open [Triage](/triage) and select **Waiting on you**.
2. The first message opens. Press j to go to the next one and k to go back.
3. The bar at the foot of the message shows the next step: select **Do it**, **Have Nova do it** or **Review reply** (or press Enter), and finish each step under **Actions**.
4. When every step is done, select **Mark handled** (e). Tick **Do this on its own next time** to have Nova handle messages like it from now on. **Wait for reply** parks it under **Waiting on them** until they answer, and **Ignore** (#) sets it aside. The next waiting message opens, and **Undo** in the notice puts it back.

### Hand a message to an agent in Desk
Keywords: desk agent, ask an agent, help with a message
1. Open the message in [Triage](/triage).
2. Select **Work on it in Desk** at the foot of the actions, choose an agent and send.
3. When the agent answers, select **Yes, do it** to start its next step in Triage.

### Snooze, assign or act on several messages at once
Keywords: snooze email, assign email, bulk handle, select messages
1. In [Triage](/triage), right-click a message (or press and hold it on a touch screen) to open its menu: **Mark handled**, **Ignore**, **Snooze**, **Assign**, **Correct the kind**, **Copy link** and more.
2. To act on several, select them with x, or ⌘/Ctrl-click, or Shift-click for a range, then use the bar at the foot of the list: **Snooze until tomorrow**, **Assign to me**, **Ignore** or **Handled**.

### Link a message to the right shipment or customer
Keywords: wrong shipment, match email, Nova could not tell, attach email to load
1. Open the message in [Triage](/triage) and press l, or right-click it and select **Link to a record…**.
2. Choose the **Shipment** or **Customer**, fill in **Why this is the right record**, and select **Link**.

### Let Nova handle a kind of message on its own
Keywords: rule, automation, auto file, let nova handle
1. In [Triage](/triage), select **Rules** in the rail.
2. Select **New rule** and fill in the blanks of the sentence: which kind of message, from whom, on which source, and what Nova should do.
3. Check how many messages from the last 30 days it would have matched and how many look off, then select **Turn on**.
4. Hover a rule and select **See what it matched** to spot-check it; the switch beside a rule pauses it.

### Connect a source
Keywords: connect mailbox, add email address, forward email, new source
1. In [Triage](/triage), select **Add** beside **Sources** (or **Add a source** when there is none yet).
2. Choose the kind of source and the address Nova should watch, then follow the steps to forward mail to the address Trenova gives you.
3. Select **Send a test email** to see a message arrive.

### Find a message and keep the search
Keywords: search email, find by sender, find by subject, saved search
1. Open a folder in [Triage](/triage) and type in the search box at the top of the list (press / to jump there).
2. Choose **Due first** or **Newest** to change the order.
3. Select **Save** to keep the search as a view under **Views**.

## Notes
Needs read access to inbound messages. Messages that look like phishing are held back and also land in **Waiting on you**; the bar offers **Ignore** and **Not phishing**.

Sources are set up under [Inbound mailboxes](/admin/inbound-mailboxes). A source that has stopped receiving mail shows a dot in the rail, and its folder offers **Send a test email**. Press ? in Triage for the **Keyboard shortcuts**.

The folder, search and open message are kept in the page address, so a link to a message opens exactly that message. Old links to the inbox open Triage.
