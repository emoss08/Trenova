---
path: /admin/service-failure-reason-codes
aliases: [service failure reasons, exception reasons, late reasons, missed appointment reasons, EDI 214 reason codes, delay codes]
related:
  - /shipment-management/service-failures
  - /admin/shipment-controls
  - /edi/partners
---

## What it's for
Service failure reason codes are the list of reasons people and the system pick from when a
pickup or delivery goes wrong (a late delivery, a missed appointment, a weather delay). Each code
has a stable **Reason code** identifier used for reporting and integrations, a **Display name**
shown to operations and billing users, a **Category** (such as **Carrier**, **Customer**,
**Facility**, **Weather** or **Driver**), and which stops it **Applies To**. Each code can also
carry EDI defaults: the X12 status, reason and exception codes and a default note used when a
service failure is sent on an EDI 214.

The table shows **Active**, **Code**, **Label**, **Category**, **Applies To**, **Description**,
**X12 status**, **X12 reason**, **Sort** and **Created**. Administrators and operations managers
maintain it.

## Tasks

### Add a reason code
Keywords: new reason code, add exception reason, late delivery code
1. Open [Service failure reason codes](/admin/service-failure-reason-codes).
2. Select **New service failure reason code**.
3. Leave **Active** on so the reason can be used for detected and manual service failures.
4. Enter the **Reason code** (for example LATE_DELIVERY) and the **Display name**.
5. Choose a **Category** and what the code **Applies To**: **Pickup**, **Delivery**, **Pickup &
   delivery** or **All stops**. Optionally describe when to use it in **Details**.
6. Under **EDI defaults**, optionally enter the **Status code**, **Reason code** and **Exception
   code** (up to 3 characters each) and a **Default note** for EDI 214 messages.
7. Under **Ordering**, set the **Sort order** (lower values appear first) and select **Save**.

### Change a reason code
Keywords: edit reason code, update EDI defaults
1. Open [Service failure reason codes](/admin/service-failure-reason-codes).
2. Select the code's row (or right-click it and choose **Edit**).
3. Change the fields and select **Save**, or choose **Save & close** from the save button's menu.

### Archive or reactivate a reason code
Keywords: retire reason, disable reason code, deactivate, restore reason
1. Open [Service failure reason codes](/admin/service-failure-reason-codes).
2. Right-click an active code and choose **Archive** to stop it being offered.
3. To bring an archived code back, right-click it and choose **Reactivate**.

### Find a reason code
Keywords: search reason codes, filter by category
1. Open [Service failure reason codes](/admin/service-failure-reason-codes).
2. Use the search box, or select **Filter** to narrow by columns such as **Active**,
   **Category** or **Applies To**. Most columns can be sorted.

## Notes
Viewing the page needs read access to service failure reason codes; **New service failure reason
code** appears only for people who can create them. Archiving makes a code inactive rather than
deleting it; an inactive code is not available for detected or manual service failures.
