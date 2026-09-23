---
path: /hr/time-attendance
aliases: [time clock, timesheets, punch in, clock in, hourly staff hours, timecards, overtime, hourly payroll]
related:
  - /hr/workers
  - /hr/scheduling
  - /hr/my-team
---

## What it's for
Time & attendance records the hours of staff paid by the clock. People clock in and out, each
week's punches build a timesheet, and a manager approves the week before payroll runs from it.
Once a week is handed over its totals are frozen, so what the manager approves is exactly what
payroll is paid from. Figures at the top show **Awaiting approval**, **On the clock now**,
**This week so far** and, for people who run payroll, **Approved, not paid**.

The page has three tabs: **Clock** (punching in and out and correcting entries), **Timesheets**
(the approval queue) and **Payroll** (sending approved weeks to payroll). Supervisors, HR and
payroll staff use it.

## Tasks

### Clock a worker in or out
Keywords: punch in, punch out, start shift, end shift
1. Open [Time & attendance](/hr/time-attendance) on the **Clock** tab.
2. Pick the **Worker**. Their card shows whether they are on the clock and their hours this week.
3. Select **Clock in** or **Clock out**.
4. Anyone still punched in is listed under **On the clock now**; select **Clock out** beside a
   name to punch them out from there. Turn on **My team** to list only your own team.

### Record or correct hours by hand
Keywords: missed punch, forgot to clock out, fix timecard, add hours
1. Open [Time & attendance](/hr/time-attendance) on the **Clock** tab and pick the **Worker**.
2. Select **Record hours**, or the edit button beside an entry under **Last two weeks** to
   correct it.
3. Set **Started**, **Finished** and any **Unpaid break (minutes)**, give the reason, and add a
   **Note** if useful.
4. Select **Record** (or **Save correction**). To delete a wrong punch, use the remove button
   beside it, give the reason in **Why** and select **Remove**.

### Approve or send back a week
Keywords: approve timesheet, reject timesheet, submit timesheet, hand over week
1. Open [Time & attendance](/hr/time-attendance) and select the **Timesheets** tab.
2. Choose **Awaiting approval**, **Open**, **Approved** or **Paid**, and optionally pick a worker
   or turn on **My team**.
3. On a handed-over week, select **Approve**, or **Send back** to return it for another look.
4. On an open week, select **Hand over** to submit it for approval.

### Run payroll from approved weeks
Keywords: export hours, payroll file, payroll CSV, void payroll run
1. Open [Time & attendance](/hr/time-attendance) and select the **Payroll** tab.
2. Under **Run payroll**, choose the period length (**Weekly**, **Fortnightly** or
   **Four weeks**) and step to the right period with the arrows.
3. Select the run button, which reads **Nothing to run** when no approved weeks are waiting.
   The weeks in the run lock to it, so the same period cannot be sent twice.
4. Under **Runs**, select **CSV** to download a run's file, or **Void** and then **Void run** to
   undo it; its weeks go back to approved.

## Notes
- The page appears only for organizations that run their own assets (asset operations) and needs
  read access to timesheets. Clocking in and out needs create access; recording, correcting and
  removing hours needs update access; approving, handing over and the **Payroll** tab each need
  their own timesheet permission.
- Every hand-entered or corrected entry keeps its reason, which shows on the timesheet.
