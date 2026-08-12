import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { CarrierInvoiceMatchRow } from "@/lib/graphql/carrier-settlement";
import MatchingWorkspace from "./matching-workspace";

const mocks = vi.hoisted(() => ({
  fetchEdiCarrierInvoices: vi.fn(),
  fetchCarrierInvoiceMatches: vi.fn(),
  fetchCarrierSettlementControl: vi.fn(),
}));

vi.mock("sonner", () => ({
  toast: { error: vi.fn(), success: vi.fn(), info: vi.fn() },
}));

vi.mock("./link-carrier-dialog", () => ({
  LinkCarrierDialog: () => null,
}));

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

function match(overrides: Partial<CarrierInvoiceMatchRow> = {}): CarrierInvoiceMatchRow {
  return {
    id: "cim_manual",
    ediCarrierInvoiceId: "edici_01",
    documentAiExtractionId: null,
    carrierId: "car_01",
    carrierAssignmentId: "casn_01",
    carrierSettlementId: null,
    adjustmentCostEventId: null,
    status: "Matched",
    matchedVia: "Manual",
    invoiceNumber: "INV-MANUAL",
    invoiceTotalMinor: 175_075,
    expectedTotalMinor: 175_075,
    varianceMinor: 0,
    currencyCode: "USD",
    resolutionNote: "",
    resolvedById: null,
    resolvedAt: null,
    version: 1,
    createdAt: 1_720_000_000,
    updatedAt: 1_720_000_000,
    carrier: { id: "car_01", code: "KNGT", name: "Knight Logistics", scac: "KNGT" },
    carrierAssignment: null,
    ...overrides,
  } as CarrierInvoiceMatchRow;
}

const AUTO_MATCH = match({
  id: "cim_auto",
  matchedVia: "Auto",
  invoiceNumber: "INV-AUTO",
});

const AUTO_ACCEPTED = match({
  id: "cim_auto_accepted",
  matchedVia: "Auto",
  invoiceNumber: "INV-AUTOACCEPT",
  status: "Resolved",
  resolvedById: null,
  resolvedAt: 1_720_000_500,
});

const MANUALLY_RESOLVED = match({
  id: "cim_manual_resolved",
  matchedVia: "Manual",
  invoiceNumber: "INV-MANRESOLVED",
  status: "Resolved",
  resolvedById: "usr_01",
  resolvedAt: 1_720_000_500,
});

function control(overrides: Record<string, unknown> = {}) {
  return {
    varianceToleranceMinor: 500,
    autoMatchInboundInvoices: true,
    autoAcceptWithinTolerance: false,
    ...overrides,
  };
}

function renderWorkspace() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <MatchingWorkspace />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

async function openMatchesTab() {
  const user = userEvent.setup();
  await user.click(await screen.findByRole("button", { name: /^Matches \(/ }));
  return user;
}

describe("MatchingWorkspace provenance surfaces", () => {
  beforeEach(() => {
    mocks.fetchEdiCarrierInvoices.mockResolvedValue({ items: [] });
    mocks.fetchCarrierInvoiceMatches.mockResolvedValue({ items: [] });
    mocks.fetchCarrierSettlementControl.mockResolvedValue(control());
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("marks auto-created matches and leaves dispatcher-created ones unmarked", async () => {
    mocks.fetchCarrierInvoiceMatches.mockResolvedValue({ items: [AUTO_MATCH, match()] });

    renderWorkspace();
    await openMatchesTab();

    expect(await screen.findByText("Auto-matched")).toBeInTheDocument();
    expect(screen.getAllByText("Auto-matched")).toHaveLength(1);
  });

  it("filters the match queue by provenance", async () => {
    mocks.fetchCarrierInvoiceMatches.mockResolvedValue({ items: [AUTO_MATCH, match()] });

    renderWorkspace();
    const user = await openMatchesTab();
    const list = () => within(screen.getByTestId("carrier-match-list"));

    await waitFor(() => expect(list().getByText("INV-AUTO")).toBeInTheDocument());
    expect(list().getByText("INV-MANUAL")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Auto" }));
    await waitFor(() => expect(list().queryByText("INV-MANUAL")).not.toBeInTheDocument());
    expect(list().getByText("INV-AUTO")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Manual" }));
    await waitFor(() => expect(list().queryByText("INV-AUTO")).not.toBeInTheDocument());
    expect(list().getByText("INV-MANUAL")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "All" }));
    await waitFor(() => expect(list().getByText("INV-AUTO")).toBeInTheDocument());
    expect(list().getByText("INV-MANUAL")).toBeInTheDocument();
  });

  it("marks a match resolved with no actor as auto-accepted, and one with an actor as not", async () => {
    mocks.fetchCarrierInvoiceMatches.mockResolvedValue({
      items: [AUTO_ACCEPTED, MANUALLY_RESOLVED],
    });

    renderWorkspace();
    const user = await openMatchesTab();
    await user.click(screen.getByRole("button", { name: "Resolved" }));

    expect(await screen.findByText("Auto-accepted")).toBeInTheDocument();
    expect(screen.getAllByText("Auto-accepted")).toHaveLength(1);
    expect(screen.getByText("INV-MANRESOLVED")).toBeInTheDocument();
  });

  it("reports the org's automation switches in the header, linked to the control page", async () => {
    mocks.fetchCarrierSettlementControl.mockResolvedValue(
      control({ autoMatchInboundInvoices: true, autoAcceptWithinTolerance: false }),
    );

    renderWorkspace();

    const status = await screen.findByTestId("carrier-match-automation-status");
    expect(status).toHaveTextContent("Auto-match: On");
    expect(status).toHaveTextContent("Auto-accept within tolerance: Off");
    expect(screen.getByRole("link", { name: /automation/i })).toHaveAttribute(
      "href",
      "/admin/carrier-settlement-control",
    );
  });

  it("reports both switches off when the org has not enabled automation", async () => {
    mocks.fetchCarrierSettlementControl.mockResolvedValue(
      control({ autoMatchInboundInvoices: false, autoAcceptWithinTolerance: false }),
    );

    renderWorkspace();

    const status = await screen.findByTestId("carrier-match-automation-status");
    expect(status).toHaveTextContent("Auto-match: Off");
    expect(status).toHaveTextContent("Auto-accept within tolerance: Off");
  });
});
