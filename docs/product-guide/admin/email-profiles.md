---
path: /organization/email-profiles
aliases: [sender identity, from address, email sender, outgoing email settings, Resend, Postmark, system email, sent by Trenova]
related:
  - /organization/email-logs
---

## What it's for
Email profiles are the sender identities the organization sends email from. Each profile names an
email provider (Resend or Postmark), the sender name and verified sender email recipients see, and
an optional reply-to address.

The **Purpose assignments** panel at the top of the page routes each kind of email to a profile:
General, Billing, Reporting, Operations, Authentication and Notifications. Below it, the table
lists every profile with its **Profile**, **Sender**, **Provider**, **Status**, **Reply-To** and
**Updated** columns. Administrators use the page to set
up and maintain outgoing email.

### What Trenova sends for you
On Trenova Cloud, Trenova sends your organization's own system email for you when no profile
covers it: invitations to new users and drivers, password resets, notification emails and
digests, reminders about proposals waiting on someone, records shared with a colleague, and
scheduled reports to your own users. It goes out from Trenova's address as "<your organization>
via Trenova", and the page shows **System email is sent by Trenova on your behalf** while it does.
It only reaches your own users (and the people you invite), and it stops being used as soon as
you assign a profile to **General**.

Email to anyone outside your organization always needs your own profile and domain: invoices,
statements, payment reminders, status updates and tracking links, tender offers, rate
confirmations, detention notices, factoring letters, replies to inbound email, and test sends.
Without a profile, these are not sent and Trenova says an email profile is needed.

## Tasks

### Add an email profile
Keywords: new sender, add from address, set up email provider
1. Open [Email profiles](/organization/email-profiles).
2. Select **New email profile**.
3. Enter a **Profile name**, choose the **Status** and **Provider**, and enter the **Sender name**
   and **Sender email**.
4. Optionally enter a **Reply-to email** (blank uses the sender email) and a **Description**.
5. Select **Save**.

### Choose which profile sends each kind of email
Keywords: purpose routing, billing emails sender, route email purpose
1. Open [Email profiles](/organization/email-profiles).
2. In **Purpose assignments**, pick an active profile for each purpose, such as **Billing**.
3. Clear a purpose to remove its assignment.
4. Select **Save assignments**.

### Send a test email from a profile
Keywords: test sender, verify email profile, check provider
1. Open [Email profiles](/organization/email-profiles).
2. Right-click an active profile and choose **Send test**.
3. Enter the **Recipient email** and select **Send test**.
4. Check the result on [Email logs](/organization/email-logs).

### Edit or delete a profile
Keywords: change sender, deactivate profile, remove profile
1. Open [Email profiles](/organization/email-profiles).
2. Select a row to edit it, change the fields and select **Save**.
3. To delete one, right-click the row, choose **Delete** and confirm with **Delete**. The profile
   is also removed from any purpose it was assigned to. This cannot be undone.

## Notes
Viewing the page needs read access to email profiles. Creating, editing, deleting, sending a test
and saving purpose assignments each need the matching create, update or delete access.
Inactive profiles cannot be assigned to purposes, and **Send test** is only available for active
profiles.
