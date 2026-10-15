import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { settleWithin } from "../async";

describe("settleWithin", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns what the promise settles to when it settles in time", async () => {
    const settled = settleWithin(Promise.resolve("answer"), 100, "fallback");

    await expect(settled).resolves.toBe("answer");
  });

  it("returns the fallback once the time is up, whatever the promise does later", async () => {
    let finish: (value: string) => void = () => undefined;
    const slow = new Promise<string>((resolve) => {
      finish = resolve;
    });
    const settled = settleWithin(slow, 100, "fallback");

    await vi.advanceTimersByTimeAsync(100);
    finish("late answer");

    await expect(settled).resolves.toBe("fallback");
  });

  it("returns the fallback when the promise rejects", async () => {
    const settled = settleWithin(Promise.reject(new Error("offline")), 100, null);

    await expect(settled).resolves.toBeNull();
  });

  it("reads a negative wait as no wait at all", async () => {
    const settled = settleWithin(new Promise<string>(() => undefined), -5, "fallback");

    await vi.advanceTimersByTimeAsync(0);

    await expect(settled).resolves.toBe("fallback");
  });
});
