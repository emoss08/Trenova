---
path: /accounting/journal-reversals
aliases: [reverse journal entry, reversing entry, undo journal entry, JE reversal, backout entry]
related:
  - /accounting/manual-journals
  - /accounting/reports/trial-balance
covers:
  - /accounting/journal-entries/:id
---

## What it's for
Journal reversals are requests to back out a journal entry that has already been posted to the
general ledger. The page lists every reversal request with its **Status**,
**Original journal entry**, **Reason code**, **Reason** and **Created At**.

A reversal request starts as **Requested**, can go to pending approval, is **Approved** or
**Rejected**, and an approved reversal is posted to the ledger (**Posted**). Accountants raise the
requests; approvers review and post them.

The page also leads to the journal entry detail page, which shows an entry's number, type, status,
accounting date, **Reference type**, **Reference**, whether it **Is reversal**, the entry it is a
**Reversal Of** or is **Reversed By**, and its **Line items** with totals.

## Tasks

### Request a journal entry reversal
Keywords: reverse JE, undo posting, back out entry
1. Open [Journal reversals](/accounting/journal-reversals).
2. Select **New journal reversal**.
3. Enter the **Original journal entry ID** of the posted entry and the **Requested accounting date**
   the reversal should post on.
4. Enter a **Reason code** (for example ERROR or DUPLICATE) and a detailed **Reason**.
5. Select **Create reversal**.

### Approve or reject a reversal
Keywords: review reversal, sign off reversal
1. Open [Journal reversals](/accounting/journal-reversals).
2. Select a reversal whose **Status** is pending approval.
3. Under **Actions**, select **Approve**, or select **Reject**, type the reason and select
   **Confirm reject**.

### Post an approved reversal
Keywords: post reversing entry
1. Open [Journal reversals](/accounting/journal-reversals).
2. Select an **Approved** reversal.
3. Under **Actions**, select **Post**. The panel then shows the **Reversal entry** that was posted.

### Cancel a reversal request
Keywords: withdraw reversal
1. Open [Journal reversals](/accounting/journal-reversals).
2. Select a reversal that is requested, pending approval or approved.
3. Select **Cancel reversal**, type the reason, then select **Confirm cancel**.

### Look at the original journal entry
Keywords: view journal entry, JE detail, entry lines
1. Open [Journal reversals](/accounting/journal-reversals).
2. Select the ID in the **Original journal entry** column. The journal entry page opens with its
   details and **Line items**.
3. On the journal entry page, select the **Reference** link to open the record that produced the
   entry, or select **Back** to return.

## Notes
Opening the page needs read access to journal reversals. Opening a journal entry page needs read
access to journal entries.

A reversal does not delete the original entry: posting it adds a reversing entry, and the original
entry then shows who it was **Reversed By**.
