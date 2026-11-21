---
path: /organization/data-retention
aliases: [driver qualification file retention, DQ file retention, retention policy, purge settings, audit log retention, legal hold retention, EDI payload retention, data purge, AI audit trail retention, AI feedback retention]
related:
  - /admin/audit-logs
  - /admin/agent-control
  - /edi/messages
  - /edi/inbound-files
---

## What it's for
Data retention sets how long the organization keeps audit entries, raw EDI payloads and the
records AI and agents leave, before the nightly purges remove them for good, and how long a
driver's qualification file is held before it is flagged for review. The page has four
sections, **Audit trail**, **EDI**, **Driver files** and **AI and agents**, with one row per kind of data: what it
is and what the purge removes on the left, the number of days it is kept on the right. Under each
period, the page names the date the next nightly purge reaches back to, so you can see what a
change will remove before you save it.

Administrators use it to balance storage and privacy against how far back people need to look at
audit history or replay EDI traffic.

## Tasks

### Change how long audit entries are kept
Keywords: audit history, audit log retention, keep audit logs, financial audit retention, seven years
1. Open [Data retention](/organization/data-retention).
2. Under **Audit trail**, enter the number of days for each kind of entry: **Reads and downloads**
   (at least 90), **Operational changes** (at least two years), **Financial changes** (at least
   seven years) and **Security entries** (at least two years). None can be kept more than 3,650
   days.
3. Check the date under each period: entries from before it are deleted in the next nightly purge.
4. Select **Save changes**.

### Change how long the AI audit trail is kept
Keywords: AI audit retention, keep AI decisions, seven years, audit trail purge
1. Open [Data retention](/organization/data-retention).
2. Under **AI and agents**, enter the number of days for **AI audit trail**. It must be at least
   365; the default of 2,555 days is seven years, which is what most audits of AI decisions ask
   for.
3. Select **Save changes**.

### Change how long driver qualification files are held
Keywords: DQ file retention, driver qualification file, 49 CFR 391.51, terminated driver files, flagged for review
1. Open [Data retention](/organization/data-retention).
2. Under **Driver files**, enter the number of days for **Driver qualification files**. It must be
   at least 1,095, three years after employment ends, the federal minimum under 49 CFR 391.51; the
   default is 1,095.
3. Check the date under the period: files of drivers who left before it are flagged for review.
4. Select **Save changes**.

Old files are only flagged for review. Nothing on this page ever deletes a driver qualification
file, so a longer period only delays the flag.

### Limit how long raw EDI payloads are kept
Keywords: EDI purge, X12 retention, raw file retention, blank payloads
1. Open [Data retention](/organization/data-retention).
2. Under **EDI**, turn on **Blank old inbound files** or **Blank old message payloads**.
3. Enter the number of days in **Blank files after** or **Blank payloads after**. A new limit
   starts at 90 days.
4. Select **Save changes**.

Turn the switch off again to keep the raw payloads with no limit. Turning it back on brings back
the last number of days it had.

### Limit how long agent evaluation cases are kept
Keywords: evaluation case retention, agent replays, eval purge
1. Open [Data retention](/organization/data-retention).
2. Under **AI and agents**, turn on **Delete old evaluation cases** and enter the number of days in
   **Delete cases after**. A new limit starts at 365 days.
3. Select **Save changes**.

## Notes
Opening the page needs read access to the organization; people without update access see the
page read-only, and **Save changes** appears only once something has changed.

Everything a purge removes is gone for good. Audit entries older than their kind's period are
deleted by the nightly purge, unless a legal hold on [Audit log](/admin/audit-logs) covers them.
The shortest periods are floors that cannot be lowered, so the trail always meets common audit
requirements. A change to these periods is itself recorded as a security entry. Events on the
[AI audit trail](/admin/agent-control) older than its period are deleted by their own nightly
sweep, which cuts only where the trail's chain was sealed, so what is left still verifies. For EDI,
the purge blanks the raw contents but keeps the file and message records and their details. A
blanked EDI message can no longer be replayed from [EDI messages](/edi/messages). Agent evaluation
cases with no limit last until they expire or their conversation is deleted.
