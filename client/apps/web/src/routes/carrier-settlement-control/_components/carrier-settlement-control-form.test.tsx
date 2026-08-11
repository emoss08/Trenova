import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Suspense, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import CarrierSettlementControlForm from "./carrier-settlement-control-form";

const mocks = vi.hoisted(() => ({
  fetchCarrierSettlementControl: vi.fn(),
  updateCarrierSettlementControl: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

vi.mock("@/lib/graphql/carrier-settlement", () => ({
  fetchCarrierSettlementControl: mocks.fetchCarrierSettlementControl,
  updateCarrierSettlementControl: mocks.updateCarrierSettlementControl,
}));

vi.mock("@/components/autocomplete-fields", () => ({
  GLAccountAutocompleteField: () => null,
}));

function controlData(overrides: Record<string, unknown> = {}) {
  return {
    id: "carstlc_01",
    organizationId: "org_01",
    businessUnitId: "bu_01",
    payTrigger: "ShipmentDelivered",
    payPeriodFrequency: "Weekly",
    periodEndDayOfWeek: 6,
    payDelayDays: 5,
    autoGenerateBatches: false,
    autoPostOnApprove: false,
    varianceToleranceMinor: 500,
    autoMatchInboundInvoices: false,
    autoAcceptWithinTolerance: false,
    defaultApAccountId: null,
    defaultPurchasedTransportationAccountId: null,
    version: 3,
    createdAt: 1_720_000_000,
    updatedAt: 1_720_000_000,
    ...overrides,
  };
}

function renderForm() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <Suspense fallback={<div>loading</div>}>{children}</Suspense>
      </QueryClientProvider>
    );
  }
  return render(<CarrierSettlementControlForm />, { wrapper: Wrapper });
}

describe("CarrierSettlementControlForm auto-match toggles", () => {
  beforeEach(() => {
    mocks.fetchCarrierSettlementControl.mockReset();
    mocks.updateCarrierSettlementControl.mockReset();
  });

  it("renders both auto-match toggles, disabling auto-accept while auto-match is off", async () => {
    mocks.fetchCarrierSettlementControl.mockResolvedValue(controlData());

    renderForm();

    const autoMatch = await screen.findByRole("switch", { name: "Auto-Match Inbound Invoices" });
    const autoAccept = screen.getByRole("switch", { name: "Auto-Accept Within Tolerance" });
    expect(autoMatch).not.toBeChecked();
    expect(autoAccept).toHaveAttribute("aria-disabled", "true");
  });

  it("enables the auto-accept toggle once auto-match is turned on", async () => {
    mocks.fetchCarrierSettlementControl.mockResolvedValue(controlData());

    renderForm();

    const autoMatch = await screen.findByRole("switch", { name: "Auto-Match Inbound Invoices" });
    await userEvent.click(autoMatch);

    const autoAccept = screen.getByRole("switch", { name: "Auto-Accept Within Tolerance" });
    expect(autoAccept).not.toHaveAttribute("aria-disabled", "true");
  });

  it("keeps a stored auto-accept setting visible when auto-match is already on", async () => {
    mocks.fetchCarrierSettlementControl.mockResolvedValue(
      controlData({ autoMatchInboundInvoices: true, autoAcceptWithinTolerance: true }),
    );

    renderForm();

    const autoAccept = await screen.findByRole("switch", { name: "Auto-Accept Within Tolerance" });
    expect(autoAccept).not.toHaveAttribute("aria-disabled", "true");
    expect(autoAccept).toBeChecked();
  });

  it("submits both flags in the update payload", async () => {
    mocks.fetchCarrierSettlementControl.mockResolvedValue(controlData());
    mocks.updateCarrierSettlementControl.mockResolvedValue({ id: "carstlc_01", version: 4 });

    const { container } = renderForm();

    const autoMatch = await screen.findByRole("switch", { name: "Auto-Match Inbound Invoices" });
    await userEvent.click(autoMatch);
    const autoAccept = screen.getByRole("switch", { name: "Auto-Accept Within Tolerance" });
    await userEvent.click(autoAccept);

    const form = container.querySelector("form");
    expect(form).not.toBeNull();
    fireEvent.submit(form as HTMLFormElement);

    await waitFor(() => expect(mocks.updateCarrierSettlementControl).toHaveBeenCalledTimes(1));
    expect(mocks.updateCarrierSettlementControl).toHaveBeenCalledWith(
      expect.objectContaining({
        version: 3,
        autoMatchInboundInvoices: true,
        autoAcceptWithinTolerance: true,
      }),
    );
  });
});
