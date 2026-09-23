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

Pricing staff build and test templates in the Formula Studio, which opens when you create a template or select one in the list. The studio shows a **Live preview** of the charge as you type, pinned test **Scenarios**, and a **Reference** of the variables and functions you can use. Templates go through review before they can rate shipments.

The table shows each template's **Name**, **Status**, **Type**, **Version**, **In use**, **Scenarios**, **Approved** and **Updated**.

## Tasks

### Create a formula template
Keywords: new formula, new rating method, write formula
1. Open [Formula templates](/billing/configuration-files/formula-templates).
2. Select **New formula template**, then **New formula template** in the menu. The Formula Studio opens.
3. Under **Template details**, enter a **Name**, choose the **Type** (**Freight charge** or **Accessorial charge**) and add a **Description**.
4. Write the **Expression**, or pick a template under **Start from a standard** or **Or copy an existing template**. You can also select **Generate with AI**, describe the charge, then **Insert into editor**.
5. Set the **Charge policy** (**Minimum charge**, **Maximum charge**, **Rounding mode**, **Rounding precision**) and check the result in **Live preview**.
6. Select **Create template**. The template is saved as a draft.

### Test a template with scenarios
Keywords: test formula, expected charge, backtest, pin scenario
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select a saved template.
2. In **Scenarios**, select **Add**, enter a **Name**, the **Input values** and the **Expected charge ($)**, then **Save scenario**. From **Live preview**, **Pin as scenario** saves the current inputs and result as one.
3. Select **Run all** to check every scenario against the current formula.
4. To compare the formula against real freight, open **More actions** and choose **Backtest**, then **Run backtest**.

### Send a template through review
Keywords: approve formula, activate template, submit template
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template.
2. Save any changes, then select **Submit for review**.
3. A reviewer selects **Approve** to activate it, **Request changes** to send it back to the author with a comment, or **Reject** to archive it.

### Edit an active template
Keywords: change formula, update rating method
1. Open [Formula templates](/billing/configuration-files/formula-templates) and select the template. The header shows where it is used.
2. Make your changes and select **Save changes**.
3. If the change alters what the template computes, confirm with **Save and return to Draft**; the template stops rating shipments until it is approved again. Name and description edits do not need this.

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

Only an active template rates shipments. Rejecting archives the template; use **Request changes** to send it back for more work instead. **Scenarios** can only be added after the template is saved, and a scenario keeps gating approval until it is deleted.
