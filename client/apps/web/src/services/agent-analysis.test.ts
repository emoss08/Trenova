import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AgentAnalysisService } from "./agent-analysis";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

afterEach(() => {
  vi.clearAllMocks();
});

// One analysis as the agentAnalysisRun query answers it: no agent, run or thread is an empty
// id, and nothing shown is an empty list.
const stored = {
  id: "aanr_1",
  status: "Succeeded",
  title: "Lane margin",
  purpose: "Margin by lane for Q3",
  code: "show_figure('Lanes', 2)",
  datasets: [],
  output: "2\n",
  tables: [],
  figures: [{ label: "Lanes", value: 2, kind: "count", currency: "", note: "" }],
  charts: [],
  layout: [],
  errorName: "",
  errorValue: "",
  traceback: "",
  failure: "",
  durationMs: 90,
  agentDefinitionId: "",
  agentRunId: "",
  threadId: "",
  requestedById: "usr_1",
  startedAt: 1_790_000_000,
  finishedAt: 1_790_000_001,
};

describe("AgentAnalysisService.run", () => {
  it("reads one analysis by its id, forwarding the signal", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ agentAnalysisRun: stored });
    const controller = new AbortController();

    const run = await new AgentAnalysisService().run("aanr_1", { signal: controller.signal });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "AgentAnalysisRun",
        variables: { id: "aanr_1" },
        signal: controller.signal,
      }),
    );
    expect(run.datasets).toEqual([]);
    expect(run.tables).toEqual([]);
    expect(run.figures).toEqual([
      { label: "Lanes", value: 2, kind: "count", currency: "", note: "" },
    ]);
    expect(run.agentDefinitionId).toBe("");
    expect(run.requestedById).toBe("usr_1");
  });
});

describe("AgentAnalysisService.listRuns", () => {
  it("asks for the page after the last run of the one before, by its time and id", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      agentAnalysisRuns: { items: [], nextBefore: 0, nextBeforeId: "" },
    });

    await new AgentAnalysisService().listRuns({
      cursor: { before: 1_790_000_000, beforeId: "aanr_9" },
      limit: 25,
    });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "AgentAnalysisRuns",
        variables: { before: 1_790_000_000, beforeId: "aanr_9", limit: 25 },
      }),
    );
  });

  it("asks for the first page with no cursor and reads its summaries", async () => {
    const { code: _code, output: _output, figures: _figures, ...summary } = stored;
    vi.mocked(requestGraphQL).mockResolvedValue({
      agentAnalysisRuns: { items: [summary], nextBefore: 0, nextBeforeId: "" },
    });

    const page = await new AgentAnalysisService().listRuns({});

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        variables: { before: null, beforeId: null, limit: null },
      }),
    );
    expect(page.items).toEqual([
      {
        id: "aanr_1",
        status: "Succeeded",
        title: "Lane margin",
        purpose: "Margin by lane for Q3",
        durationMs: 90,
        agentDefinitionId: "",
        agentRunId: "",
        threadId: "",
        requestedById: "usr_1",
        startedAt: 1_790_000_000,
        finishedAt: 1_790_000_001,
      },
    ]);
    expect(page.nextBeforeId).toBe("");
  });
});
