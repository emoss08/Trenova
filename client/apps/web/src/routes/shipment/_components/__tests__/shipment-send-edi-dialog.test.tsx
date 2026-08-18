import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Shipment } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ShipmentSendEDIDialog } from "../shipment-send-edi-dialog";

const submitLoadTender = vi.fn().mockResolvedValue({});

vi.mock("@/services/api", () => ({
  apiService: {
    get ediService() {
      return { submitLoadTender };
    },
  },
}));

function renderDialog(shipment: Shipment) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ShipmentSendEDIDialog open onOpenChange={() => {}} shipment={shipment} />
    </QueryClientProvider>,
  );
}

const shipment = {
  id: "sp_1",
  proNumber: "PRO-12345",
  status: "New",
  customer: {
    id: "cus_1",
    name: "EDI Inc",
    ediPartner: { id: "edip_1", name: "Acme Logistics", code: "ACME" },
  },
} as unknown as Shipment;

afterEach(() => {
  cleanup();
  submitLoadTender.mockClear();
});

describe("ShipmentSendEDIDialog", () => {
  it("confirms the customer's linked partner without asking for a partner selection", () => {
    renderDialog(shipment);

    expect(screen.getByText(/Acme Logistics \(ACME\)/)).toBeInTheDocument();
    expect(screen.queryByPlaceholderText(/search edi partners/i)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /send tender/i })).toBeEnabled();
  });

  it("submits the load tender with only the shipment id on confirm", async () => {
    const user = userEvent.setup();
    renderDialog(shipment);

    await user.click(screen.getByRole("button", { name: /send tender/i }));

    expect(submitLoadTender).toHaveBeenCalledTimes(1);
    expect(submitLoadTender).toHaveBeenCalledWith({ sourceShipmentId: "sp_1" });
  });
});
