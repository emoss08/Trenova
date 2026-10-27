import { analysisViewOf } from "@/lib/analysis-run";
import type { AnalysisRun } from "@/types/analysis";
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

// What analysis_run answers when its script showed something, less what only the model
// reads (scriptAnswer in script_tool.go, analysisArtifact in analysis_artifact.go).
const shownChart = {
  kind: "bar",
  title: "Open by customer",
  x: { label: "Customer", kind: "text", values: ["Acme", "Peak", "FreshHaul"] },
  series: [
    { name: "Open", values: [36912.16, null, 6000000] },
    { name: "Billed", values: [40000, 1200, "not a number"] },
  ],
  valueKind: "money",
  currency: "USD",
};

describe("analysisFrom", () => {
  it("reads what analysis_run showed and the datasets it read", () => {
    expect(
      analysisFrom(
        artifactOf({
          analysisRunId: "aanr_1",
          status: "Succeeded",
          title: "Open AR by customer",
          purpose: "Who owes us the most",
          code: "show_chart('bar', x, y)",
          datasets: [
            {
              name: "aging",
              source: "ReportRun",
              label: "AR Aging by Customer",
              rows: 42,
              columns: [{ name: "Customer", kind: "text" }],
            },
            { name: "notes", source: "Attachment", label: "notes.txt", rows: 0, text: true },
          ],
          output: "3 customers\n",
          durationMs: 120,
          charts: [shownChart],
          layout: [{ kind: "chart", index: 0 }],
        }),
      ),
    ).toEqual({
      runId: "aanr_1",
      status: "Succeeded",
      title: "Open AR by customer",
      code: "show_chart('bar', x, y)",
      datasets: [
        {
          name: "aging",
          source: "ReportRun",
          sourceId: "",
          label: "AR Aging by Customer",
          rows: 42,
        },
        { name: "notes", source: "Attachment", sourceId: "", label: "notes.txt", rows: 0 },
      ],
      output: "3 customers\n",
      charts: [
        {
          kind: "bar",
          title: "Open by customer",
          x: { label: "Customer", kind: "text", values: ["Acme", "Peak", "FreshHaul"] },
          series: [
            { name: "Open", values: [36912.16, null, 6000000] },
            { name: "Billed", values: [40000, 1200, null] },
          ],
          valueKind: "money",
          currency: "USD",
          stacked: false,
          horizontal: false,
        },
      ],
      tables: [],
      figures: [],
      layout: [{ kind: "chart", index: 0 }],
      error: null,
      failure: "",
      truncated: false,
    });
  });

  it("reads the tables, figures and order the script showed, and its own title", () => {
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

  it("drops what it cannot read rather than the whole analysis", () => {
    const analysis = analysisFrom(
      artifactOf({ tables: "not a list", charts: [{ kind: "radar" }], datasets: 3 }),
    );

    expect(analysis.tables).toEqual([]);
    expect(analysis.charts).toEqual([]);
    expect(analysis.datasets).toEqual([]);
  });

  it("reads a status it does not know as a failure, never a success", () => {
    expect(analysisFrom(artifactOf({ status: "Running" })).status).toBe("Failed");
    expect(analysisFrom(artifactOf({})).status).toBe("Failed");
  });

  it("keeps the exception and why it stopped", () => {
    const analysis = analysisFrom(
      artifactOf({
        status: "Stopped",
        error: { name: "KeyError", value: "'margin'", traceback: "tb" },
        failure: "",
      }),
    );
    expect(analysis.status).toBe("Stopped");
    expect(analysis.error).toEqual({ name: "KeyError", value: "'margin'", traceback: "tb" });
  });
});

describe("analysisViewOf", () => {
  const run: AnalysisRun = {
    id: "aanr_1",
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
        rows: 12,
      },
    ],
    output: "loading\n",
    tables: [],
    figures: [],
    charts: [],
    layout: [],
    errorName: "ZeroDivisionError",
    errorValue: "division by zero",
    traceback: "tb",
    failure: "",
    durationMs: 1400,
    agentDefinitionId: "",
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
          source: "Attachment",
          sourceId: "doc_1",
          label: "invoices.xlsx",
          rows: 12,
        },
      ],
      output: "loading",
      charts: [],
      tables: [],
      figures: [],
      layout: [],
      error: { name: "ZeroDivisionError", value: "division by zero", traceback: "tb" },
      failure: "",
      truncated: false,
    });
  });

  it("titles a run that has none by its purpose", () => {
    expect(analysisViewOf({ ...run, title: "" }).title).toBe("Margin by lane for Q3");
  });
});
