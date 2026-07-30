import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import {
  documentContainsMutation,
  GraphQLRequestError,
  requestGraphQL,
  requestGraphQLResult,
  resolveGraphQLURL,
  setPartialErrorReporter,
  setSessionExpiryHandler,
} from "@trenova/shared/lib/graphql";

const PROBLEM_BASE = "https://trenova.app/problems/";

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
      document: "query TractorTable($first: Int) { tractors { totalCount } }",
      variables: { first: 10 },
    });

    const [url] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/graphql?op=TractorTable");
  });

  // A persisted document is a String object carrying __meta__.hash, which is what the
  // codegen client preset emits. Constructed here rather than imported from
  // @trenova/graphql so the shared package does not depend on the app's generated output.
  it("promotes a document's persisted hash into the request extensions", async () => {
    setCsrfToken("graphql-token");

    const document = Object.assign(
      new String("query TractorTable($first: Int) { tractors { totalCount } }"),
      { __meta__: { hash: `sha256:${"a".repeat(64)}` } },
    ) as unknown as string;

    const data = await requestGraphQL({
      document,
      operationName: "TractorTable",
      variables: { first: 10 },
    });

    expect(data).toEqual({ ok: true });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(JSON.parse(init.body as string)).toMatchObject({
      query: expect.stringContaining("query TractorTable"),
      extensions: {
        persistedQuery: { version: 1, sha256Hash: `sha256:${"a".repeat(64)}` },
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

describe("requestGraphQL partial success", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("graphql-token");
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    clearCsrfToken();
    setPartialErrorReporter(null);
    vi.unstubAllGlobals();
  });

  it("returns data when the response carries data and errors together", async () => {
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        data: { shipment: { id: "shp_1", customer: null } },
        errors: [
          {
            message: "Customer is not visible to this role",
            path: ["shipment", "customer"],
            extensions: {
              code: "FORBIDDEN",
              type: `${PROBLEM_BASE}authorization-error`,
              traceId: "trace-partial",
            },
          },
        ],
      }),
    );

    const data = await requestGraphQL<{
      shipment: { id: string; customer: null };
    }>({
      document: "query Test { shipment { id customer { name } } }",
      operationName: "Test",
    });

    expect(data).toEqual({ shipment: { id: "shp_1", customer: null } });
  });

  it("exposes the accompanying errors through requestGraphQLResult", async () => {
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        data: { shipment: { id: "shp_1", customer: null } },
        errors: [
          {
            message: "Customer is not visible to this role",
            path: ["shipment", "customer"],
            extensions: { code: "FORBIDDEN", type: `${PROBLEM_BASE}authorization-error` },
          },
        ],
      }),
    );

    const result = await requestGraphQLResult<{
      shipment: { id: string; customer: null };
    }>({
      document: "query Test { shipment { id customer { name } } }",
      operationName: "Test",
    });

    expect(result.data.shipment.id).toBe("shp_1");
    expect(result.errors).toHaveLength(1);
    expect(result.errors[0]).toMatchObject({
      code: "FORBIDDEN",
      message: "Customer is not visible to this role",
      path: ["shipment", "customer"],
    });
  });

  it("reports partial errors so they are not silently swallowed", async () => {
    const reporter = vi.fn();
    setPartialErrorReporter(reporter);
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        data: { ok: true },
        errors: [{ message: "Partially degraded" }],
      }),
    );

    await requestGraphQL({ document: "query Test { ok }", operationName: "Test" });

    expect(reporter).toHaveBeenCalledTimes(1);
    const [errors, context] = reporter.mock.calls[0] as [unknown[], { operationName?: string }];
    expect(errors).toHaveLength(1);
    expect(context.operationName).toBe("Test");
  });

  it("does not invoke the partial reporter when no errors accompany the data", async () => {
    const reporter = vi.fn();
    setPartialErrorReporter(reporter);
    fetchMock.mockResolvedValueOnce(createGraphQLResponse({ data: { ok: true } }));

    await requestGraphQL({ document: "query Test { ok }", operationName: "Test" });

    expect(reporter).not.toHaveBeenCalled();
  });

  it("still throws when errors arrive with a null data document", async () => {
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({ data: null, errors: [{ message: "Resolver exploded" }] }),
    );

    await expect(
      requestGraphQL({ document: "query Test { ok }", operationName: "Test" }),
    ).rejects.toThrow("Resolver exploded");
  });
});

