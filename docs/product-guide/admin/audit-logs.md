---
path: /admin/audit-logs
aliases: [audit entries, audit log, audit trail, activity log, change history, who changed what, change log, legal hold, audit export, security alerts]
related:
  - /admin/users
  - /admin/roles
  - /organization/data-retention
---

## What it's for
The audit log is the record of activity across your organization: who created, changed, read or
deleted what, when, and how the change arrived. Each entry shows the **Resource ID**, the
**Record** it was about, a **Description**, the kind of **Resource**, the **Action**, the
**Timestamp**, **Who** did it and the **Channel** it came through (the web app, an API key, an
agent, a SCIM directory or a background job). You see the entries for the kinds of record you can
read; security entries need security audit access.

Opening an entry shows its **Entry details** (event ID, action code, channel, agent run, how long
it is **Kept as**, IP address, **Request ID**, **Trace ID** and user agent), the **Changes** made
field by field with the previous and current value, the other entries written by the **Same
request**, and the **Metadata** behind it. A created record keeps its **Current state** and a
deleted one its **Previous state**. Entries are written with the change itself, so they appear at
once, and every hour they are sealed so a changed or removed entry is found. The page is
read-only.

## Tasks

### Find who changed a record
Keywords: record history, who edited, trace change, investigate change
1. Open [Audit log](/admin/audit-logs).
2. Select **Filter** and filter **Resource ID** with the record's ID, or **Record** with its
   number or name, or filter **Resource** to the kind of record.
3. Narrow further by **Action**, **Who** or **Timestamp** if needed, and sort by **Timestamp**.

### Read the details of an entry
Keywords: before and after, field changes, view change, previous value, field history
1. Open [Audit log](/admin/audit-logs).
2. Select an entry's row to open its details.
3. Read **Changes** for each field's **Previous value** and **Current value**. A value marked
   **Sensitive** was left out of the log. Select **History** beside a field to see every earlier
   change to it.
4. Read **Same request** to see everything else the same request changed.

### Review what a person did
Keywords: user activity, actions by user
1. Open [Audit log](/admin/audit-logs).
2. Select **Filter** and filter **Who** to the person, and sort by **Timestamp** to see their
   activity in order.

### Export the audit log
Keywords: download audit log, CSV export, JSON export
1. Open [Audit log](/admin/audit-logs) and select **Exports**.
2. Select **New export**, choose the **Format**, the **First day** and the **Last day**, and
   whether to **Include security entries**.
3. Select **Export**. The file is written in the background; select **Download** when it is ready.
   Only you can download it, and the download is recorded.

### Keep entries for litigation
Keywords: legal hold, preserve audit entries, litigation hold
1. Open [Audit log](/admin/audit-logs) and select **Legal holds**.
2. Select **New legal hold**, give it a **Name** and a **Reason**, and to hold less than every
   entry give the **Kind of record** and, for one record, its **Record ID**.
3. Select **Place hold**. To end it, open the hold's menu, select **Release** and give a reason.

### Choose which security alerts you receive
Keywords: security alerts, notify on role change, bulk delete alert
1. Open [Audit log](/admin/audit-logs) and select **Alerts**.
2. Turn each alert on or off. For the alerts on many deletes or many exports, set
   how many entries in how many minutes raise it.

## Notes
Viewing the page needs read access to audit logs; exports need export access; legal holds need
audit legal hold access; alerts and the seal status need security audit access. How long each
kind of entry is kept is set on [Data retention](/organization/data-retention). Sign-ins are on
**Sign-in activity**, and what agents did on **AI audit trail**.
