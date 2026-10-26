import type { AnalysisRun, AnalysisRunSummary, AnalysisRunsResponse } from "@/types/analysis";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

const listRuns =
  vi.fn<
    (params: {
      cursor?: { before: number; beforeId: string } | null;
      limit?: number;
    }) => Promise<AnalysisRunsResponse>
  >();
const runOf = vi.fn<(runId: string) => Promise<AnalysisRun>>();

vi.mock("@/services/api", () => ({
  apiService: {
    agentAnalysisService: {
      listRuns: (params: Parameters<typeof listRuns>[0]) => listRuns(params),
      run: (runId: string) => runOf(runId),
    },
  },
}));

const { AnalysisRuns } = await import("../analysis-runs");

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function summary(id: string, title: string, startedAt: number): AnalysisRunSummary {
  return {
    id,
    status: "Succeeded",
    title,
    purpose: "",
    durationMs: 420,
    agentDefinitionId: "",
    agentRunId: "",
    threadId: "athr_1",
    requestedById: "usr_1",
    startedAt,
    finishedAt: startedAt + 1,
  };
}

function whole(run: AnalysisRunSummary): AnalysisRun {
  return {
    ...run,
    code: "show_figure('Open', 2)",
    datasets: [],
    output: "",
    tables: [],
    figures: [{ label: "Open invoices", value: 2, kind: "count", currency: "", note: "" }],
    charts: [],
    layout: [],
    errorName: "",
    errorValue: "",
    traceback: "",
    failure: "",
  };
}

function renderRuns(search = "") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter searchParams={search}>{children}</NuqsTestingAdapter>
    </QueryClientProvider>
  );
  return render(<AnalysisRuns />, { wrapper });
}

describe("AnalysisRuns", () => {
  it("pages to older analyses from the last one shown, by its time and id", async () => {
    listRuns
      .mockResolvedValueOnce({
        items: [summary("aanr_2", "Open AR by customer", 1_800_000_100)],
        nextBefore: 1_800_000_100,
        nextBeforeId: "aanr_2",
      })
      .mockResolvedValueOnce({
        items: [summary("aanr_1", "Lane margin", 1_800_000_000)],
        nextBefore: 0,
        nextBeforeId: "",
      });
    renderRuns();

    expect(await screen.findByText("Open AR by customer")).toBeTruthy();
    await userEvent.click(screen.getByRole("button", { name: "Show older analyses" }));

    expect(await screen.findByText("Lane margin")).toBeTruthy();
    expect(listRuns).toHaveBeenNthCalledWith(1, { cursor: null, limit: 25 });
    expect(listRuns).toHaveBeenNthCalledWith(2, {
      cursor: { before: 1_800_000_100, beforeId: "aanr_2" },
      limit: 25,
    });
    expect(screen.queryByRole("button", { name: "Show older analyses" })).toBeNull();
  });

  it("reads an analysis whole only when it is opened", async () => {
    const run = summary("aanr_2", "Open AR by customer", 1_800_000_100);
    listRuns.mockResolvedValue({ items: [run], nextBefore: 0, nextBeforeId: "" });
    runOf.mockResolvedValue(whole(run));
    renderRuns();

    await userEvent.click(await screen.findByRole("button", { name: /Open AR by customer/ }));

    expect(await screen.findByText("Open invoices")).toBeTruthy();
    expect(runOf).toHaveBeenCalledWith("aanr_2");
  });

  it("opens a linked analysis that is older than the pages loaded", async () => {
    const old = summary("aanr_0", "Fuel outliers", 1_700_000_000);
    listRuns.mockResolvedValue({
      items: [summary("aanr_2", "Open AR by customer", 1_800_000_100)],
      nextBefore: 1_800_000_100,
      nextBeforeId: "aanr_2",
    });
    runOf.mockResolvedValue(whole(old));
    renderRuns("?analysisRun=aanr_0");

    expect(await screen.findByText("Fuel outliers")).toBeTruthy();
    expect(await screen.findByText("Open invoices")).toBeTruthy();
    expect(runOf).toHaveBeenCalledWith("aanr_0");
  });

  it("says a linked analysis was not found when it cannot be read", async () => {
    listRuns.mockResolvedValue({ items: [], nextBefore: 0, nextBeforeId: "" });
    runOf.mockRejectedValue(new Error("not found"));
    renderRuns("?analysisRun=aanr_0");

    expect(await screen.findByText("The analysis you opened was not found.")).toBeTruthy();
  });

  it("says when there are no analyses yet", async () => {
    listRuns.mockResolvedValue({ items: [], nextBefore: 0, nextBeforeId: "" });
    renderRuns();

    expect(
      await screen.findByText(
        "No analyses yet. Each script an agent runs that shows someone a figure, table or chart is listed here.",
      ),
    ).toBeTruthy();
  });
});
