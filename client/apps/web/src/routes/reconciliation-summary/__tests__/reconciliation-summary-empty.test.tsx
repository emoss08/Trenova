import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import type { ReconciliationSummary } from "@/types/bank-receipt";
import { cleanup, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ReconciliationSummaryPage } from "../page";

const mocks = vi.hoisted(() => ({ getSummary: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { bankReceiptService: { getSummary: mocks.getSummary } },
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));

function summary(over: Partial<ReconciliationSummary> = {}): ReconciliationSummary {
  return {
    asOfDate: 1_780_000_000,
    importedCount: 0,
    importedAmount: 0,
    matchedCount: 0,
    matchedAmount: 0,
    exceptionCount: 0,
    exceptionAmount: 0,
    activeWorkItemCount: 0,
    assignedWorkItemCount: 0,
    inReviewWorkItemCount: 0,
    exceptionAging: { currentCount: 0, days1To3Count: 0, days4To7Count: 0, daysOver7Count: 0 },
    ...over,
  };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("Reconciliation summary", () => {
  // The loading state is the loaded page drawn in grey: the four figures,
  // the aging table and the work items, so the page does not reflow when the
  // summary lands.
  it("draws the page's own shape while the summary is still being read", () => {
    mocks.getSummary.mockReturnValue(new Promise(() => {}));
    renderAccountingPage(<ReconciliationSummaryPage />);

    const loading = screen.getByLabelText("Loading the summary");
    expect(loading).toHaveAttribute("aria-busy", "true");
    const hidden = { hidden: true } as const;
    const aging = within(loading).getByRole("table", { name: "Exception aging", ...hidden });
    expect(within(aging).getAllByRole("row", hidden).length).toBeGreaterThan(1);
    expect(
      within(loading).getByRole("list", { name: "Work items", ...hidden }),
    ).toBeInTheDocument();
    expect(within(loading).queryByRole("table")).toBeNull();
  });

  // A summary of zeros is a programme nobody has opened, not a clean book.
  // Drawing "0" in every card would read as a result; the sketch says what
  // the page is for and the link is the way forward.
  it("sketches the page and points at importing when nothing has been reconciled", async () => {
    mocks.getSummary.mockResolvedValue(summary());
    renderAccountingPage(<ReconciliationSummaryPage />);

    expect(
      await screen.findByRole("heading", { name: "Nothing to reconcile yet" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Import a bank receipt file/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Import a batch/ })).toHaveAttribute(
      "href",
      "/accounting/reconciliation/import-batches",
    );
    expect(screen.queryByText("Match Rate")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Work Queue/ })).not.toBeInTheDocument();
  });

  // Work can be open with nothing imported in the current window; that is
  // still a programme in use, and the figures belong on the page.
  it("shows the figures whenever any count is above zero", async () => {
    mocks.getSummary.mockResolvedValue(summary({ inReviewWorkItemCount: 1 }));
    renderAccountingPage(<ReconciliationSummaryPage />);

    expect(await screen.findByText("Match Rate")).toBeInTheDocument();
    expect(screen.getByText("0%")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Nothing to reconcile yet" })).toBeNull();
    expect(screen.getByRole("link", { name: /Work Queue/ })).toHaveAttribute(
      "href",
      "/accounting/reconciliation/work-queue",
    );
  });

  it("reads the match rate as matched over imported", async () => {
    mocks.getSummary.mockResolvedValue(
      summary({ importedCount: 8, importedAmount: 80_000, matchedCount: 6, matchedAmount: 60_000 }),
    );
    renderAccountingPage(<ReconciliationSummaryPage />);

    expect(await screen.findByText("75%")).toBeInTheDocument();
    expect(screen.getByText("$800.00")).toBeInTheDocument();
  });
});
