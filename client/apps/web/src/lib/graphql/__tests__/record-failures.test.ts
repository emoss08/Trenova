import { DeskRecordFailuresDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchRecordFailures } from "../record-failures";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables: Record<string, unknown>;
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

function failure(id: string) {
  return {
    id,
    number: id.toUpperCase(),
    type: "LateDelivery",
    status: "Open",
    stopType: "Delivery",
    lateMinutes: 45,
    actualArrival: 1_791_720_000,
    shipment: { id: "shp_1", proNumber: "S1" },
    stop: null,
  };
}

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("fetchRecordFailures", () => {
  it("reads a location's first page newest first with its count, and later pages without one", async () => {
    requestGraphQLMock.mockResolvedValue({
      serviceFailures: {
        edges: [{ node: failure("sf_1") }],
        pageInfo: { hasNextPage: true, endCursor: "c1" },
        totalCount: 9,
      },
    } as never);
    const controller = new AbortController();

    const first = await fetchRecordFailures(
      { kind: "location", id: "loc_1" },
      { first: 5, after: null },
      { signal: controller.signal },
    );

    expect(first).toMatchObject({ total: 9, endCursor: "c1", hasNextPage: true });
    expect(first.rows.map((row) => row.id)).toEqual(["sf_1"]);
    expect(lastCall()).toMatchObject({
      document: DeskRecordFailuresDocument,
      operationName: "DeskRecordFailures",
      signal: controller.signal,
      variables: {
        locationId: "loc_1",
        shipmentId: null,
        input: { first: 5, after: null, sort: [{ field: "detectedAt", direction: "desc" }] },
        includeTotalCount: true,
      },
    });

    requestGraphQLMock.mockResolvedValue({
      serviceFailures: {
        edges: [{ node: failure("sf_2") }],
        pageInfo: { hasNextPage: false, endCursor: null },
      },
    } as never);

    const next = await fetchRecordFailures(
      { kind: "location", id: "loc_1" },
      { first: 5, after: "c1" },
    );

    expect(next).toMatchObject({ total: null, endCursor: null, hasNextPage: false });
    expect(lastCall().variables).toMatchObject({
      includeTotalCount: false,
      input: { after: "c1" },
    });
  });

  it("filters on the shipment for a load's failures, and on no location", async () => {
    requestGraphQLMock.mockResolvedValue({
      serviceFailures: {
        edges: [],
        pageInfo: { hasNextPage: false, endCursor: null },
        totalCount: 0,
      },
    } as never);

    const page = await fetchRecordFailures(
      { kind: "shipment", id: "shp_1" },
      { first: 5, after: null },
    );

    expect(page).toEqual({ rows: [], total: 0, endCursor: null, hasNextPage: false });
    expect(lastCall().variables).toMatchObject({ locationId: null, shipmentId: "shp_1" });
  });

  it("counts the rows it got when the server leaves the total out of the first page", async () => {
    requestGraphQLMock.mockResolvedValue({
      serviceFailures: {
        edges: [{ node: failure("sf_1") }, { node: failure("sf_2") }],
        pageInfo: { hasNextPage: false, endCursor: null },
        totalCount: null,
      },
    } as never);

    const page = await fetchRecordFailures(
      { kind: "shipment", id: "shp_1" },
      { first: 5, after: null },
    );

    expect(page.total).toBe(2);
  });
});
