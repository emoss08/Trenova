---
path: /billing/configuration-files/formula-templates
aliases: [rating formulas, rating methods, pricing formulas, charge formulas, formula studio, tariff, tariffs, per mile formula, rate calculation, property test, test coverage, find references, promote to input, lower of, best of]
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

Pricing staff write and test templates in the Formula Studio, which opens when you create a template or select one in the list. The bar across the top shows the template's name, where it is in review (**Draft**, **In review**, **Active**) and its version. On the left, the outline lists the template's inputs, charges (with each choice between methods and its options), total rules, and tests and properties (each with a green or red dot); select one to jump to it. In the middle are the name and description, the **Source**, the **Inputs** it declares and its **Tests**. On the right, **Preview**, **Tests**, **Reference** and **Nova** sit in tabs. Templates go through review before they can rate shipments.

The table shows each template's **Name**, **Status**, **Libraries** it imports, **Version**, **In use**, **Tests**, **Approved** and **Updated**.

## Tasks

### Create a formula template
Keywords: new formula, new rating method, write tariff
1. Open [Formula templates](/billing/configuration-files/formula-templates).
2. Select **New formula template**, then **New formula template** in the menu. The Formula Studio opens.
3. Type the name over **Untitled rating method** and add a description under it. The name is the `tariff "…"` line at the top of the source: typing either one changes the other.
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

### Fix a charge code your catalog doesn't have
Keywords: unknown charge code, accessorial code, FSC, add accessorial charge, charge code missing
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template.
2. A `charge … as CODE` whose code isn't in your accessorial charges is listed under the editor once, with every line that uses it. When a code in your catalog is spelled nearly the same, or its description matches the charge's label, the fix beside the problem changes every use to it.
3. To bill the line without a code, choose the fix that bills it as a freight line.
4. To add the code instead, select **Add … to accessorial charges** beside the problem (shown when you may create accessorial charges). The new charge opens with the code and the charge's label filled in; finish it and save, and the problem clears.

### Catch values a contract can never set
Keywords: never true, always true, where bounds, typo, dead code, unreachable
1. When a template compares an input with a value its `where` bounds rule out, as in `service == "Guarenteed"` when `service` may only be `"Standard"` or `"Guaranteed"`, the editor warns that the comparison is never true and offers the nearest allowed value as the fix.
2. It also warns about comparisons the bounds already decide (`rate > $20/mi` when `rate` stops at $15/mi), `match` arms the bounds rule out, and tests that set an input to a value no contract could set.

### Find every use of a name
Keywords: find references, where is this used, highlight uses, references
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template.
2. Put the cursor on a name in the source. After a moment every use of it in the file is tinted; where its value is set (its declaration, or a `var` assignment) is tinted differently from where it is read.
3. Press Shift+F12, or open **Editor actions** under the editor and choose **Find references**, to list every use with its line. Uses in this file come first, then the declaration in an imported library. Select one, or move with the arrow keys and press Enter, to go to it; a use in a library opens that library.

### Turn a number into an input
Keywords: promote to input, make it a contract input, hard-coded rate, refactor
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template.
2. Put the cursor on a number, an amount or a rate written in the source (for example `$425`), or select it.
3. Press Ctrl+. (⌘. on a Mac), or open **Editor actions** and choose **Promote to input**. The menu offers to promote that value, and when the same value is written elsewhere, every use of it at once; pick one with the arrow keys and Enter.
4. The editor adds an `input` with that value as its default after the other inputs, puts its name where the value was, and opens rename on the new name: type the name you want and press Enter. One undo takes the whole change back.
5. Where nothing can become an input (a keyword, a test, a function body), the editor says so beside the cursor and changes nothing.

### Bound what a contract can set
Keywords: input constraint, where clause, minimum rate, maximum rate, allowed values
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template.
2. Add a `where` clause to an input: a range (`input rate: Money/mi = $2.35/mi where $0.50/mi..=$15/mi`) or a list of the values it may take (`where ["Standard", "Guaranteed"]`).
3. Hover the input to see its bounds. Under **Inputs**, each bounded input shows them under its field, an input bounded by a list is a choice of those values, and a sample value outside them is refused at the field.
4. Rate agreement lanes then refuse a value outside the bounds when it is typed and when the agreement is saved.

### Bill the lower of several methods
Keywords: best, worst, lower of, greater of, per mile or flat
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template.
2. Write a `best "Label"` block with an `option` for each method (`worst` keeps the highest instead). An option that refuses, with `require` or `fail`, is left out for that load.
3. **Preview** shows each choice under the bill: which option it chose, every option's subtotal, the options that lost struck through, and why a refused option did not apply. **Trace** lists the choice where it was made, and the shipment's bill shows the same once the template rates it.

