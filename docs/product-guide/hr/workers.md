---
path: /hr/workers
aliases: [drivers, employees, staff, roster, driver list, personnel, driver file, driver record, worker page, paid time off, PTO requests]
related:
  - /hr/my-team
  - /hr/credential-types
  - /hr/training-courses
  - /hr/pto-policies
  - /hr/checklist-templates
  - /hr/fleet-safety
  - /equipment/tractors
covers:
  - /hr/workers/new
  - /hr/workers/$id
---

## What it's for
Workers is the roster of every driver and employee, with their status, tenure, worker type,
driver type, fleet code and at-a-glance compliance, training, safety and drug and alcohol
standing. HR, safety and operations staff use it to add people, keep their records current and
manage paid time off. The **Paid time off** tab lists every time-off request so managers can
approve, reject or cancel them, above a chart of approved time off and a list of requested PTO.

Selecting a worker's row opens that worker's own page, which holds their whole file. **New
worker** opens a page of its own for adding someone.

The top of a worker's page is their record header: name, a standing badge (**Good standing**,
**Worth watching**, **At risk** or **Cannot be assigned**), status, worker type and driver type,
then fleet code, position, where they live, phone and tenure. Hovering the standing badge says
why. On the right are **Add to record**, which opens the form for any kind of entry (an
employment event, a credential, a drug or alcohol collection, a Clearinghouse query, a training
result, a safety event, a recognition, a review, a time off request, a leave case or a
document) on the tab it belongs to, and a **More actions** menu with **Copy link**, **Open fleet
codes** and **End employment…**. A worker you opened from the roster also has previous and next
buttons beside the page title, so you can step through the same list, in the same order and
with the same filters, without going back. Entries and tabs appear only where the plan includes
them and you may use them.

The page has ten tabs: **Overview**, **Profile**, **Employment**, **Qualification**,
**Development**, **Safety**, **Time**, **Pay**, **Files** and **Activity**. A tab that holds
several parts shows them as a row of views under the header, each with a one-line summary of
how that part stands:

- **Employment**: **History** (the employment events) and **Checklists**.
- **Qualification**: **Credentials**, **DQ file** and **Drug and alcohol**.
- **Development**: **Training** and **Reviews**.
- **Time**: **Hours of service**, **Schedule**, **Time off** and **Leave**.
- **Pay**: **Pay profile** and **Benefits**.
- **Files**: **Documents**, **Driver portal** and **Agent memory**.

A tab carries a red dot when something in it needs action and an amber dot when something is
worth a look; **Files** shows how many documents are on file. Forms open in a pane beside the
view, and decisions (archive, void, cancel) ask in a dialog.

### What the Overview tells you
The **Overview** answers whether this person can be put on a load. The band at the top reads
**Cleared to drive** with the date it holds through, or **Not cleared to drive** with the
reason and since when. Under it, **What comes due** lays the dated requirements (license and
endorsements, medical, reviews and queries, training) on a time axis from today, marking which
ones stop driving when they lapse and which only stop hazmat loads. **Right now** shows the
driver's hours, when they were **Last home**, their **Next time off**, and whether dispatch
holds them (**Assignment**: **Dispatch**, **Qualified**, **Compliance**, **ELD**). **Needs
attention** ranks what to do next under **Needs action**, **Coming due** and **For
information**, each with a button that opens the place to fix it. **The file** is a grid with
one cell for every part of the record and where it stands; select a cell to open that part.

## Tasks

### Add a worker
Keywords: new driver, hire, create employee, onboard driver
1. Open [Workers](/hr/workers).
2. Select **New worker**. The page opens at its own address.
3. Under **Identity**, fill in **First name**, **Last name**, **Worker type**, **Driver type**,
   **Fleet code** and **Position**. Under **Home address**, pick the street address from the list
   when Google Maps is connected: that records where home is, so dispatch can favour moves that
   head a driver home. Under **Contact**, add **Email** and **Phone**.
4. Under **Employment dates**, enter the **Hire date**. Under **License and clearance**, enter
   the **License number**, **Issuing state**, **CDL class** and **License expires**, plus any
   medical details you know.
5. Select **Create worker**. The note above the form says what creating the worker starts: the
   onboarding checklist, required training and the default time off policy. Select **Cancel** to
   leave without creating anyone.

### Update a worker's record
Keywords: edit driver, change driver type, deactivate worker, terminate, save worker
1. Open [Workers](/hr/workers), find the worker with the search box or **Filter**, then select
   their row.
2. Go to the **Profile** tab and change the fields in its cards: **Identity**, **Home address**,
   **Contact**, **Employment dates**, **License and clearance** and the **Qualification** card.
3. Select **Save changes** in the bar that appears at the bottom (**Reset** puts every field
   back). The tab shows an unsaved dot until you save, and leaving the page asks first.
