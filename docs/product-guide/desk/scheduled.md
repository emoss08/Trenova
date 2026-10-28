---
path: /desk/scheduled
aliases: [scheduled requests, edit a schedule, change a schedule, my schedules, recurring requests, recurring reports, scheduled reports, automations, what have I scheduled, see my schedules, stop a schedule, cancel a schedule, pause a schedule, every morning report, daily report, weekly report, run a schedule now, schedule this every day, everyday, run daily]
related:
  - /desk
---

## What it's for
**Scheduled requests** lists every request you asked Desk to repeat, such as "every Monday at 8am, send me invoices more than 60 days past due", across all of your conversations. Open it from **Scheduled** in the [Desk](/desk) sidebar.

Each run asks the conversation's agent the request again, with the access you have at the time it runs, and the answer posts into the conversation the request was made in. A request runs at the time it names, on the clock it names ("at 9:30am EST", "at 8 Pacific time"), or else on your own clock, or your organization's when you have not set one.

Requests that will run are listed under **Coming up**, the soonest first. Each shows how often it runs and when it runs next, the conversation its answers post into, and when it last ran. Paused requests are listed under **Paused**.

## Tasks

### See what you have scheduled
Keywords: my schedules, what have I scheduled, recurring requests, list schedules
1. Open [Scheduled requests](/desk/scheduled).
2. **Coming up** lists the requests that will run, the soonest first. **Paused** lists the ones that will not run until you resume them.
3. The line under each request says how often it runs and when it runs next. The line below that names the conversation its answers post into and when it last ran, or **Hasn't run yet**.
4. Select the conversation's name to open it and read the answers.

### Run a scheduled request now
Keywords: run now, run it again, try my schedule, test a schedule
1. On [Scheduled requests](/desk/scheduled), select **Run now** on the request. It runs even when it is paused.
2. A note says it is running and where the answer will post. Select **Open** on the note to go to the conversation.
3. Its next scheduled run is not changed.

### Change a scheduled request
Keywords: edit schedule, change schedule time, change what it asks, reschedule, change day
1. On [Scheduled requests](/desk/scheduled), or on the request's **Scheduled** card in its conversation, open **Schedule actions** (the three dots) and select **Edit**.
2. Change **What to ask**. Under **Cadence**, choose how often it runs (every day, every weekday or every week), the time it runs at, and for every week the day it runs on. The time is on the clock the request was made on, which the editor names.
3. Select **Save**. The request's next run is worked out again from the new time.

### Pause or resume a scheduled request
Keywords: pause, stop for now, resume, turn off a schedule, turn on a schedule
1. On [Scheduled requests](/desk/scheduled), open **Schedule actions** (the three dots) on the request and select **Pause**. It moves to **Paused** and does not run until you resume it.
2. To start it again, open **Schedule actions** on it under **Paused** and select **Resume**.

### Delete a scheduled request
Keywords: delete schedule, cancel schedule, remove schedule, stop for good
1. On [Scheduled requests](/desk/scheduled), open **Schedule actions** (the three dots) on the request and select **Delete**.
2. The row asks **Delete this schedule?** Select **Delete** to remove it, or **Cancel** to keep it.
3. Answers it already posted stay in the conversation. Its card there says **Schedule deleted**.

### Schedule a new request
Keywords: new schedule, add schedule, schedule a report, recurring report, schedule this, every day, everyday, daily
1. In a conversation on [Desk](/desk), send a message that starts with when it should run, then what to ask. For example: "every weekday at 7:30am, list shipments still not dispatched for today" or "everyday at 9:30am EST, give me today's operations board". "Every day", "everyday" and "daily" all run every day; "every weekday" runs Monday to Friday; "every Monday" (or any other day) runs once a week. You can also type /schedule, then the when and the request.
2. Or ask the agent to schedule something it already answered, such as "schedule this to run every day at 9:30 AM Eastern". It writes out the request in full and proposes the schedule: select **Approve** on the proposal to keep it. Its **Scheduled** card then says **From a proposal you approved**, and the request it shows is the one the agent wrote out.
3. A **Scheduled** card appears in the conversation, and the request is listed on [Scheduled requests](/desk/scheduled).

## Notes
- Answers post only into the conversation the request was made in. They are not emailed.
- Deleting a conversation deletes the requests scheduled in it, and they stop running at once. A conversation with scheduled requests asks before it is deleted and lists the ones that will go with it.
- A conversation can keep up to 25 scheduled requests, and you can keep up to 100 in all.
- A request can run every day, every weekday or once a week. It cannot run hourly or monthly.
- Send one schedule per message. A message with a second line that starts with when to run, such as "daily: …", is turned down so you can send each one on its own.
- When you have set your own time zone and it differs from your organization's, the agent gives a schedule's time on your clock, which is the one the schedule runs on.
- A time zone named as an abbreviation, such as EST or PST, means that region's clock all year: "9:30am EST" runs at 9:30 in New York in summer too. When a request runs on another clock than yours, its card names that clock.
- An agent never schedules a request on its own: it proposes one and you approve it.
- If you lose access to a conversation's agent, its scheduled requests stop running. You can still pause or delete them.
- **Run now** needs permission to ask agents. Without it you can still pause, resume and delete your requests.
