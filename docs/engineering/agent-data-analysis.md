# Data analysis

Data analysis is the third agent extension capability (see
[agent-extensions.md](agent-extensions.md)). It lets an agent answer what a report alone
does not — margin by lane over a quarter, which customers drift past their payment terms,
where detention clusters by facility and hour — by running Python over report results and
files a person attached, in a sandbox on the organization's own vendor account. The first
vendor is E2B.

It has one tool, `report_analyze_data`, a read. The capability is the plan capability
`agent.data_analysis` and the permission resource `data_analysis` (`read`, in the Desk
bundle). The history is managed under `agent_extension`, like the extension itself.

## When an agent analyzes

An agent that holds `report_analyze_data` is told, in Trenova's own rules
(`agentdefinition.trenovaRules`), that a figure no tool stated (a total, a share, a ranking, an
age bucket, a trend, anything across more rows than a result showed it) is worked out with
`report_analyze_data`, and that a request to analyze, explore, dig into, compare or chart data is a call
to it in that turn. An agent without it keeps the plain rule: do not calculate, say what you would
need. The rule depends only on whether the tool is held, so the cached part of the prompt does not
change from turn to turn, and it holds over instructions an organization saved before the
extension existed ("never calculate what the report did not"). `report_run` and `report_get_run`
say plainly when their rows are a sample, "not the data to total, count, rank or compare", and
`report_analyze_data` loads whenever either of them or `agent_read_attachment` loads (see "follows" in
[agent-runtime.md](agent-runtime.md)). Before these, the Report analyst asked to "run AR aging by
customer and do analysis on it" read the sample, wrote its own commentary and offered the
analysis as a next step; `deskbench/scenarios/analysis.yaml` holds that case and the rest.

The rule also holds when every row is in front of the agent: a share of two columns or a total of
one is still worked out by `report_analyze_data`, and `report_run` says of a complete result that a
figure its rows do not state is not read off them. An attached CSV, XLSX or text file is read by
`report_analyze_data` as it is, so an agent never tells the person such a file cannot be read because
`agent_read_attachment` found no text in it. The scope classifier (`agentguard`) treats a request to
analyze, chart, total or compare the organization's own data as operations work even when it names
Python, pandas or a package, and keeps a computation unrelated to the organization's data (a
prime, a benchmark) and a request for the sandbox's environment or keys out of scope.

The prompt is not the only control. A reply that files nothing is held to the money and
percentages the conversation read (see "What the reply may claim" in
[agent-runtime.md](agent-runtime.md)): a share worked out in the agent's head is sent back once
with a correction telling an agent that holds `report_analyze_data` to work it out there.

## What the code reads

The tool takes a short title (three to eight words, which names the result on the Desk), a
one-sentence purpose, the code, and up to five datasets, each named for the DataFrame and file
the code reads and given exactly one source:

- **A report run** (`reportRunId`, kind `report_run`). Only a run the same person ran
  (`RequestedByID`), and only while they may read reports (`report:read`, checked by the
  tool). Its stored rows are read through `reporting.Service.ReadRunRows`, which audits the
  read, and written as CSV by `reportrows.Envelope.WriteCSV`: the header is each column's id
  and values stay raw (decimals as exact text, times as unix seconds), so a program reads
  them without parsing a person's formatting. Totals are left out.
- **An attached file** (`documentId`). Only one the person attached to this conversation
  (`Document.AttachedBy`), read through `ConversationFileReader`. CSV, XLSX and text
  files only: what pandas reads without fetching anything.

