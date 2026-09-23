---
path: /fuel/unassigned-cards
aliases: [unregistered fuel cards, unknown cards, unmatched fuel cards, cards to assign]
related:
  - /fuel/configuration-files/fuel-cards
  - /fuel/feed-runs
  - /fuel/purchases
---

## What it's for
Unassigned cards lists the fuel cards a connected fuel card feed saw a transaction on before anybody had registered them. Until such a card is tied to a tractor or driver, its purchases can only be matched by the unit number on the receipt, and rows that cannot be matched wait on [Feed runs](/fuel/feed-runs).

Fuel and fleet staff work this list down by assigning each card. It shows the same columns as [Fuel cards](/fuel/configuration-files/fuel-cards): **Status**, **Label**, **Provider**, **Card**, **Worker**, **Tractor**, **Expires** and **Updated**. There is no create button here; cards appear on their own.

## Tasks

### Assign a card the feed found
Keywords: register card from feed, match fuel card, tie card to tractor
1. Open [Unassigned cards](/fuel/unassigned-cards).
2. Right-click the card's row and choose **Assign card**.
3. Pick the **Tractor**, the **Driver**, or both, and select **Assign**. The card leaves this list and its purchases start matching on their own.

### Fill in a card's details
Keywords: label card, edit unassigned card
1. Open [Unassigned cards](/fuel/unassigned-cards) and select the card's row.
2. Give it a **Label**, check the **Provider** and **Last four digits**, set the **Driver** and **Tractor** under **Assignment**, and select **Save**.

### Cancel a card you don't recognise
Keywords: reject card, block card, unknown card
1. Open [Unassigned cards](/fuel/unassigned-cards).
2. Right-click the card's row and choose **Cancel card**, enter a **Reason** and select **Cancel card**.

## Notes
This page is only available when your organization runs its own trucks (asset operations). Viewing it needs read access to fuel cards; assigning and cancelling need permission to update them.
