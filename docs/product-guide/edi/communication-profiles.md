---
path: /edi/communication-profiles
aliases: [transport profile, AS2 setup, SFTP setup, VAN mailbox, EDI connection settings, ISA IDs, EDI credentials, EDI endpoint, AS2 inbound URL, AS2 receiving URL]
related:
  - /edi/partners
  - /edi/messages
  - /edi/inbound-files
---

## What it's for
Communication profiles hold how Trenova exchanges documents with a trading partner: the transport method (Internal, AS2, SFTP or VAN), the endpoint and its credentials, whether it reaches the partner's test or production system, and delivery retry limits. The interchange IDs, group codes, X12 version and acknowledgments live in the partner's Lading project. A partner's **Default transport profile** points at one of these.

EDI implementation teams use this page when onboarding a partner or when a partner changes its endpoint, certificates or IDs.

## Tasks

### Create a communication profile
Keywords: new transport profile, add AS2 endpoint, add SFTP connection
1. Open [Communication profiles](/edi/communication-profiles).
2. Select New EDI communication profile at the top of the table.
3. On the **Overview** tab, enter a **Name**, choose the **Method** and **Status**, and pick the **Partner** this profile delivers for.
4. On the **Transport** tab, fill in the connection details for the method: for AS2 the **Local AS2 ID**, **Partner AS2 ID**, **Endpoint URL** and certificates; for SFTP the **Host**, **Port**, **Username**, **Authentication** and directories; for VAN the mailbox and gateway details. Set **Max attempts** and the backoff under **Delivery retry** if the defaults do not suit.
5. On the **Envelope** tab, pick the **Environment**.
6. On the **Secrets** tab, enter the password or private key the method needs. For AS2 you can keep decryption apart from signing: add an encryption certificate on the **Transport** tab, give it to your partner to encrypt to, and enter its decryption key here. **Test connection** warns when one key does both.
7. Select **Save** (or **Save & close**).

### Give a partner your AS2 inbound URL
Keywords: AS2 receiving URL, where partner sends AS2, our AS2 endpoint, rotate AS2 URL
1. Open [Communication profiles](/edi/communication-profiles) and select the AS2 profile's row.
2. On the **Transport** tab, find **Inbound URL** and select **Copy**. Each AS2 profile has its own URL, and Trenova accepts the partner's messages only at the URL of their profile, so send this one to the partner with your AS2 ID and certificate.
3. If the partner returns receipts asynchronously, enter the same URL as the **Async MDN return URL**.
4. To retire a URL that has been shared too widely, select **Rotate URL** and confirm. The old URL stops working at once; give the partner the new one.

### Update credentials or endpoint details
Keywords: rotate password, new certificate, change SFTP host
1. Open [Communication profiles](/edi/communication-profiles) and select the profile's row.
2. Change the fields on the **Transport** or **Envelope** tab.
3. On the **Secrets** tab, enter only the values you want to replace. Leaving a secret blank keeps the saved value; **Saved secrets** shows which ones are already stored.
4. Select **Save** (or **Save & close**).

### Test a partner connection
Keywords: verify AS2, check SFTP login, connectivity test
1. Open [Communication profiles](/edi/communication-profiles) and select the profile's row.
2. Save any changes first; the test is unavailable while the form has unsaved edits.
3. Select **Test connection** at the top of the panel. A message reports whether certificates, credentials and endpoint reachability passed, passed with warnings, or failed.

## Notes
Viewing profiles needs read access to EDI; creating needs create access and editing needs update access to EDI. **Test connection** is not offered for Internal profiles. Rotating an AS2 inbound URL needs update access to EDI. An AS2 ID pair can be used by only one active profile in your organization. Internal profiles route through an accepted organization connection, chosen under **Connection** and **Connected organization**, and do not use X12 envelope IDs or stored credentials. Deliveries that use up all retry attempts are dead-lettered and show on [Messages](/edi/messages).
