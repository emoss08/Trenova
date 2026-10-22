---
path: /admin/roles
aliases: [permissions, access control, security roles, user groups, RBAC, permission sets, page access, view as role, role preview, what a role can see]
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

### See what a grant means
Keywords: what does this permission do, without read, reference view, which screens use this resource
1. On the create or edit page for a role, go to **Permissions** and expand a category.
2. Under each resource's name, a sentence says what the role sees: with read on the resource,
   every field of its records; without it, what other records still show of it through its
   reference view, or that it sees none of them. When some of those fields rank above the role's
   **Max sensitivity**, a second line says they stay withheld too. The sentences follow the
   permissions and the sensitivity level as you change them.
3. Select the screens link under the sentence (for example "3 screens this affects") to open
   **Screens this affects**. It lists each screen the grant changes and how the screen uses the
   resource: **Required** (the screen's route or an endpoint it calls checks it), **Reads** (the
   screen lists or opens these records itself), **Asks for more** (it asks for fields beyond the
   reference view) or **Reference view** (it shows them only through their reference view).
   Select a screen's name to open it.

### Preview a screen as the role you are editing
Keywords: test a role, try a role before saving, role preview, what would this role see
1. On the create or edit page for a role, go to **Preview a screen**, below **Permissions**.
2. Under **Screen**, choose a screen from **Choose a screen to preview**.
3. The card shows what a person holding only this role, and the roles it inherits, would need and
   see there, worked out from the form before you save: **What this screen needs** lists each
   permission the screen checks as **Held** or **Missing**, and **Records on this screen** lists
   each kind of record it shows as **Can read** or **No read**, with the linked fields that come
   back withheld. A warning says when the screen would not open, or would open empty.
4. Change the permissions or **Max sensitivity**; the preview updates a moment later.

### See what a screen needs and what you see on it
Keywords: page access, why can't I see this, missing permission, screen permissions, access panel
1. On any page, select the **Page access** button in the page bar, or press Ctrl+Alt+P. The
   **Page access** panel opens for the screen you are on.
2. **What this screen needs** lists each permission the screen's route and endpoints check, marked
   **Held** or **Missing** for you.
3. **Records on this screen** lists each kind of record the screen shows, whether you **Can read**
   it, and the linked fields that come back withheld: **Needs read** when they need read on that
   kind, **Above ceiling** when they rank above your sensitivity ceiling.

### View a screen as another role
Keywords: view as role, impersonate role, check a role's access, see what a role sees
1. Open the **Page access** panel on the screen (the page bar button, or Ctrl+Alt+P).
2. Under **View as**, pick a role. The panel shows what a person holding only that role, and the
   roles it inherits, would need and see on this screen.
3. Clear the role to go back to **You, with your own roles**.

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

The **Page access** button and Ctrl+Alt+P are offered to people with read access to roles, and
**View as** and **Preview a screen** need it too. Neither reads any record: they work out access
from the role's grants alone. Organization access policies, data scopes and row-level security can
narrow what a person holding the role reads, never widen it, so the real screen may show less.
