---
path: /reports
aliases: [report library, reporting, analytics, custom reports, saved reports, report gallery, dashboards, scheduled reports, BI]
related:
  - /reports/runs
  - /reports/builder
  - /reports/explore
  - /insights
covers:
  - /reports/dashboards/$dashboardId
---

## What it's for
The report library is where people find, run, schedule and share reports over the organization's data. **My reports** holds the reports you and your teammates have saved, **Gallery** holds ready-made reports that come with Trenova, and **Dashboards** holds boards of report tiles. Managers, analysts and back-office staff use it to answer questions about shipments, billing and operations and to send results out on a schedule.

## Tasks

### Run a report and get the file
Keywords: export report, download report, CSV, Excel, PDF
1. Open the [Report library](/reports) and find the report on **My reports** or **Gallery** (search, **Sort By**, or filter by category or status).
2. Select **Run** on its card.
3. Fill in any values the report asks for, choose the **Format** (CSV, Excel (XLSX), PDF or JSON), and select **Run report**.
4. When it finishes, download the file from [Run history](/reports/runs).

### Start from a gallery report
Keywords: canned report, template report, copy report, customize built-in report
1. Open the [Report library](/reports) and select **Gallery**.
2. Select **Explore** to look at the results, **Run** to export them, or **Customize** to copy the report into your own reports and open it in the builder.

### Create a report
Keywords: build report, new custom report, report builder
1. Open the [Report library](/reports) and select **New report**. The [report builder](/reports/builder) opens.

### Schedule a report
Keywords: email report, recurring report, send report automatically, report subscription, alert
1. Open the [Report library](/reports) and open the report card's actions menu on **My reports**.
2. Select **Schedules**, then **Add schedule**.
3. Set the **Cadence** and **Timezone**, the **Formats**, and under **Delivery** the **Email recipients** and **In-app recipients**. Choose whether to **Show the results in the email** and **Attach the report file**.
4. Optionally, under **Alert**, turn on **Only send when there is something to report** and set the condition.
5. Leave **Enabled** on and select **Create schedule**.

### Manage a saved report
Keywords: edit report, copy report, delete report, reset report
1. Open the [Report library](/reports) and open the report card's actions menu.
2. Choose **Explore results**, **Edit in builder**, **Duplicate**, **Schedules**, **Reset to default** (for a report customized from the gallery) or **Delete**.

### Build a dashboard
Keywords: new dashboard, report tiles, KPI board
1. Open the [Report library](/reports), select **Dashboards**, then **New dashboard**. An untitled dashboard opens.
2. Select **Edit**, enter the **Dashboard name**, and select **Tile** to add a tile: choose what it **Shows**, the **Report**, and the **Chart** or **Measure**, give it a **Title**, and select **Save tile**.
3. Use **Filters** to add dashboard-wide filters, then select **Save**.
4. Later, open the dashboard from **Dashboards** and select **Edit** to change it or **Export** to export it.

### Review the Desk's workload and AI spend
Keywords: cases handled, watchtower items by person, exceptions by person, watchtower report, token usage, AI cost, workload report
1. Open the [Report library](/reports), select **Gallery** and filter by the **Desk** category.
2. Choose a report and select **Explore** or **Run**:
   - Cases Handled by Person: cases each person closed out, Watchtower items resolved, average time to resolve and how many were resolved after they were due.
   - Open Watchtower Items by Person: open Watchtower items by assignee, unassigned included, split into not yet picked up, acknowledged, in progress and snoozed, with overdue, escalated and the oldest.
   - Open Case Load by Owner: cases with something still open, by case owner, unowned included.
   - Watchtower Backlog by Kind: what is open on the Watchtower now by kind and severity.
   - Watchtower Resolution Reasons: why Watchtower items closed, by a person, an agent or the source clearing.
   - Watchtower Response Times by Week: how quickly Watchtower items were acknowledged and resolved, week by week.
   - Desk Token Usage by Person: who used the most tokens through the Desk, with the estimated cost; choose Features to widen it.
   - AI Usage and Cost by Feature and Model and AI Usage and Cost by Week: where the organization's model calls went and what they cost.
3. Set **Window (days)** where the report asks for it.

## Notes
Needs read access to reports. **New report**, **New dashboard**, **Duplicate** and customizing a gallery report need create permission; **Schedules** needs export permission on reports. Each card shows whether the report is **Shared** or **Private**, set by its **Visibility** when saved.

The Watchtower and case reports in the Desk category need the See every case permission on Watchtower, and the AI usage reports need **View** on AI usage and spend (and **Export** on it to run or schedule them, as the Watchtower reports need **Export** on Watchtower); people without them do not see those reports in the gallery. Watchtower items resolved more than 25 months ago are no longer counted, and the usage reports count a call whose provider has no price under Calls Without a Price rather than in **Estimated cost**.

A gallery report that reads data from an add-on your organization's plan does not include is shown as a locked card with **Not on your plan** and the add-on it is part of. It cannot be explored, run or customized, and the dashboard tile picker does not offer it, until the add-on is added.

A dashboard tile whose report reads an add-on the plan does not include shows a locked card naming the add-on, and a dashboard export leaves that tile out and names it on the first sheet. A scheduled run of such a report fails, and its owner is told which add-on it needs.

**Schedules** lists the schedules you set up, and on a report you own every schedule on it, because deleting the report deletes them too. A schedule runs with the permissions of the person who set it up, and only they can change, turn off or delete it; a teammate's schedule shows **Set up by a teammate**.
