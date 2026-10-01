import { describe, expect, it } from "vitest";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import {
  IDEMPOTENCY_KEY_HEADER,
  keepsIdempotencyKey,
  newIdempotencyKey,
  withIdempotencyKeyHeader,
} from "@trenova/shared/lib/idempotency";

function requestError(kind: "graphql" | "transport", status: number) {
  return new GraphQLRequestError({ kind, message: "failed", status });
}

describe("keepsIdempotencyKey", () => {
  it("keeps the key when the request got no answer", () => {
    expect(keepsIdempotencyKey(new TypeError("Failed to fetch"))).toBe(true);
    expect(keepsIdempotencyKey(undefined)).toBe(true);
  });

  it("keeps the key while the first request is still running", () => {
    expect(keepsIdempotencyKey(requestError("transport", 409))).toBe(true);
  });

  it("drops the key once the server has answered", () => {
    expect(keepsIdempotencyKey(requestError("graphql", 200))).toBe(false);
    expect(keepsIdempotencyKey(requestError("transport", 400))).toBe(false);
    expect(keepsIdempotencyKey(requestError("transport", 503))).toBe(false);
  });
});

describe("withIdempotencyKeyHeader", () => {
  it("adds the header only for a key", () => {
    expect(withIdempotencyKeyHeader({ Accept: "a" }, "k")).toEqual({
      Accept: "a",
      [IDEMPOTENCY_KEY_HEADER]: "k",
    });
    expect(withIdempotencyKeyHeader({ Accept: "a" }, undefined)).toEqual({ Accept: "a" });
  });
});

describe("newIdempotencyKey", () => {
  it("makes a distinct key each time", () => {
    expect(newIdempotencyKey()).not.toBe(newIdempotencyKey());
  });
});
