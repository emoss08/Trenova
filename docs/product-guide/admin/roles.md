---
path: /admin/roles
aliases: [permissions, access control, security roles, user groups, RBAC, permission sets]
related:
  - /admin/users
  - /admin/home-layouts
  - /admin/audit-logs
  - /admin/agent-control
covers:
  - /admin/roles/new
  - /admin/roles/:id/edit
---

## What it's for
Roles decide what people in your organization can see and do. Each role holds a set of
permissions: for every resource (shipments, customers, invoices and so on) which operations the
role may perform, such as read, create or update, and how much data it reaches (**Own data
only**, **Organization** or **All data**). A role also sets a **Max sensitivity** level (**Public**,
**Internal**, **Restricted** or **Confidential**) that caps which sensitive fields its members can
see.

The table lists each role's **Name**, **Description**, **Max sensitivity** and **Created At**.
Administrators build roles here and then assign them to people on
[Users](/admin/users).

## Tasks

### Create a role
Keywords: new role, add permissions, permission set, build role
1. Open [Roles](/admin/roles).
2. Select **New role**. The **Create role** page opens.
3. Under **Role details**, enter a **Name**, choose the **Max sensitivity level**, optionally pick
   a **Core responsibility**, and add a **Description**.
4. Under **Permissions**, optionally start from a template button at the top right: Viewer
   (read-only access to all resources), Editor (read and update), Manager (read, create and update
   on all resources) or Custom (start empty).
5. Adjust the permissions (see below), then select **Create role** at the top of the page.

### Set a role's permissions
Keywords: grant access, remove access, data scope, operations
1. On the create or edit page for a role, go to **Permissions**.
2. Use **Search resources...** to find a resource, or expand a category.
3. Select an operation chip on a resource's row to grant or remove that operation. The box at
   the start of the row toggles **Grant all** or **Remove all** for that resource, and **All** or
   **None** on a category header does the same for every resource in the category.
4. For a granted resource, choose how much data it reaches in the scope selector: **Own data
   only**, **Organization** or **All data**.

### Edit a role
Keywords: change role, update permissions, rename role
1. Open [Roles](/admin/roles) and select the role's row. The edit page opens.
2. Change the **Name**, **Max sensitivity**, **Description** or the permissions.
3. Select **Save changes**. **Cancel** returns to the list without saving.

### Give a role access to an agent
Keywords: role agents, grant agent, AI agent access, which agents a role can use, remove agent from role
1. Open [Roles](/admin/roles) and select the role's row. The edit page opens.
2. Go to **Agents**. It lists the agents this role is granted.
3. Select **Add an agent** and pick the agent. It is saved at once, without **Save changes**.
4. To take one away, select the remove button at the end of its row; that is saved at once too.

### Find a role
Keywords: search roles
1. Open [Roles](/admin/roles).
2. Use the search box, or select **Filter** to narrow the list.

## Notes
Viewing the page needs read access to roles; **New role** needs create access. System roles show
the banner "This is a system role. Some properties may be restricted." and their name, sensitivity
and description cannot be changed. Changes to a role apply to everyone who holds it. A role's job
function (**Core responsibility**) is also what [Home screens](/admin/home-layouts) can assign by.

A role's agents matter for agents limited to specific roles in
[AI control](/admin/agent-control): people holding the role, or a role that inherits it, can use
those agents and decide what they propose. Agents open to everyone who can use the assistant are
included for every role automatically and need not be added; one that is granted anyway shows
**Open to everyone**, and the grant applies once it is limited to specific roles. System agents
cannot be granted to a role. Changing a role's agents needs update access to roles.
