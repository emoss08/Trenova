---
path: /admin/document-parsing-rules
aliases: [parsing rules, extraction rules, rate confirmation parser, document templates for extraction, rule sets]
related:
  - /admin/document-intelligence
  - /admin/document-operations
---

## What it's for
Document parsing rules teach Trenova how to read documents from a particular provider or format,
such as one broker's rate confirmation. Each rule set is listed in the sidebar on the left;
selecting one opens it with four tabs: **Metadata** (name, document kind, priority and the
published version), **Versions** (draft and published versions of the rules), **Fixtures**
(saved sample documents with expected results) and **Simulation** (run a version against pasted
document text and see what it extracts).

A version says which documents it matches (**Match config**) and what to pull out of them
(**Rule builder**: sections, fields and stops). Drafts can be edited; publishing a draft makes it
the version used in production.

## Tasks

### Create a rule set
Keywords: new parser, add provider rules
1. Open [Parsing rules](/admin/document-parsing-rules).
2. Select **New** at the top of the sidebar (or **Create rule set** when there are none).
3. Enter a **Name** that identifies the provider or document format and select **Create**.
4. On the **Metadata** tab, set the **Document kind**, **Priority** and **Description**, then
   select **Save changes**.

### Write and publish a version of the rules
Keywords: edit rules, draft version, publish parser, extraction fields
1. Open [Parsing rules](/admin/document-parsing-rules) and select the rule set in the sidebar.
2. Open the **Versions** tab and select **New draft**. The new draft opens.
3. Under **Version settings**, give it a **Label** and choose the **Parser mode**.
4. On **Match config**, fill in **Provider fingerprints**, **File name contains**, **Requires
   all**, **Requires any** and **Section anchors** so the rules match the right documents.
5. On **Rule builder**, use **Add section**, **Add field** and **Add stop** to define what to
   extract, then select **Save changes**.
6. When it is ready, select **Publish** and confirm with **Publish version**. The previously
   published version is archived and this version becomes read-only.

### Test rules against sample documents
Keywords: fixture, test parser, simulate extraction, dry run
1. Open [Parsing rules](/admin/document-parsing-rules) and select the rule set.
2. To keep a sample, open **Fixtures** and select **New fixture**. In the fixture that opens,
   paste the **Full document text**, describe the expected results under **Assertions**, and
   select **Save fixture**.
3. To try a version, open **Simulation**, choose the **Version**, paste the **Document text**,
   optionally fill in **File name** and **Provider fingerprint**, and select **Run simulation**.

### Delete a rule set
Keywords: remove parser
1. Open [Parsing rules](/admin/document-parsing-rules) and select the rule set.
2. Select the delete button in its header and confirm with **Delete**.

## Notes
Viewing the page needs read access to document parsing rules. Creating rule sets, drafts and
fixtures needs create access, deleting needs delete access, and **Publish** appears only for
people who may activate parsing rules. Only draft and published versions can be simulated, and
published or archived versions cannot be edited.
