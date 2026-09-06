import type { ShipmentEventFieldsFragment } from "@trenova/graphql/generated/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { listShipmentEventsGraphQL } from "@/lib/graphql/shipment";
import { ShipmentEventService } from "./shipment-event";

vi.mock("@/lib/graphql/shipment", () => ({
  listShipmentEventsGraphQL: vi.fn(),
}));

vi.mock("sonner", () => ({
  toast: { error: vi.fn() },
}));

const listShipmentEventsGraphQLMock = vi.mocked(listShipmentEventsGraphQL);

type GraphQLEvent = ShipmentEventFieldsFragment[][number];

const ENVELOPE = {
  id: "se_1",
  organizationId: "org_1",
  businessUnitId: "bu_1",
  shipmentId: "shp_1",
  severity: "brand",
  actorType: "system",
  actorId: null,
  actorLabel: "System",
  summary: "Status changed",
  metadata: {},
  occurredAt: 1_800_000_000,
  correlationId: null,
  actor: null,
  shipment: { id: "shp_1", proNumber: "SHP-100" },
} satisfies Omit<GraphQLEvent, "__typename" | "type">;

describe("ShipmentEventService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("delegates to GraphQL and lifts the lifecycle payload", async () => {
    const events: ShipmentEventFieldsFragment[] = [
      {
        ...ENVELOPE,
        __typename: "ShipmentLifecycleEvent",
        type: "StatusChanged",
        proNumber: "SHP-100",
        previousStatus: "New",
        newStatus: "InTransit",
        reason: null,
      },
    ];
    listShipmentEventsGraphQLMock.mockResolvedValueOnce(events);

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
    expect(response).toEqual([
      {
        __typename: "ShipmentLifecycleEvent",
        id: "se_1",
        organizationId: "org_1",
        businessUnitId: "bu_1",
        shipmentId: "shp_1",
        type: "StatusChanged",
        severity: "brand",
        actorType: "system",
        actorId: undefined,
        actorLabel: "System",
        summary: "Status changed",
        metadata: {},
        occurredAt: 1_800_000_000,
        correlationId: undefined,
        actor: undefined,
        shipment: { id: "shp_1", proNumber: "SHP-100" },
        proNumber: "SHP-100",
        previousStatus: "New",
        newStatus: "InTransit",
        reason: undefined,
      },
    ]);
  });

  it("normalizes null envelope and payload fields to undefined", async () => {
    const events: ShipmentEventFieldsFragment[] = [
      {
        ...ENVELOPE,
        __typename: "ShipmentTenderEvent",
        type: "TenderWithdrawn",
        tenderId: "tdr_1",
        offerId: null,
        moveId: null,
        carrierName: null,
        rank: null,
        channel: null,
        source: null,
        reason: "Covered elsewhere",
        action: null,
        mode: null,
        error: null,
        reasons: [],
        warnings: [],
      },
    ];
    listShipmentEventsGraphQLMock.mockResolvedValueOnce(events);

    const [event] = await new ShipmentEventService().list();

    expect(event?.__typename).toBe("ShipmentTenderEvent");
    if (event?.__typename !== "ShipmentTenderEvent") throw new Error("wrong variant");
    expect(event.tenderId).toBe("tdr_1");
    expect(event.reason).toBe("Covered elsewhere");
    expect(event.offerId).toBeUndefined();
    expect(event.moveId).toBeUndefined();
    expect(event.carrierName).toBeUndefined();
    expect(event.rank).toBeUndefined();
    expect(event.channel).toBeUndefined();
    expect(event.reasons).toEqual([]);
    expect(event.warnings).toEqual([]);
    expect(event.actorId).toBeUndefined();
    expect(event.correlationId).toBeUndefined();
    expect(event.actor).toBeUndefined();
  });

  it("lifts the typed payload of every concrete event type", async () => {
    const events: ShipmentEventFieldsFragment[] = [
      {
        ...ENVELOPE,
        __typename: "ShipmentOwnershipEvent",
        type: "OwnershipTransferred",
        proNumber: "SHP-100",
        previousOwnerId: "usr_1",
        newOwnerId: "usr_2",
      },
      {
        ...ENVELOPE,
        __typename: "ShipmentMoveEvent",
        type: "MoveDeparted",
        moveId: "smv_1",
        stopId: null,
        previousStatus: "Assigned",
        newStatus: "InTransit",
      },
      {
        ...ENVELOPE,
        __typename: "ShipmentAssignmentEvent",
        type: "DriverAssigned",
        moveId: "smv_1",
        assignmentId: "asg_1",
        primaryWorkerId: "wrk_1",
        secondaryWorkerId: null,
        tractorId: "trc_1",
        trailerId: null,
        driverName: "S. Ndiaye",
      },
      {
        ...ENVELOPE,
        __typename: "ShipmentCarrierEvent",
        type: "CarrierAssigned",
        moveId: "smv_1",
        carrierId: "car_1",
        carrierName: "Blue Ridge Freight",
        totalCost: "2450.00",
        reason: null,
        proNumber: "BRF-88213",
      },
      {
        ...ENVELOPE,
        __typename: "ShipmentTenderEvent",
        type: "TenderEntrySkipped",
        tenderId: "tdr_1",
        offerId: null,
        moveId: "smv_1",
        carrierName: "Sunset Logistics",
        rank: 3,
        channel: null,
        source: null,
        reason: null,
        action: null,
        mode: null,
        error: null,
        reasons: ["Insurance policy expired", "Compliance status is Unqualified"],
        warnings: [],
      },
      {
        ...ENVELOPE,
        __typename: "ShipmentHoldEvent",
        type: "HoldPlaced",
        holdId: "hld_1",
        holdType: "ComplianceHold",
        holdSeverity: "Blocking",
        holdSource: "Rule",
      },
      {
        ...ENVELOPE,
        __typename: "ShipmentCommentEvent",
        type: "CommentPosted",
        commentId: "scm_1",
        commentBody: "hello @ops",
        commentType: "Dispatch",
        commentVisibility: "Operations",
        commentPriority: "High",
        mentionedUserIds: ["usr_1", "usr_2"],
      },
    ];
    listShipmentEventsGraphQLMock.mockResolvedValueOnce(events);

    const response = await new ShipmentEventService().list();

    expect(response.map((event) => event.__typename)).toEqual([
      "ShipmentOwnershipEvent",
      "ShipmentMoveEvent",
      "ShipmentAssignmentEvent",
      "ShipmentCarrierEvent",
      "ShipmentTenderEvent",
      "ShipmentHoldEvent",
      "ShipmentCommentEvent",
    ]);
    expect(response[0]).toMatchObject({ previousOwnerId: "usr_1", newOwnerId: "usr_2" });
    expect(response[1]).toMatchObject({
      moveId: "smv_1",
      previousStatus: "Assigned",
      newStatus: "InTransit",
    });
    expect(response[2]).toMatchObject({
      assignmentId: "asg_1",
      primaryWorkerId: "wrk_1",
      tractorId: "trc_1",
      driverName: "S. Ndiaye",
    });
    expect(response[3]).toMatchObject({
      carrierId: "car_1",
      carrierName: "Blue Ridge Freight",
      totalCost: "2450.00",
      proNumber: "BRF-88213",
    });
    expect(response[4]).toMatchObject({
      tenderId: "tdr_1",
      moveId: "smv_1",
      rank: 3,
      reasons: ["Insurance policy expired", "Compliance status is Unqualified"],
    });
    expect(response[5]).toMatchObject({
      holdId: "hld_1",
      holdType: "ComplianceHold",
      holdSeverity: "Blocking",
      holdSource: "Rule",
    });
    expect(response[6]).toMatchObject({
      commentId: "scm_1",
      commentBody: "hello @ops",
      commentType: "Dispatch",
      commentVisibility: "Operations",
      commentPriority: "High",
      mentionedUserIds: ["usr_1", "usr_2"],
    });
  });

  it("rejects an event whose __typename is not a known concrete type", async () => {
    listShipmentEventsGraphQLMock.mockResolvedValueOnce([
      {
        ...ENVELOPE,
        __typename: "ShipmentFutureEvent",
        type: "StatusChanged",
      },
    ] as unknown as ShipmentEventFieldsFragment[]);

    await expect(new ShipmentEventService().list()).rejects.toThrow();
  });

  it("rejects a hold event whose hold type is outside the HoldType enum", async () => {
    listShipmentEventsGraphQLMock.mockResolvedValueOnce([
      {
        ...ENVELOPE,
        __typename: "ShipmentHoldEvent",
        type: "HoldPlaced",
        holdId: "hld_1",
        holdType: "Operational",
        holdSeverity: null,
        holdSource: null,
      },
    ] as unknown as ShipmentEventFieldsFragment[]);

    await expect(new ShipmentEventService().list()).rejects.toThrow();
  });
});
