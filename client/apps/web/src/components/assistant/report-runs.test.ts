import { describe, expect, it } from "vitest";
import { reportRunsFrom } from "./report-runs";
import type { ToolExchange } from "./thread-view";

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
        JSON.stringify({
          runId: "rrun_01M3034Q2N7JD99RA1D8DGH1ZF",
          reportKey: "expiring-worker-credentials",
          status: "queued",
          finished: false,
        }),
      ),
    ]);

    expect(runs).toEqual([
      { runId: "rrun_01M3034Q2N7JD99RA1D8DGH1ZF", reportKey: "expiring-worker-credentials" },
    ]);
  });

  it("shows one card when a run is started and then checked on", () => {
    const payload = { runId: "rrun_1", reportKey: "expiring-worker-credentials" };
    const runs = reportRunsFrom([
      exchange("run_report", JSON.stringify({ ...payload, status: "queued" })),
      exchange("get_report_run", JSON.stringify({ ...payload, status: "succeeded" })),
    ]);

    expect(runs).toHaveLength(1);
  });

  it("ignores tools that carry no run", () => {
    expect(reportRunsFrom([exchange("list_reports", JSON.stringify({ results: [] }))])).toEqual([]);
  });

  it("survives a tool result that is not the JSON it should be", () => {
    // Results are truncated when long and can carry a plain-text error. A parse
    // failure must not take the whole message down with it.
    expect(reportRunsFrom([exchange("run_report", '{"runId":"rrun_1","stat')])).toEqual([]);
    expect(reportRunsFrom([exchange("run_report", "the report could not be authorized")])).toEqual(
      [],
    );
  });

  it("ignores a failed call, which has no run to follow", () => {
    const failed = exchange("run_report", JSON.stringify({ runId: "rrun_1" }), true);
    expect(reportRunsFrom([failed])).toEqual([]);
  });
});
