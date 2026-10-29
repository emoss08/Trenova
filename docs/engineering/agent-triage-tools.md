# Agent tools for the Triage queue

What an agent can do with the mail that arrives on the organization's addresses: who owns a
message, when it is parked and why, the sender, the reply and load drafted on it, where its
attachments are filed, the exceptions and disputes it reports, and the conflicts Triage found
on it. Each tool says how far it may run without a person, and this page lists which Triage
writes no agent makes.

Read [agent-runtime.md](agent-runtime.md) for tiers, egress classes and taint, and
[proposal-previews.md](proposal-previews.md) for how a proposal is previewed and approved.
The generated [ai-tool-safety.md](ai-tool-safety.md) is the authority on each policy, and
[agent-write-coverage.md](agent-write-coverage.md) on which write each tool covers; this
page explains them.

## The rule

The queue and sender tools live in `agenttools/triagequeuetools` and the reply, load, filing,
exception and conflict tools in `agenttools/triageworktools`; both build on the shared triage
pieces in `agenttools/inboundtools`, beside the tools that came before them
(`inbox_link_inbound_message`, `inbox_mark_inbound_message`, `inbox_send_reply`); the reads
(`inbox_get_inbound_message`, `inbox_list_triage_assignees`) live in `querytools/inboxquerytools`.

Every write calls the service its GraphQL mutation calls (`inboundmessageservice`,
`triageworkservice`, `inboundreplyservice`, `triageconflictservice`) and never writes
around it. Each service exposes the plan its write runs on, and the tool's preview and
`Validate` run that plan:

- The queue writes (`Assign`, `Reopen`, `Snooze`, `Wake`, `WaitForReply`, `StopWaiting`,
  `Unlink`, `HandOff`) are transitions: a check, an eligibility rule and a `QueuePatch`
  applied in SQL. `Plan<Write>` builds the same transition, reads the messages without a
  lock, runs the check, and applies the patch to a copy with `QueuePatch.ApplyTo`, the one
  Go rendering of what the SQL sets. The service's test fake applies patches with the same
  method, so the parity tests in `queueplan_test.go` hold the plan to what the write saves.
- The work writes (`BlockSender`, `UnblockSender`, `LogException`, `DecideStops`,
  `DraftLoad`, `DiscardLoad`, `CreateLoad`, `File`, `Unfile`) share their prelude with
  `Plan<Write>` and change a copy with the same `Apply…` function the write uses.
- The reply and conflict writes (`SaveDraft`, `DraftReply`, `DiscardDraft`, `Resolve`,
  `Reopen`) do the same with `savedDraft`, `canDraft`, `currentDraft` and the conflict's
  `apply` closure.

A refusal only a tool knows (nothing in the queue would change, the attachment belongs to
another message, the sender is not blocked, there is no draft to discard) is a `would_fail`
warning on the preview, so the runtime files nothing, and the tool's run step checks again
and refuses.

## Queue state

| Tool | Resource / operation | Class | Default → most | Covers |
| --- | --- | --- | --- | --- |
| `inbox_list_triage_assignees` | inbound message / read | reads only | automatic | — |
| `inbox_assign_inbound_message` | inbound message / assign | inside | Propose → Ask first | `assignInboundMessages` |
| `inbox_assign_queue` | inbound message / assign | inside | Propose → Ask first | `handOffTriage` |
| `inbox_reopen_inbound_message` | inbound message / update | inside | Propose → Ask first | `reopenInboundMessages` |
| `inbox_snooze_inbound_message` | inbound message / update | inside | Propose → Ask first | `snoozeInboundMessages` |
| `inbox_await_reply` | inbound message / update | inside | Propose → Ask first | `waitForInboundReply` |
| `inbox_wake_inbound_message` | inbound message / update | inside | Propose → Automatic | `wakeInboundMessage`, `stopWaitingForInboundReply` |
| `inbox_unlink_inbound_message` | inbound message / update | inside | Propose → Automatic, as the mailbox allows | `unlinkInboundMessage` |

The bulk tools take at most 50 message ids and never a folder. The assign tools are gated
on the assign operation, which is what lets them give a message to someone other than the
caller, and the fixed agent table (`permission.IsAgentAllowed`) grants it on inbound
messages for them; the owner comes from `inbox_list_triage_assignees`, which also names the
on-call person.
`inbox_wake_inbound_message` reads how the message is parked and calls `Wake` or `StopWaiting`.
A hand-off preview shows the first batch and says when there is more.