4. **Status** and the termination date are not typed in: use **Record event** beside **Status**
   (or **Employment**, then **Record employment event**) so the change lands in the history.
5. **Compliance** is graded from the worker's credentials and cannot be edited; **Qualified to
   drive**, **Available for dispatch** and **Can be assigned to equipment** are switches.
6. When a driver was home and dispatch would not otherwise know, set **Last home**.
   Dispatch also counts the end of approved time off, and telematics seeing the driver's tractor
   within 25 miles of home, as a day at home.
7. To change several workers at once, tick their rows on the roster and use **Update status**,
   **Update type** or **Update driver type** in the bar at the bottom of the table.

### Record an employment event
Keywords: hire, rehire, terminate, suspend, leave of absence, change driver type, change fleet code, pay rate change, end employment
1. Open the worker and choose **Add to record**, then **Employment event**. You can also go to
   **Employment**, open **History** and select **Record event**.
2. In **Record employment event**, pick the **Event**, set **Effective** and fill in what that
   event asks for (for example **Fleet code** or **Manager**, **Leave type**, **Driver type**,
   **Worker type**, **New rate** and **Unit**), with a **Reason** and **Notes**.
3. Read **What this will do** under the form before saving. It lists what the event changes:
   taking the worker off the dispatch board, ending the pay assignment or Dash access, starting
   a checklist, assigning required courses, cancelling upcoming time off and paying out or
   forfeiting time off.
4. Select **Record event**. The event joins the **History** list. To correct one, select
   **Amend** on it; the original stays on the record beside the amendment.
5. To end employment, use **End employment…** from **More actions**.

### Work a checklist
Keywords: onboarding checklist, offboarding checklist, checklist item
1. Open the worker, go to **Employment** and select the **Checklists** view.
2. A checklist in progress lists its items. Use an item's actions to **Complete**, **Skip**,
   **Mark not applicable** or **Reopen** it; items marked **Auto** complete themselves from
   evidence.
3. To start another, select **Start a checklist**, pick the **Template** and select **Start**.
   To stop one, select **Cancel checklist** and give a reason.

### Add or renew a credential
Keywords: CDL, medical card, TWIC, hazmat endorsement, license, certificate
1. Open [Workers](/hr/workers), select the worker's row, go to **Qualification** and open
   **Credentials**. You can also choose **Add to record**, then **Credential**.
2. Select **Add credential**, or open the actions on a credential on file and choose **Renew**,
   **Edit**, **Verify** or **Archive**. A required type that is not on file has its own add
   button in the table.
3. Pick the **Credential type**, fill in **Number**, **Issuing authority**, **Issued** and
   **Expires**, and drop the scan into **Document**.
4. Select **Add credential** (or **Save renewal** when renewing, **Save changes** when editing).
   A renewal moves the current credential to **Earlier versions**.

### Verify a credential
Keywords: verify CDL, confirm document, verified by
1. Open the worker, go to **Qualification** and open **Credentials**.
2. Select the credential, then **Verify**. A credential needs an attached document before it can
   be verified; if there is none, add one with **Edit**.
3. The credential then shows **Verified**. Changing its number, dates or document clears the
   verification.

### Find workers missing a credential
Keywords: no road test, missing medical card, credential gap, who lacks, not on file
1. Open [Workers](/hr/workers).
2. Select **Missing credential** in the toolbar and pick the credential type.
3. The table shows every worker with no valid credential of that type: none on file, archived
   or expired. The filter shows as a chip; remove the chip to see every worker again.

### Work the driver qualification file
Keywords: DQ file, DQF, previous employer, safety performance history, 391.51, 391.23
1. Open the worker, go to **Qualification** and open **DQ file**.
2. The file lists its sections: **Licenses and reviews**, **Documents**, **Employers** and
   **Drug and alcohol**, each with what is outstanding. **Next steps** names what to do, each
   with a button that does it.
3. To record a previous employer from the three years before hire, select **Add previous
   employer** and fill in the employer, contact and request details.
4. On an employer row, select **Send the request**, then **Chase again** while there is no
   answer. Record the answer with **Edit**, or **Remove** an employer added by mistake.
5. **Retention** at the bottom says how long to hold the file.

### Record a drug or alcohol collection and a Clearinghouse query
Keywords: drug test, alcohol test, random test, pre-employment test, Clearinghouse, MRO result
1. Open the worker, go to **Qualification** and open **Drug and alcohol**, or choose **Add to
   record**, then **Drug or alcohol collection** or **Clearinghouse query**.
2. Select **Record a collection**, pick the **Reason for the test** and **What is analyzed**,
   fill in when and where it was collected, and select **Record collection**.
3. When the result arrives, open the test's actions and choose **Record result**, enter the
   **Result**, **Laboratory** and **Medical review officer**, and select **Record result**.
   **Void test** removes one entered against the wrong driver.
