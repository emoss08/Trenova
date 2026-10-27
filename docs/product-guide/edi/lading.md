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
Keywords: edit EDI map, fix mapping, add a file
1. Open [Lading studio](/edi/lading), select the partner and pick the draft in the version picker. To change a version that is not a draft, open **Version actions** and select **New draft from this version**.
2. Pick a file in **Files**, or select **Add a file**, choose **Source file** or **Sample**, enter the **Path** and select **Add**.
3. Edit the file. Problems show under the text as you type and in the **Problems** tab; select one to go to it.
4. Select **Save**. Select **Discard** to go back to the saved version.

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
1. Save the draft, then in the **Release** tab pick each inbound map under **Map to replay**, choose a period and select **Start replay**. Each replay reads the partner's past messages through this version and the active one and lists every difference.
2. Select **Certify**. A version is certified when it has no errors, its tests pass and each inbound map replays without new rejections.
3. To try it on live messages first, open **Version actions** and pick how many days to shadow it. The shadow sends nothing and records where it differs from the active version.
4. Select **Activate**. The previous version is superseded and every new message is read and written with this one.

### Change a partner setting
Keywords: partner options, auto accept, EDI settings
1. Open the partner's active version and the **Settings** tab.
2. Change the values (blank keeps the default) and select **Save settings**.

## Notes
Viewing the studio needs read access to EDI; starting a project or a new draft needs create access, and saving, certifying, activating, shadowing, replaying, archiving and changing settings need update access to EDI. Only a draft can change; every other version opens read-only. An active or shadowing version cannot be archived until it is stopped.
