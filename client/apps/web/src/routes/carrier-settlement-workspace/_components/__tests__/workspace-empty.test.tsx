import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Workspace from "../workspace";

const mocks = vi.hoisted(() => ({
  fetchCarrierSettlementWorkspaceSummary: vi.fn(),
  fetchWorkspaceCarrierSettlements: vi.fn(),
  generateCarrierSettlementBatch: vi.fn(),
}));

vi.mock("@/lib/graphql/carrier-settlement", () => ({
  fetchCarrierSettlementWorkspaceSummary: mocks.fetchCarrierSettlementWorkspaceSummary,
  fetchWorkspaceCarrierSettlements: mocks.fetchWorkspaceCarrierSettlements,
  generateCarrierSettlementBatch: mocks.generateCarrierSettlementBatch,
  invalidateCarrierWorkspace: vi.fn(),
  submitCarrierSettlement: vi.fn(),
  approveCarrierSettlement: vi.fn(),
  postCarrierSettlement: vi.fn(),
  markCarrierSettlementPaid: vi.fn(),
  voidCarrierSettlement: vi.fn(),
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn(), info: vi.fn() } }));
vi.mock("@/routes/carrier-settlement/_components/settlement-detail", () => ({
  CarrierSettlementDetail: () => <div data-testid="settlement-detail" />,
}));
vi.mock("../carrier-context-rail", () => ({ CarrierContextRail: () => null }));
vi.mock("../workspace-summary", () => ({
  WorkspaceSummaryStrip: ({ actions }: { actions: ReactNode }) => <div>{actions}</div>,
}));

function settlement(over: Record<string, unknown> = {}) {
  return {
    id: "cst_1",
    settlementNumber: "CS-0001",
    status: "Draft",
    carrierId: "car_1",
    carrier: { id: "car_1", name: "Knight Logistics", code: "KNGT" },
    netPayableMinor: 125_000,
    currencyCode: "USD",
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
  mocks.fetchCarrierSettlementWorkspaceSummary.mockResolvedValue({
    periodStart: 1_756_000_000,
    periodEnd: 1_757_000_000,
    pendingEventCount: 3,
    pendingCarrierCount: 2,
  });
  mocks.fetchWorkspaceCarrierSettlements.mockResolvedValue([]);
  mocks.generateCarrierSettlementBatch.mockResolvedValue({ settlementCount: 2 });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("carrier settlement workspace empty states", () => {
  // The period is empty but cost is waiting: the way forward is to build the
  // statements, and the button does exactly what the header button does.
  it("draws the workspace it will become and offers to generate when cost is waiting", async () => {
    const user = userEvent.setup();
    renderWorkspace();

    expect(await screen.findByText("No settlements this period yet")).toBeInTheDocument();
    expect(screen.getByText(/3 cost events across 2 carriers/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Generate settlements" }));
    await waitFor(() => expect(mocks.generateCarrierSettlementBatch).toHaveBeenCalledTimes(1));
  });

  it("does not offer to generate when no cost is waiting", async () => {
    mocks.fetchCarrierSettlementWorkspaceSummary.mockResolvedValue({
      periodStart: 1_756_000_000,
      periodEnd: 1_757_000_000,
      pendingEventCount: 0,
      pendingCarrierCount: 0,
    });
    renderWorkspace();

    expect(await screen.findByText("No settlements this period yet")).toBeInTheDocument();
    expect(screen.getByText(/accrue automatically/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Generate settlements" })).not.toBeInTheDocument();
  });

  it("says the queue is voided out when every settlement has been voided", async () => {
    mocks.fetchWorkspaceCarrierSettlements.mockResolvedValue([settlement({ status: "Voided" })]);
    renderWorkspace();

    expect(await screen.findByText("Nothing to work")).toBeInTheDocument();
    expect(screen.getByText("Nothing open")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  it("offers to clear a status filter that hides every settlement", async () => {
    mocks.fetchWorkspaceCarrierSettlements.mockResolvedValue([settlement()]);
    const user = userEvent.setup();
    renderWorkspace();

    expect(await screen.findByText("Knight Logistics")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /^Paid/ }));
    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(await screen.findByText("Knight Logistics")).toBeInTheDocument();
  });
});
