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
