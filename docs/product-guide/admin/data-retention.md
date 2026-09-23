---
path: /organization/data-retention
aliases: [retention policy, purge settings, audit log retention, EDI payload retention, data purge]
related:
  - /admin/audit-logs
  - /edi/messages
  - /edi/inbound-files
---

## What it's for
Data retention sets how long the organization keeps audit entries and raw EDI payloads before the
nightly purge jobs remove them. The page is one form, **Retention windows**, with three values in
days: **Audit Retention (days)**, **EDI Inbound File Retention (days)** and **EDI Message
Retention (days)**.

Administrators use it to balance storage and privacy against how far back people need to look at
audit history or replay EDI traffic.

## Tasks

### Change how long audit entries are kept
Keywords: audit history, audit log retention, keep audit logs
1. Open [Data retention](/organization/data-retention).
2. Enter the number of days in **Audit Retention (days)**. It must be at least 1.
3. Select **Save settings**.

### Limit how long raw EDI payloads are kept
Keywords: EDI purge, X12 retention, raw file retention, blank payloads
1. Open [Data retention](/organization/data-retention).
2. Enter the number of days in **EDI Inbound File Retention (days)** for raw inbound EDI file
   contents, and in **EDI Message Retention (days)** for raw X12 and payload snapshots of delivered
   and inbound messages.
3. Enter 0 in either field to keep those raw payloads forever.
4. Select **Save settings**.

## Notes
Opening the page needs read access to the organization; **Save settings** appears only for people
with update access to the organization.

Audit entries older than the audit window are deleted by the nightly purge. For EDI, the purge
blanks the raw contents but keeps the file and message records and their metadata. A purged EDI
message can no longer be replayed from [EDI messages](/edi/messages).
