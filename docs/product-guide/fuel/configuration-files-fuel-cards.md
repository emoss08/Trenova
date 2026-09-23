---
path: /fuel/configuration-files/fuel-cards
aliases: [fuel card list, Comdata cards, EFS cards, WEX cards, driver fuel cards, card registry]
related:
  - /fuel/unassigned-cards
  - /fuel/purchases
  - /fuel/feed-runs
  - /equipment/tractors
  - /hr/workers
---

## What it's for
Fuel cards is the register of the cards drivers fuel with: which provider issued each one (Comdata, EFS, WEX or other), who carries it, and which tractor it is tied to. Imported fuel card statements are matched to purchases through these cards, so a card should be registered and assigned before its statement is imported.

Fleet and fuel staff maintain the list. The table shows each card's **Status**, **Label**, **Provider**, **Card** (last four digits), **Worker**, **Tractor**, **Expires** and **Updated**.

## Tasks

### Register a fuel card
Keywords: add fuel card, new card, issue card to driver
1. Open [Fuel cards](/fuel/configuration-files/fuel-cards).
2. Select **New fuel card**.
3. Under **Card**, choose the **Provider**, enter the **Last four digits**, a **Label** (how the card is named in pickers and on purchases), the **Status**, and optionally the **Provider card ID** and **Expires** date.
4. Under **Assignment**, pick the **Driver** who normally carries it and the **Tractor** it lives in, and add any **Notes**.
5. Select **Save**.

### Assign or reassign a card
Keywords: give card to driver, move card to tractor, tie card to unit
1. Open [Fuel cards](/fuel/configuration-files/fuel-cards).
2. Right-click the card's row and choose **Assign card**.
3. Pick the **Tractor**, the **Driver**, or both, and select **Assign**. Purchases on the card then match to what it is assigned to. Leaving both empty puts the card back on [Unassigned cards](/fuel/unassigned-cards).

### Cancel a lost or retired card
Keywords: deactivate card, lost card, stolen card
1. Open [Fuel cards](/fuel/configuration-files/fuel-cards).
2. Right-click the card's row and choose **Cancel card**.
3. Enter a **Reason** and select **Cancel card**. The card stays on file for the purchases already made with it.

### Find a card
Keywords: search fuel cards, card by driver, card last four
1. Open [Fuel cards](/fuel/configuration-files/fuel-cards).
2. Type in the search box, or use **Filter** and **Sort** in the table toolbar.

## Notes
This page is only available when your organization runs its own trucks (asset operations). Viewing it needs read access to fuel cards; **Assign card** and **Cancel card** need permission to update fuel cards, and neither is offered on a card that is already cancelled.

Only the last four digits of a card are stored.
