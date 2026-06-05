import { describe, expect, it, vi } from "vitest";
import {
  ShipmentCommandCenterTableDocument,
  ShipmentEventsDocument,
  ShipmentSavedViewCountsDocument,
} from "@/graphql/generated/graphql";
import {
  getShipmentSavedViewCountsGraphQL,
  listShipmentEventsGraphQL,
  listShipmentsGraphQL,
} from "../graphql/shipment";

const requestGraphQLMock = vi.hoisted(() => vi.fn());

vi.mock("@/lib/graphql", () => ({
  requestGraphQL: requestGraphQLMock,
}));

describe("shipment GraphQL helpers", () => {
  it("maps shipment table requests to GraphQL variables and normalizes connections", async () => {
    requestGraphQLMock.mockResolvedValueOnce({
      shipments: {
        edges: [{ node: { id: "shp_1", proNumber: "SHP-1" } }],
        pageInfo: {
          hasNextPage: true,
          endCursor: "cursor-1",
        },
        totalCount: 25,
      },
    });

    const response = await listShipmentsGraphQL({
      limit: 20,
      offset: 10,
      query: "SHP",
      fieldFilters: [{ field: "status", operator: "eq", value: "InTransit" }],
      filterGroups: [
        {
          filters: [{ field: "customerId", operator: "eq", value: "cus_1" }],
        },
      ],
      sort: [{ field: "proNumber", direction: "asc" }],
    });

    expect(requestGraphQLMock).toHaveBeenCalledWith({
      document: ShipmentCommandCenterTableDocument,
      operationName: "ShipmentCommandCenterTable",
      variables: {
        first: 20,
        offset: 10,
        query: "SHP",
        fieldFilters: [{ field: "status", operator: "eq", value: "InTransit" }],
        filterGroups: [
          {
            filters: [{ field: "customerId", operator: "eq", value: "cus_1" }],
          },
        ],
        sort: [{ field: "proNumber", direction: "asc" }],
        expandShipmentDetails: true,
      },
    });
    expect(response).toEqual({
      results: [{ id: "shp_1", proNumber: "SHP-1" }],
      count: 25,
      next: "30",
      prev: "0",
      pageInfo: {
        mode: "cursor",
        hasNextPage: true,
        endCursor: "cursor-1",
        totalCount: 25,
      },
    });
  });

  it("requests saved view counts with the user's timezone", async () => {
    requestGraphQLMock.mockResolvedValueOnce({
      shipmentAnalytics: {
        savedViewCounts: {
          all: 12,
          transit: 4,
          atRisk: 2,
          unassigned: 3,
          deliveringToday: 1,
        },
      },
    });

    const counts = await getShipmentSavedViewCountsGraphQL("America/Chicago");

    expect(requestGraphQLMock).toHaveBeenCalledWith({
      document: ShipmentSavedViewCountsDocument,
      operationName: "ShipmentSavedViewCounts",
      variables: { timezone: "America/Chicago" },
    });
    expect(counts).toEqual({
      all: 12,
      transit: 4,
      atRisk: 2,
      unassigned: 3,
      deliveringToday: 1,
    });
  });

  it("requests shipment events with cursor pagination variables", async () => {
    const events = [
      {
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
      },
    ];
    requestGraphQLMock.mockResolvedValueOnce({ shipmentEvents: events });

    const response = await listShipmentEventsGraphQL({
      shipmentId: "shp_1",
      types: ["StatusChanged"],
      limit: 20,
      before: 1_700_000_000,
    });

    expect(requestGraphQLMock).toHaveBeenCalledWith({
      document: ShipmentEventsDocument,
      operationName: "ShipmentEvents",
      variables: {
        shipmentId: "shp_1",
        types: ["StatusChanged"],
        limit: 20,
        before: 1_700_000_000,
      },
    });
    expect(response).toBe(events);
  });
});
