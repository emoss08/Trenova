---
path: /edi/designer
aliases: [EDI template editor, X12 template, 204 template, EDI mapping designer, document profile, template certification]
related:
  - /edi/test-cases
  - /edi/partners
  - /edi/messages
---

## What it's for
The Template designer is where X12 document templates are written, validated, certified and activated. A template has versions; each version moves from draft to certified to active, and only a draft can be edited. The **Document preview & archive** tab ties a partner to a template through a document profile, previews the X12 a document would produce, and keeps an archive of generated messages.

EDI implementation specialists use it to build or change the documents sent to and received from trading partners.

## Tasks

### Create a template
Keywords: new X12 template, add EDI document
1. Open [Template designer](/edi/designer) and stay on the **Templates** tab.
2. In the **Templates** list, select **New**.
3. Choose the **Document type**, enter a **Name** and optional **Description**, and set the **X12 version**, **Group** and **Status** for the first draft.
4. Submit the dialog. The new template opens with its first draft version.

### Edit a draft version
Keywords: change segments, edit elements, template scripts
1. Open [Template designer](/edi/designer) and select the template in the **Templates** list (use **Search templates** or **Filter** to find it).
2. Pick a draft in **Versions**. If only a certified or active version exists, select **New draft** to copy it into a new draft.
3. Work in the **Elements** and **Scripts** tabs, using the **Segment outline** to move between segments.
4. Select **Save draft** to store segment, element and script changes, and **Save metadata** to store version details. Nothing is sent until you save.

### Validate, certify and activate a version
Keywords: publish template, go live, certify EDI template
1. Open [Template designer](/edi/designer) and select the template and draft version.
2. Select **Validate** and review the results in **Diagnostics** or on the **Validation** tab.
3. When there are no validation errors and no unsaved changes, select **Certify**.
4. Select **Activate** to make the certified version the one in use.
5. To retire a version that is not active, select **Archive version**.

### Set up a partner document profile and preview output
Keywords: partner document settings, preview X12, envelope, acknowledgment settings
1. Open [Template designer](/edi/designer) and switch to the **Document preview & archive** tab.
2. Under **Document profile**, choose the partner and either pick an existing document profile or start a new one.
3. Set the **Profile name**, template, **Version override**, **Group**, **Status**, **Validation** mode, envelope and acknowledgment settings, and any **Partner settings**, then select **Save profile**.
4. On the **Preview** tab, fill in the source document values and select **Preview provisional controls** to see the X12 that would be produced.
5. Select **Generate archive message** to generate the message and store it in the archive.

### Find a generated message in the archive
Keywords: archived EDI, download X12, control numbers
1. Open [Template designer](/edi/designer), switch to **Document preview & archive**, and open the **Archive** tab.
2. Filter by partner, **Transaction**, **Direction**, **Status**, **Search** text or the **Generated From** and **Generated To** dates.
3. Use the row actions to open the detail, copy the control numbers, copy the raw X12, or download it.

## Notes
Viewing the designer needs read access to EDI. Certified, active, superseded, deprecated and archived versions are read-only; create a new draft to change them. Certification fails while a version has validation errors, only certified versions can be activated, and the active version cannot be archived. Check a template against saved scenarios on [Test cases](/edi/test-cases).
