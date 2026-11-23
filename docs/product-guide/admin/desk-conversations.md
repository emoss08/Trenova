---
path: /admin/desk-conversations
aliases: [desk transcripts, conversation transcripts, read someone's desk conversation, export desk conversations, download transcripts, AI chat history, chat logs, conversation oversight, compliance export, e-discovery]
related:
  - /admin/audit-logs
  - /admin/roles
  - /desk
---

## What it's for
Administrators read and take out of Trenova the Desk conversations of everyone in the
organization: who talked to which agent, when, and what was said, as the person saw it. Each
person's conversations stay private on their own Desk; this page is the one place someone else can
read them, and only with the permission "Desk conversations of other people".

View lists every conversation (person, agent, title, where it began, how many messages, when
it started and its last message) and opens one read-only. The list never shows what was said;
titles can, because a conversation's title is taken from its first question. Export pulls
transcripts out of Trenova: one conversation at a time, the conversations you select, or every
conversation in a timeframe. The Organization Administrator holds both. Any other role needs the
"Read and export other people's Desk conversations" capability added to it on purpose. A
conversation shows whatever its owner was shown, including records the reader's own role may not
reach, so grant it only to people whose job is oversight. Trenova support, agents and API keys can
never read other people's conversations.

Opening a conversation changes nothing for its owner: it is not marked read, nothing moves on
their Desk, and they are not told. Every conversation opened, every transcript downloaded and
every export requested or downloaded is written to the [audit log](/admin/audit-logs), naming who
looked, whose conversation it was and, for an export, what it covered.

A timeframe is over each conversation's last message, in your organization's time zone: **Last 7
days** is today and the six days before it, and **Last 12 months** starts on today's date a year
ago. The list and an export use the same timeframes, so an export holds what the list showed.

An export is one zip: a Markdown transcript for each conversation, in a folder per person and
named by the day it started and its title, the same file the owner gets when they download their
own conversation with their name at the top, and `conversations.jsonl`, one conversation per line
with its person, agent, title, times and every message, for an archive or a search tool. One
export holds at most 5,000 conversations, or 200 selected ones, and at most 1 GB; one export is
written at a time for the organization. You are notified when it is ready, only you can download
it, and it is deleted after seven days. Command palette questions and background agent runs are
not here: agent runs and their transcripts are in [AI control](/admin/agent-control).

## Tasks

### Read someone's conversation
Keywords: open a conversation, see what someone asked, read a transcript
1. Open [Desk conversations](/admin/desk-conversations).
2. Narrow the list with the **Timeframe**, **Person**, **Agent** and **Where it began** controls
   above it.
3. Select a conversation. It opens beside the list, read-only, with a note that opening it was
   recorded.

### Download one conversation's transcript
Keywords: download a transcript, save a conversation, markdown transcript
1. Open [Desk conversations](/admin/desk-conversations) and select the conversation.
2. Select **Download**. You can also right-click its row and choose **Download transcript**.

### Download the conversations you pick
Keywords: download several transcripts, select conversations, bulk download
1. Open [Desk conversations](/admin/desk-conversations).
2. Tick the conversations you want, on as many pages as you need, up to 200.
3. Select the download action in the bar at the bottom of the page. One conversation downloads at
   once; several are written to one zip, which appears under **Exports**.

### Export every conversation in a timeframe
Keywords: export desk conversations, how many days back, compliance archive, all transcripts
1. Open [Desk conversations](/admin/desk-conversations) and narrow the list to what you want.
2. Select **Export transcripts**. The dialog opens with the list's filters.
3. Choose the **Timeframe**, and a **Person** or **Agent** if you want only theirs. The dialog says
   how many conversations that is before you start, and stops you when it is more than one export
   can hold.
4. Select **Export**. When you are notified that it is ready, open **Exports** and select
   **Download**.
