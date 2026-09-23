---
path: /dispatch/carrier-sourcing
aliases: [find carriers, carrier search, carrier lookup, FMCSA lookup, USDOT lookup, new carriers, carrier onboarding]
related:
  - /dispatch/carriers
  - /dispatch/carrier-monitoring
  - /admin/integrations
---

## What it's for
Carrier sourcing lets carrier sales and compliance staff find carriers that are not yet in Trenova, check them against your vetting rules, and import the ones you want as carrier records. You can search the market by name, or enter a USDOT or MC number to pull a single carrier. Each result shows the carrier's authority, fleet, risk and findings, and whether it is already in Trenova.

Searching needs a connected carrier data provider. Some providers look carriers up by USDOT or MC number only; with those, name search is not available.

## Tasks

### Search for carriers
Keywords: find carriers by state, carriers on a lane, hazmat carriers, carrier search
1. Open [Carrier sourcing](/dispatch/carrier-sourcing).
2. Type a carrier name in the search bar and press Enter, or pick one of the example searches.
3. Select **Filter** to narrow by **Home state**, **Lane origin** and **Lane destination**, **Power units**, **Authority age** or **Screening** (**Hazmat carriers only**, **Hide blocked carriers**, **Hide carriers in Trenova**).
4. Change the order with the sort menu: **Best match**, **Fleet size** or **Authority age**.
5. Select **Load more** to see the next page of results.

### Look up one carrier by USDOT or MC number
Keywords: DOT lookup, MC lookup, check a carrier
1. Open [Carrier sourcing](/dispatch/carrier-sourcing).
2. Enter the USDOT or MC number in the search bar and press Enter.
3. Select the result to open its details. Select **Pull full profile** for network signals and lanes; the button shows the provider cost when there is one.

### Import a carrier
Keywords: add carrier from FMCSA, onboard carrier, create carrier from search
1. Find the carrier by search or lookup.
2. Select **Import** on its row, or open it and select **Import carrier**.
3. Check the **Carrier code** (generated from the name) and turn on **Monitor this carrier** to watch it for authority, insurance and safety changes.
4. Select **Import carrier**. The carrier is created from its FMCSA record and starts in Pending compliance until someone reviews it.
5. Select **Open carrier** in the confirmation to finish its setup on [Carriers](/dispatch/carriers).

## Notes
The page is available only to organizations with brokerage turned on, and needs read access to carrier sourcing. Importing needs import access to carrier sourcing and create permission for carriers.

A carrier that already exists shows **In Trenova** (or **Open in Trenova** in its details) instead of an import button.

If no provider is connected, the page offers **Open integrations** to connect one.
