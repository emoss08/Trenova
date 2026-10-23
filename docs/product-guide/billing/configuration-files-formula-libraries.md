---
path: /billing/configuration-files/formula-libraries
aliases: [tariff libraries, shared functions, shared rating functions, org libraries, formula modules, library properties, library coverage, find references]
related:
  - /billing/configuration-files/formula-templates
  - /billing/configuration-files/accessorial-charges
covers:
  - /billing/configuration-files/formula-libraries/new
  - /billing/configuration-files/formula-libraries/:id/edit
---

## What it's for
A formula library holds Tariff functions, constants, types and enums that several formula templates share: how your extra stops are billed, your hazmat fee, your delivery zones, your fuel surcharge table. A template imports a library by its path, `org.` and the library's name (`use org.accessorials.{ extraStops }`), so the rule is written once and every template that imports it bills it the same way. A library carries its own tests, which pin what each function returns.

Libraries are versioned and reviewed like templates. When a template version is approved it pins the approved version of each library it imports, so editing, approving or archiving a library never changes what an already approved template bills; a template picks up a new library version only when a new version of the template is approved.

The table shows each library's **Module** path and description, **Status**, **Version**, **Approved**, **Tests**, the other libraries it **Imports** and **Updated**.

## Tasks

### Create a formula library
Keywords: new library, shared functions, write library
1. Open [Formula libraries](/billing/configuration-files/formula-libraries).
2. Select **New formula library**. The library studio opens.
3. Type the library's name after `org.`: one word starting with a lowercase letter, such as `accessorials` or `zoneRates`. Templates import it by this name. Add a description under it.
4. Write the source, or select **Use the starter** for a small working library to rewrite. The source starts with `library "org.<name>"`; the editor warns when the header names something else.
5. Add a `test` block for every function; a library needs at least one test before it can be reviewed.
6. Select **Create library**. The library is saved as a draft.

### Write and test a library
Keywords: library editor, library tests, failing test
1. Open [Formula libraries](/billing/configuration-files/formula-libraries) and select a library. The editor checks the source as you type, with the same suggestions, hover, go to definition (F12), find references (Shift+F12), rename (F2) and formatting as the Formula Studio.
2. The outline on the left lists the library's functions, constants and tests; a test's dot is green when it passes and red when it fails. **Tests** in the side panel shows each test with the values behind a failing expectation; select one to jump to it, or **Add a test** to start one.
3. **Reference** lists what the library can use: Tariff's standard library, your other libraries, your rate tables and charge codes.
4. Select **Ask Nova** above the source, or the **Nova** tab, to have Nova add functions, write tests or fix what the editor underlines. Each change lands in the editor with the lines it changed and whether it checks and its tests pass; **Undo** puts the editor back. Nova only puts a change in the editor when it checks and every test passes; otherwise it says what stops it. Nothing is saved until you save.

### Check a function over many values with a property
Keywords: property, vary, library property test, counterexample
1. Open [Formula libraries](/billing/configuration-files/formula-libraries) and select a library.
2. Add a `property` block that varies a new name over a range or a list (`vary stops in 0..20`) and states what every value must satisfy, such as `expect extraStops(stops) >= $0`. A library's property varies its own names; it has no shipment to vary.
3. In **Tests**, a property is marked **Property** with the number of cases it ran. When one fails, its **Counterexample** shows the simplest failing value; select **Save as test** to add it to the source as a test. A failing property blocks review like a failing test.

### See what no test reaches
Keywords: library coverage, untested function, unreached
1. Open [Formula libraries](/billing/configuration-files/formula-libraries) and select a library.
2. A bar in the editor's margin marks every line no test or property case runs, such as a function no test calls; hover it to see what part it is.
3. **Tests** says how many of the library's parts its tests reach; expand the list to go to each part none reaches. They are listed as a warning under **Readiness** when you submit, and never block it.

### Find every use of a name
Keywords: find references, where is this used
1. Open [Formula libraries](/billing/configuration-files/formula-libraries) and select a library.
2. Put the cursor on a name: after a moment its uses in the file are tinted, where it is declared or set differently from where it is read.
3. Press Shift+F12, or open **Editor actions** under the editor and choose **Find references**, to list each use with its line; select one to go to it.

### See which templates import a library
Keywords: library usage, who uses this library, pinned version
1. Open [Formula libraries](/billing/configuration-files/formula-libraries) and select a library.
2. Open **Imported by** in the side panel. Each template version that imports the library is listed with its status and the library version it pinned; select a template to open it.

### Send a library through review
Keywords: approve library, submit library, activate library
1. Open [Formula libraries](/billing/configuration-files/formula-libraries) and select the library.
2. Save any changes, then select **Submit for review**.
3. Check the readiness list and what changed since the approved version, choose a reviewer, add a note and select **Submit**.
4. A reviewer selects **Approve** to make it the version new template versions pin, or **Request changes** to send it back with a comment. **Reject and archive** is under **More actions**.

### Edit an approved library
Keywords: change library, update shared function
1. Open [Formula libraries](/billing/configuration-files/formula-libraries) and select the library.
2. Make your changes and select **Save changes**. A change to the source or the name saves a new version and returns the library to **Draft**; confirm with **Save as draft**.
3. Templates keep billing with the version they pinned. Once the new version is approved, a template uses it when a new version of the template is approved.

### See versions or roll back
Keywords: library history, compare library versions, restore library version
1. Open [Formula libraries](/billing/configuration-files/formula-libraries) and select the library.
2. Open **History** in the side panel for every version and the review conversation.
3. Select the compare button on a version to see its source against another, or the restore button to bring an earlier version back as a new draft version (save or discard your changes first).

### Duplicate or archive a library
Keywords: copy library, archive library
1. Open [Formula libraries](/billing/configuration-files/formula-libraries).
2. Right-click a library for **Duplicate** or **Archive**, or tick several and use the bar that appears. In a library, both are under **More actions**.
3. Archiving a library templates still import asks you to confirm and lists them. They keep the version they pinned, but no new template version can import an archived library.

## Notes
Viewing the page needs read access to formula libraries, and creating one needs create access. **Duplicate**, **Archive**, **Submit for review**, **Approve**, **Request changes** and **Reject** each need their own permission and only appear for people who have it. The person who submitted a library cannot approve it.

Submitting needs a description, a source with no errors and at least one test, all passing. Renaming a library that templates import changes the path they import it by: their approved versions keep working, but a new version of each must import the new name.
