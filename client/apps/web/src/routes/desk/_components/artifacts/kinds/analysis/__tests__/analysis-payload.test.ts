import { analysisViewOf } from "@/lib/analysis-run";
import type { AnalysisRun } from "@/types/agent-extension";
import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { analysisFrom } from "../../../artifact-payloads";

function artifactOf(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "aart_1",
    threadId: "athr_1",
    kind: "analysis",
    status: "Ready",
    title: "Margin by lane for Q3",
    payload,
  } as unknown as AssistantArtifact;
}

describe("analysisFrom", () => {
  it("reads what the report_analyze_data tool published", () => {
    expect(
      analysisFrom(
        artifactOf({
          analysisRunId: "aanr_1",
          status: "Succeeded",
          purpose: "Margin by lane for Q3",
          code: "print(1)",
          datasets: [
            {
              name: "lanes",
              path: "/home/user/data/lanes.csv",
              source: "ReportRun",
              label: "Margin by lane",
              rows: 42,
            },
          ],
          output: "margin 0.18",
          results: ["0.18", ""],
          charts: [{ position: 1, title: "Margin by lane" }, { position: 0 }],
        }),
      ),
    ).toEqual({
      runId: "aanr_1",
      status: "Succeeded",
      title: "Margin by lane for Q3",
      code: "print(1)",
      datasets: [
        {
          name: "lanes",
          path: "/home/user/data/lanes.csv",
          source: "ReportRun",
          label: "Margin by lane",
          rows: 42,
        },
      ],
      output: "margin 0.18",
      results: ["0.18"],
      charts: [{ position: 1, title: "Margin by lane" }],
      tables: [],
      figures: [],
      layout: [],
      error: null,
      failure: "",
      truncated: false,
    });
  });

  it("reads the tables, figures and order the analysis showed, and its own title", () => {
    const analysis = analysisFrom(
      artifactOf({
        analysisRunId: "aanr_2",
        status: "Succeeded",
        title: "Open AR by customer",
        tables: [
          {
            title: "Open balance",
            columns: [
              { name: "Customer", kind: "text" },
              { name: "Open Balance", kind: "money", currency: "USD" },
              { name: "Share", kind: "share-of-wallet" },
            ],
            rows: [["Acme", 36912.16, { nested: true }]],
            totalRows: 40,
          },
        ],
        figures: [{ label: "Total open", value: 6045662.16, kind: "money", currency: "USD" }],
        layout: [
          { kind: "table", index: 0 },
          { kind: "chart", index: 1 },
        ],
      }),
    );

    expect(analysis.title).toBe("Open AR by customer");
    expect(analysis.tables).toEqual([
      {
        title: "Open balance",
        columns: [
          { name: "Customer", kind: "text", currency: "" },
          { name: "Open Balance", kind: "money", currency: "USD" },
          { name: "Share", kind: "text", currency: "" },
        ],
        rows: [["Acme", 36912.16, null]],
        totalRows: 40,
      },
    ]);
    expect(analysis.figures).toEqual([
      { label: "Total open", value: 6045662.16, kind: "money", currency: "USD", note: "" },
    ]);
    expect(analysis.layout).toEqual([
      { kind: "table", index: 0 },
      { kind: "chart", index: 1 },
    ]);
  });

  it("drops tables it cannot read rather than the whole analysis", () => {
    expect(analysisFrom(artifactOf({ tables: "not a list" })).tables).toEqual([]);
  });

  it("reads a status it does not know as a failure, never a success", () => {
    expect(analysisFrom(artifactOf({ status: "Exploded" })).status).toBe("Failed");
  });

  it("keeps the exception and why it stopped", () => {
    const analysis = analysisFrom(
      artifactOf({
        status: "Errored",
        error: { name: "KeyError", value: "'margin'", traceback: "tb" },
        failure: "",
      }),
    );
    expect(analysis.error).toEqual({ name: "KeyError", value: "'margin'", traceback: "tb" });
  });
});

describe("analysisViewOf", () => {
  const run: AnalysisRun = {
    id: "aanr_1",
    extensionType: "E2B",
    status: "Errored",
    title: "Lane margin",
    purpose: "Margin by lane for Q3",
    code: "print(1/0)",
    datasets: [
      {
        name: "invoices",
        source: "Attachment",
        sourceId: "doc_1",
        label: "invoices.xlsx",
        path: "/home/user/data/invoices.xlsx",
        rows: 0,
        bytes: 2048,
      },
    ],
    stdout: "loading\n",
    stderr: "warning: dtype",
    results: [],
    tables: [],
    figures: [],
    layout: [],
    errorName: "ZeroDivisionError",
    errorValue: "division by zero",
    traceback: "tb",
    chartCount: 2,
    failure: "",
    durationMs: 1400,
    costUsd: 0.0003,
    agentRunId: "",
    threadId: "athr_1",
    requestedById: "usr_1",
    startedAt: 1_790_000_000,
    finishedAt: 1_790_000_002,
  };

  it("draws a stored run the way the Desk draws its artifact", () => {
    expect(analysisViewOf(run)).toEqual({
      runId: "aanr_1",
      status: "Errored",
      title: "Lane margin",
      code: "print(1/0)",
      datasets: [
        {
          name: "invoices",
          path: "/home/user/data/invoices.xlsx",
          source: "Attachment",
          label: "invoices.xlsx",
          rows: 0,
        },
      ],
      output: "loading\n[stderr]\nwarning: dtype",
      results: [],
      tables: [],
      figures: [],
      layout: [],
      charts: [
        { position: 1, title: "" },
        { position: 2, title: "" },
      ],
      error: { name: "ZeroDivisionError", value: "division by zero", traceback: "tb" },
      failure: "",
      truncated: false,
    });
  });

  it("shows only stdout when nothing went to stderr", () => {
    expect(analysisViewOf({ ...run, stderr: "" }).output).toBe("loading");
  });
});
