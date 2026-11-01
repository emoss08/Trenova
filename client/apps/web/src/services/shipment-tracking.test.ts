import { beforeEach, describe, expect, it, vi } from "vitest";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { ShipmentTrackingService } from "./shipment-tracking";

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL: vi.fn(),
}));

// Shaped from CustomerTrackingView in services/tms/internal/api/graphql/schema/customertracking.graphqls:
// a part the customer's settings hide is an empty string, null or an empty list.
function preview(overrides: Record<string, unknown> = {}) {
  return {
    companyName: "Acme Freight",
    shipmentProNumber: "PRO-1001",
    fields: ["Status", "Windows"],
    expiresAt: 1_800_000_000,
    closed: false,
    status: "InTransit",
    bol: "",
    customerReference: "",
    carrierName: "",
    equipment: "",
    pieces: null,
    weight: null,
    lastLocation: null,
    stops: [
      {
        sequence: 1,
        type: "Pickup",
        city: "Dallas",
        stateCode: "TX",
        locationName: "",
        addressLine: "",
        windowStart: 1_700_000_000,
        windowEnd: null,
        arrivedAt: null,
        departedAt: null,
        estimatedArrival: null,
        running: "",
      },
    ],
    holds: [],
    notes: [],
    proofOfDelivery: null,
    ...overrides,
  };
}

describe("ShipmentTrackingService.previewForShipment", () => {
  beforeEach(() => {
    vi.mocked(requestGraphQL).mockReset();
  });

  it("reads the preview through GraphQL and leaves out every hidden part", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ shipmentCustomerTrackingPreview: preview() });

    const view = await new ShipmentTrackingService().previewForShipment("shp_1");

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "ShipmentCustomerTrackingPreview",
        variables: { shipmentId: "shp_1" },
      }),
    );
    expect(view.status).toBe("InTransit");
    expect(view.bol).toBeUndefined();
    expect(view.carrierName).toBeUndefined();
    expect(view.pieces).toBeUndefined();
    expect(view.lastLocation).toBeUndefined();
    expect(view.holds).toBeUndefined();
    expect(view.notes).toBeUndefined();
    expect(view.stops[0]).toEqual({
      sequence: 1,
      type: "Pickup",
      city: "Dallas",
      stateCode: "TX",
      windowStart: 1_700_000_000,
    });
  });

  it("keeps the parts the settings show, and a page with no status has none", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      shipmentCustomerTrackingPreview: preview({
        status: null,
        bol: "BOL-77",
        pieces: 12,
        holds: [{ type: "Customs", label: "", startedAt: 1_700_000_500 }],
        notes: [{ text: "Gate code 1234", postedAt: 1_700_000_600 }],
      }),
    });

    const view = await new ShipmentTrackingService().previewForShipment("shp_1");

    expect(view.status).toBeUndefined();
    expect(view.bol).toBe("BOL-77");
    expect(view.pieces).toBe(12);
    expect(view.holds).toEqual([{ type: "Customs", startedAt: 1_700_000_500 }]);
    expect(view.notes).toEqual([{ text: "Gate code 1234", postedAt: 1_700_000_600 }]);
  });
});
