import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import persistedDocuments from "@trenova/graphql/generated/persisted-documents.json";
import { equipmentTableGraphQLConfigs } from "@/lib/graphql/equipment-table";
import { workerTableGraphQLConfigs } from "@/lib/graphql/worker-table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";
import type { GraphQLExecutableDocument } from "@trenova/shared/types/graphql";
import type { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { afterEach, beforeEach, describe, expect, expectTypeOf, it, vi } from "vitest";
import { fetchDataTablePage, fetchGraphQLData } from "../use-data-table-query";

type EquipmentTypeRow = DataTableConfigRow<typeof equipmentTableGraphQLConfigs.equipmentType>;

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

  it("forwards the abort signal to fetch", async () => {
    setCsrfToken("table-token");
    const controller = new AbortController();

    await fetchDataTablePage({
      pageSize: 25,
      graphql: equipmentTableGraphQLConfigs.equipmentType,
      signal: controller.signal,
    });

    const init = fetchMock.mock.calls.at(-1)?.[1] as RequestInit | undefined;
    expect(init?.signal).toBe(controller.signal);
  });

  it("asks for the total count on the first page", async () => {
    setCsrfToken("table-token");

    await fetchDataTablePage({
      pageSize: 25,
      graphql: equipmentTableGraphQLConfigs.equipmentType,
    });

    expect(requestBody().variables).toMatchObject({ includeTotalCount: true });
    expect(requestBody().variables.input).not.toHaveProperty("includeTotalCount");
  });

  it("skips the total count on cursor pages", async () => {
    setCsrfToken("table-token");

    await fetchGraphQLData(25, equipmentTableGraphQLConfigs.equipmentType, {
      cursor: "cursor-page-3",
    });

    expect(requestBody().variables).toMatchObject({
      includeTotalCount: false,
      input: { after: "cursor-page-3" },
    });
  });

  it("leaves the total count null when the server omits it", async () => {
    setCsrfToken("table-token");
    fetchMock.mockResolvedValueOnce(
      createJSONResponse({
        data: {
          equipmentTypes: {
            edges: [{ node: { id: "et_2", code: "REEFER", class: "Trailer" } }],
            pageInfo: {
              hasNextPage: true,
              endCursor: "cursor-et-2",
            },
          },
        },
      }),
    );

    const result = await fetchGraphQLData(10, equipmentTableGraphQLConfigs.equipmentType, {
      cursor: "cursor-et-1",
    });

    expect(result).toEqual({
      results: [{ id: "et_2", code: "REEFER", class: "Trailer" }],
      count: 1,
      next: null,
      prev: null,
      pageInfo: {
        mode: "cursor",
        hasNextPage: true,
        endCursor: "cursor-et-2",
        totalCount: null,
      },
    });
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

    const mapped = defineDataTableGraphQLConfig({
      ...equipmentTableGraphQLConfigs.equipmentType,
      mapNode: (node) => ({ ...node, mapped: true }),
    });
    const result = await fetchGraphQLData(10, mapped);

    expectTypeOf(result.results[0]).toExtend<EquipmentTypeRow>();
    expectTypeOf(result.results[0].mapped).toEqualTypeOf<boolean>();
    expect(result.results).toEqual([{ id: "et_1", code: "VAN", class: "Trailer", mapped: true }]);
  });

  it("types the page results as the config's unmasked connection node", async () => {
    setCsrfToken("table-token");

    const result = await fetchGraphQLData(10, equipmentTableGraphQLConfigs.equipmentType);
    const page = await fetchDataTablePage({
      pageSize: 10,
      graphql: equipmentTableGraphQLConfigs.equipmentType,
    });

    expectTypeOf(result.results).toEqualTypeOf<EquipmentTypeRow[]>();
    expectTypeOf(page.results).toEqualTypeOf<EquipmentTypeRow[]>();
    expectTypeOf<EquipmentTypeRow["code"]>().toEqualTypeOf<string>();
    expectTypeOf<EquipmentTypeRow>().not.toHaveProperty(" $fragmentRefs");
  });

  it("merges static input extra variables into input, not beside it", async () => {
    setCsrfToken("table-token");
    fetchMock.mockResolvedValueOnce(
      createJSONResponse({
        data: {
          workerPTOEntries: { edges: [], pageInfo: { hasNextPage: false, endCursor: null } },
        },
      }),
    );

    await fetchDataTablePage({ pageSize: 20, graphql: workerTableGraphQLConfigs.pto });

    const { variables } = requestBody();
    expect(variables.input).toMatchObject({ first: 20, includeWorker: true });
    expect(variables).not.toHaveProperty("includeWorker");
  });

  it("resolves function-form input extra variables from the page request", async () => {
    setCsrfToken("table-token");
    fetchMock.mockResolvedValueOnce(
      createJSONResponse({
        data: {
          workerPTOEntries: { edges: [], pageInfo: { hasNextPage: false, endCursor: null } },
        },
      }),
    );

    await fetchDataTablePage({
      pageSize: 5,
      options: { query: "vacation" },
      graphql: defineDataTableGraphQLConfig({
        ...workerTableGraphQLConfigs.pto,
        inputExtraVariables: ({ pageSize, options }) => ({
          includeWorker: pageSize > 1 && options?.query === "vacation",
        }),
      }),
    });

    expect(requestBody().variables.input).toMatchObject({
      first: 5,
      query: "vacation",
      includeWorker: true,
    });
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

// Generated documents carry only their persisted hash, so the SDL under test is read
// back from the safelist the server embeds — the same text the server will execute.
function persistedSdl(document: GraphQLExecutableDocument): string {
  const hash = typeof document === "string" ? undefined : document.__meta__?.hash;
  const sdl = hash ? (persistedDocuments as Record<string, string>)[hash] : undefined;
  if (!sdl) {
    throw new Error("document is not in the persisted safelist");
  }
  return sdl;
}

describe("equipment table GraphQL configs", () => {
  it("defines equipment types with the standard DataTableConnectionInput document", () => {
    const document = persistedSdl(equipmentTableGraphQLConfigs.equipmentType.document);

    expect(equipmentTableGraphQLConfigs.equipmentType.connectionKey).toBe("equipmentTypes");
    expect(equipmentTableGraphQLConfigs.equipmentType).not.toHaveProperty("buildVariables");
    expect(document).toContain("query EquipmentTypeTable");
    expect(document).toContain("$input: DataTableConnectionInput!");
    expect(document).toContain("equipmentTypes(input: $input");
    expect(document).toContain("$includeTotalCount: Boolean = true");
    expect(document).toContain("totalCount @include(if: $includeTotalCount)");
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
