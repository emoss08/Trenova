---
path: /hr/scheduling
aliases: [rota, roster, shift schedule, work schedule, shift patterns, who is working, shift swaps]
related:
  - /hr/workers
  - /hr/time-attendance
  - /hr/holidays
  - /hr/my-team
---

## What it's for
Scheduling shows who is expected to work on which days. The **Rota** board has a row per worker
and a cell per day, built fresh every time from their shift pattern, approved time off, open
leave and the loads dispatch has already assigned, so it never goes stale. Figures at the top show
**On the board**, **Cover today**, **Conflicts** (people rostered on a day they cannot work) and
**Swaps waiting on you**, and **Needs a look** lists the rows that need attention.

The **Shifts** tab holds the shift patterns (days, start time, length and rotation), and the
**Swaps** tab holds shift swaps drivers have asked for. Operations managers and HR use the page to
plan cover and settle swaps.

## Tasks

### Check who is working
Keywords: who is on today, cover, staffing, rota view
1. Open [Scheduling](/hr/scheduling) on the **Rota** tab.
2. Use the arrows or **Today** to move between weeks, and choose **Week**, **2 weeks** or
   **4 weeks** to see more at once.
3. Search by name, shift or terminal, pick a fleet (**All fleets** by default), or turn on
   **My team** to see only the people you answer for.
4. Point at a day to see details such as loads already assigned.

### Add or change a shift pattern
Keywords: new shift, create shift template, rotation, night shift
1. Open [Scheduling](/hr/scheduling) and select the **Shifts** tab.
2. Select **Add a shift**, or **Edit** on an existing shift.
3. Fill in **Code** and **Name**, tick the **Working days**, and set **Starts at**,
   **Length (hours)** and **Rotation (weeks)**. Optionally set a **Colour** and **Description**.
4. Select **Add shift** (or **Save**). Set **Status** to **Retired** to stop using a shift.

### Put a worker on a shift
Keywords: assign shift, roster a driver, move to another shift
1. Open [Workers](/hr/workers), select the worker and go to the **Schedule** tab.
2. Select **Put on a shift** (or **Move to another shift**).
3. Pick the **Shift**, the **Effective from** date and, for rotating shifts, the
   **Rotation offset (weeks)**; add **Notes** if useful.
4. Select **Assign**. Their row appears on the board.

### Approve or reject a shift swap
Keywords: swap request, trade shifts, shift change request
1. Open [Scheduling](/hr/scheduling) and select the **Swaps** tab.
2. Keep **Waiting** selected to see swaps that need a decision, or choose **Everything**.
3. On a swap the colleague has accepted, select **Approve** or **Reject**. Swaps still
   **Waiting on the colleague** cannot be decided yet.

## Notes
- The page appears only for organizations that run their own assets (asset operations) and needs
  read access to worker schedules. The **Shifts** and **Swaps** tabs need read access to shift
  templates and shift swaps; adding, editing, approving and rejecting each need their own access.
- The board cannot be edited directly: change a worker's shift, time off or leave and the board
  follows.
