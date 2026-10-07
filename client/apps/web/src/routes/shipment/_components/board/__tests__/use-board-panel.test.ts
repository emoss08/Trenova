import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useBoardPanel } from "../use-board-panel";

const { urlState, setUrl } = vi.hoisted(() => ({
  urlState: { panel: null as boolean | null },
  setUrl: vi.fn(),
}));

vi.mock("../url-state", () => ({ useShipmentBoardUrl: () => [urlState, setUrl] }));

describe("useBoardPanel", () => {
  beforeEach(() => {
    urlState.panel = null;
    setUrl.mockReset();
    window.localStorage.clear();
    vi.stubGlobal("innerWidth", 1600);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("starts closed on a wide screen when the link does not say", () => {
    const { result } = renderHook(() => useBoardPanel());
    expect(result.current[0]).toBe(false);
  });

  it("remembers that the dispatcher opened it on the next visit", () => {
    vi.stubGlobal("innerWidth", 600);
    const first = renderHook(() => useBoardPanel());
    act(() => first.result.current[1](true));
    expect(setUrl).toHaveBeenCalledWith({ panel: true });
    first.unmount();

    const next = renderHook(() => useBoardPanel());
    expect(next.result.current[0]).toBe(true);
  });

  it("follows the link when it names the panel's state", () => {
    window.localStorage.setItem("trenova.shipments.panel-open", "true");
    urlState.panel = false;
    const { result } = renderHook(() => useBoardPanel());
    expect(result.current[0]).toBe(false);
  });
});