describe("requestGraphQL error classification", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("graphql-token");
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    clearCsrfToken();
    vi.unstubAllGlobals();
  });

  it("classifies an HTTP failure as transport without consulting status", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({}), {
        status: 500,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const error = await requestGraphQL({
      document: "query Test { ok }",
      operationName: "Test",
    }).catch((caught: unknown) => caught);

    expect(error).toBeInstanceOf(GraphQLRequestError);
    const gqlError = error as GraphQLRequestError;
    expect(gqlError.kind).toBe("transport");
    expect(gqlError.isTransportError()).toBe(true);
    expect(gqlError.isResolverError()).toBe(false);
  });

  it("classifies a resolver failure on HTTP 200 as a graphql error", async () => {
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        errors: [
          {
            message: "Shipment lookup failed",
            extensions: { code: "SYSTEM_ERROR", type: `${PROBLEM_BASE}internal-error` },
          },
        ],
      }),
    );

    const error = await requestGraphQL({
      document: "query Test { ok }",
      operationName: "Test",
    }).catch((caught: unknown) => caught);

    expect(error).toBeInstanceOf(GraphQLRequestError);
    const gqlError = error as GraphQLRequestError;
    expect(gqlError.kind).toBe("graphql");
    expect(gqlError.isResolverError()).toBe(true);
    expect(gqlError.isTransportError()).toBe(false);
  });

  it("keeps body errors attached to a transport failure", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          errors: [
            {
              message: "Unknown field on Shipment",
              extensions: { code: "INVALID", type: `${PROBLEM_BASE}validation-error` },
            },
          ],
        }),
        { status: 422, headers: { "Content-Type": "application/json" } },
      ),
    );

    const error = (await requestGraphQL({
      document: "query Test { ok }",
      operationName: "Test",
    }).catch((caught: unknown) => caught)) as GraphQLRequestError;

    expect(error.kind).toBe("transport");
    expect(error.message).toBe("Unknown field on Shipment");
    expect(error.graphQLErrors).toHaveLength(1);
    expect(error.code).toBe("INVALID");
  });

  it("preserves errors beyond the first with their own extensions", async () => {
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        errors: [
          {
            message: "Pro number is required",
            path: ["updateShipment"],
            extensions: {
              code: "REQUIRED",
              type: `${PROBLEM_BASE}validation-error`,
              traceId: "trace-1",
              errors: [{ field: "proNumber", message: "Pro number is required", code: "REQUIRED" }],
            },
          },
          {
            message: "Customer is inactive",
            path: ["updateShipment", "customer"],
            extensions: {
              code: "BUSINESS_LOGIC",
              type: `${PROBLEM_BASE}business-rule-violation`,
              traceId: "trace-2",
              errors: [
                { field: "customerId", message: "Customer is inactive", code: "BUSINESS_LOGIC" },
              ],
            },
          },
          {
            message: "Rate table unavailable",
            extensions: {
              code: "SYSTEM_ERROR",
              type: `${PROBLEM_BASE}database-error`,
              traceId: "trace-3",
            },
          },
        ],
      }),
    );

    const error = (await requestGraphQL({
      document: "mutation Test { updateShipment { id } }",
      operationName: "Test",
    }).catch((caught: unknown) => caught)) as GraphQLRequestError;

    expect(error.graphQLErrors).toHaveLength(3);
    expect(error.graphQLErrors[1].extensions.traceId).toBe("trace-2");
    expect(error.graphQLErrors[2].extensions.code).toBe("SYSTEM_ERROR");
    expect(error.graphQLErrors[1].path).toEqual(["updateShipment", "customer"]);

    expect(error.getCodes()).toEqual(["REQUIRED", "BUSINESS_LOGIC", "SYSTEM_ERROR"]);
    expect(error.getTraceIds()).toEqual(["trace-1", "trace-2", "trace-3"]);
    expect(error.getProblemTypes()).toEqual([
      "validation-error",
      "business-rule-violation",
      "database-error",
    ]);
    expect(error.hasProblemType("database-error")).toBe(true);

    expect(error.getFieldErrors()).toEqual([
      { field: "proNumber", message: "Pro number is required", code: "REQUIRED" },
      { field: "customerId", message: "Customer is inactive", code: "BUSINESS_LOGIC" },
    ]);
  });

  it("keeps the legacy single-error fields sourced from the first error", async () => {
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        errors: [
          {
            message: "First",
            extensions: {
              code: "REQUIRED",
              type: `${PROBLEM_BASE}validation-error`,
              traceId: "trace-1",
            },
          },
          {
            message: "Second",
            extensions: { code: "SYSTEM_ERROR", type: `${PROBLEM_BASE}internal-error` },
          },
        ],
      }),
    );

    const error = (await requestGraphQL({
      document: "query Test { ok }",
      operationName: "Test",
    }).catch((caught: unknown) => caught)) as GraphQLRequestError;

    expect(error.message).toBe("First");
    expect(error.code).toBe("REQUIRED");
    expect(error.traceId).toBe("trace-1");
    expect(error.getProblemType()).toBe("validation-error");
    expect(error.isValidationError()).toBe(true);
  });
});

