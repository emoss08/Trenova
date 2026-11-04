import type { ShipmentUpdateInput } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it, vi } from "vitest";
import { updateShipmentGraphQL } from "../shipment";

const mocks = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL: mocks.requestGraphQL,
}));

afterEach(() => {
  vi.clearAllMocks();
});

function stop(handling: "Live" | "Drop") {
  return {
    id: "stp_1",
    shipmentMoveId: "smv_1",
    locationId: "loc_1",
    status: "New",
    type: "Pickup",
    scheduleType: "Appointment",
    handling,
    sequence: 0,
    scheduledWindowStart: 1_900_000_000,
    scheduledWindowEnd: null,
    addressLine: "",
  };
}

describe("saving a shipment's stops", () => {
  it("sends each stop's handling, so detention can tell a drop from a live stop", async () => {
    mocks.requestGraphQL.mockResolvedValue({
      updateShipment: { id: "shp_1", moves: [], additionalCharges: [], freightAllocations: [] },
    });

    await updateShipmentGraphQL("shp_1", {
      moves: [{ id: "smv_1", stops: [stop("Drop"), stop("Live")] }],
    } as unknown as ShipmentUpdateInput);

    const [{ variables }] = mocks.requestGraphQL.mock.calls[0] ?? [];
    expect(variables.input.moves[0].stops.map((s: { handling: string }) => s.handling)).toEqual([
      "Drop",
      "Live",
    ]);
  });
});