`inbox_await_reply` parks the message for the people working the queue. It is not the
agent's own wait: to pick its work up when the sender answers, the agent also calls
`agent_create_wait` with a `Reply` wait ([agent-waits.md](agent-waits.md)). There is one wait
mechanism per owner, and these tools add none.

## Senders, replies and loads

| Tool | Resource / operation | Class | Default → most | Covers |
| --- | --- | --- | --- | --- |
| `inbox_block_sender` | inbound message / update | inside | Propose → Ask first | `blockInboundSender` |
| `inbox_unblock_sender` | inbound message / update | inside | Propose → Ask first | `unblockInboundSender` |
| `inbox_draft_reply` | inbound message / update | inside | Propose → Automatic, as the mailbox allows | `draftInboundReply`, `saveInboundReplyDraft` |
| `inbox_discard_reply_draft` | inbound message / update | inside | Propose → Ask first | `discardInboundReplyDraft` |
| `inbox_draft_load` | inbound message / update | inside | Propose → Automatic, as the mailbox allows | `draftInboundLoad` |
| `inbox_discard_load` | inbound message / update | inside | Propose → Automatic, as the mailbox allows | `discardInboundLoad` |
| `inbox_create_load` | shipment / create | inside | Ask first → Ask first, idempotent | `createInboundLoad` |

Blocking silently drops an outsider's future mail, which can hide a customer or carrier from
everyone, so it is always asked first. Unblocking names the sender by one of their messages,
normalised the way a block records it. `inbox_draft_reply` only drafts: with a body it saves
the agent's words, without one it has Triage's drafter write it (that preview is partial,
since the words are not known yet). Sending stays with `inbox_send_reply` or a
person. `inbox_create_load` is never unattended, like `shipment_create`; the Intake desk
uses it only for a tender written in the body, because a tender whose attachment became a
document already wakes shipment intake.

## Filing, exceptions, disputes and conflicts

| Tool | Resource / operation | Class | Default → most | Covers |
| --- | --- | --- | --- | --- |
| `inbox_file_attachment` | document / update | inside | Propose → Automatic, as the mailbox allows | `fileInboundAttachment` |
| `inbox_unfile_attachment` | document / update | inside | Propose → Ask first | `unfileInboundAttachment` |
| `inbox_record_discrepancy` | shipment hold / create | inside | Propose → Automatic, as the mailbox allows | `logInboundException` |
| `detention_resolve_dispute` | detention / update | money | Propose → Propose | `decideInboundStops` |
| `inbox_resolve_conflict` | inbound message / update | inside | Propose → Ask first | `resolveInboundConflict` |
| `inbox_reopen_conflict` | inbound message / update | inside | Propose → Automatic, as the mailbox allows | `reopenInboundConflict` |

Filing and unfiling take the message as well as the attachment, so the mailbox's condition
applies and the attachment is checked to belong to it. A split preview creates one document a
range; when the attachment's page count is not stored yet it is partial, and the write
refuses a range past the last page. Unfiling a split deletes the documents it made.
`detention_resolve_dispute` waives or stands by the disputed charge as the caller and has
Triage draft the answer; its preview shows the billed amount a waiver gives up, from the
disputed occurrence's stop times.

## Who holds them

The Intake desk holds every tool above but `detention_resolve_dispute`, which the detention
desk holds with `inbox_get_inbound_message` and `inbox_list_inbound_messages` (searched by `query`) to read
the dispute.
`inbox_get_inbound_message` shows what the ids come from: the owner, why a message is parked, the
reply and load drafted on it, each conflict with its id, and where each attachment is filed.

The descriptions name the read tools their ids come from rather than "the run's subject",
and avoid words a background run's ranking text carries: a run opens with the eight tools
that best match its request, and wording that matched "work the record this run is about"
would have opened every inbox run with write tools in place of the reads it starts from.

## What no agent does

- `draftInboundMany` and `fileInboundMany` are exempt as duplicates: their tools do the same
  for each message, one approval at a time.
- Mailboxes, their credentials, response targets, Triage settings and reply templates are
  configuration; rules are agent administration; corrections and flagged facts are
  attestation. `writecoverage.yml` gives each its reason.
