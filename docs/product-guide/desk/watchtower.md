---
path: /desk/watchtower
aliases: [alerts feed, exceptions feed, notifications, things to look at, operational alerts, AI monitoring]
related:
  - /desk
  - /desk/decisions
  - /inbox
  - /shipment-management/service-failures
---

## What it's for
Watchtower is a live feed of things across the operation that are worth a look: failed agent runs, service failures, expiring credentials, quarantined EDI, inbound mail waiting on a person, and similar events. Each item shows its severity, what kind of event it is, a short summary and how long ago it happened, and new items since your last visit are marked. From an item you can open the record, ask an agent about it, or hand it to the agents that watch that kind of event.

## Tasks

### Review what needs attention
Keywords: check alerts, what went wrong, exceptions today
1. Open [Watchtower](/desk/watchtower).
2. Filter by severity (**Critical**, **Warning**, **Info**) and by kind of event with the chips across the top.
3. Hover an item and select the arrow to open the record it is about.

### Ask an agent about an item
Keywords: investigate alert, explain this exception
1. Open [Watchtower](/desk/watchtower) and hover the item.
2. Select **Ask about this**. A conversation opens on the [Desk](/desk) about that item.

### Hand an item to an agent
Keywords: delegate to AI, let the agent handle it, assign to agent
1. Open [Watchtower](/desk/watchtower) and hover the item.
2. Select **Hand off**. The agents that subscribe to that kind of event pick it up; if none does, Watchtower says so.

### Clear an item from the feed
Keywords: dismiss alert, hide item
1. Open [Watchtower](/desk/watchtower) and hover the item.
2. Select the dismiss (X) button.

## Notes
Needs read access to the assistant and to Watchtower. **Ask about this** appears only when an agent is available to ask, and **Hand off** only for items an agent can take.
