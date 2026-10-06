---
path: /admin/integrations
aliases: [connected apps, marketplace, third-party connections, API keys for services, Samsara, PC*Miler, Google Maps, telematics setup, email provider, fuel card feed, CarrierOk, FMCSA, QuickBooks, QuickBooks Online, accounting sync, Intuit, ledger mode, send journal entries, opening balances, Xero, Xero organisation, Xero app, Xero webhook key, choose organisation, Business Central, Dynamics 365 Business Central, Microsoft Entra app, Business Central environment, Business Central company, webhook subscriptions]
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
  - /admin/integrations/xero/callback
  - /admin/integrations/business-central/callback
---

## What it's for
Integrations is where administrators connect the outside services Trenova works with. Each
service is a card grouped by category (such as Email, Telematics, Mapping & Routing, Weather,
Financial Data, Fuel Cards, Carrier Compliance and Accounting) with its description, links to
its docs, a button to open its settings and a switch showing whether it is connected. Services
include Resend and Postmark (email), Samsara (telematics), Google Maps and PC*Miler (mileage and
routing), OpenWeatherMap, OANDA Exchange Rates, EIA Fuel Prices, the WEX, Comdata and Ramp fuel
card feeds, CarrierOk and FMCSA QCMobile (carrier intelligence), QuickBooks Online, Xero and
Business Central (accounting). Some cards describe planned providers that cannot be configured yet. An
organization keeps its books in one accounting system at a time.

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
4. On **What is sent**, choose **Send documents** to send invoices, payments and bills and let
   QuickBooks keep the ledger, or **Send journal entries** to keep the ledger in Trenova and send
   QuickBooks every journal entry it posts. With journal entries, choose **Detailed** for one entry
   per Trenova entry or **Daily summary** for one entry per day, then select **Continue**. The
   choice is fixed once sending starts.
5. On **Match records**, Trenova reads the company's accounts, items, customers and vendors and
   proposes a match for each Trenova record. Tick the proposals that are right and select the
   confirm button, which names how many are ticked, or select a row to choose another record.
   When journal entries are sent, every GL account that carries posted entries needs a QuickBooks
   account.
6. When every required mapping is confirmed, select **Finish setup**. The rest can be done later
   on [Mappings](/accounting/sync/mappings) (**Open all mappings**).
7. Choose the **Start date**: documents dated before it are never sent. Leave **Send posted
   documents automatically** on to send each document as it is posted, or turn it off to hold each
   one in the [Sync ledger](/accounting/sync) until someone releases it. When the start date is in
   the past, tick **Also send documents already posted since the start date** to queue those too.
   Turn on **Send owner-operator settlements** to also send owner-operator settlements as bills;
   it is off unless you turn it on, and company driver pay is never sent. When journal entries
   are sent, tick **Send opening balances** instead to send one entry, dated the day before the
   start date, with every account's balance up to then.
8. Select **Start sending**. From then on Trenova sends invoices, credit and debit memos, customer
   payments, credit applications, carrier settlements and their payments as they are posted, and
   checks the connection every fifteen minutes.

### Send journal entries instead of documents
Keywords: ledger mode, journal entry sync, general ledger to QuickBooks, daily summary, detailed journal entries, opening balances
1. Open [Integrations](/admin/integrations) and connect QuickBooks Online.
2. On **What is sent**, choose **Send journal entries**. Under **Journal entries**, choose
   **Detailed** to send each posted Trenova entry as its own QuickBooks entry, or **Daily summary**
   to send one entry per day summed per account and customer or vendor. A day is sent again when
   an entry is posted to it later.
3. Select **Continue**, confirm a QuickBooks account for each GL account on **Match records**, then
   choose the **Start date**. Tick **Send opening balances** when QuickBooks does not already hold
   the balances up to that day.
4. Select **Start sending**. Invoices, payments and bills stay in Trenova; QuickBooks receives their
   journal entries once they are posted, including entries posted by hand on
   [Journals to post](/accounting/journals-to-post). The connection shows **What is sent** and when
   **Opening balances** were sent. The choice cannot be changed once sending starts, and the start
   date cannot move once opening balances are sent.

