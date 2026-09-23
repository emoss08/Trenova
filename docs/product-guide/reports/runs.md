---
path: /reports/runs
aliases: [report runs, report downloads, generated reports, report exports, report queue, report jobs]
related:
  - /reports
  - /reports/explore
---

## What it's for
Run history lists every report run in your organization: which report, its **Status**, **Format**, **Trigger** (for example run by hand or by a schedule), how many **Rows** it returned, its **Size** and **Duration**, when it was requested (**Requested At**) and when its file **Expires**. Anyone who runs or schedules reports uses it to download finished files and to cancel runs that are still in progress.

## Tasks

### Download a finished report
Keywords: get report file, export CSV, download Excel, report output
1. Open [Run history](/reports/runs).
2. Find the run (filter or sort by **Status**, **Format** or **Requested At** if needed).
3. Select the row, or right-click it and choose **Download**. Only finished runs whose file has not expired can be downloaded.

### Cancel a running report
Keywords: stop report, abort run, kill report job
1. Open [Run history](/reports/runs).
2. Right-click the queued or running run and choose **Cancel run**.

### Run a report again
Keywords: rerun report, refresh report output
1. Open the [Report library](/reports) and find the report.
2. Select **Run**, choose the **Format**, and select **Run report**. The new run appears in [Run history](/reports/runs).

## Notes
Needs read access to reports. Downloading needs export permission on reports. Report files are kept for a limited time; after the time in **Expires**, the file can no longer be downloaded and the report has to be run again. A row marked **(truncated)** returned more rows than the run could hold.
