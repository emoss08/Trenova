import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Workspace from "../workspace";

const mocks = vi.hoisted(() => ({
  fetchSettlementWorkspaceSummary: vi.fn(),
  fetchWorkspaceSettlements: vi.fn(),
  generateSettlementBatch: vi.fn(),
}));

vi.mock("@/lib/graphql/driver-settlement", () => ({
  fetchSettlementWorkspaceSummary: mocks.fetchSettlementWorkspaceSummary,
  fetchWorkspaceSettlements: mocks.fetchWorkspaceSettlements,
  generateSettlementBatch: mocks.generateSettlementBatch,
  bulkDriverSettlementAction: vi.fn(),
  submitDriverSettlement: vi.fn(),
  approveDriverSettlement: vi.fn(),
  postDriverSettlement: vi.fn(),
  markDriverSettlementPaid: vi.fn(),
  voidDriverSettlement: vi.fn(),
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn(), info: vi.fn() } }));
vi.mock("@/routes/driver-settlement/_components/settlement-detail", () => ({
  SettlementDetail: () => <div data-testid="settlement-detail" />,
}));
vi.mock("@/components/settlements/bulk-mark-paid-dialog", () => ({
  BulkMarkPaidDialog: () => null,
}));
vi.mock("../driver-context-rail", () => ({ DriverContextRail: () => null }));
vi.mock("../instant-pay-dialog", () => ({ InstantPayDialog: () => null }));
vi.mock("../unsettled-drivers-dialog", () => ({ UnsettledDriversDialog: () => null }));
vi.mock("../workspace-summary", () => ({
  WorkspaceSummaryStrip: ({ actions }: { actions: ReactNode }) => <div>{actions}</div>,
}));

function settlement(over: Record<string, unknown> = {}) {
  return {
    id: "dst_1",
    settlementNumber: "DS-0001",
    status: "Draft",
    workerId: "wrk_1",
    worker: { id: "wrk_1", firstName: "Ada", lastName: "Byrne" },
    netPayMinor: 182_500,
    currencyCode: "USD",
    hasExceptions: false,
    ...over,
  };
}

function renderWorkspace() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <Workspace />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mocks.fetchSettlementWorkspaceSummary.mockResolvedValue({
    periodStart: 1_756_000_000,
    periodEnd: 1_757_000_000,
    unsettledEventCount: 4,
    unsettledWorkerCount: 2,
  });
  mocks.fetchWorkspaceSettlements.mockResolvedValue([]);
  mocks.generateSettlementBatch.mockResolvedValue({ settlementCount: 2 });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("driver settlement workspace empty states", () => {
  it("draws the workspace it will become and offers to generate when pay is waiting", async () => {
    const user = userEvent.setup();
    renderWorkspace();

    expect(await screen.findByText("No settlements this period yet")).toBeInTheDocument();
    expect(screen.getByText(/4 pay events across 2 drivers/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Generate settlements" }));
    await waitFor(() => expect(mocks.generateSettlementBatch).toHaveBeenCalledTimes(1));
  });

  it("does not offer to generate when no pay is waiting", async () => {
    mocks.fetchSettlementWorkspaceSummary.mockResolvedValue({
      periodStart: 1_756_000_000,
      periodEnd: 1_757_000_000,
      unsettledEventCount: 0,
      unsettledWorkerCount: 0,
    });
    renderWorkspace();

    expect(await screen.findByText("No settlements this period yet")).toBeInTheDocument();
    expect(screen.getByText(/accrue automatically/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Generate settlements" })).not.toBeInTheDocument();
  });

  it("says the queue is voided out when every settlement has been voided", async () => {
    mocks.fetchWorkspaceSettlements.mockResolvedValue([settlement({ status: "Voided" })]);
    renderWorkspace();

    expect(await screen.findByText("Nothing to work")).toBeInTheDocument();
    expect(screen.getByText("Nothing open")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  // Needs Review is a chip like any other: an empty review lane is good news,
  // and clearing it brings the whole queue back.
  it("offers to clear a chip that hides every settlement", async () => {
    mocks.fetchWorkspaceSettlements.mockResolvedValue([settlement()]);
    const user = userEvent.setup();
    renderWorkspace();

    expect(await screen.findByText("Ada Byrne")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /^Needs Review/ }));
    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(await screen.findByText("Ada Byrne")).toBeInTheDocument();
  });
});
