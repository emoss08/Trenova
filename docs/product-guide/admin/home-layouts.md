---
path: /admin/home-layouts
aliases: [home layouts, home screen presets, dashboards by role, landing page, default home, widget layouts, role home screens]
related:
  - /admin/roles
  - /
covers:
  - /admin/home-layouts/new
  - /admin/home-layouts/:id
---

## What it's for
Home screens lets administrators author a home screen once (the widgets people land on when
they open Trenova) and assign it to the roles that should see it. Each home screen in the list
shows its name, an **Org default** or **Locked** badge where they apply, how many widgets it has,
who it reaches (its roles, a job function, or everyone without another assignment) and how many
people it reaches today.

Until an administrator authors one, everyone lands on the home screen Trenova ships for their
role. When someone matches more than one home screen, the one with the highest **Priority** wins.

## Tasks

### Create a home screen for a role
Keywords: new home screen, role dashboard, default widgets, landing layout
1. Open [Home screens](/admin/home-layouts).
2. Select **New home screen** (or **Create one** when the list is empty).
3. Enter a **Name** and optional **Description**.
4. Select **Add widget** (or **Add your first widget**), pick widgets from **Add a widget**, and
   select **Done**. Some widgets ask you to choose what they show before they land.
5. Drag widgets to rearrange them. Use a widget's options menu to change its **Width** and
   **Height**, **Configure** it, or **Remove** it.
6. Under **Assign to roles**, tick the roles that should land on it, or choose a function in **Or
   assign by job function** to reach every role tagged with that function, including roles added
   later.
7. Select **Create**. The editor stays open on the new home screen.

### Edit a home screen
Keywords: change widgets, reassign home screen, rearrange layout
1. Open [Home screens](/admin/home-layouts) and select **Edit** on the home screen (or select
   its name).
2. Change the widgets, name, roles or settings, then select **Save**. **Discard** throws away
   unsaved changes.

### Set the organization default or lock a home screen
Keywords: default home screen, stop users rearranging, lock layout, priority
1. Open the home screen from [Home screens](/admin/home-layouts).
2. Turn on **Organization default** to make it where anyone without a role assignment lands. Only
   one home screen can hold this.
3. Turn on **Lock this home screen** so the people assigned it cannot rearrange it. Anything they
   saved earlier is kept and returns if you unlock it.
4. Set **Priority** (0 to 1000) to decide which home screen wins when someone matches more than
   one, then select **Save**.

### Preview what a role will see
Keywords: preview home screen, see as role
1. Open a saved home screen from [Home screens](/admin/home-layouts).
2. Choose a role in **Preview as**. The canvas shows what a member of that role resolves to today,
   read-only and drawn with your own permissions. Unsaved edits are not included.
3. Select **Stop** to go back to editing.

### Delete a home screen
Keywords: remove home screen
1. Open [Home screens](/admin/home-layouts).
2. Select the trash button on the home screen's row and confirm with **Delete**. The people it
   reached fall back to the next home screen that matches them. This cannot be undone.

## Notes
Viewing the page needs read access to home layout presets. **New home screen** appears only for
people who can create them, and the trash button only for people who can delete them.
