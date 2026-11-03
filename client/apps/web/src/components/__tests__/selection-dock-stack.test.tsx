import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useSelectionDockSlot } from "../selection-dock-stack";

describe("selection dock stack", () => {
  it("puts a second open dock above the first, and lets it down when the first closes", () => {
    const queue = renderHook(({ showing }) => useSelectionDockSlot(showing), {
      initialProps: { showing: true },
    });
    const documents = renderHook(({ showing }) => useSelectionDockSlot(showing), {
      initialProps: { showing: true },
    });

    expect(queue.result.current).toBe(0);
    expect(documents.result.current).toBe(1);

    queue.rerender({ showing: false });

    expect(documents.result.current).toBe(0);
    queue.unmount();
    documents.unmount();
  });

  it("takes no place in the stack while the dock is hidden", () => {
    const hidden = renderHook(() => useSelectionDockSlot(false));
    const shown = renderHook(() => useSelectionDockSlot(true));

    expect(shown.result.current).toBe(0);
    hidden.unmount();
    shown.unmount();
  });
});
