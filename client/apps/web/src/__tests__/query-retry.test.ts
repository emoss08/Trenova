import { describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { retryDelay, shouldRetry } from "@/lib/query-retry";

const PROBLEM_BASE = "https://trenova.app/problems/";

function graphQLError(problemType: string, code = "SYSTEM_ERROR"): GraphQLRequestError {
  return new GraphQLRequestError({
    kind: "graphql",
    message: `failed with ${problemType}`,
    status: 200,
    graphQLErrors: [
      {
        extensions: { code, type: `${PROBLEM_BASE}${problemType}` },
        code,
        message: `failed with ${problemType}`,
        type: `${PROBLEM_BASE}${problemType}`,
      },
    ],
  });
}

function transportError(status: number, problemType?: string): GraphQLRequestError {
  return new GraphQLRequestError({
    kind: "transport",
    message: `HTTP ${status}`,
    status,
    graphQLErrors: problemType
      ? [
          {
            extensions: { type: `${PROBLEM_BASE}${problemType}` },
            message: `HTTP ${status}`,
            type: `${PROBLEM_BASE}${problemType}`,
          },
        ]
      : [],
  });
}

describe("shouldRetry", () => {
  it("retries an internal-error", () => {
    expect(shouldRetry(0, graphQLError("internal-error"))).toBe(true);
  });

  it("retries a database-error", () => {
    expect(shouldRetry(0, graphQLError("database-error"))).toBe(true);
  });

  it.each([
    "validation-error",
    "business-rule-violation",
    "authentication-error",
    "authorization-error",
    "resource-not-found",
    "rate-limit-exceeded",
    "resource-conflict",
    "request-timeout",
  ])("does not retry %s", (problemType) => {
    expect(shouldRetry(0, graphQLError(problemType))).toBe(false);
  });

  it("retries when any error in a multi-error response is retryable", () => {
    const error = new GraphQLRequestError({
      kind: "graphql",
      message: "Pro number is required",
      status: 200,
      graphQLErrors: [
        {
          extensions: { type: `${PROBLEM_BASE}validation-error` },
          message: "Pro number is required",
          type: `${PROBLEM_BASE}validation-error`,
        },
        {
          extensions: { type: `${PROBLEM_BASE}database-error` },
          message: "Rate table unavailable",
          type: `${PROBLEM_BASE}database-error`,
        },
      ],
    });

    expect(shouldRetry(0, error)).toBe(true);
  });

  it("caps at three total attempts", () => {
    const error = graphQLError("internal-error");
    expect(shouldRetry(0, error)).toBe(true);
    expect(shouldRetry(1, error)).toBe(true);
    expect(shouldRetry(2, error)).toBe(false);
  });

  it("applies the same retryable set to REST errors", () => {
    const retryable = new ApiRequestError(500, {
      type: `${PROBLEM_BASE}database-error`,
      title: "Database Error",
      status: 500,
    });
    const notRetryable = new ApiRequestError(400, {
      type: `${PROBLEM_BASE}validation-error`,
      title: "Validation Failed",
      status: 400,
    });

    expect(shouldRetry(0, retryable)).toBe(true);
    expect(shouldRetry(0, notRetryable)).toBe(false);
  });

  // Anything rejected before gqlgen runs carries no extensions, so status is the only
  // signal available. These are the cases the type-only predicate used to drop.
  it.each([500, 502, 503, 504])("retries HTTP %i with no extensions", (status) => {
    expect(shouldRetry(0, transportError(status))).toBe(true);
  });

  it.each([400, 401, 403, 404, 409, 413, 422, 429])("does not retry HTTP %i", (status) => {
    expect(shouldRetry(0, transportError(status))).toBe(false);
  });

  it("does not retry a 4xx even when the body classified it as a server fault", () => {
    expect(shouldRetry(0, transportError(429, "rate-limit-exceeded"))).toBe(false);
    expect(shouldRetry(0, transportError(400, "internal-error"))).toBe(false);
  });

  it("retries a 5xx even when the body classified it as the caller's fault", () => {
    expect(shouldRetry(0, transportError(503, "validation-error"))).toBe(true);
  });

  it("retries a connection failure that produced no response", () => {
    expect(shouldRetry(0, new TypeError("Failed to fetch"))).toBe(true);
  });

  it("retries the REST client's synthetic status-0 network error", () => {
    const error = new ApiRequestError(0, {
      type: `${PROBLEM_BASE}internal-error`,
      title: "Network error",
      detail: "Failed to connect to server",
      status: 0,
    });

    expect(shouldRetry(0, error)).toBe(true);
  });

  it("does not retry an aborted request", () => {
    expect(shouldRetry(0, new DOMException("Aborted", "AbortError"))).toBe(false);
  });

  it("does not retry errors it cannot classify", () => {
    expect(shouldRetry(0, new Error("boom"))).toBe(false);
    expect(shouldRetry(0, undefined)).toBe(false);
  });
});

describe("retryDelay", () => {
  it("grows exponentially and stays within the jitter window", () => {
    const random = vi.spyOn(Math, "random");

    random.mockReturnValue(0);
    expect(retryDelay(0)).toBe(250);
    expect(retryDelay(1)).toBe(500);
    expect(retryDelay(2)).toBe(1000);

    random.mockReturnValue(0.999999);
    expect(retryDelay(0)).toBe(500);
    expect(retryDelay(1)).toBe(1000);
    expect(retryDelay(2)).toBe(2000);

    random.mockRestore();
  });

  it("caps the ceiling so a long backoff cannot run away", () => {
    const random = vi.spyOn(Math, "random").mockReturnValue(0.999999);
    expect(retryDelay(20)).toBe(8000);
    random.mockRestore();
  });

  it("produces varying delays for the same attempt", () => {
    const delays = new Set(Array.from({ length: 50 }, () => retryDelay(2)));
    expect(delays.size).toBeGreaterThan(1);
  });
});
