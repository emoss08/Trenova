---
path: /admin/audit-logs
aliases: [audit entries, audit log, audit trail, activity log, change history, who changed what, change log, legal hold, audit export, security alerts, alert review, needs a review, who changed a record, who edited a customer, who changed a setting, field change history]
related:
  - /admin/users
  - /admin/roles
  - /organization/data-retention
---

## What it's for
The audit log is the record of activity across your organization: who created, changed, read or
deleted what, when, and how the change arrived. You see the entries for the kinds of record you
can read; security entries need security audit access.

The top of the page sums up a window you choose (**Today**, **7 days**, **30 days** or 90
days): Nova's summary, five figures (**Changes**, **Who acted**, **Sensitive actions**, **Made
by agents** and **People working off hours**), a chart of **When changes happened** split into
people, agents, integrations and jobs, **Who made the changes**, and what **Needs a review**. The
figures are refreshed every minute. Select **Hide overview** to fold it into one line.

Below it, the **Trail** lists every entry newest first, grouped by day; while nothing narrows it,
each day says how many entries it holds. Each row says what
happened in a sentence, with the fields it changed and their values before and after, who did it,
the record and how it arrived (**Via**). With **One row per request** on, a request that wrote many entries shows once, at its first entry, with "+2 more in this request" (or however many there are) to open the rest; while you search or filter, every match is shown. The chip beside the title says whether the entries'
seals verify; entries are sealed every hour so a changed or removed entry is found. The page is
read-only apart from marking reviews.

## Tasks

### Find who changed a record
Keywords: record history, who edited, trace change, investigate change, who changed a customer, who changed a credit limit, who changed a rate
1. Open [Audit log](/admin/audit-logs).
2. Select **Filter** and filter **Record** with the record's number or name, **Record ID** with
   its ID, or **Kind of record** to the kind of record.
3. Narrow further by **Action**, **Who** or **Time** if needed. From an open entry, select
   **Record history** to see every entry for its record.

### Read the details of an entry
Keywords: before and after, field changes, view change, previous value, field history
1. Open [Audit log](/admin/audit-logs).
2. Select an entry's row. It opens beside the trail; use the arrow keys to move to the next entry
   and Esc to close it.
3. Read **What changed** for each field's old and new value. A value marked **Sensitive** was left
   out of the log. Select **History** under a field to see every earlier value it held.
4. Read **Same request** for everything else the same request changed, **Who** for the person or
   agent, the IP address and the device, and **Raw** for the record before and after.

### Review what a person or agent did
Keywords: user activity, actions by user, agent activity
1. Open [Audit log](/admin/audit-logs).
2. Pick them under **Who made the changes**, or select **Filter** and filter **Who** to them.
3. To see only what agents did, select **By agents** above the trail.

### Look at a span of time
Keywords: what happened last night, changes at a time, off hours
1. Open [Audit log](/admin/audit-logs) and choose the window.
2. Drag across **When changes happened** over the hours or days you want. The trail narrows to
   them; clear the time filter to see everything again.

### Review a security alert
Keywords: needs a review, mark reviewed, bulk delete, role changed alert, reopen alert
1. Open [Audit log](/admin/audit-logs). Open alerts are under **Needs a review** and pinned on the
   chart.
2. Select an alert to open the entry that set it off, and read it.
3. Select **Mark reviewed**. To put it back in front of the security reviewers, open its entry
   and select **Reopen**.

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
1. Open [Audit log](/admin/audit-logs) and select **Alert rules**.
2. Turn each alert on or off. For the alerts on many deletes or many exports, set
   how many entries in how many minutes raise it.

## Notes
Viewing the page needs read access to audit logs; exports need export access; legal holds need
audit legal hold access; alerts, their reviews and the seal status need security audit access, and
marking a review needs security audit update access. Off hours are 8 PM to 6 AM in your
organization's time zone. How long each kind of entry is kept is set on
[Data retention](/organization/data-retention). Sign-ins are on **Sign-in activity**, and what
agents did on **AI audit trail**.
