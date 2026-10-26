import {
  DeskCarrierCardDocument,
  DeskCarrierEligibilityDocument,
  DeskCarrierIntelDocument,
  DeskCarrierSettlementsDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  fetchCarrierEligibility,
  fetchCarrierIntel,
  fetchCarrierSettlements,
  fetchDeskCarrier,
} from "../carrier-card";

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

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("fetchDeskCarrier", () => {
  it("reads one carrier by id, passing the signal on", async () => {
    const signal = new AbortController().signal;
    requestGraphQLMock.mockResolvedValue({ carrier: { id: "car_1", name: "Red River" } } as never);

    expect(await fetchDeskCarrier("car_1", { signal })).toMatchObject({ id: "car_1" });
    expect(lastCall()).toMatchObject({
      document: DeskCarrierCardDocument,
      operationName: "DeskCarrierCard",
      variables: { id: "car_1" },
      signal,
    });
  });

  it("says a carrier the server does not return could not be found", async () => {
    requestGraphQLMock.mockResolvedValue({ carrier: null } as never);

    await expect(fetchDeskCarrier("car_gone")).rejects.toThrow("Carrier not found");
  });
});

describe("fetchCarrierEligibility", () => {
  it("asks the dispatch preview about the carrier alone", async () => {
    const preview = {
      blockers: [],
      warnings: ["Cargo expires soon"],
      advisories: [],
      findings: [],
    };
    requestGraphQLMock.mockResolvedValue({ dispatchCarrierAssignmentPreview: preview } as never);

    expect(await fetchCarrierEligibility("car_1")).toEqual(preview);
    expect(lastCall()).toMatchObject({
      document: DeskCarrierEligibilityDocument,
      variables: { carrierId: "car_1" },
    });
  });
});

describe("fetchCarrierIntel", () => {
  it("reads the current snapshot, and nothing for a carrier never vetted or not returned", async () => {
    requestGraphQLMock.mockResolvedValue({
      carrier: { id: "car_1", intelligence: { id: "snap_1", riskLevel: "Low" } },
    } as never);
    expect(await fetchCarrierIntel("car_1")).toMatchObject({ id: "snap_1" });
    expect(lastCall()).toMatchObject({
      document: DeskCarrierIntelDocument,
      variables: { id: "car_1" },
    });

    requestGraphQLMock.mockResolvedValue({ carrier: { id: "car_1", intelligence: null } } as never);
    expect(await fetchCarrierIntel("car_1")).toBeNull();

    requestGraphQLMock.mockResolvedValue({ carrier: null } as never);
    expect(await fetchCarrierIntel("car_1")).toBeNull();
  });
});

describe("fetchCarrierSettlements", () => {
  it("filters to the carrier, newest pay date first, counting on the first page only", async () => {
    requestGraphQLMock.mockResolvedValue({
      carrierSettlements: {
        edges: [{ node: { id: "cs_1" } }],
        pageInfo: { hasNextPage: true, endCursor: "c1" },
        totalCount: 12,
        totalCountExact: true,
      },
    } as never);

    const first = await fetchCarrierSettlements("car_1", { first: 5, after: null });

    expect(first).toEqual({
      rows: [{ id: "cs_1" }],
      total: 12,
      totalExact: true,
      endCursor: "c1",
      hasNextPage: true,
    });
    expect(lastCall()).toMatchObject({
      document: DeskCarrierSettlementsDocument,
      variables: {
        input: {
          first: 5,
          after: null,
          fieldFilters: [{ field: "carrierId", operator: "eq", value: "car_1" }],
          sort: [{ field: "payDate", direction: "desc" }],
        },
        includeTotalCount: true,
      },
    });

    requestGraphQLMock.mockResolvedValue({
      carrierSettlements: { edges: [], pageInfo: { hasNextPage: false, endCursor: null } },
    } as never);
    const next = await fetchCarrierSettlements("car_1", { first: 5, after: "c1" });
    expect(next.total).toBeNull();
    expect(lastCall().variables).toMatchObject({ includeTotalCount: false });
  });
});
