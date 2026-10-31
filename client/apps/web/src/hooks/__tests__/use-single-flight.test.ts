import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useSingleFlight } from "../use-single-flight";

function deferred() {
  let resolve!: () => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<void>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe("useSingleFlight", () => {
  it("ignores a call made while one is still running, and is pending until it settles", async () => {
    const request = deferred();
    const action = vi.fn(() => request.promise);
    const { result } = renderHook(() => useSingleFlight(action));

    let first!: Promise<void>;
    act(() => {
      first = result.current.run();
      void result.current.run();
      void result.current.run();
    });

    expect(action).toHaveBeenCalledTimes(1);
    expect(result.current.pending).toBe(true);

    await act(async () => {
      request.resolve();
      await first;
    });

    expect(result.current.pending).toBe(false);
  });

  it("runs again once the first call has settled", async () => {
    const action = vi.fn(() => Promise.resolve());
    const { result } = renderHook(() => useSingleFlight(action));

    await act(() => result.current.run());
    await act(() => result.current.run());

    expect(action).toHaveBeenCalledTimes(2);
  });

  it("releases the lock when the action fails, without rejecting", async () => {
    const request = deferred();
    const action = vi.fn(() => request.promise);
    const { result } = renderHook(() => useSingleFlight(action));

    let first!: Promise<void>;
    act(() => {
      first = result.current.run();
    });
    await act(async () => {
      request.reject(new Error("offline"));
      await first;
    });

    expect(result.current.pending).toBe(false);
    await act(() => result.current.run());
    expect(action).toHaveBeenCalledTimes(2);
  });

  it("passes the call's arguments to the action", async () => {
    const action = vi.fn((_id: string) => Promise.resolve());
    const { result } = renderHook(() => useSingleFlight(action));

    await act(() => result.current.run("ratecon_01"));

    expect(action).toHaveBeenCalledWith("ratecon_01");
  });
});
