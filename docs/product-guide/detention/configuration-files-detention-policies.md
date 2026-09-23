---
path: /detention/configuration-files/detention-policies
aliases: [detention terms, detention rules, free time, dwell charges, waiting time charges, demurrage, layover rules, detention contract]
related:
  - /detention/desk
  - /detention/intelligence
  - /billing/configuration-files/accessorial-charges
  - /billing/configuration-files/customers
---

## What it's for
Detention policies encode each contract's detention terms: how much free time a driver gets at a stop, when the clock starts, how waiting time is rounded and priced, what caps apply, and whether the customer must be given notice before detention can be billed. The detention desk and the charges it produces follow whichever policy fits the stop.

Billing and operations staff set policies up here, and can try them out before they touch a shipment: the **Live preview** tab prices worked examples and the **Backtest** tab replays the terms against past stops. The table shows each policy's **Status**, **Name**, **Code**, **Free time**, **Rate source**, **Specificity** and **Description**.

## Tasks

### Add a detention policy
Keywords: new detention policy, set free time, detention rate
1. Open [Detention policies](/detention/configuration-files/detention-policies).
2. Select **New detention policy**.
3. On **Terms**, fill in **Name**, **Code**, **Status** and optionally **Priority** and **Description**.
4. Under **Scope**, decide where the policy applies: turn on **Organization default** for the fallback policy, or narrow it by **Customer**, **Facility**, **Shipment types**, **Service types**, **Commodities** and **Stop types**, with optional **Effective From** and **Expires** dates.
5. Under **The clock**, choose **Clock starts at**, what happens **If the driver arrives late**, and the **Free time** (with optional **Pickup override**, **Delivery override** and **Driver pay free time**), plus **Minimum billable**, **Rounding** and **Increment**.
6. Under **Rate**, pick the **Rate source**: **Flat accessorial rate** (then choose the **Accessorial charge**) or **Graduated tiers** (select **Add tier** for each rung, with **From**, **To**, **Rate** and **Unit**).
7. Set any **Ceilings**, the **Customer notice** rules and the **Approval** thresholds, then select **Save**.

### Preview what a policy would charge
Keywords: test detention policy, worked example, simulate detention
1. Open [Detention policies](/detention/configuration-files/detention-policies) and open the policy, or start a new one.
2. Open the **Live preview** tab to see worked examples priced by the same engine that bills real shipments, including gross, driver pay and net margin.
3. Change fields on **Terms** and return to **Live preview** to see the effect.

### Backtest a policy against past stops
Keywords: replay detention, what would we have billed, revenue impact
1. Open [Detention policies](/detention/configuration-files/detention-policies) and open the policy.
2. Open the **Backtest** tab, choose the **Backtest window** and select **Run backtest**.
3. Read the **Revenue change**, how many stops matched and would bill, and the **Biggest movers**. Turn on **Assume notices were sent on time** to see what the terms are worth if the notice process works.

### Change or retire a policy
Keywords: edit detention terms, deactivate policy
1. Open [Detention policies](/detention/configuration-files/detention-policies) and select the policy's row.
2. Change the terms, or set **Status** to **Inactive** to stop using it, then select **Save**.

## Notes
Viewing the page needs read access to detention policies; **New detention policy** only appears for people who can create them.

New policies start as drafts. When more than one policy could apply to a stop, the more specific one wins (the **Specificity** column), and **Priority** breaks ties. If **Notice requirement** is **Required to bill**, detention on a stop where the customer was not notified in time follows the **If the notice is missed** setting.
