import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { Shipment, ShipmentBillingReadiness } from "@trenova/shared/types/shipment";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ShipmentBillingReadinessPanel } from "../shipment-billing-readiness-panel";

const mocks = vi.hoisted(() => ({
  queueItems: vi.fn(),
  invoices: vi.fn(),
}));

vi.mock("@/routes/billing-queue/billing-queue-queries", () => ({
  billingQueueItemsByShipmentQuery: (shipmentId: string) => ({
    queryKey: ["billing-queue-by-shipment", shipmentId],
    queryFn: () => mocks.queueItems(shipmentId),
  }),
}));
vi.mock("@/lib/graphql/invoice", () => ({
  fetchInvoiceArContext: vi.fn(),
  fetchInvoicesByShipment: (shipmentId: string) => mocks.invoices(shipmentId),
}));
vi.mock("@/services/api", () => ({ apiService: {} }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

beforeEach(() => {
  mocks.queueItems.mockResolvedValue({
    results: [
      { id: "bqi_intel", billToCustomerId: "cus_intel", status: "Approved" },
      { id: "bqi_amd", billToCustomerId: "cus_amd", status: "OnHold" },
    ],
    count: 2,
  });
  mocks.invoices.mockResolvedValue([
    {
      id: "inv_1",
      number: "INV-1",
      customerId: "cus_intel",
      status: "Posted",
      settlementStatus: "Unpaid",
      totalAmount: "600.00",
      currencyCode: "USD",
      billToName: "Intel",
      billType: "Invoice",
      scope: "Shipment",
      isSplitBill: true,
    },
  ]);
});

function readiness(overrides: Partial<ShipmentBillingReadiness>): ShipmentBillingReadiness {
  return {
    shipmentId: "shp_1",
    shipmentStatus: "ReadyToInvoice",
    policy: {},
    requirements: [],
    missingRequirements: [],
    validationFailures: [],
    warnings: [],
    serviceFailureContext: { hasUnresolved: false, unresolvedCount: 0, serviceFailureIds: [] },
    canMarkReadyToInvoice: true,
    shouldAutoMarkReadyToInvoice: false,
    shouldAutoTransferToBilling: false,
    shouldAutoApproveBilling: false,
    payers: [],
    ...overrides,
  } as unknown as ShipmentBillingReadiness;
}

function renderPanel(value: ShipmentBillingReadiness) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ShipmentBillingReadinessPanel
          readiness={value}
          shipment={{ id: "shp_1", status: "ReadyToInvoice" } as Shipment}
          onUploadRequired={vi.fn()}
          onMarkReadyToInvoice={vi.fn()}
          isMarkingReady={false}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ShipmentBillingReadinessPanel payers", () => {
  // Each payer moves through billing on their own. One has already been
  // invoiced while the other's queue item is on hold; a single shipment-level
  // status would hide exactly the payer who is stuck.
  it("lists each payer with their share and where their billing stands", async () => {
    renderPanel(
      readiness({
        payers: [
          {
            payerId: "cus_intel",
            payerName: "Intel",
            payerCode: "INTEL",
            isPrimary: true,
            shareAmount: 600,
            creditStatus: "Active",
            creditHold: false,
            shouldAutoApproveBilling: true,
          },
          {
            payerId: "cus_amd",
            payerName: "AMD",
            payerCode: "AMD",
            isPrimary: false,
            shareAmount: 400,
            creditStatus: "Hold",
            creditHold: true,
            shouldAutoApproveBilling: false,
          },
        ],
      }),
    );

    const intel = await screen.findByTestId("billing-payer-row-cus_intel");
    expect(intel).toHaveTextContent("INTEL – Intel");
    expect(intel).toHaveTextContent("$600.00");
    expect(await screen.findByRole("link", { name: /INV-1/ })).toHaveAttribute(
      "href",
      "/billing/invoices?item=inv_1",
    );

    const amd = screen.getByTestId("billing-payer-row-cus_amd");
    expect(amd).toHaveTextContent("$400.00");
    expect(amd).toHaveTextContent("Credit hold");
    expect(await screen.findByText("On hold")).toBeInTheDocument();
    expect(mocks.queueItems).toHaveBeenCalledWith("shp_1");
  });

  it("says nothing about payers on an ordinary single-payer shipment", () => {
    renderPanel(
      readiness({
        payers: [
          {
            payerId: "cus_intel",
            payerName: "Intel",
            payerCode: "INTEL",
            isPrimary: true,
            shareAmount: 1000,
            creditStatus: "Active",
            creditHold: false,
            shouldAutoApproveBilling: true,
          },
        ],
      }),
    );

    expect(screen.queryByTestId("billing-payer-rows")).not.toBeInTheDocument();
    expect(mocks.queueItems).not.toHaveBeenCalled();
  });
});
