---
path: /admin/capture
aliases: [scanning settings, Trenova Capture settings, scanner profiles, scan settings, virtual printer settings, paired computers, scanner fleet]
related:
  - /intake
  - /capture/devices
  - /admin/document-intelligence
---

## What it's for
Scanning and printing is where administrators run Trenova Capture, the Windows companion that scans paper and catches print jobs into Trenova. **Settings** turns it on for the organization and sets how long unfiled pages are kept and which versions of the companion may connect. **Scan profiles** are the settings people pick from when they start a scan: resolution, color, both sides, and how a stack is split into documents. **Computers** lists every computer paired to scan, whose it is, and whether it is online.

## Tasks

### Turn on scanning and printing
Keywords: enable Trenova Capture, allow scanning, turn on virtual printer
1. Open [Scanning and printing](/admin/capture) and select **Settings**.
2. Turn on **Turn on scanning and printing into Trenova**.
3. Optionally turn on **File documents behind a cover sheet on their own**, and set **Keep unfiled pages for (days)**.
4. Select **Save changes**.

### Add a scan profile
Keywords: scanner settings, DPI, color scanning, duplex, split stack
1. Open [Scanning and printing](/admin/capture) and select **Scan profiles**.
2. Select **New scan profile**.
3. Fill in **Name**, choose the **Resolution** and **Color mode**, and tick what to **Split the stack on**.
4. Turn on **Use when nobody picks one** to make it the default, then select **Create profile**.

### Stop a computer from scanning
Keywords: revoke scanner, lost laptop, employee left, remove device
1. Open [Scanning and printing](/admin/capture) and select **Computers**.
2. Find the computer, select **Revoke**, optionally say why, and select **Revoke** again. It stops at once.

## Notes
Settings needs access to document settings; scan profiles and computers each have their own permission. Turning scanning off stops every paired computer without unpairing it. A change to a profile reaches the next scan started with it; stacks already scanned keep the settings they were scanned with.

People pair their own computers from [My scanners](/capture/devices).
