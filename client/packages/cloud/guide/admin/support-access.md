---
path: /admin/support-access
aliases: [trenova support, support access, let support in, vendor access, help desk access, impersonation, support session, revoke support]
related:
  - /admin/organization-settings
  - /admin/plan-usage
---

## What it's for
Support access decides whether Trenova support may come into this organization to help, for how
long, and whether it may change anything. Nobody from Trenova can open the organization unless an
administrator allows it here. It appears in Organization settings only on Trenova Cloud.

While access is allowed the page shows **Trenova support can access this organization** with the
**Access** level, **Allowed until**, **Allowed since** and any **Note**. The **Support sessions**
panel lists every session support opened: the **Staff member**, its **Status**, the **Reason** and
**Ticket** they gave, when it **Started**, and how often they used **Write access**.

## Tasks

### Let Trenova support in
Keywords: allow support, give access to support, support needs access, grant access
1. Open [Support access](/admin/support-access).
2. Choose how long access lasts in **For**: a day, three days, a week or up to fourteen days.
3. Choose the **Access**: **Read-only** lets support look without changing anything; **Read-write**
   also lets them make changes after they sign in again, thirty minutes at a time.
4. Add a **Note for Trenova support** about what to look at, then select **Allow access**.

### End Trenova support's access
Keywords: revoke support, remove access, stop support, kick support out
1. Open [Support access](/admin/support-access).
2. Select **Revoke access** and confirm. Any support session in progress ends immediately.

### Change how long or how much support can do
Keywords: extend support access, make support read-only, allow changes
1. Open [Support access](/admin/support-access) and select **Change access**.
2. Choose the new **For** and **Access**, then select **Allow access**. The new access replaces the
   old and ends any session in progress.

### See what support did
Keywords: support audit, what did support change, support history
1. Read the **Support sessions** panel on [Support access](/admin/support-access).
2. Open the [Audit log](/admin/audit-logs) for the detail. Every session start, every change to
   write access, every session end, every refused action and every change support made is recorded
   under "Trenova Support" followed by the staff member's name.

## Notes
Allowing or revoking access needs permission to update the organization; viewing the page needs read
access to the organization. Administrators who can change organization settings are notified when a
support session starts. Support sessions start read-only, last at most four hours and never outlast
the access allowed here. Even with read-write access, support cannot change users, roles, API keys,
sign-in and single sign-on settings, two-factor settings, billing or this page.
