---
path: /admin/integrations
aliases: [connected apps, marketplace, third-party connections, API keys for services, Samsara, PC*Miler, Google Maps, telematics setup, email provider, fuel card feed, CarrierOk, FMCSA]
related:
  - /admin/inbound-mailboxes
  - /admin/api-keys
  - /dispatch/carrier-monitoring
  - /fuel/feed-runs
  - /fuel/configuration-files/surcharge
---

## What it's for
Integrations is where administrators connect the outside services Trenova works with. Each
service is a card grouped by category (such as Email, Telematics, Mapping & Routing, Weather,
Financial Data, Fuel Cards and Carrier Compliance) with its description, links to its docs, a
button to open its settings and a switch showing whether it is connected. Services include
Resend and Postmark (email), Samsara (telematics), Google Maps and PC*Miler (mileage and routing),
OpenWeatherMap, OANDA Exchange Rates, EIA Fuel Prices, the WEX, Comdata and Ramp fuel card feeds,
and CarrierOk and FMCSA QCMobile (carrier intelligence). Some cards describe planned providers
that cannot be configured yet.

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

## Notes
Viewing the page needs read access to integrations; saving or testing a connection needs update
access to integrations. Starting a Samsara worker sync needs update access to workers. The carrier
intelligence tabs need read access to carrier intelligence, and changing them needs manage access.
