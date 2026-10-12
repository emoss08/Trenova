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

The rail on the left holds the **Briefing**, the lanes (**Waiting on you**, **Assigned to me**, **Snoozed**, **Handled**, **Ignored**, **Everything**), the kinds still **Waiting**, saved **Views**, your **Sources** and **Rules**. Each message's response target counts down beside it: what is past its target is listed under **Over target**.

## Tasks

### Work through what is waiting on you
Keywords: triage email, review inbound messages, clear the queue, next message
1. Open [Triage](/triage) and select **Waiting on you**.
2. The first message opens. Press j to go to the next one and k to go back.
3. Follow the steps listed under **Actions**, then select **Mark handled** (e) or **Ignore** (#). The next waiting message opens, and **Undo** in the notice puts it back.

### Snooze, assign or act on several messages at once
Keywords: snooze email, assign email, bulk handle, select messages
1. In [Triage](/triage), right-click a message (or press and hold it on a touch screen) to open its menu: **Mark handled**, **Ignore**, **Snooze**, **Assign**, **Correct the kind**, **Copy link** and more.
2. To act on several, select them with x, or ⌘/Ctrl-click, or Shift-click for a range, then use the bar at the foot of the list: **Snooze until tomorrow**, **Assign to me**, **Ignore** or **Handled**.

### Link a message to the right shipment or customer
Keywords: wrong shipment, match email, Nova could not tell, attach email to load
1. Open the message in [Triage](/triage) and press l, or right-click it and select **Link to a record…**.
2. Choose the **Shipment** or **Customer**, fill in **Why this is the right record**, and select **Link**.

### Find a message and keep the search
Keywords: search email, find by sender, find by subject, saved search
1. Open a folder in [Triage](/triage) and type in the search box at the top of the list (press / to jump there).
2. Choose **Due first** or **Newest** to change the order.
3. Select **Save** to keep the search as a view under **Views**.

## Notes
Needs read access to inbound messages. Messages that look like phishing are held back and also land in **Waiting on you**.

Sources are set up under [Inbound mailboxes](/admin/inbound-mailboxes). A source that has stopped receiving mail shows a dot in the rail, and its folder offers **Send a test email**.

The folder, search and open message are kept in the page address, so a link to a message opens exactly that message. Old links to the inbox open Triage.
