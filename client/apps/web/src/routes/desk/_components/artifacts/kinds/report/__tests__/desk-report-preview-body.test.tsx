import type { AssistantArtifact } from "@/types/assistant";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskReportPreviewBody } from "../desk-report-preview-body";

const fetchers = vi.hoisted(() => ({ rerun: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { assistantService: { runArtifactReport: fetchers.rerun } },
}));

/** A report_preview artifact as the server writes it: columns, rows keyed by label, totals. */
function preview(payload: Record<string, unknown> = {}): AssistantArtifact {
  return {
    id: "art_2",
    threadId: "athr_1",
    messageId: "amsg_1",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "report_preview",
    status: "Ready",
    title: "Revenue by customer",
    payload: {
      name: "Revenue by customer",
      dataset: "invoice",
      columns: [
        { id: "customer", label: "Customer", type: "string" },
        { id: "revenue", label: "Revenue", type: "decimal", format: "money" },
      ],
      rows: [
        { Customer: "Acme Foods", Revenue: "300.00" },
        { Customer: "Globex", Revenue: "100.00" },
      ],
      totals: { Customer: null, Revenue: "400.00" },
      rowCount: 2,
      truncated: false,
      tool: "report_preview",
      reportKey: "revenue-by-customer",
      readWith: { reportKey: "revenue-by-customer" },
      readBy: { id: "agdef_1", name: "Billing Desk", icon: "", accent: "violet" },
      ...payload,
    },
    sourceToolCallId: "call_2",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: "revenue-by-customer",
    turn: "",
    createdAt: 1_793_114_042,
    updatedAt: 1_793_114_042,
  } as AssistantArtifact;
}

function draw(artifact: AssistantArtifact) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskReportPreviewBody artifact={artifact} versions={null} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  seedPermissions({ [Resource.Report]: Operation.Read | Operation.Export });
  fetchers.rerun.mockReset().mockResolvedValue(preview());
});

afterEach(() => {
  clearPermissions();
});

describe("DeskReportPreviewBody", () => {
  it("draws the preview's rows on the run's sheet, ranked, with each share", async () => {
    draw(preview());

    expect(screen.getByText(/Billing Desk previewed it/u)).toBeInTheDocument();
    const table = await screen.findByRole("table");
    const rows = within(table).getAllByRole("row");
    expect(rows).toHaveLength(4);
    expect(rows[1]).toHaveTextContent("Acme Foods");
    expect(rows[1]).toHaveTextContent("75.0%");
    expect(rows[3]).toHaveTextContent("All 2 rows");
    expect(table.querySelectorAll(".dk-rr-cbar i")).toHaveLength(2);
  });

  it("runs the full report it previewed", async () => {
    draw(preview());
    fireEvent.click(await screen.findByRole("button", { name: "Run the full report" }));
    await waitFor(() => expect(fetchers.rerun).toHaveBeenCalledWith("athr_1", "art_2"));
    expect(screen.getByRole("link", { name: "Open in Reports" })).toHaveAttribute(
      "href",
      "/reports/explore?canned=revenue-by-customer",
    );
  });

  it("offers no full run of a definition that was never saved", async () => {
    draw(preview({ reportKey: undefined, readWith: { definition: {} } }));
    await screen.findByRole("table");
    expect(screen.queryByRole("button", { name: "Run the full report" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Open in Reports" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Download CSV" })).toBeInTheDocument();
  });

  it("says when the preview read fewer rows than the report returned", async () => {
    draw(preview({ rowCount: 63 }));
    expect(
      await screen.findByText(
        "The preview shows the first 2 of 63 rows; run the full report for every row.",
      ),
    ).toBeInTheDocument();
  });
});
