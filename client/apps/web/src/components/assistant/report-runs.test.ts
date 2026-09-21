import { describe, expect, it } from "vitest";
import { reportRunsFrom } from "./report-runs";
import type { ToolExchange } from "./thread-view";

/**
 * How the server actually stores a tool result: fenced as untrusted data under
 * a "Result from <tool>" line, never as bare JSON. Building fixtures any other
 * way tests a contract nothing implements.
 */
function fenced(payload: string): string {
  return `Result from run_report:\n<untrusted_data>\n${payload}\n</untrusted_data>`;
}

function exchange(name: string, content: string, toolFailed = false): ToolExchange {
  return {
    call: { id: `call_${name}_${content.length}`, name, arguments: {} },
    result: {
      id: "amsg_1",
      threadId: "athr_1",
      sequence: 1,
      role: "Tool",
      content,
      toolCallId: `call_${name}_${content.length}`,
      toolName: name,
      toolFailed,
      refused: false,
      scopeStage: "",
      scopeCategory: "",
      scopeReason: "",
      model: "",
      providerId: "",
      inputTokens: 0,
      outputTokens: 0,
      createdAt: 0,
    } as ToolExchange["result"],
  };
}

describe("reportRunsFrom", () => {
  it("finds the run a report tool started", () => {
    const runs = reportRunsFrom([
      exchange(
        "run_report",
        fenced(
          JSON.stringify({
            runId: "rrun_01M3034Q2N7JD99RA1D8DGH1ZF",
            reportKey: "expiring-worker-credentials",
            status: "queued",
            finished: false,
          }),
        ),
      ),
    ]);

    expect(runs).toEqual([
      {
        runId: "rrun_01M3034Q2N7JD99RA1D8DGH1ZF",
        reportKey: "expiring-worker-credentials",
        reportName: "",
      },
    ]);
  });

  /**
   * A saved report has no canned key, only an id and a name. The tool answers
   * with the name so the card can read "Revenue by customer" rather than
   * "Report", which is what a key-less run used to be called.
   */
  it("carries the report's name when the tool gives one", () => {
    const runs = reportRunsFrom([
      exchange(
        "run_report",
        fenced(
          JSON.stringify({
            runId: "rrun_01M3034Q2N7JD99RA1D8DGH1ZF",
            definitionId: "rdef_01M3034Q2N7JD99RA1D8DGH1ZF",
            reportName: "Revenue by customer",
            status: "queued",
            finished: false,
          }),
        ),
      ),
    ]);

    expect(runs).toEqual([
      {
        runId: "rrun_01M3034Q2N7JD99RA1D8DGH1ZF",
        reportKey: "",
        reportName: "Revenue by customer",
      },
    ]);
  });

  it("shows one card when a run is started and then checked on", () => {
    const payload = { runId: "rrun_1", reportKey: "expiring-worker-credentials" };
    const runs = reportRunsFrom([
      exchange("run_report", fenced(JSON.stringify({ ...payload, status: "queued" }))),
      exchange("get_report_run", fenced(JSON.stringify({ ...payload, status: "succeeded" }))),
    ]);

    expect(runs).toHaveLength(1);
  });

  it("ignores tools that carry no run", () => {
    expect(
      reportRunsFrom([exchange("list_reports", fenced(JSON.stringify({ results: [] })))]),
    ).toEqual([]);
  });

  it("survives a tool result that is not the JSON it should be", () => {
    // Results are truncated when long and can carry a plain-text error. A parse
    // failure must not take the whole message down with it.
    expect(reportRunsFrom([exchange("run_report", fenced('{"runId":"rrun_1","stat'))])).toEqual([]);
    expect(
      reportRunsFrom([
        exchange("run_report", 'Tool "run_report" failed: you do not have permission'),
      ]),
    ).toEqual([]);
  });

  it("ignores a failed call, which has no run to follow", () => {
    const failed = exchange("run_report", fenced(JSON.stringify({ runId: "rrun_1" })), true);
    expect(reportRunsFrom([failed])).toEqual([]);
  });
});
