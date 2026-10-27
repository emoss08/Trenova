---
path: /edi/lading
aliases: [EDI maps, EDI mapping, X12 mapping, partner project, Lading, EDI rules, EDI flows, EDI implementation]
related:
  - /edi/partners
  - /edi/flows
  - /edi/messages
covers:
  - /edi/lading/:partnerId
---

## What it's for
The Lading studio is where you write how Trenova reads and writes each trading partner's X12. Every partner has one Lading project: a partner file with its interchange identity and settings, a map for each transaction set, the partner's rules, the flow that follows each tender from its 204 to its 210, samples of the partner's real messages, and tests. The studio checks the whole project as you type, runs maps on samples, and releases a version only after its tests and replays pass.

EDI coordinators and implementation teams use it to onboard a partner's EDI, change a map when a partner changes its guide, and release the change safely.

## Tasks

### Start a partner's Lading project
Keywords: new EDI map, set up partner EDI, starter project
1. Open [Lading studio](/edi/lading) and select the partner's row.
2. Under **Trenova is**, pick **The carrier: the partner tenders to us** or **The broker: we tender to the partner**.
3. Under **Sets traded**, keep the sets you exchange with the partner.
4. Select **Start the project**. The starter opens as a draft that sends nothing until it is certified and activated.

### Change a map, rule or flow
Keywords: edit EDI map, fix mapping, add a file, new map, new guide, new rules, new flow, library
1. Open [Lading studio](/edi/lading), select the partner and pick the draft in the version picker. To change a version that is not a draft, open **Version actions** and select **New draft from this version**.
2. Pick a file in the file tree. Under the partner file, each flow lists the maps and rules it uses, then **Shared** guides and libraries, **Not in any flow** for files no flow uses, and **Samples**. Press Ctrl+P (⌘P on a Mac) to filter by path. To start a new file, select **Add to the project** (or the add button beside **Flows**), type to find **Map**, **Rules**, **Flow**, **Guide**, **Library** or **Partner file** (or press its letter), fill in its options and **Name**, and select the add button. **Starts as** shows the text the file opens with.
3. Edit the file. Problems show under the text as you type and in the **Problems** tab; select one to go to it.
4. Select **Save**. Select **Discard** to go back to the saved version.

### Add the partner's sample messages
Keywords: upload EDI sample, add sample, test message, X12 file, CSV, JSON
1. Open the partner's draft in [Lading studio](/edi/lading).
2. Select **Add to the project**, then **Upload from your computer** (or press U), and drop or choose the files. Files ending in .x12, .edi or .txt are read as X12; .csv and .json as named.
3. Select **Add samples**. To write one by hand instead, choose **X12 message**, **CSV file**, **JSON file** or **Trenova document**, paste its text under **Content**, and select **Add sample**.

### Rename, move or remove a file
Keywords: rename EDI file, move sample, delete file
1. Open the partner's draft in [Lading studio](/edi/lading).
2. In **Files**, open the file's actions and select **Rename** (or press F2 on it). Type the new path and press Enter; renaming a sample also changes the tests that read it.
3. Select **Remove** in the same menu to take a file out of the draft. **Discard** brings it back until you save.

### Try a map on a sample
Keywords: run EDI map, test X12, preview mapping
1. In the studio, open the **Run** tab.
2. Pick a map and a sample.
3. Select **Run here** to run the files as they are now, or **Run with live data** to run the saved version with lookups answered from your records.
4. Read each transaction's findings and outcomes, the document it produced and the X12 it wrote. Hover a field in the editor to see the value it came to.

### Run the tests
Keywords: EDI tests, certify readiness, coverage
1. In the studio, open the **Tests** tab and select **Run the tests**.
2. Select a failing test to open it. Lines no test reached are marked in the editor, and **Sent by the partner, read by no map** lists elements the samples carry that nothing reads.

### Release a version
Keywords: certify EDI, activate map, go live, replay, shadow
1. Save the draft and open the **Release** tab. The steps across the top (**Saved**, **Replayed**, **Certified**, **Shadow**, **Live**) show how far the version has come; hover one to see what it needs.
2. Under **Replays**, pick each inbound map under **Map to replay**, choose a period and select **Replay**. Each replay reads the partner's past messages through this version and the active one; select a replay to see how many read the same and every field that differs.
3. Select **Certify**. A version is certified when it has no errors, its tests pass and each inbound map replays without new rejections.
4. To try it on live messages first, open **Version actions** and pick how many days to shadow it. The shadow sends nothing; under **Shadow**, select a message to see where it read differently from the active version.
5. Select **Activate**. The previous version is superseded and every new message is read and written with this one.

### Change a partner setting
Keywords: partner options, auto accept, EDI settings
1. Open the partner's active version and the **Settings** tab.
2. Change the values (blank keeps the default) and select **Save settings**.

## Notes
Viewing the studio needs read access to EDI; starting a project or a new draft needs create access, and saving, certifying, activating, shadowing, replaying, archiving and changing settings need update access to EDI. Only a draft can change; every other version opens read-only. An active or shadowing version cannot be archived until it is stopped.