describe("requestGraphQL session expiry", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("graphql-token");
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    clearCsrfToken();
    setSessionExpiryHandler(null);
    vi.unstubAllGlobals();
  });

  it("triggers the expiry path for an authentication-error extension", async () => {
    const onExpiry = vi.fn();
    setSessionExpiryHandler(onExpiry);
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        errors: [
          {
            message: "Session has expired",
            extensions: {
              code: "UNAUTHORIZED",
              type: `${PROBLEM_BASE}authentication-error`,
            },
          },
        ],
      }),
    );

    await expect(
      requestGraphQL({ document: "query Test { ok }", operationName: "Test" }),
    ).rejects.toThrow("Session has expired");

    expect(onExpiry).toHaveBeenCalledTimes(1);
  });

  it("triggers the expiry path for a bare HTTP 401", async () => {
    const onExpiry = vi.fn();
    setSessionExpiryHandler(onExpiry);
    fetchMock.mockResolvedValueOnce(
      new Response("", { status: 401, headers: { "Content-Type": "application/json" } }),
    );

    await expect(
      requestGraphQL({ document: "query Test { ok }", operationName: "Test" }),
    ).rejects.toBeInstanceOf(GraphQLRequestError);

    expect(onExpiry).toHaveBeenCalledTimes(1);
  });

  it("leaves the session alone for an authorization-error", async () => {
    const onExpiry = vi.fn();
    setSessionExpiryHandler(onExpiry);
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        errors: [
          {
            message: "Forbidden",
            extensions: { code: "FORBIDDEN", type: `${PROBLEM_BASE}authorization-error` },
          },
        ],
      }),
    );

    await expect(
      requestGraphQL({ document: "query Test { ok }", operationName: "Test" }),
    ).rejects.toThrow("Forbidden");

    expect(onExpiry).not.toHaveBeenCalled();
  });
});

