---
path: /admin/users
aliases: [users & roles, user accounts, staff accounts, logins, team members, add employee login, deactivate user]
related:
  - /admin/roles
  - /admin/audit-logs
  - /admin/api-keys
---

## What it's for
Users is where administrators manage the people who can sign in to Trenova and the roles they
hold. It has two tabs: **Users**, the list of accounts, and **Roles & permissions**, the same role
list as [Roles](/admin/roles). The **Users** table shows each person's name (with a dot showing
whether they are online now), **Status**, **Username**, **Last login** and **Created At**.

Opening a user shows their details, the roles assigned to them, and which organizations in the
business unit they can access.

## Tasks

### Add a user
Keywords: new user, create account, invite user, add login, send invite, set password
1. Open [Users](/admin/users) on the **Users** tab.
2. Select **New user**.
3. Set **Status**, and enter the **Full name**, **Username**, **Email address** and
   **Timezone**. Pick their **Time format** and **Language** if they differ from the defaults.
4. Choose one or more **Roles**, then select **Save**. You can only grant access you hold
   yourself.
5. Trenova emails the person a link to choose their own password and finish setting up the
   account; the confirmation names the address it went to. You never see or set their password.
   An inactive user is not sent an invitation.

### Resend an invitation or send a password reset link
Keywords: resend invite, invitation expired, forgot password, reset link, locked out
1. Open [Users](/admin/users) and select the user's row.
2. Under **Sign-in access**, select **Resend invitation** for someone who has not set up their
   account yet, or **Send password reset link** for someone who has. Any earlier link stops
   working.

### Change a user's roles
Keywords: assign role, remove role, temporary role, role expiry, grant permissions to user
1. Open [Users](/admin/users) and select the user's row (or right-click an active user and choose
   **Manage memberships**).
2. Under **Assigned roles**, select **Assign role**, choose the **Role**, and optionally set
   **Expires At** (leave it empty for a permanent assignment). Select **Assign role** to confirm.
3. To take a role away, select the remove button beside it in **Assigned roles**.

### Give a user access to other organizations
Keywords: organization access, multi-organization, business unit access
1. Open [Users](/admin/users) and select the user's row.
2. Under **Organization access**, tick each organization the user may open in the current business
   unit.
3. Select **Save organization access**.

### Edit or deactivate a user
Keywords: change email, disable user, inactivate account, bulk deactivate
1. Open [Users](/admin/users) and select the user's row.
2. Change **Status**, **Full name**, **Email address**, **Timezone**, **Time format**,
   **Language**, **Job title** or **Phone number** and select **Save**. Only the fields you
   change are saved. The **Username** cannot be changed.
3. Turn on **Account locked** to stop the person signing in, or **Require a password change at
   next sign-in** to make them choose a new password. You cannot lock or deactivate your own
   account.
4. To change several users at once, tick their rows, then use **Update status** in the bar that
   appears and pick **Active** or **Inactive**.

### Manage roles from this page
Keywords: roles tab, permissions tab
1. Open [Users](/admin/users) and select the **Roles & permissions** tab.
2. Select **New role** to build one, or select a role's row to edit its permissions. See
   [Roles](/admin/roles) for the steps.

## Notes
Viewing the page needs read access to users; **New user** needs create access and editing needs
update access. **Manage memberships** is not offered for inactive users. Role assignments and
organization access are saved as soon as you confirm them, separately from the panel's **Save**.
