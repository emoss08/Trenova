---
path: /admin/support-access
aliases: [giving Trenova support access, let Trenova in, support login, vendor access, staff access, impersonation, support session, send logs, share logs with support, diagnostic logs]
related:
  - /admin/audit-logs
  - /admin/users
---

## What it's for
Trenova support can look at your organization only when you let them. On a Managed Trenova
instance an administrator grants one named Trenova staff member access, says why, and chooses how
long it lasts (1, 4, 24 or 72 hours) and whether it is **Read-only** or **Full access**. Staff
never have a password here: each time they enter, Trenova's control plane signs a one-use pass for
that staff member, and your instance lets them in only while a grant for exactly that address is
live. The access ends on its own, and you can end it sooner.

Everything done under a grant is in the audit log, credited to "Trenova support" with the staff
member's address, and listed under the grant on this page. Administrators who can see this page
are notified each time a support session starts, and every page of a support session shows a bar
naming the staff member and when the session ends.

Full access is an administrator's reach with exceptions that never change: support may not
manage users, roles or sign-in settings, create API keys, grant or end support access, see or
change stored credentials (integrations, AI provider keys, single sign-on, mailboxes), change
bank or factoring remittance details, or export data in bulk.

On a Managed Trenova instance this page also sends the instance's logs to Trenova support when
they are looking into a problem. You choose how far back the logs reach, and Trenova keeps them
for 30 days. Staff can also pull logs themselves, but only while a grant names their address, and
each pull is written to your audit log.

## Tasks

### Send logs to Trenova support
Keywords: send logs, share logs, diagnostic logs, logs for a ticket
1. Open [Trenova support access](/admin/support-access).
2. Under **Send logs to Trenova support**, choose **Last hour**, **Last 24 hours** or
   **Last 7 days**.
3. Select **Send logs**. The request shows **Requested**, then **Collecting**, then sent with the
   date, without reloading the page. You are notified when it is sent or if it fails, with the
   reason.

### Grant Trenova support access
Keywords: let support in, give access, open a support session, allow staff
1. Open [Trenova support access](/admin/support-access).
2. Select **Grant access**.
3. Enter the **Staff member's email** Trenova gave you. Only a Trenova staff address is accepted.
4. Enter the **Reason**, such as the ticket you opened.
5. Choose how long under **Ends after**, and **Read-only** or **Full access** under **Access**.
6. Select **Grant access**. Tell the staff member they can enter now.

### See what support did
Keywords: support activity, audit support, what did Trenova change
1. Open [Trenova support access](/admin/support-access).
2. On the grant, select **Show what support did**.
3. Select an entry to open it in the [audit log](/admin/audit-logs).

### End support access now
Keywords: revoke support, remove access, stop support session
1. Open [Trenova support access](/admin/support-access).
2. On a live grant, select **End access**.
3. Enter the **Reason** and select **End access**. Any session the staff member has open ends at
   once.

## Notes
- Needs the Trenova Support Access permission: View to see this page, Grant to grant, Revoke to
  end. It is never given to an API key or an agent, and a support session cannot use it.
- A staff member has at most one live grant at a time, and at most five grants are live at once.
- Self-hosted installations have no support access: nobody at Trenova can enter them, and they
  cannot send logs to Trenova.
- Sending logs needs the Organization Update permission, as Your plan does. The logs cover the
  whole instance, every organization on it, so only one request can be under way at a time, and
  the instance can send logs at most five times in 24 hours. A request the instance does not
  finish in time is marked failed and says why.
