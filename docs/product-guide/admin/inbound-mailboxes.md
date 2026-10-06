---
path: /admin/inbound-mailboxes
aliases: [inbound email, email intake, mailbox addresses, tender inbox, POD inbox, inbound webhook, Resend inbound, Postmark inbound, Resend API key, missing email body]
related:
  - /inbox
  - /admin/integrations
  - /admin/agent-control
---

## What it's for
Inbound mailboxes are the email addresses Trenova listens on, such as one for tenders, one for
PODs and one for carrier invoices. Each mailbox has an **Address**, the **Provider** that receives
the mail and posts it to Trenova (Resend or Postmark), a webhook URL the provider posts to, a
signing secret that proves each delivery really came from the provider, and a trust level saying
how much the desk may do with its mail without a person. Resend sends only a message's sender,
recipients and subject to the webhook, so a Resend mailbox also needs an API key: the body and
attachments are read from Resend with it after the message arrives.

Each mailbox card shows whether it is **Listening** or **Not listening**, what it is **Trusted
to** do, its **Purpose** and when it was **Created**. A mailbox without a signing secret refuses
every delivery, and a Resend mailbox without an API key accepts mail but cannot read its body or
attachments, so each message waits for a person saying why. Resend forwards everything its
account receives to every webhook, so a Resend mailbox only keeps mail addressed to its own
**Address** (including Cc and Bcc). Administrators set mailboxes up here; the mail itself is
read in the [Inbox](/inbox).

## Tasks

### Create a mailbox
Keywords: new mailbox, add inbound address, set up email intake
1. Open [Inbound mailboxes](/admin/inbound-mailboxes).
2. Select **New mailbox**.
3. Enter a **Name** and the **Address**, choose the **Provider**, and set **Status** to
   **Listening**. Optionally describe its **Purpose**.
4. Choose **How much it may do alone**: **A person reviews everything**, **The desk acts when it
   is sure enough** (then set the **Confidence bar**, between 0.5 and 1) or **The desk acts on
   everything it can**.
5. For Resend, optionally paste the **Signing secret (optional)** and the **API key (optional)**
   now. Select **Create mailbox**.
6. Copy the webhook URL shown in the next dialog and set it as the provider's inbound webhook
   (in Resend, a webhook for the email.received event), then select **I have copied it**. The
   URL cannot be shown again.

### Set or replace the signing secret
Keywords: webhook secret, whsec, Postmark credentials, mailbox refusing mail
1. Open [Inbound mailboxes](/admin/inbound-mailboxes) and find the mailbox.
2. Select **Set signing secret** (or **Replace secret**).
3. For Resend, paste the webhook's signing secret (it starts with whsec_) into **Signing secret**
   and select **Save secret**.
4. For Postmark, select **Generate credentials** (or **Replace the credentials**), copy the user
   and password into the webhook URL in Postmark, and select **I have copied them**.

### Set or replace a Resend mailbox's API key
Keywords: Resend API key, empty email body, attachments missing, re_ key, full access key
1. In Resend, create an API key with full access. A sending access key cannot read received mail.
2. Open [Inbound mailboxes](/admin/inbound-mailboxes) and find the mailbox.
3. Select **Set API key** (or **Replace API key**).
4. Paste the key (it starts with re_) into **API key** and select **Save API key**. Mail that
   arrives afterwards is read in full; messages that arrived before stay as they are.

### Rotate a mailbox's webhook URL
Keywords: new webhook URL, lost webhook URL, leaked URL
1. Open [Inbound mailboxes](/admin/inbound-mailboxes) and select **Rotate URL** on the mailbox.
2. Confirm with **Rotate**. The old URL stops working immediately and mail is refused until the
   provider is pointed at the new one.
3. Copy the new URL into the provider and select **I have copied it**.

### Change a mailbox or stop it listening
Keywords: edit mailbox, pause mailbox, change trust level
1. Open [Inbound mailboxes](/admin/inbound-mailboxes) and select **Edit** on the mailbox.
2. Change the fields, for example set **Status** to **Not listening**, and select **Save
   changes**. Changing the provider clears the signing secret and the API key, so set the new
   provider's before mail is accepted and read again.

### Read a mailbox's mail
Keywords: view emails, mailbox messages
1. Open [Inbound mailboxes](/admin/inbound-mailboxes).
2. Select **Open its mail** on the mailbox to open the [Inbox](/inbox) filtered to it.

## Notes
Viewing the page needs read access to inbound mailboxes. **New mailbox** needs create access, and
**Edit**, **Rotate URL**, the secret buttons and the API key buttons need update access. The webhook URL carries the
mailbox's token, which is stored only as a hash; if it is lost, rotate for a new one. The signing
secret and the API key are stored encrypted and never shown again.

If mail sent to a mailbox never appears in the [Inbox](/inbox), check the webhook's delivery log
in the provider: a 404 means the URL's token is wrong or the mailbox is **Not listening**, a 401
means the signing secret does not match, and a 200 marked ignored means the message was not
addressed to the mailbox's **Address**.
