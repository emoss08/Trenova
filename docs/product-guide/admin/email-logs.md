---
path: /organization/email-logs
aliases: [sent emails, email history, delivery log, bounced emails, email delivery status]
related:
  - /organization/email-profiles
---

## What it's for
Email logs show the organization's transactional email send and delivery history, newest first.
Each row lists the **Subject**, the **Purpose** (such as Billing or Notifications), the
**Recipients**, the **Status**, the number of send **Attempts** and when it was **Created**. When a
send has failed, the last error appears in red under the subject.

Administrators use it to confirm an email went out and to find out why one did not arrive. The page
is read-only.

## Tasks

### Check whether an email was delivered
Keywords: did the email send, delivery status, email not received
1. Open [Email logs](/organization/email-logs).
2. Find the message by its **Subject** and **Recipients**.
3. Read its **Status**: statuses include Queued, Sending, Sent, Delivered, Opened, Clicked, Failed,
   Bounced, Complained and Suppressed.

### Find out why an email failed
Keywords: email error, bounced, failed send
1. Open [Email logs](/organization/email-logs).
2. Look for rows with a Failed or Bounced **Status**.
3. Read the error shown under the **Subject**, and check **Attempts** to see how many times the send
   was tried.
4. If the problem is the sender, review the profile on
   [Email profiles](/organization/email-profiles).

## Notes
Viewing the page needs read access to email logs. The page shows the most recent messages; it has
no search or filters.
