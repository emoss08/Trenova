import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import MatchingWorkspace from "./matching-workspace";

const mocks = vi.hoisted(() => ({
  fetchEdiCarrierInvoices: vi.fn(),
  fetchCarrierInvoiceMatches: vi.fn(),
  fetchCarrierSettlementControl: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn(), info: vi.fn() } }));
vi.mock("./link-carrier-dialog", () => ({ LinkCarrierDialog: () => null }));
vi.mock("@/lib/graphql/carrier-settlement", () => ({
  fetchEdiCarrierInvoices: mocks.fetchEdiCarrierInvoices,
  fetchCarrierInvoiceMatches: mocks.fetchCarrierInvoiceMatches,
  fetchCarrierSettlementControl: mocks.fetchCarrierSettlementControl,
  acceptCarrierInvoiceMatch: vi.fn(),
  acceptCarrierInvoiceMatchWithVariance: vi.fn(),
  createCarrierInvoiceMatch: vi.fn(),
  rejectCarrierInvoiceMatch: vi.fn(),
  suggestCarrierForEdiInvoice: vi.fn(),
}));

function renderWorkspace() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MatchingWorkspace />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  mocks.fetchEdiCarrierInvoices.mockResolvedValue({ items: [] });
  mocks.fetchCarrierInvoiceMatches.mockResolvedValue({ items: [] });
  mocks.fetchCarrierSettlementControl.mockResolvedValue({
    varianceToleranceMinor: 500,
    autoMatchInboundInvoices: true,
    autoAcceptWithinTolerance: false,
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("carrier invoice matching empty states", () => {
  // The list opens on the attention view, so an empty list there is good
  // news, not an empty inbox; showing all is one click away.
  it("says nothing needs attention, then that there are no invoices at all", async () => {
    const user = userEvent.setup();
    renderWorkspace();

    expect(await screen.findByText("Nothing needs attention")).toBeInTheDocument();
    expect(screen.getByText("Nothing open")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(await screen.findByText("No invoices yet")).toBeInTheDocument();
    expect(screen.getByText(/EDI 210/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  it("offers to clear a search that hides every match", async () => {
    const user = userEvent.setup();
    renderWorkspace();

    await user.click(await screen.findByRole("button", { name: /^Matches \(/ }));
    expect(await screen.findByText("No matches yet")).toBeInTheDocument();
    expect(screen.getByText("Nothing open")).toBeInTheDocument();
    expect(screen.getByText(/auto-match sweep/)).toBeInTheDocument();

    const search = screen.getByPlaceholderText("Search invoice, pro number, carrier...");
    await user.type(search, "zzz");
    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(search).toHaveValue("");
    expect(await screen.findByText(/auto-match sweep/)).toBeInTheDocument();
  });
});
