---
path: /desk/watchtower
aliases: [watchtower, watch tower, alerts feed, exceptions feed, exception feed, notifications, things to look at, operational alerts, AI monitoring, failed agent runs, agent exceptions, quarantined EDI]
related:
  - /desk
  - /desk/decisions
  - /inbox
  - /shipment-management/service-failures
---

## What it's for
Watchtower is a live feed on the Desk of everything across the operation that is worth a look, newest first. It brings together items from many places: agent insights, proposals and plans, failed agent runs, agent exceptions, drops in agent quality, service failures, carrier changes, hours-of-service problems, weather, quarantined EDI, billing exceptions, detention, inbound messages, expiring driver credentials, moves whose coverage is at risk, truck arrivals and departures that could not be recorded on their stop, and accounting sync problems.

Each item shows a coloured dot for its severity (Critical, Warning or Info), a title, a short summary, what kind of event it is and how long ago it happened. Items that arrived since your last visit have their titles in heavier text. You only see the kinds of items whose records you are allowed to open. An item leaves the feed when the problem behind it is resolved, or when someone dismisses it. The number next to **Watchtower** in the Desk sidebar is how many open items you can see.

## Tasks

### Review what needs attention
Keywords: check alerts, what went wrong, exceptions today, open alerts
1. Open the [Desk](/desk) and select **Watchtower** in the sidebar, or go straight to [Watchtower](/desk/watchtower).
2. Read down the list. The newest items are at the top, and new ones since your last visit have their titles in heavier text.
3. Opening Watchtower marks everything in it as seen, so those items are no longer counted as new the next time you come back.

### Filter by severity or kind of event
Keywords: show only critical, filter alerts, critical alerts, by event type
1. Open [Watchtower](/desk/watchtower).
2. In the row of chips across the top, select **Critical**, **Warning** or **Info** to show only items of that severity. You can turn on more than one.
3. After the severity chips come one chip per kind of event that has open items, each with its count, such as service failures, EDI quarantined or credentials expiring. Select one or more to narrow the list to those kinds.
4. Select a chip again to turn it off. With every chip off, the feed shows everything.

### Open the record behind an item
Keywords: go to the shipment, view the source, jump to record
1. Open [Watchtower](/desk/watchtower) and hover the item.
2. Select the arrow (**Open the record**) to go to the page of the record the item is about. Items that have no record don't show the arrow.

### Ask an agent about an item
Keywords: investigate alert, explain this exception, ask AI about alert
1. Open [Watchtower](/desk/watchtower) and hover the item.
2. Select **Ask about this**. A new conversation opens on the [Desk](/desk) about the record behind the item. It uses the agent you last talked to, or your first available agent if you haven't talked to one yet.

### Hand an item to an agent
Keywords: delegate to AI, let the agent handle it, assign to agent
1. Open [Watchtower](/desk/watchtower) and hover the item.
2. Select **Hand off**. This button appears only on items about a record an agent can work on.
3. A message tells you what happened. If agents are set up to handle that kind of event, the item goes to them and the message names them. If none is, the message says **No agent subscribes to this yet** and names an agent that could take it. If no agent at all can take it, you see **No agent can take this one yet**.

### Dismiss an item
Keywords: clear alert, hide item, remove from feed, close alert
1. Open [Watchtower](/desk/watchtower) and hover the item.
2. Select the X (**Dismiss**). The item comes off Watchtower for everyone in your organization. The record behind it doesn't change: a dismissed service failure is still a service failure on its own page.

### Rate an item raised by an agent
Keywords: thumbs up, thumbs down, agent feedback, rate insight
1. Open [Watchtower](/desk/watchtower) and hover an item raised by an agent's work: an insight, proposal, plan, failed agent run or agent exception.
2. Select the thumbs up (**Helpful**) or thumbs down (**Not helpful**). The rating is saved right away, and a small box opens where you can say why. You can close it without answering.
3. Select the same thumb again to take the rating back.

## Notes
- You need read access to the assistant to open the Desk, and read access to Watchtower to open this page. Inside it, you only see each kind of item if you can read that kind of record. For example, EDI items need EDI access and credential items need driver access.
- Dismissing, handing off and marking items seen need update access to Watchtower.
- **Ask about this** appears only when you have at least one agent available. **Hand off** appears only on items about a record an agent can work on.
- When nothing is open, the page says **Nothing is asking for you**.
