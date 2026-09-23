---
path: /dispatch/carriers
aliases: [trucking companies, brokered carriers, outside carriers, partner carriers, vendors, MC number, DOT number]
related:
  - /dispatch/carrier-monitoring
  - /dispatch/carrier-sourcing
  - /dispatch/routing-guides
  - /carrier-settlements/settlements
---

## What it's for
Carriers are the outside trucking companies your brokerage tenders loads to. Each carrier record holds its identity and operating authority (DOT and MC numbers), compliance status, safety rating and insurance policies, tax details for 1099 reporting, remittance and payment terms, and contacts, including who receives rate confirmations. Carrier sales, compliance and accounting staff keep these records current.

Once a carrier is saved, its **Intelligence** tab shows what your carrier intelligence provider reports about it: authority, insurance, safety record, and any findings that would block a tender.

## Tasks

### Add a carrier
Keywords: new carrier, onboard carrier, set up carrier
1. Open [Carriers](/dispatch/carriers).
2. Select **New carrier**.
3. On the **Identity** tab, fill in **Status**, **Code**, **Name** and **Carrier type**, and add the **DOT number**, **MC number**, address and contact details you have.
4. Fill in the other tabs as needed: **Compliance & insurance** (**Compliance status**, **Safety rating**, and policies with **Add policy**), **Tax** (**Tax ID**, **Tax ID type**, **W-9 on file**, **1099 Eligible**), **Remittance** (**Payment method**, **Payment term days** and the remit-to address) and **Contacts** (**Add contact**).
5. Select **Save**.

### Set who receives rate confirmations
Keywords: rate con email, carrier dispatcher contact
1. Open [Carriers](/dispatch/carriers) and select the carrier.
2. Open the **Contacts** tab and select **Add contact**, or edit an existing contact.
3. Fill in **Name**, **Email**, **Phone** and **Title**, and turn on **Receives rate confirmations**. An email is required for that contact. Turn on **Primary contact** for the main point of contact.
4. Select **Save**.

### Vet a carrier
Keywords: check authority, FMCSA check, carrier compliance check, verify insurance
1. Open [Carriers](/dispatch/carriers) and select the carrier. It needs a **DOT number** saved on the **Identity** tab.
2. Open the **Intelligence** tab and select **Vet carrier** (or **Vet now** if it was vetted before).
3. Choose the **Depth**, optionally turn on **Force a fresh pull**, and select **Vet now**.
4. Review the result on the **Findings**, **Profile**, **Timeline** and **History** views.

### Update the carrier record from its latest vetting
Keywords: sync carrier data, apply suggestions, update insurance from provider
1. Open the carrier and go to the **Intelligence** tab.
2. Open **Sync** to see **Suggested updates** from the last vetting.
3. Tick the suggestions you want and select the apply button.

### Watch a carrier for changes
Keywords: monitor carrier, continuous monitoring, alerts on authority
1. Open the carrier and go to the **Intelligence** tab.
2. Turn on **Continuous monitoring**. Changes then appear on the carrier's **Timeline** and on [Carrier monitoring](/dispatch/carrier-monitoring).

### Change the status of several carriers
Keywords: do not use, inactivate carriers, bulk status
1. Open [Carriers](/dispatch/carriers).
2. Tick the rows to change.
3. Select **Update status** in the bar that appears and choose **Active**, **Inactive** or **Do not use**.

## Notes
The page is available only to organizations with brokerage turned on.

Adding needs create permission for carriers and editing needs update permission. The **Intelligence** tab needs read access to carrier intelligence; vetting and monitoring need update access to it, granting a finding override (**Grant override**) needs approve access, and applying sync suggestions also needs carrier update permission.

If no carrier intelligence provider is connected, the **Intelligence** tab offers **Open integrations** to connect one (CarrierOK or the free FMCSA QCMobile service).
