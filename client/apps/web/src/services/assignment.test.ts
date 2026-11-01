import { beforeEach, describe, expect, it, vi } from "vitest";

import { requestGraphQL } from "@trenova/shared/lib/graphql";

import { AssignmentService } from "./assignment";

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL: vi.fn(),
}));

const move = {
  id: "sm_1",
  shipmentId: "shp_1",
  status: "InTransit",
  loaded: true,
  sequence: 0,
  distance: null,
  coverageType: "driver",
  stops: [
    {
      id: "stp_1",
      locationId: "loc_1",
      status: "InTransit",
      type: "Pickup",
      scheduleType: "Open",
      sequence: 0,
      scheduledWindowStart: 1767700000,
      scheduledWindowEnd: null,
      actualArrival: 1767787200,
      actualDeparture: null,
      addressLine: "1 Dock Rd",
      location: null,
    },
  ],
  assignment: null,
  carrierAssignment: null,
};

describe("AssignmentService.recordStopActual", () => {
  beforeEach(() => {
    vi.mocked(requestGraphQL).mockReset();
  });

  it("records a backdated arrival through RecordShipmentStopActual with the idempotency key", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ recordShipmentStopActual: move });

    const service = new AssignmentService();
    const result = await service.recordStopActual(
      "sm_1",
      "stp_1",
      { action: "Arrive", occurredAt: 1767787200 },
      { idempotencyKey: "key-1" },
    );

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "RecordShipmentStopActual",
        idempotencyKey: "key-1",
        variables: {
          moveId: "sm_1",
          stopId: "stp_1",
          input: { action: "Arrive", occurredAt: 1767787200 },
        },
      }),
    );
    expect(result.status).toBe("InTransit");
    expect(result.stops[0]?.actualArrival).toBe(1767787200);
  });

  it("sends no time when recording at the current time", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      recordShipmentStopActual: { ...move, status: "Completed", stops: [] },
    });

    const service = new AssignmentService();
    await service.recordStopActual("sm_1", "stp_2", { action: "Depart" });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "RecordShipmentStopActual",
        idempotencyKey: undefined,
        variables: {
          moveId: "sm_1",
          stopId: "stp_2",
          input: { action: "Depart", occurredAt: null },
        },
      }),
    );
  });
});

describe("AssignmentService assignment writes", () => {
  beforeEach(() => {
    vi.mocked(requestGraphQL).mockReset();
  });

  const assignment = {
    id: "asn_1",
    shipmentMoveId: "sm_1",
    status: "New",
    primaryWorkerId: "wrk_1",
    tractorId: "trc_1",
    trailerId: null,
    secondaryWorkerId: null,
    tractor: { id: "trc_1", code: "T-100" },
    trailer: null,
    primaryWorker: null,
    secondaryWorker: null,
    version: 1,
  };

  it("assigns through AssignShipmentMove, sending absent equipment and co-driver as null", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ assignShipmentMove: assignment });

    const service = new AssignmentService();
    const result = await service.assignToMove("sm_1", {
      primaryWorkerId: "wrk_1",
      tractorId: "trc_1",
      trailerId: "",
      secondaryWorkerId: null,
    });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "AssignShipmentMove",
        variables: {
          moveId: "sm_1",
          input: {
            primaryWorkerId: "wrk_1",
            tractorId: "trc_1",
            trailerId: null,
            secondaryWorkerId: null,
          },
        },
      }),
    );
    expect(result.id).toBe("asn_1");
    expect(result.tractor?.code).toBe("T-100");
  });

  it("replaces coverage through ReassignShipmentMove", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ reassignShipmentMove: assignment });

    const service = new AssignmentService();
    await service.reassign("sm_1", {
      primaryWorkerId: "wrk_1",
      tractorId: "trc_1",
      trailerId: "trl_9",
      secondaryWorkerId: "wrk_2",
    });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "ReassignShipmentMove",
        variables: {
          moveId: "sm_1",
          input: {
            primaryWorkerId: "wrk_1",
            tractorId: "trc_1",
            trailerId: "trl_9",
            secondaryWorkerId: "wrk_2",
          },
        },
      }),
    );
  });

  it("reports compliant drivers as valid", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ checkWorkerCompliance: true });

    const service = new AssignmentService();
    const result = await service.checkWorkerCompliance("sm_1", { primaryWorkerId: "wrk_1" });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "CheckWorkerCompliance",
        variables: {
          input: { shipmentMoveId: "sm_1", primaryWorkerId: "wrk_1", secondaryWorkerId: null },
        },
      }),
    );
    expect(result).toEqual({ valid: true });
  });
});
