---
path: /admin/jurisdiction-rule-overrides
aliases: [jurisdiction rule overrides, oversize overrides, stricter state limits, fleet limits, permit overrides, company oversize policy]
related:
  - /admin/jurisdiction-rules
  - /admin/hazmat-segregation-rules
---

## What it's for
Carrier overrides hold your own fleet to stricter oversize limits than a state requires. Each
override names one state and can narrow its maximum width, height, length and weight, require
more permit lead time, or add daylight-only and holiday restrictions the state does not impose.
An override can only tighten a limit, never loosen one, and applies to your organization alone;
any field left blank defers to the state's own rule.

The table lists each override with its **State**, a **Narrows** column showing which limits it
tightens, and the **Reason** it exists. Safety, compliance and permitting staff maintain it.

## Tasks

### Add a carrier override
Keywords: new override, tighten state limit, stricter oversize rule, company policy
1. Open [Carrier overrides](/admin/jurisdiction-rule-overrides).
2. Select **New carrier override**.
3. Under **Jurisdiction**, choose the **State**.
4. Under **Tighter limits**, enter only the limits you want to narrow: **Max width**, **Max
   height**, **Max length** (feet) or **Max weight** (pounds). Leave the rest blank to defer to
   the state. A value above the state's limit is rejected.
5. Under **Lead time**, optionally enter a **Permit lead time** in days. It may require more
   notice than the state, never less.
6. Under **Added restrictions**, tick **Daylight only** or **Holiday restricted** if your fleet
   does not run oversize at night or on holidays in that state.
7. Under **Reason**, explain why you run tighter than the statute (at least 10 characters).
8. Select **Save**.

### Change an override
Keywords: edit override, update limit
1. Open [Carrier overrides](/admin/jurisdiction-rule-overrides).
2. Select the override's row (or right-click it and choose **Edit**).
3. Change the limits, restrictions or **Reason** and select **Save**.

### Remove an override
Keywords: delete override, revert to state limits
1. Open [Carrier overrides](/admin/jurisdiction-rule-overrides) and select the override's row.
2. Select **Remove override** at the bottom of the panel. The state reverts to its statutory
   limits for your organization.

### Find overrides
Keywords: search overrides, filter by reason
1. Open [Carrier overrides](/admin/jurisdiction-rule-overrides).
2. Use the search box, or select **Filter** to filter on **Reason**. The **State** column can be
   sorted.

## Notes
Viewing the page needs read access to carrier overrides; **New carrier override** appears only for
people who can create them. A restriction the state already imposes cannot be lifted here. The
statutory limits themselves live on [Jurisdiction rules](/admin/jurisdiction-rules).
