import { describe, expect, it } from "vitest";
import { mergePersisted, useAssistantStore } from "../assistant-store";

const current = () => useAssistantStore.getInitialState();

/**
 * The panel used to be expanded or not; it now has three layouts. What a
 * browser saved under the old shape is read once more on the next load, and
 * someone who left the assistant filling the screen finds it that way again.
 */
describe("restoring the assistant's layout", () => {
  it("opens full screen for someone who left it expanded", () => {
    expect(mergePersisted({ expanded: true }, current()).layout).toBe("full");
  });

  it("opens compact for someone who left it in the corner", () => {
    expect(mergePersisted({ expanded: false }, current()).layout).toBe("compact");
  });

  it("keeps a saved layout, whatever the old flag says", () => {
    expect(mergePersisted({ layout: "side" }, current()).layout).toBe("side");
    expect(mergePersisted({ layout: "side", expanded: true }, current()).layout).toBe("side");
    expect(mergePersisted({ layout: "compact", expanded: true }, current()).layout).toBe("compact");
  });

  it("falls back to compact for a layout this build does not know", () => {
    expect(mergePersisted({ layout: "floating" }, current()).layout).toBe("compact");
    expect(mergePersisted({ layout: 2 }, current()).layout).toBe("compact");
  });

  it("starts compact when nothing was saved", () => {
    expect(mergePersisted(undefined, current()).layout).toBe("compact");
    expect(mergePersisted({}, current()).layout).toBe("compact");
  });

  it("leaves the old flag behind", () => {
    expect(mergePersisted({ expanded: true }, current())).not.toHaveProperty("expanded");
  });

  it("keeps the sidebar folded only when it was explicitly folded", () => {
    expect(mergePersisted({ sidebarCollapsed: true }, current()).sidebarCollapsed).toBe(true);
    expect(mergePersisted({ sidebarCollapsed: "yes" }, current()).sidebarCollapsed).toBe(false);
  });

  it("saves the layout, not the old flag", () => {
    useAssistantStore.getState().setLayout("side");
    const saved = JSON.parse(localStorage.getItem("trenova-assistant") ?? "{}") as {
      state?: Record<string, unknown>;
    };

    expect(saved.state?.layout).toBe("side");
    expect(saved.state).not.toHaveProperty("expanded");
  });
});
