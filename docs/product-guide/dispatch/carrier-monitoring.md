---
path: /dispatch/carrier-monitoring
aliases: [carrier alerts, authority changes, insurance changes, carrier watchlist, FMCSA monitoring, carrier compliance alerts]
related:
  - /dispatch/carriers
  - /dispatch/carrier-sourcing
  - /admin/integrations
---

## What it's for
Carrier monitoring collects the authority, insurance and safety changes your carrier intelligence provider detects on the carriers you watch, so compliance and carrier sales staff can act on them in one place. The strip at the top shows **Needs attention**, **Awaiting review** and **Monitored carriers** (and **Month-to-date spend** for people who manage carrier intelligence); selecting a figure jumps to the matching tab.

The page has four tabs. **Inbox** lists detected carrier changes to acknowledge and resolve. **Review** lists carriers whose latest vetting is waiting for someone to sign off. **Watchlist** lists the carriers enrolled in monitoring. **Usage** shows provider calls and spend by month.

## Tasks

### Work through new carrier changes
Keywords: carrier alerts, acknowledge change, resolve event, insurance lapsed, authority revoked
1. Open [Carrier monitoring](/dispatch/carrier-monitoring) and stay on the **Inbox** tab.
2. Use the status switch to show **Needs attention**, **Acknowledged**, **Resolved** or **All**, and search by carrier, USDOT or change. Narrow further by **Severity**, **Category**, **Source** or **Carrier**.
3. Select a change to see its details and the carrier's current standing. The j and k keys move down and up the list, and x ticks the current change.
4. Select **Acknowledge** (or press E) to show you have picked it up. To acknowledge several at once, tick them and select **Acknowledge** in the toolbar.
5. When it is handled, select **Resolve**, choose a **Resolution**, add a **Note** and confirm with **Resolve**. A note is required when you mark it a false positive.
6. Select **Open carrier** to go to the carrier's intelligence on [Carriers](/dispatch/carriers).

### Sign off a carrier's vetting
Keywords: review carrier, approve vetting, carrier review queue
1. Open [Carrier monitoring](/dispatch/carrier-monitoring) and choose the **Review** tab.
2. Search by carrier or USDOT if needed, and check each carrier's findings.
3. Select **Mark reviewed**, write a **Review note** saying what was checked and concluded, and confirm with **Mark reviewed**.

### Add or remove carriers from monitoring
Keywords: watchlist, enroll carrier, stop watching carrier
1. Open [Carrier monitoring](/dispatch/carrier-monitoring) and choose the **Watchlist** tab.
2. Tick the carriers and select **Monitor** or **Stop monitoring** in the bar that appears, or right-click a single row for the same choices.
3. The provider's watchlist catches up on the next monitoring sync.

### Check provider spend
Keywords: carrier intelligence cost, lookups this month, spend cap
1. Open [Carrier monitoring](/dispatch/carrier-monitoring) and choose the **Usage** tab.
2. Step between months with **Previous month** and **Next month**, or select **Back to this month**.

## Notes
The page needs read access to carrier intelligence. Acknowledging, resolving, marking reviewed and changing the watchlist need update access; the **Usage** tab and spend figure need manage access.

If no carrier intelligence provider is connected, the page offers **Open integrations** to connect one. When monitoring is paused, a notice says changes are not being detected until it resumes.

You can also turn monitoring on for a single carrier with **Continuous monitoring** on its **Intelligence** tab in [Carriers](/dispatch/carriers).
