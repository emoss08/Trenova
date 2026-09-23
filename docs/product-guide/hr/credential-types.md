---
path: /hr/credential-types
aliases: [licence types, license types, certifications, endorsements, driver documents, medical card]
related:
  - /hr/workers
  - /hr/training-courses
  - /hr/checklist-templates
---

## What it's for
Credential types is the list of licences, cards, endorsements and certificates workers can hold,
such as a CDL, a medical card or a TWIC card. Each type says whether it is required and for which
driver types, how far ahead of expiry it is flagged, its typical validity, and whether a number
or a scanned document is needed. HR and safety administrators maintain the list; the credentials
themselves are recorded on each worker's **Credentials** tab.

## Tasks

### Add a credential type
Keywords: new credential, track a certificate, add endorsement type
1. Open [Credential types](/hr/credential-types).
2. Select **New credential type**.
3. Under **General**, fill in **Code**, **Name**, **Category** and **Status**, and optionally a
   **Description**.
4. Under **Compliance**, turn on **Required** if workers must hold it to be compliant, pick the
   **Required for driver types**, and set the **Renewal alert window** and **Typical validity**.
5. Turn on **Requires a number** or **Requires a document** if the credential cannot be saved
   without a number or verified without a scan.
6. Select **Save**, or choose **Save & close** from the save button's menu.

### Edit a credential type
1. Open [Credential types](/hr/credential-types).
2. Select the type's row, change the fields, and select **Save** or **Save & close**.

### Deactivate or restore credential types
Keywords: retire credential type, archive, reactivate
1. Open [Credential types](/hr/credential-types).
2. Tick the types, then choose **Deactivate** or **Restore** in the bar at the bottom of the
   table. You can also right-click a single type.

## Notes
- The page needs read access to credential types and the asset operations feature.
  **Deactivate** and **Restore** only appear for people allowed to archive and restore types.
- Required types appear on every matching worker; a missing or expired one makes the worker
  non-compliant. Leave every **Required for driver types** box unchecked to require it for every
  worker.
- A worker holds one active credential of each type; renewing replaces the earlier one.
- System types ship with Trenova and their code cannot be changed. Types that mirror a
  worker-profile field cannot be deactivated.
