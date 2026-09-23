---
path: /admin/database-sessions
aliases: [DB sessions, database locks, lock contention, blocked queries, stuck transactions, kill query, terminate session]
related:
  - /admin/audit-logs
  - /admin/document-operations
---

## What it's for
Database sessions shows lock contention in the Trenova database: each card is one session that is
waiting on another. The card title reads "PID {blocked} blocked by PID {blocking}" and shows the
wait type and database, then both sides of the pair with their application, database user, state,
**Tx age** (how long the transaction has been open) and **Query age**. Ages over two minutes are
highlighted. Expanding **Queries** shows the **Blocked query** and the **Blocking query**.

The status bar at the top counts the blocked sessions; the list refreshes itself every 15 seconds.
When nothing is waiting, the page reads "All database sessions are running clean". Technical
administrators use this page when screens hang or saves time out because something is holding a
lock.

## Tasks

### Check for blocked database sessions
Keywords: slow saves, hanging page, lock wait, what is blocking
1. Open [Database sessions](/admin/database-sessions).
2. Read the count in the status bar. Select the refresh button (**Refresh**) to check again
   straight away.
3. On a card, compare **Tx age** and **Query age** on both sides, and expand **Queries** to see
   what each session is running.

### Terminate a blocking session
Keywords: kill session, release lock, cancel blocking transaction, unblock
1. Open [Database sessions](/admin/database-sessions) and find the card for the blocked pair.
2. Select **Terminate** on the card.
3. Read the confirmation, then select the confirm button (it reads "Terminate PID" followed by the
   blocking process number) to end the blocking session. Its in-flight transaction is cancelled
   and the blocked session is released.

## Notes
Viewing the page needs read access to database sessions; terminating one needs delete access to
database sessions. Terminating cancels whatever the blocking session was in the middle of, so any
unsaved work in that transaction is lost. Check the **Blocking query** first.
