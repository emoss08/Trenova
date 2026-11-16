---
path: /admin/plan
aliases: [subscription, what our plan includes, add-ons, which add-ons we have, plan changes, dropped add-on, plan usage, how many shipments are left, storage used, buy an add-on, add-on ending]
related:
  - /admin/organization-settings
  - /admin/support-access
---

## What it's for
On a Managed Trenova instance, Trenova sells a base plan and add-ons such as Dispatch
Intelligence or Trenova AI. This page shows an administrator how much of the plan has been used
this billing period, each add-on on the plan and each one that could be added, with what it adds,
and what every plan includes. Nothing here changes the plan: **Manage plan** and each add-on's
**Add** link open your Trenova account in a new tab, where the plan is changed.

**Usage** shows each allowance for the current period: how much is used of what the plan
includes. A bar turns amber when a metered allowance is nearly used. A limit the plan never goes
past, such as the number of organizations, stays neutral even when it is reached.

An add-on whose removal has been scheduled stays on the plan, marked with the day it ends, until
that day. **Recent changes** lists when an add-on joined or left the plan, when one was set to end,
and when a scheduled end was called off and the add-on stays. When an add-on leaves, its pages and
actions are hidden and its background work stops, but everything entered with it is kept and comes
back if you add it again. Administrators are also notified each time an add-on joins or leaves.

When an owner has set the plan to end, a notice at the top of the page says the day it ends.
Trenova stays available until then, with everything the plan includes; after that, the workspace
is taken down and a final backup is kept for the number of days Trenova sets, which the notice
says. Administrators are notified when the plan is set to end, three days before it ends, and if
it is kept.

## Tasks

### Keep a plan that is set to end
Keywords: cancel cancellation, undo cancel, plan ending, workspace going away
1. Open [Your plan](/admin/plan).
2. Select **Keep plan** in the notice at the top. Your Trenova account opens in a new tab, where
   an owner or admin keeps the plan.

### See how much of your plan you have used
Keywords: usage, shipments left, storage used, allowance, overage
1. Open [Your plan](/admin/plan).
2. Read **Usage**. The period it covers is in the panel's header.

### See what your plan includes
Keywords: subscription, which features do we have, what are we paying for
1. Open [Your plan](/admin/plan).
2. Read **On your plan** and **Included in every plan**. Select an add-on's name to see every
   feature it adds.

### Add an add-on
Keywords: upgrade, buy an add-on, more features
1. Open [Your plan](/admin/plan).
2. In **You can add**, select **Add** beside the add-on. Your Trenova account opens in a new tab
   with that add-on ready to confirm.

### See when an add-on leaves the plan
Keywords: dropped add-on, feature disappeared, missing page, add-on ending
1. Open [Your plan](/admin/plan).
2. An add-on that is ending shows the day it ends in **On your plan**. **Recent changes** lists
   when it was set to end and when an add-on left.

## Notes
Needs permission to update the organization. The page exists only on a Managed Trenova instance;
a self-hosted install has every feature. **Manage plan**, the **Add** links and **Usage** appear
once Trenova has sent them to the instance.