4. Select **Log a query**, pick the **Query** type and **Requested on**, add the consent dates
   and **Reference**, and select **Log query**. When the Clearinghouse answers, choose **Record
   answer** on the query. A query that is still pending does not restart the 12-month clock.

### Assign training
Keywords: training course, required training, refresher, orientation
1. Open [Workers](/hr/workers).
2. Tick the workers who need the training and choose **Assign training** in the bar at the
   bottom of the table.
3. Pick the **Courses**, optionally a **Due date** and **Notes**, then confirm.
4. For one worker, open their row, go to **Development** and open **Training**, then use
   **Assign course** or **Assign required**. A course that is not assigned yet has **Assign**
   on its card, and a lapsed one has **Assign renewal**.

### Record a training result
Keywords: completed course, passed, failed, certificate, waive course
1. Open the worker, go to **Development** and open **Training**, or choose **Add to record**,
   then **Training result**.
2. On the course, choose **Record result**. Enter **Completed**, the **Score** and **Notes**,
   and drop the **Certificate** in.
3. Select **Record result**, or **Record fail** when the driver did not pass.
4. To close a course out without a result, choose **Waive** on it and give a reason. **Cancel
   assignment** takes it off the driver's list in Dash.

### Start a performance review
Keywords: driver review, annual review, performance review
1. Open the worker, go to **Development** and open **Reviews**.
2. Select **Start a review**, pick the **Template**, set **Period from** and **Period to**, and
   select **Start review**.
3. Rate the criteria, write the **Summary**, **Strengths** and **Where to improve**, add
   **Goals**, then **Save draft**. When it is ready, send it to the worker with the submit button.

### Record a safety event and issue an action
Keywords: accident, citation, inspection, discipline, warning, suspension, recognition, kudos
1. Open the worker and go to the **Safety** tab, or choose **Add to record**, then **Safety
   event**.
2. Select **Record event**. Pick **What happened**, **Severity**, **When** and **Where**, and
   set **Outcome**, **Level**, **Preventable** and **Points**; the suggested points count toward
   the safety score for two years. Select **Record event**.
3. To close an event, choose **Close event** on it and say what was done.
4. Select **Issue action** under **Discipline** (or on the event). Pick the **Level**, give the
   **Reason**, and select the button for that step, such as **Issue written warning** (coaching through termination).
   **Rescind…** takes an action back; it stays on the record.
5. To praise a driver, select **Add recognition**, pick the **Kind**, write a **Title** and
   **Message**, and turn on **Share with the driver** to show it in Dash.
6. Injuries and illnesses are recorded here too: under **Injuries and illnesses**, select
   **Record a case**. See [OSHA 300 log](/hr/osha-log).

### Request time off for a worker
Keywords: PTO, vacation, day off, leave request, book time off
1. Open [Workers](/hr/workers) and select the **Paid time off** tab.
2. Select the New worker PTO button above the table, pick the **Worker** and **Type**, set the
   **First day** and **Last day**, and add a **Reason**.
3. Select **Request PTO**. The request waits for approval.
4. From a worker's page, go to **Time**, open **Time off** and select **Request time off**, or
   choose **Add to record**, then **Time off request**. The form shows what the days leave of
   the balance before you save.

### Approve or reject time off
Keywords: PTO approval, deny vacation, cancel time off
1. Open [Workers](/hr/workers) and select the **Paid time off** tab.
2. Tick the requests, then choose **Approve**, **Reject** or **Cancel** in the bar at the bottom
   of the table. You can also right-click a single request.
3. Confirm the action, adding a reason where asked.
4. For one worker, go to **Time**, open **Time off** and use **Approve** or **Reject** on a
   waiting request under **Requests**, or **Cancel request…** in its options. The **Ledger**
   under it lists every accrual, use and adjustment; **Adjust balance** posts a correction.

### Open a leave case
Keywords: FMLA, medical leave, family leave, leave of absence, certification
1. Open the worker, go to **Time** and open **Leave**, or choose **Add to record**, then
   **Leave case**.
2. Select **Open a case**, pick the **Type of leave** and **How it is taken**, give the
   **Qualifying reason**, **Leave begins** and **Expected to end**, and select **Open case**.
3. On the case, choose **Approve and designate FMLA** or **Approve only**, or **Deny**. Use
   **Request certification** and **Mark received** to track the medical certification, **Record
   a day** for each day taken, and **Close** when it ends.

### Change a driver's pay profile
Keywords: assign pay, driver rate, pay package, change pay
1. Open the worker, go to **Pay** and open **Pay profile**.
2. Select **Assign profile** (or **Change profile** when one is in force).
3. Pick the profile, set **Effective** and any rate overrides, and confirm.
4. **Assignment history** lists earlier profiles; **Owed and held** shows what is unsettled.
   End the current assignment from **Pay profile options**.

