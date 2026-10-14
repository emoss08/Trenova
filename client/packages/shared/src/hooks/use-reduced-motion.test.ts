import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useReducedMotion } from "./use-reduced-motion";

describe("useReducedMotion", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("follows the system setting for reduced motion", () => {
    vi.stubGlobal(
      "matchMedia",
      vi.fn((query: string) => ({
        matches: query === "(prefers-reduced-motion: reduce)",
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      })),
    );

    expect(renderHook(() => useReducedMotion()).result.current).toBe(true);
  });

  it("assumes full motion where the browser cannot say", () => {
    vi.stubGlobal("matchMedia", undefined);

    expect(renderHook(() => useReducedMotion()).result.current).toBe(false);
  });
});
