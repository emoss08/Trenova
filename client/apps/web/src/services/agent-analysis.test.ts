import { api } from "@trenova/shared/lib/api";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AgentAnalysisService } from "./agent-analysis";

vi.mock("@trenova/shared/lib/api", () => ({
  api: { get: vi.fn() },
}));

afterEach(() => {
  vi.clearAllMocks();
});

// One analysis as the Go handler writes it: analysis.Run, with a nil id as null and a slice
// nothing was shown in as null.
const stored = {
  id: "aanr_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  status: "Succeeded",
  title: "Lane margin",
  purpose: "Margin by lane for Q3",
  code: "show_figure('Lanes', 2)",
  datasets: null,
  output: "2\n",
  tables: null,
  figures: [{ label: "Lanes", value: 2, kind: "count" }],
  charts: null,
  layout: null,
  errorName: "",
  errorValue: "",
  traceback: "",
  failure: "",
  durationMs: 90,
  agentDefinitionId: null,
  agentRunId: null,
  threadId: null,
  requestedById: "usr_1",
  startedAt: 1_790_000_000,
  finishedAt: 1_790_000_001,
  createdAt: 1_790_000_001,
};

describe("AgentAnalysisService.run", () => {
  it("reads a run with nothing shown and no agent, run or thread as empty", async () => {
    vi.mocked(api.get).mockResolvedValue(stored);

    const run = await new AgentAnalysisService().run("aanr_1");

    expect(api.get).toHaveBeenCalledWith("/agent-analysis/runs/aanr_1/", { signal: undefined });
    expect(run.datasets).toEqual([]);
    expect(run.tables).toEqual([]);
    expect(run.charts).toEqual([]);
    expect(run.layout).toEqual([]);
    expect(run.figures).toEqual([
      { label: "Lanes", value: 2, kind: "count", currency: "", note: "" },
    ]);
    expect(run.agentDefinitionId).toBe("");
    expect(run.agentRunId).toBe("");
    expect(run.threadId).toBe("");
    expect(run.requestedById).toBe("usr_1");
  });
});

describe("AgentAnalysisService.listRuns", () => {
  it("asks for the page after the last run of the one before, by its time and id", async () => {
    vi.mocked(api.get).mockResolvedValue({ items: [], nextBefore: 0, nextBeforeId: null });

    await new AgentAnalysisService().listRuns({
      cursor: { before: 1_790_000_000, beforeId: "aanr_9" },
      limit: 25,
    });

    expect(api.get).toHaveBeenCalledWith(
      "/agent-analysis/runs/?before=1790000000&beforeId=aanr_9&limit=25",
      { signal: undefined },
    );
  });

  it("reads the last page, whose next cursor is empty", async () => {
    vi.mocked(api.get).mockResolvedValue({
      items: [{ ...stored, code: "", output: "", figures: null }],
      nextBefore: 0,
      nextBeforeId: null,
    });

    const page = await new AgentAnalysisService().listRuns({});

    expect(api.get).toHaveBeenCalledWith("/agent-analysis/runs/", { signal: undefined });
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
