import { renderHook } from "@testing-library/react";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { describe, expect, it } from "vitest";
import { useIdempotencyKey } from "../use-idempotency-key";

function keyRecorder() {
  const keys: string[] = [];
  return {
    keys,
    succeed: async (key: string) => {
      keys.push(key);
      return key;
    },
    fail: (error: unknown) => async (key: string) => {
      keys.push(key);
      throw error;
    },
  };
}

describe("useIdempotencyKey", () => {
  it("reuses the key for the same payload after a lost response", async () => {
    const { result } = renderHook(() => useIdempotencyKey());
    const rec = keyRecorder();

    const offline = new TypeError("offline");

    await expect(result.current({ bol: "A" }, rec.fail(offline))).rejects.toThrow();
    await result.current({ bol: "A" }, rec.succeed);

    expect(rec.keys[0]).toBe(rec.keys[1]);
  });

  it("starts a new key after a success", async () => {
    const { result } = renderHook(() => useIdempotencyKey());
    const rec = keyRecorder();

    await result.current({ bol: "A" }, rec.succeed);
    await result.current({ bol: "A" }, rec.succeed);

    expect(rec.keys[0]).not.toBe(rec.keys[1]);
  });

  it("starts a new key after the server answers with a failure", async () => {
    const { result } = renderHook(() => useIdempotencyKey());
    const rec = keyRecorder();
    const invalid = new GraphQLRequestError({ kind: "graphql", message: "invalid", status: 200 });

    await expect(result.current({ bol: "A" }, rec.fail(invalid))).rejects.toThrow("invalid");
    await result.current({ bol: "A" }, rec.succeed);

    expect(rec.keys[0]).not.toBe(rec.keys[1]);
  });

  it("keeps the key while the first request is still running", async () => {
    const { result } = renderHook(() => useIdempotencyKey());
    const rec = keyRecorder();
    const busy = new GraphQLRequestError({ kind: "transport", message: "busy", status: 409 });

    await expect(result.current({ bol: "A" }, rec.fail(busy))).rejects.toThrow("busy");
    await result.current({ bol: "A" }, rec.succeed);

    expect(rec.keys[0]).toBe(rec.keys[1]);
  });

  it("starts a new key when the payload changes", async () => {
    const { result } = renderHook(() => useIdempotencyKey());
    const rec = keyRecorder();

    const offline = new TypeError("offline");

    await expect(result.current({ bol: "A" }, rec.fail(offline))).rejects.toThrow();
    await result.current({ bol: "B" }, rec.succeed);

    expect(rec.keys[0]).not.toBe(rec.keys[1]);
  });
});
