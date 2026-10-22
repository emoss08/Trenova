---
path: /reports/builder
title: Report builder
aliases: [build report, custom report designer, query builder, pivot table, report editor, locked field, unavailable field, report access]
related:
  - /reports
  - /reports/explore
  - /reports/runs
covers:
  - /reports/builder/:definitionId
---

## What it's for
The report builder is where a custom report is designed. You pick what the report is about (shipments, customers, invoices and so on), add fields from the catalog on the left as columns, and see a **Live preview** of the first rows update as you build. The inspector on the right sets **Columns**, **Filters**, **Charts**, **Options** (sorting, row limit, totals and pivot) and **Params** (values people fill in when they run it). Analysts, managers and back-office staff use it to build the reports the rest of the team runs.

## Tasks

### Build a new report
Keywords: create report, design report, add columns
1. Open the [Report library](/reports) and select **New report**, or open the [report builder](/reports/builder) directly.
2. Under **What is this report about?**, pick the entity.
3. Search the field catalog (**Search fields...**) and add fields. On **Columns**, set each column's **Kind** (**Dimension** or **Measure**), its **Aggregation** for measures, a **Bucket** for dates, and a **Column name**. Select **Calculation** to add a computed column.
4. On **Filters**, add **Row filters** and **Measure filters**; use **Condition** and **Group** with **Match all** or **Match any**.
5. On **Options**, set the **Sort**, **Row limit** and **Total row**, and a **Pivot** if needed.
6. Select **Save**, then enter the **Name**, **Description**, **Category**, **Default format**, **Visibility**, **Status** and **Tags**, and select **Save report**.

### Add a chart
Keywords: graph, bar chart, line chart, map, visualize report
1. Open the report in the [report builder](/reports/builder) and add at least one measure column.
2. On **Charts**, select **Chart**, choose its **Type**, set the **Horizontal axis** or **Group by** column and the **Measures**, and adjust options such as **Stacked**, **Show values** and **Legend**.
3. Select **Save**.

### Let people choose values when they run it
Keywords: report parameters, prompt for date range, runtime filter
1. Open the report in the [report builder](/reports/builder) and go to **Params**.
2. Select **Parameter**, set its **Label**, **Type**, **Default**, whether it is **Required** or **Multiple**, and any **Allowed values**.
3. On **Filters**, point a condition at the parameter instead of a **Fixed value**, then select **Save**.

### Edit an existing report
Keywords: change report, fix report
1. Open the [Report library](/reports), open the report card's actions menu, and select **Edit in builder**.
2. Make the changes and select **Save**, then **Save report** to keep them. Saving creates a new revision.
3. Select **Run** to run the saved report.

### Understand locked and reference-only fields
Keywords: locked field, greyed out field, field unavailable, why can't I add this field, reference only, field group, sensitivity level
1. Open the report in the [report builder](/reports/builder) and look at the field catalog on the
   left. A field with a lock that you can still add belongs to a protected field group; its badge
   names the group, such as **Personal** or **Margins**.
2. A field you cannot add is greyed out, with a lock and its reason beside it: "Needs the
   Personal field group" when your role does not hold that group, "Unavailable at your role's
   sensitivity level" when it ranks above your role's sensitivity level, or "Needs Customers
   read" (with that kind of record named) when it needs read on that kind.
3. A related record you can join to but cannot read is marked **Reference only**. Expanding it
   lists only the fields other records show of it, under a note such as "Reference only — the
   rest needs Customers read".
4. A related record you cannot join to at all is listed with a lock and "Needs Customers read"
   (naming the kind it needs) and cannot be expanded.
5. To use a locked field, ask an administrator to grant the read, field group or sensitivity
   level on your role in [Roles](/admin/roles).

### Fix a report that uses fields you can no longer use
Keywords: report unavailable fields, report refused, cannot run report, lost access, remove fields
1. Open the report in the [report builder](/reports/builder). When it uses fields your role cannot
   reach, a warning at the top lists each one and why, and says the preview and runs are refused
   until they are removed or your access changes.
2. Select **Remove unavailable fields** to take those columns, filters and the pivot out of the
   report, then select **Save** and **Save report**. Or ask an administrator for the access instead.
3. When the report is built on a kind of record your role cannot read at all, the builder shows
   **You cannot open this report's data** instead of the report. Ask an administrator for read
   access to that kind, or select **New report** to start a different one.

## Notes
Needs create permission for reports. **Save** stays disabled until the report has at least one column, and **Run** works only for a report whose status is **Active**. A report flagged **Needs attention** has a problem (for example a field that no longer exists) described at the top of the builder.

The entity picker lists only the kinds of record your role can read. The field catalog decides
each field by the same rule as the preview and runs, so a field you can add always compiles, and a
saved report someone with a wider role built may name fields that are locked for you.