describe("partial data by operation type", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  function respondWithPartialData(): void {
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({
        data: { updateManualJournalDraft: { id: "mj_1" } },
        errors: [
          {
            message: "Only draft journals can be updated",
            path: ["submitManualJournal"],
            extensions: { code: "INVALID", type: `${PROBLEM_BASE}validation-error` },
          },
        ],
      }),
    );
  }

  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("graphql-token");
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    clearCsrfToken();
    setPartialErrorReporter(null);
    vi.unstubAllGlobals();
  });

  // The defect this guards: a partially-failed write resolving as success, so onSuccess
  // runs and an optimistic update commits rather than rolling back.
  it("throws when a mutation returns data alongside errors", async () => {
    respondWithPartialData();

    const error = await requestGraphQL({
      document: "mutation M { updateManualJournalDraft { id } submitManualJournal { id } }",
      operationName: "M",
    }).catch((caught: unknown) => caught);

    expect(error).toBeInstanceOf(GraphQLRequestError);
    expect((error as GraphQLRequestError).message).toBe("Only draft journals can be updated");
  });

  it("throws for a mutation even through requestGraphQLResult", async () => {
    respondWithPartialData();

    await expect(
      requestGraphQLResult({
        document: "mutation M { updateManualJournalDraft { id } }",
        operationName: "M",
      }),
    ).rejects.toBeInstanceOf(GraphQLRequestError);
  });

  it("does not report a mutation's errors as merely partial", async () => {
    const reporter = vi.fn();
    setPartialErrorReporter(reporter);
    respondWithPartialData();

    await requestGraphQL({
      document: "mutation M { updateManualJournalDraft { id } }",
      operationName: "M",
    }).catch(() => undefined);

    expect(reporter).not.toHaveBeenCalled();
  });

  it("resolves a mutation whose errors array is present but empty", async () => {
    fetchMock.mockResolvedValueOnce(
      createGraphQLResponse({ data: { updateManualJournalDraft: { id: "mj_1" } }, errors: [] }),
    );

    await expect(
      requestGraphQL({
        document: "mutation M { updateManualJournalDraft { id } }",
        operationName: "M",
      }),
    ).resolves.toEqual({ updateManualJournalDraft: { id: "mj_1" } });
  });

  it("still resolves a query with data alongside errors", async () => {
    respondWithPartialData();

    const result = await requestGraphQLResult({
      document: "query Q { manualJournal { id } }",
      operationName: "Q",
    });

    expect(result.data).toEqual({ updateManualJournalDraft: { id: "mj_1" } });
    expect(result.errors).toHaveLength(1);
  });

  it("treats an anonymous mutation as a mutation", async () => {
    respondWithPartialData();

    await expect(
      requestGraphQL({ document: "mutation { updateManualJournalDraft { id } }" }),
    ).rejects.toBeInstanceOf(GraphQLRequestError);
  });

  it("treats a mutation preceded by a fragment definition as a mutation", async () => {
    respondWithPartialData();

    await expect(
      requestGraphQL({
        document:
          "fragment JournalFields on ManualJournal { id }\n" +
          "mutation M { updateManualJournalDraft { ...JournalFields } }",
        operationName: "M",
      }),
    ).rejects.toBeInstanceOf(GraphQLRequestError);
  });
});

describe("documentContainsMutation", () => {
  it("recognises operation definitions at the top level", () => {
    expect(documentContainsMutation("mutation M { a }")).toBe(true);
    expect(documentContainsMutation("mutation { a }")).toBe(true);
    expect(documentContainsMutation("fragment F on T { a }\nmutation M { b }")).toBe(true);
    expect(documentContainsMutation("query Q { a }\nmutation M { b }")).toBe(true);
    expect(documentContainsMutation("query Q { a }")).toBe(false);
    expect(documentContainsMutation("{ a }")).toBe(false);
    expect(documentContainsMutation("subscription S { a }")).toBe(false);
  });

  // A selection-set field named "mutation" is legal GraphQL and must not be mistaken for
  // an operation, or every query touching it would lose partial-data handling.
  it("ignores the keyword inside a selection set", () => {
    expect(documentContainsMutation("query Q { mutation { id } }")).toBe(false);
    expect(documentContainsMutation("query Q { a { b } mutation { id } }")).toBe(false);
  });

  it("ignores the keyword inside comments and block strings", () => {
    expect(documentContainsMutation("# mutation M { a }\nquery Q { a }")).toBe(false);
    expect(documentContainsMutation('query Q($s: String = """mutation""") { a }')).toBe(false);
  });
});
