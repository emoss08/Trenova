import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  REQUEST_TIMEOUT_MS,
  RequestTimeoutError,
  withRequestDeadline,
} from "@trenova/shared/lib/request-errors";

function untilAborted(signal: AbortSignal): Promise<never> {
  return new Promise((_, reject) => {
    signal.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")));
  });
}

describe("withRequestDeadline", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("answers with what the request returned", async () => {
    await expect(withRequestDeadline(undefined, () => Promise.resolve("done"))).resolves.toBe(
      "done",
    );
  });

  it("ends a request that has not answered by the deadline", async () => {
    const pending = withRequestDeadline(undefined, untilAborted);
    const settled = expect(pending).rejects.toBeInstanceOf(RequestTimeoutError);

    await vi.advanceTimersByTimeAsync(REQUEST_TIMEOUT_MS);

    await settled;
  });

  it("leaves a request alone just before the deadline", async () => {
    let aborted = false;
    const pending = withRequestDeadline(undefined, (signal) => {
      signal.addEventListener("abort", () => (aborted = true));
      return new Promise<string>((resolve) => setTimeout(() => resolve("late"), 1_000));
    });

    await vi.advanceTimersByTimeAsync(1_000);

    await expect(pending).resolves.toBe("late");
    expect(aborted).toBe(false);
  });

  it("keeps a caller's cancellation a cancellation, not a timeout", async () => {
    const controller = new AbortController();
    const pending = withRequestDeadline(controller.signal, untilAborted);

    controller.abort();

    await expect(pending).rejects.toMatchObject({ name: "AbortError" });
  });

  it("starts already aborted when the caller's signal is", async () => {
    const controller = new AbortController();
    controller.abort();

    await withRequestDeadline(controller.signal, async (signal) => {
      expect(signal.aborted).toBe(true);
    });
  });

  it("does not turn a request's own failure into a timeout", async () => {
    const failure = new Error("refused");

    await expect(withRequestDeadline(undefined, () => Promise.reject(failure))).rejects.toBe(
      failure,
    );
  });

  it("leaves no timer behind once the request settles", async () => {
    await withRequestDeadline(undefined, () => Promise.resolve());

    expect(vi.getTimerCount()).toBe(0);
  });
});