### Change how documents are sent
Keywords: automatic sync, hold documents, owner-operator settlements, 1099 drivers, driver bills
1. Open [Integrations](/admin/integrations) and open the QuickBooks Online card.
2. Under **Sync settings**, turn **Send posted documents automatically** on or off. When it is
   off, each new document waits in the [Sync ledger](/accounting/sync) until someone releases it;
   documents already held stay held.
   When journal entries are sent, only this setting is shown; the rest apply to documents.
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

### Connect Xero
Keywords: Xero setup, connect Xero, Xero sign in, Xero organisation, accounting sync with Xero
1. Open [Integrations](/admin/integrations) and open the Xero card. When QuickBooks Online is
   still connected, disconnect it first: an organization sends to one accounting system at a time.
2. Select the connect button. Trenova sends you to Xero's own page; sign in there and allow access
   to the organisation to connect. Trenova never sees the Xero password.
3. Xero sends you back to Trenova. When the sign-in allowed more than one organisation, Trenova
   asks you to **Choose the organisation**: select the one whose books this Trenova organization
   keeps, then **Connect this organisation**. Trenova gives up its access to the others. The
   choice has to be made within a few minutes; after that, start the connection again.
4. Trenova shows the organisation it connected: its name, legal name, country, **Home currency**,
   **Multicurrency** and **Books closed through** (Xero's lock date). Check it is the right one,
   then select **Continue**.
5. On **What is sent**, keep **Send documents**. Xero cannot receive Trenova's journal entries,
   because its manual journals cannot post to receivable, payable or bank accounts or name a
   customer or supplier, so the journal entry option is shown as unavailable with that reason.
   Select **Continue**.
6. On **Match records**, Trenova reads the organisation's accounts, items, contacts and proposes a
   match for each Trenova record. Xero invoice lines carry an account, so each charge type is
   matched to a revenue account rather than an item; the freight line's account and the deposit
   account are required. Xero has no payment terms or payment methods to match.
7. Select **Finish setup**, choose the **Start date**, then select **Start sending**, as for
   QuickBooks Online. Xero's authorization has no fixed end date, so there is no date to
   reconnect by.

### Use your own Xero app
Keywords: Xero app keys, Xero developer portal, client ID, client secret, redirect URI, Xero webhook key, self-hosted Xero
1. On the Xero developer portal, create a Web app.
2. Open [Integrations](/admin/integrations) and open the Xero card. When this server has no Xero
   app of its own the keys form is already open; otherwise select **Use your own app**.
3. Copy the **Redirect URI** into the app's redirect URIs on the Xero developer portal exactly as
   written.
4. Enter the **Client ID** and **Client secret** from the app, then select **Save keys**. Xero has
   no sandbox, so there is no environment to choose; development uses Xero's demo company.
   Trenova checks the keys with Xero before saving them.
5. To have Xero tell Trenova about changes, select **Change keys**, copy the address under
   **Webhook endpoint (optional)**, which is specific to your app, into the app's webhooks on the
   Xero developer portal, enter the app's webhook key in the webhook key field and select **Save
   keys** again.
6. To go back to the server's app select **Remove**. While an organisation is connected, only the
   client secret and webhook key can change.

### Check or disconnect Xero
Keywords: Xero not syncing, Xero connection failing, reconnect Xero, disconnect Xero
1. Open [Integrations](/admin/integrations) and open the Xero card to see the connection's status,
   when it was **Last checked** and its **Last successful call**.
2. Select **Check now** to test the connection immediately.
3. When Xero no longer accepts Trenova's access, select **Reconnect** and approve the same
   organisation on Xero's page.
4. To stop, select **Disconnect** and confirm. Nothing already in Xero is changed.

### Connect Business Central
Keywords: Business Central setup, connect Business Central, Dynamics 365 Business Central, Microsoft sign in, choose company, Business Central environment, sandbox
1. Open [Integrations](/admin/integrations) and open the Business Central card. When another
   accounting system is still connected, disconnect it first: an organization sends to one
   accounting system at a time.
2. Select the connect button. Trenova sends you to Microsoft's sign-in page; sign in with a work
   account that can use Business Central and allow access. Trenova never sees the password.
3. Microsoft sends you back to Trenova. Trenova lists every company in every Business Central
   environment the account can open, each named with its environment and marked **(Sandbox)**
   when it is a sandbox. When there is more than one, select the company whose books this
   Trenova organization keeps, then connect it. The choice has to be made within a few minutes;
   after that, start the connection again.
4. Trenova shows the company it connected: its name, legal name, country, **Home currency**,
   **Multicurrency** and **Books closed through** (the day before Business Central's allowed
   posting dates begin, or the end of the last closed accounting period). Check it is the right
   one, then select **Continue**.
5. On **What is sent**, keep **Send documents**. Business Central cannot receive Trenova's journal
   entries, because its journal lines cannot post to a customer or vendor or apply to an invoice,
   so the journal entry option is shown as unavailable with that reason. Select **Continue**.
6. On **Match records**, Trenova reads the company's accounts, items, customers, vendors and
   payment terms and proposes a match for each Trenova record. Invoice lines are items, so each
   charge type is matched to a Business Central item; the freight line's item and the deposit
   account are required. Business Central has no payment methods on payments to match.
7. Select **Finish setup**, choose the **Start date**, then select **Start sending**, as for
   QuickBooks Online. Trenova posts every invoice, credit memo, bill and payment it sends.
   Payments go through a payment journal Trenova creates for each deposit or bank account, named
   with **TRN** and a short code.
8. Payments recorded in Business Central are not brought into Trenova: its API does not say which
   invoices a payment paid. An invoice paid in Business Central shows as a balance difference on
   the drift page, where it can be settled in Trenova. The sync settings say this in place of
   the payment setting.

### Use your own Microsoft Entra app
Keywords: Business Central app keys, Entra app registration, Azure app registration, client ID, client secret, redirect URI, self-hosted Business Central
1. In Microsoft Entra, register a multitenant web app. Under **API permissions**, add the
   delegated **Dynamics 365 Business Central** permission **Financials.ReadWrite.All**.
2. Open [Integrations](/admin/integrations) and open the Business Central card. When this server
   has no Entra app of its own the keys form is already open; otherwise select **Use your own
   app**.
3. Under the app's **Authentication**, add a web platform with the **Redirect URI** Trenova shows,
   exactly as written.
4. Enter the **Client ID** (the application ID) and a **Client secret** from **Certificates &
   secrets**, then select **Save keys**. Trenova checks the keys with Microsoft before saving
   them. Business Central has no webhook key: once a company is connected, Trenova subscribes to
   its changes itself and renews the subscriptions every few days.
5. To go back to the server's app select **Remove**. While a company is connected, only the client
   secret can change. Entra client secrets expire; enter the new one here before the old one
   does.

### Check or disconnect Business Central
Keywords: Business Central not syncing, Business Central connection failing, reconnect Business Central, disconnect Business Central, change subscriptions, webhook renewal
1. Open [Integrations](/admin/integrations) and open the Business Central card to see the
   connection's status, when it was **Last checked**, its **Last successful call**, how many
   **Change subscriptions** are active and when the **Next renewal due** is.
2. When the server has no public https address, the card says so and Trenova reads changes every
   five minutes instead of being told about them. When a subscription cannot be kept, the card
   shows why; Trenova retries within the hour.
3. Select **Check now** to test the connection immediately.
4. When Business Central no longer accepts Trenova's access, select **Reconnect** and approve the
   same company. Microsoft keeps the sign-in alive as long as Trenova uses it at least every 90
   days, which it does while connected.
5. To stop, select **Disconnect** and confirm. Trenova deletes its sign-in and its change
   subscriptions; nothing already in Business Central is changed. Microsoft gives no way to
   withdraw the consent from Trenova's side, so to remove it an administrator removes the app
   from **Enterprise applications** in Microsoft Entra.

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
by a signed-in person, as do saving or removing the Intuit app keys, choosing what is sent,
choosing the start date and changing the sync settings.
Seeing, connecting and disconnecting Xero or Business Central need the same access as QuickBooks
Online.
A QuickBooks company, Xero organisation or Business Central company can be connected to only one Trenova organization at a
time, and a Trenova organization sends to only one accounting system at a time: disconnect one
before connecting the other. When the connection fails or its authorization is about to run out, Watchtower raises an
item that links back here.
