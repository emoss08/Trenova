import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clearCsrfToken, setCsrfToken } from "../api";
import { requestGraphQL, resolveGraphQLURL } from "../graphql";

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
    expect(url).toBe("/graphql");
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
});
