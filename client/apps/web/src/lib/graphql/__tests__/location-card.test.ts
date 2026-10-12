import {
  DeskLocationCardDocument,
  DeskLocationDwellDocument,
  DeskLocationFailuresDocument,
  DeskLocationPoliciesDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  fetchDeskLocation,
  fetchLocationDwell,
  fetchLocationFailures,
  fetchLocationPolicies,
} from "../location-card";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables: Record<string, unknown> & { input?: Record<string, unknown> };
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

function stat(locationId: string, avgDwellMinutes: number) {
  return {
    locationId,
    stopCount: 12,
    breachCount: 3,
    avgDwellMinutes,
    medianDwellMinutes: 80,
    p90DwellMinutes: 190,
    billedAmount: "450",
    waivedAmount: "0",
    disputeCount: 0,
  };
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
  };
}

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("fetchDeskLocation", () => {
  it("reads one location by id, passing the signal on", async () => {
    const signal = new AbortController().signal;
    requestGraphQLMock.mockResolvedValue({
      location: { id: "loc_1", name: "Dallas yard" },
    } as never);

    const location = await fetchDeskLocation("loc_1", { signal });

    expect(location).toMatchObject({ id: "loc_1", name: "Dallas yard" });
    expect(lastCall()).toMatchObject({
      document: DeskLocationCardDocument,
      operationName: "DeskLocationCard",
      variables: { id: "loc_1" },
      signal,
    });
  });

  it("says a location the server does not return could not be found", async () => {
    requestGraphQLMock.mockResolvedValue({ location: null } as never);

    await expect(fetchDeskLocation("loc_gone")).rejects.toThrow("Location not found");
  });
});

describe("fetchLocationDwell", () => {
  it("asks for the one location over the window and reads its row", async () => {
    requestGraphQLMock.mockResolvedValue({
      detentionFacilityStats: [stat("loc_1", 96)],
    } as never);

    const dwell = await fetchLocationDwell("loc_1", { from: 100, to: 200 });

    expect(dwell?.avgDwellMinutes).toBe(96);
    expect(lastCall()).toMatchObject({
      document: DeskLocationDwellDocument,
      variables: { locationId: "loc_1", from: 100, to: 200 },
    });
  });

  it("reads nothing when no stop there was scored, and never another location's row", async () => {
    requestGraphQLMock.mockResolvedValue({ detentionFacilityStats: [] } as never);
    expect(await fetchLocationDwell("loc_1", { from: 100, to: 200 })).toBeNull();

    requestGraphQLMock.mockResolvedValue({
      detentionFacilityStats: [stat("loc_2", 300)],
    } as never);
    expect(await fetchLocationDwell("loc_1", { from: 100, to: 200 })).toBeNull();
  });
});

describe("fetchLocationPolicies", () => {
  it("filters the policies to the location, highest priority first, with how many in all", async () => {
    requestGraphQLMock.mockResolvedValue({
      detentionPolicies: { totalCount: 7, edges: [{ node: { id: "dp_1" } }] },
    } as never);

    const policies = await fetchLocationPolicies("loc_1", 3);

    expect(policies).toEqual({ rows: [{ id: "dp_1" }], total: 7 });
    expect(lastCall()).toMatchObject({
      document: DeskLocationPoliciesDocument,
      variables: {
        input: {
          first: 3,
          fieldFilters: [{ field: "locationId", operator: "eq", value: "loc_1" }],
          sort: [{ field: "priority", direction: "desc" }],
        },
      },
    });
  });

  it("counts the rows it got when the server leaves the total out", async () => {
    requestGraphQLMock.mockResolvedValue({
      detentionPolicies: {
        totalCount: null,
        edges: [{ node: { id: "dp_1" } }, { node: { id: "dp_2" } }],
      },
    } as never);

    expect((await fetchLocationPolicies("loc_1", 3)).total).toBe(2);
  });
});

describe("fetchLocationFailures", () => {
  it("reads the first page newest first with its count, and later pages without one", async () => {
    requestGraphQLMock.mockResolvedValue({
      serviceFailures: {
        edges: [{ node: failure("sf_1") }],
        pageInfo: { hasNextPage: true, endCursor: "c1" },
        totalCount: 9,
      },
    } as never);

    const first = await fetchLocationFailures("loc_1", { first: 5, after: null });

    expect(first).toMatchObject({ total: 9, endCursor: "c1", hasNextPage: true });
    expect(first.rows.map((row) => row.id)).toEqual(["sf_1"]);
    expect(lastCall()).toMatchObject({
      document: DeskLocationFailuresDocument,
      variables: {
        locationId: "loc_1",
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

    const next = await fetchLocationFailures("loc_1", { first: 5, after: "c1" });

    expect(next).toMatchObject({ total: null, endCursor: null, hasNextPage: false });
    expect(lastCall().variables).toMatchObject({
      includeTotalCount: false,
      input: { after: "c1" },
    });
  });
});
