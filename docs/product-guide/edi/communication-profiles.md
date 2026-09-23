---
path: /edi/communication-profiles
aliases: [transport profile, AS2 setup, SFTP setup, VAN mailbox, EDI connection settings, ISA IDs, EDI credentials, EDI endpoint]
related:
  - /edi/partners
  - /edi/messages
  - /edi/inbound-files
---

## What it's for
Communication profiles hold how Trenova exchanges documents with a trading partner: the transport method (Internal, AS2, SFTP or VAN), the endpoint and its credentials, the X12 envelope identifiers (ISA and GS sender and receiver IDs, X12 version, environment, acknowledgment preference) and delivery retry limits. A partner's **Default transport profile** points at one of these.

EDI implementation teams use this page when onboarding a partner or when a partner changes its endpoint, certificates or IDs.

## Tasks

### Create a communication profile
Keywords: new transport profile, add AS2 endpoint, add SFTP connection
1. Open [Communication profiles](/edi/communication-profiles).
2. Select New EDI communication profile at the top of the table.
3. On the **Overview** tab, enter a **Name**, choose the **Method** and **Status**, and pick the **Partner** this profile delivers for.
4. On the **Transport** tab, fill in the connection details for the method: for AS2 the **Local AS2 ID**, **Partner AS2 ID**, **Endpoint URL** and certificates; for SFTP the **Host**, **Port**, **Username**, **Authentication** and directories; for VAN the mailbox and gateway details. Set **Max attempts** and the backoff under **Delivery retry** if the defaults do not suit.
5. On the **Envelope** tab, enter the **ISA sender ID**, **ISA receiver ID**, **GS sender ID**, **GS receiver ID**, **X12 version**, **Environment** and **Acknowledgment preference**.
6. On the **Secrets** tab, enter the password or private key the method needs.
7. Select **Save** (or **Save & close**).

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
Viewing profiles needs read access to EDI; creating needs create access and editing needs update access to EDI. **Test connection** is not offered for Internal profiles. Internal profiles route through an accepted organization connection, chosen under **Connection** and **Connected organization**, and do not use X12 envelope IDs or stored credentials. Deliveries that use up all retry attempts are dead-lettered and show on [Messages](/edi/messages).
