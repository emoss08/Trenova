---
path: /desk/watchtower
aliases: [watchtower, watch tower, alerts feed, exceptions feed, exception feed, exception queue, notifications, things to look at, operational alerts, AI monitoring, failed agent runs, agent exceptions, quarantined EDI, deadlines]
related:
  - /desk
  - /desk/decisions
  - /triage
  - /shipment-management/service-failures
---

## What it's for
Watchtower is the Desk's queue of everything across the operation that needs someone, worked one item at a time, soonest deadline first. It brings together items from many places: agent insights, proposals and plans, failed agent runs, agent exceptions, drops in agent quality, service failures, carrier changes, hours-of-service problems, weather, quarantined EDI, billing exceptions, detention, inbound messages, expiring driver credentials, moves whose coverage is at risk, late or missed stops, missing PODs, exhausted tenders, missing check calls, truck arrivals and departures that could not be recorded on their stop, accounting sync problems and factoring.

The queue down the left is grouped by how long is left before each item's deadline: **Act now** (four hours or less, or already past), **Today**, **Coming up** (more than a day out) and **For your information** (no deadline). Each row shows the kind of item on a tile coloured by severity (red for Critical, amber for Warning), its title, the record it is about, and the countdown when the deadline is less than a day away. A blue dot marks an item you have not opened yet. When an agent is working an item, the row shows the agent instead and says it is on it, or that it needs you. Across the top, the bar shows how many items you have cleared today out of all of them, and **On it** shows a ring for each agent at work; select a ring to open that item.

The middle of the page is the item in front of you: a clock counting down to its deadline and what happens then (such as "Minneapolis dock closes in"), what kind of item it is and when it was raised, its title and summary, **At stake** (the loads, money or customers involved), a button that opens the record it is about with its key facts, the events that led to it, and either the agent working it with its progress or the agent best placed to take it.

You only see the kinds of items whose records you are allowed to open. An item leaves the queue when you mark it done, when you snooze it, or when the problem behind it clears. The number next to **Watchtower** in the Desk sidebar is how many items are in the queue.

## Tasks

### Work through the queue
Keywords: check alerts, what went wrong, exceptions today, open alerts, next item, keyboard
1. Open the [Desk](/desk) and select **Watchtower** in the sidebar, or go straight to [Watchtower](/desk/watchtower).
2. The queue opens on the item with the soonest deadline. Select any row to open it instead.
3. Use the arrows under the card, or press J (or the down arrow) for the next item and K (or the up arrow) for the previous one.
4. Opening an item marks it as seen, so its blue dot goes away.

### Mark an item done
Keywords: clear alert, resolve, close alert, finished, remove from queue
1. Open [Watchtower](/desk/watchtower) on the item.
2. Select **Done**, or press E. The card slides away, the next item takes its place and the cleared count goes up.
3. If the item changed while you had it open, it comes back as it is now. Look at it again and select **Done** if it still is.

### Snooze an item
Keywords: later, put aside, remind me tomorrow, hide until tomorrow
1. Open [Watchtower](/desk/watchtower) on the item.
2. Select **Snooze**, or press S. The item leaves the queue until eight tomorrow morning in your time zone, then comes back on its own.

### Hand an item to an agent
Keywords: delegate to AI, let the agent handle it, assign to agent
1. Open [Watchtower](/desk/watchtower) on the item.
2. Select **Hand off**, or press H, to give it to the agent the card suggests. To choose another agent, select the arrow beside **Hand off** and pick one from the list.
3. The agent starts at once. The item stays in the queue, now showing the agent and a bar of its steps: reading the record, working out what to do, and asking you to approve.
4. **Hand off** appears only on items about a record an agent can work on, and only while no agent is working the item.

### Approve what an agent drafted
Keywords: agent needs me, review agent work, approve agent change
1. Open [Watchtower](/desk/watchtower) on an item whose agent needs you; its row says so.
2. Select **Review & approve** in the agent's panel. [Decisions](/desk/decisions) opens on the change the agent proposed.
3. If the agent's run stopped before it finished, the panel says why and offers **See the run** instead.

### Ask an agent about an item
Keywords: investigate alert, explain this exception, ask AI about alert
1. Open [Watchtower](/desk/watchtower) on the item.
2. Select **Ask about this**. A new conversation opens on the [Desk](/desk) about the item: the agent sees the record behind it, its deadline, its facts and what is at stake. It uses the agent you last talked to, or your first available agent if you haven't talked to one yet.

### Open the record behind an item
Keywords: go to the shipment, view the source, jump to record
1. Open [Watchtower](/desk/watchtower) on the item.
2. Select the record button under **At stake**. It names the record and its key facts, and opens the record's page.

### See the critical items
Keywords: show only critical, critical alerts, what is on fire
1. On the Desk's **Today** page, select the link that says how many items are critical in Watchtower, or the **Critical** block of the morning briefing.
2. Watchtower opens on the first critical item in the queue.

## Notes
- You need read access to the assistant to open the Desk, and read access to Watchtower to open this page. Inside it, you only see each kind of item if you can read that kind of record. For example, EDI items need EDI access and credential items need driver access.
- Marking items done, snoozing them and handing them off need update access to Watchtower. With read access only, the bar says you can read the items but not act on them.
- A deadline the record reports itself, such as a dock closing or a certificate lapsing, is the item's deadline. Otherwise the deadline comes from the due times for each severity in dispatch controls.
- **Ask about this** needs at least one agent available to you.
- When nothing is left, the page says **All clear** and how many items you cleared today.
