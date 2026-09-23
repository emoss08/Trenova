---
path: /hr/pto-policies
aliases: [time off policy, vacation policy, PTO accrual, sick leave policy, leave accrual rules, carryover rules]
related:
  - /hr/workers
  - /hr/holidays
  - /hr/leave-settings
---

## What it's for
PTO policies set how paid time off is earned and tracked. Each policy has one accrual rule per PTO
type, saying how days are earned, the most a worker can hold, how much carries over into the new
year and what happens when employment ends. It also sets whether requests need approval and
whether balances are enforced. The table shows **Status**, **Code**, **Name**, **Year**,
**Tracked types**, **Workers** on the policy and when it was **Updated**.

HR administrators maintain the policies and then assign each worker to one.

## Tasks

### Add a PTO policy
Keywords: new PTO policy, vacation accrual, set up time off rules
1. Open [PTO policies](/hr/pto-policies).
2. Select **New PTO policy**.
3. Under **General**, fill in **Code**, **Name** and **Status**. Turn on
   **Default for new hires** to enrol new workers in it from their hire date.
4. Under **Year & counting**, set the **Policy year**, **Waiting period** and whether to
   **Count weekends**.
5. Under **Accrual rules**, set each rule's **PTO type**, **Accrual** method and amount,
   **Max balance**, **Carryover cap**, **Carryover expires** and **When employment ends**. Select
   **Add type** for another PTO type, and **Add tier** to give longer-serving workers more.
6. Under **Enforcement**, choose **Requires approval**, **Enforce balance** and
   **Allow negative balance** (with a **Negative floor**).
7. Select **Save**.

### Edit a PTO policy
Keywords: change accrual rate, update carryover
1. Open [PTO policies](/hr/pto-policies) and select the policy's row.
2. Change the fields and select **Save**. Rule changes apply going forward: days already in a
   worker's ledger are not recalculated.

### Assign a policy to a worker
Keywords: enrol worker in PTO policy, change worker's PTO policy
1. Open [Workers](/hr/workers), select the worker and go to the **Time off** tab.
2. Select **Assign policy** (or **Change policy**).
3. Pick the **Policy** and the **Effective from** date, and confirm. Accruals start from that date
   on the next nightly run.

### Archive or restore a policy
Keywords: retire PTO policy, deactivate policy
1. Open [PTO policies](/hr/pto-policies).
2. Right-click the policy and choose **Archive**, or **Restore** on an archived one. To do several
   at once, tick them and use **Archive** or **Restore** in the bar that appears.

## Notes
- The page appears only for organizations that run their own assets (asset operations) and needs
  read access to PTO policies. Adding needs create access and editing needs update access;
  archiving and restoring each need their own permission.
- A policy that still has workers on it cannot be archived.
- PTO types without a rule can still be requested but are not tracked against a balance. With
  **Requires approval** off, requests are approved and booked immediately; with
  **Enforce balance** off, balances are informational and never block a request.
