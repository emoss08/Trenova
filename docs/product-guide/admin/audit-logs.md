---
path: /admin/audit-logs
aliases: [audit entries, audit log, audit trail, activity log, change history, who changed what, change log]
related:
  - /admin/users
  - /admin/roles
  - /organization/data-retention
---

## What it's for
Audit entries is the record of activity across your organization: who created, changed or
deleted what, and when. Each entry shows the **Resource ID**, a **Description**, the kind of
**Resource** (shipment, customer, user and so on), the **Action**, the **Timestamp** and the
**User** who did it.

Opening an entry shows its **Entry details** (event ID, operation, IP address, category,
correlation ID, user agent and whether it was critical), the **Changes** made field by field with
the previous and current value, and the **Metadata**, **Previous state**, **Current state** and
**Full event data** behind it. Administrators and auditors use it to answer "who changed this
record, and what did it say before?". The page is read-only.

## Tasks

### Find who changed a record
Keywords: record history, who edited, trace change, investigate change
1. Open [Audit entries](/admin/audit-logs).
2. Select **Filter** and filter **Resource ID** (contains) with the record's ID, or filter
   **Resource** to the kind of record.
3. Narrow further by **Action** or **Timestamp** if needed, and sort by **Timestamp**.

### Read the details of an entry
Keywords: before and after, field changes, view change, previous value
1. Open [Audit entries](/admin/audit-logs).
2. Select an entry's row to open its details.
3. Read **Changes** for each field's **Previous value** and **Current value**. A value marked
   **Sensitive** was left out of the log.
4. For a list or object value, select **View JSON** to expand it, or scroll to **Full event
   data** for the complete payload.

### Review what a person did
Keywords: user activity, actions by user
1. Open [Audit entries](/admin/audit-logs).
2. Use the search box or **Filter** on **Description**, **Resource** and **Action**, and sort by
   **Timestamp** to see activity in order. The **User** column shows who did each action.

### Export audit entries
Keywords: download audit log, CSV export
1. Open [Audit entries](/admin/audit-logs) and filter to the entries you need.
2. Select **Export to CSV** in the toolbar.

## Notes
Viewing the page needs read access to audit logs; exporting needs export access. Audit logs are
processed in batches, so a change you just made may take a few moments to appear; refresh the page
after a brief wait if it is missing.
