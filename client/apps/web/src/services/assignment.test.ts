import { describe, expect, it, vi } from "vitest";

import { api } from "@trenova/shared/lib/api";

import { AssignmentService } from "./assignment";

vi.mock("@trenova/shared/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("AssignmentService.recordStopActual", () => {
  it("posts a backdated arrival to the record-actual endpoint and parses the move", async () => {
    vi.mocked(api.post).mockResolvedValue({
      id: "sm_1",
      shipmentId: "shp_1",
      status: "InTransit",
      stops: [
        {
          id: "stp_1",
          locationId: "loc_1",
          status: "InTransit",
          type: "Pickup",
          sequence: 0,
          scheduledWindowStart: 1767700000,
          actualArrival: 1767787200,
          actualDeparture: null,
        },
      ],
    });

    const service = new AssignmentService();
    const move = await service.recordStopActual("sm_1", "stp_1", {
      action: "Arrive",
      occurredAt: 1767787200,
    });

    expect(api.post).toHaveBeenCalledWith("/shipment-moves/sm_1/stops/stp_1/record-actual/", {
      action: "Arrive",
      occurredAt: 1767787200,
    });
    expect(move.status).toBe("InTransit");
    expect(move.stops[0]?.actualArrival).toBe(1767787200);
  });

  it("omits occurredAt when recording at the current time", async () => {
    vi.mocked(api.post).mockResolvedValue({
      id: "sm_1",
      shipmentId: "shp_1",
      status: "Completed",
      stops: [],
    });

    const service = new AssignmentService();
    await service.recordStopActual("sm_1", "stp_2", { action: "Depart" });

    expect(api.post).toHaveBeenCalledWith("/shipment-moves/sm_1/stops/stp_2/record-actual/", {
      action: "Depart",
    });
  });
});
