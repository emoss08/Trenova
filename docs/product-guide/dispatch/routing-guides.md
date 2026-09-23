---
path: /dispatch/routing-guides
aliases: [carrier waterfall, tender waterfall, carrier ladder, preferred carriers by lane, lane carriers, auto-tender, carrier ranking]
related:
  - /dispatch/console
  - /dispatch/carriers
  - /dispatch/locations
---

## What it's for
A routing guide is a ranked list of carriers for a lane. When a move on that lane is tendered, the carriers are offered the load in rank order (the waterfall): rank 1 first, then down the list. Each entry sets the carrier, its rate and rate method, how long it holds the offer, and whether the offer goes by email or EDI. Carrier sales and brokerage dispatch staff build these for their contract lanes.

A lane can be described three ways: **Exact locations** (one facility pair), **City + state** (a metro lane) or **State only** (a regional lane). When a move tenders, the most specific active guide that matches wins.

## Tasks

### Create a routing guide
Keywords: new waterfall, set up lane carriers, rank carriers for a lane
1. Open [Routing guides](/dispatch/routing-guides) and select **New routing guide**.
2. Under **Identity**, enter a **Name**, set the **Status**, and add a **Description**.
3. Under **Lane**, pick **Exact locations**, **City + state** or **State only**, and fill in the matching fields: **Origin location** and **Destination location**, **Origin city** / **Origin state** and **Destination city** / **Destination state**, or just the two states.
4. Under **Carrier waterfall**, select **Add carrier** for each carrier and set its **Carrier**, **Rank**, **Rate method**, **Rate**, **Offer expiry** and **Channel**. Turn on **Price from the contract** to offer whatever the carrier's contract says today.
5. Select **Save**.

### Change the carrier order
Keywords: rerank carriers, move carrier up, add backup carrier
1. Open [Routing guides](/dispatch/routing-guides) and select the guide.
2. Under **Carrier waterfall**, change each carrier's **Rank**, add carriers with **Add carrier**, or take one out with **Remove**.
3. Select **Save**.

### Stop using a routing guide
Keywords: deactivate guide, retire waterfall, delete routing guide
1. Open [Routing guides](/dispatch/routing-guides) and select the guide.
2. Change its **Status** so it is no longer Active and select **Save**. Only Active guides are matched when a move is tendered.
3. Or delete it with the delete button in the panel header and confirm with **Delete guide**.

### Tender a move with its routing guide
Keywords: start waterfall, send load to carriers
1. Open [Console](/dispatch/console), select an uncovered move, and select **Tender to carriers**.
2. On **Waterfall**, keep **Use the matched guide** or choose one under **Override guide**, and select **Start waterfall**.

## Notes
The page is available only to organizations with brokerage turned on, and needs read access to routing guides; creating, editing and deleting need the matching permissions.

An entry sent by EDI requires a default EDI channel on the carrier. Both ends of the lane must be described at the same level.
