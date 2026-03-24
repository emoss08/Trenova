import { renderHook, act } from "@testing-library/react";
import { describe, expect, it, beforeEach } from "vitest";
import { useLocalStorage } from "../use-local-storage";

beforeEach(() => {
  localStorage.clear();
});

describe("useLocalStorage", () => {
  it("returns initial value and a setter", () => {
    const { result } = renderHook(() => useLocalStorage("key", "init"));
    expect(result.current[0]).toBe("init");
    expect(typeof result.current[1]).toBe("function");
  });

  it("setValue returns undefined (not the state setter)", () => {
    const { result } = renderHook(() => useLocalStorage("key", "init"));
    let returnValue: unknown;
    act(() => {
      returnValue = result.current[1]("new");
    });
    expect(returnValue).toBeUndefined();
  });

  it("persists value to localStorage", () => {
    const { result } = renderHook(() => useLocalStorage("key", "init"));
    act(() => {
      result.current[1]("saved");
    });
    expect(result.current[0]).toBe("saved");
    expect(JSON.parse(localStorage.getItem("key")!)).toBe("saved");
  });

  it("reads initial value from localStorage", () => {
    localStorage.setItem("key", JSON.stringify("existing"));
    const { result } = renderHook(() => useLocalStorage("key", "fallback"));
    expect(result.current[0]).toBe("existing");
  });

  it("supports function updater form", () => {
    const { result } = renderHook(() => useLocalStorage("count", 0));
    act(() => {
      result.current[1]((prev) => prev + 1);
    });
    expect(result.current[0]).toBe(1);
    expect(JSON.parse(localStorage.getItem("count")!)).toBe(1);
  });
});
