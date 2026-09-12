import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useShipmentBillingActions } from "../use-shipment-billing-actions";

const shipmentService = vi.hoisted(() => ({
  get: vi.fn(),
  update: vi.fn(),
  transferToBilling: vi.fn(),
}));

const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));

vi.mock("@/services/api", () => ({ apiService: { shipmentService } }));

vi.mock("sonner", () => ({ toast }));

vi.mock("@/lib/queries", () => ({
  queries: {
    shipment: {
      get: (shipmentId: string, params?: Record<string, string>) => ({
        queryKey: ["shipment", "get", shipmentId, params],
      }),
      billingReadiness: (shipmentId: string) => ({
        queryKey: ["shipment", "billing-readiness", shipmentId],
      }),
    },
  },
}));

vi.mock("@/routes/shipment/_components/shipment-queries", () => ({
  SHIPMENT_LIST_KEY: "shipment-list",
}));

const SHIPMENT_ID = "shp_01HTEST0000000000000000001";

const completedShipment = {
  id: SHIPMENT_ID,
  version: 7,
  status: "Completed",
  billingTransferStatus: null,
  proNumber: "PRO-1001",
} as unknown as Shipment;

function setup() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const { result } = renderHook(() => useShipmentBillingActions(), { wrapper });
  return { result, invalidateQueries };
}

function shipmentWith(overrides: Partial<Shipment>): Shipment {
  return { ...completedShipment, ...overrides } as Shipment;
}

beforeEach(() => {
  shipmentService.get.mockResolvedValue(completedShipment);
  shipmentService.update.mockImplementation(async (_id: string, payload: Shipment) => payload);
  shipmentService.transferToBilling.mockResolvedValue({ id: "bqi_1" });
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("useShipmentBillingActions", () => {
  it("marks ready from the freshly fetched shipment and does not transfer", async () => {
    const { result, invalidateQueries } = setup();

    await act(() => result.current.markReadyToBill.run(SHIPMENT_ID));

    expect(shipmentService.get).toHaveBeenCalledWith(SHIPMENT_ID);
    expect(shipmentService.update).toHaveBeenCalledWith(SHIPMENT_ID, {
      ...completedShipment,
      status: "ReadyToInvoice",
    });
    expect(shipmentService.transferToBilling).not.toHaveBeenCalled();
    expect(toast.success).toHaveBeenCalledWith("Shipment marked ready to bill");
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["shipment-list"] });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["shipment", "get", SHIPMENT_ID],
    });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["shipment", "billing-readiness", SHIPMENT_ID],
    });
  });

  it("transfers only after the mark has been saved", async () => {
    const calls: string[] = [];
    shipmentService.update.mockImplementation(async (_id: string, payload: Shipment) => {
      calls.push("update");
      return payload;
    });
    shipmentService.transferToBilling.mockImplementation(async () => {
      calls.push("transfer");
      return { id: "bqi_1" };
    });
    const { result } = setup();

    await act(() => result.current.markReadyAndTransferToBilling.run(SHIPMENT_ID));

    expect(calls).toEqual(["update", "transfer"]);
    expect(shipmentService.transferToBilling).toHaveBeenCalledWith(SHIPMENT_ID);
  });

  it("does not transfer when marking ready fails, and settles without rejecting", async () => {
    shipmentService.update.mockRejectedValue(
      new Error(
        "Shipment cannot be marked ready to invoice until billing requirements are satisfied",
      ),
    );
    const { result, invalidateQueries } = setup();

    await expect(
      act(() => result.current.markReadyAndTransferToBilling.run(SHIPMENT_ID)),
    ).resolves.toBeUndefined();

    expect(shipmentService.transferToBilling).not.toHaveBeenCalled();
    expect(toast.success).not.toHaveBeenCalled();
    expect(toast.error).toHaveBeenCalledWith("Error", {
      description:
        "Shipment cannot be marked ready to invoice until billing requirements are satisfied",
    });
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["shipment-list"] });
  });

  it("reports a failed transfer without rejecting", async () => {
    shipmentService.transferToBilling.mockRejectedValue(
      new Error("Shipment has already been transferred to billing"),
    );
    const { result } = setup();

    await expect(
      act(() => result.current.transferToBilling.run(SHIPMENT_ID)),
    ).resolves.toBeUndefined();

    expect(toast.error).toHaveBeenCalledWith("Error", {
      description: "Shipment has already been transferred to billing",
    });
  });

  it("offers marking ready only for a completed shipment", () => {
    const { result } = setup();
    const { markReadyToBill, markReadyAndTransferToBilling } = result.current;

    for (const status of ["New", "InTransit", "ReadyToInvoice", "Invoiced", "Canceled"] as const) {
      expect(markReadyToBill.isAvailable(shipmentWith({ status }))).toBe(false);
      expect(markReadyAndTransferToBilling.isAvailable(shipmentWith({ status }))).toBe(false);
    }
    expect(markReadyToBill.isAvailable(completedShipment)).toBe(true);
    expect(markReadyAndTransferToBilling.isAvailable(completedShipment)).toBe(true);
  });

  it("offers transfer only for a ready shipment that billing does not already hold", () => {
    const { result } = setup();
    const { transferToBilling } = result.current;

    expect(transferToBilling.isAvailable(completedShipment)).toBe(false);
    expect(
      transferToBilling.isAvailable(
        shipmentWith({ status: "ReadyToInvoice", billingTransferStatus: null }),
      ),
    ).toBe(true);
    expect(
      transferToBilling.isAvailable(
        shipmentWith({ status: "ReadyToInvoice", billingTransferStatus: undefined }),
      ),
    ).toBe(true);
    expect(
      transferToBilling.isAvailable(
        shipmentWith({ status: "ReadyToInvoice", billingTransferStatus: "SentBackToOps" }),
      ),
    ).toBe(true);
    expect(
      transferToBilling.isAvailable(
        shipmentWith({ status: "ReadyToInvoice", billingTransferStatus: "ReadyForReview" }),
      ),
    ).toBe(false);
  });
});
