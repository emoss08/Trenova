---
path: /billing/configuration-files/rate-zones
aliases: [zones, market areas, KMA, regions, pricing zones, lane zones, postal zones]
related:
  - /billing/rate-agreements
  - /billing/configuration-files/rate-matrices
---

## What it's for
Rate zones lets you name a market area once, such as "Southeast" or a metro, and price against that name instead of listing every state, city or postal prefix it covers. A zone is a list of places; a lane on a rate agreement can use a zone as its origin or destination, and a rate matrix can use zones as its rows or columns.

Pricing staff maintain zones here. The table shows each zone's **Status**, **Code**, **Name**, **Description** and **Created** date.

## Tasks

### Add a rate zone
Keywords: new zone, create market area, define region
1. Open [Rate zones](/billing/configuration-files/rate-zones).
2. Select **New rate zone**.
3. Fill in **Status**, **Zone kind** (for example **Custom**, **Region** or **Metro**), **Code** (the short name lanes refer to) and **Name**, and optionally a **Description**.
4. Under **Places**, select **Add place** for each part of the zone. Choose a **Place type** (**State**, **City**, **Postal prefix**, **Postal code**, **Location** or **Country**) and fill in the value: a state, a city with its state, the digits of the postal code or prefix, a single facility, or a country.
5. Select **Save**.

### Change what a zone covers
Keywords: edit zone, add state to zone, remove postal code
1. Open [Rate zones](/billing/configuration-files/rate-zones) and select the zone's row.
2. Under **Places**, select **Add place** to add an area, or **Remove** next to a place to drop it.
3. Select **Save**. Every lane and matrix that prices against the zone uses the new list from then on.

### Retire a rate zone
Keywords: deactivate zone, disable zone
1. Open [Rate zones](/billing/configuration-files/rate-zones) and select the zone's row.
2. Set **Status** to **Inactive** and select **Save**.

### Find a rate zone
Keywords: search zones, filter zones
1. Open [Rate zones](/billing/configuration-files/rate-zones).
2. Type a code or name in the search box, or use **Filter** and **Sort** in the table toolbar.

## Notes
Viewing the page needs read access to rate zones; **New rate zone** only appears for people who can create them.

A zone with no places matches nothing, so any lane written against it never applies. An inactive zone also stops matching, and every lane written against it stops with it. A zone cannot contain another zone; list its states, cities, postal codes or locations directly. City names are matched without regard to case.