### Enroll a worker in a benefit
Keywords: benefits, enroll, decline coverage, insurance
1. Open the worker, go to **Pay** and open **Benefits**.
2. Select **Enroll or decline**, pick the **Plan** and **Coverage** tier, and set **Effective
   from**. Turn on **They declined the cover** to record a waiver with the reason in **Why**.
3. Select **Enroll** (or **Record** for a waiver). **Change coverage** and **End cover…** are in
   a plan's options.

### Put a worker on a shift or a time off policy
Keywords: schedule, shift, rota, PTO policy, accrual
1. Open the worker and go to **Time**.
2. In **Schedule**, select **Put on a shift** (or **Move to another shift**), pick the **Shift**
   and **Effective from**, and confirm.
3. In **Time off**, open **Time off options** and choose **Change policy** (or **Enroll in a
   policy**) to set the policy the worker accrues under.

### Invite a driver to Dash
Keywords: driver app, portal, send invitation, revoke access, Dash
1. Open the worker, go to **Files** and open **Driver portal**.
2. Under **Dash access**, select **Send invitation**; add an email if none is on file, or
   **Copy the invitation link** when no email provider is set up.
3. The access shows **Invited** until the driver signs in, then **Linked**. **Cancel
   invitation** and **Revoke access** end it.
4. **Policies signed** lists the policies the driver has signed in Dash.

### Review a driver's profile change
Keywords: profile change request, approve change, turn down change
1. A change a driver asks for in Dash shows as an alert at the top of **Profile** with
   **Review**, and under **Profile changes** in **Driver portal**.
2. Open the request to see **What they changed**, add an **Office note**, then **Approve** it or
   **Turn down** (a note is needed to turn it down; the driver sees it).

### Read the activity trail
Keywords: audit, who changed, history of changes, change log
1. Open the worker and go to the **Activity** tab.
2. The trail lists every change to the worker and to everything in their file.
   Narrow it with **Search changes**, **Who** (**Everyone**, **You**, **From Dash** or **Any
   agent**) and **Kind of change**, and **Clear the filters** to start again.

### Add a document to a worker's file
Keywords: upload document, attach file, scan, new version, approve document, reject document, delete document
1. Open the worker, go to **Files** and open **Documents**. The table lists **Name**, **Type**,
   **Added by**, **Added** and **Size**. Narrow it with the search field (**Search documents**)
   and the **Type** filter, and use **Clear the filters** to see everything again.
2. Select **Upload**. **Upload documents** opens beside the table. Drop one or more files in or
   choose them, then select **Upload** (the button counts the files when you picked more than one).
   You can also drag files onto the table and drop them where it says to add them to the
   worker's file; the same pane opens with those files ready.
3. Select a row to read the document beside the table. A row's actions menu offers **Open**,
   **Download**, **What was read from it**, **Versions**, **Upload a new version**, **Approve**,
   **Reject…**, **Record the signature** and **Delete…**. Each is shown only where the document
   and your permissions allow it.
4. **Upload a new version** makes the new file the document's current version; the old one stays
   under **Versions**.
5. **Delete…** asks **Delete this document?** first. Confirm with **Delete document**.
6. To delete several, open **More document actions**, choose **Select documents**, tick the rows
   and use **Delete** in the bar that appears. It asks before it deletes.

### Keep a note for agents about a worker
Keywords: agent memory, remember, instruction, fact, forget note
1. Open the worker, go to **Files** and open **Agent memory**.
2. The panel, titled with what agents remember about the worker, lists each note with where it came
   from, when it was saved and what kind it is.
3. Select **Add a note**, write what agents should remember, choose **Fact** or **Rule** and
   whether it is for **Everyone** or **Just me**, then select **Remember**.
4. To change a note you may edit, select **Edit this note**, change the text and select
   **Save**.
5. To take one away, select **Remove this note** and confirm **Remove note**. Agents stop using
   it from their next run. This cannot be undone.

## Notes
- The page needs read access to workers and the asset operations feature. Adding a worker needs
  create access to workers.
- Each tab and view has its own permissions and plan feature: a view you may not read, or that is
  not on the plan, is left out, and a tab with nothing to show is not drawn. Verifying and
  archiving credentials, assigning training, recording safety events, and approving, rejecting
  or cancelling time off are only offered to people allowed to do them. **Activity** needs read
  access to the audit log.
- **Hours of service** comes from Samsara; if it is not connected the view says **Samsara is not
  connected** and links to the integrations page.
- Agents read the notes in **Agent memory** before they plan a load or message the worker. A note
  waiting for an administrator is marked as such.
- A link from a notification, an agent or a report opens the worker's page on the tab, view and
  record it means.
