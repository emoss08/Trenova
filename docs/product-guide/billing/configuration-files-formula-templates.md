---
path: /billing/configuration-files/formula-templates
aliases: [rating formulas, rating methods, pricing formulas, charge formulas, formula studio, tariff, tariffs, per mile formula, rate calculation]
related:
  - /billing/rate-agreements
  - /billing/configuration-files/formula-libraries
  - /billing/configuration-files/rate-matrices
  - /billing/configuration-files/accessorial-charges
  - /billing/queue
covers:
  - /billing/configuration-files/formula-templates/new
  - /billing/configuration-files/formula-templates/:id/edit
---

## What it's for
Formula templates are the rating methods that turn a shipment into a bill: a per-mile linehaul with a fuel surcharge, a per-hundredweight LTL rate, a fee for every stop past the second. Each template is one file written in Tariff, Trenova's rating language. The file declares the inputs a contract sets (a rate, a fuel percent, a minimum), the charges it bills (each with its label and, for an accessorial, its charge code), the total rules (a minimum, a maximum, rounding) and its own tests. A template can import functions from your organization's [formula libraries](/billing/configuration-files/formula-libraries) and from Tariff's standard library. Rate agreement lanes and rate matrices name a template as their **Rating method**, and each sets the template's inputs.

Pricing staff write and test templates in the Formula Studio, which opens when you create a template or select one in the list. The bar across the top shows the template's name, where it is in review (**Draft**, **In review**, **Active**) and its version. On the left, the outline lists the template's inputs, charges, total rules and tests (each test with a green or red dot); select one to jump to it. In the middle are the name and description, the **Source**, the **Inputs** it declares and its **Tests**. On the right, **Preview**, **Tests**, **Reference** and **Nova** sit in tabs. Templates go through review before they can rate shipments.

The table shows each template's **Name**, **Status**, **Libraries** it imports, **Version**, **In use**, **Tests**, **Approved** and **Updated**.

## Tasks

### Create a formula template
Keywords: new formula, new rating method, write tariff
1. Open [Formula templates](/billing/configuration-files/formula-templates).
2. Select **New formula template**, then **New formula template** in the menu. The Formula Studio opens.
3. Type the name over **Untitled rating method** and add a description under it.
4. Under **Source**, describe the method and select **Draft it** to have Nova write it, or pick one from **Standard library** or **Your templates**, or write the tariff yourself.
5. Check what it bills in **Preview** and that its **Tests** pass.
6. Select **Create template**. The template is saved as a draft.

### Write and read the source
Keywords: tariff editor, formula error, autocomplete, hover, rename, go to definition, format
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template. The editor underlines anything wrong (a name it does not know, a weight added to a distance, a field that can be empty used without a fallback, a charge code your organization does not have) and lists each problem under the editor with a fix beside it where there is one. Select a row to jump to it.
2. Press Ctrl+Space for suggestions: the shipment's fields, units after a number, your rate tables and charge codes, and the functions of the libraries you import.
3. Hover a name to see its type, its documentation and what it came to on the preview load. Type inside a function's parentheses to see its parameters. Inferred types and parameter names are drawn in the text in grey.
4. Press F12 (or Ctrl-click) on a name to jump to where it is declared, including into a library it comes from, and F2 to rename a binding everywhere it is used.
5. The format button under the editor (or Shift+Alt+F) lays the file out in Tariff's one style. A file with syntax errors is left as it is until it parses.

### Edit the source with Vim keys
Keywords: vim, vim mode, neovim, astronvim, jj escape, keybindings, modal editing
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template, then select **Vim** in the bar under the editor (or turn on **Vim keys in the formula editor** in [Settings](/settings) under **Preferences**). The choice stays on this device and applies to formula libraries too.
2. The editor has normal, insert and visual modes, with the mode shown under the text. Type jj or jk to leave insert mode.
3. Space is the leader, as in AstroNvim: gd goes to a definition, K shows the hover card, gl shows the line's problems and ]d or [d moves between them, Space l f formats, Space l r renames, gcc or Space / comments a line, and Space w or :w saves.

