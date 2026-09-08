import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { BankReceiptBatchDetailPage } from "../detail-page";
import { BankReceiptBatchPage } from "../page";

const mocks = vi.hoisted(() => ({ list: vi.fn(), getById: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { bankReceiptBatchService: { list: mocks.list, getById: mocks.getById } },
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("../_components/import-batch-dialog", () => ({
  ImportBatchDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="import-dialog" /> : null,
}));

const BATCH = {
  id: "brb_1",
  reference: "JULY-ACH",
  source: "Manual",
  status: "Completed",
  importedCount: 0,
  importedAmountMinor: 0,
  matchedCount: 0,
  matchedAmountMinor: 0,
  exceptionCount: 0,
  exceptionAmountMinor: 0,
  createdAt: 1_757_000_000,
  receipts: [],
};

beforeEach(() => {
  mocks.list.mockResolvedValue([]);
  mocks.getById.mockResolvedValue({ ...BATCH, batch: BATCH });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("import batch empty states", () => {
  it("offers the first import when there are no batches", async () => {
    const user = userEvent.setup();
    renderAccountingPage(<BankReceiptBatchPage />);

    expect(await screen.findByText("No batches yet")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Import a batch" }));
    expect(screen.getByTestId("import-dialog")).toBeInTheDocument();
  });

  it("says a batch carried nothing when it has no receipts", async () => {
    renderAccountingPage(
      <Routes>
        <Route path="/batches/:batchId" element={<BankReceiptBatchDetailPage />} />
      </Routes>,
      ["/batches/brb_1"],
    );

    expect(await screen.findByText("No receipts in this batch")).toBeInTheDocument();
  });
});
