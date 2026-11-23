---
path: /admin/data-entry-controls
aliases: [case formatting, uppercase codes, text formatting rules, capitalization settings]
related:
  - /admin/organization-settings
  - /admin/sequence-configs
---

## What it's for
Data entry control sets how text is formatted automatically when people enter data, so codes,
names, email addresses and cities look the same everywhere. The page has one section, **Case
formatting**, with four settings.

Administrators use it once when setting up the organization and whenever they want to change the
house style for entered text.

## Tasks

### Change how entered text is capitalized
Keywords: uppercase, lowercase, title case, as entered
1. Open [Data entry controls](/admin/data-entry-controls).
2. In **Case formatting**, choose a format for each kind of text:
   - **Codes** applies to the codes of customers, carriers, factoring companies, locations, fleet
     codes, equipment types, service and shipment types, accessorials, hazardous materials,
     document types, account types and GL accounts.
   - **Names** applies to the names of workers and their emergency contacts, customers, carriers
     and their contacts, factoring companies, locations, commodities, hazardous materials,
     document types, account types, GL accounts and fiscal years.
   - **Email addresses** applies to worker, carrier, carrier contact and factoring company emails
     and to customer status update and billing email recipients. Sign-in emails are never
     changed.
   - **Cities** applies to the city on workers, customers, carriers (including the remit-to
     city), factoring companies and locations.
3. The formats are **As entered** (kept exactly as typed), upper case, lower case and
   **Title case** (the first letter of each word capitalized and the rest left as typed, so
   "acme freight LLC" becomes "Acme Freight LLC"). Email addresses offer every format except
   **Title case**.
4. Select **Save changes** in the bar that appears at the bottom of the page.

## Notes
Opening the page needs read access to data entry control; saving needs update access. The rules
apply to the whole organization. Every format also trims extra spaces, and codes lose their spaces entirely, so
"ACME 01" is saved as "ACME01". A record is formatted when it is saved, so changing a setting
reformats a record the next time someone saves it. Matching an inbound email to a customer or
carrier ignores case, so any email format still matches.