### Preview what a template bills
Keywords: preview charge, test on a shipment, bill, trace
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template. **Preview** bills the **Sample load** with the inputs' sample values, which you can change under **Inputs**; select **Real shipment** to bill a saved shipment instead.
2. The bill lists each charge with its code, quantity, rate and amount, the minimum or rounding adjustments, and the freight, accessorial and grand totals. If the tariff refuses to rate the load, the preview says why in the tariff's own words.
3. Open **Trace** to follow how the bill was reached, step by step; select a step to light its line in the source.

### Test a template
Keywords: test tariff, expected charge, save as test, suggest tests, backtest
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template. Its tests are the `test` blocks in the source; they run as you edit, and **Tests** shows each one passing or failing, with the values behind a failing expectation. Select a failing test to jump to the expectation that failed, or **Why does this fail?** to have Nova say whether the tariff or the test looks wrong.
2. In **Preview**, select **Save as test** to add a test that pins what the load is billed. On a real shipment the test carries that shipment's facts.
3. Select **Add a test** under **Tests** to start one by hand, or **Suggest tests** to have Nova write tests from real loads on the template's edges. Each suggestion has already run against the template: select **Add** beside one, or add every one that passes at once.
4. To compare the template against real freight, open **More actions** and choose **Backtest**.

### Get help from Nova
Keywords: AI tariff, generate tariff, explain tariff, fix tariff
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template, or start a new one.
2. Select **Ask Nova** beside **Source**, or the **Nova** tab, and describe what to change or ask about the tariff in the editor.
3. Select **Explain** above the source to have each statement explained on the preview load; hover an explanation to light its line. Select **Ask Nova** beside a problem to have Nova fix it.
4. Ask for any change in words (add a weekend fee, set a minimum, add tests) and Nova makes it in the editor the moment it answers. The card under the reply shows the lines it added and removed, whether the result checks and its tests pass, and what it bills the sample load and recent shipments; select **Undo** to put the editor back. Nova only puts a change in the editor when it checks and every test passes; otherwise it says what stops it. Nothing is saved until you save the template.
5. Under **Inputs**, the Describe button (it counts the inputs with no label) has Nova write the label, the doc comment, of each of them. One undo takes them all back.

### Send a template through review
Keywords: approve tariff, activate template, submit template, reviewer note
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template.
2. Save any changes, then select **Submit for review**.
3. Check the readiness list, choose a reviewer, edit the note Nova drafts (or select **Rewrite**), and select **Submit**.
4. A reviewer sees what changed since the approved version, the libraries it pins and what the change does to recent freight, then selects **Approve** to activate it or **Request changes** to send it back to the author with a comment. **Reject and archive** is under **More actions**.

### Edit an active template
Keywords: change tariff, update rating method
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template. Hover the count of places it is in use to see them.
2. Make your changes and select **Save changes**.
3. If the change alters the source, confirm with **Save and return to Draft**; the approved version keeps rating shipments until the new one is approved. Description edits do not need this.

### See versions or roll back
Keywords: version history, rollback, compare versions, schedule activation
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template.
2. Open **More actions** and choose **Version history**.
3. Use **Compare previous** to see how the source and the pinned libraries changed, **Schedule activation** to set when a version takes effect, or **Rollback** to restore an earlier version.

### Copy, fork, export or import templates
Keywords: duplicate tariff, fork, export JSON, import templates, standard templates
1. Open [Formula templates](/billing/configuration-files/formula-templates).
2. Right-click a template for **Fork template**, **View lineage**, **Duplicate**, **Export** or **Archive**, or tick several and use **Duplicate**, **Export** or **Archive** in the bar that appears.
3. To bring templates in, select **New formula template** and choose **Import templates** (from a file exported from Trenova) or **Install standard templates** (Trenova's standard rating methods, each with its own tests).

## Notes
Viewing the page needs read access to formula templates, and creating one needs create access. **Submit for review**, **Approve**, **Request changes** and **Reject** each need their own permission and only appear for people who have it.

Nova needs permission to start assistant conversations and access to the Formula assistant agent, on top of formula template access; without them the Nova tab says which is missing. Every amount Nova shows comes from the same rating engine that bills shipments.

Only an active template rates shipments. A template version pins the version of every library it imports, so changing a library never changes what an approved template bills. Rejecting archives the template; use **Request changes** to send it back for more work instead. Submitting needs a name, a description, a source with no errors and every test passing.
