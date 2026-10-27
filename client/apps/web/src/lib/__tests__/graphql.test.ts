import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { fetchGraphQLSelectOptions } from "../graphql/select-options";

const selectOptionCursor =
  "eyJjcmVhdGVkQXQiOjE3ODA0MTU4ODMsImlkIjoidHJhY18wMUtUNEdXVDlNS1EwRjZCQ0NHQTBWUjJZNSJ9";

function createGraphQLResponse(data: unknown): Response {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

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
            totalCountExact: true,
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
      totalCountExact: true,
      next: "graphql-select-options://20",
      prev: "graphql-select-options://0",
      pageInfo: {
        mode: "offset",
        hasNextPage: true,
        endCursor: selectOptionCursor,
        totalCount: 3,
        totalCountExact: true,
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

  it("carries an estimated total as inexact", async () => {
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        data: {
          selectOptions: {
            edges: [],
            pageInfo: { hasNextPage: true, endCursor: null },
            totalCount: 1_250_000,
            totalCountExact: false,
          },
        },
      }),
    );

    const response = await fetchGraphQLSelectOptions({ resource: "TRACTOR", page: 1 });

    expect(response.count).toBe(1_250_000);
    expect(response.totalCountExact).toBe(false);
    expect(response.pageInfo?.totalCountExact).toBe(false);
    expect(response.next).toBe("graphql-select-options://20");
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