Each dataset is written to `/home/user/data/<name>.<ext>` before the code runs, and loaded
for it (see [Prepared before the code](#prepared-before-the-code)). A name is lower-case
letters, digits and underscores starting with a letter, so it is a Python name and a file
name without quoting. A dataset is at most 20 MB and an analysis at most 40 MB. A run with no
person behind it (a background run) can analyze no datasets.

## Prepared before the code

Before the agent's code, Trenova runs its own cell in the same sandbox
(`agentextensionservice/analysis_prelude.py`, embedded in the binary), reading
`/home/user/.trenova/analysis.json`: the organization's timezone and, for each report run,
its columns (`AnalysisColumnSpec`: id, label, type, and how its values read, from the
report's format). It:

- **Loads each dataset as a DataFrame named after it.** A report run's columns are headed by
  their labels, as `report_run` showed them to the model (so `df["Open Balance"]`, not
  `open_balance`), integers as nullable integers, decimals as floats, times as dates in the
  organization's timezone and flags as booleans. A file is read with pandas' own reader. How
  each loaded (rows, columns and dtypes, or the error) comes back as JSON and is handed to the
  agent under `datasets[].columns`, so a retry names real columns.
- **Styles charts for either theme**: a clear background, a neutral ink that reads on white
  and on black, a fixed palette, no top or right spine, a light grid, and no scientific
  notation; `plt.show()` and the inline backend's flush both turn a plain number axis into
  compact figures (6M, 36.9K).
- **Defines `show_figure(label, value, kind)` and `show_table(df, title, formats)`**, which
  display JSON (`{"trenova": "figure" | "table", ...}`) rather than text. A kind is `text`,
  `number`, `money`, `count`, `percent` (a fraction), `date`, `datetime` or `bool`; a
  dataset column's kind comes from the report, so a money column reads as money unasked.
  Only money carries a currency (`agentextension.AnalysisCurrency`), though a report's
  display gives every column the organization's. `show_table` shows a frame's index as a
  column only when it holds values (a name, several levels, or labels that are not whole
  numbers); the unnamed row positions a sort or filter leaves behind are dropped.

Every step is guarded, so the preparation never fails an analysis; a preparation that cannot
run at all (the cell raised or stopped) ends the run `Failed` with a message that does not
blame the agent's code.

## What keeps it safe

- **The sandbox cannot reach the network.** Every analysis starts a fresh E2B sandbox with
  `allow_internet_access: false` (all egress denied) and `allowPublicTraffic: false` (its own
  URLs need its traffic token). Whatever the data says and whatever the code tries, nothing
  it reads can be sent anywhere but back to Trenova. The sandbox is stopped when the analysis
  ends, whatever happened, and its lifetime is the code's limit plus two minutes (the
  preparation runs in that margin), so one Trenova failed to stop ends on its own.
- **Starting a sandbox is never retried**: a create that failed after the vendor began it
  would leave a second sandbox running and billed. Code is never re-run either.
- **Only the caller's own data.** Report runs another person ran and files from another
  conversation are refused as though they did not exist. A report run is read only by the
  person who ran it, on every surface (`reporting.Service.GetRun`, `ListRuns`,
  `ListRunsConnection`): it holds what the report returned under that person's data scope, and
  may be a run of their private report.
- **Taint follows the data.** The tool declares `ReadsExternal: marked` with the
  `attachment` source: a result that read an attached file names it
  (`analysisAnswer.TaintedRecords`), and the run is tainted as if the file had been read
  with `agent_read_attachment`. A report run is the organization's own data and taints nothing,
  as `report_run` does not. Analysis is not in the outside-content list, so it does not cap
  later writes the way a web read does.
- **What leaves Trenova** is the code and the datasets' bytes, sent to E2B for as long as
  the analysis runs. The catalog's data notice says so.

## How an analysis runs

`agentextensionservice.Analyze`, inside the tool call:

1. Check the code (at most 20,000 characters) and the datasets, then reserve one unit of the
   daily limit.
2. Record the run in `agent_analysis_runs` (ids `aanr_`) before the vendor is called, so a
   sandbox Trenova started is never one it has no record of.
3. Start the sandbox, write the datasets and the configuration, run the preparation (at most
   a minute), run the code as one cell for at most `runSeconds` (15 to 180, 90 by default),
   and stop the sandbox.
4. Sort what the code displayed, in order: up to four PNG charts (each at most 2 MB, kept in
   `agent_analysis_charts` with the chart's title when the interpreter recognized one), up to
   six tables (20 columns, 200 rows, kept as values in `tables`), up to eight headline
   figures (`figures`), and anything else as text. `layout` records the order the charts and
   tables were shown in.
5. Record how it ended and settle the cost: E2B's per-second price for the processors and
   memory the sandbox held, against the day the run started.

How it ends:

| Status | Means | The agent is told |
|---|---|---|
| `Succeeded` | The code ran to the end. | The figures, tables (first 25 rows) and charts, and to say what they mean without listing them again; or, when nothing was shown, to show results rather than print them next time. |
| `Errored` | The code raised. | The exception and traceback, the datasets' real columns, and to fix the code rather than run it unchanged. |
| `TimedOut` | It ran past `runSeconds` and was stopped. | Why, and to narrow the data or simplify. |
| `Failed` | The sandbox could not run it (start, load, or the cell stopped without finishing, usually out of memory). | Why. |

The tool returns the title, the datasets with their loaded columns, the figures, the tables,
the charts by position and title, the order they were shown in, the output (the end kept past
6,000 characters, stderr after stdout), the text results, and the error.

## On the Desk

An analysis is the `analysis` artifact kind (family `report`; the Desk draws it as its own
"Analysis" kind), titled by its short title. Only an analysis that ran to the end and showed
something (a figure, a table or a chart) makes one: an attempt whose code raised is the agent's
to fix, and the person sees the result it reaches, not each attempt; a run that only printed is
the agent looking at the data, and a card for it was one more the reply had to name. Its lineage is its title (case and spacing aside), so running it again under
the same title, after a fix or over newer data, is the next version of one card rather than a
second. It is made on purpose, not a view of a lookup, so it is kept whether or not the reply
names it, and `ShownArtifact.MustBeLinked` holds the reply to naming it.

The card shows the headline figures as a strip, then each chart and table in the order the
code showed them (money, counts, percentages and dates formatted as their kinds, numbers
aligned right), what it could not do, and where the data came from; what it printed and the
code stay folded beneath. It reads each chart from
`GET /agent-analysis/runs/:runID/charts/:position/`, and a table longer than the 25 rows the
agent was shown from `GET /agent-analysis/runs/:runID/`. Both serve the person the analysis
ran for and anyone who may read agent extensions; everyone else is told it does not exist.

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
| Running an analysis, the history, chart and run access | `internal/modules/agent/app/agentextensionservice/analysis.go` |
| The preparation the sandbox runs first | `internal/modules/agent/app/agentextensionservice/analysis_prelude.py` |
| Tables, figures, layout and their bounds | `internal/modules/agent/domain/agentextension/analysisresult.go` |
| The tool and dataset resolution | `internal/modules/agent/app/agentquerytoolservice/analysis_tool.go` |
| Report rows as CSV | `pkg/reportrows/csv.go` |
| Vendor connector | `internal/infrastructure/agentextension/e2bconnector/` over `shared/e2b` |
| The Desk artifact | `assistantservice/analysis_artifact.go`; client `routes/desk/_components/artifacts/kinds/analysis/` |
| The result view both surfaces draw | `client/apps/web/src/components/assistant/analysis-result.tsx` |
| REST API | `/agent-analysis/runs/` (list), `/agent-analysis/runs/:runID/`, `/agent-analysis/runs/:runID/charts/:position/` |
| AI control | `client/apps/web/src/routes/agent-control/_components/extensions/analysis-runs.tsx` |
