---
path: /admin/integrations
aliases: [connected apps, marketplace, third-party connections, API keys for services, Samsara, PC*Miler, Google Maps, telematics setup, email provider, fuel card feed, CarrierOk, FMCSA, QuickBooks, QuickBooks Online, accounting sync, Intuit]
related:
  - /accounting/sync
  - /admin/inbound-mailboxes
  - /admin/api-keys
  - /dispatch/carrier-monitoring
  - /fuel/feed-runs
  - /fuel/configuration-files/surcharge
  - /accounting/sync/mappings
covers:
  - /admin/integrations/quickbooks/callback
---

## What it's for
Integrations is where administrators connect the outside services Trenova works with. Each
service is a card grouped by category (such as Email, Telematics, Mapping & Routing, Weather,
Financial Data, Fuel Cards, Carrier Compliance and Accounting) with its description, links to
its docs, a button to open its settings and a switch showing whether it is connected. Services
include Resend and Postmark (email), Samsara (telematics), Google Maps and PC*Miler (mileage and
routing), OpenWeatherMap, OANDA Exchange Rates, EIA Fuel Prices, the WEX, Comdata and Ramp fuel
card feeds, CarrierOk and FMCSA QCMobile (carrier intelligence), and QuickBooks Online
(accounting). Some cards describe planned providers that cannot be configured yet.

## Tasks

### Find an integration
Keywords: search integrations, connected services, what is connected
1. Open [Integrations](/admin/integrations).
2. Type in **Search integrations**, or narrow the list with the category filter (**All
   categories** or one category) and the status filter (**All statuses**, **Connected** or
   **Disconnected**).
3. Change **Sort By** to **Name (A-Z)** or **Name (Z-A)** if needed.

### Connect a service
Keywords: set up integration, add API key, enable integration, turn on mileage
1. Open [Integrations](/admin/integrations) and find the service's card.
2. Select the card's button (or its switch) to open the service's settings.
3. Turn on the service's enable switch (for example **Enable Google maps** or
   **Enable Samsara**) and enter its credentials, such as the API key. When a key is already
   saved, leave the field blank to keep it.
4. Where offered, select **Test connection** to check the credentials work.
5. Select **Save changes** (or **Save configuration** for Samsara).

### Set up Samsara telematics
Keywords: Samsara webhook, sync drivers to Samsara, worker sync, telematics
1. Open [Integrations](/admin/integrations) and open the Samsara card.
2. On **Configuration**, turn on **Enable Samsara**, enter the token, and select **Save
   configuration**. A token is required before Samsara can be enabled.
3. Copy the URL under **Webhook endpoint** and point Samsara's webhook at it. It is unique to your
   organization and appears after the configuration is saved.
4. Select **Worker sync** and choose **Start sync** to push Trenova worker records into Samsara.
   **Detect drift** and **Repair drift** find and fix workers that no longer match.

### Configure carrier intelligence
Keywords: carrier vetting, CarrierOk, FMCSA QCMobile, carrier monitoring rules, carrier data spend
1. Open [Integrations](/admin/integrations) and open the CarrierOk or FMCSA QCMobile card.
2. On **Connection**, turn on the provider, enter its credentials and save. Select **Make
   primary** to make it the primary carrier intelligence provider; another enabled provider acts
   as the fallback.
3. On **Rules**, set the **Vetting rules** and **Enforcement** (for example **Refresh before
   tender** and **Disqualify on block**).
4. On **Monitoring**, choose which carriers are enrolled for continuous monitoring and the
   **Refresh cadence**.
5. On **Spend**, set a **Monthly spend cap** and how long raw data is kept, then select **Save
   changes**.

### Connect QuickBooks Online
Keywords: QuickBooks setup, connect accounting, Intuit sign in, accounting sync
1. Open [Integrations](/admin/integrations) and open the QuickBooks Online card.
2. Select the connect button. Trenova sends you to Intuit's own page; sign in there and choose
   the company to connect. Trenova never sees the QuickBooks password.
3. Intuit sends you back to Trenova, which finishes the connection and shows the company it
   connected: its name, legal name, country, **Home currency**, **Multicurrency** and **Books
   closed through**. Check it is the right company, then select **Continue**.
