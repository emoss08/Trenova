---
path: /organization/data-retention
aliases: [retention policy, purge settings, audit log retention, EDI payload retention, data purge, AI audit trail retention, AI feedback retention]
related:
  - /admin/audit-logs
  - /admin/agent-control
  - /edi/messages
  - /edi/inbound-files
---

## What it's for
Data retention sets how long the organization keeps audit entries, raw EDI payloads and the AI
records before the nightly purge jobs remove them. The page is one table, grouped under **Audit**,
**EDI** and **AI**. Each row names the data, how long it is **Kept for** in days, the date it
**Removes before** today, and what happens **Then**: deleted, or for EDI, the raw content blanked
while the record stays.

Administrators use it to balance storage and privacy against how far back people need to look at
audit history or replay EDI traffic.

## Tasks

### Change how long audit entries are kept
Keywords: audit history, audit log retention, keep audit logs
1. Open [Data retention](/organization/data-retention).
2. Enter the number of days for **Audit entries**. It must be at least 1.
3. Select **Save changes**.

### Change how long the AI audit trail is kept
Keywords: AI audit retention, keep AI decisions, seven years, audit trail purge
1. Open [Data retention](/organization/data-retention).
2. Enter the number of days for **AI audit trail**. It must be at least 365; the default of 2555
   days is seven years, which is what most audits of AI decisions ask for.
3. Select **Save changes**.

### Limit how long raw EDI payloads are kept
Keywords: EDI purge, X12 retention, raw file retention, blank payloads
1. Open [Data retention](/organization/data-retention).
2. For **Inbound file contents** or **Message payloads**, turn off **Forever** and enter the
   number of days. A limit starts at 90 days.
3. Turn **Forever** back on to keep those raw payloads with no limit.
4. Select **Save changes**.

## Notes
Opening the page needs read access to the organization; people without update access see the
table read-only, and **Save changes** appears only once something has changed.

Audit entries older than the audit window are deleted by the nightly purge. Events on the
[AI audit trail](/admin/agent-control) older than its window are deleted by their own nightly
sweep, which cuts only where the trail's chain was sealed, so what is left still verifies. For EDI,
the purge blanks the raw contents but keeps the file and message records and their metadata. A
purged EDI message can no longer be replayed from [EDI messages](/edi/messages). Agent evaluation
cases kept **Forever** last until they expire or their conversation is deleted.
