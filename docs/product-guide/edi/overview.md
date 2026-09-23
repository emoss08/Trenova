---
path: /edi/overview
aliases: [EDI dashboard, EDI health, EDI operations, EDI status, EDI monitoring]
related:
  - /edi/messages
  - /edi/inbound-files
  - /edi/transfers/inbound
  - /edi/partners
---

## What it's for
The EDI operations page is the live health view of electronic data interchange for your organization. It counts documents that need attention (dead-lettered messages, quarantined files, stuck transfers, overdue acknowledgments), shows pipeline state, charts document volume and delivery success over time, scores each trading partner, and lists the most recent failures.

EDI coordinators and support teams use it to spot problems with partner deliveries and inbound processing and to jump straight to the record that needs work.

## Tasks

### Check EDI health for a time window
Keywords: EDI failures today, delivery success rate, how is EDI doing
1. Open [Overview](/edi/overview).
2. Pick a window from the range buttons at the top right: **4h**, **24h**, **7d**, **30d** or **All**. The page opens on **24h**.
3. Read the **Needs attention** tiles (**Dead-lettered messages**, **Quarantined files**, **Stuck transfers**, **Overdue acknowledgments**) and the **Pipeline state** tiles (**Failed deliveries**, **Partially processed files**, **Pending approval**, **Rejected acknowledgments**).
4. Select a tile to open the matching list: message tiles open [Messages](/edi/messages), file tiles open [Inbound files](/edi/inbound-files), and transfer tiles open [Inbound transfers](/edi/transfers/inbound).

### Compare trading partner performance
Keywords: partner scorecard, partner success rate, acknowledgment time
1. Open [Overview](/edi/overview) and choose a time window.
2. Scroll to **Partner scorecards**. Each row shows **Sent**, **Failed**, **Dead-lettered**, **Received**, **Success rate**, acknowledgment timing (**Ack avg**, **Ack p95**) and **Overdue acks**.
3. Select a partner name to open [Messages](/edi/messages) filtered to that partner.

### Open a recent failure
Keywords: dead letter, quarantined file, EDI error
1. Open [Overview](/edi/overview).
2. Scroll to **Recent failures**, which lists dead-lettered messages and quarantined files with the partner and error.
3. Select a row to open that message or inbound file, or use **View all messages** or **View all inbound files** to see the full lists.

## Notes
Viewing EDI pages needs read access to EDI. When nothing moved through EDI in the chosen window, the page says so and offers a button that widens the window to all time; with no traffic at all it offers **Set up a trading partner**, which opens [Partners](/edi/partners).