4. On **Match records**, Trenova reads the company's accounts, items, customers and vendors and
   proposes a match for each Trenova record. Tick the proposals that are right and select the
   confirm button, which names how many are ticked, or select a row to choose another record.
5. When every required mapping is confirmed, select **Finish setup**. The rest can be done later
   on [Mappings](/accounting/sync/mappings) (**Open all mappings**).
6. Choose the **Start date**: documents dated before it are never sent. Leave **Send posted
   documents automatically** on to send each document as it is posted, or turn it off to hold each
   one in the [Sync ledger](/accounting/sync) until someone releases it. When the start date is in
   the past, tick **Also send documents already posted since the start date** to queue those too.
   Turn on **Send owner-operator settlements** to also send owner-operator settlements as bills;
   it is off unless you turn it on, and company driver pay is never sent.
7. Select **Start sending**. From then on Trenova sends invoices, credit and debit memos, customer
   payments, credit applications, carrier settlements and their payments as they are posted, and
   checks the connection every fifteen minutes.

### Change how documents are sent
Keywords: automatic sync, hold documents, owner-operator settlements, 1099 drivers, driver bills
1. Open [Integrations](/admin/integrations) and open the QuickBooks Online card.
2. Under **Sync settings**, turn **Send posted documents automatically** on or off. When it is
   off, each new document waits in the [Sync ledger](/accounting/sync) until someone releases it;
   documents already held stay held.
3. Turn **Send owner-operator settlements** on to send owner-operator settlements to QuickBooks
   as bills, with a 1099 vendor for each driver. Settlements posted from then on are sent; request
   a backfill from the [Sync ledger](/accounting/sync) to send earlier ones.
4. Choose what happens to payments recorded in QuickBooks Online: **Wait for someone to apply them** lists
   each one on [Payments from the books](/accounting/sync/inbound), **Apply them automatically**
   brings in those that match, and **Leave them out of Trenova** stops reading them.
5. Select **Save**.

### Use your own Intuit app
Keywords: Intuit app keys, client ID, client secret, redirect URI, QuickBooks developer app, self-hosted QuickBooks
1. On the Intuit developer portal, create an app with the com.intuit.quickbooks.accounting scope.
2. Open [Integrations](/admin/integrations) and open the QuickBooks Online card. When this server
   has no Intuit app of its own the keys form is already open; otherwise select **Use your own
   app**.
3. Copy the **Redirect URI** into the app's redirect URIs on the Intuit developer portal exactly as
   written. Development keys accept http://localhost; production keys need https.
4. Choose the **Environment**, enter the **Client ID** and **Client secret** from the same
   environment, and optionally the webhook verifier token, then select **Save keys**. Trenova
   checks the keys with Intuit before saving them.
5. To change the keys later select **Change keys**; to go back to the server's app select
   **Remove**. While a company is connected, only the client secret and verifier token can change.

### Check or disconnect QuickBooks Online
Keywords: QuickBooks not syncing, QuickBooks connection failing, reconnect QuickBooks, revoke QuickBooks
1. Open [Integrations](/admin/integrations) and open the QuickBooks Online card to see the
   connection's status, when it was **Last checked**, its **Last successful call** and the date
   to **Reconnect by**.
2. Select **Check now** to test the connection immediately instead of waiting for the next check.
3. When QuickBooks no longer accepts Trenova's access, or the reconnect date is near, select
   **Reconnect** and approve the same company on Intuit's page.
4. To stop, select **Disconnect** and confirm. Trenova revokes its access; nothing already in
   QuickBooks is changed.

## Notes
Viewing the page needs read access to integrations; saving or testing a connection needs update
access to integrations. Starting a Samsara worker sync needs update access to workers. The carrier
intelligence tabs need read access to carrier intelligence, and changing them needs manage access.

Seeing the QuickBooks Online connection needs read access to the accounting integration, **Check now** needs
update access, and connecting, reconnecting or disconnecting needs manage access and must be done
by a signed-in person, as do saving or removing the Intuit app keys, choosing the start date and
changing the sync settings.
A QuickBooks company can be connected to only one Trenova organization at a time. When the connection fails or its authorization is about to run out, Watchtower raises an
item that links back here.
