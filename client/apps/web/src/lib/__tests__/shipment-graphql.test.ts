import { describe, expect, it, vi } from "vitest";
import {
  ShipmentCommentsDocument,
  ShipmentCommandCenterTableDocument,
  ShipmentEventsDocument,
  ShipmentSavedViewCountsDocument,
} from "@trenova/graphql/generated/graphql";
import {
  getShipmentSavedViewCountsGraphQL,
  listShipmentCommentsGraphQL,
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
      after: "cursor-0",
      query: "SHP",
      fieldFilters: [{ field: "status", operator: "eq", value: "InTransit" }],
			filterGroups: [
				{
					filters: [{ field: "customerId", operator: "eq", value: "cus_1" }],
				},
			],
		});

    expect(requestGraphQLMock).toHaveBeenCalledWith({
      document: ShipmentCommandCenterTableDocument,
      operationName: "ShipmentCommandCenterTable",
      variables: {
        input: {
          first: 20,
          after: "cursor-0",
          query: "SHP",
          fieldFilters: [{ field: "status", operator: "eq", value: "InTransit" }],
          filterGroups: [
            {
              filters: [{ field: "customerId", operator: "eq", value: "cus_1" }],
            },
          ],
          expandShipmentDetails: true,
        },
      },
    });
    expect(response).toEqual({
      results: [{ id: "shp_1", proNumber: "SHP-1" }],
      count: 25,
      next: "cursor-1",
      prev: null,
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
        input: {
          shipmentId: "shp_1",
          types: ["StatusChanged"],
          limit: 20,
          before: 1_700_000_000,
        },
      },
    });
    expect(response).toBe(events);
  });

  it("normalizes shipment comments with null cursors at list edges", async () => {
    requestGraphQLMock.mockResolvedValueOnce({
      shipmentComments: {
        edges: [
          {
            node: {
              id: "comment_1",
              shipmentId: "shp_1",
              comment: "Follow up with dispatch.",
            },
          },
        ],
        pageInfo: {
          hasNextPage: false,
          endCursor: null,
        },
        totalCount: 1,
      },
    });

    const response = await listShipmentCommentsGraphQL({
      shipmentId: "shp_1",
      limit: 20,
    });

    expect(requestGraphQLMock).toHaveBeenCalledWith({
      document: ShipmentCommentsDocument,
      operationName: "ShipmentComments",
      variables: {
        shipmentId: "shp_1",
        first: 20,
        after: undefined,
      },
    });
    expect(response).toEqual({
      results: [
        {
          id: "comment_1",
          shipmentId: "shp_1",
          comment: "Follow up with dispatch.",
        },
      ],
      count: 1,
      next: null,
      prev: null,
      pageInfo: {
        mode: "cursor",
        hasNextPage: false,
        endCursor: null,
        totalCount: 1,
      },
    });
  });
});
