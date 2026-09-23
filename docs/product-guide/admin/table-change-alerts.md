---
path: /admin/table-change-alerts
aliases: [TCA, change alerts, data change notifications, record watch, database alerts, subscriptions]
related:
  - /admin/audit-logs
---

## What it's for
Table change alerts notify people when records change. A subscription watches one table (optionally
one record) for inserts, updates or deletes, can narrow the match with conditions, and sends a
notification with a chosen priority and an optional custom title and message.

The page has two tabs. **Subscriptions** lists subscriptions with their **Name**, **Table**,
**Events**, **Status**, **Priority** and **Conditions**. **Notifications** shows your recent
notifications, newest first, with unread ones highlighted. Administrators and operations leads use
it to keep an eye on changes that matter to them, such as a shipment's status changing.

## Tasks

### Create a subscription
Keywords: new alert, watch table, notify me when, change alert
1. Open [Table change alerts](/admin/table-change-alerts).
2. On the **Subscriptions** tab, select **New subscription**.
3. Enter a **Name** and choose the **Table** to monitor.
4. Optionally enter a **Record ID** to watch a single record, and set the **Priority**.
5. Under the event types, tick Insert, Update and/or Delete. At least one is required.
6. Select **Save**.

### Only alert on specific changes
Keywords: conditions, status changed to, watched columns, filter alerts
1. Open the subscription from the **Subscriptions** tab, or start a new one.
2. Under **Conditions**, select **Add condition**, then enter the field, choose an operator (such as
   **Equals** or **Changed to**) and enter the value.
3. Set **Condition matching** to **Match ALL conditions** or **Match ANY condition**.
4. Optionally list **Watched columns** (comma-separated) so updates only trigger when those columns
   change.
5. Select **Save**.

### Customize the notification text
Keywords: alert title, alert message, template
1. Open the subscription from the **Subscriptions** tab.
2. Under **Notification**, enter a **Topic**, a **Custom title** and a **Custom message**. The title
   can use placeholders such as {{table}}, {{operation}}, {{new.field}} and {{old.field}}.
3. Leave **Custom message** empty to use the auto-generated summary.
4. Select **Save**.

### Review and clear notifications
Keywords: unread alerts, mark read
1. Open [Table change alerts](/admin/table-change-alerts).
2. Select the **Notifications** tab.
3. Hover over an unread notification and select the check mark to mark it read, or select **Mark
   all read**.

## Notes
Viewing the page needs read access to table change alerts; **New subscription** appears only for
people who can create them. Only tables the system allows for alerts appear in **Table**.
