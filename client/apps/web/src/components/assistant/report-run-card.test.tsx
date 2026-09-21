import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ useReportRun: vi.fn(), downloadReportRun: vi.fn() }));

vi.mock("@/hooks/use-reports", () => ({
  useReportRun: mocks.useReportRun,
  downloadReportRun: mocks.downloadReportRun,
  isReportRunActive: (status: string) => status === "queued" || status === "running",
}));

import { ReportRunCard } from "./report-run-card";

const run = { runId: "rrun_1", reportKey: "expiring-worker-credentials", reportName: "" };

describe("ReportRunCard", () => {
  beforeEach(() => {
    mocks.useReportRun.mockReset();
  });

  it("names the report while the run is still queued", () => {
    mocks.useReportRun.mockReturnValue({ data: { status: "queued" }, isPending: false });

    render(<ReportRunCard run={run} />);

    expect(screen.getByText("Expiring Worker Credentials")).toBeInTheDocument();
    expect(screen.getByText("Queued — waiting to start.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /download/i })).not.toBeInTheDocument();
  });

  it("prefers the report's own name to a key read as one", () => {
    mocks.useReportRun.mockReturnValue({ data: { status: "queued" }, isPending: false });

    render(<ReportRunCard run={{ ...run, reportKey: "", reportName: "Revenue by customer" }} />);

    expect(screen.getByText("Revenue by customer")).toBeInTheDocument();
  });

  it("offers the download once the run has succeeded", () => {
    mocks.useReportRun.mockReturnValue({
      data: { status: "succeeded", rowCount: 2, truncated: false },
      isPending: false,
    });

    render(<ReportRunCard run={run} />);

    expect(screen.getByText("Finished with 2 rows.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /download/i })).toBeInTheDocument();
  });

  it("says a finished run matched nothing rather than looking unfinished", () => {
    mocks.useReportRun.mockReturnValue({
      data: { status: "succeeded", rowCount: 0 },
      isPending: false,
    });

    render(<ReportRunCard run={run} />);

    expect(screen.getByText("Finished with no matching rows.")).toBeInTheDocument();
  });

  it("gives the reason a run failed", () => {
    mocks.useReportRun.mockReturnValue({
      data: { status: "failed", error: { message: "the dataset timed out" } },
      isPending: false,
    });

    render(<ReportRunCard run={run} />);

    expect(screen.getByText("Failed: the dataset timed out")).toBeInTheDocument();
  });
});
