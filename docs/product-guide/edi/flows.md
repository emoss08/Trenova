---
path: /edi/flows
aliases: [EDI conversations, held tenders, escalated EDI, tender follow-up, 990 overdue, EDI timers]
related:
  - /edi/lading
  - /edi/messages
  - /edi/partners
---

## What it's for
Flow instances lists each load, tender and invoice a partner's Lading flow is following: which flow, the partner's key for it, whether it is open, held, escalated or closed, why it waits and its next deadline. A flow holds an instance for a planner when the partner's rules say so (a changed rate, a missing stop), and escalates it when an answer or acknowledgment does not arrive in time.

EDI coordinators and planners use it to find tenders waiting on a person and release them once the cause is fixed.

## Tasks

### Find tenders waiting on a planner
Keywords: held EDI tender, escalated tender, stuck EDI
1. Open [Flow instances](/edi/flows).
2. Filter **Status** to **Held** or **Escalated**. **Why it waits** says what the flow stopped for.

### Release a held flow
Keywords: resume EDI flow, unblock tender
1. Open [Flow instances](/edi/flows) and select the row.
2. Read **Why it waits**, the **Events** and anything **Awaiting acknowledgment**, and fix the cause, such as the load or the partner's mapping.
3. Select **Release**. The flow carries on from where it stopped.

## Notes
Viewing flow instances needs read access to EDI; releasing one needs update access to EDI. Only a held or escalated instance can be released.
