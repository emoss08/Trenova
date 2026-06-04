import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clearCsrfToken, setCsrfToken } from "@/lib/api";
import {
  fetchDataTablePage,
  fetchGraphQLData,
} from "../use-data-table-query";
import { equipmentTableGraphQLConfigs } from "@/lib/graphql/equipment-table";

function createJSONResponse(data: unknown): Response {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

describe("data table query fetching", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    clearCsrfToken();
    fetchMock = vi.fn(async () =>
      createJSONResponse({
        data: {
          tractors: {
            edges: [{ node: { id: "tr_1", code: "T-100" } }],
            pageInfo: {
              hasNextPage: true,
              endCursor: "cursor-tr-1",
            },
            totalCount: 12,
          },
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    clearCsrfToken();
    vi.unstubAllGlobals();
  });

  it("builds GraphQL variables from table state and normalizes connection results", async () => {
    setCsrfToken("table-token");

    const result = await fetchDataTablePage({
      link: "/tractors/",
      pageIndex: 2,
      pageSize: 25,
      graphql: equipmentTableGraphQLConfigs.tractor,
      options: {
        query: "T-100",
        fieldFilters: [{ field: "status", operator: "eq", value: "Available" }],
        filterGroups: [
          {
            filters: [{ field: "fleetCode.code", operator: "contains", value: "MID" }],
          },
        ],
        sort: [{ field: "createdAt", direction: "desc" }],
      },
    });

    expect(result).toEqual({
      results: [{ id: "tr_1", code: "T-100" }],
      count: 12,
      next: null,
      prev: null,
      pageInfo: {
        mode: "cursor",
        hasNextPage: true,
        endCursor: "cursor-tr-1",
        totalCount: 12,
      },
    });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(typeof init.body).toBe("string");
    const body = JSON.parse(init.body as string);
    expect(body).toMatchObject({
      operationName: "TractorTable",
      variables: {
        first: 25,
        query: "T-100",
        includeEquipmentDetails: true,
        includeFleetDetails: true,
        includeWorkerDetails: true,
        fieldFilters: [{ field: "status", operator: "eq", value: "Available" }],
        filterGroups: [
          {
            filters: [{ field: "fleetCode.code", operator: "contains", value: "MID" }],
          },
        ],
        sort: [{ field: "createdAt", direction: "desc" }],
      },
    });
    expect(body.variables).not.toHaveProperty("offset");
  });

  it("uses REST fetching when no GraphQL config is provided", async () => {
    fetchMock.mockResolvedValueOnce(
      createJSONResponse({
        results: [{ id: "trl_1", code: "TRL-1" }],
        count: 1,
        next: null,
        prev: null,
      }),
    );

    const result = await fetchDataTablePage({
      link: "/trailers/",
      pageIndex: 1,
      pageSize: 50,
      options: {
        query: "TRL",
        fieldFilters: [{ field: "status", operator: "eq", value: "Available" }],
        filterGroups: [],
        sort: [{ field: "code", direction: "asc" }],
        extraSearchParams: { includeFleetDetails: true },
      },
    });

    expect(result.results).toEqual([{ id: "trl_1", code: "TRL-1" }]);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const requestURL = new URL(url);
    expect(requestURL.pathname).toBe("/api/v1/trailers/");
    expect(requestURL.searchParams.get("limit")).toBe("50");
    expect(requestURL.searchParams.get("offset")).toBe("50");
    expect(requestURL.searchParams.get("query")).toBe("TRL");
    expect(requestURL.searchParams.get("includeFleetDetails")).toBe("true");
    expect(init.credentials).toBe("include");
  });

  it("applies mapNode before returning normalized GraphQL results", async () => {
    setCsrfToken("table-token");

    const result = await fetchGraphQLData(10, {
      document: "query TractorTable { tractors { totalCount } }",
      operationName: "TractorTable",
      connectionKey: "tractors",
      mapNode: (node) => ({ ...(node as { id: string }), mapped: true }),
    });

    expect(result.results).toEqual([{ id: "tr_1", code: "T-100", mapped: true }]);
    expect(result.count).toBe(12);
    expect(result.pageInfo).toMatchObject({
      hasNextPage: true,
      endCursor: "cursor-tr-1",
    });
  });

  it("sends the current cursor instead of an offset for GraphQL pages", async () => {
    setCsrfToken("table-token");

    await fetchGraphQLData(25, equipmentTableGraphQLConfigs.tractor, {
      cursor: "cursor-page-3",
    });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(typeof init.body).toBe("string");
    expect(JSON.parse(init.body as string).variables).toMatchObject({
      first: 25,
      after: "cursor-page-3",
    });
  });
});

describe("tractor and trailer GraphQL table configs", () => {
  it("opts tractors into the tractor connection with required include variables", () => {
    expect(equipmentTableGraphQLConfigs.tractor.connectionKey).toBe("tractors");
    expect(equipmentTableGraphQLConfigs.tractor.variables).toMatchObject({
      includeEquipmentDetails: true,
      includeFleetDetails: true,
      includeWorkerDetails: true,
    });
    expect(equipmentTableGraphQLConfigs.tractor.document).toContain("primaryWorker");
    expect(equipmentTableGraphQLConfigs.tractor.document).toContain("customFields");
    expect(equipmentTableGraphQLConfigs.tractor.document).toContain("totalCount");
    expect(equipmentTableGraphQLConfigs.tractor.document).toContain("pageInfo");
    expect(equipmentTableGraphQLConfigs.tractor.document).toContain("$after: String");
    expect(equipmentTableGraphQLConfigs.tractor.document).not.toContain("$offset");
  });

  it("opts trailers into the trailer connection with required include variables", () => {
    expect(equipmentTableGraphQLConfigs.trailer.connectionKey).toBe("trailers");
    expect(equipmentTableGraphQLConfigs.trailer.variables).toMatchObject({
      includeEquipmentDetails: true,
      includeFleetDetails: true,
    });
    expect(equipmentTableGraphQLConfigs.trailer.document).toContain("lastKnownLocationName");
    expect(equipmentTableGraphQLConfigs.trailer.document).toContain("customFields");
    expect(equipmentTableGraphQLConfigs.trailer.document).toContain("totalCount");
    expect(equipmentTableGraphQLConfigs.trailer.document).toContain("pageInfo");
    expect(equipmentTableGraphQLConfigs.trailer.document).toContain("$after: String");
    expect(equipmentTableGraphQLConfigs.trailer.document).not.toContain("$offset");
  });
});
