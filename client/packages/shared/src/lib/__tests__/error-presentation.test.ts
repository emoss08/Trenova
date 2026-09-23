import { ApiRequestError } from "@trenova/shared/lib/api";
import {
  describeError,
  formatErrorReport,
  parseStackFrames,
  shortenSourcePath,
} from "@trenova/shared/lib/error-presentation";
import { GraphQLRequestError, type NormalizedGraphQLError } from "@trenova/shared/lib/graphql";
import { afterEach, describe, expect, it, vi } from "vitest";

const PROBLEM_BASE = "https://trenova.app/problems";

function apiError(status: number, type: string, detail?: string, traceId?: string) {
  return new ApiRequestError(status, {
    type: `${PROBLEM_BASE}/${type}`,
    title: type,
    status,
    detail,
    traceId,
  });
}

function graphQLError(overrides: Partial<NormalizedGraphQLError>): NormalizedGraphQLError {
  return { message: "failed", extensions: {}, ...overrides };
}

function routeErrorResponse(status: number, statusText: string, data: unknown) {
  return { status, statusText, internal: false, data };
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("describeError: connectivity", () => {
  it.each([
    ["Chromium", "Failed to fetch"],
    ["Firefox", "NetworkError when attempting to fetch resource."],
    ["Safari", "Load failed"],
    ["React Native polyfills", "Network request failed"],
  ])("classifies the %s fetch rejection as a network failure", (_browser, message) => {
    const described = describeError(new TypeError(message));

    expect(described.kind).toBe("network");
    expect(described.retryable).toBe(true);
    expect(described.detail).toBeUndefined();
  });

  it("says offline when the browser reports no connection", () => {
    vi.spyOn(navigator, "onLine", "get").mockReturnValue(false);

    expect(describeError(new TypeError("Failed to fetch")).kind).toBe("offline");
  });

  it("does not treat an ordinary TypeError as a network failure", () => {
    const described = describeError(
      new TypeError("Cannot read properties of undefined (reading 'id')"),
    );

    expect(described.kind).toBe("unexpected");
  });

  it.each([
    "Failed to fetch dynamically imported module: https://app.trenova.app/assets/page-4f1c.js",
    "error loading dynamically imported module",
    "Importing a module script failed.",
    "Unable to preload CSS for /assets/page-4f1c.css",
  ])("classifies a stale chunk (%s) as an update rather than a network failure", (message) => {
    const described = describeError(new TypeError(message));

    expect(described.kind).toBe("stale-build");
    expect(described.retryable).toBe(false);
  });
});

describe("describeError: API problems", () => {
  it.each([
    [401, "authentication-error", "session-expired"],
    [403, "authorization-error", "forbidden"],
    [404, "resource-not-found", "not-found"],
    [429, "rate-limit-exceeded", "rate-limited"],
    [504, "request-timeout", "timeout"],
    [500, "internal-error", "server"],
    [500, "database-error", "server"],
    [422, "validation-error", "request"],
    [422, "business-rule-violation", "request"],
    [409, "resource-conflict", "request"],
  ] as const)("maps %i %s to %s", (status, type, kind) => {
    expect(describeError(apiError(status, type)).kind).toBe(kind);
  });

  it("classifies by status when the problem type is one the client does not know", () => {
    expect(describeError(apiError(503, "gateway-unavailable")).kind).toBe("server");
    expect(describeError(apiError(403, "gateway-denied")).kind).toBe("forbidden");
  });

  it("shows a message the server wrote for people", () => {
    const described = describeError(
      apiError(422, "business-rule-violation", "The shipment is already billed"),
    );

    expect(described.detail).toBe("The shipment is already billed");
  });

  it("keeps the server's own fault out of the words shown to people", () => {
    const described = describeError(
      apiError(500, "database-error", "pq: duplicate key value violates unique constraint"),
    );

    expect(described.detail).toBeUndefined();
    expect(described.message).toContain("duplicate key");
  });

  it("carries the trace id through as the support reference", () => {
    expect(describeError(apiError(500, "internal-error", "boom", "trace-9f2")).traceId).toBe(
      "trace-9f2",
    );
  });

  it("classifies a resolver error inside a 200 response by its problem type, not its status", () => {
    const error = new GraphQLRequestError({
      kind: "graphql",
      message: "Missing shipment:read",
      status: 200,
      graphQLErrors: [
        graphQLError({
          message: "Missing shipment:read",
          type: `${PROBLEM_BASE}/authorization-error`,
          code: "FORBIDDEN",
          traceId: "trace-1",
        }),
      ],
    });

    const described = describeError(error);

    expect(described.kind).toBe("forbidden");
    expect(described.code).toBe("FORBIDDEN");
    expect(described.traceId).toBe("trace-1");
    expect(described.status).toBe(200);
  });

  it("classifies a transport failure by its HTTP status", () => {
    const error = new GraphQLRequestError({
      kind: "transport",
      message: "Bad gateway",
      status: 502,
    });

    expect(describeError(error).kind).toBe("server");
  });

  it("shows the only field error rather than the summary", () => {
    const error = new GraphQLRequestError({
      kind: "graphql",
      message: "Validation failed",
      status: 200,
      graphQLErrors: [
        graphQLError({
          type: `${PROBLEM_BASE}/validation-error`,
          errors: [{ field: "name", message: "Name is required" }],
        }),
      ],
    });

    expect(describeError(error).detail).toBe("Name is required");
  });
});

describe("describeError: route errors and non-errors", () => {
  it("reads a route loader's refusal", () => {
    const described = describeError(
      routeErrorResponse(403, "Missing shipment:read", "Missing permission: shipment:read"),
    );

    expect(described.kind).toBe("forbidden");
    expect(described.status).toBe(403);
    expect(described.detail).toBe("Missing permission: shipment:read");
  });

  it("falls back to the status text when a route response has no body", () => {
    expect(describeError(routeErrorResponse(404, "Not Found", null)).detail).toBe("Not Found");
  });

  it("treats a 500 route response as the server's fault and hides its body", () => {
    const described = describeError(routeErrorResponse(500, "Internal", "panic: nil map"));

    expect(described.kind).toBe("server");
    expect(described.detail).toBeUndefined();
  });

  it.each([
    ["a string", "exploded", "exploded"],
    ["null", null, "null"],
    ["an object", { reason: "x" }, "[object Object]"],
  ])("describes %s thrown in place of an Error", (_label, thrown, message) => {
    const described = describeError(thrown);

    expect(described.kind).toBe("unexpected");
    expect(described.message).toBe(message);
  });
});

describe("parseStackFrames", () => {
  it("reads Chromium frames and shortens Vite dev-server paths to the repository", () => {
    const stack = [
      "TypeError: Failed to fetch",
      "    at requestGraphQLResult (http://localhost:5173/@fs/C:/Projects/Trenova/client/packages/shared/src/lib/graphql.ts?t=1790193673376:310:25)",
      "    at async TableConfigurationService.list (http://localhost:5173/src/services/table-configuration.ts?t=1790193673376:17:20)",
      "    at http://localhost:5173/src/main.tsx:4:1",
    ].join("\n");

    expect(parseStackFrames(stack)).toEqual([
      {
        fn: "requestGraphQLResult",
        file: "client/packages/shared/src/lib/graphql.ts",
        line: "310",
        column: "25",
      },
      {
        fn: "async TableConfigurationService.list",
        file: "src/services/table-configuration.ts",
        line: "17",
        column: "20",
      },
      { fn: "anonymous", file: "src/main.tsx", line: "4", column: "1" },
    ]);
  });

  it("reads Gecko and WebKit frames", () => {
    const stack = [
      "requestGraphQLResult@http://localhost:5173/src/lib/graphql.ts:310:25",
      "@http://localhost:5173/src/main.tsx:4:1",
    ].join("\n");

    expect(parseStackFrames(stack)).toEqual([
      { fn: "requestGraphQLResult", file: "src/lib/graphql.ts", line: "310", column: "25" },
      { fn: "anonymous", file: "src/main.tsx", line: "4", column: "1" },
    ]);
  });

  it("returns nothing for an absent stack", () => {
    expect(parseStackFrames(undefined)).toEqual([]);
  });

  it("leaves a production asset URL readable", () => {
    expect(shortenSourcePath("https://app.trenova.app/assets/index-4f1c.js")).toBe(
      "assets/index-4f1c.js",
    );
  });
});

describe("formatErrorReport", () => {
  it("leads with what support searches by and omits what is absent", () => {
    const report = formatErrorReport(
      describeError(apiError(500, "internal-error", "boom", "trace-9f2")),
      {
        title: "Something went wrong on our side",
        occurredAt: new Date("2026-09-23T14:05:00Z"),
        location: "https://app.trenova.app/shipments",
      },
    );

    const lines = report.split("\n");
    expect(lines[0]).toBe("Something went wrong on our side");
    expect(report).toContain("Occurred: 2026-09-23T14:05:00.000Z");
    expect(report).toContain("Reference: trace-9f2");
    expect(report).toContain("Status: 500");
    expect(report).toContain("Problem: internal-error");
    expect(report).toContain("Page: https://app.trenova.app/shipments");
    expect(report).not.toContain("Code:");
    expect(report).not.toContain("Component stack:");
  });
});
