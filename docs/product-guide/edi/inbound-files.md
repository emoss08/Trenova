---
path: /edi/inbound-files
aliases: [received EDI files, partner mailbox files, quarantined files, EDI inbox, failed inbound EDI, reprocess EDI]
related:
  - /edi/overview
  - /edi/messages
  - /edi/communication-profiles
---

## What it's for
Inbound files are the files Trenova has picked up from partner mailboxes, with their processing state: received, parsed, processed, partial, quarantined or duplicate. Opening a file shows where it came from (partner, method, remote path, ISA sender and receiver), what processing produced, the transactions inside it with their acknowledgment status, and the raw content.

EDI support staff use it to find files that failed processing, fix the cause, and run them again. The list refreshes itself every 30 seconds.

## Tasks

### Find out why a file failed
Keywords: quarantined file, inbound error, EDI parse error
1. Open [Inbound files](/edi/inbound-files).
2. Look for rows whose **Status** is Quarantined or Partial.
3. Select the row and read **Processing notes** under **Processing**, then check the **Source** details and **Raw content**.

### Reprocess a file
Keywords: rerun inbound file, retry file, process again
1. Open [Inbound files](/edi/inbound-files) and select the file's row.
2. After fixing the cause (for example a missing partner, mapping or communication profile), select **Reprocess file**. It is offered for quarantined and partially processed files.

### Reprocess several files at once
Keywords: bulk reprocess
1. Open [Inbound files](/edi/inbound-files).
2. Tick the rows you want. A bar appears at the bottom of the screen.
3. Select **Reprocess**. Files that are not quarantined or partially processed are skipped.

## Notes
Viewing inbound files needs read access to EDI; reprocessing needs update access to EDI. How long raw EDI payloads are kept is set on [Data retention](/organization/data-retention).
