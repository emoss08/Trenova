# Data analysis

Data analysis is the third agent extension capability (see
[agent-extensions.md](agent-extensions.md)). It lets an agent answer what a report alone
does not — margin by lane over a quarter, which customers drift past their payment terms,
where detention clusters by facility and hour — by running Python over report results and
files a person attached, in a sandbox on the organization's own vendor account. The first
vendor is E2B.

It has one tool, `analyze_data`, a read. The capability is the plan capability
`agent.data_analysis` and the permission resource `data_analysis` (`read`, in the Desk
bundle). The history is managed under `agent_extension`, like the extension itself.

## What the code reads

The tool takes the code, a one-sentence purpose a person reads, and up to five datasets, each
named for the file the code opens and given exactly one source:

- **A report run** (`reportRunId`, kind `report_run`). Only a run the same person ran
  (`RequestedByID`), and only while they may read reports (`report:read`, checked by the
  tool). Its stored rows are read through `reporting.Service.ReadRunRows`, which audits the
  read, and written as CSV by `reportrows.Envelope.WriteCSV`: the header is each column's id
  and values stay raw (decimals as exact text, times as unix seconds), so a program reads
  them without parsing a person's formatting. Totals are left out.
- **An attached file** (`documentId`). Only one the person attached to this conversation
  (`Document.AttachedBy`), read through `ConversationFileReader`. CSV, XLSX, JSON and text
  files only: what pandas reads without fetching anything.

Each dataset is written to `/home/user/data/<name>.<ext>` before the code runs. A name is
lower-case letters, digits and underscores starting with a letter, so the code opens it
without quoting. A dataset is at most 20 MB and an analysis at most 40 MB. A run with no
person behind it (a background run) can analyze no datasets.

## What keeps it safe

- **The sandbox cannot reach the network.** Every analysis starts a fresh E2B sandbox with
  `allow_internet_access: false` (all egress denied) and `allowPublicTraffic: false` (its own
  URLs need its traffic token). Whatever the data says and whatever the code tries, nothing
  it reads can be sent anywhere but back to Trenova. The sandbox is stopped when the analysis
  ends, whatever happened, and its lifetime is the code's limit plus a minute, so one Trenova
  failed to stop ends on its own.
- **Starting a sandbox is never retried**: a create that failed after the vendor began it
  would leave a second sandbox running and billed. Code is never re-run either.
- **Only the caller's own data.** Report runs another person ran and files from another
  conversation are refused as though they did not exist.
- **Taint follows the data.** The tool declares `ReadsExternal: marked` with the
  `attachment` source: a result that read an attached file names it
  (`analysisAnswer.TaintedRecords`), and the run is tainted as if the file had been read
  with `read_attachment`. A report run is the organization's own data and taints nothing,
  as `run_report` does not. Analysis is not in the outside-content list, so it does not cap
  later writes the way a web read does.
- **What leaves Trenova** is the code and the datasets' bytes, sent to E2B for as long as
  the analysis runs. The catalog's data notice says so.

## How an analysis runs

`agentextensionservice.Analyze`, inside the tool call:

1. Check the code (at most 20,000 characters) and the datasets, then reserve one unit of the
   daily limit.
2. Record the run in `agent_analysis_runs` (ids `aanr_`) before the vendor is called, so a
   sandbox Trenova started is never one it has no record of.
3. Start the sandbox, write the datasets, run the code as one cell for at most
   `runSeconds` (15 to 180, 90 by default), and stop the sandbox.
4. Keep up to four PNG images the code displayed (`plt.show()`), each at most 2 MB, in
   `agent_analysis_charts`, with the chart's title when the interpreter recognized one.
5. Record how it ended and settle the cost: E2B's per-second price for the processors and
   memory the sandbox held, against the day the run started.

How it ends:

| Status | Means | The agent is told |
|---|---|---|
| `Succeeded` | The code ran to the end. | The figures, the charts by position, and to say what they show. |
| `Errored` | The code raised. | The exception and traceback, and to fix the code rather than run it unchanged. |
| `TimedOut` | It ran past `runSeconds` and was stopped. | Why, and to narrow the data or simplify. |
| `Failed` | The sandbox could not run it (start, load, or the cell stopped without finishing, usually out of memory). | Why. |

The tool returns the output (the end kept past 6,000 characters, stderr after stdout), the
text results, the charts by position and title, and the error.

## On the Desk

An analysis is the `analysis` artifact kind (family `report`): the purpose as its title, the
charts, results, error, output and the code. It is made on purpose, not a view of a lookup,
so it is kept whether or not the reply names it, and `ShownArtifact.MustBeLinked` holds the
reply to naming it. The card reads each chart from
`GET /agent-analysis/runs/:runID/charts/:position/`, which serves the person the analysis ran
for and anyone who may read agent extensions; everyone else is told it does not exist.

A run is the `analysis_run` record kind; its record link
(`/admin/agent-control?tab=extensions&analysisRun=<id>`) opens the extension serving the
capability with that analysis expanded in its history, which lists every analysis with its
purpose, data, code, output, charts, duration and cost.

## Settings

E2B's settings are its API key, the seconds an analysis may run, the daily limit (100 by
default) and the sandbox template (`code-interpreter-v1`, which has Python with pandas, numpy
and matplotlib; an organization that built its own template from it can name that). The
connection test lists the account's running sandboxes, starting none.

## Where things live

| Piece | Location |
|---|---|
| Capability, settings, runs, charts | `internal/modules/agent/domain/agentextension/` (`capability.go`, `e2b.go`, `analysis.go`) |
| Stored runs and charts | `agent_analysis_runs`, `agent_analysis_charts` |
| Running an analysis, the history, chart access | `internal/modules/agent/app/agentextensionservice/analysis.go` |
| The tool and dataset resolution | `internal/modules/agent/app/agentquerytoolservice/analysis_tool.go` |
| Report rows as CSV | `pkg/reportrows/csv.go` |
| Vendor connector | `internal/infrastructure/agentextension/e2bconnector/` over `shared/e2b` |
| The Desk artifact | `assistantservice/analysis_artifact.go`; client `routes/desk/_components/artifacts/kinds/analysis/` |
| The result view both surfaces draw | `client/apps/web/src/components/assistant/analysis-result.tsx` |
| REST API | `/agent-analysis/runs/` (list), `/agent-analysis/runs/:runID/charts/:position/` |
| AI control | `client/apps/web/src/routes/agent-control/_components/extensions/analysis-runs.tsx` |
