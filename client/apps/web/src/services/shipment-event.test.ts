import { beforeEach, describe, expect, it, vi } from "vitest";
import { listShipmentEventsGraphQL } from "@/lib/graphql/shipment";
import type { ShipmentEvent } from "@/types/shipment-event";
import { ShipmentEventService } from "./shipment-event";

vi.mock("@/lib/graphql/shipment", () => ({
  listShipmentEventsGraphQL: vi.fn(),
}));

const listShipmentEventsGraphQLMock = vi.mocked(listShipmentEventsGraphQL);

describe("ShipmentEventService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("delegates to GraphQL and preserves shipment event zod parsing", async () => {
    const event: ShipmentEvent = {
      id: "se_1",
      organizationId: "org_1",
      businessUnitId: "bu_1",
      shipmentId: "shp_1",
      type: "StatusChanged",
      severity: "brand",
      actorType: "system",
      actorLabel: "System",
      summary: "Status changed",
      metadata: {},
      occurredAt: 1_800_000_000,
    };
    listShipmentEventsGraphQLMock.mockResolvedValueOnce([event]);

    const response = await new ShipmentEventService().list({
      shipmentId: "shp_1",
      types: ["StatusChanged"],
      limit: 20,
      before: 1_700_000_000,
    });

    expect(listShipmentEventsGraphQLMock).toHaveBeenCalledWith({
      shipmentId: "shp_1",
      types: ["StatusChanged"],
      limit: 20,
      before: 1_700_000_000,
    });
    expect(response).toEqual([event]);
  });

  it("normalizes GraphQL null optional fields during zod parsing", async () => {
    const event = {
      id: "se_1",
      organizationId: "org_1",
      businessUnitId: "bu_1",
      shipmentId: "shp_1",
      moveId: null,
      stopId: null,
      assignmentId: null,
      commentId: null,
      holdId: null,
      type: "StatusChanged",
      severity: "brand",
      actorType: "system",
      actorId: null,
      actorLabel: "System",
      summary: "Status changed",
      metadata: {},
      occurredAt: 1_800_000_000,
      correlationId: null,
      actor: null,
      shipment: {
        id: "shp_1",
        proNumber: "SHP-100",
      },
    };
    listShipmentEventsGraphQLMock.mockResolvedValueOnce([
      event,
    ] as unknown as ShipmentEvent[]);

    const response = await new ShipmentEventService().list();

    expect(response).toHaveLength(1);
    expect(response[0]).toMatchObject({
      id: "se_1",
      shipmentId: "shp_1",
      type: "StatusChanged",
      severity: "brand",
      actorType: "system",
      actorLabel: "System",
      summary: "Status changed",
      metadata: {},
      occurredAt: 1_800_000_000,
      shipment: {
        id: "shp_1",
        proNumber: "SHP-100",
      },
    });
    expect(response[0]?.moveId).toBeUndefined();
    expect(response[0]?.stopId).toBeUndefined();
    expect(response[0]?.assignmentId).toBeUndefined();
    expect(response[0]?.commentId).toBeUndefined();
    expect(response[0]?.holdId).toBeUndefined();
    expect(response[0]?.actorId).toBeUndefined();
    expect(response[0]?.correlationId).toBeUndefined();
    expect(response[0]?.actor).toBeUndefined();
  });
});
