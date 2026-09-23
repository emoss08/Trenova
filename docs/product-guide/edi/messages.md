---
path: /edi/messages
aliases: [X12 documents, EDI documents, sent EDI, EDI delivery status, acknowledgments, 997, 999, dead letter queue]
related:
  - /edi/overview
  - /edi/communication-profiles
  - /edi/inbound-files
---

## What it's for
The EDI Messages page lists every X12 document Trenova has generated or received, with its transaction set, partner, direction, delivery status, acknowledgment status and control numbers. Opening a message shows its delivery attempts and last error, the acknowledgment details and the raw X12.

EDI support staff use it to confirm a document reached a partner, chase missing acknowledgments, and resend deliveries that failed. The list refreshes itself every 30 seconds.

## Tasks

### Check whether a document was delivered and acknowledged
Keywords: was the invoice sent, 997 received, ack status, control number lookup
1. Open [Messages](/edi/messages).
2. Search or filter by **Partner**, **Transaction** or **Control number**, and read the **Delivery** and **Acknowledgment** columns.
3. Select the row to see **Delivery** (status, **Attempts**, **Remote path**, **Last error**), **Acknowledgment** and the **Raw X12**.

### Retry a failed delivery
Keywords: resend EDI, dead-lettered message, delivery failed
1. Open [Messages](/edi/messages) and select the message's row.
2. Select **Retry delivery**. It is offered for outbound messages that are queued, failed or dead-lettered.

### Retry deliveries for several messages
Keywords: bulk retry, resend many
1. Open [Messages](/edi/messages).
2. Tick the rows you want. A bar appears at the bottom of the screen.
3. Select **Retry delivery**. Messages that cannot be retried are skipped.

### Send a delivered document again
Keywords: replay, resend sent document, partner lost file
1. Open [Messages](/edi/messages) and select the message's row.
2. Select **Replay delivery** to queue the already-delivered document for another delivery to the partner. It is offered for outbound messages with status Sent whose raw content is still kept.

## Notes
Viewing messages needs read access to EDI; retrying and replaying need update access to EDI. How many attempts a delivery gets before it is dead-lettered is set on the partner's [communication profile](/edi/communication-profiles). How long raw payloads are kept is set on [Data retention](/organization/data-retention); once a payload is purged the message can no longer be replayed.
