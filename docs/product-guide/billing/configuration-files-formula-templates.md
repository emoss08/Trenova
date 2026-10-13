---
path: /billing/configuration-files/formula-templates
aliases: [rating formulas, rating methods, pricing formulas, charge formulas, formula studio, per mile formula, rate calculation]
related:
  - /billing/rate-agreements
  - /billing/configuration-files/rate-matrices
  - /billing/configuration-files/accessorial-charges
  - /billing/queue
covers:
  - /billing/configuration-files/formula-templates/new
  - /billing/configuration-files/formula-templates/:id/edit
---

## What it's for
Formula templates are the rating methods that turn shipment facts (distance, weight, stops, hazmat and so on) into a charge, for example a per-mile rate or a per-hundredweight rate. Each template is a formula (its **Expression**), plus a charge policy (minimum, maximum and rounding) and an optional charge breakdown. Rate agreement lanes and rate matrices name a template as their **Rating method**.

Pricing staff build and test templates in the Formula Studio, which opens when you create a template or select one in the list. The bar across the top shows the template's name, where it is in review (**Draft**, **In review**, **Active**) and its version. On the left, the **Rate trace** follows one load through the template: the values it reads, what the **Formula** comes to, what the **Charge policy** does to it, the **Breakdown**, and what is **Billed**, with the scenarios and what review still needs below it. On the right, **Preview**, **Scenarios**, **Reference** and **Nova** sit in tabs. Templates go through review before they can rate shipments.

The table shows each template's **Name**, **Status**, **Type**, **Version**, **In use**, **Scenarios**, **Approved** and **Updated**.

## Tasks

### Create a formula template
Keywords: new formula, new rating method, write formula
1. Open [Formula templates](/billing/configuration-files/formula-templates).
2. Select **New formula template**, then **New formula template** in the menu. The Formula Studio opens.
3. Type the name over **Untitled rating method**, add a description under it, and choose **Freight** or **Accessorial** beside **Charges**.
4. Under **Expression**, describe the method and select **Draft it** to have Nova write it, or pick one from **Standard library** or **Your templates**, or write the expression yourself.
5. Add any **Inputs** the formula needs, set the **Charge policy** (**Minimum charge**, **Maximum charge**, **Rounding**, **Decimal places**) and check the amount in **Preview**.
6. Optionally add **Scenarios**; they are saved with the template.
7. Select **Create template**. The template is saved as a draft.

### Get help from Nova
Keywords: AI formula, generate formula, explain formula, ask about a formula, fix formula
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template, or start a new one.
2. Select **Ask Nova** beside **Expression**, or the **Nova** tab, and describe how the template should price a shipment, what to change, or ask about the formula in the editor.
3. Select **Explain** to have each line of the formula explained, priced on the preview load. Select **Fix with Nova** beside a problem to have Nova fix it. The Nova button under **Inputs** fills in descriptions the inputs are missing.
4. Ask for any change in words — add a fee, set a minimum, change an input's default, itemize the invoice — and Nova makes it in the editor the moment it answers. The card under the reply shows what it added and removed, the inputs it declared, the charge policy it moved, and what the formula engine bills the sample load and recent shipments; select **Undo** to put the editor back. A change the engine cannot run stays in the thread with **Apply to editor** instead. Nothing is saved until you save the template.

### Read what the editor says about a formula
Keywords: formula error, unknown variable, declare input, inline values, hover
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template. The editor underlines anything wrong — a name it does not know, a function given the wrong number of arguments, a flag multiplied like a number, a field that can be empty — and lists each problem under the editor with its line and a fix beside it: declare the name as an input, change it to the field it meant, or give an empty field a fallback. Select a row to jump to it; a long list folds to its first rows and can be narrowed to errors or warnings.
2. Hover a name to see what it is, whether it can be empty, and what it was worth on the preview load; type inside a function's parentheses to see its signature. Each line shows what it came to on the preview load at its end; the eye button under the editor hides these values.
3. Press Ctrl+Space for suggestions.
4. The format button under the editor (or Shift+Alt+F) lays the formula out one statement per line with a space around every operator, breaking a long sum one term per line; what Nova writes lands already formatted. A formula that does not parse is left as it is until it does.

### Test a template with scenarios
Keywords: test formula, expected charge, backtest, pin scenario, edge cases
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a template.
2. In **Preview**, select **Pin as scenario** to save the load and its amount, or in **Scenarios** select **Add** and name it. **Suggest edge cases** asks Nova for loads to test, priced by the engine.
3. Scenarios re-run as you edit. On a failing scenario, select **Why?** to see what changed since it was recorded, then accept the new amount or select **Fix with Nova**.
4. To compare the formula against real freight, open **More actions** and choose **Backtest**.

### Send a template through review
Keywords: approve formula, activate template, submit template, reviewer note
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template.
2. Save any changes, then select **Submit for review**.
3. Check the readiness list, choose a reviewer, edit the note Nova drafts (or select **Rewrite**), and select **Submit**.
4. A reviewer selects **Approve** to activate it or **Request changes** to send it back to the author with a comment. **Reject and archive** is under **More actions**.

### Edit an active template
Keywords: change formula, update rating method
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template. Hover the count of places it is in use to see them.
2. Make your changes and select **Save changes**.
3. If the change alters what the template computes, confirm with **Save and return to Draft**; the approved version keeps rating shipments until the new one is approved. Name and description edits do not need this.

### See versions or roll back
Keywords: version history, rollback, compare versions, schedule activation
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template.
2. Open **More actions** and choose **Version history**.
3. Use **Compare previous** to see what changed, **Schedule activation** to set when a version takes effect, or **Rollback** to restore an earlier version.

### Copy, fork, export or import templates
Keywords: duplicate formula, fork, export JSON, import templates, standard templates
1. Open [Formula templates](/billing/configuration-files/formula-templates).
2. Right-click a template for **Fork template**, **View lineage**, **Duplicate**, **Export** or **Archive**, or tick several and use **Duplicate**, **Export** or **Archive** in the bar that appears.
3. To bring templates in, select **New formula template** and choose **Import templates** (from an exported JSON file) or **Install standard templates** (the standard rating library).

## Notes
Viewing the page needs read access to formula templates, and creating one needs create access. **Submit for review**, **Approve**, **Request changes** and **Reject** each need their own permission and only appear for people who have it.

Nova needs permission to start assistant conversations and access to the Formula assistant agent, on top of formula template access; without them the Nova tab says which is missing. Every amount Nova shows comes from the formula engine.

Only an active template rates shipments. Rejecting archives the template; use **Request changes** to send it back for more work instead. Submitting needs a name, a description, a valid expression and at least one passing scenario. A scenario keeps gating approval until it is deleted.
