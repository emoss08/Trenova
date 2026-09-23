---
path: /admin/jurisdiction-rules
aliases: [oversize limits, overweight limits, state permit rules, legal load limits, superload thresholds, OS/OW rules]
related:
  - /admin/jurisdiction-rule-overrides
  - /admin/hazmat-segregation-rules
  - /shipment-management/shipments
---

## What it's for
Jurisdiction rules hold each state's oversize and overweight limits: the legal width, height,
length and weight above which a load needs a permit, superload thresholds, permit lead time,
validity and fees, and travel restrictions such as daylight-only movement. The permit engine uses
active rules.

These rules are shared by every organization on the platform, not just yours. To hold your own fleet
to a stricter limit, record a carrier override on
[Jurisdiction rule overrides](/admin/jurisdiction-rule-overrides) instead. The table lists each
rule's **State**, **Verification**, **Status**, **Max width**, **Max height**, **Max length**, **Max
weight**, **Lead time** and **Verified** date.

## Tasks

### Add a state's limits
Keywords: new jurisdiction rule, add state limits, permit limits
1. Open [Jurisdiction rules](/admin/jurisdiction-rules).
2. Select **New jurisdiction rule**.
3. Under **Jurisdiction**, choose the **State** and **Status**. Only **Active** rules are used.
4. Under **Legal limits**, enter **Max width**, **Max height**, **Max length** and **Max weight**.
5. Optionally fill in **Superload thresholds**, **Permit terms** (**Lead time**, **Validity**,
   **Base fee**, **Per mile fee**) and **Travel restrictions**.
6. Under **Source**, record the **Source note** and **Source URL** the figures came from.
7. Select **Save**. A new rule starts unverified.

### Correct a state's limits
Keywords: edit jurisdiction rule, update limits
1. Open [Jurisdiction rules](/admin/jurisdiction-rules).
2. Select the state's row.
3. Change the values and select **Save**.

### Verify a rule against the state
Keywords: confirm limits, dispute rule, verification
1. Open [Jurisdiction rules](/admin/jurisdiction-rules) and select the state's row.
2. Select **Verify**.
3. Choose the **Outcome**: verified if it matches the issuing state, or disputed if the state
   contradicts it.
4. Describe **What you checked** (at least 10 characters) and optionally add a **Source URL**.
5. Select **Record verification**.

## Notes
Viewing the page needs read access to jurisdiction rules. Creating needs create access, editing
needs update access, and recording a verification needs approve access. Changing any limit clears
the rule's verification; editing only the **Source** section keeps it. Verifying does not change the
limits, so fix wrong numbers by editing the rule.
