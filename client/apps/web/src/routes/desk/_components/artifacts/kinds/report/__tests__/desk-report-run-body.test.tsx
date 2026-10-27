import type { ReportRunDeskCard } from "@/lib/graphql/report-desk-card";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskReportRunBody } from "../desk-report-run-body";
import { finishedCard, runArtifact } from "./fixtures";

const fetchers = vi.hoisted(() => ({
  card: vi.fn(),
  rerun: vi.fn(),
  cancel: vi.fn(),
  download: vi.fn(),
}));

vi.mock("@/lib/graphql/report-desk-card", () => ({
  fetchReportRunDeskCard: fetchers.card,
}));
vi.mock("@/services/api", () => ({
  apiService: { assistantService: { runArtifactReport: fetchers.rerun } },
}));
vi.mock("@/lib/graphql/reports", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/reports")>()),
  cancelReportRun: fetchers.cancel,
}));
vi.mock("@/hooks/use-reports", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/use-reports")>()),
  downloadReportRun: fetchers.download,
}));

function draw(card: ReportRunDeskCard, payload: Record<string, unknown> = {}) {
  fetchers.card.mockResolvedValue(card);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskReportRunBody artifact={runArtifact(payload)} versions={null} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function withRun(overrides: Partial<ReportRunDeskCard["run"]>, card = finishedCard()) {
  return { ...card, run: { ...card.run, ...overrides } };
}

beforeEach(() => {
  seedPermissions({ [Resource.Report]: Operation.Read | Operation.Export });
  fetchers.rerun.mockReset().mockResolvedValue(runArtifact());
  fetchers.cancel.mockReset();
  fetchers.download.mockReset();
});

afterEach(() => {
  clearPermissions();
});

describe("DeskReportRunBody", () => {
  it("shows what was asked, how the run went and the first page of its result", async () => {
    const { container } = draw(finishedCard());

    expect(await screen.findByText("Finished")).toBeInTheDocument();
    expect(screen.getByText("Billing · Built in")).toBeInTheDocument();
    expect(
      screen.getByText("Freight, accessorial and total charges per customer."),
    ).toBeInTheDocument();
    expect(screen.getByText(/Dispatch Desk ran it for you/u)).toBeInTheDocument();
    const chip = screen.getByText("Window (days)").parentElement as HTMLElement;
    expect(chip).toHaveTextContent("Window (days)30 · Sep 26 – Oct 25");

    const stamp = container.querySelector(".dk-rr-stamp") as HTMLElement;
    expect(within(stamp).getByText("214 rows")).toBeInTheDocument();
    expect(within(stamp).getByText("3.2 s")).toBeInTheDocument();
    expect(within(stamp).getByText(/^XLSX · 48(\.0)? KB$/u)).toBeInTheDocument();

    const table = await screen.findByRole("table");
    const rows = within(table).getAllByRole("row");
    // The header, the eight rows a sheet shows, and the totals.
    expect(rows).toHaveLength(10);
    expect(rows[1]).toHaveTextContent("Acme Foods");
    expect(rows[1]).toHaveTextContent("$241,880.50");
    expect(rows[1]).toHaveTextContent("18.8%");
    expect(rows[9]).toHaveTextContent("All 214 rows");
    expect(rows[9]).toHaveTextContent("$1,284,310.42");
    expect(screen.getByText("206 more rows are in the file")).toBeInTheDocument();
    expect(screen.getByText("Ranked by total revenue")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open in Reports" })).toHaveAttribute(
      "href",
      "/reports/explore?canned=revenue-by-customer",
    );
  });

  it("puts each total beside the same total in the run before", async () => {
    const { container } = draw(finishedCard());
    await screen.findByRole("table");

    const kpis = [...container.querySelectorAll<HTMLElement>(".dk-rr-kpi")];
    expect(kpis.map((kpi) => kpi.querySelector("span")?.textContent)).toEqual([
      "Shipments",
      "Total revenue",
      "Rows",
    ]);
    expect(kpis[1]).toHaveTextContent("$1,284,310.42");
    expect(kpis[1].querySelector(".dk-rr-delta")).toHaveAttribute("data-direction", "up");
    expect(kpis[1]).toHaveTextContent("8.2%");
    expect(kpis[1]).toHaveTextContent(/vs Oct \d+, 2026/u);
    expect(kpis[2]).toHaveTextContent("214");
    expect(kpis[2]).toHaveTextContent("2.9%");
  });

  it("downloads the file and runs the report again", async () => {
    draw(finishedCard());
    fireEvent.click(await screen.findByRole("button", { name: "Download XLSX" }));
    expect(fetchers.download).toHaveBeenCalledWith({ id: "rrun_1" });

    fireEvent.click(screen.getByRole("button", { name: "Run again" }));
    await waitFor(() => expect(fetchers.rerun).toHaveBeenCalledWith("athr_1", "art_1"));
  });

  it("offers no run again to someone who may not run reports", async () => {
    seedPermissions({ [Resource.Report]: Operation.Read });
    draw(finishedCard());
    await screen.findByRole("table");
    expect(screen.queryByRole("button", { name: "Run again" })).not.toBeInTheDocument();
  });

  it("follows a run still going and offers to cancel it", async () => {
    draw(
      withRun(
        { status: "running", completedAt: null, rowCount: 0, byteSize: 0 },
        { ...finishedCard(), sample: null },
      ),
    );

    expect(await screen.findByText("Running")).toBeInTheDocument();
    expect(screen.getByLabelText("Running the report")).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByRole("button", { name: "Run again" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel run" }));
    await waitFor(() => expect(fetchers.cancel.mock.calls[0]?.[0]).toBe("rrun_1"));
  });

  it("says why a run failed, with its code, and leads with running it again", async () => {
    draw(
      withRun(
        {
          status: "failed",
          rowCount: 0,
          byteSize: 0,
          error: {
            code: "QUERY_TIMEOUT",
            message: "The report took longer than its 30 second limit.",
            detail: null,
          },
        },
        { ...finishedCard(), sample: null },
      ),
    );

    expect(
      await screen.findByText("The report took longer than its 30 second limit."),
    ).toBeInTheDocument();
    expect(screen.getByText("QUERY_TIMEOUT")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Download/u })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Run again" })).toBeInTheDocument();
  });

  it("withholds the result and the file from someone who lost access", async () => {
    draw({ ...finishedCard(), sample: null, previous: null, withheld: "access" });

    expect(
      await screen.findByText("You can no longer read everything this report shows."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Download/u })).not.toBeInTheDocument();
  });

  it("warns that a capped run's figures are of the kept rows", async () => {
    draw(withRun({ truncated: true, rowCount: 50000 }));

    expect(
      await screen.findByText("Capped at 50,000 rows. This isn't the whole set."),
    ).toBeInTheDocument();
    await screen.findByRole("table");
    expect(screen.getAllByText("Kept rows only").length).toBeGreaterThan(0);
    expect(screen.getByText("Rows kept")).toBeInTheDocument();
  });

  it("says a run's file is gone once it expired", async () => {
    draw(withRun({ status: "expired" }, { ...finishedCard(), sample: null }));

    expect(await screen.findByText("Expired")).toBeInTheDocument();
    expect(screen.getByText(/This run's file and rows were removed on/u)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Download/u })).not.toBeInTheDocument();
  });

  it("names the person when they ran it again themselves", async () => {
    draw(finishedCard(), { ranByPerson: true });
    expect(await screen.findByText(/^You ran it · /u)).toBeInTheDocument();
    expect(screen.queryByText(/Dispatch Desk ran it/u)).not.toBeInTheDocument();
  });

  it("says a finished run that matched nothing is an answer", async () => {
    const card = finishedCard();
    draw({
      ...withRun({ rowCount: 0 }, card),
      sample: { ...card.sample!, rows: [], totals: null, more: false },
    });
    expect(await screen.findByText("Nothing matched")).toBeInTheDocument();
  });
});