### Edit the source with Vim keys
Keywords: vim, vim mode, neovim, astronvim, jj escape, keybindings, modal editing
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template, then select **Vim** in the bar under the editor (or turn on **Vim keys in the formula editor** in [Settings](/settings) under **Preferences**). The choice stays on this device and applies to formula libraries too.
2. The editor has normal, insert and visual modes, with the mode shown under the text. Type jj or jk to leave insert mode.
3. Space is the leader, as in AstroNvim: gd goes to a definition, K shows the hover card, gl shows the line's problems and ]d or [d moves between them, Space l f formats, Space l r renames, Space l R or grr finds every use of a name, gcc or Space / comments a line, and Space w or :w saves.

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

### Check a rule over many loads with a property
Keywords: property, vary, heavier never bills less, counterexample, monotone, weight break inversion
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template.
2. Add a `property` block: `vary` a fact over a range (`vary shipment.weight in 1 lb..40_000 lb`), and `expect` what must hold for every load, such as `expect total nondecreasing in shipment.weight` (a heavier load never bills less) or `expect total >= $95`. Use `assume` to skip the loads a rule is not about.
3. Properties run with the tests on every change. In **Tests**, a property is marked **Property** and shows how many cases it rated and how many `assume` skipped.
4. When one fails, its **Counterexample** shows the simplest failing load: each varied value and, for a rule like "never bills less", the two neighbouring loads that broke it with what each billed (999 lb at $309.69, 1,000 lb at $250.00).
5. Select **Save as test** to add that load to the source as an ordinary test, so the bug stays fixed. A failing property blocks submitting for review, like a failing test.

### See what no test reaches
Keywords: coverage, untested branch, unreached, test gaps
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template.
2. A bar in the editor's margin marks every line no test or property case runs; hover it to see what part it is (a branch of an `if`, a match arm, an option, a charge, a refusal).
3. **Tests** says how many of the file's parts the tests reach. Expand the list under it to see each part none reaches, and select one to go to it.
4. Parts no test reaches are listed as a warning under **Readiness** when you submit for review. They never block it; add a test that reaches each, or check the reviewer can see why it is not needed.

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

### Adopt a newer rate matrix version
Keywords: new tariff rates, matrix published, newer table version, pinned rate table
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template.
2. Under **Rate tables**, each table the source reads shows the published version tests and the preview read, the version the template pins, and **Newer version** when a later version of the matrix has been published. **Open matrix** opens it on [Rate matrices](/billing/configuration-files/rate-matrices).
3. Save the template and send it through review. Approving the new template version pins the newest published version of every table; until then, contracts keep pricing with the version they pinned.

### Edit an active template
Keywords: change tariff, update rating method
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template. Hover the count of places it is in use to see them.
2. Make your changes and select **Save changes**.
3. If the change alters the source, confirm with **Save and return to Draft**; the approved version keeps rating shipments until the new one is approved. Description edits do not need this.

### See versions or roll back
Keywords: version history, rollback, compare versions, schedule activation
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template.
2. Open **More actions** and choose **Version history**.
3. Each version lists the rate table versions it pins. Use **Compare previous** to see how the source and the pinned libraries changed, **Schedule activation** to set when a version takes effect, or **Rollback** to restore an earlier version.

### Copy, fork, export or import templates
Keywords: duplicate tariff, fork, export JSON, import templates, standard templates
1. Open [Formula templates](/billing/configuration-files/formula-templates).
2. Right-click a template for **Fork template**, **View lineage**, **Duplicate**, **Export** or **Archive**, or tick several and use **Duplicate**, **Export** or **Archive** in the bar that appears.
3. To bring templates in, select **New formula template** and choose **Import templates** (from a file exported from Trenova) or **Install standard templates** (Trenova's standard rating methods, each with its own tests).

## Notes
Viewing the page needs read access to formula templates, and creating one needs create access. **Submit for review**, **Approve**, **Request changes** and **Reject** each need their own permission and only appear for people who have it. Someone who submitted a template, or edited the revision under review, cannot approve it unless their role has the "Approve own changes" permission for formula templates, which administrators have; an approval made that way is recorded in the audit log as approved without a second reviewer.

Nova needs permission to start assistant conversations and access to the Formula assistant agent, on top of formula template access; without them the Nova tab says which is missing. Every amount Nova shows comes from the same rating engine that bills shipments.

Properties are tests: they run on every change, show as dots in the outline and must pass before review. Coverage counts the template's own file; a library's parts are covered by the library's tests.

Only an active template rates shipments. A template version pins the version of every library it imports and of every rate matrix it reads, so changing a library or publishing a matrix version never changes what an approved template bills. Rejecting archives the template; use **Request changes** to send it back for more work instead. Submitting needs a name, a description, a source with no errors and every test passing.
