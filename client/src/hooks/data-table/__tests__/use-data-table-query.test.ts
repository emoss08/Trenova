import { clearCsrfToken, setCsrfToken } from "@/lib/api";
import { equipmentTableGraphQLConfigs } from "@/lib/graphql/equipment-table";
import type { GraphQLRequestError } from "@/lib/graphql";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fetchDataTablePage, fetchGraphQLData } from "../use-data-table-query";

let fetchMock: ReturnType<typeof vi.fn>;

function createJSONResponse(data: unknown, status = 200): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("data table GraphQL query fetching", () => {
  beforeEach(() => {
    clearCsrfToken();
    fetchMock = vi.fn(async () =>
      createJSONResponse({
        data: {
          equipmentTypes: {
            edges: [{ node: { id: "et_1", code: "VAN", class: "Trailer" } }],
            pageInfo: {
              hasNextPage: true,
              endCursor: "cursor-et-1",
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

  it("sends the default DataTableConnectionInput first value", async () => {
    setCsrfToken("table-token");

    await fetchDataTablePage({
      pageSize: 25,
      graphql: equipmentTableGraphQLConfigs.equipmentType,
    });

    const body = requestBody();
    expect(body).toMatchObject({
      operationName: "EquipmentTypeTable",
      variables: {
        input: {
          first: 25,
          fieldFilters: [],
          filterGroups: [],
          sort: [],
        },
      },
    });
    expect(body.variables).not.toHaveProperty("first");
    expect(body.variables.input).not.toHaveProperty("after");
  });

  it("sends the current cursor in input.after", async () => {
    setCsrfToken("table-token");

    await fetchGraphQLData(25, equipmentTableGraphQLConfigs.equipmentType, {
      cursor: "cursor-page-3",
    });

    expect(requestBody().variables.input).toMatchObject({
      first: 25,
      after: "cursor-page-3",
    });
  });

  it("sends search query, field filters, filter groups, and sort in input", async () => {
    setCsrfToken("table-token");

    await fetchDataTablePage({
      pageSize: 15,
      graphql: equipmentTableGraphQLConfigs.equipmentType,
      options: {
        query: "VAN",
        fieldFilters: [{ field: "status", operator: "eq", value: "Active" }],
        filterGroups: [
          {
            filters: [{ field: "class", operator: "in", value: ["Trailer"] }],
          },
        ],
        sort: [{ field: "code", direction: "asc" }],
      },
    });

    expect(requestBody().variables.input).toEqual({
      first: 15,
      query: "VAN",
      fieldFilters: [{ field: "status", operator: "eq", value: "Active" }],
      filterGroups: [
        {
          filters: [{ field: "class", operator: "in", value: ["Trailer"] }],
        },
      ],
      sort: [{ field: "code", direction: "asc" }],
    });
  });

  it("merges static extra variables beside input", async () => {
    setCsrfToken("table-token");

    await fetchDataTablePage({
      pageSize: 20,
      graphql: {
        ...equipmentTableGraphQLConfigs.equipmentType,
        extraVariables: { classes: ["Trailer"] },
      },
    });

    expect(requestBody().variables).toMatchObject({
      input: { first: 20 },
      classes: ["Trailer"],
    });
  });

  it("merges dynamic extra variables beside input", async () => {
    setCsrfToken("table-token");

    await fetchDataTablePage({
      pageSize: 20,
      graphql: {
        ...equipmentTableGraphQLConfigs.equipmentType,
        extraVariables: ({ options }) => ({
          classes: options?.query ? ["Trailer"] : ["Tractor"],
        }),
      },
      options: { query: "VAN" },
    });

    expect(requestBody().variables).toMatchObject({
      input: { first: 20, query: "VAN" },
      classes: ["Trailer"],
    });
  });

  it("normalizes GraphQL cursor connections into table results", async () => {
    setCsrfToken("table-token");

    const result = await fetchGraphQLData(10, equipmentTableGraphQLConfigs.equipmentType);

    expect(result).toEqual({
      results: [{ id: "et_1", code: "VAN", class: "Trailer" }],
      count: 12,
      next: null,
      prev: null,
      pageInfo: {
        mode: "cursor",
        hasNextPage: true,
        endCursor: "cursor-et-1",
        totalCount: 12,
      },
    });
  });

  it("normalizes empty GraphQL pages", async () => {
    setCsrfToken("table-token");
    fetchMock.mockResolvedValueOnce(
      createJSONResponse({
        data: {
          equipmentTypes: {
            edges: [],
            pageInfo: {
              hasNextPage: false,
              endCursor: null,
            },
            totalCount: 0,
          },
        },
      }),
    );

    const result = await fetchGraphQLData(10, equipmentTableGraphQLConfigs.equipmentType);

    expect(result).toEqual({
      results: [],
      count: 0,
      next: null,
      prev: null,
      pageInfo: {
        mode: "cursor",
        hasNextPage: false,
        endCursor: null,
        totalCount: 0,
      },
    });
  });

  it("applies mapNode before returning normalized GraphQL results", async () => {
    setCsrfToken("table-token");

    const result = await fetchGraphQLData(10, {
      ...equipmentTableGraphQLConfigs.equipmentType,
      mapNode: (node) => ({ ...(node as { id: string }), mapped: true }),
    });

    expect(result.results).toEqual([{ id: "et_1", code: "VAN", class: "Trailer", mapped: true }]);
  });

  it("throws GraphQL request errors", async () => {
    setCsrfToken("table-token");
    fetchMock.mockResolvedValueOnce(
      createJSONResponse(
        {
          errors: [
            {
              message: "input.first must be greater than zero",
              extensions: { code: "BAD_USER_INPUT", traceId: "trace-1" },
            },
          ],
        },
        200,
      ),
    );

    await expect(
      fetchGraphQLData(0, equipmentTableGraphQLConfigs.equipmentType),
    ).rejects.toMatchObject({
      name: "GraphQLRequestError",
      message: "input.first must be greater than zero",
      code: "BAD_USER_INPUT",
      traceId: "trace-1",
    } satisfies Partial<GraphQLRequestError>);
  });
});

describe("equipment table GraphQL configs", () => {
  it("defines equipment types with the standard DataTableConnectionInput document", () => {
    const document = equipmentTableGraphQLConfigs.equipmentType.document.toString();

    expect(equipmentTableGraphQLConfigs.equipmentType.connectionKey).toBe("equipmentTypes");
    expect(equipmentTableGraphQLConfigs.equipmentType).not.toHaveProperty("buildVariables");
    expect(document).toContain("query EquipmentTypeTable");
    expect(document).toContain("$input: DataTableConnectionInput!");
    expect(document).toContain("equipmentTypes(input: $input");
    expect(document).toContain("totalCount");
    expect(document).toContain("pageInfo");
    expect(document).not.toContain("$offset: Int");
  });

  it("keeps resource-specific include flags as extra variables", () => {
    expect(equipmentTableGraphQLConfigs.tractor.extraVariables).toMatchObject({
      includeEquipmentDetails: true,
      includeFleetDetails: true,
      includeWorkerDetails: true,
    });
    expect(equipmentTableGraphQLConfigs.trailer.extraVariables).toMatchObject({
      includeEquipmentDetails: true,
      includeFleetDetails: true,
    });
  });
});

function requestBody() {
  const [, init] = fetchMockCall();
  expect(typeof init.body).toBe("string");
  return JSON.parse(init.body as string) as {
    operationName: string;
    variables: Record<string, any>;
  };
}

function fetchMockCall(): [string, RequestInit] {
  return fetchMock.mock.calls[0] as [string, RequestInit];
}
