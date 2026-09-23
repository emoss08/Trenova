---
path: /shipment-management/service-failures
aliases: [late pickups, late deliveries, missed appointments, on-time failures, OTP exceptions, service exceptions, lateness]
related:
  - /shipment-management/shipments
  - /admin/service-failure-reason-codes
  - /insights
---

## What it's for
Service failures lists every late or missed pickup and delivery: late pickup, late delivery, missed pickup, missed delivery, appointment missed or other. Most are detected automatically from stop times; others come in manually, over EDI or from an integration (the **Source** column says which). Each row shows the shipment and stop, how late it was, the reason, notes and when it was detected. Operations and customer service staff use it to assign a reason to each failure, review it and close it out, and to report the failure to the customer by EDI 214.

A failure moves through Open, Reviewed and Resolved, or is Voided when it should not count.

## Tasks

### Work the open service failures
Keywords: review late loads, triage failures, assign reason code
1. Open [Service failures](/shipment-management/service-failures).
2. Filter the **Status** column to Open, and by **Type** or **Source** if needed.
3. Select a failure to open it, choose the **Reason code**, and add **Operations notes** (customer-facing) and **Internal notes**.
4. Select **Save**.

### Review, resolve or void a failure
Keywords: close service failure, approve failure, dismiss failure
1. Open [Service failures](/shipment-management/service-failures).
2. Right-click the failure and choose **Review**, **Resolve** or **Void**. **Review** and **Resolve** need a reason code on the failure first; **Void** asks for a reason.

### Override the EDI codes for one failure
Keywords: EDI 214 status code, exception code, AT7 codes
1. Open the failure from [Service failures](/shipment-management/service-failures).
2. Under **EDI overrides**, set the **Status code**, **Reason code** or **Exception code** to send for this failure only.
3. Select **Save**.

### Build the EDI 214 payload
Keywords: send 214, status message to customer, copy EDI payload
1. Open [Service failures](/shipment-management/service-failures).
2. Right-click the failure and choose **Build EDI 214 payload**. The payload is copied to your clipboard, with any diagnostics listed in the confirmation.

### Check one shipment's failures
Keywords: failures for a load, evaluate shipment
1. Open the shipment from [Shipments](/shipment-management/shipments) and go to its service failures tab.
2. Select **Evaluate** to check its stops for new failures, and review, resolve or void them there.

## Notes
Needs read access to service failures. **Review** needs approve permission, **Resolve** and editing need update permission, **Void** needs archive permission, and **Build EDI 214 payload** needs export permission. Failures cannot be created from this page.

Reason codes are maintained under [Service failure reason codes](/admin/service-failure-reason-codes).
