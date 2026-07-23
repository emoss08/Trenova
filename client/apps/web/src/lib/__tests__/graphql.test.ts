import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TractorTableDocument } from "@trenova/graphql/generated/graphql";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { GraphQLRequestError, requestGraphQL, resolveGraphQLURL } from "@trenova/shared/lib/graphql";
import { fetchGraphQLSelectOptions } from "../graphql/select-options";

const selectOptionCursor =
  "eyJjcmVhdGVkQXQiOjE3ODA0MTU4ODMsImlkIjoidHJhY18wMUtUNEdXVDlNS1EwRjZCQ0NHQTBWUjJZNSJ9";

function createGraphQLResponse(data: unknown): Response {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

describe("resolveGraphQLURL", () => {
  it("uses /graphql for relative API base URLs", () => {
    expect(resolveGraphQLURL("/api/v1")).toBe("/graphql");
  });

  it("uses the absolute API origin with /graphql", () => {
    expect(resolveGraphQLURL("https://api.example.com/api/v1")).toBe(
      "https://api.example.com/graphql",
    );
  });
});

describe("requestGraphQL", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    clearCsrfToken();
    fetchMock = vi.fn(async () => createGraphQLResponse({ data: { ok: true } }));
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    clearCsrfToken();
    vi.unstubAllGlobals();
  });

  it("posts JSON with credentials and CSRF headers", async () => {
    setCsrfToken("graphql-token");

    await requestGraphQL({
      document: "query Test { ok }",
      operationName: "Test",
      variables: { first: 10 },
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/graphql?op=Test");
    expect(init.method).toBe("POST");
    expect(init.credentials).toBe("include");

    const headers = new Headers(init.headers);
    expect(headers.get("Content-Type")).toBe("application/json");
    expect(headers.get("X-CSRF-Token")).toBe("graphql-token");
    expect(typeof init.body).toBe("string");
    expect(JSON.parse(init.body as string)).toEqual({
      query: "query Test { ok }",
      operationName: "Test",
      variables: { first: 10 },
    });
  });

  it("labels the request URL with the operation name derived from the document", async () => {
    setCsrfToken("graphql-token");

    await requestGraphQL({
      document: TractorTableDocument,
      variables: { first: 10 },
    });

    const [url] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/graphql?op=TractorTable");
  });

  it("posts generated typed documents as GraphQL strings", async () => {
    setCsrfToken("graphql-token");

    const data = await requestGraphQL({
      document: TractorTableDocument,
      operationName: "TractorTable",
      variables: { first: 10 },
    });

    expect(data).toEqual({ ok: true });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(typeof init.body).toBe("string");
    expect(JSON.parse(init.body as string)).toMatchObject({
      query: expect.stringContaining("query TractorTable"),
      extensions: {
        persistedQuery: {
          version: 1,
          sha256Hash: expect.stringMatching(/^sha256:[a-f0-9]{64}$/),
        },
      },
      operationName: "TractorTable",
      variables: { first: 10 },
    });
  });

  it("throws the first GraphQL error message", async () => {
    setCsrfToken("graphql-token");
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        errors: [{ message: "No tractor access" }],
      }),
    );

    await expect(
      requestGraphQL({
        document: "query Test { tractors { totalCount } }",
        operationName: "Test",
      }),
    ).rejects.toThrow("No tractor access");
  });

  it("preserves structured GraphQL error extensions", async () => {
    setCsrfToken("graphql-token");
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        errors: [
          {
            message: "Validation failed",
            extensions: {
              code: "VALIDATION_ERROR",
              type: "validation",
              traceId: "trace-123",
              params: { shipmentId: "shp_123" },
              errors: { proNumber: ["Pro number is required"] },
              retryable: false,
            },
          },
        ],
      }),
    );

    try {
      await requestGraphQL({
        document: "mutation Test { updateShipment { id } }",
        operationName: "Test",
      });
      expect.fail("Expected requestGraphQL to reject");
    } catch (error: unknown) {
      expect(error).toBeInstanceOf(GraphQLRequestError);
      expect(error).toMatchObject({
        message: "Validation failed",
        status: 200,
        code: "VALIDATION_ERROR",
        type: "validation",
        traceId: "trace-123",
        params: { shipmentId: "shp_123" },
        errors: { proNumber: ["Pro number is required"] },
        extensions: {
          retryable: false,
        },
      });

      const gqlError = error as GraphQLRequestError;
      expect(gqlError.graphQLErrors).toHaveLength(1);
      expect(gqlError.graphQLErrors[0].extensions.retryable).toBe(false);
    }
  });

  it("throws HTTP errors when no GraphQL error is present", async () => {
    setCsrfToken("graphql-token");
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({}), {
        status: 500,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await expect(
      requestGraphQL({
        document: "query Test { ok }",
        operationName: "Test",
      }),
    ).rejects.toMatchObject({
      message: "GraphQL request failed with HTTP 500",
      status: 500,
      graphQLErrors: [],
    });
  });

  it("throws when the response omits data", async () => {
    setCsrfToken("graphql-token");
    fetchMock.mockResolvedValueOnce(createGraphQLResponse({}));

    await expect(
      requestGraphQL({
        document: "query Test { ok }",
        operationName: "Test",
      }),
    ).rejects.toThrow("GraphQL response did not include data");
  });
});

describe("fetchGraphQLSelectOptions", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("graphql-token");
    fetchMock = vi.fn(async () =>
      createGraphQLResponse({
        data: {
          selectOptions: {
            edges: [
              {
                cursor: selectOptionCursor,
                node: {
                  id: "trac_123",
                  label: "TRC-123",
                  description: null,
                  meta: {
                    primaryWorkerId: "wrk_primary",
                    secondaryWorkerId: null,
                  },
                },
              },
            ],
            pageInfo: {
              hasNextPage: true,
              endCursor: selectOptionCursor,
            },
            totalCount: 3,
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

  it("posts select-option variables and normalizes the connection response", async () => {
    const response = await fetchGraphQLSelectOptions({
      resource: "TRACTOR",
      query: "TRC",
      page: 2,
      initialLimit: 10,
      filters: { status: "Available" },
    });

    expect(response).toEqual({
      results: [
        {
          id: "trac_123",
          label: "TRC-123",
          description: null,
          meta: {
            primaryWorkerId: "wrk_primary",
            secondaryWorkerId: null,
          },
        },
      ],
      count: 3,
      next: "graphql-select-options://20",
      prev: "graphql-select-options://0",
      pageInfo: {
        mode: "offset",
        hasNextPage: true,
        endCursor: selectOptionCursor,
        totalCount: 3,
      },
    });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(JSON.parse(init.body as string)).toMatchObject({
      operationName: "SelectOptions",
      variables: {
        input: {
          resource: "TRACTOR",
          query: "TRC",
          first: 10,
          offset: 10,
          filters: { status: "Available" },
        },
      },
    });
  });

  it("uses ids for selected-value lookups and ignores page offset", async () => {
    await fetchGraphQLSelectOptions({
      resource: "WORKER",
      ids: ["wrk_123"],
      page: 4,
      initialLimit: 1,
    });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(JSON.parse(init.body as string)).toMatchObject({
      variables: {
        input: {
          resource: "WORKER",
          first: 1,
          offset: 0,
          ids: ["wrk_123"],
        },
      },
    });
  });
});
